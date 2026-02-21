from __future__ import annotations

from dataclasses import dataclass
from typing import Any
from uuid import UUID

from app.mcp_tokens import McpTokenAuthContext
from app.models import (
    AdminSessionDeleteRequest,
    ChatLifecyclePolicyUpdateRequest,
    ChatMessageCreateRequest,
    ChatSessionCreateRequest,
    ContinueSessionRequest,
    PinDocumentRequest,
    PinEngramRequest,
)

from .errors import McpRpcError


@dataclass(frozen=True)
class ChatSessionLifecycleDispatchContext:
    actor: dict[str, Any]
    actor_user_id: UUID
    method: str
    params: dict[str, Any]


@dataclass(frozen=True)
class ChatSessionLifecycleDispatchDependencies:
    memory_admin_service: Any
    parse_uuid: Any
    require_session_access: Any


def dispatch_chat_session_lifecycle_tool(
    *,
    dependencies: ChatSessionLifecycleDispatchDependencies,
    context: ChatSessionLifecycleDispatchContext,
) -> dict[str, Any] | None:
    if context.method == "chat.delete_session":
        session_id = dependencies.parse_uuid(context.params, "session_id")
        dependencies.require_session_access(actor=context.actor, session_id=session_id)
        deleted = dependencies.memory_admin_service.delete_session(
            session_id=session_id,
            actor_user_id=context.actor_user_id,
            payload=AdminSessionDeleteRequest(
                delete_linked_engrams=bool(
                    context.params.get("delete_linked_engrams", False)
                ),
                reason=context.params.get("reason"),
            ),
        )
        return {"result": deleted.model_dump(mode="json")}

    if context.method == "chat.restore_session":
        session_id = dependencies.parse_uuid(context.params, "session_id")
        dependencies.require_session_access(actor=context.actor, session_id=session_id)
        restored = dependencies.memory_admin_service.restore_session(session_id=session_id)
        return {"result": restored.model_dump(mode="json")}

    return None


