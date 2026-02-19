from __future__ import annotations

import json
from datetime import UTC, datetime
from typing import Any
from uuid import UUID

from fastapi import HTTPException
from pydantic import ValidationError

from app.chat.errors import ChatProviderExecutionError, ChatServiceError
from app.chat.service import ChatService
from app.ingestion.service import DocumentIngestionService
from app.mcp_tokens import McpTokenAuthContext
from app.memory_admin import MemoryAdminService
from app.models import (
    AdminEngramDeleteRequest,
    AdminEngramMoveRequest,
    AdminEngramUpdateRequest,
    AdminSessionDeleteRequest,
    ChatLifecyclePolicyUpdateRequest,
    ChatMessageCreateRequest,
    ChatSessionCreateRequest,
    ContinueSessionRequest,
    EngramCollectionCreateRequest,
    EngramCollectionDeleteRequest,
    EngramCollectionItemsUpdateRequest,
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

from .errors import McpRpcError

_READ_TOOL_NAMES = {
    "chat.list_sessions",
    "chat.get_session",
    "chat.get_lifecycle_policy",
    "chat.list_messages",
    "chat.list_timeline",
    "chat.list_pinned_engrams",
    "chat.list_pinned_documents",
    "chat.list_project_documents",
    "project.list",
    "project.get_default",
    "engram.list",
    "engram.get",
    "engram.collection_list",
    "engram.query",
    "engram.rehydrate",
    "user.get_profile",
    "user.list_projects",
}

_WRITE_TOOL_NAMES = {
    "chat.create_session",
    "chat.update_lifecycle_policy",
    "chat.send_message",
    "chat.pin_engram",
    "chat.unpin_engram",
    "chat.pin_document",
    "chat.unpin_document",
    "chat.save_as_engram",
    "chat.continue_session",
    "chat.delete_session",
    "chat.restore_session",
    "project.create",
    "project.set_default",
    "engram.create",
    "engram.create_from_conversation",
    "engram.pin_to_session",
    "engram.update",
    "engram.move_project",
    "engram.delete",
    "engram.restore",
    "engram.collection_create",
    "engram.collection_update",
    "engram.collection_delete",
    "engram.collection_add_items",
    "engram.collection_remove_items",
}

# Alias maps to existing implementation branch but should obey the same scope semantics.
_TOOL_ALIASES = {"engram.pin_to_session": "chat.pin_engram"}

_OPTIONAL_PROJECT_TOOLS = {
    "engram.query",
    "chat.list_sessions",
    "chat.list_project_documents",
    "engram.list",
    "engram.collection_list",
}

_PROJECT_FALLBACK_TOOLS = {
    "chat.save_as_engram",
    "engram.create",
    "engram.create_from_conversation",
    "engram.collection_create",
}

_TOOL_NAMESPACE_PREFIXES = ("chat", "engram", "project", "user")


def _to_public_tool_name(canonical_name: str) -> str:
    """Expose VS Code-compatible tool names (no dots)."""
    return canonical_name.replace(".", "_")


def _to_dotted_tool_name(tool_name: str) -> str:
    """Convert external tool names back to dotted canonical method names.

    We intentionally support both forms:
    - dotted (`chat.send_message`) for backward compatibility
    - underscore (`chat_send_message`) for strict MCP clients.
    """
    if "." in tool_name:
        return tool_name
    for namespace in _TOOL_NAMESPACE_PREFIXES:
        prefix = f"{namespace}_"
        if tool_name.startswith(prefix):
            return f"{namespace}.{tool_name[len(prefix) :]}"
    return tool_name


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
        explicit_project_id = self._normalize_project_id(requested_project_id)
        resolved_project_input = explicit_project_id
        used_default_project = False

        if not explicit_project_id and token_auth and token_auth.allowed_project_ids:
            if len(token_auth.allowed_project_ids) > 1:
                raise McpRpcError(
                    code=-32602,
                    message="Invalid params",
                    data={"missing": "project_id", "reason": "token_has_multiple_allowed_projects"},
                )
            resolved_project_input = next(iter(token_auth.allowed_project_ids))
        elif not explicit_project_id:
            used_default_project = True

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
        # Keep this catalog synchronized with dispatcher behavior and tests.
        # External MCP clients depend on stable tool names and input schemas.
        return [
            {
                "name": "chat.create_session",
                "description": "Create a chat session in a project.",
                "inputSchema": {
                    "type": "object",
                    "required": ["project_id", "title"],
                    "properties": {
                        "project_id": {"type": "string"},
                        "title": {"type": "string"},
                        "provider": {"type": "string"},
                        "model_id": {"type": "string"},
                        "system_prompt": {"type": "string"},
                        "visibility_scope": {"type": "string", "enum": ["private", "project"]},
                        "autosave_enabled": {"type": "boolean"},
                        "autosave_strategy": {
                            "type": "string",
                            "enum": ["off", "interval", "message_count"],
                        },
                        "autosave_interval_minutes": {"type": "integer", "minimum": 1},
                        "autosave_min_messages": {"type": "integer", "minimum": 1},
                        "retention_days": {"type": "integer", "minimum": 1},
                        "retention_max_snapshots": {"type": "integer", "minimum": 1},
                    },
                },
            },
            {
                "name": "chat.list_sessions",
                "description": "List chat sessions for the authenticated user.",
                "inputSchema": {
                    "type": "object",
                    "properties": {
                        "project_id": {"type": "string"},
                        "limit": {"type": "integer", "minimum": 1},
                        "offset": {"type": "integer", "minimum": 0},
                    },
                },
            },
            {
                "name": "chat.get_session",
                "description": "Get chat session metadata by session_id.",
                "inputSchema": {
                    "type": "object",
                    "required": ["session_id"],
                    "properties": {"session_id": {"type": "string", "format": "uuid"}},
                },
            },
            {
                "name": "chat.get_lifecycle_policy",
                "description": "Get autosave/retention lifecycle policy for a session.",
                "inputSchema": {
                    "type": "object",
                    "required": ["session_id"],
                    "properties": {"session_id": {"type": "string", "format": "uuid"}},
                },
            },
            {
                "name": "chat.update_lifecycle_policy",
                "description": "Update autosave/retention lifecycle policy for a session.",
                "inputSchema": {
                    "type": "object",
                    "required": ["session_id"],
                    "properties": {
                        "session_id": {"type": "string", "format": "uuid"},
                        "autosave_enabled": {"type": "boolean"},
                        "autosave_strategy": {
                            "type": "string",
                            "enum": ["off", "interval", "message_count"],
                        },
                        "autosave_interval_minutes": {"type": "integer", "minimum": 1},
                        "autosave_min_messages": {"type": "integer", "minimum": 1},
                        "retention_days": {"type": "integer", "minimum": 1},
                        "retention_max_snapshots": {"type": "integer", "minimum": 1},
                    },
                },
            },
            {
                "name": "chat.list_messages",
                "description": "List persisted messages for a chat session.",
                "inputSchema": {
                    "type": "object",
                    "required": ["session_id"],
                    "properties": {
                        "session_id": {"type": "string", "format": "uuid"},
                        "limit": {"type": "integer", "minimum": 1},
                        "offset": {"type": "integer", "minimum": 0},
                    },
                },
            },
            {
                "name": "chat.list_timeline",
                "description": "List timeline events (manual saves, autosaves, consolidation) for a session.",
                "inputSchema": {
                    "type": "object",
                    "required": ["session_id"],
                    "properties": {
                        "session_id": {"type": "string", "format": "uuid"},
                        "limit": {"type": "integer", "minimum": 1},
                        "offset": {"type": "integer", "minimum": 0},
                    },
                },
            },
            {
                "name": "chat.send_message",
                "description": "Send a user message in a chat session (streaming supported).",
                "inputSchema": {
                    "type": "object",
                    "required": ["session_id", "content_text"],
                    "properties": {
                        "session_id": {"type": "string", "format": "uuid"},
                        "content_text": {"type": "string"},
                        "stream": {"type": "boolean"},
                    },
                },
            },
            {
                "name": "chat.list_pinned_engrams",
                "description": "List engrams pinned to a chat session.",
                "inputSchema": {
                    "type": "object",
                    "required": ["session_id"],
                    "properties": {"session_id": {"type": "string", "format": "uuid"}},
                },
            },
            {
                "name": "chat.pin_engram",
                "description": "Pin an engram to a chat session context chain.",
                "inputSchema": {
                    "type": "object",
                    "required": ["session_id", "engram_id"],
                    "properties": {
                        "session_id": {"type": "string", "format": "uuid"},
                        "engram_id": {"type": "string", "format": "uuid"},
                    },
                },
            },
            {
                "name": "chat.unpin_engram",
                "description": "Remove a pinned engram from a chat session.",
                "inputSchema": {
                    "type": "object",
                    "required": ["session_id", "engram_id"],
                    "properties": {
                        "session_id": {"type": "string", "format": "uuid"},
                        "engram_id": {"type": "string", "format": "uuid"},
                    },
                },
            },
            {
                "name": "chat.list_pinned_documents",
                "description": "List uploaded documents pinned to a chat session.",
                "inputSchema": {
                    "type": "object",
                    "required": ["session_id"],
                    "properties": {"session_id": {"type": "string", "format": "uuid"}},
                },
            },
            {
                "name": "chat.pin_document",
                "description": "Pin an uploaded document into chat context.",
                "inputSchema": {
                    "type": "object",
                    "required": ["session_id", "document_id"],
                    "properties": {
                        "session_id": {"type": "string", "format": "uuid"},
                        "document_id": {"type": "string", "format": "uuid"},
                    },
                },
            },
            {
                "name": "chat.unpin_document",
                "description": "Remove a pinned document from chat context.",
                "inputSchema": {
                    "type": "object",
                    "required": ["session_id", "document_id"],
                    "properties": {
                        "session_id": {"type": "string", "format": "uuid"},
                        "document_id": {"type": "string", "format": "uuid"},
                    },
                },
            },
            {
                "name": "chat.list_project_documents",
                "description": "List ingested documents visible in a project scope.",
                "inputSchema": {
                    "type": "object",
                    "properties": {
                        "project_id": {"type": "string"},
                        "limit": {"type": "integer", "minimum": 1},
                        "offset": {"type": "integer", "minimum": 0},
                    },
                },
            },
            {
                "name": "chat.save_as_engram",
                "description": (
                    "Save as engram. Supports either a chat session snapshot "
                    "or direct conversation markdown when no session_id exists."
                ),
                "inputSchema": {
                    "type": "object",
                    "properties": {
                        "session_id": {"type": "string", "format": "uuid"},
                        "project_id": {"type": "string"},
                        "conversation_markdown": {"type": "string"},
                        "thread_id": {"type": "string"},
                        "title": {"type": "string"},
                        "abstract": {"type": "string"},
                        "visibility_scope": {"type": "string", "enum": ["private", "project"]},
                        "tags": {"type": "array", "items": {"type": "string"}},
                        "keywords": {"type": "array", "items": {"type": "string"}},
                        "retrieval_text": {"type": "string"},
                        "source_session_id": {"type": "string", "format": "uuid"},
                    },
                },
            },
            {
                "name": "chat.continue_session",
                "description": "Create a continued session carrying pinned engram context.",
                "inputSchema": {
                    "type": "object",
                    "required": ["session_id"],
                    "properties": {
                        "session_id": {"type": "string", "format": "uuid"},
                        "title": {"type": "string"},
                    },
                },
            },
            {
                "name": "engram.create",
                "description": "Create a new memory engram with summary metadata.",
                "inputSchema": {
                    "type": "object",
                    "required": ["title", "detailed_summary_markdown"],
                    "properties": {
                        "project_id": {"type": "string"},
                        "thread_id": {"type": "string"},
                        "title": {"type": "string"},
                        "abstract": {"type": "string"},
                        "detailed_summary_markdown": {"type": "string"},
                        "visibility_scope": {"type": "string", "enum": ["private", "project"]},
                        "tags": {"type": "array", "items": {"type": "string"}},
                        "keywords": {"type": "array", "items": {"type": "string"}},
                    },
                },
            },
            {
                "name": "engram.create_from_conversation",
                "description": (
                    "Create an engram from conversation markdown with optional "
                    "fill-empty metadata enrichment."
                ),
                "inputSchema": {
                    "type": "object",
                    "required": ["conversation_markdown"],
                    "properties": {
                        "project_id": {"type": "string"},
                        "conversation_markdown": {"type": "string"},
                        "thread_id": {"type": "string"},
                        "title": {"type": "string"},
                        "abstract": {"type": "string"},
                        "visibility_scope": {"type": "string", "enum": ["private", "project"]},
                        "tags": {"type": "array", "items": {"type": "string"}},
                        "keywords": {"type": "array", "items": {"type": "string"}},
                        "retrieval_text": {"type": "string"},
                        "source_session_id": {"type": "string", "format": "uuid"},
                    },
                },
            },
            {
                "name": "engram.query",
                "description": "Query engrams by semantic text plus metadata filters.",
                "inputSchema": {
                    "type": "object",
                    "required": ["query"],
                    "properties": {
                        "query": {"type": "string"},
                        "top_k": {"type": "integer", "minimum": 1, "maximum": 50},
                        "project_id": {"type": "string"},
                        "tags": {"type": "array", "items": {"type": "string"}},
                        "keywords": {"type": "array", "items": {"type": "string"}},
                        "created_after": {"type": "string", "format": "date-time"},
                        "created_before": {"type": "string", "format": "date-time"},
                    },
                },
            },
            {
                "name": "engram.rehydrate",
                "description": "Return a compact and citation-packed rehydration bundle.",
                "inputSchema": {
                    "type": "object",
                    "required": ["engram_id"],
                    "properties": {"engram_id": {"type": "string", "format": "uuid"}},
                },
            },
            {
                "name": "engram.pin_to_session",
                "description": "Pin an engram to a chat session context chain.",
                "inputSchema": {
                    "type": "object",
                    "required": ["session_id", "engram_id"],
                    "properties": {
                        "session_id": {"type": "string", "format": "uuid"},
                        "engram_id": {"type": "string", "format": "uuid"},
                    },
                },
            },
            {
                "name": "chat.delete_session",
                "description": "Soft-delete a chat session with optional linked-engram deletion.",
                "inputSchema": {
                    "type": "object",
                    "required": ["session_id"],
                    "properties": {
                        "session_id": {"type": "string", "format": "uuid"},
                        "delete_linked_engrams": {"type": "boolean"},
                        "reason": {"type": "string"},
                    },
                },
            },
            {
                "name": "chat.restore_session",
                "description": "Restore a previously soft-deleted chat session.",
                "inputSchema": {
                    "type": "object",
                    "required": ["session_id"],
                    "properties": {"session_id": {"type": "string", "format": "uuid"}},
                },
            },
            {
                "name": "project.list",
                "description": "List visible projects for the authenticated actor.",
                "inputSchema": {
                    "type": "object",
                    "properties": {
                        "include_archived": {"type": "boolean"},
                        "limit": {"type": "integer", "minimum": 1},
                        "offset": {"type": "integer", "minimum": 0},
                    },
                },
            },
            {
                "name": "project.create",
                "description": "Create a project; admins can optionally set owner_user_id.",
                "inputSchema": {
                    "type": "object",
                    "required": ["project_id", "name"],
                    "properties": {
                        "project_id": {"type": "string"},
                        "name": {"type": "string"},
                        "description": {"type": "string"},
                        "owner_user_id": {"type": "string", "format": "uuid"},
                    },
                },
            },
            {
                "name": "project.get_default",
                "description": "Get the authenticated actor's default project id.",
                "inputSchema": {"type": "object", "properties": {}},
            },
            {
                "name": "project.set_default",
                "description": "Set the authenticated actor's default project id.",
                "inputSchema": {
                    "type": "object",
                    "required": ["project_id"],
                    "properties": {"project_id": {"type": "string"}},
                },
            },
            {
                "name": "engram.list",
                "description": "List engrams with project/session/query filters.",
                "inputSchema": {
                    "type": "object",
                    "properties": {
                        "project_id": {"type": "string"},
                        "session_id": {"type": "string", "format": "uuid"},
                        "q": {"type": "string"},
                        "include_deleted": {"type": "boolean"},
                        "limit": {"type": "integer", "minimum": 1},
                        "offset": {"type": "integer", "minimum": 0},
                    },
                },
            },
            {
                "name": "engram.get",
                "description": "Get an engram in management format with editable source payload.",
                "inputSchema": {
                    "type": "object",
                    "required": ["engram_id"],
                    "properties": {
                        "engram_id": {"type": "string", "format": "uuid"},
                        "include_deleted": {"type": "boolean"},
                    },
                },
            },
            {
                "name": "engram.update",
                "description": "Update engram metadata/markdown/sources with optimistic concurrency.",
                "inputSchema": {
                    "type": "object",
                    "required": ["engram_id"],
                    "properties": {
                        "engram_id": {"type": "string", "format": "uuid"},
                        "title": {"type": "string"},
                        "abstract": {"type": "string"},
                        "detailed_summary_markdown": {"type": "string"},
                        "tags": {"type": "array", "items": {"type": "string"}},
                        "keywords": {"type": "array", "items": {"type": "string"}},
                        "visibility_scope": {"type": "string", "enum": ["private", "project"]},
                        "expected_updated_at": {"type": "string", "format": "date-time"},
                        "sources": {
                            "type": "array",
                            "items": {
                                "type": "object",
                                "required": ["captured_at", "url", "title", "snippet"],
                                "properties": {
                                    "captured_at": {"type": "string", "format": "date-time"},
                                    "url": {"type": "string"},
                                    "title": {"type": "string"},
                                    "snippet": {"type": "string"},
                                    "content_text": {"type": "string"},
                                    "content_hash": {"type": "string"},
                                },
                            },
                        },
                    },
                },
            },
            {
                "name": "engram.move_project",
                "description": "Move an engram to another project and auto-detach invalid collections.",
                "inputSchema": {
                    "type": "object",
                    "required": ["engram_id", "target_project_id"],
                    "properties": {
                        "engram_id": {"type": "string", "format": "uuid"},
                        "target_project_id": {"type": "string"},
                        "reason": {"type": "string"},
                        "expected_updated_at": {"type": "string", "format": "date-time"},
                    },
                },
            },
            {
                "name": "engram.delete",
                "description": "Soft-delete an engram.",
                "inputSchema": {
                    "type": "object",
                    "required": ["engram_id"],
                    "properties": {
                        "engram_id": {"type": "string", "format": "uuid"},
                        "reason": {"type": "string"},
                    },
                },
            },
            {
                "name": "engram.restore",
                "description": "Restore a soft-deleted engram.",
                "inputSchema": {
                    "type": "object",
                    "required": ["engram_id"],
                    "properties": {"engram_id": {"type": "string", "format": "uuid"}},
                },
            },
            {
                "name": "engram.collection_list",
                "description": "List engram collections (project-bounded groups).",
                "inputSchema": {
                    "type": "object",
                    "properties": {
                        "project_id": {"type": "string"},
                        "include_deleted": {"type": "boolean"},
                        "limit": {"type": "integer", "minimum": 1},
                        "offset": {"type": "integer", "minimum": 0},
                    },
                },
            },
            {
                "name": "engram.collection_create",
                "description": "Create an engram collection for a project.",
                "inputSchema": {
                    "type": "object",
                    "required": ["name"],
                    "properties": {
                        "project_id": {"type": "string"},
                        "name": {"type": "string"},
                        "description": {"type": "string"},
                    },
                },
            },
            {
                "name": "engram.collection_update",
                "description": "Update collection metadata with optimistic concurrency.",
                "inputSchema": {
                    "type": "object",
                    "required": ["collection_id"],
                    "properties": {
                        "collection_id": {"type": "string", "format": "uuid"},
                        "name": {"type": "string"},
                        "description": {"type": "string"},
                        "expected_updated_at": {"type": "string", "format": "date-time"},
                    },
                },
            },
            {
                "name": "engram.collection_delete",
                "description": "Soft-delete a collection.",
                "inputSchema": {
                    "type": "object",
                    "required": ["collection_id"],
                    "properties": {
                        "collection_id": {"type": "string", "format": "uuid"},
                        "reason": {"type": "string"},
                    },
                },
            },
            {
                "name": "engram.collection_add_items",
                "description": "Add one or more engrams to a collection.",
                "inputSchema": {
                    "type": "object",
                    "required": ["collection_id", "engram_ids"],
                    "properties": {
                        "collection_id": {"type": "string", "format": "uuid"},
                        "engram_ids": {
                            "type": "array",
                            "items": {"type": "string", "format": "uuid"},
                        },
                    },
                },
            },
            {
                "name": "engram.collection_remove_items",
                "description": "Remove one engram from a collection.",
                "inputSchema": {
                    "type": "object",
                    "required": ["collection_id", "engram_id"],
                    "properties": {
                        "collection_id": {"type": "string", "format": "uuid"},
                        "engram_id": {"type": "string", "format": "uuid"},
                    },
                },
            },
            {
                "name": "user.get_profile",
                "description": "Get the authenticated user profile.",
                "inputSchema": {"type": "object", "properties": {}},
            },
            {
                "name": "user.list_projects",
                "description": "List project IDs visible to the authenticated user.",
                "inputSchema": {"type": "object", "properties": {}},
            },
        ]

    def _required_scope_for_tool(self, tool_name: str) -> str:
        canonical = self._canonical_tool_name(tool_name)
        if canonical in _WRITE_TOOL_NAMES:
            return "write"
        if canonical in _READ_TOOL_NAMES:
            return "read"
        return "read"

    def _visible_tool_catalog(self, token_auth: McpTokenAuthContext | None) -> list[dict[str, Any]]:
        if token_auth is None:
            return [
                {**item, "name": _to_public_tool_name(item["name"])}
                for item in self._tool_catalog()
            ]

        allowed_tools = token_auth.allowed_tools
        allowed_canonical = (
            {self._canonical_tool_name(item) for item in allowed_tools} if allowed_tools else set()
        )

        visible: list[dict[str, Any]] = []
        for item in self._tool_catalog():
            canonical_name = item["name"]
            public_name = _to_public_tool_name(canonical_name)
            required_scope = self._required_scope_for_tool(canonical_name)
            if token_auth.scope == "read" and required_scope == "write":
                continue
            if allowed_tools and (
                canonical_name not in allowed_canonical
                and public_name not in allowed_tools
                and canonical_name not in allowed_tools
            ):
                continue
            visible.append({**item, "name": public_name})
        return visible

    def _project_id_for_tool(
        self,
        *,
        actor_user_id: UUID,
        tool_name: str,
        params: dict[str, Any],
    ) -> str | None:
        canonical_tool = self._canonical_tool_name(tool_name)

        if canonical_tool in {
            "chat.create_session",
            "engram.create",
            "engram.create_from_conversation",
            "project.create",
            "project.set_default",
            "engram.collection_create",
        }:
            raw_project = params.get("project_id")
            return self._normalize_project_id(str(raw_project) if raw_project else None)

        if canonical_tool in {
            "chat.get_session",
            "chat.get_lifecycle_policy",
            "chat.update_lifecycle_policy",
            "chat.list_messages",
            "chat.list_timeline",
            "chat.send_message",
            "chat.list_pinned_engrams",
            "chat.pin_engram",
            "chat.unpin_engram",
            "chat.list_pinned_documents",
            "chat.pin_document",
            "chat.unpin_document",
            "chat.continue_session",
            "engram.pin_to_session",
            "chat.delete_session",
            "chat.restore_session",
        }:
            session = self._memory_admin_service.get_session(
                session_id=self._parse_uuid(params, "session_id"),
                include_deleted=True,
            )
            return session.project_id if session else None

        if canonical_tool == "chat.save_as_engram":
            raw_session_id = params.get("session_id")
            if raw_session_id:
                session = self._chat_service.get_session(
                    actor_user_id=actor_user_id,
                    session_id=self._parse_uuid(params, "session_id"),
                )
                return session.project_id
            raw_project = params.get("project_id")
            return self._normalize_project_id(str(raw_project) if raw_project else None)

        if canonical_tool in {
            "engram.get",
            "engram.update",
            "engram.move_project",
            "engram.delete",
            "engram.restore",
        }:
            engram = self._memory_admin_service.find_engram(
                engram_id=self._parse_uuid(params, "engram_id"),
                include_deleted=True,
            )
            if not engram:
                return None
            if canonical_tool == "engram.move_project":
                raw_target = params.get("target_project_id")
                target_project = self._normalize_project_id(str(raw_target) if raw_target else None)
                if target_project:
                    return target_project
            return engram.project_id

        if canonical_tool in {
            "engram.collection_update",
            "engram.collection_delete",
            "engram.collection_add_items",
            "engram.collection_remove_items",
        }:
            collection = self._memory_admin_service.find_collection(
                collection_id=self._parse_uuid(params, "collection_id"),
                include_deleted=True,
            )
            return collection.project_id if collection else None

        if canonical_tool == "engram.rehydrate":
            bundle = get_rehydration_bundle(
                self._parse_uuid(params, "engram_id"),
                actor_user_id=actor_user_id,
            )
            return bundle.project_id if bundle else None

        if canonical_tool in _OPTIONAL_PROJECT_TOOLS:
            raw_project = params.get("project_id")
            return self._normalize_project_id(str(raw_project) if raw_project else None)

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
        if token_auth.scope == "read" and required_scope == "write":
            raise McpRpcError(
                code=-32003,
                message="Token scope does not allow this tool",
                data={
                    "tool": tool_name,
                    "required_scope": required_scope,
                    "token_scope": token_auth.scope,
                },
            )

        allowed_tools = token_auth.allowed_tools
        allowed_canonical = (
            {self._canonical_tool_name(item) for item in allowed_tools} if allowed_tools else set()
        )
        if allowed_tools and (
            canonical_tool not in allowed_canonical
            and tool_name not in allowed_tools
            and _to_public_tool_name(canonical_tool) not in allowed_tools
        ):
            raise McpRpcError(
                code=-32003,
                message="Tool not allowed by token policy",
                data={
                    "tool": tool_name,
                    "required_scope": required_scope,
                    "token_scope": token_auth.scope,
                },
            )

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
                data={
                    "tool": tool_name,
                    "required_scope": required_scope,
                    "token_scope": token_auth.scope,
                    "project_id": project_id,
                },
            )
        return normalized_params

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

    def _dispatch_tool(
        self,
        *,
        actor: dict[str, Any],
        actor_user_id: UUID,
        method: str,
        params: dict[str, Any],
        token_auth: McpTokenAuthContext | None,
    ) -> dict[str, Any]:
        method = self._canonical_tool_name(method)
        actor_role = str(actor.get("role", ""))

        if method == "chat.create_session":
            created = self._chat_service.create_session(
                actor_user_id=actor_user_id,
                payload=ChatSessionCreateRequest(**params),
            )
            return {"session": created.model_dump(mode="json")}

        if method == "chat.list_sessions":
            sessions = self._chat_service.list_sessions(
                actor_user_id=actor_user_id,
                project_id=params.get("project_id"),
                limit=int(params.get("limit", 50)),
                offset=int(params.get("offset", 0)),
            )
            return {"sessions": [item.model_dump(mode="json") for item in sessions]}

        if method == "chat.get_session":
            session = self._chat_service.get_session(
                actor_user_id=actor_user_id,
                session_id=self._parse_uuid(params, "session_id"),
            )
            return {"session": session.model_dump(mode="json")}

        if method == "chat.get_lifecycle_policy":
            policy = self._chat_service.get_lifecycle_policy(
                actor_user_id=actor_user_id,
                session_id=self._parse_uuid(params, "session_id"),
            )
            return {"lifecycle_policy": policy.model_dump(mode="json")}

        if method == "chat.update_lifecycle_policy":
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

        if method == "chat.list_messages":
            messages = self._chat_service.list_messages(
                actor_user_id=actor_user_id,
                session_id=self._parse_uuid(params, "session_id"),
                limit=int(params.get("limit", 200)),
                offset=int(params.get("offset", 0)),
            )
            return {"messages": [item.model_dump(mode="json") for item in messages]}

        if method == "chat.list_timeline":
            events = self._chat_service.list_timeline_events(
                actor_user_id=actor_user_id,
                session_id=self._parse_uuid(params, "session_id"),
                limit=int(params.get("limit", 100)),
                offset=int(params.get("offset", 0)),
            )
            return {"events": [item.model_dump(mode="json") for item in events]}

        if method == "chat.send_message":
            message = self._chat_service.send_message(
                actor_user_id=actor_user_id,
                session_id=self._parse_uuid(params, "session_id"),
                payload=ChatMessageCreateRequest(content_text=params.get("content_text", "")),
            )
            return {"message": message.model_dump(mode="json")}

        if method == "chat.list_pinned_engrams":
            pinned = self._chat_service.list_pinned_engrams(
                actor_user_id=actor_user_id,
                session_id=self._parse_uuid(params, "session_id"),
            )
            return {"pinned_engrams": [item.model_dump(mode="json") for item in pinned]}

        if method in {"chat.pin_engram", "engram.pin_to_session"}:
            pinned = self._chat_service.pin_engram(
                actor_user_id=actor_user_id,
                session_id=self._parse_uuid(params, "session_id"),
                payload=PinEngramRequest(engram_id=self._parse_uuid(params, "engram_id")),
            )
            return {"pinned": pinned.model_dump(mode="json")}

        if method == "chat.unpin_engram":
            self._chat_service.unpin_engram(
                actor_user_id=actor_user_id,
                session_id=self._parse_uuid(params, "session_id"),
                engram_id=self._parse_uuid(params, "engram_id"),
            )
            return {"removed": True}

        if method == "chat.list_pinned_documents":
            pinned = self._chat_service.list_pinned_documents(
                actor_user_id=actor_user_id,
                session_id=self._parse_uuid(params, "session_id"),
            )
            return {"pinned_documents": [item.model_dump(mode="json") for item in pinned]}

        if method == "chat.pin_document":
            pinned = self._chat_service.pin_document(
                actor_user_id=actor_user_id,
                session_id=self._parse_uuid(params, "session_id"),
                payload=PinDocumentRequest(document_id=self._parse_uuid(params, "document_id")),
            )
            return {"pinned": pinned.model_dump(mode="json")}

        if method == "chat.unpin_document":
            self._chat_service.unpin_document(
                actor_user_id=actor_user_id,
                session_id=self._parse_uuid(params, "session_id"),
                document_id=self._parse_uuid(params, "document_id"),
            )
            return {"removed": True}

        if method == "chat.list_project_documents":
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

        if method == "chat.save_as_engram":
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
                    data={
                        "missing": "conversation_markdown",
                    },
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

        if method == "chat.continue_session":
            continued = self._chat_service.continue_session(
                actor_user_id=actor_user_id,
                session_id=self._parse_uuid(params, "session_id"),
                payload=ContinueSessionRequest(title=params.get("title")),
            )
            return {"continuation": continued.model_dump(mode="json")}

        if method == "engram.create":
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

        if method == "engram.create_from_conversation":
            created, report = self._create_engram_from_conversation(
                actor_user_id=actor_user_id,
                actor_role=actor_role,
                token_auth=token_auth,
                params=params,
                enrichment_origin="mcp.engram.create_from_conversation",
            )
            return {"engram": created, "enrichment_report": report}

        if method == "engram.query":
            results = query_engrams(
                request=EngramQueryRequest(**params),
                embedding_dim=self._embedding_dim,
                actor_user_id=actor_user_id,
            )
            return {"results": [item.model_dump(mode="json") for item in results]}

        if method == "engram.rehydrate":
            engram_id = self._parse_uuid(params, "engram_id")
            bundle = get_rehydration_bundle(engram_id, actor_user_id=actor_user_id)
            if not bundle:
                raise McpRpcError(
                    code=-32004,
                    message="Engram not found",
                    data={"engram_id": str(engram_id)},
                )
            return {"bundle": bundle.model_dump(mode="json")}

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

        if method == "engram.list":
            session_id_value = params.get("session_id")
            session_id = (
                self._parse_uuid({"session_id": session_id_value}, "session_id")
                if session_id_value is not None
                else None
            )
            listed = self._memory_admin_service.list_engrams(
                project_id=params.get("project_id"),
                session_id=session_id,
                owner_user_id=None if self._is_admin(actor) else actor_user_id,
                query_text=params.get("q"),
                include_deleted=bool(params.get("include_deleted", False)),
                limit=int(params.get("limit", 200)),
                offset=int(params.get("offset", 0)),
            )
            return {"engrams": [item.model_dump(mode="json") for item in listed]}

        if method == "engram.get":
            engram_id = self._parse_uuid(params, "engram_id")
            include_deleted = bool(params.get("include_deleted", True))
            engram = self._memory_admin_service.get_engram(
                engram_id=engram_id,
                include_deleted=include_deleted,
            )
            self._require_owner_or_admin(
                actor=actor,
                owner_user_id=engram.owner_user_id,
                resource="engram",
                resource_id=str(engram_id),
            )
            return {"engram": engram.model_dump(mode="json")}

        if method == "engram.update":
            engram_id = self._parse_uuid(params, "engram_id")
            current = self._memory_admin_service.get_engram(
                engram_id=engram_id,
                include_deleted=True,
            )
            self._require_owner_or_admin(
                actor=actor,
                owner_user_id=current.owner_user_id,
                resource="engram",
                resource_id=str(engram_id),
            )
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

        if method == "engram.move_project":
            engram_id = self._parse_uuid(params, "engram_id")
            current = self._memory_admin_service.get_engram(
                engram_id=engram_id,
                include_deleted=True,
            )
            if (
                token_auth
                and token_auth.allowed_project_ids
                and current.project_id not in token_auth.allowed_project_ids
            ):
                raise McpRpcError(
                    code=-32003,
                    message="Project not allowed by token policy",
                    data={
                        "tool": method,
                        "project_id": current.project_id,
                        "token_scope": token_auth.scope,
                    },
                )
            self._require_owner_or_admin(
                actor=actor,
                owner_user_id=current.owner_user_id,
                resource="engram",
                resource_id=str(engram_id),
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

        if method == "engram.delete":
            engram_id = self._parse_uuid(params, "engram_id")
            current = self._memory_admin_service.get_engram(
                engram_id=engram_id,
                include_deleted=True,
            )
            self._require_owner_or_admin(
                actor=actor,
                owner_user_id=current.owner_user_id,
                resource="engram",
                resource_id=str(engram_id),
            )
            deleted = self._memory_admin_service.delete_engram(
                engram_id=engram_id,
                actor_user_id=actor_user_id,
                payload=AdminEngramDeleteRequest(reason=params.get("reason")),
            )
            return {"result": deleted.model_dump(mode="json")}

        if method == "engram.restore":
            engram_id = self._parse_uuid(params, "engram_id")
            current = self._memory_admin_service.get_engram(
                engram_id=engram_id,
                include_deleted=True,
            )
            self._require_owner_or_admin(
                actor=actor,
                owner_user_id=current.owner_user_id,
                resource="engram",
                resource_id=str(engram_id),
            )
            restored = self._memory_admin_service.restore_engram(engram_id=engram_id)
            return {"result": restored.model_dump(mode="json")}

        if method == "engram.collection_list":
            collections = self._memory_admin_service.list_collections(
                project_id=params.get("project_id"),
                owner_user_id=None if self._is_admin(actor) else actor_user_id,
                include_deleted=bool(params.get("include_deleted", False)),
                limit=int(params.get("limit", 200)),
                offset=int(params.get("offset", 0)),
            )
            return {"collections": [item.model_dump(mode="json") for item in collections]}

        if method == "engram.collection_create":
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

        if method == "engram.collection_update":
            collection_id = self._parse_uuid(params, "collection_id")
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
            collection_id = self._parse_uuid(params, "collection_id")
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
            result = self._memory_admin_service.delete_collection(
                collection_id=collection_id,
                actor_user_id=actor_user_id,
                payload=EngramCollectionDeleteRequest(reason=params.get("reason")),
            )
            return {"result": result}

        if method == "engram.collection_add_items":
            collection_id = self._parse_uuid(params, "collection_id")
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
            result = self._memory_admin_service.add_collection_items(
                collection_id=collection_id,
                actor_user_id=actor_user_id,
                payload=EngramCollectionItemsUpdateRequest(
                    engram_ids=self._parse_uuid_list(params, "engram_ids")
                ),
            )
            return {"result": result}

        if method == "engram.collection_remove_items":
            collection_id = self._parse_uuid(params, "collection_id")
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
            result = self._memory_admin_service.remove_collection_item(
                collection_id=collection_id,
                engram_id=self._parse_uuid(params, "engram_id"),
            )
            return {"result": result}

        if method == "chat.delete_session":
            session_id = self._parse_uuid(params, "session_id")
            session = self._memory_admin_service.get_session(
                session_id=session_id,
                include_deleted=True,
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
            session = self._memory_admin_service.get_session(
                session_id=session_id,
                include_deleted=True,
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
            restored = self._memory_admin_service.restore_session(session_id=session_id)
            return {"result": restored.model_dump(mode="json")}

        if method == "user.get_profile":
            return {"profile": actor}

        if method == "user.list_projects":
            return {"project_ids": self._projects_for_user(actor_user_id, self._chat_service)}

        raise McpRpcError(
            code=-32601,
            message="Method not found",
            data={"method": method},
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

        # Direct streaming path retained for backward compatibility.
        if canonical_method == "chat.send_message":
            try:
                authorized_params = self._enforce_token_authorization(
                    actor_user_id=actor_user_id,
                    token_auth=token_auth,
                    tool_name=request.method,
                    params=request.params,
                )
            except McpRpcError as exc:
                yield self._error(request.id, code=exc.code, message=exc.message, data=exc.data)
                return
            yield from self._stream_chat_send_message(
                actor_user_id=actor_user_id,
                request_id=request.id,
                tool_name=request.method,
                params=authorized_params,
            )
            return

        # Standard MCP interop path for clients that only use `tools/call`.
        if request.method == "tools/call":
            try:
                tool_name, tool_params = self._tool_name_and_params_for_tools_call(request.params)
                canonical_tool_name = self._canonical_tool_name(tool_name)
                tool_params = self._enforce_token_authorization(
                    actor_user_id=actor_user_id,
                    token_auth=token_auth,
                    tool_name=tool_name,
                    params=tool_params,
                )
            except McpRpcError as exc:
                yield self._error(request.id, code=exc.code, message=exc.message, data=exc.data)
                return

            # Streaming is currently only meaningful for chat message generation.
            if canonical_tool_name == "chat.send_message" and bool(tool_params.get("stream", True)):
                yield from self._stream_chat_send_message(
                    actor_user_id=actor_user_id,
                    request_id=request.id,
                    tool_name=tool_name,
                    params=tool_params,
                    as_tool_call=True,
                )
                return

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
                direct_payload = {"message": response.model_dump(mode="json")}
                if as_tool_call:
                    yield self._success(
                        request_id, self._tool_call_success(tool_name, direct_payload)
                    )
                else:
                    yield self._success(request_id, direct_payload)
                return

            final_message: dict[str, Any] | None = None
            for event_name, event_payload in self._chat_service.stream_message_events(
                actor_user_id=actor_user_id,
                session_id=session_id,
                payload=payload,
            ):
                event_payload_dict: dict[str, Any]
                if isinstance(event_payload, dict):
                    event_payload_dict = event_payload
                else:
                    event_payload_dict = {"value": event_payload}

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

            final_payload = {"message": final_message}
            if as_tool_call:
                yield self._success(request_id, self._tool_call_success(tool_name, final_payload))
            else:
                yield self._success(request_id, final_payload)
        except McpRpcError as exc:
            yield self._error(
                request_id,
                code=exc.code,
                message=exc.message,
                data=exc.data,
            )
        except ValidationError as exc:
            yield self._error(
                request_id,
                code=-32602,
                message="Invalid params",
                data={"errors": exc.errors()},
            )
        except ChatProviderExecutionError as exc:
            yield self._error(
                request_id,
                code=-32020,
                message=exc.detail,
                data={"error_code": exc.error_code, "status_code": exc.status_code},
            )
        except ChatServiceError as exc:
            yield self._error(
                request_id,
                code=-32010,
                message=exc.detail,
                data={"status_code": exc.status_code},
            )
        except Exception as exc:
            yield self._error(
                request_id,
                code=-32000,
                message="Internal MCP error",
                data={"detail": str(exc), "timestamp": datetime.now(UTC).isoformat()},
            )
