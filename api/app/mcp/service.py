from __future__ import annotations

import json
from collections.abc import Iterator
from dataclasses import dataclass
from typing import Any
from uuid import UUID

from fastapi import HTTPException
from pydantic import ValidationError

from app.chat.errors import ChatProviderExecutionError, ChatServiceError
from app.chat.service import ChatService
from app.ingestion.service import DocumentIngestionService
from app.mcp_tokens import McpTokenAuthContext
from app.memory_admin import (
    MemoryAdminEngramListRequest,
    MemoryAdminListRequest,
    MemoryAdminService,
)
from app.models import (
    AdminEngramMoveRequest,
    AdminEngramRecord,
    AdminEngramUpdateRequest,
    AdminSessionDeleteRequest,
    ChatMessageCreateRequest,
    EngramCollectionCreateRequest,
    EngramCollectionRecord,
    EngramCreateFromConversationRequest,
    McpJsonRpcRequest,
    MemoryEngramCreate,
    SaveSessionAsEngramRequest,
)
from app.projects import ProjectService
from app.repository import (
    create_engram,
    create_engram_with_report,
    get_rehydration_bundle,
    query_engrams,
)

from .catalog import (
    _TOOL_ALIASES,
    _to_dotted_tool_name,
)
from .chat_dispatch import (
    dispatch_chat_pinning_tool,
    dispatch_chat_primary_tool,
    dispatch_chat_session_query_tool,
)
from .engram_dispatch import (
    EngramDispatchContext,
    EngramDispatchDependencies,
    dispatch_engram_mutation_tool,
    dispatch_engram_read_tool,
)
from .errors import McpRpcError
from .project_user_dispatch import dispatch_project_tool, dispatch_user_tool
from .streaming import (
    chat_send_message_success_frame,
    stream_chat_send_message_error_frame,
    stream_chat_send_message_events,
)
from .token_authorization import (
    TokenAuthorizationDependencies,
    enforce_token_authorization,
    project_id_for_tool,
    resolve_project_for_write,
    visible_tool_catalog,
)

_COLLECTION_SCOPED_TOOLS = {
    "engram.collection_update",
    "engram.collection_delete",
    "engram.collection_add_items",
    "engram.collection_remove_items",
}


@dataclass(frozen=True)
class _StreamRouteRequest:
    actor_user_id: UUID
    request_id: str | int | None
    request_method: str
    canonical_method: str
    request_params: dict[str, Any]
    token_auth: McpTokenAuthContext | None


@dataclass(frozen=True)
class _ToolDispatchContext:
    actor: dict[str, Any]
    actor_user_id: UUID
    method: str
    params: dict[str, Any]
    token_auth: McpTokenAuthContext | None


@dataclass(frozen=True)
class _CreateEngramFromConversationContext:
    actor_user_id: UUID
    actor_role: str
    token_auth: McpTokenAuthContext | None
    params: dict[str, Any]
    enrichment_origin: str


@dataclass(frozen=True)
class _AuthorizeToolCallRequest:
    actor_user_id: UUID
    request_id: str | int | None
    token_auth: McpTokenAuthContext | None
    tool_name: str
    params: dict[str, Any]


@dataclass(frozen=True)
class _StreamChatSendMessageRequest:
    actor_user_id: UUID
    request_id: str | int
    tool_name: str
    params: dict[str, Any]
    as_tool_call: bool = False