def dispatch_chat_pinning_tool(
    *,
    chat_service: Any,
    actor_user_id: UUID,
    method: str,
    params: dict[str, Any],
    parse_uuid: Any,
) -> dict[str, Any] | None:
    normalized_method = "chat.pin_engram" if method == "engram.pin_to_session" else method

    def _list_pinned_engrams() -> dict[str, Any]:
        pinned = chat_service.list_pinned_engrams(
            actor_user_id=actor_user_id,
            session_id=parse_uuid(params, "session_id"),
        )
        return {"pinned_engrams": [item.model_dump(mode="json") for item in pinned]}

    def _pin_engram() -> dict[str, Any]:
        pinned = chat_service.pin_engram(
            actor_user_id=actor_user_id,
            session_id=parse_uuid(params, "session_id"),
            payload=PinEngramRequest(engram_id=parse_uuid(params, "engram_id")),
        )
        return {"pinned": pinned.model_dump(mode="json")}

    def _unpin_engram() -> dict[str, Any]:
        chat_service.unpin_engram(
            actor_user_id=actor_user_id,
            session_id=parse_uuid(params, "session_id"),
            engram_id=parse_uuid(params, "engram_id"),
        )
        return {"removed": True}

    def _list_pinned_documents() -> dict[str, Any]:
        pinned = chat_service.list_pinned_documents(
            actor_user_id=actor_user_id,
            session_id=parse_uuid(params, "session_id"),
        )
        return {"pinned_documents": [item.model_dump(mode="json") for item in pinned]}

    def _pin_document() -> dict[str, Any]:
        pinned = chat_service.pin_document(
            actor_user_id=actor_user_id,
            session_id=parse_uuid(params, "session_id"),
            payload=PinDocumentRequest(document_id=parse_uuid(params, "document_id")),
        )
        return {"pinned": pinned.model_dump(mode="json")}

    def _unpin_document() -> dict[str, Any]:
        chat_service.unpin_document(
            actor_user_id=actor_user_id,
            session_id=parse_uuid(params, "session_id"),
            document_id=parse_uuid(params, "document_id"),
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


def dispatch_chat_session_query_tool(
    *,
    chat_service: Any,
    actor_user_id: UUID,
    method: str,
    params: dict[str, Any],
    parse_uuid: Any,
) -> dict[str, Any] | None:
    def _list_sessions() -> dict[str, Any]:
        sessions = chat_service.list_sessions(
            actor_user_id=actor_user_id,
            project_id=params.get("project_id"),
            limit=int(params.get("limit", 50)),
            offset=int(params.get("offset", 0)),
        )
        return {"sessions": [item.model_dump(mode="json") for item in sessions]}

    def _get_session() -> dict[str, Any]:
        session = chat_service.get_session(
            actor_user_id=actor_user_id,
            session_id=parse_uuid(params, "session_id"),
        )
        return {"session": session.model_dump(mode="json")}

    def _get_lifecycle_policy() -> dict[str, Any]:
        policy = chat_service.get_lifecycle_policy(
            actor_user_id=actor_user_id,
            session_id=parse_uuid(params, "session_id"),
        )
        return {"lifecycle_policy": policy.model_dump(mode="json")}

    def _update_lifecycle_policy() -> dict[str, Any]:
        policy = chat_service.update_lifecycle_policy(
            actor_user_id=actor_user_id,
            session_id=parse_uuid(params, "session_id"),
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
        messages = chat_service.list_messages(
            actor_user_id=actor_user_id,
            session_id=parse_uuid(params, "session_id"),
            limit=int(params.get("limit", 200)),
            offset=int(params.get("offset", 0)),
        )
        return {"messages": [item.model_dump(mode="json") for item in messages]}

    def _list_timeline() -> dict[str, Any]:
        events = chat_service.list_timeline_events(
            actor_user_id=actor_user_id,
            session_id=parse_uuid(params, "session_id"),
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


def dispatch_chat_primary_tool(
    *,
    chat_service: Any,
    ingestion_service: Any | None,
    actor_user_id: UUID,
    actor_role: str,
    method: str,
    params: dict[str, Any],
    token_auth: McpTokenAuthContext | None,
    parse_uuid: Any,
    dispatch_chat_save_as_engram_tool: Any,
) -> dict[str, Any] | None:
    def _list_project_documents() -> dict[str, Any]:
        if ingestion_service is None:
            raise McpRpcError(
                code=-32000,
                message="Ingestion service unavailable for MCP tool",
                data={"method": method},
            )
        documents = ingestion_service.list_documents(
            actor_user_id=actor_user_id,
            project_id=params.get("project_id"),
            limit=int(params.get("limit", 200)),
            offset=int(params.get("offset", 0)),
        )
        return {"documents": [item.model_dump(mode="json") for item in documents]}

    handlers = {
        "chat.create_session": lambda: {
            "session": chat_service.create_session(
                actor_user_id=actor_user_id,
                payload=ChatSessionCreateRequest(**params),
            ).model_dump(mode="json")
        },
        "chat.send_message": lambda: {
            "message": chat_service.send_message(
                actor_user_id=actor_user_id,
                session_id=parse_uuid(params, "session_id"),
                payload=ChatMessageCreateRequest(content_text=params.get("content_text", "")),
            ).model_dump(mode="json")
        },
        "chat.list_project_documents": _list_project_documents,
        "chat.save_as_engram": lambda: dispatch_chat_save_as_engram_tool(
            actor_user_id=actor_user_id,
            actor_role=actor_role,
            params=params,
            token_auth=token_auth,
        ),
        "chat.continue_session": lambda: {
            "continuation": chat_service.continue_session(
                actor_user_id=actor_user_id,
                session_id=parse_uuid(params, "session_id"),
                payload=ContinueSessionRequest(title=params.get("title")),
            ).model_dump(mode="json")
        },
    }
    handler = handlers.get(method)
    return handler() if handler else None
