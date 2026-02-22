from __future__ import annotations

from collections.abc import Callable
from dataclasses import dataclass
from typing import Any, NoReturn
from uuid import UUID

from fastapi import HTTPException

from app.mcp_tokens import McpTokenAuthContext

from .catalog import (
    _OPTIONAL_PROJECT_TOOLS,
    _PROJECT_FALLBACK_TOOLS,
    _READ_TOOL_NAMES,
    _WRITE_TOOL_NAMES,
    _to_public_tool_name,
    build_tool_catalog,
)
from .errors import McpRpcError
from .token_project_scope import (
    normalize_project_id,
    resolve_project_id_for_canonical_tool,
)


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


@dataclass(frozen=True)
class ResolveProjectForWriteRequest:
    actor_user_id: UUID
    actor_role: str
    requested_project_id: str | None
    token_auth: McpTokenAuthContext | None


@dataclass(frozen=True)
class EnforceTokenAuthorizationRequest:
    actor_user_id: UUID
    token_auth: McpTokenAuthContext | None
    tool_name: str
    params: dict[str, Any]


@dataclass
class _AllowedProjectResolutionContext:
    dependencies: TokenAuthorizationDependencies
    actor_user_id: UUID
    canonical_tool: str
    normalized_params: dict[str, Any]
    allowed_projects: set[str]


@dataclass(frozen=True)
class _ResolvedTokenToolContext:
    token_auth: McpTokenAuthContext
    canonical_tool: str
    required_scope: str
    error_context: _TokenPolicyErrorContext


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
    explicit_project_id = normalize_project_id(requested_project_id)
    if explicit_project_id is not None:
        return explicit_project_id, False
    token_project_id = _single_allowed_token_project_id(token_auth)
    if token_project_id is not None:
        return token_project_id, False
    return None, True


def _project_resolution_error(exc: HTTPException) -> McpRpcError:
    status_code = exc.status_code
    detail = str(exc.detail)

    if status_code >= 500:
        return McpRpcError(
            code=-32000,
            message="Internal MCP error",
            data={"status_code": status_code, "detail": detail},
        )

    if status_code in {401, 403}:
        return McpRpcError(
            code=-32003,
            message="Forbidden",
            data={
                "status_code": status_code,
                "detail": detail,
                "suggested_action": "check_mcp_token_scope_or_resource_permissions",
            },
        )

    if status_code == 404:
        return McpRpcError(
            code=-32004,
            message=detail,
            data={
                "status_code": status_code,
                "detail": detail,
                "suggested_action": "provide_existing_project_id_or_set_project_default",
            },
        )

    if status_code == 409:
        return McpRpcError(
            code=-32009,
            message=detail,
            data={
                "status_code": status_code,
                "detail": detail,
                "suggested_action": "resolve_conflict_and_retry",
            },
        )

    return McpRpcError(
        code=-32602,
        message="Invalid params",
        data={"status_code": status_code, "detail": detail},
    )


def resolve_project_for_write(
    *,
    dependencies: TokenAuthorizationDependencies,
    request: ResolveProjectForWriteRequest,
) -> tuple[str, bool]:
    resolved_project_input, used_default_project = _resolve_project_input_for_write(
        requested_project_id=request.requested_project_id,
        token_auth=request.token_auth,
    )

    try:
        resolution = dependencies.project_service.resolve_project_id_for_write(
            actor_user_id=request.actor_user_id,
            actor_role=request.actor_role,
            project_id=resolved_project_input,
        )
    except HTTPException as exc:
        raise _project_resolution_error(exc) from exc
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


def _is_tool_visible_for_token(
    *,
    canonical_name: str,
    token_auth: McpTokenAuthContext,
    allowed_tools: set[str] | None,
    allowed_canonical: set[str],
) -> bool:
    required_scope = _required_scope_for_tool(canonical_name)
    if token_auth.scope == "read" and required_scope == "write":
        return False
    return _is_tool_allowed_by_token_policy(
        canonical_tool=canonical_name,
        allowed_tools=allowed_tools,
        allowed_canonical=allowed_canonical,
    )


def _public_tool_entry(item: dict[str, Any]) -> dict[str, Any]:
    return {**item, "name": _to_public_tool_name(item["name"])}


def _public_tool_catalog() -> list[dict[str, Any]]:
    return [_public_tool_entry(item) for item in build_tool_catalog()]


def _visible_tool_catalog_for_token(
    *,
    dependencies: TokenAuthorizationDependencies,
    token_auth: McpTokenAuthContext,
) -> list[dict[str, Any]]:
    allowed_tools = token_auth.allowed_tools
    allowed_canonical = _allowed_canonical_tools(
        canonical_tool_name=dependencies.canonical_tool_name,
        allowed_tools=allowed_tools,
    )
    return [
        _public_tool_entry(item)
        for item in build_tool_catalog()
        if _is_tool_visible_for_token(
            canonical_name=item["name"],
            token_auth=token_auth,
            allowed_tools=allowed_tools,
            allowed_canonical=allowed_canonical,
        )
    ]


