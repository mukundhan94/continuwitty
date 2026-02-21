from __future__ import annotations

from dataclasses import dataclass
from typing import Any
from uuid import UUID

from fastapi import APIRouter, HTTPException, Query, Request

from .agent_models import AgentResumeRequest, AgentRunRequest, AgentRunResponse
from .models import (
    EngramCreateResponse,
    EngramQueryRequest,
    EngramQueryResult,
    EngramSourceRecord,
    EngramSummary,
    McpTokenCreateRequest,
    McpTokenCreateResponse,
    McpTokenRevokeRequest,
    McpTokenSummary,
    MemoryEngramCreate,
    RehydrationBundle,
    UserCreateRequest,
    UserRecord,
    UserRole,
    UserUpdateRequest,
)


@dataclass(frozen=True)
class MainApiRouterDependencies:
    require_roles_api: Any
    require_authenticated_api_user: Any
    get_settings: Any
    project_service: Any
    create_engram: Any
    list_engrams: Any
    query_engrams: Any
    get_rehydration_bundle: Any
    get_engram_sources: Any
    list_users: Any
    create_user: Any
    update_user: Any
    hash_password: Any
    create_token_for_owner: Any
    list_token_summaries: Any
    revoke_token_for_owner: Any
    mcp_token_pepper: str
    log_audit_event: Any
    agent_workflow: Any


def _require_admin_actor(
    request: Request,
    dependencies: MainApiRouterDependencies,
) -> dict[str, Any]:
    return dependencies.require_roles_api(request, {UserRole.admin.value})


def _build_agent_run_response(*, thread_id: str, state: dict[str, Any]) -> AgentRunResponse:
    return AgentRunResponse(
        thread_id=thread_id,
        status=state.get("status", "unknown"),
        engram_id=state.get("engram_id"),
        snapshot_engram_ids=state.get("snapshot_engram_ids", []),
        state=state,
    )


def _ensure_user_update_has_fields(payload: UserUpdateRequest) -> None:
    if payload.role is not None:
        return
    if payload.is_active is not None:
        return
    if payload.password is not None:
        return
    raise HTTPException(status_code=400, detail="No update fields provided")


def _register_user_profile_routes(
    router: APIRouter,
    dependencies: MainApiRouterDependencies,
) -> None:
    @router.get("/api/v1/me")
    def me_endpoint(request: Request) -> dict[str, Any]:
        return dependencies.require_roles_api(
            request,
            {UserRole.admin.value, UserRole.analyst.value, UserRole.viewer.value},
        )

    @router.get("/api/v1/users", response_model=list[UserRecord])
    def list_users_endpoint(
        request: Request,
        limit: int = Query(default=200, ge=1, le=500),
        offset: int = Query(default=0, ge=0),
    ) -> list[UserRecord]:
        _require_admin_actor(request, dependencies)
        return dependencies.list_users(limit=limit, offset=offset)


def _register_user_management_routes(
    router: APIRouter,
    dependencies: MainApiRouterDependencies,
) -> None:
    @router.post("/api/v1/users", response_model=UserRecord, status_code=201)
    def create_user_endpoint(request: Request, payload: UserCreateRequest) -> UserRecord:
        actor = _require_admin_actor(request, dependencies)
        try:
            created = dependencies.create_user(
                username=payload.username,
                password_hash=dependencies.hash_password(payload.password),
                role=payload.role,
                is_active=payload.is_active,
            )
            dependencies.log_audit_event(
                request=request,
                event_type="user_created",
                success=True,
                username=actor["username"],
                metadata={"target_username": payload.username, "role": payload.role.value},
            )
            return created
        except ValueError as exc:
            dependencies.log_audit_event(
                request=request,
                event_type="user_create_conflict",
                success=False,
                username=actor["username"],
                metadata={"target_username": payload.username},
                detail=str(exc),
            )
            raise HTTPException(status_code=409, detail=str(exc)) from exc

    @router.patch("/api/v1/users/{user_id}", response_model=UserRecord)
    def update_user_endpoint(
        request: Request,
        user_id: UUID,
        payload: UserUpdateRequest,
    ) -> UserRecord:
        actor = _require_admin_actor(request, dependencies)
        _ensure_user_update_has_fields(payload)
        updated = dependencies.update_user(
            user_id=user_id,
            role=payload.role,
            is_active=payload.is_active,
            password_hash=dependencies.hash_password(payload.password) if payload.password else None,
        )
        if not updated:
            dependencies.log_audit_event(
                request=request,
                event_type="user_update_missing",
                success=False,
                username=actor["username"],
                metadata={"target_user_id": str(user_id)},
            )
            raise HTTPException(status_code=404, detail="User not found")
        dependencies.log_audit_event(
            request=request,
            event_type="user_updated",
            success=True,
            username=actor["username"],
            metadata={
                "target_user_id": str(user_id),
                "role": payload.role.value if payload.role else None,
                "is_active": payload.is_active,
                "password_updated": payload.password is not None,
            },
        )
        return updated


