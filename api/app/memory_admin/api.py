from __future__ import annotations

from collections.abc import Callable
from typing import Any
from uuid import UUID

from fastapi import APIRouter, Query, Request

from app.models import (
    AdminChatSessionRecord,
    AdminEngramDeleteRequest,
    AdminEngramDeleteResponse,
    AdminEngramMoveRequest,
    AdminEngramRecord,
    AdminEngramRestoreResponse,
    AdminEngramUpdateRequest,
    AdminSessionDeleteRequest,
    AdminSessionDeleteResponse,
    AdminSessionRestoreResponse,
    EngramCollectionCreateRequest,
    EngramCollectionDeleteRequest,
    EngramCollectionItemsUpdateRequest,
    EngramCollectionRecord,
    EngramCollectionUpdateRequest,
)

from .service import MemoryAdminEngramListRequest, MemoryAdminListRequest, MemoryAdminService

OWNER_USER_ID_QUERY_DEFAULT = Query(default=None)
SESSION_ID_QUERY_DEFAULT = Query(default=None)


def _register_session_routes(
    *,
    router: APIRouter,
    memory_admin_service: MemoryAdminService,
    require_admin_actor: Callable[[Request], dict[str, Any]],
) -> None:
    @router.get("/sessions", response_model=list[AdminChatSessionRecord])
    def list_sessions(
        request: Request,
        project_id: str | None = Query(default=None),
        owner_user_id: UUID | None = OWNER_USER_ID_QUERY_DEFAULT,
        include_deleted: bool = Query(default=False),
        limit: int = Query(default=200, ge=1, le=1000),
        offset: int = Query(default=0, ge=0),
    ) -> list[AdminChatSessionRecord]:
        require_admin_actor(request)
        return memory_admin_service.list_sessions(
            request=MemoryAdminListRequest(
                project_id=project_id,
                owner_user_id=owner_user_id,
                include_deleted=include_deleted,
                limit=limit,
                offset=offset,
            )
        )

    @router.delete("/sessions/{session_id}", response_model=AdminSessionDeleteResponse)
    def delete_session(
        request: Request,
        session_id: UUID,
        payload: AdminSessionDeleteRequest,
    ) -> AdminSessionDeleteResponse:
        actor = require_admin_actor(request)
        return memory_admin_service.delete_session(
            session_id=session_id,
            actor_user_id=UUID(actor["user_id"]),
            payload=payload,
        )

    @router.post("/sessions/{session_id}/restore", response_model=AdminSessionRestoreResponse)
    def restore_session(request: Request, session_id: UUID) -> AdminSessionRestoreResponse:
        require_admin_actor(request)
        return memory_admin_service.restore_session(session_id=session_id)


def _register_engram_read_routes(
    *,
    router: APIRouter,
    memory_admin_service: MemoryAdminService,
    require_admin_actor: Callable[[Request], dict[str, Any]],
) -> None:
    @router.get("/engrams", response_model=list[AdminEngramRecord])
    def list_engrams(
        request: Request,
        project_id: str | None = Query(default=None),
        session_id: UUID | None = SESSION_ID_QUERY_DEFAULT,
        q: str | None = Query(default=None),
        include_deleted: bool = Query(default=False),
        limit: int = Query(default=200, ge=1, le=1000),
        offset: int = Query(default=0, ge=0),
    ) -> list[AdminEngramRecord]:
        require_admin_actor(request)
        return memory_admin_service.list_engrams(
            request=MemoryAdminEngramListRequest(
                project_id=project_id,
                owner_user_id=None,
                include_deleted=include_deleted,
                limit=limit,
                offset=offset,
                session_id=session_id,
                query_text=q,
            )
        )

    @router.get("/engrams/{engram_id}", response_model=AdminEngramRecord)
    def get_engram(
        request: Request,
        engram_id: UUID,
        include_deleted: bool = Query(default=True),
    ) -> AdminEngramRecord:
        require_admin_actor(request)
        return memory_admin_service.get_engram(
            engram_id=engram_id,
            include_deleted=include_deleted,
        )


def _register_engram_write_routes(
    *,
    router: APIRouter,
    memory_admin_service: MemoryAdminService,
    require_admin_actor: Callable[[Request], dict[str, Any]],
) -> None:

    @router.patch("/engrams/{engram_id}", response_model=AdminEngramRecord)
    def update_engram(
        request: Request,
        engram_id: UUID,
        payload: AdminEngramUpdateRequest,
    ) -> AdminEngramRecord:
        actor = require_admin_actor(request)
        return memory_admin_service.update_engram(
            engram_id=engram_id,
            actor_user_id=UUID(actor["user_id"]),
            payload=payload,
        )

    @router.post("/engrams/{engram_id}/move", response_model=AdminEngramRecord)
    def move_engram(
        request: Request,
        engram_id: UUID,
        payload: AdminEngramMoveRequest,
    ) -> AdminEngramRecord:
        actor = require_admin_actor(request)
        return memory_admin_service.move_engram(
            engram_id=engram_id,
            actor_user_id=UUID(actor["user_id"]),
            actor_role=actor["role"],
            payload=payload,
        )

    @router.delete("/engrams/{engram_id}", response_model=AdminEngramDeleteResponse)
    def delete_engram(
        request: Request,
        engram_id: UUID,
        payload: AdminEngramDeleteRequest,
    ) -> AdminEngramDeleteResponse:
        actor = require_admin_actor(request)
        return memory_admin_service.delete_engram(
            engram_id=engram_id,
            actor_user_id=UUID(actor["user_id"]),
            payload=payload,
        )

    @router.post("/engrams/{engram_id}/restore", response_model=AdminEngramRestoreResponse)
    def restore_engram(request: Request, engram_id: UUID) -> AdminEngramRestoreResponse:
        require_admin_actor(request)
        return memory_admin_service.restore_engram(engram_id=engram_id)


