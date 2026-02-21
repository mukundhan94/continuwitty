from __future__ import annotations

from typing import Any, Protocol
from uuid import UUID

from app.models import ProjectCreateRequest

from .errors import McpRpcError


class _UserDispatchContext(Protocol):
    actor: dict[str, Any]
    actor_user_id: UUID
    method: str


class _ProjectDispatchContext(_UserDispatchContext, Protocol):
    params: dict[str, Any]


def projects_for_user(
    *,
    actor_user_id: UUID,
    chat_service: Any,
) -> list[str]:
    sessions = chat_service.list_sessions(
        actor_user_id=actor_user_id,
        project_id=None,
        limit=1000,
        offset=0,
    )
    project_ids = {session.project_id for session in sessions if getattr(session, "project_id", None)}
    return sorted(project_ids)


def dispatch_project_tool(
    *,
    project_service: Any,
    context: _ProjectDispatchContext,
) -> dict[str, Any] | None:
    actor_role = str(context.actor.get("role", ""))

    if context.method == "project.list":
        projects = project_service.list_projects(
            actor_user_id=context.actor_user_id,
            actor_role=actor_role,
            include_archived=bool(context.params.get("include_archived", False)),
            limit=int(context.params.get("limit", 500)),
            offset=int(context.params.get("offset", 0)),
        )
        return {"projects": [item.model_dump(mode="json") for item in projects]}

    if context.method == "project.create":
        try:
            payload = ProjectCreateRequest(
                project_id=str(context.params.get("project_id", "")),
                name=str(context.params.get("name", "")),
                description=str(context.params.get("description", "")),
                owner_user_id=(
                    UUID(str(context.params["owner_user_id"]))
                    if context.params.get("owner_user_id") is not None
                    else None
                ),
            )
        except ValueError as exc:
            raise McpRpcError(
                code=-32602,
                message="Invalid params",
                data={"invalid": "owner_user_id"},
            ) from exc
        created = project_service.create_project(
            actor_user_id=context.actor_user_id,
            actor_role=actor_role,
            payload=payload,
        )
        return {"project": created.model_dump(mode="json")}

    if context.method == "project.get_default":
        return {
            "default_project_id": project_service.get_default_project_id(
                actor_user_id=context.actor_user_id
            )
        }

    if context.method == "project.set_default":
        project_id = project_service.set_default_project_id(
            actor_user_id=context.actor_user_id,
            actor_role=actor_role,
            project_id=str(context.params.get("project_id", "")),
        )
        return {"default_project_id": project_id}

    return None


def dispatch_user_tool(
    *,
    context: _UserDispatchContext,
    chat_service: Any,
) -> dict[str, Any] | None:
    if context.method == "user.get_profile":
        return {"profile": context.actor}
    if context.method == "user.list_projects":
        return {
            "project_ids": projects_for_user(
                actor_user_id=context.actor_user_id,
                chat_service=chat_service,
            )
        }
    return None
