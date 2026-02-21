from __future__ import annotations

from types import SimpleNamespace
from unittest.mock import MagicMock
from uuid import uuid4

from app.mcp.errors import McpRpcError
from app.mcp.service import McpService, _StreamChatSendMessageRequest
from app.models import McpJsonRpcRequest


def _build_service() -> McpService:
    return McpService(
        chat_service=MagicMock(),
        project_service=MagicMock(),
        memory_admin_service=MagicMock(),
        embedding_dim=1536,
    )


def _request(*, method: str, params: dict | None = None) -> McpJsonRpcRequest:
    return McpJsonRpcRequest(
        jsonrpc="2.0",
        id="req-1",
        method=method,
        params=params or {},
    )


def test_maybe_stream_direct_chat_send_message_streams_authorized_frames(monkeypatch) -> None:
    service = _build_service()
    actor_user_id = uuid4()
    expected_frames = [{"type": "chunk", "text": "hello"}]
    authorize_tool_call = MagicMock(return_value=({"stream": True}, None))
    stream_chat_send_message = MagicMock(return_value=iter(expected_frames))
    monkeypatch.setattr(service, "_authorize_tool_call", authorize_tool_call)
    monkeypatch.setattr(service, "_stream_chat_send_message", stream_chat_send_message)

    handled, frames = service._maybe_stream_direct_chat_send_message(
        request_ctx=SimpleNamespace(
            actor_user_id=actor_user_id,
            request_id="req-1",
            request_method="chat.send_message",
            canonical_method="chat.send_message",
            request_params={"prompt": "hello"},
            token_auth=None,
        ),
    )

    assert handled is True
    assert list(frames or ()) == expected_frames
    authorize_tool_call.assert_called_once()
    stream_chat_send_message.assert_called_once_with(
        request=_StreamChatSendMessageRequest(
            actor_user_id=actor_user_id,
            request_id="req-1",
            tool_name="chat.send_message",
            params={"stream": True},
        )
    )


def test_maybe_stream_tools_call_chat_message_returns_error_frame_for_invalid_payload(monkeypatch) -> None:
    service = _build_service()
    actor_user_id = uuid4()
    monkeypatch.setattr(
        service,
        "_tool_name_and_params_for_tools_call",
        MagicMock(side_effect=McpRpcError(code=-32602, message="Invalid params", data={"missing": "name"})),
    )

    handled, frames = service._maybe_stream_tools_call_chat_message(
        request_ctx=SimpleNamespace(
            actor_user_id=actor_user_id,
            request_id="req-1",
            request_method="tools/call",
            canonical_method="tools/call",
            request_params={},
            token_auth=None,
        ),
    )

    assert handled is True
    payload = list(frames or ())
    assert len(payload) == 1
    assert payload[0]["error"]["code"] == -32602
    assert payload[0]["error"]["data"] == {"missing": "name"}


def test_stream_dispatch_non_stream_result_emits_mcp_error_frame(monkeypatch) -> None:
    service = _build_service()
    actor_user_id = uuid4()
    actor = {"user_id": str(actor_user_id), "role": "user"}
    request = _request(method="chat.list_sessions")
    monkeypatch.setattr(
        service,
        "_dispatch_non_stream",
        MagicMock(side_effect=McpRpcError(code=-32003, message="Forbidden", data={"tool": "chat.list_sessions"})),
    )

    frames = list(
        service._stream_dispatch_non_stream_result(
            actor=actor,
            actor_user_id=actor_user_id,
            request=request,
            token_auth=None,
        )
    )

    assert len(frames) == 1
    assert frames[0]["error"]["code"] == -32003
    assert frames[0]["error"]["data"] == {"tool": "chat.list_sessions"}


def test_stream_call_routes_to_tool_call_stream_frames(monkeypatch) -> None:
    service = _build_service()
    actor_user_id = uuid4()
    actor = {"user_id": str(actor_user_id), "role": "user"}
    request = _request(method="tools/call", params={"name": "chat_send_message", "arguments": {"stream": True}})
    direct_helper = MagicMock(return_value=(False, None))
    tool_helper = MagicMock(return_value=(True, iter([{"event": "chunk"}])))
    non_stream_helper = MagicMock(return_value=iter([{"event": "done"}]))
    monkeypatch.setattr(service, "_maybe_stream_direct_chat_send_message", direct_helper)
    monkeypatch.setattr(service, "_maybe_stream_tools_call_chat_message", tool_helper)
    monkeypatch.setattr(service, "_stream_dispatch_non_stream_result", non_stream_helper)

    frames = list(service.stream_call(actor=actor, request=request, token_auth=None))

    assert frames == [{"event": "chunk"}]
    direct_helper.assert_called_once()
    tool_helper.assert_called_once()
    non_stream_helper.assert_not_called()


def test_stream_call_routes_to_non_stream_result_when_helpers_skip(monkeypatch) -> None:
    service = _build_service()
    actor_user_id = uuid4()
    actor = {"user_id": str(actor_user_id), "role": "user"}
    request = _request(method="chat.list_sessions")
    monkeypatch.setattr(service, "_maybe_stream_direct_chat_send_message", MagicMock(return_value=(False, None)))
    monkeypatch.setattr(service, "_maybe_stream_tools_call_chat_message", MagicMock(return_value=(False, None)))
    monkeypatch.setattr(
        service,
        "_stream_dispatch_non_stream_result",
        MagicMock(return_value=iter([{"jsonrpc": "2.0", "id": "req-1", "result": {"sessions": []}}])),
    )

    frames = list(service.stream_call(actor=actor, request=request, token_auth=None))

    assert frames == [{"jsonrpc": "2.0", "id": "req-1", "result": {"sessions": []}}]


def test_stream_chat_send_message_requires_done_completion() -> None:
    service = _build_service()
    actor_user_id = uuid4()
    session_id = uuid4()
    service._chat_service.stream_message_events.return_value = iter([("chunk", {"text": "hello"})])
    frames = list(
        service._stream_chat_send_message(
            request=_StreamChatSendMessageRequest(
                actor_user_id=actor_user_id,
                request_id="req-1",
                tool_name="chat.send_message",
                params={"session_id": str(session_id), "content_text": "hi", "stream": True},
                as_tool_call=False,
            )
        )
    )

    assert frames[-1]["error"]["code"] == -32021


def test_stream_chat_send_message_non_stream_as_tool_call_wraps_success() -> None:
    service = _build_service()
    actor_user_id = uuid4()
    session_id = uuid4()
    service._chat_service.send_message.return_value = MagicMock(
        model_dump=MagicMock(return_value={"message_id": "m1"})
    )

    frames = list(
        service._stream_chat_send_message(
            request=_StreamChatSendMessageRequest(
                actor_user_id=actor_user_id,
                request_id="req-1",
                tool_name="chat.send_message",
                params={"session_id": str(session_id), "content_text": "hi", "stream": False},
                as_tool_call=True,
            )
        )
    )

    assert frames == [
        service._success(
            "req-1",
            service._tool_call_success("chat.send_message", {"message": {"message_id": "m1"}}),
        )
    ]