class McpService:
    """JSON-RPC tool dispatcher for MCP-over-SSE.

    Design intent:
    - Keep transport in `mcp/api.py` and business logic in domain services/repositories.
    - Support both legacy direct tool methods (`chat.*`, `engram.*`, `user.*`)
      and interoperable MCP-style methods (`initialize`, `tools/list`, `tools/call`).
    - Keep auth and visibility parity with REST handlers by requiring a resolved actor.
    """

    def __init__(
        self,
        chat_service: ChatService,
        project_service: ProjectService,
        memory_admin_service: MemoryAdminService,
        embedding_dim: int,
        ingestion_service: DocumentIngestionService | None = None,
        server_version: str = "0.1.0",
    ) -> None:
        self._chat_service = chat_service
        self._project_service = project_service
        self._memory_admin_service = memory_admin_service
        self._embedding_dim = embedding_dim
        self._ingestion_service = ingestion_service
        self._server_version = server_version

    @staticmethod
    def _success(id_value: str | int | None, result: dict[str, Any]) -> dict[str, Any]:
        return {"jsonrpc": "2.0", "id": id_value, "result": result}

    @staticmethod
    def _error(
        id_value: str | int | None,
        *,
        code: int,
        message: str,
        data: dict[str, Any] | None = None,
    ) -> dict[str, Any]:
        payload: dict[str, Any] = {"code": code, "message": message}
        if data is not None:
            payload["data"] = data
        return {"jsonrpc": "2.0", "id": id_value, "error": payload}

    @staticmethod
    def _event(
        id_value: str | int | None,
        *,
        tool: str,
        event_name: str,
        event_payload: dict[str, Any],
    ) -> dict[str, Any]:
        return {
            "jsonrpc": "2.0",
            "method": "mcp.event",
            "params": {
                "id": id_value,
                "tool": tool,
                "event": event_name,
                "data": event_payload,
            },
        }

    @staticmethod
    def _parse_uuid(params: dict[str, Any], key: str) -> UUID:
        raw = params.get(key)
        if raw is None:
            raise McpRpcError(
                code=-32602,
                message="Invalid params",
                data={"missing": key},
            )
        try:
            return UUID(str(raw))
        except ValueError as exc:
            raise McpRpcError(
                code=-32602,
                message="Invalid params",
                data={"invalid": key},
            ) from exc

    @staticmethod
    def _parse_uuid_list(params: dict[str, Any], key: str) -> list[UUID]:
        raw = params.get(key, [])
        if raw is None:
            return []
        if not isinstance(raw, list):
            raise McpRpcError(
                code=-32602,
                message="Invalid params",
                data={"invalid": key},
            )
        parsed: list[UUID] = []
        for index, item in enumerate(raw):
            try:
                parsed.append(UUID(str(item)))
            except ValueError as exc:
                raise McpRpcError(
                    code=-32602,
                    message="Invalid params",
                    data={"invalid": key, "index": index},
                ) from exc
        return parsed

    @staticmethod
    def _tool_call_success(tool_name: str, payload: dict[str, Any]) -> dict[str, Any]:
        # `tools/call` responses include a human-readable text content field and
        # a structured payload. `default=str` keeps UUID/datetime values serializable
        # without losing deterministic machine-readable `structuredContent`.
        return {
            "tool_name": tool_name,
            "structuredContent": payload,
            "content": [{"type": "text", "text": json.dumps(payload, default=str)}],
            "isError": False,
        }

    @staticmethod
    def _canonical_tool_name(tool_name: str) -> str:
        dotted = _to_dotted_tool_name(tool_name)
        return _TOOL_ALIASES.get(dotted, dotted)

    @staticmethod
    def _is_admin(actor: dict[str, Any]) -> bool:
        return str(actor.get("role", "")).lower() == "admin"

    def _require_owner_or_admin(
        self,
        *,
        actor: dict[str, Any],
        owner_user_id: UUID | None,
        resource: str,
        resource_id: str,
    ) -> None:
        if self._is_admin(actor):
            return
        actor_user_id = UUID(str(actor["user_id"]))
        if owner_user_id and owner_user_id == actor_user_id:
            return
        raise McpRpcError(
            code=-32003,
            message="Resource ownership policy denied this action",
            data={"resource": resource, "resource_id": resource_id},
        )

    def _authorization_dependencies(self) -> TokenAuthorizationDependencies:
        return TokenAuthorizationDependencies(
            project_service=self._project_service,
            chat_service=self._chat_service,
            memory_admin_service=self._memory_admin_service,
            parse_uuid=self._parse_uuid,
            canonical_tool_name=self._canonical_tool_name,
            get_rehydration_bundle=get_rehydration_bundle,
        )

    def _resolve_project_for_write(
        self,
        *,
        actor_user_id: UUID,
        actor_role: str,
        requested_project_id: str | None,
        token_auth: McpTokenAuthContext | None,
    ) -> tuple[str, bool]:
        """Resolve project scope for writes with token-aware fallback semantics.

        Priority:
        1) explicit project_id in params
        2) single allowed project from MCP token policy
        3) user default project from project settings
        """
        return resolve_project_for_write(
            dependencies=self._authorization_dependencies(),
            actor_user_id=actor_user_id,
            actor_role=actor_role,
            requested_project_id=requested_project_id,
            token_auth=token_auth,
        )

    def _create_engram_payload_with_project_resolution(
        self,
        *,
        actor_user_id: UUID,
        actor_role: str,
        token_auth: McpTokenAuthContext | None,
        payload: MemoryEngramCreate,
    ) -> tuple[MemoryEngramCreate, str, bool]:
        project_id, used_default = self._resolve_project_for_write(
            actor_user_id=actor_user_id,
            actor_role=actor_role,
            requested_project_id=payload.project_id,
            token_auth=token_auth,
        )
        return payload.model_copy(update={"project_id": project_id}), project_id, used_default

    def _create_engram_from_conversation(
        self,
        *,
        context: _CreateEngramFromConversationContext,
    ) -> tuple[dict[str, Any], dict[str, Any]]:
        """Create an engram from raw conversation text (no chat session required)."""
        request = EngramCreateFromConversationRequest(**context.params)
        resolved_payload, resolved_project_id, used_default_project = (
            self._create_engram_payload_with_project_resolution(
                actor_user_id=context.actor_user_id,
                actor_role=context.actor_role,
                token_auth=context.token_auth,
                payload=MemoryEngramCreate(
                    project_id=request.project_id,
                    thread_id=request.thread_id,
                    title=request.title,
                    abstract=request.abstract,
                    detailed_summary_markdown=request.conversation_markdown,
                    tags=request.tags,
                    keywords=request.keywords,
                    visibility_scope=request.visibility_scope.value,
                    retrieval_text=request.retrieval_text,
                    source_session_id=request.source_session_id,
                ),
            )
        )
        created, enrichment_report = create_engram_with_report(
            payload=resolved_payload,
            embedding_dim=self._embedding_dim,
            owner_user_id=context.actor_user_id,
            enrichment_origin=context.enrichment_origin,
        )
        engram_payload = created.model_dump(mode="json")
        engram_payload["resolved_project_id"] = resolved_project_id
        engram_payload["used_default_project"] = used_default_project
        report_payload = {
            "enrichment_applied": enrichment_report.get("enrichment_applied", False),
            "auto_tags": enrichment_report.get("auto_tags", []),
            "auto_keywords": enrichment_report.get("auto_keywords", []),
            "abstract_derived": enrichment_report.get("abstract_derived", False),
            "resolved_project_id": resolved_project_id,
            "used_default_project": used_default_project,
        }
        return engram_payload, report_payload

    def _visible_tool_catalog(self, token_auth: McpTokenAuthContext | None) -> list[dict[str, Any]]:
        return visible_tool_catalog(
            dependencies=self._authorization_dependencies(),
            token_auth=token_auth,
        )

    def _project_id_for_tool(
        self,
        *,
        actor_user_id: UUID,
        tool_name: str,
        params: dict[str, Any],
    ) -> str | None:
        return project_id_for_tool(
            dependencies=self._authorization_dependencies(),
            actor_user_id=actor_user_id,
            tool_name=tool_name,
            params=params,
        )

    def _enforce_token_authorization(
        self,
        *,
        actor_user_id: UUID,
        token_auth: McpTokenAuthContext | None,
        tool_name: str,
        params: dict[str, Any],
    ) -> dict[str, Any]:
        return enforce_token_authorization(
            dependencies=self._authorization_dependencies(),
            actor_user_id=actor_user_id,
            token_auth=token_auth,
            tool_name=tool_name,
            params=params,
        )

    @staticmethod
    def _tool_name_and_params_for_tools_call(
        params: dict[str, Any],
    ) -> tuple[str, dict[str, Any]]:
        tool_name = params.get("name")
        if not isinstance(tool_name, str) or not tool_name:
            raise McpRpcError(code=-32602, message="Invalid params", data={"missing": "name"})

        tool_params = params.get("arguments", {})
        if tool_params is None:
            tool_params = {}
        if not isinstance(tool_params, dict):
            raise McpRpcError(code=-32602, message="Invalid params", data={"invalid": "arguments"})
        return tool_name, tool_params

    def _require_session_access(self, *, actor: dict[str, Any], session_id: UUID) -> Any:
        session = self._memory_admin_service.get_session(
            session_id=session_id, include_deleted=True
        )
        if not session:
            raise McpRpcError(
                code=-32004,
                message="Session not found",
                data={"session_id": str(session_id)},
            )
        self._require_owner_or_admin(
            actor=actor,
            owner_user_id=session.owner_user_id,
            resource="session",
            resource_id=str(session_id),
        )
        return session

    def _require_engram_access(
        self,
        *,
        actor: dict[str, Any],
        engram_id: UUID,
        include_deleted: bool = True,
    ) -> Any:
        engram = self._memory_admin_service.find_engram(
            engram_id=engram_id,
            include_deleted=include_deleted,
        )
        if not engram:
            raise McpRpcError(
                code=-32004,
                message="Engram not found",
                data={"engram_id": str(engram_id)},
            )
        self._require_owner_or_admin(
            actor=actor,
            owner_user_id=engram.owner_user_id,
            resource="engram",
            resource_id=str(engram_id),
        )
        return engram

    def _require_collection_access(self, *, actor: dict[str, Any], collection_id: UUID) -> Any:
        collection = self._memory_admin_service.find_collection(
            collection_id=collection_id,
            include_deleted=True,
        )
        if not collection:
            raise McpRpcError(
                code=-32004,
                message="Collection not found",
                data={"collection_id": str(collection_id)},
            )
        self._require_owner_or_admin(
            actor=actor,
            owner_user_id=collection.owner_user_id,
            resource="collection",
            resource_id=str(collection_id),
        )
        return collection

    def _dispatch_chat_save_as_engram_tool(
        self,
        *,
        actor_user_id: UUID,
        actor_role: str,
        params: dict[str, Any],
        token_auth: McpTokenAuthContext | None,
    ) -> dict[str, Any]:
        session_id = params.get("session_id")
        if session_id:
            saved = self._chat_service.save_session_as_engram(
                actor_user_id=actor_user_id,
                session_id=self._parse_uuid(params, "session_id"),
                payload=SaveSessionAsEngramRequest(
                    title=params.get("title", "Session Snapshot"),
                    abstract=params.get("abstract", ""),
                    visibility_scope=params.get("visibility_scope", "private"),
                    tags=params.get("tags", []),
                    keywords=params.get("keywords", []),
                ),
            )
            return {"saved_engram": saved.model_dump(mode="json")}

        if not params.get("conversation_markdown"):
            raise McpRpcError(
                code=-32602,
                message="Invalid params",
                data={"missing": "conversation_markdown"},
            )

        fallback_params = dict(params)
        if not fallback_params.get("title"):
            fallback_params["title"] = "Conversation Snapshot"
        created, report = self._create_engram_from_conversation(
            context=_CreateEngramFromConversationContext(
                actor_user_id=actor_user_id,
                actor_role=actor_role,
                token_auth=token_auth,
                params=fallback_params,
                enrichment_origin="mcp.chat.save_as_engram",
            )
        )
        return {"saved_engram": created, "enrichment_report": report}

    def _dispatch_chat_session_lifecycle_tool(
        self,
        *,
        actor: dict[str, Any],
        actor_user_id: UUID,
        method: str,
        params: dict[str, Any],
    ) -> dict[str, Any] | None:
        if method == "chat.delete_session":
            session_id = self._parse_uuid(params, "session_id")
            self._require_session_access(actor=actor, session_id=session_id)
            deleted = self._memory_admin_service.delete_session(
                session_id=session_id,
                actor_user_id=actor_user_id,
                payload=AdminSessionDeleteRequest(
                    delete_linked_engrams=bool(params.get("delete_linked_engrams", False)),
                    reason=params.get("reason"),
                ),
            )
            return {"result": deleted.model_dump(mode="json")}

        if method == "chat.restore_session":
            session_id = self._parse_uuid(params, "session_id")
            self._require_session_access(actor=actor, session_id=session_id)
            restored = self._memory_admin_service.restore_session(session_id=session_id)
            return {"result": restored.model_dump(mode="json")}

        return None

    def _dispatch_chat_tool(
        self,
        *,
        context: _ToolDispatchContext,
    ) -> dict[str, Any] | None:
        actor_role = str(context.actor.get("role", ""))

        chat_session_query_result = dispatch_chat_session_query_tool(
            chat_service=self._chat_service,
            actor_user_id=context.actor_user_id,
            method=context.method,
            params=context.params,
            parse_uuid=self._parse_uuid,
        )
        if chat_session_query_result is not None:
            return chat_session_query_result

        chat_pinning_result = dispatch_chat_pinning_tool(
            chat_service=self._chat_service,
            actor_user_id=context.actor_user_id,
            method=context.method,
            params=context.params,
            parse_uuid=self._parse_uuid,
        )
        if chat_pinning_result is not None:
            return chat_pinning_result

        chat_primary_result = dispatch_chat_primary_tool(
            chat_service=self._chat_service,
            ingestion_service=self._ingestion_service,
            actor_user_id=context.actor_user_id,
            actor_role=actor_role,
            method=context.method,
            params=context.params,
            token_auth=context.token_auth,
            parse_uuid=self._parse_uuid,
            dispatch_chat_save_as_engram_tool=self._dispatch_chat_save_as_engram_tool,
        )
        if chat_primary_result is not None:
            return chat_primary_result

        session_lifecycle_result = self._dispatch_chat_session_lifecycle_tool(
            actor=context.actor,
            actor_user_id=context.actor_user_id,
            method=context.method,
            params=context.params,
        )
        if session_lifecycle_result is not None:
            return session_lifecycle_result

        return None

    def _dispatch_engram_collection_create_tool(
        self,
        *,
        actor_user_id: UUID,
        actor_role: str,
        params: dict[str, Any],
        token_auth: McpTokenAuthContext | None,
    ) -> dict[str, Any]:
        resolved_project_id, used_default_project = self._resolve_project_for_write(
            actor_user_id=actor_user_id,
            actor_role=actor_role,
            requested_project_id=params.get("project_id"),
            token_auth=token_auth,
        )
        created = self._memory_admin_service.create_collection(
            actor_user_id=actor_user_id,
            actor_role=actor_role,
            payload=EngramCollectionCreateRequest(
                project_id=resolved_project_id,
                name=str(params.get("name", "")),
                description=str(params.get("description", "")),
            ),
        )
        return {
            "collection": created.model_dump(mode="json"),
            "resolved_project_id": resolved_project_id,
            "used_default_project": used_default_project,
        }

    def _dispatch_engram_move_project_tool(
        self,
        *,
        actor: dict[str, Any],
        actor_user_id: UUID,
        params: dict[str, Any],
        token_auth: McpTokenAuthContext | None,
    ) -> dict[str, Any]:
        actor_role = str(actor.get("role", ""))
        engram_id = self._parse_uuid(params, "engram_id")
        current = self._require_engram_access(actor=actor, engram_id=engram_id, include_deleted=True)
        allowed_project_ids = token_auth.allowed_project_ids if token_auth else set()
        if allowed_project_ids and current.project_id not in allowed_project_ids:
            raise McpRpcError(
                code=-32003,
                message="Project not allowed by token policy",
                data={
                    "tool": "engram.move_project",
                    "project_id": current.project_id,
                    "token_scope": token_auth.scope,
                },
            )
        moved = self._memory_admin_service.move_engram(
            engram_id=engram_id,
            actor_user_id=actor_user_id,
            actor_role=actor_role,
            payload=AdminEngramMoveRequest(
                target_project_id=str(params.get("target_project_id", "")),
                expected_updated_at=params.get("expected_updated_at"),
                reason=params.get("reason"),
            ),
        )
        return {"engram": moved.model_dump(mode="json")}

    def _dispatch_engram_create_tool(
        self,
        *,
        actor: dict[str, Any],
        actor_user_id: UUID,
        params: dict[str, Any],
        token_auth: McpTokenAuthContext | None,
    ) -> dict[str, Any]:
        actor_role = str(actor.get("role", ""))
        resolved_payload, resolved_project_id, used_default_project = (
            self._create_engram_payload_with_project_resolution(
                actor_user_id=actor_user_id,
                actor_role=actor_role,
                token_auth=token_auth,
                payload=MemoryEngramCreate(**params),
            )
        )
        created = create_engram(
            payload=resolved_payload,
            embedding_dim=self._embedding_dim,
            owner_user_id=actor_user_id,
            enrichment_origin="mcp.engram.create",
        )
        engram = created.model_dump(mode="json")
        engram["resolved_project_id"] = resolved_project_id
        engram["used_default_project"] = used_default_project
        return {"engram": engram}

    def _dispatch_engram_create_from_conversation_tool(
        self,
        *,
        actor: dict[str, Any],
        actor_user_id: UUID,
        params: dict[str, Any],
        token_auth: McpTokenAuthContext | None,
    ) -> dict[str, Any]:
        actor_role = str(actor.get("role", ""))
        created, report = self._create_engram_from_conversation(
            context=_CreateEngramFromConversationContext(
                actor_user_id=actor_user_id,
                actor_role=actor_role,
                token_auth=token_auth,
                params=params,
                enrichment_origin="mcp.engram.create_from_conversation",
            )
        )
        return {"engram": created, "enrichment_report": report}

    def _dispatch_engram_update_tool(
        self,
        *,
        actor: dict[str, Any],
        actor_user_id: UUID,
        params: dict[str, Any],
        token_auth: McpTokenAuthContext | None,  # noqa: ARG002
    ) -> dict[str, Any]:
        engram_id = self._parse_uuid(params, "engram_id")
        self._require_engram_access(actor=actor, engram_id=engram_id, include_deleted=True)
        updated = self._memory_admin_service.update_engram(
            engram_id=engram_id,
            actor_user_id=actor_user_id,
            payload=AdminEngramUpdateRequest(
                title=params.get("title"),
                abstract=params.get("abstract"),
                detailed_summary_markdown=params.get("detailed_summary_markdown"),
                tags=params.get("tags"),
                keywords=params.get("keywords"),
                visibility_scope=params.get("visibility_scope"),
                expected_updated_at=params.get("expected_updated_at"),
                sources=params.get("sources"),
            ),
        )
        return {"engram": updated.model_dump(mode="json")}

    def _dispatch_engram_primary_tool(
        self,
        *,
        context: _ToolDispatchContext,
    ) -> dict[str, Any] | None:
        actor_role = str(context.actor.get("role", ""))

        handlers = {
            "engram.create": lambda: self._dispatch_engram_create_tool(
                actor=context.actor,
                actor_user_id=context.actor_user_id,
                params=context.params,
                token_auth=context.token_auth,
            ),
            "engram.create_from_conversation": lambda: self._dispatch_engram_create_from_conversation_tool(
                actor=context.actor,
                actor_user_id=context.actor_user_id,
                params=context.params,
                token_auth=context.token_auth,
            ),
            "engram.update": lambda: self._dispatch_engram_update_tool(
                actor=context.actor,
                actor_user_id=context.actor_user_id,
                params=context.params,
                token_auth=context.token_auth,
            ),
            "engram.move_project": lambda: self._dispatch_engram_move_project_tool(
                actor=context.actor,
                actor_user_id=context.actor_user_id,
                params=context.params,
                token_auth=context.token_auth,
            ),
            "engram.collection_create": lambda: self._dispatch_engram_collection_create_tool(
                actor_user_id=context.actor_user_id,
                actor_role=actor_role,
                params=context.params,
                token_auth=context.token_auth,
            ),
        }
        handler = handlers.get(context.method)
        return handler() if handler else None

    def _dispatch_engram_tool(
        self,
        *,
        context: _ToolDispatchContext,
    ) -> dict[str, Any] | None:
        engram_primary_result = self._dispatch_engram_primary_tool(
            context=context,
        )
        if engram_primary_result is not None:
            return engram_primary_result

        dispatch_context = EngramDispatchContext(
            actor=context.actor,
            actor_user_id=context.actor_user_id,
            method=context.method,
            params=context.params,
        )
        dispatch_dependencies = EngramDispatchDependencies(
            memory_admin_service=self._memory_admin_service,
            embedding_dim=self._embedding_dim,
            parse_uuid=self._parse_uuid,
            parse_uuid_list=self._parse_uuid_list,
            require_collection_access=self._require_collection_access,
            require_engram_access=self._require_engram_access,
            list_engrams_for_actor=self._list_engrams_for_actor,
            list_collections_for_actor=self._list_collections_for_actor,
            query_engrams=query_engrams,
            get_rehydration_bundle=get_rehydration_bundle,
        )

        engram_read_result = dispatch_engram_read_tool(
            dependencies=dispatch_dependencies,
            context=dispatch_context,
        )
        if engram_read_result is not None:
            return engram_read_result

        mutation_result = dispatch_engram_mutation_tool(
            dependencies=dispatch_dependencies,
            context=dispatch_context,
        )
        if mutation_result is not None:
            return mutation_result

        return None

    def _list_engrams_for_actor(
        self,
        *,
        actor: dict[str, Any],
        actor_user_id: UUID,
        session_id: UUID | None,
        params: dict[str, Any],
    ) -> list[AdminEngramRecord]:
        return self._memory_admin_service.list_engrams(
            request=MemoryAdminEngramListRequest(
                project_id=params.get("project_id"),
                owner_user_id=None if self._is_admin(actor) else actor_user_id,
                include_deleted=bool(params.get("include_deleted", False)),
                limit=int(params.get("limit", 200)),
                offset=int(params.get("offset", 0)),
                session_id=session_id,
                query_text=params.get("q"),
            )
        )

    def _list_collections_for_actor(
        self,
        *,
        actor: dict[str, Any],
        actor_user_id: UUID,
        params: dict[str, Any],
    ) -> list[EngramCollectionRecord]:
        return self._memory_admin_service.list_collections(
            request=MemoryAdminListRequest(
                project_id=params.get("project_id"),
                owner_user_id=None if self._is_admin(actor) else actor_user_id,
                include_deleted=bool(params.get("include_deleted", False)),
                limit=int(params.get("limit", 200)),
                offset=int(params.get("offset", 0)),
            )
        )

    def _dispatch_tool(
        self,
        *,
        context: _ToolDispatchContext,
    ) -> dict[str, Any]:
        chat_result = self._dispatch_chat_tool(
            context=context,
        )
        if chat_result is not None:
            return chat_result

        engram_result = self._dispatch_engram_tool(
            context=context,
        )
        if engram_result is not None:
            return engram_result

        project_result = dispatch_project_tool(
            project_service=self._project_service,
            actor=context.actor,
            actor_user_id=context.actor_user_id,
            method=context.method,
            params=context.params,
        )
        if project_result is not None:
            return project_result

        user_result = dispatch_user_tool(
            actor=context.actor,
            actor_user_id=context.actor_user_id,
            method=context.method,
            chat_service=self._chat_service,
        )
        if user_result is not None:
            return user_result

        raise McpRpcError(
            code=-32601,
            message="Method not found",
            data={"method": context.method},
        )

    def _dispatch_non_stream(
        self,
        *,
        actor: dict[str, Any],
        actor_user_id: UUID,
        request: McpJsonRpcRequest,
        token_auth: McpTokenAuthContext | None = None,
    ) -> dict[str, Any]:
        params = request.params
        method = request.method

        # Interop surface for standard MCP clients.
        if method == "initialize":
            return {
                "protocolVersion": "2024-11-05",
                "serverInfo": {"name": "engram-vault-mcp", "version": self._server_version},
                "capabilities": {"tools": {"listChanged": False}},
            }

        if method == "tools/list":
            return {"tools": self._visible_tool_catalog(token_auth)}

        if method == "tools/call":
            tool_name, tool_params = self._tool_name_and_params_for_tools_call(params)
            canonical_tool_name = self._canonical_tool_name(tool_name)
            authorized_params = self._enforce_token_authorization(
                actor_user_id=actor_user_id,
                token_auth=token_auth,
                tool_name=tool_name,
                params=tool_params,
            )
            tool_payload = self._dispatch_tool(
                context=_ToolDispatchContext(
                    actor=actor,
                    actor_user_id=actor_user_id,
                    method=canonical_tool_name,
                    params=authorized_params,
                    token_auth=token_auth,
                ),
            )
            return self._tool_call_success(tool_name, tool_payload)

        # Backward-compatible direct method path.
        canonical_method = self._canonical_tool_name(method)
        authorized_params = self._enforce_token_authorization(
            actor_user_id=actor_user_id,
            token_auth=token_auth,
            tool_name=method,
            params=params,
        )
        return self._dispatch_tool(
            context=_ToolDispatchContext(
                actor=actor,
                actor_user_id=actor_user_id,
                method=canonical_method,
                params=authorized_params,
                token_auth=token_auth,
            ),
        )

    def _authorize_tool_call(
        self,
        *,
        request: _AuthorizeToolCallRequest,
    ) -> tuple[dict[str, Any] | None, dict[str, Any] | None]:
        try:
            authorized_params = self._enforce_token_authorization(
                actor_user_id=request.actor_user_id,
                token_auth=request.token_auth,
                tool_name=request.tool_name,
                params=request.params,
            )
        except McpRpcError as exc:
            return None, self._error(request.request_id, code=exc.code, message=exc.message, data=exc.data)
        return authorized_params, None

    def _maybe_stream_direct_chat_send_message(
        self,
        *,
        request_ctx: _StreamRouteRequest,
    ) -> tuple[bool, Iterator[dict[str, Any]] | None]:
        if request_ctx.canonical_method != "chat.send_message":
            return False, None
        authorized_params, error_frame = self._authorize_tool_call(
            request=_AuthorizeToolCallRequest(
                actor_user_id=request_ctx.actor_user_id,
                request_id=request_ctx.request_id,
                token_auth=request_ctx.token_auth,
                tool_name=request_ctx.request_method,
                params=request_ctx.request_params,
            ),
        )
        if error_frame is not None:
            return True, iter((error_frame,))
        assert authorized_params is not None
        return (
            True,
            self._stream_chat_send_message(
                request=_StreamChatSendMessageRequest(
                    actor_user_id=request_ctx.actor_user_id,
                    request_id=request_ctx.request_id,
                    tool_name=request_ctx.request_method,
                    params=authorized_params,
                ),
            ),
        )

    def _maybe_stream_tools_call_chat_message(
        self,
        *,
        request_ctx: _StreamRouteRequest,
    ) -> tuple[bool, Iterator[dict[str, Any]] | None]:
        if request_ctx.request_method != "tools/call":
            return False, None

        try:
            tool_name, tool_params = self._tool_name_and_params_for_tools_call(
                request_ctx.request_params
            )
            canonical_tool_name = self._canonical_tool_name(tool_name)
        except McpRpcError as exc:
            return (
                True,
                iter(
                    (
                        self._error(
                            request_ctx.request_id,
                            code=exc.code,
                            message=exc.message,
                            data=exc.data,
                        ),
                    )
                ),
            )

        authorized_tool_params, error_frame = self._authorize_tool_call(
            request=_AuthorizeToolCallRequest(
                actor_user_id=request_ctx.actor_user_id,
                request_id=request_ctx.request_id,
                token_auth=request_ctx.token_auth,
                tool_name=tool_name,
                params=tool_params,
            ),
        )
        if error_frame is not None:
            return True, iter((error_frame,))
        assert authorized_tool_params is not None

        if canonical_tool_name == "chat.send_message" and bool(authorized_tool_params.get("stream", True)):
            return (
                True,
                self._stream_chat_send_message(
                    request=_StreamChatSendMessageRequest(
                        actor_user_id=request_ctx.actor_user_id,
                        request_id=request_ctx.request_id,
                        tool_name=tool_name,
                        params=authorized_tool_params,
                        as_tool_call=True,
                    ),
                ),
            )
        return False, None

    def _stream_dispatch_non_stream_result(
        self,
        *,
        actor: dict[str, Any],
        actor_user_id: UUID,
        request: McpJsonRpcRequest,
        token_auth: McpTokenAuthContext | None,
    ) -> Iterator[dict[str, Any]]:
        try:
            result = self._dispatch_non_stream(
                actor=actor,
                actor_user_id=actor_user_id,
                request=request,
                token_auth=token_auth,
            )
            yield self._success(request.id, result)
        except McpRpcError as exc:
            yield self._error(
                request.id,
                code=exc.code,
                message=exc.message,
                data=exc.data,
            )
        except ValidationError as exc:
            yield self._error(
                request.id,
                code=-32602,
                message="Invalid params",
                data={"errors": exc.errors()},
            )
        except HTTPException as exc:
            # Surface REST-style validation/authorization failures as structured
            # MCP errors without leaking transport-specific status handling.
            error_code = -32003 if exc.status_code in {401, 403} else -32602
            yield self._error(
                request.id,
                code=error_code,
                message="Invalid params" if exc.status_code < 500 else "Internal MCP error",
                data={"status_code": exc.status_code, "detail": str(exc.detail)},
            )
        except ChatProviderExecutionError as exc:
            yield self._error(
                request.id,
                code=-32020,
                message=exc.detail,
                data={"error_code": exc.error_code, "status_code": exc.status_code},
            )
        except ChatServiceError as exc:
            yield self._error(
                request.id,
                code=-32010,
                message=exc.detail,
                data={"status_code": exc.status_code},
            )
        except Exception as exc:
            yield self._error(
                request.id,
                code=-32000,
                message="Internal MCP error",
                data={"detail": str(exc)},
            )

    def stream_call(
        self,
        *,
        actor: dict[str, Any],
        request: McpJsonRpcRequest,
        token_auth: McpTokenAuthContext | None = None,
    ):
        if request.jsonrpc != "2.0":
            yield self._error(
                request.id,
                code=-32600,
                message="Invalid Request",
                data={"jsonrpc": request.jsonrpc},
            )
            return

        try:
            actor_user_id = UUID(str(actor["user_id"]))
        except Exception:
            yield self._error(
                request.id,
                code=-32001,
                message="Unauthorized actor context",
                data={"user_id": actor.get("user_id")},
            )
            return

        canonical_method = self._canonical_tool_name(request.method)
        request_ctx = _StreamRouteRequest(
            actor_user_id=actor_user_id,
            request_id=request.id,
            request_method=request.method,
            canonical_method=canonical_method,
            request_params=request.params,
            token_auth=token_auth,
        )
        handled, direct_frames = self._maybe_stream_direct_chat_send_message(
            request_ctx=request_ctx,
        )
        if handled:
            assert direct_frames is not None
            yield from direct_frames
            return

        handled, tool_call_frames = self._maybe_stream_tools_call_chat_message(
            request_ctx=request_ctx,
        )
        if handled:
            assert tool_call_frames is not None
            yield from tool_call_frames
            return

        yield from self._stream_dispatch_non_stream_result(
            actor=actor,
            actor_user_id=actor_user_id,
            request=request,
            token_auth=token_auth,
        )

    def handle_notification(
        self,
        *,
        request: McpJsonRpcRequest,
    ) -> None:
        """Best-effort handling for JSON-RPC notifications.

        Streamable HTTP clients (including VS Code MCP) can send lifecycle
        notifications such as `notifications/initialized` without an `id`.
        We currently do not require side effects for these notifications, so we
        accept and ignore them to maintain protocol compatibility.
        """
        _ = request

    def _stream_chat_send_message(
        self,
        *,
        request: _StreamChatSendMessageRequest,
    ):
        # This method emits progress frames (`mcp.event`) plus a final success/error
        # JSON-RPC frame. `as_tool_call=True` wraps the final success payload in the
        # `tools/call` envelope so external MCP clients get a consistent shape.
        try:
            session_id = self._parse_uuid(request.params, "session_id")
            payload = ChatMessageCreateRequest(content_text=request.params.get("content_text", ""))
            stream_enabled = bool(request.params.get("stream", True))
            if not stream_enabled:
                response = self._chat_service.send_message(
                    actor_user_id=request.actor_user_id,
                    session_id=session_id,
                    payload=payload,
                )
                yield chat_send_message_success_frame(
                    request_id=request.request_id,
                    tool_name=request.tool_name,
                    payload={"message": response.model_dump(mode="json")},
                    as_tool_call=request.as_tool_call,
                    success=self._success,
                    tool_call_success=self._tool_call_success,
                )
                return

            final_message = yield from stream_chat_send_message_events(
                chat_service=self._chat_service,
                actor_user_id=request.actor_user_id,
                session_id=session_id,
                payload=payload,
                request_id=request.request_id,
                tool_name=request.tool_name,
                event=self._event,
            )
            yield chat_send_message_success_frame(
                request_id=request.request_id,
                tool_name=request.tool_name,
                payload={"message": final_message},
                as_tool_call=request.as_tool_call,
                success=self._success,
                tool_call_success=self._tool_call_success,
            )
        except Exception as exc:
            yield stream_chat_send_message_error_frame(
                request_id=request.request_id,
                exc=exc,
                error=self._error,
            )
