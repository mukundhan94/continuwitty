from __future__ import annotations

import json
from collections.abc import Iterator
from datetime import UTC, datetime
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
    AdminEngramDeleteRequest,
    AdminEngramMoveRequest,
    AdminEngramRecord,
    AdminEngramUpdateRequest,
    AdminSessionDeleteRequest,
    ChatLifecyclePolicyUpdateRequest,
    ChatMessageCreateRequest,
    ChatSessionCreateRequest,
    ContinueSessionRequest,
    EngramCollectionCreateRequest,
    EngramCollectionDeleteRequest,
    EngramCollectionItemsUpdateRequest,
    EngramCollectionRecord,
    EngramCollectionUpdateRequest,
    EngramCreateFromConversationRequest,
    EngramQueryRequest,
    McpJsonRpcRequest,
    MemoryEngramCreate,
    PinDocumentRequest,
    PinEngramRequest,
    ProjectCreateRequest,
    SaveSessionAsEngramRequest,
)
from app.projects import ProjectService
from app.repository import (
    create_engram,
    create_engram_with_report,
    get_rehydration_bundle,
    list_engrams,
    query_engrams,
)

from .catalog import (
    _ENGRAM_SCOPED_TOOLS,
    _OPTIONAL_PROJECT_TOOLS,
    _PROJECT_FALLBACK_TOOLS,
    _READ_TOOL_NAMES,
    _SESSION_SCOPED_TOOLS,
    _TOOL_ALIASES,
    _WRITE_TOOL_NAMES,
    _to_dotted_tool_name,
    _to_public_tool_name,
    build_tool_catalog,
)
from .errors import McpRpcError

