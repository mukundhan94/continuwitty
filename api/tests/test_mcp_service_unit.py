from __future__ import annotations

from dataclasses import dataclass
from types import SimpleNamespace
from unittest.mock import MagicMock
from uuid import uuid4

import pytest

from app.mcp.errors import McpRpcError
from app.mcp.service import McpService


@dataclass
class _Dumpable:
    payload: dict
    owner_user_id: object | None = None
    project_id: str | None = None

    def model_dump(self, mode: str = "json") -> dict:  # noqa: ARG002
        return self.payload


def _build_service() -> tuple[McpService, MagicMock, MagicMock, MagicMock]:
    chat_service = MagicMock()
    project_service = MagicMock()
    memory_admin_service = MagicMock()
    service = McpService(
        chat_service=chat_service,
        project_service=project_service,
        memory_admin_service=memory_admin_service,
        embedding_dim=1536,
    )
    return service, chat_service, project_service, memory_admin_service


def _token_auth(
    *,
    scope: str = "write",
    allowed_tools: set[str] | None = None,
    allowed_project_ids: set[str] | None = None,
) -> SimpleNamespace:
    return SimpleNamespace(
        scope=scope,
        allowed_tools=allowed_tools if allowed_tools is not None else set(),
        allowed_project_ids=allowed_project_ids if allowed_project_ids is not None else set(),
    )


def test_dispatch_tool_routes_chat_domain() -> None:
    service, chat_service, _, _ = _build_service()
    actor_user_id = uuid4()
    actor = {"user_id": str(actor_user_id), "role": "user"}
    chat_service.list_sessions.return_value = [_Dumpable(payload={"session_id": "s1"})]

    result = service._dispatch_tool(
        actor=actor,
        actor_user_id=actor_user_id,
        method="chat.list_sessions",
        params={},
        token_auth=None,
    )

    assert result == {"sessions": [{"session_id": "s1"}]}
    chat_service.list_sessions.assert_called_once()


def test_dispatch_tool_routes_engram_and_project_domains() -> None:
    service, _, project_service, memory_admin_service = _build_service()
    actor_user_id = uuid4()
    actor = {"user_id": str(actor_user_id), "role": "user"}
    memory_admin_service.list_engrams.return_value = [_Dumpable(payload={"engram_id": "e1"})]
    project_service.get_default_project_id.return_value = "project-alpha"

    engram_result = service._dispatch_tool(
        actor=actor,
        actor_user_id=actor_user_id,
        method="engram.list",
        params={"project_id": "project-alpha"},
        token_auth=None,
    )
    project_result = service._dispatch_tool(
        actor=actor,
        actor_user_id=actor_user_id,
        method="project.get_default",
        params={},
        token_auth=None,
    )

    assert engram_result == {"engrams": [{"engram_id": "e1"}]}
    assert project_result == {"default_project_id": "project-alpha"}


def test_dispatch_tool_routes_user_domain() -> None:
    service, _, _, _ = _build_service()
    actor_user_id = uuid4()
    actor = {"user_id": str(actor_user_id), "role": "user", "username": "alice"}

    result = service._dispatch_tool(
        actor=actor,
        actor_user_id=actor_user_id,
        method="user.get_profile",
        params={},
        token_auth=None,
    )

    assert result == {"profile": actor}


def test_dispatch_tool_unknown_method_returns_not_found_error() -> None:
    service, _, _, _ = _build_service()
    actor_user_id = uuid4()
    actor = {"user_id": str(actor_user_id), "role": "user"}

    with pytest.raises(McpRpcError) as exc_info:
        service._dispatch_tool(
            actor=actor,
            actor_user_id=actor_user_id,
            method="unknown.tool",
            params={},
            token_auth=None,
        )

    assert exc_info.value.code == -32601
    assert exc_info.value.data == {"method": "unknown.tool"}


