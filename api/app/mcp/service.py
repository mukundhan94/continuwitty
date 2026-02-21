from __future__ import annotations

import json
from dataclasses import dataclass
from typing import Any
from uuid import UUID

from app.chat.service import ChatService
from app.ingestion.service import DocumentIngestionService
from app.mcp_tokens import McpTokenAuthContext
from app.memory_admin import MemoryAdminService
from app.models import (
    AdminEngramMoveRequest,
    AdminEngramUpdateRequest,
    EngramCollectionCreateRequest,
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
    ChatSessionLifecycleDispatchContext,
    ChatSessionLifecycleDispatchDependencies,
    dispatch_chat_pinning_tool,
    dispatch_chat_primary_tool,
    dispatch_chat_session_lifecycle_tool,
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
from .service_access import McpServiceAccessMixin
from .service_stream import (
    McpServiceStreamMixin,
    _StreamChatSendMessageRequest,
)
from .token_authorization import (
    TokenAuthorizationDependencies,
    enforce_token_authorization,
    project_id_for_tool,
    resolve_project_for_write,
    visible_tool_catalog,
)

__all__ = ["McpService", "_ToolDispatchContext", "_StreamChatSendMessageRequest"]


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


class McpService(McpServiceAccessMixin, McpServiceStreamMixin):
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

        session_lifecycle_result = dispatch_chat_session_lifecycle_tool(
            dependencies=ChatSessionLifecycleDispatchDependencies(
                memory_admin_service=self._memory_admin_service,
                parse_uuid=self._parse_uuid,
                require_session_access=self._require_session_access,
            ),
            context=ChatSessionLifecycleDispatchContext(
                actor=context.actor,
                actor_user_id=context.actor_user_id,
                method=context.method,
                params=context.params,
            ),
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
