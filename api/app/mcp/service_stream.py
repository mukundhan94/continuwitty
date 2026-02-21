from __future__ import annotations

from collections.abc import Iterator
from dataclasses import dataclass
from typing import Any
from uuid import UUID

from fastapi import HTTPException
from pydantic import ValidationError

from app.chat.errors import ChatProviderExecutionError, ChatServiceError
from app.mcp_tokens import McpTokenAuthContext
from app.models import ChatMessageCreateRequest, McpJsonRpcRequest

from .errors import McpRpcError
from .streaming import (
    chat_send_message_success_frame,
    stream_chat_send_message_error_frame,
    stream_chat_send_message_events,
)


@dataclass(frozen=True)
class _StreamRouteRequest:
    actor_user_id: UUID
    request_id: str | int | None
    request_method: str
    canonical_method: str
    request_params: dict[str, Any]
    token_auth: McpTokenAuthContext | None


@dataclass(frozen=True)
class _AuthorizeToolCallRequest:
    actor_user_id: UUID
    request_id: str | int | None
    token_auth: McpTokenAuthContext | None
    tool_name: str
    params: dict[str, Any]


@dataclass(frozen=True)
class _StreamChatSendMessageRequest:
    actor_user_id: UUID
    request_id: str | int
    tool_name: str
    params: dict[str, Any]
    as_tool_call: bool = False


