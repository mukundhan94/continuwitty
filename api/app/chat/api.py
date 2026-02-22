from __future__ import annotations

import json
from collections.abc import Callable, Iterable
from dataclasses import dataclass
from typing import Any, TypeVar
from uuid import UUID

from fastapi import APIRouter, HTTPException, Query, Request
from fastapi.encoders import jsonable_encoder
from fastapi.responses import StreamingResponse

from app.models import (
    ChatLifecyclePolicy,
    ChatLifecyclePolicyUpdateRequest,
    ChatMessageCreateRequest,
    ChatMessageRecord,
    ChatSendResponse,
    ChatSessionCreateRequest,
    ChatSessionRecord,
    ChatSessionUpdateRequest,
    ChatTimelineEvent,
    ContinueSessionRequest,
    ContinueSessionResponse,
    EngramSummary,
    PinDocumentRequest,
    PinEngramRequest,
    PinnedDocumentRecord,
    PinnedEngramRecord,
    SaveSessionAsEngramRequest,
    SaveSessionAsEngramResponse,
)

from .errors import ChatServiceError
from .service import ChatService

T = TypeVar("T")
ActorResolver = Callable[[Request], dict[str, Any]]


@dataclass(frozen=True)
class _SessionRouteContext:
    request: Request
    session_id: UUID


@dataclass(frozen=True)
class _PinnedResourceConfig:
    list_path: str
    list_route_name: str
    list_response_model: Any
    pin_path: str
    pin_route_name: str
    pin_response_model: Any
    pin_payload_model: type[PinEngramRequest] | type[PinDocumentRequest]
    unpin_path: str
    unpin_route_name: str
    resource_id_field: str
    list_operation_name: str
    pin_operation_name: str
    unpin_operation_name: str


def _to_http_exception(exc: ChatServiceError) -> HTTPException:
    return HTTPException(status_code=exc.status_code, detail=exc.detail)


def _sse_event(event: str, payload: dict[str, Any]) -> str:
    data = json.dumps(jsonable_encoder(payload))
    return f"event: {event}\ndata: {data}\n\n"


def _actor_user_id(request: Request, require_api_actor: ActorResolver) -> UUID:
    actor = require_api_actor(request)
    return UUID(actor["user_id"])


def _handle_chat_service_error(operation: Callable[[], T]) -> T:
    try:
        return operation()
    except ChatServiceError as exc:
        raise _to_http_exception(exc) from exc


def _stream_sse_events(events: Iterable[tuple[str, dict[str, Any]]]) -> Iterable[str]:
    for event_name, event_payload in events:
        yield _sse_event(event_name, event_payload)


def _run_session_operation(
    *,
    context: _SessionRouteContext,
    require_api_actor: ActorResolver,
    operation: Callable[..., T],
    extra_kwargs: dict[str, Any] | None = None,
) -> T:
    actor_user_id = _actor_user_id(context.request, require_api_actor)
    operation_kwargs: dict[str, Any] = {
        "actor_user_id": actor_user_id,
        "session_id": context.session_id,
    }
    if extra_kwargs:
        operation_kwargs.update(extra_kwargs)
    return _handle_chat_service_error(lambda: operation(**operation_kwargs))


def _pinned_resource_configs() -> tuple[_PinnedResourceConfig, _PinnedResourceConfig]:
    return (
        _PinnedResourceConfig(
            list_path="/sessions/{session_id}/engrams",
            list_route_name="list_pinned_engrams",
            list_response_model=list[EngramSummary],
            pin_path="/sessions/{session_id}/engrams/pin",
            pin_route_name="pin_engram",
            pin_response_model=PinnedEngramRecord,
            pin_payload_model=PinEngramRequest,
            unpin_path="/sessions/{session_id}/engrams/{resource_id}",
            unpin_route_name="unpin_engram",
            resource_id_field="engram_id",
            list_operation_name="list_pinned_engrams",
            pin_operation_name="pin_engram",
            unpin_operation_name="unpin_engram",
        ),
        _PinnedResourceConfig(
            list_path="/sessions/{session_id}/documents",
            list_route_name="list_pinned_documents",
            list_response_model=list[PinnedDocumentRecord],
            pin_path="/sessions/{session_id}/documents/pin",
            pin_route_name="pin_document",
            pin_response_model=PinnedDocumentRecord,
            pin_payload_model=PinDocumentRequest,
            unpin_path="/sessions/{session_id}/documents/{resource_id}",
            unpin_route_name="unpin_document",
            resource_id_field="document_id",
            list_operation_name="list_pinned_documents",
            pin_operation_name="pin_document",
            unpin_operation_name="unpin_document",
        ),
    )