def _register_engram_routes(
    *,
    router: APIRouter,
    memory_admin_service: MemoryAdminService,
    require_admin_actor: Callable[[Request], dict[str, Any]],
) -> None:
    _register_engram_read_routes(
        router=router,
        memory_admin_service=memory_admin_service,
        require_admin_actor=require_admin_actor,
    )
    _register_engram_write_routes(
        router=router,
        memory_admin_service=memory_admin_service,
        require_admin_actor=require_admin_actor,
    )


def _register_collection_read_routes(
    *,
    router: APIRouter,
    memory_admin_service: MemoryAdminService,
    require_admin_actor: Callable[[Request], dict[str, Any]],
) -> None:
    @router.get("/collections", response_model=list[EngramCollectionRecord])
    def list_collections(
        request: Request,
        project_id: str | None = Query(default=None),
        include_deleted: bool = Query(default=False),
        limit: int = Query(default=200, ge=1, le=1000),
        offset: int = Query(default=0, ge=0),
    ) -> list[EngramCollectionRecord]:
        require_admin_actor(request)
        return memory_admin_service.list_collections(
            request=MemoryAdminListRequest(
                project_id=project_id,
                owner_user_id=None,
                include_deleted=include_deleted,
                limit=limit,
                offset=offset,
            )
        )


def _register_collection_write_routes(
    *,
    router: APIRouter,
    memory_admin_service: MemoryAdminService,
    require_admin_actor: Callable[[Request], dict[str, Any]],
) -> None:

    @router.post("/collections", response_model=EngramCollectionRecord, status_code=201)
    def create_collection(
        request: Request,
        payload: EngramCollectionCreateRequest,
    ) -> EngramCollectionRecord:
        actor = require_admin_actor(request)
        return memory_admin_service.create_collection(
            actor_user_id=UUID(actor["user_id"]),
            actor_role=actor["role"],
            payload=payload,
        )

    @router.patch("/collections/{collection_id}", response_model=EngramCollectionRecord)
    def update_collection(
        request: Request,
        collection_id: UUID,
        payload: EngramCollectionUpdateRequest,
    ) -> EngramCollectionRecord:
        require_admin_actor(request)
        return memory_admin_service.update_collection(collection_id=collection_id, payload=payload)

    @router.delete("/collections/{collection_id}")
    def delete_collection(
        request: Request,
        collection_id: UUID,
        payload: EngramCollectionDeleteRequest,
    ) -> dict[str, bool]:
        actor = require_admin_actor(request)
        return memory_admin_service.delete_collection(
            collection_id=collection_id,
            actor_user_id=UUID(actor["user_id"]),
            payload=payload,
        )

    @router.post("/collections/{collection_id}/items")
    def add_items(
        request: Request,
        collection_id: UUID,
        payload: EngramCollectionItemsUpdateRequest,
    ) -> dict[str, int]:
        actor = require_admin_actor(request)
        return memory_admin_service.add_collection_items(
            collection_id=collection_id,
            actor_user_id=UUID(actor["user_id"]),
            payload=payload,
        )

    @router.delete("/collections/{collection_id}/items/{engram_id}")
    def remove_item(request: Request, collection_id: UUID, engram_id: UUID) -> dict[str, bool]:
        require_admin_actor(request)
        return memory_admin_service.remove_collection_item(
            collection_id=collection_id,
            engram_id=engram_id,
        )


def _register_collection_routes(
    *,
    router: APIRouter,
    memory_admin_service: MemoryAdminService,
    require_admin_actor: Callable[[Request], dict[str, Any]],
) -> None:
    _register_collection_read_routes(
        router=router,
        memory_admin_service=memory_admin_service,
        require_admin_actor=require_admin_actor,
    )
    _register_collection_write_routes(
        router=router,
        memory_admin_service=memory_admin_service,
        require_admin_actor=require_admin_actor,
    )


def create_memory_admin_router(
    *,
    memory_admin_service: MemoryAdminService,
    require_admin_actor: Callable[[Request], dict[str, Any]],
) -> APIRouter:
    router = APIRouter(prefix="/api/v1/admin/memory", tags=["admin-memory"])
    _register_session_routes(
        router=router,
        memory_admin_service=memory_admin_service,
        require_admin_actor=require_admin_actor,
    )
    _register_engram_routes(
        router=router,
        memory_admin_service=memory_admin_service,
        require_admin_actor=require_admin_actor,
    )
    _register_collection_routes(
        router=router,
        memory_admin_service=memory_admin_service,
        require_admin_actor=require_admin_actor,
    )
    return router
