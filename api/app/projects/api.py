from __future__ import annotations

from collections.abc import Callable
from typing import Any
from uuid import UUID

from fastapi import APIRouter, Query, Request

from app.models import (
    ProjectCreateRequest,
    ProjectDefaultResponse,
    ProjectDefaultUpdateRequest,
    ProjectRecord,
)

from .service import ProjectService


def create_projects_router(
    *,
    project_service: ProjectService,
    require_api_actor: Callable[[Request], dict[str, Any]],
) -> APIRouter:
    router = APIRouter(prefix="/api/v1/projects", tags=["projects"])

    @router.get("", response_model=list[ProjectRecord])
    def list_projects(
        request: Request,
        include_archived: bool = Query(default=False),
        limit: int = Query(default=500, ge=1, le=1000),
        offset: int = Query(default=0, ge=0),
    ) -> list[ProjectRecord]:
        actor = require_api_actor(request)
        return project_service.list_projects(
            actor_user_id=UUID(actor["user_id"]),
            actor_role=actor["role"],
            include_archived=include_archived,
            limit=limit,
            offset=offset,
        )

    @router.post("", response_model=ProjectRecord, status_code=201)
    def create_project(request: Request, payload: ProjectCreateRequest) -> ProjectRecord:
        actor = require_api_actor(request)
        return project_service.create_project(
            actor_user_id=UUID(actor["user_id"]),
            actor_role=actor["role"],
            payload=payload,
        )

    @router.get("/default", response_model=ProjectDefaultResponse)
    def get_default_project(request: Request) -> ProjectDefaultResponse:
        actor = require_api_actor(request)
        return ProjectDefaultResponse(
            default_project_id=project_service.get_default_project_id(
                actor_user_id=UUID(actor["user_id"])
            )
        )

    @router.patch("/default", response_model=ProjectDefaultResponse)
    def set_default_project(
        request: Request, payload: ProjectDefaultUpdateRequest
    ) -> ProjectDefaultResponse:
        actor = require_api_actor(request)
        default_project_id = project_service.set_default_project_id(
            actor_user_id=UUID(actor["user_id"]),
            actor_role=actor["role"],
            project_id=payload.project_id,
        )
        return ProjectDefaultResponse(default_project_id=default_project_id)

    return router
