from __future__ import annotations

from dataclasses import dataclass
from datetime import UTC, datetime
from typing import Any
from uuid import UUID

from pydantic import ValidationError

from app.chat.errors import ChatProviderExecutionError, ChatServiceError

from .errors import McpRpcError


@dataclass(frozen=True)
class StreamSuccessFrameRequest:
    request_id: str | int
    tool_name: str
    payload: dict[str, Any]
    as_tool_call: bool


@dataclass(frozen=True)
class StreamChatSendMessageEventsRequest:
    chat_service: Any
    actor_user_id: UUID
    session_id: UUID
    payload: Any
    request_id: str | int
    tool_name: str


def chat_send_message_success_frame(
    *,
    request: StreamSuccessFrameRequest,
    success: Any,
    tool_call_success: Any,
) -> dict[str, Any]:
    if request.as_tool_call:
        return success(
            request.request_id,
            tool_call_success(request.tool_name, request.payload),
        )
    return success(request.request_id, request.payload)


def stream_event_payload(event_payload: Any) -> dict[str, Any]:
    if isinstance(event_payload, dict):
        return event_payload
    return {"value": event_payload}


def stream_chat_send_message_events(
    *,
    request: StreamChatSendMessageEventsRequest,
    event: Any,
):
    final_message: dict[str, Any] | None = None
    for event_name, event_payload in request.chat_service.stream_message_events(
        actor_user_id=request.actor_user_id,
        session_id=request.session_id,
        payload=request.payload,
    ):
        event_payload_dict = stream_event_payload(event_payload)
        yield event(
            request.request_id,
            tool=request.tool_name,
            event_name=event_name,
            event_payload=event_payload_dict,
        )
        if event_name == "error":
            raise McpRpcError(
                code=-32020,
                message=event_payload_dict.get("detail", "chat.send_message stream failed"),
                data=event_payload_dict,
            )
        if event_name == "done":
            final_message = event_payload_dict
    if final_message is None:
        raise McpRpcError(
            code=-32021,
            message="chat.send_message stream ended without completion",
        )
    return final_message


def stream_chat_send_message_error_frame(
    *,
    request_id: str | int,
    exc: Exception,
    error: Any,
) -> dict[str, Any]:
    if isinstance(exc, McpRpcError):
        return error(
            request_id,
            code=exc.code,
            message=exc.message,
            data=exc.data,
        )
    if isinstance(exc, ValidationError):
        return error(
            request_id,
            code=-32602,
            message="Invalid params",
            data={"errors": exc.errors()},
        )
    if isinstance(exc, ChatProviderExecutionError):
        return error(
            request_id,
            code=-32020,
            message=exc.detail,
            data={"error_code": exc.error_code, "status_code": exc.status_code},
        )
    if isinstance(exc, ChatServiceError):
        return error(
            request_id,
            code=-32010,
            message=exc.detail,
            data={"status_code": exc.status_code},
        )
    return error(
        request_id,
        code=-32000,
        message="Internal MCP error",
        data={"detail": str(exc), "timestamp": datetime.now(UTC).isoformat()},
    )
