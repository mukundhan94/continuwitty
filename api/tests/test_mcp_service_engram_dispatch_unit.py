from __future__ import annotations

from dataclasses import dataclass
from datetime import UTC, datetime
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


@dataclass(frozen=True)
class _PrimaryDispatchCase:
    method: str
    helper_name: str
    actor_role: str
    params: dict[str, str]


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


def test_dispatch_tool_routes_engram_collection_list() -> None:
    service, _, _, memory_admin_service = _build_service()
    actor_user_id = uuid4()
    actor = {"user_id": str(actor_user_id), "role": "user"}
    memory_admin_service.list_collections.return_value = [_Dumpable(payload={"collection_id": "c1"})]

    result = service._dispatch_tool(
        actor=actor,
        actor_user_id=actor_user_id,
        method="engram.collection_list",
        params={},
        token_auth=None,
    )

    assert result == {"collections": [{"collection_id": "c1"}]}


def test_dispatch_tool_routes_engram_get_with_include_deleted_flag() -> None:
    service, _, _, memory_admin_service = _build_service()
    actor_user_id = uuid4()
    actor = {"user_id": str(actor_user_id), "role": "user"}
    engram_id = uuid4()
    _mock_owned_engram(
        memory_admin_service,
        actor_user_id=actor_user_id,
        engram_id=engram_id,
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


def test_dispatch_tool_routes_engram_update_with_payload_mapping() -> None:
    service, _, _, memory_admin_service = _build_service()
    actor_user_id = uuid4()
    actor = {"user_id": str(actor_user_id), "role": "user"}
    engram_id = uuid4()
    _mock_owned_engram(
        memory_admin_service,
        actor_user_id=actor_user_id,
        engram_id=engram_id,
    )
    memory_admin_service.update_engram.return_value = _Dumpable(payload={"engram_id": str(engram_id)})

    result = service._dispatch_tool(
        actor=actor,
        actor_user_id=actor_user_id,
        method="engram.update",
        params={
            "engram_id": str(engram_id),
            "title": "Updated title",
            "abstract": "Updated abstract",
            "detailed_summary_markdown": "## Updated",
            "tags": ["incident"],
            "keywords": ["memory"],
            "visibility_scope": "private",
            "expected_updated_at": "2026-02-21T00:00:00Z",
            "sources": [
                {
                    "captured_at": "2026-02-21T00:00:00Z",
                    "url": "https://example.com/m1",
                    "title": "Message source",
                    "snippet": "snippet",
                }
            ],
        },
        token_auth=None,
    )

    assert result == {"engram": {"engram_id": str(engram_id)}}
    call_kwargs = memory_admin_service.update_engram.call_args.kwargs
    assert call_kwargs["engram_id"] == engram_id
    assert call_kwargs["actor_user_id"] == actor_user_id
    payload = call_kwargs["payload"]
    assert payload.title == "Updated title"
    assert payload.abstract == "Updated abstract"
    assert payload.detailed_summary_markdown == "## Updated"
    assert payload.tags == ["incident"]
    assert payload.keywords == ["memory"]
    assert getattr(payload.visibility_scope, "value", payload.visibility_scope) == "private"
    assert payload.expected_updated_at == datetime(2026, 2, 21, 0, 0, tzinfo=UTC)
    assert len(payload.sources) == 1
    assert payload.sources[0].url == "https://example.com/m1"


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


@pytest.mark.parametrize(
    "case",
    [
        _PrimaryDispatchCase(
            method="engram.create",
            helper_name="_dispatch_engram_create_tool",
            actor_role="user",
            params={"engram_id": "e-1"},
        ),
        _PrimaryDispatchCase(
            method="engram.create_from_conversation",
            helper_name="_dispatch_engram_create_from_conversation_tool",
            actor_role="user",
            params={"engram_id": "e-1"},
        ),
        _PrimaryDispatchCase(
            method="engram.update",
            helper_name="_dispatch_engram_update_tool",
            actor_role="user",
            params={"engram_id": "e-1"},
        ),
        _PrimaryDispatchCase(
            method="engram.collection_create",
            helper_name="_dispatch_engram_collection_create_tool",
            actor_role="admin",
            params={"name": "Collection A"},
        ),
    ],
)
def test_dispatch_engram_primary_routes_extracted_helpers(
    monkeypatch,
    case: _PrimaryDispatchCase,
) -> None:
    service, _, _, _ = _build_service()
    actor_user_id = uuid4()
    actor = {"user_id": str(actor_user_id), "role": case.actor_role}
    expected = {"method": case.method}
    helper = MagicMock(return_value=expected)
    monkeypatch.setattr(service, case.helper_name, helper)

    result = service._dispatch_engram_primary_tool(
        actor=actor,
        actor_user_id=actor_user_id,
        method=case.method,
        params=case.params,
        token_auth=None,
    )

    assert result == expected
    expected_call = {
        "actor_user_id": actor_user_id,
        "params": case.params,
        "token_auth": None,
    }
    if case.method == "engram.collection_create":
        expected_call["actor_role"] = case.actor_role
    else:
        expected_call["actor"] = actor
    helper.assert_called_once_with(**expected_call)
