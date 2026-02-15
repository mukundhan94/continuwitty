from __future__ import annotations

import json
from collections.abc import Callable
from typing import Any
from uuid import UUID

from fastapi import APIRouter, HTTPException, Query, Request
from fastapi.encoders import jsonable_encoder
from fastapi.responses import StreamingResponse

from app.models import (
    ChatMessageCreateRequest,
    ChatMessageRecord,
    ChatSendResponse,
    ChatSessionCreateRequest,
    ChatSessionRecord,
    ChatSessionUpdateRequest,
    ContinueSessionRequest,
    ContinueSessionResponse,
    EngramSummary,
    PinEngramRequest,
    PinnedEngramRecord,
    SaveSessionAsEngramRequest,
    SaveSessionAsEngramResponse,
)

from .errors import ChatServiceError
from .service import ChatService


def _to_http_exception(exc: ChatServiceError) -> HTTPException:
    return HTTPException(status_code=exc.status_code, detail=exc.detail)


def _sse_event(event: str, payload: dict[str, Any]) -> str:
    data = json.dumps(jsonable_encoder(payload))
    return f"event: {event}\ndata: {data}\n\n"


def create_chat_router(
    *,
    chat_service: ChatService,
    require_api_actor: Callable[[Request], dict[str, Any]],
) -> APIRouter:
    router = APIRouter(prefix="/api/v1/chat", tags=["chat"])

    @router.post("/sessions", response_model=ChatSessionRecord, status_code=201)
    def create_session(request: Request, payload: ChatSessionCreateRequest) -> ChatSessionRecord:
        actor = require_api_actor(request)
        try:
            return chat_service.create_session(
                actor_user_id=UUID(actor["user_id"]),
                payload=payload,
            )
        except ChatServiceError as exc:
            raise _to_http_exception(exc) from exc

    @router.get("/sessions", response_model=list[ChatSessionRecord])
    def list_sessions(
        request: Request,
        project_id: str | None = Query(default=None),
        limit: int = Query(default=50, ge=1, le=200),
        offset: int = Query(default=0, ge=0),
    ) -> list[ChatSessionRecord]:
        actor = require_api_actor(request)
        return chat_service.list_sessions(
            actor_user_id=UUID(actor["user_id"]),
            project_id=project_id,
            limit=limit,
            offset=offset,
        )

    @router.get("/sessions/{session_id}", response_model=ChatSessionRecord)
    def get_session(request: Request, session_id: UUID) -> ChatSessionRecord:
        actor = require_api_actor(request)
        try:
            return chat_service.get_session(
                actor_user_id=UUID(actor["user_id"]),
                session_id=session_id,
            )
        except ChatServiceError as exc:
            raise _to_http_exception(exc) from exc

    @router.patch("/sessions/{session_id}", response_model=ChatSessionRecord)
    def update_session(
        request: Request,
        session_id: UUID,
        payload: ChatSessionUpdateRequest,
    ) -> ChatSessionRecord:
        actor = require_api_actor(request)
        try:
            return chat_service.update_session(
                actor_user_id=UUID(actor["user_id"]),
                session_id=session_id,
                payload=payload,
            )
        except ChatServiceError as exc:
            raise _to_http_exception(exc) from exc

    @router.get("/sessions/{session_id}/messages", response_model=list[ChatMessageRecord])
    def list_messages(
        request: Request,
        session_id: UUID,
        limit: int = Query(default=200, ge=1, le=500),
        offset: int = Query(default=0, ge=0),
    ) -> list[ChatMessageRecord]:
        actor = require_api_actor(request)
        try:
            return chat_service.list_messages(
                actor_user_id=UUID(actor["user_id"]),
                session_id=session_id,
                limit=limit,
                offset=offset,
            )
        except ChatServiceError as exc:
            raise _to_http_exception(exc) from exc

    @router.get("/sessions/{session_id}/engrams", response_model=list[EngramSummary])
    def list_pinned_engrams(request: Request, session_id: UUID) -> list[EngramSummary]:
        actor = require_api_actor(request)
        try:
            return chat_service.list_pinned_engrams(
                actor_user_id=UUID(actor["user_id"]),
                session_id=session_id,
            )
        except ChatServiceError as exc:
            raise _to_http_exception(exc) from exc

    @router.post(
        "/sessions/{session_id}/messages", response_model=ChatSendResponse, status_code=201
    )
    def send_message(
        request: Request,
        session_id: UUID,
        payload: ChatMessageCreateRequest,
    ) -> ChatSendResponse:
        actor = require_api_actor(request)
        try:
            return chat_service.send_message(
                actor_user_id=UUID(actor["user_id"]),
                session_id=session_id,
                payload=payload,
            )
        except ChatServiceError as exc:
            raise _to_http_exception(exc) from exc

    @router.post("/sessions/{session_id}/messages/stream")
    def send_message_stream(
        request: Request,
        session_id: UUID,
        payload: ChatMessageCreateRequest,
    ) -> StreamingResponse:
        actor = require_api_actor(request)
        try:
            events = chat_service.stream_message_events(
                actor_user_id=UUID(actor["user_id"]),
                session_id=session_id,
                payload=payload,
            )
        except ChatServiceError as exc:
            raise _to_http_exception(exc) from exc

        def _stream():
            for event_name, event_payload in events:
                yield _sse_event(event_name, event_payload)

        return StreamingResponse(_stream(), media_type="text/event-stream")

    @router.post("/sessions/{session_id}/engrams/pin", response_model=PinnedEngramRecord)
    def pin_engram(
        request: Request,
        session_id: UUID,
        payload: PinEngramRequest,
    ) -> PinnedEngramRecord:
        actor = require_api_actor(request)
        try:
            return chat_service.pin_engram(
                actor_user_id=UUID(actor["user_id"]),
                session_id=session_id,
                payload=payload,
            )
        except ChatServiceError as exc:
            raise _to_http_exception(exc) from exc

    @router.delete("/sessions/{session_id}/engrams/{engram_id}")
    def unpin_engram(request: Request, session_id: UUID, engram_id: UUID) -> dict[str, bool]:
        actor = require_api_actor(request)
        try:
            chat_service.unpin_engram(
                actor_user_id=UUID(actor["user_id"]),
                session_id=session_id,
                engram_id=engram_id,
            )
        except ChatServiceError as exc:
            raise _to_http_exception(exc) from exc
        return {"removed": True}

    @router.post(
        "/sessions/{session_id}/save-engram",
        response_model=SaveSessionAsEngramResponse,
        status_code=201,
    )
    def save_as_engram(
        request: Request,
        session_id: UUID,
        payload: SaveSessionAsEngramRequest,
    ) -> SaveSessionAsEngramResponse:
        actor = require_api_actor(request)
        try:
            return chat_service.save_session_as_engram(
                actor_user_id=UUID(actor["user_id"]),
                session_id=session_id,
                payload=payload,
            )
        except ChatServiceError as exc:
            raise _to_http_exception(exc) from exc

    @router.post(
        "/sessions/{session_id}/continue", response_model=ContinueSessionResponse, status_code=201
    )
    def continue_session(
        request: Request,
        session_id: UUID,
        payload: ContinueSessionRequest,
    ) -> ContinueSessionResponse:
        actor = require_api_actor(request)
        try:
            return chat_service.continue_session(
                actor_user_id=UUID(actor["user_id"]),
                session_id=session_id,
                payload=payload,
            )
        except ChatServiceError as exc:
            raise _to_http_exception(exc) from exc

    return router