_COLLECTION_SCOPED_TOOLS = {
    "engram.collection_update",
    "engram.collection_delete",
    "engram.collection_add_items",
    "engram.collection_remove_items",
}


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
    def _projects_for_user(actor_user_id: UUID, chat_service: ChatService) -> list[str]:
        project_ids: set[str] = set()
        for item in list_engrams(limit=1000, offset=0, actor_user_id=actor_user_id):
            project_ids.add(item.project_id)
        for item in chat_service.list_sessions(
            actor_user_id=actor_user_id,
            project_id=None,
            limit=1000,
            offset=0,
        ):
            project_ids.add(item.project_id)
        return sorted(project_ids)

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

    @staticmethod
    def _normalize_project_id(value: str | None) -> str | None:
        normalized = (value or "").strip()
        return normalized or None

    @staticmethod
    def _single_allowed_token_project_id(token_auth: McpTokenAuthContext | None) -> str | None:
        if token_auth is None or not token_auth.allowed_project_ids:
            return None
        if len(token_auth.allowed_project_ids) > 1:
            raise McpRpcError(
                code=-32602,
                message="Invalid params",
                data={"missing": "project_id", "reason": "token_has_multiple_allowed_projects"},
            )
        return next(iter(token_auth.allowed_project_ids))

    def _resolve_project_input_for_write(
        self,
        *,
        requested_project_id: str | None,
        token_auth: McpTokenAuthContext | None,
    ) -> tuple[str | None, bool]:
        explicit_project_id = self._normalize_project_id(requested_project_id)
        if explicit_project_id is not None:
            return explicit_project_id, False
        token_project_id = self._single_allowed_token_project_id(token_auth)
        if token_project_id is not None:
            return token_project_id, False
        return None, True

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
        resolved_project_input, used_default_project = self._resolve_project_input_for_write(
            requested_project_id=requested_project_id,
            token_auth=token_auth,
        )

        try:
            resolution = self._project_service.resolve_project_id_for_write(
                actor_user_id=actor_user_id,
                actor_role=actor_role,
                project_id=resolved_project_input,
            )
        except HTTPException as exc:
            raise McpRpcError(
                code=-32602,
                message="Invalid params",
                data={"detail": str(exc.detail)},
            ) from exc
        return resolution.project_id, (used_default_project and resolution.used_default_project)

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
        actor_user_id: UUID,
        actor_role: str,
        token_auth: McpTokenAuthContext | None,
        params: dict[str, Any],
        enrichment_origin: str,
    ) -> tuple[dict[str, Any], dict[str, Any]]:
        """Create an engram from raw conversation text (no chat session required)."""
        request = EngramCreateFromConversationRequest(**params)
        resolved_payload, resolved_project_id, used_default_project = (
            self._create_engram_payload_with_project_resolution(
                actor_user_id=actor_user_id,
                actor_role=actor_role,
                token_auth=token_auth,
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
            owner_user_id=actor_user_id,
            enrichment_origin=enrichment_origin,
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

    @staticmethod
    def _tool_catalog() -> list[dict[str, Any]]:
        return build_tool_catalog()

    def _required_scope_for_tool(self, tool_name: str) -> str:
        canonical = self._canonical_tool_name(tool_name)
        if canonical in _WRITE_TOOL_NAMES:
            return "write"
        if canonical in _READ_TOOL_NAMES:
            return "read"
        return "read"

    def _allowed_canonical_tools(self, allowed_tools: set[str] | None) -> set[str]:
        if not allowed_tools:
            return set()
        return {self._canonical_tool_name(item) for item in allowed_tools}

    def _is_tool_allowed_by_token_policy(
        self,
        *,
        canonical_tool: str,
        allowed_tools: set[str] | None,
        allowed_canonical: set[str],
    ) -> bool:
        if not allowed_tools:
            return True
        return canonical_tool in allowed_canonical

    @staticmethod
    def _public_tool_catalog_entry(item: dict[str, Any]) -> dict[str, Any]:
        return {**item, "name": _to_public_tool_name(item["name"])}

    def _visible_tool_catalog_entry(
        self,
        *,
        item: dict[str, Any],
        token_scope: str,
        allowed_tools: set[str] | None,
        allowed_canonical: set[str],
    ) -> dict[str, Any] | None:
        canonical_name = item["name"]
        required_scope = self._required_scope_for_tool(canonical_name)
        if token_scope == "read" and required_scope == "write":
            return None
        if not self._is_tool_allowed_by_token_policy(
            canonical_tool=canonical_name,
            allowed_tools=allowed_tools,
            allowed_canonical=allowed_canonical,
        ):
            return None
        return self._public_tool_catalog_entry(item)

    def _visible_tool_catalog(self, token_auth: McpTokenAuthContext | None) -> list[dict[str, Any]]:
        if token_auth is None:
            return [self._public_tool_catalog_entry(item) for item in self._tool_catalog()]

        allowed_tools = token_auth.allowed_tools
        allowed_canonical = self._allowed_canonical_tools(allowed_tools)

        visible: list[dict[str, Any]] = []
        for item in self._tool_catalog():
            entry = self._visible_tool_catalog_entry(
                item=item,
                token_scope=token_auth.scope,
                allowed_tools=allowed_tools,
                allowed_canonical=allowed_canonical,
            )
            if entry is not None:
                visible.append(entry)
        return visible

    def _resolve_project_from_session(self, *, session_id: UUID) -> str | None:
        session = self._memory_admin_service.get_session(
            session_id=session_id, include_deleted=True
        )
        return session.project_id if session else None

    def _resolve_project_from_engram(
        self, *, engram_id: UUID, target_project_id: str | None = None
    ) -> str | None:
        normalized_target = self._normalize_project_id(target_project_id)
        if normalized_target:
            return normalized_target
        engram = self._memory_admin_service.find_engram(engram_id=engram_id, include_deleted=True)
        return engram.project_id if engram else None

    def _project_id_from_input_params(self, params: dict[str, Any]) -> str | None:
        raw_project = params.get("project_id")
        return self._normalize_project_id(str(raw_project) if raw_project else None)

    def _project_id_for_chat_save_as_engram(
        self, *, actor_user_id: UUID, params: dict[str, Any]
    ) -> str | None:
        raw_session_id = params.get("session_id")
        if raw_session_id:
            session = self._chat_service.get_session(
                actor_user_id=actor_user_id,
                session_id=self._parse_uuid(params, "session_id"),
            )
            return session.project_id
        return self._project_id_from_input_params(params)

    def _project_id_for_engram_scoped_tool(
        self, *, canonical_tool: str, params: dict[str, Any]
    ) -> str | None:
        target_project_id: str | None = None
        if canonical_tool == "engram.move_project":
            raw_target = params.get("target_project_id")
            target_project_id = str(raw_target) if raw_target is not None else None
        return self._resolve_project_from_engram(
            engram_id=self._parse_uuid(params, "engram_id"),
            target_project_id=target_project_id,
        )

    def _project_id_for_collection_scoped_tool(self, *, params: dict[str, Any]) -> str | None:
        collection = self._memory_admin_service.find_collection(
            collection_id=self._parse_uuid(params, "collection_id"),
            include_deleted=True,
        )
        return collection.project_id if collection else None

    def _project_id_for_rehydrate_tool(
        self, *, actor_user_id: UUID, params: dict[str, Any]
    ) -> str | None:
        bundle = get_rehydration_bundle(
            self._parse_uuid(params, "engram_id"),
            actor_user_id=actor_user_id,
        )
        return bundle.project_id if bundle else None

    def _project_id_for_tool(
        self,
        *,
        actor_user_id: UUID,
        tool_name: str,
        params: dict[str, Any],
    ) -> str | None:
        canonical_tool = self._canonical_tool_name(tool_name)
        project_input_tools = {
            "chat.create_session",
            "engram.create",
            "engram.create_from_conversation",
            "project.create",
            "project.set_default",
            "engram.collection_create",
        }

        if canonical_tool in project_input_tools:
            return self._project_id_from_input_params(params)

        if canonical_tool in _SESSION_SCOPED_TOOLS:
            return self._resolve_project_from_session(
                session_id=self._parse_uuid(params, "session_id"),
            )

        if canonical_tool == "chat.save_as_engram":
            return self._project_id_for_chat_save_as_engram(
                actor_user_id=actor_user_id,
                params=params,
            )

        if canonical_tool in _ENGRAM_SCOPED_TOOLS:
            return self._project_id_for_engram_scoped_tool(
                canonical_tool=canonical_tool,
                params=params,
            )

        if canonical_tool in _COLLECTION_SCOPED_TOOLS:
            return self._project_id_for_collection_scoped_tool(params=params)

        if canonical_tool == "engram.rehydrate":
            return self._project_id_for_rehydrate_tool(
                actor_user_id=actor_user_id,
                params=params,
            )

        if canonical_tool in _OPTIONAL_PROJECT_TOOLS:
            return self._project_id_from_input_params(params)

        return None

    def _needs_project_autofill_for_token(
        self, canonical_tool: str, params: dict[str, Any]
    ) -> bool:
        if canonical_tool in _OPTIONAL_PROJECT_TOOLS:
            return True
        if canonical_tool in _PROJECT_FALLBACK_TOOLS:
            if canonical_tool == "chat.save_as_engram":
                # Session snapshot mode infers project from the session itself.
                return not bool(params.get("session_id"))
            return True
        return False

    def _enforce_single_allowed_project_autofill(
        self,
        *,
        canonical_tool: str,
        normalized_params: dict[str, Any],
        allowed_projects: set[str],
    ) -> dict[str, Any]:
        if not self._needs_project_autofill_for_token(canonical_tool, normalized_params):
            return normalized_params
        if len(allowed_projects) > 1:
            raise McpRpcError(
                code=-32602,
                message="Invalid params",
                data={"missing": "project_id", "reason": "token_has_multiple_allowed_projects"},
            )
        normalized_params.setdefault("project_id", next(iter(allowed_projects)))
        return normalized_params

    @staticmethod
    def _token_error_data(
        *,
        tool_name: str,
        required_scope: str,
        token_scope: str,
        project_id: str | None = None,
    ) -> dict[str, Any]:
        data: dict[str, Any] = {
            "tool": tool_name,
            "required_scope": required_scope,
            "token_scope": token_scope,
        }
        if project_id:
            data["project_id"] = project_id
        return data

    def _enforce_token_scope(
        self,
        *,
        tool_name: str,
        token_scope: str,
        required_scope: str,
    ) -> None:
        if token_scope == "read" and required_scope == "write":
            raise McpRpcError(
                code=-32003,
                message="Token scope does not allow this tool",
                data=self._token_error_data(
                    tool_name=tool_name,
                    required_scope=required_scope,
                    token_scope=token_scope,
                ),
            )

    def _enforce_token_tool_allowlist(
        self,
        *,
        token_auth: McpTokenAuthContext,
        tool_name: str,
        canonical_tool: str,
        required_scope: str,
    ) -> None:
        allowed_tools = token_auth.allowed_tools
        allowed_canonical = self._allowed_canonical_tools(allowed_tools)
        if self._is_tool_allowed_by_token_policy(
            canonical_tool=canonical_tool,
            allowed_tools=allowed_tools,
            allowed_canonical=allowed_canonical,
        ):
            return
        raise McpRpcError(
            code=-32003,
            message="Tool not allowed by token policy",
            data=self._token_error_data(
                tool_name=tool_name,
                required_scope=required_scope,
                token_scope=token_auth.scope,
            ),
        )

    def _enforce_token_project_allowlist(
        self,
        *,
        actor_user_id: UUID,
        token_auth: McpTokenAuthContext,
        tool_name: str,
        canonical_tool: str,
        required_scope: str,
        params: dict[str, Any],
    ) -> dict[str, Any]:
        normalized_params = dict(params)
        allowed_projects = token_auth.allowed_project_ids
        if not allowed_projects:
            return normalized_params

        project_id = self._project_id_for_tool(
            actor_user_id=actor_user_id,
            tool_name=canonical_tool,
            params=normalized_params,
        )

        if not project_id:
            normalized_params = self._enforce_single_allowed_project_autofill(
                canonical_tool=canonical_tool,
                normalized_params=normalized_params,
                allowed_projects=allowed_projects,
            )
            project_id = self._project_id_for_tool(
                actor_user_id=actor_user_id,
                tool_name=canonical_tool,
                params=normalized_params,
            )

        if project_id and project_id not in allowed_projects:
            raise McpRpcError(
                code=-32003,
                message="Project not allowed by token policy",
                data=self._token_error_data(
                    tool_name=tool_name,
                    required_scope=required_scope,
                    token_scope=token_auth.scope,
                    project_id=project_id,
                ),
            )
        return normalized_params

    def _enforce_token_authorization(
        self,
        *,
        actor_user_id: UUID,
        token_auth: McpTokenAuthContext | None,
        tool_name: str,
        params: dict[str, Any],
    ) -> dict[str, Any]:
        if token_auth is None:
            return params

        if tool_name in {"initialize", "tools/list"}:
            return params

        canonical_tool = self._canonical_tool_name(tool_name)
        required_scope = self._required_scope_for_tool(canonical_tool)
        self._enforce_token_scope(
            tool_name=tool_name,
            token_scope=token_auth.scope,
            required_scope=required_scope,
        )
        self._enforce_token_tool_allowlist(
            token_auth=token_auth,
            tool_name=tool_name,
            canonical_tool=canonical_tool,
            required_scope=required_scope,
        )
        return self._enforce_token_project_allowlist(
            actor_user_id=actor_user_id,
            token_auth=token_auth,
            tool_name=tool_name,
            canonical_tool=canonical_tool,
            required_scope=required_scope,
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
            actor_user_id=actor_user_id,
            actor_role=actor_role,
            token_auth=token_auth,
            params=fallback_params,
            enrichment_origin="mcp.chat.save_as_engram",
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

    def _dispatch_chat_pinning_tool(
        self,
        *,
        actor_user_id: UUID,
        method: str,
        params: dict[str, Any],
    ) -> dict[str, Any] | None:
        normalized_method = "chat.pin_engram" if method == "engram.pin_to_session" else method

        def _list_pinned_engrams() -> dict[str, Any]:
            pinned = self._chat_service.list_pinned_engrams(
                actor_user_id=actor_user_id,
                session_id=self._parse_uuid(params, "session_id"),
            )
            return {"pinned_engrams": [item.model_dump(mode="json") for item in pinned]}

        def _pin_engram() -> dict[str, Any]:
            pinned = self._chat_service.pin_engram(
                actor_user_id=actor_user_id,
                session_id=self._parse_uuid(params, "session_id"),
                payload=PinEngramRequest(engram_id=self._parse_uuid(params, "engram_id")),
            )
            return {"pinned": pinned.model_dump(mode="json")}

        def _unpin_engram() -> dict[str, Any]:
            self._chat_service.unpin_engram(
                actor_user_id=actor_user_id,
                session_id=self._parse_uuid(params, "session_id"),
                engram_id=self._parse_uuid(params, "engram_id"),
            )
            return {"removed": True}

        def _list_pinned_documents() -> dict[str, Any]:
            pinned = self._chat_service.list_pinned_documents(
                actor_user_id=actor_user_id,
                session_id=self._parse_uuid(params, "session_id"),
            )
            return {"pinned_documents": [item.model_dump(mode="json") for item in pinned]}

        def _pin_document() -> dict[str, Any]:
            pinned = self._chat_service.pin_document(
                actor_user_id=actor_user_id,
                session_id=self._parse_uuid(params, "session_id"),
                payload=PinDocumentRequest(document_id=self._parse_uuid(params, "document_id")),
            )
            return {"pinned": pinned.model_dump(mode="json")}

        def _unpin_document() -> dict[str, Any]:
            self._chat_service.unpin_document(
                actor_user_id=actor_user_id,
                session_id=self._parse_uuid(params, "session_id"),
                document_id=self._parse_uuid(params, "document_id"),
            )
            return {"removed": True}

        handlers = {
            "chat.list_pinned_engrams": _list_pinned_engrams,
            "chat.pin_engram": _pin_engram,
            "chat.unpin_engram": _unpin_engram,
            "chat.list_pinned_documents": _list_pinned_documents,
            "chat.pin_document": _pin_document,
            "chat.unpin_document": _unpin_document,
        }
        handler = handlers.get(normalized_method)
        return handler() if handler else None

    def _dispatch_chat_session_query_tool(
        self,
        *,
        actor_user_id: UUID,
        method: str,
        params: dict[str, Any],
    ) -> dict[str, Any] | None:
        def _list_sessions() -> dict[str, Any]:
            sessions = self._chat_service.list_sessions(
                actor_user_id=actor_user_id,
                project_id=params.get("project_id"),
                limit=int(params.get("limit", 50)),
                offset=int(params.get("offset", 0)),
            )
            return {"sessions": [item.model_dump(mode="json") for item in sessions]}

        def _get_session() -> dict[str, Any]:
            session = self._chat_service.get_session(
                actor_user_id=actor_user_id,
                session_id=self._parse_uuid(params, "session_id"),
            )
            return {"session": session.model_dump(mode="json")}

        def _get_lifecycle_policy() -> dict[str, Any]:
            policy = self._chat_service.get_lifecycle_policy(
                actor_user_id=actor_user_id,
                session_id=self._parse_uuid(params, "session_id"),
            )
            return {"lifecycle_policy": policy.model_dump(mode="json")}

        def _update_lifecycle_policy() -> dict[str, Any]:
            policy = self._chat_service.update_lifecycle_policy(
                actor_user_id=actor_user_id,
                session_id=self._parse_uuid(params, "session_id"),
                payload=ChatLifecyclePolicyUpdateRequest(
                    autosave_enabled=params.get("autosave_enabled"),
                    autosave_strategy=params.get("autosave_strategy"),
                    autosave_interval_minutes=params.get("autosave_interval_minutes"),
                    autosave_min_messages=params.get("autosave_min_messages"),
                    retention_days=params.get("retention_days"),
                    retention_max_snapshots=params.get("retention_max_snapshots"),
                ),
            )
            return {"lifecycle_policy": policy.model_dump(mode="json")}

        def _list_messages() -> dict[str, Any]:
            messages = self._chat_service.list_messages(
                actor_user_id=actor_user_id,
                session_id=self._parse_uuid(params, "session_id"),
                limit=int(params.get("limit", 200)),
                offset=int(params.get("offset", 0)),
            )
            return {"messages": [item.model_dump(mode="json") for item in messages]}

        def _list_timeline() -> dict[str, Any]:
            events = self._chat_service.list_timeline_events(
                actor_user_id=actor_user_id,
                session_id=self._parse_uuid(params, "session_id"),
                limit=int(params.get("limit", 100)),
                offset=int(params.get("offset", 0)),
            )
            return {"events": [item.model_dump(mode="json") for item in events]}

        handlers = {
            "chat.list_sessions": _list_sessions,
            "chat.get_session": _get_session,
            "chat.get_lifecycle_policy": _get_lifecycle_policy,
            "chat.update_lifecycle_policy": _update_lifecycle_policy,
            "chat.list_messages": _list_messages,
            "chat.list_timeline": _list_timeline,
        }
        handler = handlers.get(method)
        return handler() if handler else None

    def _dispatch_chat_primary_tool(
        self,
        *,
        actor_user_id: UUID,
        actor_role: str,
        method: str,
        params: dict[str, Any],
        token_auth: McpTokenAuthContext | None,
    ) -> dict[str, Any] | None:
        def _list_project_documents() -> dict[str, Any]:
            if self._ingestion_service is None:
                raise McpRpcError(
                    code=-32000,
                    message="Ingestion service unavailable for MCP tool",
                    data={"method": method},
                )
            documents = self._ingestion_service.list_documents(
                actor_user_id=actor_user_id,
                project_id=params.get("project_id"),
                limit=int(params.get("limit", 200)),
                offset=int(params.get("offset", 0)),
            )
            return {"documents": [item.model_dump(mode="json") for item in documents]}

        handlers = {
            "chat.create_session": lambda: {
                "session": self._chat_service.create_session(
                    actor_user_id=actor_user_id,
                    payload=ChatSessionCreateRequest(**params),
                ).model_dump(mode="json")
            },
            "chat.send_message": lambda: {
                "message": self._chat_service.send_message(
                    actor_user_id=actor_user_id,
                    session_id=self._parse_uuid(params, "session_id"),
                    payload=ChatMessageCreateRequest(content_text=params.get("content_text", "")),
                ).model_dump(mode="json")
            },
            "chat.list_project_documents": _list_project_documents,
            "chat.save_as_engram": lambda: self._dispatch_chat_save_as_engram_tool(
                actor_user_id=actor_user_id,
                actor_role=actor_role,
                params=params,
                token_auth=token_auth,
            ),
            "chat.continue_session": lambda: {
                "continuation": self._chat_service.continue_session(
                    actor_user_id=actor_user_id,
                    session_id=self._parse_uuid(params, "session_id"),
                    payload=ContinueSessionRequest(title=params.get("title")),
                ).model_dump(mode="json")
            },
        }
        handler = handlers.get(method)
        return handler() if handler else None

    def _dispatch_chat_tool(
        self,
        *,
        actor: dict[str, Any],
        actor_user_id: UUID,
        method: str,
        params: dict[str, Any],
        token_auth: McpTokenAuthContext | None,
    ) -> dict[str, Any] | None:
        actor_role = str(actor.get("role", ""))

        chat_session_query_result = self._dispatch_chat_session_query_tool(
            actor_user_id=actor_user_id,
            method=method,
            params=params,
        )
        if chat_session_query_result is not None:
            return chat_session_query_result

        chat_pinning_result = self._dispatch_chat_pinning_tool(
            actor_user_id=actor_user_id,
            method=method,
            params=params,
        )
        if chat_pinning_result is not None:
            return chat_pinning_result

        chat_primary_result = self._dispatch_chat_primary_tool(
            actor_user_id=actor_user_id,
            actor_role=actor_role,
            method=method,
            params=params,
            token_auth=token_auth,
        )
        if chat_primary_result is not None:
            return chat_primary_result

        session_lifecycle_result = self._dispatch_chat_session_lifecycle_tool(
            actor=actor,
            actor_user_id=actor_user_id,
            method=method,
            params=params,
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

    def _dispatch_engram_collection_mutation_tool(
        self,
        *,
        actor: dict[str, Any],
        actor_user_id: UUID,
        method: str,
        params: dict[str, Any],
    ) -> dict[str, Any] | None:
        collection_mutation_tools = {
            "engram.collection_update",
            "engram.collection_delete",
            "engram.collection_add_items",
            "engram.collection_remove_items",
        }
        if method not in collection_mutation_tools:
            return None

        collection_id = self._parse_uuid(params, "collection_id")
        self._require_collection_access(actor=actor, collection_id=collection_id)

        if method == "engram.collection_update":
            updated_collection = self._memory_admin_service.update_collection(
                collection_id=collection_id,
                payload=EngramCollectionUpdateRequest(
                    name=params.get("name"),
                    description=params.get("description"),
                    expected_updated_at=params.get("expected_updated_at"),
                ),
            )
            return {"collection": updated_collection.model_dump(mode="json")}

        if method == "engram.collection_delete":
            result = self._memory_admin_service.delete_collection(
                collection_id=collection_id,
                actor_user_id=actor_user_id,
                payload=EngramCollectionDeleteRequest(reason=params.get("reason")),
            )
            return {"result": result}

        if method == "engram.collection_add_items":
            result = self._memory_admin_service.add_collection_items(
                collection_id=collection_id,
                actor_user_id=actor_user_id,
                payload=EngramCollectionItemsUpdateRequest(
                    engram_ids=self._parse_uuid_list(params, "engram_ids")
                ),
            )
            return {"result": result}

        result = self._memory_admin_service.remove_collection_item(
            collection_id=collection_id,
            engram_id=self._parse_uuid(params, "engram_id"),
        )
        return {"result": result}

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

    def _dispatch_engram_state_mutation_tool(
        self,
        *,
        actor: dict[str, Any],
        actor_user_id: UUID,
        method: str,
        params: dict[str, Any],
    ) -> dict[str, Any] | None:
        if method not in {"engram.delete", "engram.restore"}:
            return None

        engram_id = self._parse_uuid(params, "engram_id")
        self._require_engram_access(actor=actor, engram_id=engram_id, include_deleted=True)

        if method == "engram.delete":
            deleted = self._memory_admin_service.delete_engram(
                engram_id=engram_id,
                actor_user_id=actor_user_id,
                payload=AdminEngramDeleteRequest(reason=params.get("reason")),
            )
            return {"result": deleted.model_dump(mode="json")}

        restored = self._memory_admin_service.restore_engram(engram_id=engram_id)
        return {"result": restored.model_dump(mode="json")}

    def _dispatch_engram_mutation_tool(
        self,
        *,
        actor: dict[str, Any],
        actor_user_id: UUID,
        method: str,
        params: dict[str, Any],
    ) -> dict[str, Any] | None:
        collection_mutation_result = self._dispatch_engram_collection_mutation_tool(
            actor=actor,
            actor_user_id=actor_user_id,
            method=method,
            params=params,
        )
        if collection_mutation_result is not None:
            return collection_mutation_result
        return self._dispatch_engram_state_mutation_tool(
            actor=actor,
            actor_user_id=actor_user_id,
            method=method,
            params=params,
        )

    def _dispatch_engram_read_tool(
        self,
        *,
        actor: dict[str, Any],
        actor_user_id: UUID,
        method: str,
        params: dict[str, Any],
    ) -> dict[str, Any] | None:
        def _query() -> dict[str, Any]:
            results = query_engrams(
                request=EngramQueryRequest(**params),
                embedding_dim=self._embedding_dim,
                actor_user_id=actor_user_id,
            )
            return {"results": [item.model_dump(mode="json") for item in results]}

        def _rehydrate() -> dict[str, Any]:
            engram_id = self._parse_uuid(params, "engram_id")
            bundle = get_rehydration_bundle(engram_id, actor_user_id=actor_user_id)
            if not bundle:
                raise McpRpcError(
                    code=-32004,
                    message="Engram not found",
                    data={"engram_id": str(engram_id)},
                )
            return {"bundle": bundle.model_dump(mode="json")}

        def _list_engrams() -> dict[str, Any]:
            session_id_value = params.get("session_id")
            session_id = (
                self._parse_uuid({"session_id": session_id_value}, "session_id")
                if session_id_value is not None
                else None
            )
            listed = self._list_engrams_for_actor(
                actor=actor,
                actor_user_id=actor_user_id,
                session_id=session_id,
                params=params,
            )
            return {"engrams": [item.model_dump(mode="json") for item in listed]}

        def _get_engram() -> dict[str, Any]:
            engram_id = self._parse_uuid(params, "engram_id")
            include_deleted = bool(params.get("include_deleted", True))
            engram = self._require_engram_access(
                actor=actor,
                engram_id=engram_id,
                include_deleted=include_deleted,
            )
            return {"engram": engram.model_dump(mode="json")}

        def _list_collections() -> dict[str, Any]:
            collections = self._list_collections_for_actor(
                actor=actor,
                actor_user_id=actor_user_id,
                params=params,
            )
            return {"collections": [item.model_dump(mode="json") for item in collections]}

        handlers = {
            "engram.query": _query,
            "engram.rehydrate": _rehydrate,
            "engram.list": _list_engrams,
            "engram.get": _get_engram,
            "engram.collection_list": _list_collections,
        }
        handler = handlers.get(method)
        return handler() if handler else None

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
            actor_user_id=actor_user_id,
            actor_role=actor_role,
            token_auth=token_auth,
            params=params,
            enrichment_origin="mcp.engram.create_from_conversation",
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
        actor: dict[str, Any],
        actor_user_id: UUID,
        method: str,
        params: dict[str, Any],
        token_auth: McpTokenAuthContext | None,
    ) -> dict[str, Any] | None:
        actor_role = str(actor.get("role", ""))

        handlers = {
            "engram.create": lambda: self._dispatch_engram_create_tool(
                actor=actor,
                actor_user_id=actor_user_id,
                params=params,
                token_auth=token_auth,
            ),
            "engram.create_from_conversation": lambda: self._dispatch_engram_create_from_conversation_tool(
                actor=actor,
                actor_user_id=actor_user_id,
                params=params,
                token_auth=token_auth,
            ),
            "engram.update": lambda: self._dispatch_engram_update_tool(
                actor=actor,
                actor_user_id=actor_user_id,
                params=params,
                token_auth=token_auth,
            ),
            "engram.move_project": lambda: self._dispatch_engram_move_project_tool(
                actor=actor,
                actor_user_id=actor_user_id,
                params=params,
                token_auth=token_auth,
            ),
            "engram.collection_create": lambda: self._dispatch_engram_collection_create_tool(
                actor_user_id=actor_user_id,
                actor_role=actor_role,
                params=params,
                token_auth=token_auth,
            ),
        }
        handler = handlers.get(method)
        return handler() if handler else None

    def _dispatch_engram_tool(
        self,
        *,
        actor: dict[str, Any],
        actor_user_id: UUID,
        method: str,
        params: dict[str, Any],
        token_auth: McpTokenAuthContext | None,
    ) -> dict[str, Any] | None:
        engram_primary_result = self._dispatch_engram_primary_tool(
            actor=actor,
            actor_user_id=actor_user_id,
            method=method,
            params=params,
            token_auth=token_auth,
        )
        if engram_primary_result is not None:
            return engram_primary_result

        engram_read_result = self._dispatch_engram_read_tool(
            actor=actor,
            actor_user_id=actor_user_id,
            method=method,
            params=params,
        )
        if engram_read_result is not None:
            return engram_read_result

        mutation_result = self._dispatch_engram_mutation_tool(
            actor=actor,
            actor_user_id=actor_user_id,
            method=method,
            params=params,
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

    def _dispatch_project_tool(
        self,
        *,
        actor: dict[str, Any],
        actor_user_id: UUID,
        method: str,
        params: dict[str, Any],
    ) -> dict[str, Any] | None:
        actor_role = str(actor.get("role", ""))

        if method == "project.list":
            projects = self._project_service.list_projects(
                actor_user_id=actor_user_id,
                actor_role=actor_role,
                include_archived=bool(params.get("include_archived", False)),
                limit=int(params.get("limit", 500)),
                offset=int(params.get("offset", 0)),
            )
            return {"projects": [item.model_dump(mode="json") for item in projects]}

        if method == "project.create":
            try:
                payload = ProjectCreateRequest(
                    project_id=str(params.get("project_id", "")),
                    name=str(params.get("name", "")),
                    description=str(params.get("description", "")),
                    owner_user_id=(
                        UUID(str(params["owner_user_id"]))
                        if params.get("owner_user_id") is not None
                        else None
                    ),
                )
            except ValueError as exc:
                raise McpRpcError(
                    code=-32602,
                    message="Invalid params",
                    data={"invalid": "owner_user_id"},
                ) from exc
            created = self._project_service.create_project(
                actor_user_id=actor_user_id,
                actor_role=actor_role,
                payload=payload,
            )
            return {"project": created.model_dump(mode="json")}

        if method == "project.get_default":
            return {
                "default_project_id": self._project_service.get_default_project_id(
                    actor_user_id=actor_user_id
                )
            }

        if method == "project.set_default":
            project_id = self._project_service.set_default_project_id(
                actor_user_id=actor_user_id,
                actor_role=actor_role,
                project_id=str(params.get("project_id", "")),
            )
            return {"default_project_id": project_id}

        return None

    def _dispatch_user_tool(
        self,
        *,
        actor: dict[str, Any],
        actor_user_id: UUID,
        method: str,
    ) -> dict[str, Any] | None:
        if method == "user.get_profile":
            return {"profile": actor}
        if method == "user.list_projects":
            return {"project_ids": self._projects_for_user(actor_user_id, self._chat_service)}
        return None

    def _dispatch_tool(
        self,
        *,
        actor: dict[str, Any],
        actor_user_id: UUID,
        method: str,
        params: dict[str, Any],
        token_auth: McpTokenAuthContext | None,
    ) -> dict[str, Any]:
        canonical_method = self._canonical_tool_name(method)
        chat_result = self._dispatch_chat_tool(
            actor=actor,
            actor_user_id=actor_user_id,
            method=canonical_method,
            params=params,
            token_auth=token_auth,
        )
        if chat_result is not None:
            return chat_result

        engram_result = self._dispatch_engram_tool(
            actor=actor,
            actor_user_id=actor_user_id,
            method=canonical_method,
            params=params,
            token_auth=token_auth,
        )
        if engram_result is not None:
            return engram_result

        project_result = self._dispatch_project_tool(
            actor=actor,
            actor_user_id=actor_user_id,
            method=canonical_method,
            params=params,
        )
        if project_result is not None:
            return project_result

        user_result = self._dispatch_user_tool(
            actor=actor,
            actor_user_id=actor_user_id,
            method=canonical_method,
        )
        if user_result is not None:
            return user_result

        raise McpRpcError(
            code=-32601,
            message="Method not found",
            data={"method": canonical_method},
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
                actor=actor,
                actor_user_id=actor_user_id,
                method=canonical_tool_name,
                params=authorized_params,
                token_auth=token_auth,
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
            actor=actor,
            actor_user_id=actor_user_id,
            method=canonical_method,
            params=authorized_params,
            token_auth=token_auth,
        )

    def _authorize_tool_call(
        self,
        *,
        actor_user_id: UUID,
        request_id: str | int | None,
        token_auth: McpTokenAuthContext | None,
        tool_name: str,
        params: dict[str, Any],
    ) -> tuple[dict[str, Any] | None, dict[str, Any] | None]:
        try:
            authorized_params = self._enforce_token_authorization(
                actor_user_id=actor_user_id,
                token_auth=token_auth,
                tool_name=tool_name,
                params=params,
            )
        except McpRpcError as exc:
            return None, self._error(request_id, code=exc.code, message=exc.message, data=exc.data)
        return authorized_params, None

    def _maybe_stream_direct_chat_send_message(
        self,
        *,
        actor_user_id: UUID,
        request_id: str | int | None,
        request_method: str,
        canonical_method: str,
        request_params: dict[str, Any],
        token_auth: McpTokenAuthContext | None,
    ) -> tuple[bool, Iterator[dict[str, Any]] | None]:
        if canonical_method != "chat.send_message":
            return False, None
        authorized_params, error_frame = self._authorize_tool_call(
            actor_user_id=actor_user_id,
            request_id=request_id,
            token_auth=token_auth,
            tool_name=request_method,
            params=request_params,
        )
        if error_frame is not None:
            return True, iter((error_frame,))
        assert authorized_params is not None
        return (
            True,
            self._stream_chat_send_message(
                actor_user_id=actor_user_id,
                request_id=request_id,
                tool_name=request_method,
                params=authorized_params,
            ),
        )

    def _maybe_stream_tools_call_chat_message(
        self,
        *,
        actor_user_id: UUID,
        request_id: str | int | None,
        request_method: str,
        request_params: dict[str, Any],
        token_auth: McpTokenAuthContext | None,
    ) -> tuple[bool, Iterator[dict[str, Any]] | None]:
        if request_method != "tools/call":
            return False, None

        try:
            tool_name, tool_params = self._tool_name_and_params_for_tools_call(request_params)
            canonical_tool_name = self._canonical_tool_name(tool_name)
        except McpRpcError as exc:
            return (
                True,
                iter((self._error(request_id, code=exc.code, message=exc.message, data=exc.data),)),
            )

        authorized_tool_params, error_frame = self._authorize_tool_call(
            actor_user_id=actor_user_id,
            request_id=request_id,
            token_auth=token_auth,
            tool_name=tool_name,
            params=tool_params,
        )
        if error_frame is not None:
            return True, iter((error_frame,))
        assert authorized_tool_params is not None

        if canonical_tool_name == "chat.send_message" and bool(authorized_tool_params.get("stream", True)):
            return (
                True,
                self._stream_chat_send_message(
                    actor_user_id=actor_user_id,
                    request_id=request_id,
                    tool_name=tool_name,
                    params=authorized_tool_params,
                    as_tool_call=True,
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
        handled, direct_frames = self._maybe_stream_direct_chat_send_message(
            actor_user_id=actor_user_id,
            request_id=request.id,
            request_method=request.method,
            canonical_method=canonical_method,
            request_params=request.params,
            token_auth=token_auth,
        )
        if handled:
            assert direct_frames is not None
            yield from direct_frames
            return

        handled, tool_call_frames = self._maybe_stream_tools_call_chat_message(
            actor_user_id=actor_user_id,
            request_id=request.id,
            request_method=request.method,
            request_params=request.params,
            token_auth=token_auth,
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

    def _chat_send_message_success_frame(
        self,
        *,
        request_id: str | int,
        tool_name: str,
        payload: dict[str, Any],
        as_tool_call: bool,
    ) -> dict[str, Any]:
        if as_tool_call:
            return self._success(request_id, self._tool_call_success(tool_name, payload))
        return self._success(request_id, payload)

    @staticmethod
    def _stream_event_payload(event_payload: Any) -> dict[str, Any]:
        if isinstance(event_payload, dict):
            return event_payload
        return {"value": event_payload}

    def _stream_chat_send_message_events(
        self,
        *,
        actor_user_id: UUID,
        session_id: UUID,
        payload: ChatMessageCreateRequest,
        request_id: str | int,
        tool_name: str,
    ) -> Iterator[dict[str, Any]]:
        final_message: dict[str, Any] | None = None
        for event_name, event_payload in self._chat_service.stream_message_events(
            actor_user_id=actor_user_id,
            session_id=session_id,
            payload=payload,
        ):
            event_payload_dict = self._stream_event_payload(event_payload)
            yield self._event(
                request_id,
                tool=tool_name,
                event_name=event_name,
                event_payload=event_payload_dict,
            )
            if event_name == "error":
                raise McpRpcError(
                    code=-32020,
                    message=event_payload_dict.get("detail", "chat.send_message stream failed"),
                    data=event_payload_dict,
                )
            if event_name == "done":
                final_message = event_payload_dict
        if final_message is None:
            raise McpRpcError(
                code=-32021,
                message="chat.send_message stream ended without completion",
            )
        return final_message

    def _stream_chat_send_message_error_frame(
        self,
        *,
        request_id: str | int,
        exc: Exception,
    ) -> dict[str, Any]:
        if isinstance(exc, McpRpcError):
            return self._error(
                request_id,
                code=exc.code,
                message=exc.message,
                data=exc.data,
            )
        if isinstance(exc, ValidationError):
            return self._error(
                request_id,
                code=-32602,
                message="Invalid params",
                data={"errors": exc.errors()},
            )
        if isinstance(exc, ChatProviderExecutionError):
            return self._error(
                request_id,
                code=-32020,
                message=exc.detail,
                data={"error_code": exc.error_code, "status_code": exc.status_code},
            )
        if isinstance(exc, ChatServiceError):
            return self._error(
                request_id,
                code=-32010,
                message=exc.detail,
                data={"status_code": exc.status_code},
            )
        return self._error(
            request_id,
            code=-32000,
            message="Internal MCP error",
            data={"detail": str(exc), "timestamp": datetime.now(UTC).isoformat()},
        )

    def _stream_chat_send_message(
        self,
        *,
        actor_user_id: UUID,
        request_id: str | int,
        tool_name: str,
        params: dict[str, Any],
        as_tool_call: bool = False,
    ):
        # This method emits progress frames (`mcp.event`) plus a final success/error
        # JSON-RPC frame. `as_tool_call=True` wraps the final success payload in the
        # `tools/call` envelope so external MCP clients get a consistent shape.
        try:
            session_id = self._parse_uuid(params, "session_id")
            payload = ChatMessageCreateRequest(content_text=params.get("content_text", ""))
            stream_enabled = bool(params.get("stream", True))
            if not stream_enabled:
                response = self._chat_service.send_message(
                    actor_user_id=actor_user_id,
                    session_id=session_id,
                    payload=payload,
                )
                yield self._chat_send_message_success_frame(
                    request_id=request_id,
                    tool_name=tool_name,
                    payload={"message": response.model_dump(mode="json")},
                    as_tool_call=as_tool_call,
                )
                return

            final_message = yield from self._stream_chat_send_message_events(
                actor_user_id=actor_user_id,
                session_id=session_id,
                payload=payload,
                request_id=request_id,
                tool_name=tool_name,
            )
            yield self._chat_send_message_success_frame(
                request_id=request_id,
                tool_name=tool_name,
                payload={"message": final_message},
                as_tool_call=as_tool_call,
            )
        except Exception as exc:
            yield self._stream_chat_send_message_error_frame(
                request_id=request_id,
                exc=exc,
            )