def visible_tool_catalog(
    *,
    dependencies: TokenAuthorizationDependencies,
    token_auth: McpTokenAuthContext | None,
) -> list[dict[str, Any]]:
    if token_auth is None:
        return _public_tool_catalog()
    return _visible_tool_catalog_for_token(
        dependencies=dependencies,
        token_auth=token_auth,
    )


def project_id_for_tool(
    *,
    dependencies: TokenAuthorizationDependencies,
    actor_user_id: UUID,
    tool_name: str,
    params: dict[str, Any],
) -> str | None:
    canonical_tool = dependencies.canonical_tool_name(tool_name)
    return resolve_project_id_for_canonical_tool(
        dependencies=dependencies,
        actor_user_id=actor_user_id,
        canonical_tool=canonical_tool,
        params=params,
    )


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
    project_id = resolve_project_id_for_canonical_tool(
        dependencies=context.dependencies,
        actor_user_id=context.actor_user_id,
        canonical_tool=context.canonical_tool,
        params=context.normalized_params,
    )
    if project_id:
        return context.normalized_params, project_id
    context.normalized_params = _enforce_single_allowed_project_autofill(
        canonical_tool=context.canonical_tool,
        normalized_params=context.normalized_params,
        allowed_projects=context.allowed_projects,
    )
    project_id = resolve_project_id_for_canonical_tool(
        dependencies=context.dependencies,
        actor_user_id=context.actor_user_id,
        canonical_tool=context.canonical_tool,
        params=context.normalized_params,
    )
    return context.normalized_params, project_id


def _is_token_exempt_tool_name(tool_name: str) -> bool:
    return tool_name in {"initialize", "tools/list"}


def _resolve_token_tool_context(
    *,
    dependencies: TokenAuthorizationDependencies,
    request: EnforceTokenAuthorizationRequest,
) -> _ResolvedTokenToolContext:
    canonical_tool = dependencies.canonical_tool_name(request.tool_name)
    required_scope = _required_scope_for_tool(canonical_tool)
    return _ResolvedTokenToolContext(
        token_auth=request.token_auth,
        canonical_tool=canonical_tool,
        required_scope=required_scope,
        error_context=_TokenPolicyErrorContext(
            tool_name=request.tool_name,
            required_scope=required_scope,
            token_scope=request.token_auth.scope,
        ),
    )


def _enforce_token_scope_policy(*, context: _ResolvedTokenToolContext) -> None:
    if context.token_auth.scope == "read" and context.required_scope == "write":
        _raise_token_policy_error(
            message="Token scope does not allow this tool",
            context=context.error_context,
        )


def _enforce_token_tool_allowlist_policy(
    *,
    dependencies: TokenAuthorizationDependencies,
    context: _ResolvedTokenToolContext,
) -> None:
    allowed_tools = context.token_auth.allowed_tools
    allowed_canonical = _allowed_canonical_tools(
        canonical_tool_name=dependencies.canonical_tool_name,
        allowed_tools=allowed_tools,
    )
    if _is_tool_allowed_by_token_policy(
        canonical_tool=context.canonical_tool,
        allowed_tools=allowed_tools,
        allowed_canonical=allowed_canonical,
    ):
        return
    _raise_token_policy_error(
        message="Tool not allowed by token policy",
        context=context.error_context,
    )


def _resolve_project_constrained_params(
    *,
    dependencies: TokenAuthorizationDependencies,
    request: EnforceTokenAuthorizationRequest,
    context: _ResolvedTokenToolContext,
) -> tuple[dict[str, Any], str | None]:
    normalized_params = dict(request.params)
    allowed_projects = context.token_auth.allowed_project_ids
    if not allowed_projects:
        return normalized_params, None
    return _resolve_project_for_allowed_projects(
        context=_AllowedProjectResolutionContext(
            dependencies=dependencies,
            actor_user_id=request.actor_user_id,
            canonical_tool=context.canonical_tool,
            normalized_params=normalized_params,
            allowed_projects=allowed_projects,
        ),
    )


def _enforce_token_project_allowlist_policy(
    *,
    context: _ResolvedTokenToolContext,
    project_id: str | None,
) -> None:
    allowed_projects = context.token_auth.allowed_project_ids
    if not project_id or project_id in allowed_projects:
        return
    _raise_token_policy_error(
        message="Project not allowed by token policy",
        context=_TokenPolicyErrorContext(
            tool_name=context.error_context.tool_name,
            required_scope=context.required_scope,
            token_scope=context.token_auth.scope,
            project_id=project_id,
        ),
    )


def enforce_token_authorization(
    *,
    dependencies: TokenAuthorizationDependencies,
    request: EnforceTokenAuthorizationRequest,
) -> dict[str, Any]:
    if request.token_auth is None:
        return request.params

    if _is_token_exempt_tool_name(request.tool_name):
        return request.params

    context = _resolve_token_tool_context(
        dependencies=dependencies,
        request=request,
    )
    _enforce_token_scope_policy(context=context)
    _enforce_token_tool_allowlist_policy(
        dependencies=dependencies,
        context=context,
    )

    normalized_params, project_id = _resolve_project_constrained_params(
        dependencies=dependencies,
        request=request,
        context=context,
    )
    _enforce_token_project_allowlist_policy(
        context=context,
        project_id=project_id,
    )
    return normalized_params
