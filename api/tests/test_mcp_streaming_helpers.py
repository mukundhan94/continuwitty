from __future__ import annotations

from types import SimpleNamespace
from uuid import uuid4

import pytest

from app.mcp.errors import McpRpcError
from app.mcp.streaming import (
    StreamChatSendMessageEventsRequest,
    StreamSuccessFrameRequest,
    chat_send_message_success_frame,
    stream_chat_send_message_events,
    stream_event_payload,
)


def test_chat_send_message_success_frame_wraps_tool_call_payload() -> None:
    frame = chat_send_message_success_frame(
        request=StreamSuccessFrameRequest(
            request_id="req-1",
            tool_name="chat.send_message",
            payload={"message": {"message_id": "m1"}},
            as_tool_call=True,
        ),
        success=lambda request_id, payload: {"id": request_id, "payload": payload},
        tool_call_success=lambda tool_name, payload: {"tool_name": tool_name, "wrapped": payload},
    )

    assert frame == {
        "id": "req-1",
        "payload": {
            "tool_name": "chat.send_message",
            "wrapped": {"message": {"message_id": "m1"}},
        },
    }


def test_stream_event_payload_wraps_non_dict_values() -> None:
    assert stream_event_payload({"chunk": "hello"}) == {"chunk": "hello"}
    assert stream_event_payload("hello") == {"value": "hello"}


def test_stream_chat_send_message_events_returns_done_payload() -> None:
    request = StreamChatSendMessageEventsRequest(
        chat_service=SimpleNamespace(
            stream_message_events=lambda **_kwargs: iter(
                [
                    ("chunk", "hello"),
                    ("done", {"message_id": "m1"}),
                ]
            )
        ),
        actor_user_id=uuid4(),
        session_id=uuid4(),
        payload={"content_text": "hi"},
        request_id="req-1",
        tool_name="chat.send_message",
    )

    frames: list[dict[str, object]] = []
    generator = stream_chat_send_message_events(
        request=request,
        event=lambda request_id, **payload: {"request_id": request_id, **payload},
    )

    while True:
        try:
            frames.append(next(generator))
        except StopIteration as stop:
            final_payload = stop.value
            break

    assert frames == [
        {
            "request_id": "req-1",
            "tool": "chat.send_message",
            "event_name": "chunk",
            "event_payload": {"value": "hello"},
        },
        {
            "request_id": "req-1",
            "tool": "chat.send_message",
            "event_name": "done",
            "event_payload": {"message_id": "m1"},
        },
    ]
    assert final_payload == {"message_id": "m1"}


def test_stream_chat_send_message_events_raises_mcp_error_on_error_event() -> None:
    request = StreamChatSendMessageEventsRequest(
        chat_service=SimpleNamespace(
            stream_message_events=lambda **_kwargs: iter(
                [("error", {"detail": "provider failed", "error_code": "bad_gateway"})]
            )
        ),
        actor_user_id=uuid4(),
        session_id=uuid4(),
        payload={"content_text": "hi"},
        request_id="req-1",
        tool_name="chat.send_message",
    )

    generator = stream_chat_send_message_events(
        request=request,
        event=lambda request_id, **payload: {"request_id": request_id, **payload},
    )

    with pytest.raises(McpRpcError, match="provider failed") as exc_info:
        list(generator)

    assert exc_info.value.code == -32020
    assert exc_info.value.data == {"detail": "provider failed", "error_code": "bad_gateway"}
