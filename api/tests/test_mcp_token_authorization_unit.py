from __future__ import annotations

from datetime import UTC, datetime
from types import SimpleNamespace
from uuid import UUID, uuid4

import pytest
from fastapi import HTTPException

from app.mcp.catalog import _TOOL_ALIASES, _to_dotted_tool_name
from app.mcp.errors import McpRpcError
from app.mcp.token_authorization import (
    EnforceTokenAuthorizationRequest,
    ResolveProjectForWriteRequest,
    TokenAuthorizationDependencies,
    enforce_token_authorization,
    resolve_project_for_write,
    visible_tool_catalog,
)


def _dependencies() -> TokenAuthorizationDependencies:
    def canonical_tool_name(tool_name: str) -> str:
        dotted = _to_dotted_tool_name(tool_name)
        return _TOOL_ALIASES.get(dotted, dotted)

    return TokenAuthorizationDependencies(
        project_service=SimpleNamespace(),
        chat_service=SimpleNamespace(),
        memory_admin_service=SimpleNamespace(),
        parse_uuid=lambda params, key: UUID(str(params[key])),
        canonical_tool_name=canonical_tool_name,
        get_rehydration_bundle=lambda *_args, **_kwargs: None,
    )


def _token_auth(
    *,
    scope: str = "read",
    allowed_tools: set[str] | None = None,
    allowed_project_ids: set[str] | None = None,
) -> SimpleNamespace:
    return SimpleNamespace(
        token_id=uuid4(),
        owner_user_id=uuid4(),
        scope=scope,
        allowed_tools=allowed_tools if allowed_tools is not None else set(),
        allowed_project_ids=allowed_project_ids if allowed_project_ids is not None else set(),
        expires_at=datetime.now(UTC),
    )


def test_visible_tool_catalog_hides_write_tools_for_read_scope() -> None:
    visible = visible_tool_catalog(
        dependencies=_dependencies(),
        token_auth=_token_auth(scope="read"),
    )

    names = {item["name"] for item in visible}
    assert "chat_list_sessions" in names
    assert "chat_create_session" not in names


def test_enforce_token_authorization_autofills_single_allowed_project() -> None:
    normalized = enforce_token_authorization(
        dependencies=_dependencies(),
        request=EnforceTokenAuthorizationRequest(
            actor_user_id=uuid4(),
            token_auth=_token_auth(allowed_project_ids={"project-alpha"}),
            tool_name="chat_list_sessions",
            params={},
        ),
    )

    assert normalized == {"project_id": "project-alpha"}


def test_enforce_token_authorization_requires_explicit_project_for_multi_project_token() -> None:
    with pytest.raises(McpRpcError) as exc_info:
        enforce_token_authorization(
            dependencies=_dependencies(),
            request=EnforceTokenAuthorizationRequest(
                actor_user_id=uuid4(),
                token_auth=_token_auth(allowed_project_ids={"project-a", "project-b"}),
                tool_name="chat_list_sessions",
                params={},
            ),
        )

    assert exc_info.value.code == -32602
    assert exc_info.value.data == {
        "missing": "project_id",
        "reason": "token_has_multiple_allowed_projects",
    }


def test_resolve_project_for_write_surfaces_project_not_found_as_mcp_not_found() -> None:
    dependencies = _dependencies()
    actor_user_id = uuid4()

    def _raise_not_found(**_kwargs) -> None:
        raise HTTPException(status_code=404, detail="Project not found")

    dependencies.project_service.resolve_project_id_for_write = _raise_not_found

    with pytest.raises(McpRpcError) as exc_info:
        resolve_project_for_write(
            dependencies=dependencies,
            request=ResolveProjectForWriteRequest(
                actor_user_id=actor_user_id,
                actor_role="user",
                requested_project_id="missing-project",
                token_auth=None,
            ),
        )

    assert exc_info.value.code == -32004
    assert exc_info.value.message == "Project not found"
    assert exc_info.value.data == {
        "status_code": 404,
        "detail": "Project not found",
        "suggested_action": "provide_existing_project_id_or_set_project_default",
    }
