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
    return _build_service_core(ingestion_service=None)


def _build_service_with_ingestion() -> tuple[McpService, MagicMock, MagicMock, MagicMock, MagicMock]:
    ingestion_service = MagicMock()
    service, chat_service, project_service, memory_admin_service = _build_service_core(
        ingestion_service=ingestion_service
    )
    return service, chat_service, project_service, memory_admin_service, ingestion_service


def _build_service_core(
    *, ingestion_service: MagicMock | None
) -> tuple[McpService, MagicMock, MagicMock, MagicMock]:
    chat_service = MagicMock()
    project_service = MagicMock()
    memory_admin_service = MagicMock()
    service = McpService(
        chat_service=chat_service,
        project_service=project_service,
        memory_admin_service=memory_admin_service,
        ingestion_service=ingestion_service,
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


def _dispatch_for_user(
    service: McpService,
    *,
    actor_user_id: object,
    method: str,
    params: dict[str, object],
) -> dict[str, object]:
    actor = {"user_id": str(actor_user_id), "role": "user"}
    return service._dispatch_tool(
        actor=actor,
        actor_user_id=actor_user_id,
        method=method,
        params=params,
        token_auth=None,
    )


def _mock_owned_collection(
    memory_admin_service: MagicMock,
    *,
    actor_user_id: object,
    collection_id: object,
) -> None:
    memory_admin_service.find_collection.return_value = _Dumpable(
        payload={"collection_id": str(collection_id)},
        owner_user_id=actor_user_id,
    )


def _mock_owned_engram(
    memory_admin_service: MagicMock,
    *,
    actor_user_id: object,
    engram_id: object,
    project_id: str = "project-alpha",
) -> None:
    memory_admin_service.find_engram.return_value = _Dumpable(
        payload={"engram_id": str(engram_id)},
        owner_user_id=actor_user_id,
        project_id=project_id,
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


def test_dispatch_tool_routes_engram_get_with_include_deleted_flag() -> None:
    service, _, _, memory_admin_service = _build_service()
    actor_user_id = uuid4()
    actor = {"user_id": str(actor_user_id), "role": "user"}
    engram_id = uuid4()
    memory_admin_service.find_engram.return_value = _Dumpable(
        payload={"engram_id": str(engram_id)},
        owner_user_id=actor_user_id,
        project_id="project-alpha",
    )

    result = service._dispatch_tool(
        actor=actor,
        actor_user_id=actor_user_id,
        method="engram.get",
        params={"engram_id": str(engram_id), "include_deleted": False},
        token_auth=None,
    )

    assert result == {"engram": {"engram_id": str(engram_id)}}
    memory_admin_service.find_engram.assert_called_once_with(
        engram_id=engram_id,
        include_deleted=False,
    )


def test_dispatch_tool_rehydrate_returns_not_found_when_bundle_missing(monkeypatch) -> None:
    service, _, _, _ = _build_service()
    actor_user_id = uuid4()
    actor = {"user_id": str(actor_user_id), "role": "user"}
    engram_id = uuid4()
    monkeypatch.setattr("app.mcp.service.get_rehydration_bundle", lambda *_args, **_kwargs: None)

    with pytest.raises(McpRpcError) as exc_info:
        service._dispatch_tool(
            actor=actor,
            actor_user_id=actor_user_id,
            method="engram.rehydrate",
            params={"engram_id": str(engram_id)},
            token_auth=None,
        )

    assert exc_info.value.code == -32004
    assert exc_info.value.data == {"engram_id": str(engram_id)}


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


@pytest.mark.parametrize(
    ("method", "params", "expected_code", "expected_data"),
    [
        ("unknown.tool", {}, -32601, {"method": "unknown.tool"}),
        ("chat.save_as_engram", {}, -32602, {"missing": "conversation_markdown"}),
    ],
)
def test_dispatch_tool_returns_expected_rpc_errors_for_invalid_requests(
    method: str,
    params: dict[str, object],
    expected_code: int,
    expected_data: dict[str, str],
) -> None:
    service, _, _, _ = _build_service()
    actor_user_id = uuid4()
    actor = {"user_id": str(actor_user_id), "role": "user"}

    with pytest.raises(McpRpcError) as exc_info:
        service._dispatch_tool(
            actor=actor,
            actor_user_id=actor_user_id,
            method=method,
            params=params,
            token_auth=None,
        )

    assert exc_info.value.code == expected_code
    assert exc_info.value.data == expected_data


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


@pytest.mark.parametrize(
    ("tool_name", "params", "expected_project_id"),
    [
        ("engram.create", {"project_id": " project-alpha "}, "project-alpha"),
        ("chat.list_sessions", {"project_id": "project-beta"}, "project-beta"),
        ("chat.list_sessions", {}, None),
    ],
)
def test_project_id_for_tool_resolves_project_from_input_params(
    tool_name: str,
    params: dict[str, str],
    expected_project_id: str | None,
) -> None:
    service, _, _, _ = _build_service()

    resolved = service._project_id_for_tool(
        actor_user_id=uuid4(),
        tool_name=tool_name,
        params=params,
    )

    assert resolved == expected_project_id


def test_project_id_for_tool_uses_session_project_for_save_as_engram() -> None:
    service, chat_service, _, _ = _build_service()
    actor_user_id = uuid4()
    session_id = uuid4()
    chat_service.get_session.return_value = SimpleNamespace(project_id="project-session")

    resolved = service._project_id_for_tool(
        actor_user_id=actor_user_id,
        tool_name="chat.save_as_engram",
        params={"session_id": str(session_id)},
    )

    assert resolved == "project-session"
    chat_service.get_session.assert_called_once_with(
        actor_user_id=actor_user_id,
        session_id=session_id,
    )


def test_project_id_for_tool_resolves_collection_scoped_tool() -> None:
    service, _, _, memory_admin_service = _build_service()
    collection_id = uuid4()
    memory_admin_service.find_collection.return_value = _Dumpable(
        payload={},
        project_id="project-collection",
    )

    resolved = service._project_id_for_tool(
        actor_user_id=uuid4(),
        tool_name="engram.collection_update",
        params={"collection_id": str(collection_id)},
    )

    assert resolved == "project-collection"
    memory_admin_service.find_collection.assert_called_once_with(
        collection_id=collection_id,
        include_deleted=True,
    )


def test_dispatch_chat_delete_session_routes_memory_admin_service() -> None:
    service, _, _, memory_admin_service = _build_service()
    actor_user_id = uuid4()
    session_id = uuid4()
    memory_admin_service.get_session.return_value = _Dumpable(
        payload={"session_id": str(session_id)},
        owner_user_id=actor_user_id,
    )
    memory_admin_service.delete_session.return_value = _Dumpable(payload={"deleted": True})

    result = _dispatch_for_user(
        service,
        actor_user_id=actor_user_id,
        method="chat.delete_session",
        params={
            "session_id": str(session_id),
            "delete_linked_engrams": True,
            "reason": "cleanup",
        },
    )

    assert result == {"result": {"deleted": True}}
    call_kwargs = memory_admin_service.delete_session.call_args.kwargs
    assert call_kwargs["session_id"] == session_id
    assert call_kwargs["actor_user_id"] == actor_user_id
    assert call_kwargs["payload"].delete_linked_engrams is True
    assert call_kwargs["payload"].reason == "cleanup"


@pytest.mark.parametrize(
    "case",
    [
        {
            "method": "engram.pin_to_session",
            "chat_method": "pin_engram",
            "result_key": "pinned",
            "payload_attr": "engram_id",
            "input_key": "engram_id",
        },
        {
            "method": "chat.continue_session",
            "chat_method": "continue_session",
            "result_key": "continuation",
            "payload_attr": "title",
            "input_key": "title",
        },
    ],
)
def test_dispatch_chat_routes_pin_alias_and_continue_session(case: dict[str, str]) -> None:
    service, chat_service, _, _ = _build_service()
    actor_user_id = uuid4()
    session_id = uuid4()
    input_key = case["input_key"]
    extra_value: str = str(uuid4()) if input_key == "engram_id" else "Resume thread"
    built_params = {"session_id": str(session_id), input_key: extra_value}
    chat_method_mock = getattr(chat_service, case["chat_method"])
    chat_method_mock.return_value = _Dumpable(payload={"ok": True})

    result = _dispatch_for_user(
        service,
        actor_user_id=actor_user_id,
        method=case["method"],
        params=built_params,
    )

    assert result == {case["result_key"]: {"ok": True}}
    call_kwargs = chat_method_mock.call_args.kwargs
    assert call_kwargs["actor_user_id"] == actor_user_id
    assert call_kwargs["session_id"] == session_id
    payload_value = getattr(call_kwargs["payload"], case["payload_attr"])
    if input_key == "engram_id":
        assert str(payload_value) == extra_value
    else:
        assert payload_value == extra_value


def test_dispatch_chat_unpin_document_routes_chat_service() -> None:
    service, chat_service, _, _ = _build_service()
    actor_user_id = uuid4()
    session_id = uuid4()
    document_id = uuid4()

    result = _dispatch_for_user(
        service,
        actor_user_id=actor_user_id,
        method="chat.unpin_document",
        params={"session_id": str(session_id), "document_id": str(document_id)},
    )

    assert result == {"removed": True}
    chat_service.unpin_document.assert_called_once_with(
        actor_user_id=actor_user_id,
        session_id=session_id,
        document_id=document_id,
    )


def test_dispatch_chat_update_lifecycle_policy_routes_chat_service() -> None:
    service, chat_service, _, _ = _build_service()
    actor_user_id = uuid4()
    session_id = uuid4()
    chat_service.update_lifecycle_policy.return_value = _Dumpable(
        payload={"autosave_enabled": True}
    )

    result = _dispatch_for_user(
        service,
        actor_user_id=actor_user_id,
        method="chat.update_lifecycle_policy",
        params={
            "session_id": str(session_id),
            "autosave_enabled": True,
            "autosave_strategy": "interval",
            "autosave_interval_minutes": 10,
            "autosave_min_messages": 2,
            "retention_days": 30,
            "retention_max_snapshots": 4,
        },
    )

    assert result == {"lifecycle_policy": {"autosave_enabled": True}}
    call_kwargs = chat_service.update_lifecycle_policy.call_args.kwargs
    assert call_kwargs["actor_user_id"] == actor_user_id
    assert call_kwargs["session_id"] == session_id
    payload = call_kwargs["payload"]
    assert payload.autosave_enabled is True
    assert payload.autosave_strategy == "interval"
    assert payload.autosave_interval_minutes == 10
    assert payload.autosave_min_messages == 2
    assert payload.retention_days == 30
    assert payload.retention_max_snapshots == 4


def test_dispatch_chat_list_project_documents_requires_ingestion_service() -> None:
    service, _, _, _ = _build_service()
    actor_user_id = uuid4()

    with pytest.raises(McpRpcError) as exc_info:
        _dispatch_for_user(
            service,
            actor_user_id=actor_user_id,
            method="chat.list_project_documents",
            params={},
        )

    assert exc_info.value.code == -32000
    assert exc_info.value.data == {"method": "chat.list_project_documents"}


def test_dispatch_chat_list_project_documents_routes_ingestion_service() -> None:
    service, _, _, _, ingestion_service = _build_service_with_ingestion()
    actor_user_id = uuid4()
    ingestion_service.list_documents.return_value = [_Dumpable(payload={"document_id": "d1"})]

    result = _dispatch_for_user(
        service,
        actor_user_id=actor_user_id,
        method="chat.list_project_documents",
        params={"project_id": "project-alpha", "limit": 10, "offset": 5},
    )

    assert result == {"documents": [{"document_id": "d1"}]}
    ingestion_service.list_documents.assert_called_once_with(
        actor_user_id=actor_user_id,
        project_id="project-alpha",
        limit=10,
        offset=5,
    )


def test_dispatch_engram_collection_add_items_parses_uuid_list_payload() -> None:
    service, _, _, memory_admin_service = _build_service()
    actor_user_id = uuid4()
    collection_id = uuid4()
    engram_id_a = uuid4()
    engram_id_b = uuid4()
    _mock_owned_collection(
        memory_admin_service,
        actor_user_id=actor_user_id,
        collection_id=collection_id,
    )
    memory_admin_service.add_collection_items.return_value = {"updated": True}

    result = _dispatch_for_user(
        service,
        actor_user_id=actor_user_id,
        method="engram.collection_add_items",
        params={
            "collection_id": str(collection_id),
            "engram_ids": [str(engram_id_a), str(engram_id_b)],
        },
    )

    assert result == {"result": {"updated": True}}
    call_kwargs = memory_admin_service.add_collection_items.call_args.kwargs
    assert call_kwargs["collection_id"] == collection_id
    assert call_kwargs["actor_user_id"] == actor_user_id
    assert call_kwargs["payload"].engram_ids == [engram_id_a, engram_id_b]


@pytest.mark.parametrize(
    ("method", "resource_key", "delete_mock_name", "delete_return_value"),
    [
        ("engram.collection_delete", "collection_id", "delete_collection", {"deleted": True}),
        ("engram.delete", "engram_id", "delete_engram", _Dumpable(payload={"deleted": True})),
    ],
)
def test_dispatch_delete_routes_memory_admin_service(
    method: str,
    resource_key: str,
    delete_mock_name: str,
    delete_return_value: object,
) -> None:
    service, _, _, memory_admin_service = _build_service()
    actor_user_id = uuid4()
    resource_id = uuid4()
    if resource_key == "collection_id":
        _mock_owned_collection(
            memory_admin_service,
            actor_user_id=actor_user_id,
            collection_id=resource_id,
        )
    else:
        _mock_owned_engram(
            memory_admin_service,
            actor_user_id=actor_user_id,
            engram_id=resource_id,
        )
    getattr(memory_admin_service, delete_mock_name).return_value = delete_return_value

    result = _dispatch_for_user(
        service,
        actor_user_id=actor_user_id,
        method=method,
        params={resource_key: str(resource_id), "reason": "cleanup"},
    )

    assert result == {"result": {"deleted": True}}
    call_kwargs = getattr(memory_admin_service, delete_mock_name).call_args.kwargs
    assert call_kwargs[resource_key] == resource_id
    assert call_kwargs["actor_user_id"] == actor_user_id
    assert call_kwargs["payload"].reason == "cleanup"


def test_dispatch_engram_move_project_rejects_disallowed_source_project_for_token() -> None:
    service, _, _, memory_admin_service = _build_service()
    actor_user_id = uuid4()
    actor = {"user_id": str(actor_user_id), "role": "user"}
    engram_id = uuid4()
    _mock_owned_engram(
        memory_admin_service,
        actor_user_id=actor_user_id,
        engram_id=engram_id,
        project_id="source-project",
    )
    token_auth = _token_auth(scope="write", allowed_project_ids={"target-project"})

    with pytest.raises(McpRpcError) as exc_info:
        service._dispatch_tool(
            actor=actor,
            actor_user_id=actor_user_id,
            method="engram.move_project",
            params={"engram_id": str(engram_id), "target_project_id": "target-project"},
            token_auth=token_auth,
        )

    assert exc_info.value.code == -32003
    assert exc_info.value.data == {
        "tool": "engram.move_project",
        "project_id": "source-project",
        "token_scope": "write",
    }
    memory_admin_service.move_engram.assert_not_called()