def _register_session_routes(
    *,
    router: APIRouter,
    chat_service: ChatService,
    require_api_actor: ActorResolver,
) -> None:
    @router.post("/sessions", response_model=ChatSessionRecord, status_code=201)
    def create_session(request: Request, payload: ChatSessionCreateRequest) -> ChatSessionRecord:
        actor_user_id = _actor_user_id(request, require_api_actor)
        return _handle_chat_service_error(
            lambda: chat_service.create_session(
                actor_user_id=actor_user_id,
                payload=payload,
            )
        )

    @router.get("/sessions", response_model=list[ChatSessionRecord])
    def list_sessions(
        request: Request,
        project_id: str | None = Query(default=None),
        limit: int = Query(default=50, ge=1, le=200),
        offset: int = Query(default=0, ge=0),
    ) -> list[ChatSessionRecord]:
        actor_user_id = _actor_user_id(request, require_api_actor)
        return chat_service.list_sessions(
            actor_user_id=actor_user_id,
            project_id=project_id,
            limit=limit,
            offset=offset,
        )

    @router.get("/sessions/{session_id}", response_model=ChatSessionRecord)
    def get_session(request: Request, session_id: UUID) -> ChatSessionRecord:
        actor_user_id = _actor_user_id(request, require_api_actor)
        return _handle_chat_service_error(
            lambda: chat_service.get_session(
                actor_user_id=actor_user_id,
                session_id=session_id,
            )
        )

    @router.patch("/sessions/{session_id}", response_model=ChatSessionRecord)
    def update_session(
        request: Request,
        session_id: UUID,
        payload: ChatSessionUpdateRequest,
    ) -> ChatSessionRecord:
        actor_user_id = _actor_user_id(request, require_api_actor)
        return _handle_chat_service_error(
            lambda: chat_service.update_session(
                actor_user_id=actor_user_id,
                session_id=session_id,
                payload=payload,
            )
        )


def _register_lifecycle_routes(
    *,
    router: APIRouter,
    chat_service: ChatService,
    require_api_actor: ActorResolver,
) -> None:
    @router.get("/sessions/{session_id}/lifecycle-policy", response_model=ChatLifecyclePolicy)
    def get_lifecycle_policy(request: Request, session_id: UUID) -> ChatLifecyclePolicy:
        actor_user_id = _actor_user_id(request, require_api_actor)
        return _handle_chat_service_error(
            lambda: chat_service.get_lifecycle_policy(
                actor_user_id=actor_user_id,
                session_id=session_id,
            )
        )

    @router.patch("/sessions/{session_id}/lifecycle-policy", response_model=ChatLifecyclePolicy)
    def update_lifecycle_policy(
        request: Request,
        session_id: UUID,
        payload: ChatLifecyclePolicyUpdateRequest,
    ) -> ChatLifecyclePolicy:
        actor_user_id = _actor_user_id(request, require_api_actor)
        return _handle_chat_service_error(
            lambda: chat_service.update_lifecycle_policy(
                actor_user_id=actor_user_id,
                session_id=session_id,
                payload=payload,
            )
        )

    @router.get("/sessions/{session_id}/timeline", response_model=list[ChatTimelineEvent])
    def list_timeline_events(
        request: Request,
        session_id: UUID,
        limit: int = Query(default=100, ge=1, le=500),
        offset: int = Query(default=0, ge=0),
    ) -> list[ChatTimelineEvent]:
        actor_user_id = _actor_user_id(request, require_api_actor)
        return _handle_chat_service_error(
            lambda: chat_service.list_timeline_events(
                actor_user_id=actor_user_id,
                session_id=session_id,
                limit=limit,
                offset=offset,
            )
        )


def _register_message_routes(
    *,
    router: APIRouter,
    chat_service: ChatService,
    require_api_actor: ActorResolver,
) -> None:
    @router.get("/sessions/{session_id}/messages", response_model=list[ChatMessageRecord])
    def list_messages(
        request: Request,
        session_id: UUID,
        limit: int = Query(default=200, ge=1, le=500),
        offset: int = Query(default=0, ge=0),
    ) -> list[ChatMessageRecord]:
        actor_user_id = _actor_user_id(request, require_api_actor)
        return _handle_chat_service_error(
            lambda: chat_service.list_messages(
                actor_user_id=actor_user_id,
                session_id=session_id,
                limit=limit,
                offset=offset,
            )
        )

    @router.post("/sessions/{session_id}/messages", response_model=ChatSendResponse, status_code=201)
    def send_message(
        request: Request,
        session_id: UUID,
        payload: ChatMessageCreateRequest,
    ) -> ChatSendResponse:
        actor_user_id = _actor_user_id(request, require_api_actor)
        return _handle_chat_service_error(
            lambda: chat_service.send_message(
                actor_user_id=actor_user_id,
                session_id=session_id,
                payload=payload,
            )
        )

    @router.post("/sessions/{session_id}/messages/stream")
    def send_message_stream(
        request: Request,
        session_id: UUID,
        payload: ChatMessageCreateRequest,
    ) -> StreamingResponse:
        actor_user_id = _actor_user_id(request, require_api_actor)
        events = _handle_chat_service_error(
            lambda: chat_service.stream_message_events(
                actor_user_id=actor_user_id,
                session_id=session_id,
                payload=payload,
            )
        )
        return StreamingResponse(_stream_sse_events(events), media_type="text/event-stream")


