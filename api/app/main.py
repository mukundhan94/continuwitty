from contextlib import asynccontextmanager, suppress
from pathlib import Path
from typing import Any
from uuid import UUID

from fastapi import FastAPI, Form, HTTPException, Query, Request
from fastapi.responses import HTMLResponse, RedirectResponse, Response
from fastapi.templating import Jinja2Templates
from starlette.middleware.sessions import SessionMiddleware

from .agent_models import AgentResumeRequest, AgentRunRequest, AgentRunResponse
from .agent_workflow import AgentWorkflowService
from .audit import log_audit_event
from .auth import generate_csrf_token, hash_password, verify_password
from .chat import ChatService, create_chat_router
from .config import build_debug_settings_snapshot, get_settings, should_log_settings
from .db import ensure_schema_initialized
from .login_guard import LoginAttemptGuard
from .mcp import McpService, create_mcp_router
from .models import (
    EngramCreateResponse,
    EngramQueryRequest,
    EngramQueryResult,
    EngramSourceRecord,
    EngramSummary,
    MemoryEngramCreate,
    RehydrationBundle,
    UserCreateRequest,
    UserRecord,
    UserRole,
    UserUpdateRequest,
)
from .repository import (
    create_engram,
    get_engram_sources,
    get_rehydration_bundle,
    list_engrams,
    query_engrams,
)
from .user_repository import (
    create_user,
    ensure_user_store,
    get_user_auth_record,
    list_users,
    update_user,
)


@asynccontextmanager
async def lifespan(_app: FastAPI):
    current_settings = get_settings()
    if should_log_settings(current_settings):
        print("[engram-api] parsed config", build_debug_settings_snapshot(current_settings))
    try:
        ensure_schema_initialized()
    except Exception as exc:  # pragma: no cover - defensive for local startup mismatches
        print(f"[warn] failed to initialize schema: {exc}")
    try:
        ensure_user_store()
    except Exception as exc:  # pragma: no cover - defensive for local startup mismatches
        print(f"[warn] failed to initialize user store: {exc}")
    yield
    with suppress(Exception):
        agent_workflow.close()


settings = get_settings()
app = FastAPI(
    title="Engram Vault API",
    version="0.1.0",
    description="Local-first memory engram store with semantic query and rehydration.",
    lifespan=lifespan,
)
app.add_middleware(
    SessionMiddleware,
    secret_key=settings.app_session_secret,
    same_site="lax",
    https_only=False,
)
templates = Jinja2Templates(directory=str(Path(__file__).parent / "templates"))
agent_workflow = AgentWorkflowService()
login_attempt_guard = LoginAttemptGuard(
    max_attempts=settings.login_rate_limit_max_attempts,
    window_seconds=settings.login_rate_limit_window_seconds,
    lockout_seconds=settings.login_lockout_seconds,
)
chat_service = ChatService(embedding_dim=settings.embedding_dim)
mcp_service = McpService(chat_service=chat_service, embedding_dim=settings.embedding_dim)


def _session_user(request: Request) -> dict[str, Any] | None:
    user = request.session.get("user")
    if isinstance(user, dict) and user.get("username") and user.get("role"):
        return user
    return None


def _is_authenticated(request: Request) -> bool:
    return _session_user(request) is not None


def _login_redirect() -> RedirectResponse:
    return RedirectResponse(url="/login", status_code=303)


def _csrf_token_for_request(request: Request) -> str:
    token = request.session.get("csrf_token")
    if not token:
        token = generate_csrf_token()
        request.session["csrf_token"] = token
    return token


def _verify_csrf_token(request: Request, submitted_token: str) -> bool:
    return bool(submitted_token) and submitted_token == request.session.get("csrf_token")


def _client_ip(request: Request) -> str:
    if request.client and request.client.host:
        return request.client.host
    return "unknown"


def _login_attempt_key(request: Request, username: str) -> str:
    return f"{username.lower()}:{_client_ip(request)}"


def _authenticate_user(username: str, password: str) -> dict[str, Any] | None:
    try:
        user = get_user_auth_record(username)
    except Exception:
        user = None

    if user and user.get("is_active") and verify_password(password, user["password_hash"]):
        return {
            "user_id": str(user["user_id"]),
            "username": user["username"],
            "role": user["role"],
        }

    current_settings = get_settings()
    if username != current_settings.ui_demo_username:
        return None

    if current_settings.ui_demo_password_hash:
        password_ok = verify_password(password, current_settings.ui_demo_password_hash)
    else:
        password_ok = password == current_settings.ui_demo_password

    if not password_ok:
        return None

    return {
        "user_id": "env-fallback-user",
        "username": current_settings.ui_demo_username,
        "role": UserRole.admin.value,
    }


def _require_roles_api(request: Request, allowed_roles: set[str]) -> dict[str, Any]:
    user = _session_user(request)
    if not user:
        raise HTTPException(status_code=401, detail="Authentication required")
    if user["role"] not in allowed_roles:
        raise HTTPException(status_code=403, detail="Insufficient role")
    return user


