from __future__ import annotations

from collections.abc import Callable
from typing import Any, Protocol
from uuid import UUID

from .catalog import (
    _ENGRAM_SCOPED_TOOLS,
    _OPTIONAL_PROJECT_TOOLS,
    _SESSION_SCOPED_TOOLS,
)

_COLLECTION_SCOPED_TOOLS = {
    "engram.collection_update",
    "engram.collection_delete",
    "engram.collection_add_items",
    "engram.collection_remove_items",
}

_PROJECT_INPUT_TOOLS = {
    "chat.create_session",
    "engram.create",
    "engram.create_from_conversation",
    "project.create",
    "project.set_default",
    "engram.collection_create",
}


class TokenProjectScopeDependencies(Protocol):
    chat_service: Any
    memory_admin_service: Any
    parse_uuid: Callable[[dict[str, Any], str], UUID]
    get_rehydration_bundle: Callable[..., Any]


def normalize_project_id(value: str | None) -> str | None:
    normalized = (value or "").strip()
    return normalized or None


def _resolve_project_from_session(
    *,
    dependencies: TokenProjectScopeDependencies,
    session_id: UUID,
) -> str | None:
    session = dependencies.memory_admin_service.get_session(
        session_id=session_id,
        include_deleted=True,
    )
    return session.project_id if session else None


def _resolve_project_from_engram(
    *,
    dependencies: TokenProjectScopeDependencies,
    engram_id: UUID,
    target_project_id: str | None = None,
) -> str | None:
    normalized_target = normalize_project_id(target_project_id)
    if normalized_target:
        return normalized_target
    engram = dependencies.memory_admin_service.find_engram(
        engram_id=engram_id,
        include_deleted=True,
    )
    return engram.project_id if engram else None


def _project_id_from_input_params(params: dict[str, Any]) -> str | None:
    raw_project = params.get("project_id")
    return normalize_project_id(str(raw_project) if raw_project else None)


def _project_id_for_chat_save_as_engram(
    *,
    dependencies: TokenProjectScopeDependencies,
    actor_user_id: UUID,
    params: dict[str, Any],
) -> str | None:
    raw_session_id = params.get("session_id")
    if raw_session_id:
        session = dependencies.chat_service.get_session(
            actor_user_id=actor_user_id,
            session_id=dependencies.parse_uuid(params, "session_id"),
        )
        return session.project_id
    return _project_id_from_input_params(params)


def _project_id_for_engram_scoped_tool(
    *,
    dependencies: TokenProjectScopeDependencies,
    canonical_tool: str,
    params: dict[str, Any],
) -> str | None:
    target_project_id: str | None = None
    if canonical_tool == "engram.move_project":
        raw_target = params.get("target_project_id")
        target_project_id = str(raw_target) if raw_target is not None else None
    return _resolve_project_from_engram(
        dependencies=dependencies,
        engram_id=dependencies.parse_uuid(params, "engram_id"),
        target_project_id=target_project_id,
    )


def _project_id_for_collection_scoped_tool(
    *,
    dependencies: TokenProjectScopeDependencies,
    params: dict[str, Any],
) -> str | None:
    collection = dependencies.memory_admin_service.find_collection(
        collection_id=dependencies.parse_uuid(params, "collection_id"),
        include_deleted=True,
    )
    return collection.project_id if collection else None


def _project_id_for_rehydrate_tool(
    *,
    dependencies: TokenProjectScopeDependencies,
    actor_user_id: UUID,
    params: dict[str, Any],
) -> str | None:
    bundle = dependencies.get_rehydration_bundle(
        dependencies.parse_uuid(params, "engram_id"),
        actor_user_id=actor_user_id,
    )
    return bundle.project_id if bundle else None


def resolve_project_id_for_canonical_tool(
    *,
    dependencies: TokenProjectScopeDependencies,
    actor_user_id: UUID,
    canonical_tool: str,
    params: dict[str, Any],
) -> str | None:
    if canonical_tool in _PROJECT_INPUT_TOOLS:
        return _project_id_from_input_params(params)

    if canonical_tool in _SESSION_SCOPED_TOOLS:
        return _resolve_project_from_session(
            dependencies=dependencies,
            session_id=dependencies.parse_uuid(params, "session_id"),
        )

    if canonical_tool == "chat.save_as_engram":
        return _project_id_for_chat_save_as_engram(
            dependencies=dependencies,
            actor_user_id=actor_user_id,
            params=params,
        )

    if canonical_tool in _ENGRAM_SCOPED_TOOLS:
        return _project_id_for_engram_scoped_tool(
            dependencies=dependencies,
            canonical_tool=canonical_tool,
            params=params,
        )

    if canonical_tool in _COLLECTION_SCOPED_TOOLS:
        return _project_id_for_collection_scoped_tool(
            dependencies=dependencies,
            params=params,
        )

    if canonical_tool == "engram.rehydrate":
        return _project_id_for_rehydrate_tool(
            dependencies=dependencies,
            actor_user_id=actor_user_id,
            params=params,
        )

    if canonical_tool in _OPTIONAL_PROJECT_TOOLS:
        return _project_id_from_input_params(params)

    return None