class McpServiceStreamMixin:
    def _authorize_tool_call(
        self,
        *,
        request: _AuthorizeToolCallRequest,
    ) -> tuple[dict[str, Any] | None, dict[str, Any] | None]:
        try:
            authorized_params = self._enforce_token_authorization(
                actor_user_id=request.actor_user_id,
                token_auth=request.token_auth,
                tool_name=request.tool_name,
                params=request.params,
            )
        except McpRpcError as exc:
            return None, self._error(request.request_id, code=exc.code, message=exc.message, data=exc.data)
        return authorized_params, None

    def _maybe_stream_direct_chat_send_message(
        self,
        *,
        request_ctx: _StreamRouteRequest,
    ) -> tuple[bool, Iterator[dict[str, Any]] | None]:
        if request_ctx.canonical_method != "chat.send_message":
            return False, None
        authorized_params, error_frame = self._authorize_tool_call(
            request=_AuthorizeToolCallRequest(
                actor_user_id=request_ctx.actor_user_id,
                request_id=request_ctx.request_id,
                token_auth=request_ctx.token_auth,
                tool_name=request_ctx.request_method,
                params=request_ctx.request_params,
            ),
        )
        if error_frame is not None:
            return True, iter((error_frame,))
        assert authorized_params is not None
        return (
            True,
            self._stream_chat_send_message(
                request=_StreamChatSendMessageRequest(
                    actor_user_id=request_ctx.actor_user_id,
                    request_id=request_ctx.request_id,
                    tool_name=request_ctx.request_method,
                    params=authorized_params,
                ),
            ),
        )

    def _maybe_stream_tools_call_chat_message(
        self,
        *,
        request_ctx: _StreamRouteRequest,
    ) -> tuple[bool, Iterator[dict[str, Any]] | None]:
        if request_ctx.request_method != "tools/call":
            return False, None

        try:
            tool_name, tool_params = self._tool_name_and_params_for_tools_call(
                request_ctx.request_params
            )
            canonical_tool_name = self._canonical_tool_name(tool_name)
        except McpRpcError as exc:
            return (
                True,
                iter(
                    (
                        self._error(
                            request_ctx.request_id,
                            code=exc.code,
                            message=exc.message,
                            data=exc.data,
                        ),
                    )
                ),
            )

        authorized_tool_params, error_frame = self._authorize_tool_call(
            request=_AuthorizeToolCallRequest(
                actor_user_id=request_ctx.actor_user_id,
                request_id=request_ctx.request_id,
                token_auth=request_ctx.token_auth,
                tool_name=tool_name,
                params=tool_params,
            ),
        )
        if error_frame is not None:
            return True, iter((error_frame,))
        assert authorized_tool_params is not None

        if canonical_tool_name == "chat.send_message" and bool(authorized_tool_params.get("stream", True)):
            return (
                True,
                self._stream_chat_send_message(
                    request=_StreamChatSendMessageRequest(
                        actor_user_id=request_ctx.actor_user_id,
                        request_id=request_ctx.request_id,
                        tool_name=tool_name,
                        params=authorized_tool_params,
                        as_tool_call=True,
                    ),
                ),
            )
        return False, None

    def _stream_dispatch_non_stream_result(
        self,
        *,
        actor: dict[str, Any],
        actor_user_id: UUID,
        request: McpJsonRpcRequest,
        token_auth: McpTokenAuthContext | None,
    ) -> Iterator[dict[str, Any]]:
        try:
            result = self._dispatch_non_stream(
                actor=actor,
                actor_user_id=actor_user_id,
                request=request,
                token_auth=token_auth,
            )
            yield self._success(request.id, result)
        except McpRpcError as exc:
            yield self._error(
                request.id,
                code=exc.code,
                message=exc.message,
                data=exc.data,
            )
        except ValidationError as exc:
            yield self._error(
                request.id,
                code=-32602,
                message="Invalid params",
                data={"errors": exc.errors()},
            )
        except HTTPException as exc:
            # Surface REST-style validation/authorization failures as structured
            # MCP errors without leaking transport-specific status handling.
            error_code = -32003 if exc.status_code in {401, 403} else -32602
            yield self._error(
                request.id,
                code=error_code,
                message="Invalid params" if exc.status_code < 500 else "Internal MCP error",
                data={"status_code": exc.status_code, "detail": str(exc.detail)},
            )
        except ChatProviderExecutionError as exc:
            yield self._error(
                request.id,
                code=-32020,
                message=exc.detail,
                data={"error_code": exc.error_code, "status_code": exc.status_code},
            )
        except ChatServiceError as exc:
            yield self._error(
                request.id,
                code=-32010,
                message=exc.detail,
                data={"status_code": exc.status_code},
            )
        except Exception as exc:
            yield self._error(
                request.id,
                code=-32000,
                message="Internal MCP error",
                data={"detail": str(exc)},
            )

    def stream_call(
        self,
        *,
        actor: dict[str, Any],
        request: McpJsonRpcRequest,
        token_auth: McpTokenAuthContext | None = None,
    ):
        if request.jsonrpc != "2.0":
            yield self._error(
                request.id,
                code=-32600,
                message="Invalid Request",
                data={"jsonrpc": request.jsonrpc},
            )
            return

        try:
            actor_user_id = UUID(str(actor["user_id"]))
        except Exception:
            yield self._error(
                request.id,
                code=-32001,
                message="Unauthorized actor context",
                data={"user_id": actor.get("user_id")},
            )
            return

        canonical_method = self._canonical_tool_name(request.method)
        request_ctx = _StreamRouteRequest(
            actor_user_id=actor_user_id,
            request_id=request.id,
            request_method=request.method,
            canonical_method=canonical_method,
            request_params=request.params,
            token_auth=token_auth,
        )
        handled, direct_frames = self._maybe_stream_direct_chat_send_message(
            request_ctx=request_ctx,
        )
        if handled:
            assert direct_frames is not None
            yield from direct_frames
            return

        handled, tool_call_frames = self._maybe_stream_tools_call_chat_message(
            request_ctx=request_ctx,
        )
        if handled:
            assert tool_call_frames is not None
            yield from tool_call_frames
            return

        yield from self._stream_dispatch_non_stream_result(
            actor=actor,
            actor_user_id=actor_user_id,
            request=request,
            token_auth=token_auth,
        )

    def handle_notification(
        self,
        *,
        request: McpJsonRpcRequest,
    ) -> None:
        """Best-effort handling for JSON-RPC notifications.

        Streamable HTTP clients (including VS Code MCP) can send lifecycle
        notifications such as `notifications/initialized` without an `id`.
        We currently do not require side effects for these notifications, so we
        accept and ignore them to maintain protocol compatibility.
        """
        _ = request

    def _stream_chat_send_message(
        self,
        *,
        request: _StreamChatSendMessageRequest,
    ):
        # This method emits progress frames (`mcp.event`) plus a final success/error
        # JSON-RPC frame. `as_tool_call=True` wraps the final success payload in the
        # `tools/call` envelope so external MCP clients get a consistent shape.
        try:
            session_id = self._parse_uuid(request.params, "session_id")
            payload = ChatMessageCreateRequest(content_text=request.params.get("content_text", ""))
            stream_enabled = bool(request.params.get("stream", True))
            if not stream_enabled:
                response = self._chat_service.send_message(
                    actor_user_id=request.actor_user_id,
                    session_id=session_id,
                    payload=payload,
                )
                yield chat_send_message_success_frame(
                    request_id=request.request_id,
                    tool_name=request.tool_name,
                    payload={"message": response.model_dump(mode="json")},
                    as_tool_call=request.as_tool_call,
                    success=self._success,
                    tool_call_success=self._tool_call_success,
                )
                return

            final_message = yield from stream_chat_send_message_events(
                chat_service=self._chat_service,
                actor_user_id=request.actor_user_id,
                session_id=session_id,
                payload=payload,
                request_id=request.request_id,
                tool_name=request.tool_name,
                event=self._event,
            )
            yield chat_send_message_success_frame(
                request_id=request.request_id,
                tool_name=request.tool_name,
                payload={"message": final_message},
                as_tool_call=request.as_tool_call,
                success=self._success,
                tool_call_success=self._tool_call_success,
            )
        except Exception as exc:
            yield stream_chat_send_message_error_frame(
                request_id=request.request_id,
                exc=exc,
                error=self._error,
            )