def _register_user_routes(router: APIRouter, dependencies: MainApiRouterDependencies) -> None:
    _register_user_profile_routes(router, dependencies)
    _register_user_management_routes(router, dependencies)


def _register_mcp_token_routes(router: APIRouter, dependencies: MainApiRouterDependencies) -> None:
    @router.post("/api/v1/mcp/tokens", response_model=McpTokenCreateResponse, status_code=201)
    def create_mcp_token_endpoint(
        request: Request,
        payload: McpTokenCreateRequest,
    ) -> McpTokenCreateResponse:
        actor = _require_admin_actor(request, dependencies)
        created = dependencies.create_token_for_owner(
            owner_user_id=UUID(actor["user_id"]),
            payload=payload,
            pepper=dependencies.mcp_token_pepper,
        )
        dependencies.log_audit_event(
            request=request,
            event_type="mcp_token_created",
            success=True,
            username=actor["username"],
            metadata={
                "token_id": str(created.token_id),
                "scope": created.scope.value,
                "name": created.name,
                "allowed_tools": created.allowed_tools,
                "allowed_project_ids": created.allowed_project_ids,
                "expires_at": created.expires_at.isoformat(),
            },
        )
        return created

    @router.get("/api/v1/mcp/tokens", response_model=list[McpTokenSummary])
    def list_mcp_tokens_endpoint(
        request: Request,
        limit: int = Query(default=200, ge=1, le=500),
        offset: int = Query(default=0, ge=0),
    ) -> list[McpTokenSummary]:
        actor = _require_admin_actor(request, dependencies)
        return dependencies.list_token_summaries(
            owner_user_id=UUID(actor["user_id"]),
            limit=limit,
            offset=offset,
        )

    @router.post("/api/v1/mcp/tokens/{token_id}/revoke", response_model=McpTokenSummary)
    def revoke_mcp_token_endpoint(
        request: Request,
        token_id: UUID,
        payload: McpTokenRevokeRequest,
    ) -> McpTokenSummary:
        actor = _require_admin_actor(request, dependencies)
        revoked = dependencies.revoke_token_for_owner(
            token_id=token_id,
            owner_user_id=UUID(actor["user_id"]),
        )
        if not revoked:
            raise HTTPException(status_code=404, detail="Token not found")

        dependencies.log_audit_event(
            request=request,
            event_type="mcp_token_revoked",
            success=True,
            username=actor["username"],
            metadata={"token_id": str(token_id), "reason": payload.reason},
        )
        return revoked


def _register_agent_routes(router: APIRouter, dependencies: MainApiRouterDependencies) -> None:
    @router.post("/api/v1/agent-runs", response_model=AgentRunResponse)
    def run_agent_workflow(payload: AgentRunRequest) -> AgentRunResponse:
        result = dependencies.agent_workflow.run(payload.model_dump(mode="json"))
        return _build_agent_run_response(thread_id=payload.thread_id, state=result)

    @router.get("/api/v1/agent-runs/{thread_id}", response_model=AgentRunResponse)
    def get_agent_run_state(thread_id: str) -> AgentRunResponse:
        state = dependencies.agent_workflow.get_state(thread_id)
        if not state:
            raise HTTPException(status_code=404, detail="Thread state not found")
        return _build_agent_run_response(thread_id=thread_id, state=state)

    @router.post("/api/v1/agent-runs/{thread_id}/resume", response_model=AgentRunResponse)
    def resume_agent_workflow(
        thread_id: str,
        payload: AgentResumeRequest,
    ) -> AgentRunResponse:
        result = dependencies.agent_workflow.resume(
            thread_id,
            payload.model_dump(mode="json", exclude_none=True),
        )
        if not result:
            raise HTTPException(status_code=404, detail="Thread state not found")
        return _build_agent_run_response(thread_id=thread_id, state=result)


