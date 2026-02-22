from __future__ import annotations

from uuid import UUID

from fastapi import HTTPException

from app.models import ProjectCreateRequest, ProjectRecord

from .models import ProjectResolution
from .repository import (
    create_project,
    ensure_project_exists,
    get_project_for_actor,
    get_user_default_project_id,
    list_projects_for_actor,
    set_user_default_project_id,
)


def _normalize_project_id(value: str | None) -> str:
    return (value or "").strip()


class ProjectService:
    """Project registry and default-project resolution helpers.

    Design note:
    - We keep project fallback logic centralized here so REST and MCP write paths
      share exactly the same behavior when `project_id` is omitted.
    """

    def list_projects(
        self,
        *,
        actor_user_id: UUID,
        actor_role: str,
        include_archived: bool = False,
        limit: int = 500,
        offset: int = 0,
    ) -> list[ProjectRecord]:
        return list_projects_for_actor(
            actor_user_id=actor_user_id,
            actor_role=actor_role,
            include_archived=include_archived,
            limit=limit,
            offset=offset,
        )

    def get_project(
        self,
        *,
        actor_user_id: UUID,
        actor_role: str,
        project_id: str,
        include_archived: bool = False,
    ) -> ProjectRecord | None:
        normalized = _normalize_project_id(project_id)
        if not normalized:
            return None
        return get_project_for_actor(
            project_id=normalized,
            actor_user_id=actor_user_id,
            actor_role=actor_role,
            include_archived=include_archived,
        )

    def create_project(
        self,
        *,
        actor_user_id: UUID,
        actor_role: str,
        payload: ProjectCreateRequest,
    ) -> ProjectRecord:
        owner_user_id = (
            payload.owner_user_id
            if actor_role == "admin" and payload.owner_user_id
            else actor_user_id
        )
        project_id = _normalize_project_id(payload.project_id)
        if not project_id:
            raise HTTPException(status_code=422, detail="project_id must not be blank")
        return create_project(
            project_id=project_id,
            name=payload.name.strip(),
            description=payload.description.strip(),
            owner_user_id=owner_user_id,
        )

    def get_default_project_id(self, *, actor_user_id: UUID) -> str | None:
        return get_user_default_project_id(user_id=actor_user_id)

    def set_default_project_id(
        self,
        *,
        actor_user_id: UUID,
        actor_role: str,
        project_id: str,
    ) -> str:
        normalized = _normalize_project_id(project_id)
        if not normalized:
            raise HTTPException(status_code=422, detail="project_id must not be blank")
        # Caller must be allowed to see the project they set as default.
        visible = get_project_for_actor(
            project_id=normalized,
            actor_user_id=actor_user_id,
            actor_role=actor_role,
            include_archived=False,
        )
        if not visible:
            raise HTTPException(status_code=404, detail="Project not found")
        updated = set_user_default_project_id(user_id=actor_user_id, project_id=normalized)
        if not updated:
            raise HTTPException(status_code=404, detail="User not found")
        return normalized

    def resolve_project_id_for_write(
        self,
        *,
        actor_user_id: UUID,
        actor_role: str,
        project_id: str | None,
    ) -> ProjectResolution:
        explicit = _normalize_project_id(project_id)
        if explicit:
            # Ensure the explicit project exists so downstream writes can rely on
            # first-class project metadata. Non-admin callers may only access
            # projects they own under the current ownership model.
            visible = get_project_for_actor(
                project_id=explicit,
                actor_user_id=actor_user_id,
                actor_role=actor_role,
                include_archived=False,
            )
            if not visible:
                ensure_project_exists(project_id=explicit, owner_user_id=actor_user_id)
            return ProjectResolution(project_id=explicit, used_default_project=False)

        default_project_id = get_user_default_project_id(user_id=actor_user_id)
        if not default_project_id:
            raise HTTPException(
                status_code=422,
                detail="project_id is required when no default project is configured",
            )
        visible_default = get_project_for_actor(
            project_id=default_project_id,
            actor_user_id=actor_user_id,
            actor_role=actor_role,
            include_archived=False,
        )
        if not visible_default:
            raise HTTPException(
                status_code=422,
                detail="default project is not accessible; set a valid default project first",
            )
        return ProjectResolution(project_id=default_project_id, used_default_project=True)
