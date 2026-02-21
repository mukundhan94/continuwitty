from __future__ import annotations

from collections.abc import Callable
from dataclasses import dataclass
from typing import Any, NoReturn
from uuid import UUID

from fastapi import HTTPException

from app.mcp_tokens import McpTokenAuthContext

from .catalog import (
    _ENGRAM_SCOPED_TOOLS,
    _OPTIONAL_PROJECT_TOOLS,
    _PROJECT_FALLBACK_TOOLS,
    _READ_TOOL_NAMES,
    _SESSION_SCOPED_TOOLS,
    _WRITE_TOOL_NAMES,
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

_PROJECT_INPUT_TOOLS = {
    "chat.create_session",
    "engram.create",
    "engram.create_from_conversation",
    "project.create",
    "project.set_default",
    "engram.collection_create",
}


@dataclass(frozen=True)
class TokenAuthorizationDependencies:
    project_service: Any
    chat_service: Any
    memory_admin_service: Any
    parse_uuid: Callable[[dict[str, Any], str], UUID]
    canonical_tool_name: Callable[[str], str]
    get_rehydration_bundle: Callable[..., Any]


@dataclass(frozen=True)
class _TokenPolicyErrorContext:
    tool_name: str
    required_scope: str
    token_scope: str
    project_id: str | None = None


@dataclass
class _AllowedProjectResolutionContext:
    dependencies: TokenAuthorizationDependencies
    actor_user_id: UUID
    canonical_tool: str
    normalized_params: dict[str, Any]
    allowed_projects: set[str]


def _normalize_project_id(value: str | None) -> str | None:
    normalized = (value or "").strip()
    return normalized or None


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
    *,
    requested_project_id: str | None,
    token_auth: McpTokenAuthContext | None,
) -> tuple[str | None, bool]:
    explicit_project_id = _normalize_project_id(requested_project_id)
    if explicit_project_id is not None:
        return explicit_project_id, False
    token_project_id = _single_allowed_token_project_id(token_auth)
    if token_project_id is not None:
        return token_project_id, False
    return None, True