def test_require_session_access_rejects_non_owner() -> None:
    service, _, _, memory_admin_service = _build_service()
    actor_user_id = uuid4()
    owner_user_id = uuid4()
    actor = {"user_id": str(actor_user_id), "role": "user"}
    session_id = uuid4()
    memory_admin_service.get_session.return_value = _Dumpable(
        payload={"session_id": str(session_id)},
        owner_user_id=owner_user_id,
    )

    with pytest.raises(McpRpcError) as exc_info:
        service._require_session_access(actor=actor, session_id=session_id)

    assert exc_info.value.code == -32003


def test_require_engram_access_returns_owned_engram() -> None:
    service, _, _, memory_admin_service = _build_service()
    actor_user_id = uuid4()
    actor = {"user_id": str(actor_user_id), "role": "user"}
    engram_id = uuid4()
    expected = _Dumpable(
        payload={"engram_id": str(engram_id)},
        owner_user_id=actor_user_id,
        project_id="project-alpha",
    )
    memory_admin_service.find_engram.return_value = expected

    resolved = service._require_engram_access(
        actor=actor,
        engram_id=engram_id,
        include_deleted=True,
    )

    assert resolved is expected


def test_require_collection_access_raises_not_found() -> None:
    service, _, _, memory_admin_service = _build_service()
    actor = {"user_id": str(uuid4()), "role": "user"}
    collection_id = uuid4()
    memory_admin_service.find_collection.return_value = None

    with pytest.raises(McpRpcError) as exc_info:
        service._require_collection_access(actor=actor, collection_id=collection_id)

    assert exc_info.value.code == -32004
    assert exc_info.value.data == {"collection_id": str(collection_id)}


def test_visible_tool_catalog_accepts_public_tool_name_allowlist() -> None:
    service, _, _, _ = _build_service()
    token_auth = _token_auth(allowed_tools={"engram_query"})

    visible = service._visible_tool_catalog(token_auth=token_auth)

    assert {item["name"] for item in visible} == {"engram_query"}


def test_enforce_token_authorization_allows_alias_tool_with_public_allowlist_name() -> None:
    service, _, _, _ = _build_service()
    token_auth = _token_auth(allowed_tools={"chat_pin_engram"})

    normalized = service._enforce_token_authorization(
        actor_user_id=uuid4(),
        token_auth=token_auth,
        tool_name="engram_pin_to_session",
        params={},
    )

    assert normalized == {}


def test_enforce_token_authorization_rejects_tool_outside_allowlist() -> None:
    service, _, _, _ = _build_service()
    token_auth = _token_auth(scope="read", allowed_tools={"engram_query"})

    with pytest.raises(McpRpcError) as exc_info:
        service._enforce_token_authorization(
            actor_user_id=uuid4(),
            token_auth=token_auth,
            tool_name="chat_list_sessions",
            params={},
        )

    assert exc_info.value.code == -32003
    assert exc_info.value.data == {
        "tool": "chat_list_sessions",
        "required_scope": "read",
        "token_scope": "read",
    }


@pytest.mark.parametrize(
    ("tool_name", "params", "allowed_project_ids", "expected_data"),
    [
        (
            "chat_create_session",
            {},
            None,
            {
                "tool": "chat_create_session",
                "required_scope": "write",
                "token_scope": "read",
            },
        ),
        (
            "chat_list_sessions",
            {"project_id": "project-other"},
            {"project-allowed"},
            {
                "tool": "chat_list_sessions",
                "required_scope": "read",
                "token_scope": "read",
                "project_id": "project-other",
            },
        ),
    ],
)
def test_enforce_token_authorization_rejects_scope_or_project_violations(
    tool_name: str,
    params: dict[str, object],
    allowed_project_ids: set[str] | None,
    expected_data: dict[str, str],
) -> None:
    service, _, _, _ = _build_service()
    token_auth = _token_auth(scope="read", allowed_project_ids=allowed_project_ids)

    with pytest.raises(McpRpcError) as exc_info:
        service._enforce_token_authorization(
            actor_user_id=uuid4(),
            token_auth=token_auth,
            tool_name=tool_name,
            params=params,
        )

    assert exc_info.value.code == -32003
    assert exc_info.value.data == expected_data
