from __future__ import annotations

import json
from unittest.mock import Mock
from uuid import UUID, uuid4

import pytest
from fastapi import HTTPException
from starlette.requests import Request

from app.chat.api import (
    _actor_user_id,
    _handle_chat_service_error,
    _sse_event,
    create_chat_router,
)
from app.chat.errors import ChatServiceError
from app.chat.service import ChatService


def _build_request() -> Request:
    return Request(
        {
            "type": "http",
            "method": "GET",
            "path": "/api/v1/chat/sessions",
            "headers": [],
        }
    )


def test_actor_user_id_resolves_uuid_from_actor_payload() -> None:
    expected_user_id = uuid4()

    resolved_user_id = _actor_user_id(
        _build_request(),
        lambda _request: {"user_id": str(expected_user_id)},
    )

    assert resolved_user_id == expected_user_id


def test_handle_chat_service_error_returns_success_result() -> None:
    result = _handle_chat_service_error(lambda: {"ok": True})
    assert result == {"ok": True}


def test_handle_chat_service_error_maps_chat_service_exception() -> None:
    def _raise_chat_error() -> dict[str, bool]:
        raise ChatServiceError("forbidden", status_code=403)

    with pytest.raises(HTTPException) as exc_info:
        _handle_chat_service_error(_raise_chat_error)

    assert exc_info.value.status_code == 403
    assert exc_info.value.detail == "forbidden"


def test_sse_event_encodes_payload_line() -> None:
    event = _sse_event("chunk", {"delta": "assistant", "tokens": 2})

    lines = event.splitlines()
    assert lines[0] == "event: chunk"
    assert lines[1].startswith("data: ")
    assert json.loads(lines[1].removeprefix("data: ")) == {"delta": "assistant", "tokens": 2}


def test_create_chat_router_registers_stream_endpoint() -> None:
    chat_service = Mock(spec=ChatService)
    router = create_chat_router(
        chat_service=chat_service,
        require_api_actor=lambda _request: {"user_id": str(UUID(int=1))},
    )
    paths = {route.path for route in router.routes}

    assert "/api/v1/chat/sessions/{session_id}/messages/stream" in paths