def _require_authenticated_api_user(request: Request) -> dict[str, Any]:
    return _require_roles_api(
        request,
        {UserRole.admin.value, UserRole.analyst.value, UserRole.viewer.value},
    )


app.include_router(
    create_chat_router(
        chat_service=chat_service,
        require_api_actor=_require_authenticated_api_user,
    )
)
app.include_router(
    create_mcp_router(
        mcp_service=mcp_service,
        require_api_actor=_require_authenticated_api_user,
    )
)


@app.get("/", include_in_schema=False)
def home_redirect(request: Request) -> Response:
    if _is_authenticated(request):
        return RedirectResponse(url="/ui", status_code=303)
    return _login_redirect()


@app.get("/login", response_class=HTMLResponse, include_in_schema=False)
def login_page(request: Request) -> Response:
    if _is_authenticated(request):
        return RedirectResponse(url="/ui", status_code=303)
    return templates.TemplateResponse(
        request,
        "login.html",
        {"request": request, "error": None, "csrf_token": _csrf_token_for_request(request)},
    )


@app.post("/login", response_class=HTMLResponse, include_in_schema=False)
def login_submit(
    request: Request,
    username: str = Form(...),
    password: str = Form(...),
    csrf_token: str = Form(...),
) -> Response:
    if not _verify_csrf_token(request, csrf_token):
        log_audit_event(
            request=request,
            event_type="login_csrf_rejected",
            success=False,
            username=username,
        )
        raise HTTPException(status_code=403, detail="Invalid CSRF token")

    attempt_key = _login_attempt_key(request, username)
    allowed, retry_seconds = login_attempt_guard.check(attempt_key)
    if not allowed:
        log_audit_event(
            request=request,
            event_type="login_rate_limited",
            success=False,
            username=username,
            detail=f"retry_in_seconds={retry_seconds}",
        )
        raise HTTPException(
            status_code=429,
            detail=f"Too many login attempts. Retry in {retry_seconds} seconds.",
        )

    authenticated_user = _authenticate_user(username, password)
    if authenticated_user:
        login_attempt_guard.register_success(attempt_key)
        request.session["user"] = authenticated_user
        request.session["csrf_token"] = generate_csrf_token()
        log_audit_event(
            request=request,
            event_type="login_success",
            success=True,
            username=authenticated_user["username"],
        )
        return RedirectResponse(url="/ui", status_code=303)

    login_attempt_guard.register_failure(attempt_key)
    log_audit_event(
        request=request,
        event_type="login_failed",
        success=False,
        username=username,
    )
    return templates.TemplateResponse(
        request,
        "login.html",
        {
            "request": request,
            "error": "Invalid username or password.",
            "csrf_token": _csrf_token_for_request(request),
        },
        status_code=401,
    )


@app.post("/logout", include_in_schema=False)
def logout(request: Request, csrf_token: str = Form(...)) -> Response:
    user = _session_user(request)
    if not _verify_csrf_token(request, csrf_token):
        log_audit_event(
            request=request,
            event_type="logout_csrf_rejected",
            success=False,
            username=user["username"] if user else None,
        )
        raise HTTPException(status_code=403, detail="Invalid CSRF token")
    log_audit_event(
        request=request,
        event_type="logout_success",
        success=True,
        username=user["username"] if user else None,
    )
    request.session.clear()
    return _login_redirect()


@app.get("/ui", response_class=HTMLResponse, include_in_schema=False)
def ui_dashboard(request: Request) -> Response:
    user = _session_user(request)
    if not user:
        return _login_redirect()
    return templates.TemplateResponse(
        request,
        "dashboard.html",
        {
            "request": request,
            "username": user["username"],
            "role": user["role"],
            "csrf_token": _csrf_token_for_request(request),
        },
    )


@app.get("/ui/admin", response_class=HTMLResponse, include_in_schema=False)
def ui_admin(request: Request) -> Response:
    user = _session_user(request)
    if not user:
        return _login_redirect()
    if user["role"] != UserRole.admin.value:
        raise HTTPException(status_code=403, detail="Admin role required")

    users: list[UserRecord] = []
    error: str | None = None
    try:
        users = list_users(limit=500, offset=0)
    except Exception as exc:  # pragma: no cover - local display fallback
        error = str(exc)

    return templates.TemplateResponse(
        request,
        "admin.html",
        {
            "request": request,
            "username": user["username"],
            "role": user["role"],
            "csrf_token": _csrf_token_for_request(request),
            "users": users,
            "error": error,
        },
    )


@app.get("/healthz")
def healthz() -> dict[str, str]:
    return {"status": "ok"}


@app.get("/api/v1/me")
def me_endpoint(request: Request) -> dict[str, Any]:
    return _require_roles_api(
        request, {UserRole.admin.value, UserRole.analyst.value, UserRole.viewer.value}
    )


@app.get("/api/v1/users", response_model=list[UserRecord])
def list_users_endpoint(
    request: Request,
    limit: int = Query(default=200, ge=1, le=500),
    offset: int = Query(default=0, ge=0),
) -> list[UserRecord]:
    _require_roles_api(request, {UserRole.admin.value})
    return list_users(limit=limit, offset=offset)