def _register_pinned_list_endpoint(
    *,
    router: APIRouter,
    config: _PinnedResourceConfig,
    require_api_actor: ActorResolver,
    operation: Callable[..., Any],
) -> None:
    def list_pinned_resource(request: Request, session_id: UUID) -> Any:
        return _run_session_operation(
            context=_SessionRouteContext(request=request, session_id=session_id),
            require_api_actor=require_api_actor,
            operation=operation,
        )

    router.add_api_route(
        config.list_path,
        list_pinned_resource,
        methods=["GET"],
        response_model=config.list_response_model,
        name=config.list_route_name,
    )


def _register_pinned_pin_endpoint(
    *,
    router: APIRouter,
    config: _PinnedResourceConfig,
    require_api_actor: ActorResolver,
    operation: Callable[..., Any],
) -> None:
    def pin_resource(
        request: Request,
        session_id: UUID,
        payload: PinEngramRequest | PinDocumentRequest,
    ) -> Any:
        payload_data = payload.model_dump()
        if config.resource_id_field not in payload_data:
            raise HTTPException(status_code=422, detail=f"Missing {config.resource_id_field}")

        parsed_payload = config.pin_payload_model.model_validate(payload_data)
        return _run_session_operation(
            context=_SessionRouteContext(request=request, session_id=session_id),
            require_api_actor=require_api_actor,
            operation=operation,
            extra_kwargs={"payload": parsed_payload},
        )

    router.add_api_route(
        config.pin_path,
        pin_resource,
        methods=["POST"],
        response_model=config.pin_response_model,
        name=config.pin_route_name,
    )


def _register_pinned_unpin_endpoint(
    *,
    router: APIRouter,
    config: _PinnedResourceConfig,
    require_api_actor: ActorResolver,
    operation: Callable[..., Any],
) -> None:
    def unpin_resource(
        request: Request,
        session_id: UUID,
        resource_id: UUID,
    ) -> dict[str, bool]:
        _run_session_operation(
            context=_SessionRouteContext(request=request, session_id=session_id),
            require_api_actor=require_api_actor,
            operation=operation,
            extra_kwargs={config.resource_id_field: resource_id},
        )
        return {"removed": True}

    router.add_api_route(
        config.unpin_path,
        unpin_resource,
        methods=["DELETE"],
        name=config.unpin_route_name,
    )


def _register_single_pinned_resource_routes(
    *,
    router: APIRouter,
    chat_service: ChatService,
    config: _PinnedResourceConfig,
    require_api_actor: ActorResolver,
) -> None:
    _register_pinned_list_endpoint(
        router=router,
        config=config,
        require_api_actor=require_api_actor,
        operation=getattr(chat_service, config.list_operation_name),
    )
    _register_pinned_pin_endpoint(
        router=router,
        config=config,
        require_api_actor=require_api_actor,
        operation=getattr(chat_service, config.pin_operation_name),
    )
    _register_pinned_unpin_endpoint(
        router=router,
        config=config,
        require_api_actor=require_api_actor,
        operation=getattr(chat_service, config.unpin_operation_name),
    )


def _register_pinned_routes(
    *,
    router: APIRouter,
    chat_service: ChatService,
    require_api_actor: ActorResolver,
) -> None:
    for config in _pinned_resource_configs():
        _register_single_pinned_resource_routes(
            router=router,
            chat_service=chat_service,
            config=config,
            require_api_actor=require_api_actor,
        )


def _register_session_derivative_routes(
    *,
    router: APIRouter,
    chat_service: ChatService,
    require_api_actor: ActorResolver,
) -> None:
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
        actor_user_id = _actor_user_id(request, require_api_actor)
        return _handle_chat_service_error(
            lambda: chat_service.save_session_as_engram(
                actor_user_id=actor_user_id,
                session_id=session_id,
                payload=payload,
            )
        )

    @router.post("/sessions/{session_id}/continue", response_model=ContinueSessionResponse, status_code=201)
    def continue_session(
        request: Request,
        session_id: UUID,
        payload: ContinueSessionRequest,
    ) -> ContinueSessionResponse:
        actor_user_id = _actor_user_id(request, require_api_actor)
        return _handle_chat_service_error(
            lambda: chat_service.continue_session(
                actor_user_id=actor_user_id,
                session_id=session_id,
                payload=payload,
            )
        )


def create_chat_router(
    *,
    chat_service: ChatService,
    require_api_actor: ActorResolver,
) -> APIRouter:
    router = APIRouter(prefix="/api/v1/chat", tags=["chat"])

    _register_session_routes(
        router=router,
        chat_service=chat_service,
        require_api_actor=require_api_actor,
    )
    _register_lifecycle_routes(
        router=router,
        chat_service=chat_service,
        require_api_actor=require_api_actor,
    )
    _register_message_routes(
        router=router,
        chat_service=chat_service,
        require_api_actor=require_api_actor,
    )
    _register_pinned_routes(
        router=router,
        chat_service=chat_service,
        require_api_actor=require_api_actor,
    )
    _register_session_derivative_routes(
        router=router,
        chat_service=chat_service,
        require_api_actor=require_api_actor,
    )

    return router