def _register_engram_write_routes(
    router: APIRouter,
    dependencies: MainApiRouterDependencies,
) -> None:
    @router.post("/api/v1/engrams", response_model=EngramCreateResponse)
    def create_engram_endpoint(
        request: Request,
        payload: MemoryEngramCreate,
    ) -> EngramCreateResponse:
        actor = dependencies.require_authenticated_api_user(request)
        actor_user_id = UUID(actor["user_id"])
        resolution = dependencies.project_service.resolve_project_id_for_write(
            actor_user_id=actor_user_id,
            actor_role=actor["role"],
            project_id=payload.project_id,
        )
        resolved_payload = payload.model_copy(update={"project_id": resolution.project_id})
        current_settings = dependencies.get_settings()
        created = dependencies.create_engram(
            resolved_payload,
            embedding_dim=current_settings.embedding_dim,
            owner_user_id=actor_user_id,
            enrichment_origin="api.engrams.create",
        )
        return created.model_copy(
            update={
                "resolved_project_id": resolution.project_id,
                "used_default_project": resolution.used_default_project,
            }
        )

    @router.get("/api/v1/engrams", response_model=list[EngramSummary])
    def list_engrams_endpoint(
        request: Request,
        project_id: str | None = Query(default=None),
        limit: int = Query(default=25, ge=1, le=200),
        offset: int = Query(default=0, ge=0),
    ) -> list[EngramSummary]:
        actor = dependencies.require_authenticated_api_user(request)
        return dependencies.list_engrams(
            project_id=project_id,
            limit=limit,
            offset=offset,
            actor_user_id=UUID(actor["user_id"]),
        )

    @router.post("/api/v1/engrams/query", response_model=list[EngramQueryResult])
    def query_engrams_endpoint(
        request: Request,
        payload: EngramQueryRequest,
    ) -> list[EngramQueryResult]:
        actor = dependencies.require_authenticated_api_user(request)
        current_settings = dependencies.get_settings()
        return dependencies.query_engrams(
            payload,
            embedding_dim=current_settings.embedding_dim,
            actor_user_id=UUID(actor["user_id"]),
        )


def _register_engram_read_routes(
    router: APIRouter,
    dependencies: MainApiRouterDependencies,
) -> None:
    @router.get("/api/v1/engrams/{engram_id}/sources", response_model=list[EngramSourceRecord])
    def list_engram_sources_endpoint(
        request: Request,
        engram_id: UUID,
        limit: int = Query(default=100, ge=1, le=500),
    ) -> list[EngramSourceRecord]:
        actor = dependencies.require_authenticated_api_user(request)
        actor_user_id = UUID(actor["user_id"])
        bundle = dependencies.get_rehydration_bundle(engram_id, actor_user_id=actor_user_id)
        if not bundle:
            raise HTTPException(status_code=404, detail="Engram not found")
        return dependencies.get_engram_sources(
            engram_id,
            limit=limit,
            actor_user_id=actor_user_id,
        )

    @router.get("/api/v1/engrams/{engram_id}/rehydrate", response_model=RehydrationBundle)
    def rehydrate_engram_endpoint(request: Request, engram_id: UUID) -> RehydrationBundle:
        actor = dependencies.require_authenticated_api_user(request)
        result = dependencies.get_rehydration_bundle(
            engram_id,
            actor_user_id=UUID(actor["user_id"]),
        )
        if not result:
            raise HTTPException(status_code=404, detail="Engram not found")
        return result


def _register_engram_routes(router: APIRouter, dependencies: MainApiRouterDependencies) -> None:
    _register_engram_write_routes(router, dependencies)
    _register_engram_read_routes(router, dependencies)


def create_main_api_router(*, dependencies: MainApiRouterDependencies) -> APIRouter:
    router = APIRouter()
    _register_user_routes(router, dependencies)
    _register_mcp_token_routes(router, dependencies)
    _register_agent_routes(router, dependencies)
    _register_engram_routes(router, dependencies)
    return router