@app.post("/api/v1/users", response_model=UserRecord, status_code=201)
def create_user_endpoint(request: Request, payload: UserCreateRequest) -> UserRecord:
    actor = _require_roles_api(request, {UserRole.admin.value})
    try:
        created = create_user(
            username=payload.username,
            password_hash=hash_password(payload.password),
            role=payload.role,
            is_active=payload.is_active,
        )
        log_audit_event(
            request=request,
            event_type="user_created",
            success=True,
            username=actor["username"],
            metadata={"target_username": payload.username, "role": payload.role.value},
        )
        return created
    except ValueError as exc:
        log_audit_event(
            request=request,
            event_type="user_create_conflict",
            success=False,
            username=actor["username"],
            metadata={"target_username": payload.username},
            detail=str(exc),
        )
        raise HTTPException(status_code=409, detail=str(exc)) from exc


@app.patch("/api/v1/users/{user_id}", response_model=UserRecord)
def update_user_endpoint(request: Request, user_id: UUID, payload: UserUpdateRequest) -> UserRecord:
    actor = _require_roles_api(request, {UserRole.admin.value})

    if payload.role is None and payload.is_active is None and payload.password is None:
        raise HTTPException(status_code=400, detail="No update fields provided")

    updated = update_user(
        user_id=user_id,
        role=payload.role,
        is_active=payload.is_active,
        password_hash=hash_password(payload.password) if payload.password else None,
    )
    if not updated:
        log_audit_event(
            request=request,
            event_type="user_update_missing",
            success=False,
            username=actor["username"],
            metadata={"target_user_id": str(user_id)},
        )
        raise HTTPException(status_code=404, detail="User not found")
    log_audit_event(
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


@app.post("/api/v1/agent-runs", response_model=AgentRunResponse)
def run_agent_workflow(payload: AgentRunRequest) -> AgentRunResponse:
    result = agent_workflow.run(payload.model_dump(mode="json"))
    return AgentRunResponse(
        thread_id=payload.thread_id,
        status=result.get("status", "unknown"),
        engram_id=result.get("engram_id"),
        snapshot_engram_ids=result.get("snapshot_engram_ids", []),
        state=result,
    )


@app.get("/api/v1/agent-runs/{thread_id}", response_model=AgentRunResponse)
def get_agent_run_state(thread_id: str) -> AgentRunResponse:
    state = agent_workflow.get_state(thread_id)
    if not state:
        raise HTTPException(status_code=404, detail="Thread state not found")
    return AgentRunResponse(
        thread_id=thread_id,
        status=state.get("status", "unknown"),
        engram_id=state.get("engram_id"),
        snapshot_engram_ids=state.get("snapshot_engram_ids", []),
        state=state,
    )


@app.post("/api/v1/agent-runs/{thread_id}/resume", response_model=AgentRunResponse)
def resume_agent_workflow(thread_id: str, payload: AgentResumeRequest) -> AgentRunResponse:
    result = agent_workflow.resume(thread_id, payload.model_dump(mode="json", exclude_none=True))
    if not result:
        raise HTTPException(status_code=404, detail="Thread state not found")
    return AgentRunResponse(
        thread_id=thread_id,
        status=result.get("status", "unknown"),
        engram_id=result.get("engram_id"),
        snapshot_engram_ids=result.get("snapshot_engram_ids", []),
        state=result,
    )


@app.post("/api/v1/engrams", response_model=EngramCreateResponse)
def create_engram_endpoint(payload: MemoryEngramCreate) -> EngramCreateResponse:
    current_settings = get_settings()
    return create_engram(payload, embedding_dim=current_settings.embedding_dim)


@app.get("/api/v1/engrams", response_model=list[EngramSummary])
def list_engrams_endpoint(
    project_id: str | None = Query(default=None),
    limit: int = Query(default=25, ge=1, le=200),
    offset: int = Query(default=0, ge=0),
) -> list[EngramSummary]:
    return list_engrams(project_id=project_id, limit=limit, offset=offset)


@app.post("/api/v1/engrams/query", response_model=list[EngramQueryResult])
def query_engrams_endpoint(payload: EngramQueryRequest) -> list[EngramQueryResult]:
    current_settings = get_settings()
    return query_engrams(payload, embedding_dim=current_settings.embedding_dim)


@app.get("/api/v1/engrams/{engram_id}/sources", response_model=list[EngramSourceRecord])
def list_engram_sources_endpoint(
    engram_id: UUID,
    limit: int = Query(default=100, ge=1, le=500),
) -> list[EngramSourceRecord]:
    bundle = get_rehydration_bundle(engram_id)
    if not bundle:
        raise HTTPException(status_code=404, detail="Engram not found")
    return get_engram_sources(engram_id, limit=limit)


@app.get("/api/v1/engrams/{engram_id}/rehydrate", response_model=RehydrationBundle)
def rehydrate_engram_endpoint(engram_id: UUID) -> RehydrationBundle:
    result = get_rehydration_bundle(engram_id)
    if not result:
        raise HTTPException(status_code=404, detail="Engram not found")
    return result
