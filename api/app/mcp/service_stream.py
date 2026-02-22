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
    StreamChatSendMessageEventsRequest,
    StreamSuccessFrameRequest,
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
    @staticmethod
    def _http_exception_to_mcp_error(exc: HTTPException) -> tuple[int, str, dict[str, Any]]:
        status_code = exc.status_code
        detail = str(exc.detail)
        lower_detail = detail.lower()

        if status_code in {401, 403}:
            return (
                -32003,
                "Forbidden",
                {
                    "status_code": status_code,
                    "detail": detail,
                    "suggested_action": "check_mcp_token_scope_or_resource_permissions",
                },
            )

        if status_code == 409:
            suggested_action = "resolve_conflict_and_retry"
            if "already exists" in lower_detail and "collection" in lower_detail:
                suggested_action = "use_unique_collection_name_or_update_existing_collection"
            return (
                -32009,
                detail,
                {
                    "status_code": status_code,
                    "detail": detail,
                    "suggested_action": suggested_action,
                },
            )

        if status_code >= 500:
            return (
                -32000,
                "Internal MCP error",
                {
                    "status_code": status_code,
                    "detail": detail,
                },
            )

        return (
            -32602,
            "Invalid params",
            {
                "status_code": status_code,
                "detail": detail,
            },
        )

    def _error_for_non_stream_exception(
        self,
        *,
        request_id: str | int | None,
        exc: Exception,
    ) -> dict[str, Any]:
        if isinstance(exc, McpRpcError):
            return self._error(
                request_id,
                code=exc.code,
                message=exc.message,
                data=exc.data,
            )
        if isinstance(exc, ValidationError):
            return self._error(
                request_id,
                code=-32602,
                message="Invalid params",
                data={"errors": exc.errors()},
            )
        if isinstance(exc, HTTPException):
            error_code, message, data = self._http_exception_to_mcp_error(exc)
            return self._error(
                request_id,
                code=error_code,
                message=message,
                data=data,
            )
        if isinstance(exc, ChatProviderExecutionError):
            return self._error(
                request_id,
                code=-32020,
                message=exc.detail,
                data={"error_code": exc.error_code, "status_code": exc.status_code},
            )
        if isinstance(exc, ChatServiceError):
            return self._error(
                request_id,
                code=-32010,
                message=exc.detail,
                data={"status_code": exc.status_code},
            )
        return self._error(
            request_id,
            code=-32000,
            message="Internal MCP error",
            data={"detail": str(exc)},
        )

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
        except Exception as exc:
            yield self._error_for_non_stream_exception(
                request_id=request.id,
                exc=exc,
            )

    def _stream_route_request_context(
        self,
        *,
        actor: dict[str, Any],
        request: McpJsonRpcRequest,
        token_auth: McpTokenAuthContext | None,
    ) -> tuple[_StreamRouteRequest | None, dict[str, Any] | None]:
        if request.jsonrpc != "2.0":
            return None, self._error(
                request.id,
                code=-32600,
                message="Invalid Request",
                data={"jsonrpc": request.jsonrpc},
            )

        try:
            actor_user_id = UUID(str(actor["user_id"]))
        except Exception:
            return None, self._error(
                request.id,
                code=-32001,
                message="Unauthorized actor context",
                data={"user_id": actor.get("user_id")},
            )

        return (
            _StreamRouteRequest(
                actor_user_id=actor_user_id,
                request_id=request.id,
                request_method=request.method,
                canonical_method=self._canonical_tool_name(request.method),
                request_params=request.params,
                token_auth=token_auth,
            ),
            None,
        )

    @staticmethod
    def _stream_handled_result(
        *,
        result: tuple[bool, Iterator[dict[str, Any]] | None],
    ) -> tuple[bool, Iterator[dict[str, Any]]]:
        handled, frames = result
        if not handled or frames is None:
            return False, iter(())
        return True, frames

    def stream_call(
        self,
        *,
        actor: dict[str, Any],
        request: McpJsonRpcRequest,
        token_auth: McpTokenAuthContext | None = None,
    ):
        request_ctx, error_frame = self._stream_route_request_context(
            actor=actor,
            request=request,
            token_auth=token_auth,
        )
        if error_frame is not None:
            yield error_frame
            return
        assert request_ctx is not None

        handled, frames = self._stream_handled_result(
            result=self._maybe_stream_direct_chat_send_message(
                request_ctx=request_ctx,
            )
        )
        if handled:
            yield from frames
            return

        handled, frames = self._stream_handled_result(
            result=self._maybe_stream_tools_call_chat_message(
                request_ctx=request_ctx,
            )
        )
        if handled:
            yield from frames
            return

        yield from self._stream_dispatch_non_stream_result(
            actor=actor,
            actor_user_id=request_ctx.actor_user_id,
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
                yield self._stream_non_stream_chat_success_frame(
                    request=request,
                    session_id=session_id,
                    payload=payload,
                )
                return

            final_message = yield from self._stream_chat_send_message_events(
                request=request,
                session_id=session_id,
                payload=payload,
            )
            yield chat_send_message_success_frame(
                request=StreamSuccessFrameRequest(
                    request_id=request.request_id,
                    tool_name=request.tool_name,
                    payload={"message": final_message},
                    as_tool_call=request.as_tool_call,
                ),
                success=self._success,
                tool_call_success=self._tool_call_success,
            )
        except Exception as exc:
            yield stream_chat_send_message_error_frame(
                request_id=request.request_id,
                exc=exc,
                error=self._error,
            )

    def _stream_non_stream_chat_success_frame(
        self,
        *,
        request: _StreamChatSendMessageRequest,
        session_id: UUID,
        payload: ChatMessageCreateRequest,
    ) -> dict[str, Any]:
        response = self._chat_service.send_message(
            actor_user_id=request.actor_user_id,
            session_id=session_id,
            payload=payload,
        )
        return chat_send_message_success_frame(
            request=StreamSuccessFrameRequest(
                request_id=request.request_id,
                tool_name=request.tool_name,
                payload={"message": response.model_dump(mode="json")},
                as_tool_call=request.as_tool_call,
            ),
            success=self._success,
            tool_call_success=self._tool_call_success,
        )

    def _stream_chat_send_message_events(
        self,
        *,
        request: _StreamChatSendMessageRequest,
        session_id: UUID,
        payload: ChatMessageCreateRequest,
    ) -> Iterator[dict[str, Any]]:
        return stream_chat_send_message_events(
            request=StreamChatSendMessageEventsRequest(
                chat_service=self._chat_service,
                actor_user_id=request.actor_user_id,
                session_id=session_id,
                payload=payload,
                request_id=request.request_id,
                tool_name=request.tool_name,
            ),
            event=self._event,
        )