def resolve_project_for_write(
    *,
    dependencies: TokenAuthorizationDependencies,
    actor_user_id: UUID,
    actor_role: str,
    requested_project_id: str | None,
    token_auth: McpTokenAuthContext | None,
) -> tuple[str, bool]:
    resolved_project_input, used_default_project = _resolve_project_input_for_write(
        requested_project_id=requested_project_id,
        token_auth=token_auth,
    )

    try:
        resolution = dependencies.project_service.resolve_project_id_for_write(
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


def _required_scope_for_tool(canonical_tool: str) -> str:
    if canonical_tool in _WRITE_TOOL_NAMES:
        return "write"
    if canonical_tool in _READ_TOOL_NAMES:
        return "read"
    return "read"


def _allowed_canonical_tools(
    *,
    canonical_tool_name: Callable[[str], str],
    allowed_tools: set[str] | None,
) -> set[str]:
    if not allowed_tools:
        return set()
    return {canonical_tool_name(item) for item in allowed_tools}


def _is_tool_allowed_by_token_policy(
    *,
    canonical_tool: str,
    allowed_tools: set[str] | None,
    allowed_canonical: set[str],
) -> bool:
    if not allowed_tools:
        return True
    return canonical_tool in allowed_canonical


def visible_tool_catalog(
    *,
    dependencies: TokenAuthorizationDependencies,
    token_auth: McpTokenAuthContext | None,
) -> list[dict[str, Any]]:
    if token_auth is None:
        return [{**item, "name": _to_public_tool_name(item["name"])} for item in build_tool_catalog()]

    allowed_tools = token_auth.allowed_tools
    allowed_canonical = _allowed_canonical_tools(
        canonical_tool_name=dependencies.canonical_tool_name,
        allowed_tools=allowed_tools,
    )

    visible: list[dict[str, Any]] = []
    for item in build_tool_catalog():
        canonical_name = item["name"]
        required_scope = _required_scope_for_tool(canonical_name)
        if token_auth.scope == "read" and required_scope == "write":
            continue
        if not _is_tool_allowed_by_token_policy(
            canonical_tool=canonical_name,
            allowed_tools=allowed_tools,
            allowed_canonical=allowed_canonical,
        ):
            continue
        visible.append({**item, "name": _to_public_tool_name(item["name"])})
    return visible


def _resolve_project_from_session(
    *,
    dependencies: TokenAuthorizationDependencies,
    session_id: UUID,
) -> str | None:
    session = dependencies.memory_admin_service.get_session(
        session_id=session_id,
        include_deleted=True,
    )
    return session.project_id if session else None


def _resolve_project_from_engram(
    *,
    dependencies: TokenAuthorizationDependencies,
    engram_id: UUID,
    target_project_id: str | None = None,
) -> str | None:
    normalized_target = _normalize_project_id(target_project_id)
    if normalized_target:
        return normalized_target
    engram = dependencies.memory_admin_service.find_engram(
        engram_id=engram_id,
        include_deleted=True,
    )
    return engram.project_id if engram else None


def _project_id_from_input_params(params: dict[str, Any]) -> str | None:
    raw_project = params.get("project_id")
    return _normalize_project_id(str(raw_project) if raw_project else None)


def _project_id_for_chat_save_as_engram(
    *,
    dependencies: TokenAuthorizationDependencies,
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
    dependencies: TokenAuthorizationDependencies,
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
    dependencies: TokenAuthorizationDependencies,
    params: dict[str, Any],
) -> str | None:
    collection = dependencies.memory_admin_service.find_collection(
        collection_id=dependencies.parse_uuid(params, "collection_id"),
        include_deleted=True,
    )
    return collection.project_id if collection else None


def _project_id_for_rehydrate_tool(
    *,
    dependencies: TokenAuthorizationDependencies,
    actor_user_id: UUID,
    params: dict[str, Any],
) -> str | None:
    bundle = dependencies.get_rehydration_bundle(
        dependencies.parse_uuid(params, "engram_id"),
        actor_user_id=actor_user_id,
    )
    return bundle.project_id if bundle else None


def project_id_for_tool(
    *,
    dependencies: TokenAuthorizationDependencies,
    actor_user_id: UUID,
    tool_name: str,
    params: dict[str, Any],
) -> str | None:
    canonical_tool = dependencies.canonical_tool_name(tool_name)
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


def _needs_project_autofill_for_token(canonical_tool: str, params: dict[str, Any]) -> bool:
    if canonical_tool in _OPTIONAL_PROJECT_TOOLS:
        return True
    if canonical_tool in _PROJECT_FALLBACK_TOOLS:
        if canonical_tool == "chat.save_as_engram":
            return not bool(params.get("session_id"))
        return True
    return False


def _enforce_single_allowed_project_autofill(
    *,
    canonical_tool: str,
    normalized_params: dict[str, Any],
    allowed_projects: set[str],
) -> dict[str, Any]:
    if not _needs_project_autofill_for_token(canonical_tool, normalized_params):
        return normalized_params
    if len(allowed_projects) > 1:
        raise McpRpcError(
            code=-32602,
            message="Invalid params",
            data={"missing": "project_id", "reason": "token_has_multiple_allowed_projects"},
        )
    normalized_params.setdefault("project_id", next(iter(allowed_projects)))
    return normalized_params


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


def _raise_token_policy_error(
    *,
    message: str,
    context: _TokenPolicyErrorContext,
) -> NoReturn:
    raise McpRpcError(
        code=-32003,
        message=message,
        data=_token_error_data(
            tool_name=context.tool_name,
            required_scope=context.required_scope,
            token_scope=context.token_scope,
            project_id=context.project_id,
        ),
    )


def _resolve_project_for_allowed_projects(
    *,
    context: _AllowedProjectResolutionContext,
) -> tuple[dict[str, Any], str | None]:
    project_id = project_id_for_tool(
        dependencies=context.dependencies,
        actor_user_id=context.actor_user_id,
        tool_name=context.canonical_tool,
        params=context.normalized_params,
    )
    if project_id:
        return context.normalized_params, project_id
    context.normalized_params = _enforce_single_allowed_project_autofill(
        canonical_tool=context.canonical_tool,
        normalized_params=context.normalized_params,
        allowed_projects=context.allowed_projects,
    )
    project_id = project_id_for_tool(
        dependencies=context.dependencies,
        actor_user_id=context.actor_user_id,
        tool_name=context.canonical_tool,
        params=context.normalized_params,
    )
    return context.normalized_params, project_id


def enforce_token_authorization(
    *,
    dependencies: TokenAuthorizationDependencies,
    actor_user_id: UUID,
    token_auth: McpTokenAuthContext | None,
    tool_name: str,
    params: dict[str, Any],
) -> dict[str, Any]:
    if token_auth is None:
        return params

    if tool_name in {"initialize", "tools/list"}:
        return params

    canonical_tool = dependencies.canonical_tool_name(tool_name)
    required_scope = _required_scope_for_tool(canonical_tool)
    error_context = _TokenPolicyErrorContext(
        tool_name=tool_name,
        required_scope=required_scope,
        token_scope=token_auth.scope,
    )
    if token_auth.scope == "read" and required_scope == "write":
        _raise_token_policy_error(
            message="Token scope does not allow this tool",
            context=error_context,
        )

    allowed_tools = token_auth.allowed_tools
    allowed_canonical = _allowed_canonical_tools(
        canonical_tool_name=dependencies.canonical_tool_name,
        allowed_tools=allowed_tools,
    )
    if not _is_tool_allowed_by_token_policy(
        canonical_tool=canonical_tool,
        allowed_tools=allowed_tools,
        allowed_canonical=allowed_canonical,
    ):
        _raise_token_policy_error(
            message="Tool not allowed by token policy",
            context=error_context,
        )

    normalized_params = dict(params)
    allowed_projects = token_auth.allowed_project_ids
    if not allowed_projects:
        return normalized_params

    normalized_params, project_id = _resolve_project_for_allowed_projects(
        context=_AllowedProjectResolutionContext(
            dependencies=dependencies,
            actor_user_id=actor_user_id,
            canonical_tool=canonical_tool,
            normalized_params=normalized_params,
            allowed_projects=allowed_projects,
        ),
    )

    if project_id and project_id not in allowed_projects:
        _raise_token_policy_error(
            message="Project not allowed by token policy",
            context=_TokenPolicyErrorContext(
                tool_name=tool_name,
                required_scope=required_scope,
                token_scope=token_auth.scope,
                project_id=project_id,
            ),
        )
    return normalized_params
