from __future__ import annotations

from datetime import UTC, datetime
from typing import Any
from uuid import UUID

from pydantic import ValidationError

from app.chat.errors import ChatProviderExecutionError, ChatServiceError
from app.chat.service import ChatService
from app.models import (
    ChatMessageCreateRequest,
    ChatSessionCreateRequest,
    ContinueSessionRequest,
    EngramQueryRequest,
    McpJsonRpcRequest,
    MemoryEngramCreate,
    PinEngramRequest,
    SaveSessionAsEngramRequest,
)
from app.repository import create_engram, get_rehydration_bundle, list_engrams, query_engrams

from .errors import McpRpcError


class McpService:
    def __init__(self, chat_service: ChatService, embedding_dim: int) -> None:
        self._chat_service = chat_service
        self._embedding_dim = embedding_dim

    @staticmethod
    def _success(id_value: str | int, result: dict[str, Any]) -> dict[str, Any]:
        return {"jsonrpc": "2.0", "id": id_value, "result": result}

    @staticmethod
    def _error(
        id_value: str | int,
        *,
        code: int,
        message: str,
        data: dict[str, Any] | None = None,
    ) -> dict[str, Any]:
        payload: dict[str, Any] = {"code": code, "message": message}
        if data is not None:
            payload["data"] = data
        return {"jsonrpc": "2.0", "id": id_value, "error": payload}

    @staticmethod
    def _event(
        id_value: str | int,
        *,
        tool: str,
        event_name: str,
        event_payload: dict[str, Any],
    ) -> dict[str, Any]:
        return {
            "jsonrpc": "2.0",
            "method": "mcp.event",
            "params": {
                "id": id_value,
                "tool": tool,
                "event": event_name,
                "data": event_payload,
            },
        }

    @staticmethod
    def _parse_uuid(params: dict[str, Any], key: str) -> UUID:
        raw = params.get(key)
        if raw is None:
            raise McpRpcError(
                code=-32602,
                message="Invalid params",
                data={"missing": key},
            )
        try:
            return UUID(str(raw))
        except ValueError as exc:
            raise McpRpcError(
                code=-32602,
                message="Invalid params",
                data={"invalid": key},
            ) from exc

    @staticmethod
    def _projects_for_user(actor_user_id: UUID, chat_service: ChatService) -> list[str]:
        project_ids: set[str] = set()
        for item in list_engrams(limit=1000, offset=0, actor_user_id=actor_user_id):
            project_ids.add(item.project_id)
        for item in chat_service.list_sessions(
            actor_user_id=actor_user_id,
            project_id=None,
            limit=1000,
            offset=0,
        ):
            project_ids.add(item.project_id)
        return sorted(project_ids)

    def _dispatch_non_stream(
        self,
        *,
        actor: dict[str, Any],
        actor_user_id: UUID,
        request: McpJsonRpcRequest,
    ) -> dict[str, Any]:
        params = request.params
        method = request.method

        if method == "chat.create_session":
            created = self._chat_service.create_session(
                actor_user_id=actor_user_id,
                payload=ChatSessionCreateRequest(**params),
            )
            return {"session": created.model_dump(mode="json")}

        if method == "chat.list_sessions":
            sessions = self._chat_service.list_sessions(
                actor_user_id=actor_user_id,
                project_id=params.get("project_id"),
                limit=int(params.get("limit", 50)),
                offset=int(params.get("offset", 0)),
            )
            return {"sessions": [item.model_dump(mode="json") for item in sessions]}

        if method == "chat.get_session":
            session = self._chat_service.get_session(
                actor_user_id=actor_user_id,
                session_id=self._parse_uuid(params, "session_id"),
            )
            return {"session": session.model_dump(mode="json")}

        if method == "chat.save_as_engram":
            saved = self._chat_service.save_session_as_engram(
                actor_user_id=actor_user_id,
                session_id=self._parse_uuid(params, "session_id"),
                payload=SaveSessionAsEngramRequest(
                    title=params.get("title", ""),
                    abstract=params.get("abstract", ""),
                    visibility_scope=params.get("visibility_scope", "private"),
                    tags=params.get("tags", []),
                    keywords=params.get("keywords", []),
                ),
            )
            return {"saved_engram": saved.model_dump(mode="json")}

        if method == "chat.continue_session":
            continued = self._chat_service.continue_session(
                actor_user_id=actor_user_id,
                session_id=self._parse_uuid(params, "session_id"),
                payload=ContinueSessionRequest(title=params.get("title")),
            )
            return {"continuation": continued.model_dump(mode="json")}

        if method == "engram.create":
            created = create_engram(
                payload=MemoryEngramCreate(**params),
                embedding_dim=self._embedding_dim,
                owner_user_id=actor_user_id,
            )
            return {"engram": created.model_dump(mode="json")}

        if method == "engram.query":
            results = query_engrams(
                request=EngramQueryRequest(**params),
                embedding_dim=self._embedding_dim,
                actor_user_id=actor_user_id,
            )
            return {"results": [item.model_dump(mode="json") for item in results]}

        if method == "engram.rehydrate":
            engram_id = self._parse_uuid(params, "engram_id")
            bundle = get_rehydration_bundle(engram_id, actor_user_id=actor_user_id)
            if not bundle:
                raise McpRpcError(
                    code=-32004,
                    message="Engram not found",
                    data={"engram_id": str(engram_id)},
                )
            return {"bundle": bundle.model_dump(mode="json")}

        if method == "engram.pin_to_session":
            pinned = self._chat_service.pin_engram(
                actor_user_id=actor_user_id,
                session_id=self._parse_uuid(params, "session_id"),
                payload=PinEngramRequest(engram_id=self._parse_uuid(params, "engram_id")),
            )
            return {"pinned": pinned.model_dump(mode="json")}

        if method == "user.get_profile":
            return {"profile": actor}

        if method == "user.list_projects":
            return {"project_ids": self._projects_for_user(actor_user_id, self._chat_service)}

        raise McpRpcError(
            code=-32601,
            message="Method not found",
            data={"method": method},
        )

    def stream_call(
        self,
        *,
        actor: dict[str, Any],
        request: McpJsonRpcRequest,
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

        if request.method == "chat.send_message":
            yield from self._stream_chat_send_message(
                actor_user_id=actor_user_id,
                request=request,
            )
            return

        try:
            result = self._dispatch_non_stream(
                actor=actor,
                actor_user_id=actor_user_id,
                request=request,
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

    def _stream_chat_send_message(
        self,
        *,
        actor_user_id: UUID,
        request: McpJsonRpcRequest,
    ):
        params = request.params
        try:
            session_id = self._parse_uuid(params, "session_id")
            payload = ChatMessageCreateRequest(content_text=params.get("content_text", ""))
            stream_enabled = bool(params.get("stream", True))
            if not stream_enabled:
                response = self._chat_service.send_message(
                    actor_user_id=actor_user_id,
                    session_id=session_id,
                    payload=payload,
                )
                yield self._success(request.id, {"message": response.model_dump(mode="json")})
                return

            final_message: dict[str, Any] | None = None
            for event_name, event_payload in self._chat_service.stream_message_events(
                actor_user_id=actor_user_id,
                session_id=session_id,
                payload=payload,
            ):
                event_payload_dict: dict[str, Any]
                if isinstance(event_payload, dict):
                    event_payload_dict = event_payload
                else:
                    event_payload_dict = {"value": event_payload}

                yield self._event(
                    request.id,
                    tool=request.method,
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

            yield self._success(request.id, {"message": final_message})
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
                data={"detail": str(exc), "timestamp": datetime.now(UTC).isoformat()},
            )
