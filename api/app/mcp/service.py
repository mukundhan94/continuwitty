from __future__ import annotations

import json
from datetime import UTC, datetime
from typing import Any
from uuid import UUID

from pydantic import ValidationError

from app.chat.errors import ChatProviderExecutionError, ChatServiceError
from app.chat.service import ChatService
from app.ingestion.service import DocumentIngestionService
from app.mcp_tokens import McpTokenAuthContext
from app.models import (
    ChatLifecyclePolicyUpdateRequest,
    ChatMessageCreateRequest,
    ChatSessionCreateRequest,
    ContinueSessionRequest,
    EngramCreateFromConversationRequest,
    EngramQueryRequest,
    McpJsonRpcRequest,
    MemoryEngramCreate,
    PinDocumentRequest,
    PinEngramRequest,
    SaveSessionAsEngramRequest,
)
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
    "engram.create",
    "engram.create_from_conversation",
    "engram.pin_to_session",
}

# Alias maps to existing implementation branch but should obey the same scope semantics.
_TOOL_ALIASES = {"engram.pin_to_session": "chat.pin_engram"}

_OPTIONAL_PROJECT_TOOLS = {
    "engram.query",
    "chat.list_sessions",
    "chat.list_project_documents",
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
        embedding_dim: int,
        ingestion_service: DocumentIngestionService | None = None,
    ) -> None:
        self._chat_service = chat_service
        self._embedding_dim = embedding_dim
        self._ingestion_service = ingestion_service

    @staticmethod
    def _success(id_value: str | int, result: dict[str, Any]) -> dict[str, Any]:
        return {"jsonrpc": "2.0", "id": id_value, "result": result}

    @staticmethod
    def _error(
        id_value: str | int,
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
        id_value: str | int,
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
                "description": "Save a chat session as an engram artifact.",
                "inputSchema": {
                    "type": "object",
                    "required": ["session_id", "title"],
                    "properties": {
                        "session_id": {"type": "string", "format": "uuid"},
                        "title": {"type": "string"},
                        "abstract": {"type": "string"},
                        "visibility_scope": {"type": "string", "enum": ["private", "project"]},
                        "tags": {"type": "array", "items": {"type": "string"}},
                        "keywords": {"type": "array", "items": {"type": "string"}},
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
                    "required": ["project_id", "title", "detailed_summary_markdown"],
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
                    "required": ["project_id", "conversation_markdown"],
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
        canonical = _TOOL_ALIASES.get(tool_name, tool_name)
        if canonical in _WRITE_TOOL_NAMES:
            return "write"
        if canonical in _READ_TOOL_NAMES:
            return "read"
        return "read"

    def _visible_tool_catalog(self, token_auth: McpTokenAuthContext | None) -> list[dict[str, Any]]:
        if token_auth is None:
            return self._tool_catalog()

        visible: list[dict[str, Any]] = []
        for item in self._tool_catalog():
            tool_name = item["name"]
            canonical_name = _TOOL_ALIASES.get(tool_name, tool_name)
            required_scope = self._required_scope_for_tool(tool_name)
            if token_auth.scope == "read" and required_scope == "write":
                continue
            if token_auth.allowed_tools and (
                tool_name not in token_auth.allowed_tools
                and canonical_name not in token_auth.allowed_tools
            ):
                continue
            visible.append(item)
        return visible

    def _project_id_for_tool(
        self,
        *,
        actor_user_id: UUID,
        tool_name: str,
        params: dict[str, Any],
    ) -> str | None:
        if tool_name in {
            "chat.create_session",
            "engram.create",
            "engram.create_from_conversation",
        }:
            raw_project = params.get("project_id")
            return str(raw_project) if raw_project else None

        if tool_name in {
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
            "chat.save_as_engram",
            "chat.continue_session",
            "engram.pin_to_session",
        }:
            session = self._chat_service.get_session(
                actor_user_id=actor_user_id,
                session_id=self._parse_uuid(params, "session_id"),
            )
            return session.project_id

        if tool_name == "engram.rehydrate":
            bundle = get_rehydration_bundle(
                self._parse_uuid(params, "engram_id"),
                actor_user_id=actor_user_id,
            )
            return bundle.project_id if bundle else None

        if tool_name in _OPTIONAL_PROJECT_TOOLS:
            raw_project = params.get("project_id")
            return str(raw_project) if raw_project else None

        return None

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

        canonical_tool = _TOOL_ALIASES.get(tool_name, tool_name)
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

        if token_auth.allowed_tools and (
            tool_name not in token_auth.allowed_tools
            and canonical_tool not in token_auth.allowed_tools
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
            tool_name=tool_name,
            params=normalized_params,
        )

        if tool_name in _OPTIONAL_PROJECT_TOOLS and not project_id:
            if len(allowed_projects) > 1:
                raise McpRpcError(
                    code=-32602,
                    message="Invalid params",
                    data={"missing": "project_id", "reason": "token_has_multiple_allowed_projects"},
                )
            normalized_params["project_id"] = next(iter(allowed_projects))
            return normalized_params

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
    ) -> dict[str, Any]:
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
            saved = self._chat_service.save_session_as_engram(
                actor_user_id=actor_user_id,
                session_id=self._parse_uuid(params, "session_id"),
                payload=SaveSessionAsEngramRequest(
                    title=params.get("title", ""),
                    abstract=params.get("abstract", ""),
                    visibility_scope=params.get("visibility_scope", "private"),
                    tags=params.get("tags", []),
                    keywords=params.get("keywords", []),
                ),
            )
            return {"saved_engram": saved.model_dump(mode="json")}

        if method == "chat.continue_session":
            continued = self._chat_service.continue_session(
                actor_user_id=actor_user_id,
                session_id=self._parse_uuid(params, "session_id"),
                payload=ContinueSessionRequest(title=params.get("title")),
            )
            return {"continuation": continued.model_dump(mode="json")}

        if method == "engram.create":
            created = create_engram(
                payload=MemoryEngramCreate(**params),
                embedding_dim=self._embedding_dim,
                owner_user_id=actor_user_id,
                enrichment_origin="mcp.engram.create",
            )
            return {"engram": created.model_dump(mode="json")}

        if method == "engram.create_from_conversation":
            request = EngramCreateFromConversationRequest(**params)
            created, enrichment_report = create_engram_with_report(
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
                embedding_dim=self._embedding_dim,
                owner_user_id=actor_user_id,
                enrichment_origin="mcp.engram.create_from_conversation",
            )
            return {
                "engram": created.model_dump(mode="json"),
                "enrichment_report": {
                    "enrichment_applied": enrichment_report.get("enrichment_applied", False),
                    "auto_tags": enrichment_report.get("auto_tags", []),
                    "auto_keywords": enrichment_report.get("auto_keywords", []),
                    "abstract_derived": enrichment_report.get("abstract_derived", False),
                },
            }

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
                "serverInfo": {"name": "engram-vault-mcp", "version": "0.1.0"},
                "capabilities": {"tools": {"listChanged": False}},
            }

        if method == "tools/list":
            return {"tools": self._visible_tool_catalog(token_auth)}

        if method == "tools/call":
            tool_name, tool_params = self._tool_name_and_params_for_tools_call(params)
            authorized_params = self._enforce_token_authorization(
                actor_user_id=actor_user_id,
                token_auth=token_auth,
                tool_name=tool_name,
                params=tool_params,
            )
            tool_payload = self._dispatch_tool(
                actor=actor,
                actor_user_id=actor_user_id,
                method=tool_name,
                params=authorized_params,
            )
            return self._tool_call_success(tool_name, tool_payload)

        # Backward-compatible direct method path.
        authorized_params = self._enforce_token_authorization(
            actor_user_id=actor_user_id,
            token_auth=token_auth,
            tool_name=method,
            params=params,
        )
        return self._dispatch_tool(
            actor=actor,
            actor_user_id=actor_user_id,
            method=method,
            params=authorized_params,
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

        # Direct streaming path retained for backward compatibility.
        if request.method == "chat.send_message":
            try:
                authorized_params = self._enforce_token_authorization(
                    actor_user_id=actor_user_id,
                    token_auth=token_auth,
                    tool_name="chat.send_message",
                    params=request.params,
                )
            except McpRpcError as exc:
                yield self._error(request.id, code=exc.code, message=exc.message, data=exc.data)
                return
            yield from self._stream_chat_send_message(
                actor_user_id=actor_user_id,
                request_id=request.id,
                tool_name="chat.send_message",
                params=authorized_params,
            )
            return

        # Standard MCP interop path for clients that only use `tools/call`.
        if request.method == "tools/call":
            try:
                tool_name, tool_params = self._tool_name_and_params_for_tools_call(request.params)
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
            if tool_name == "chat.send_message" and bool(tool_params.get("stream", True)):
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
