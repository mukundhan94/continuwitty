from contextlib import asynccontextmanager, suppress
from dataclasses import dataclass
from pathlib import Path
from typing import Any
from urllib.parse import unquote
from uuid import UUID

from fastapi import Depends, FastAPI, Form, HTTPException, Query, Request
from fastapi.responses import HTMLResponse, RedirectResponse, Response
from fastapi.templating import Jinja2Templates
from starlette.middleware.sessions import SessionMiddleware

from .agent_workflow import AgentWorkflowService
from .audit import log_audit_event
from .auth import generate_csrf_token, hash_password, verify_password
from .chat import ChatService, create_chat_router
from .config import build_debug_settings_snapshot, get_settings, should_log_settings
from .db import ensure_schema_initialized
from .export import ExportService, create_export_router
from .ingestion import DocumentIngestionService, create_ingestion_router
from .login_guard import LoginAttemptGuard
from .main_api_router import MainApiRouterDependencies, create_main_api_router
from .mcp import McpService, McpServiceDependencies, create_mcp_router
from .mcp.auth import McpResolvedActor, resolve_mcp_actor
from .mcp_tokens import (
    create_token_for_owner,
    list_token_summaries,
    normalize_string_list,
    revoke_token_for_owner,
)
from .memory_admin import MemoryAdminService, create_memory_admin_router
from .models import (
    AppVersionResponse,
    McpTokenCreateRequest,
    McpTokenScope,
    McpTokenSummary,
    UserRecord,
    UserRole,
)
from .oauth import create_oauth_router
from .projects import ProjectService, create_projects_router
from .repository import (
    create_engram,
    get_engram_sources,
    get_rehydration_bundle,
    list_engrams,
    query_engrams,
)
from .user_repository import (
    create_user,
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
    yield
    with suppress(Exception):
        agent_workflow.close()


settings = get_settings()
app = FastAPI(
    title="Engram Vault API",
    version=settings.app_semantic_version,
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
project_service = ProjectService()
memory_admin_service = MemoryAdminService(
    embedding_dim=settings.embedding_dim,
    project_service=project_service,
)
export_service = ExportService(
    project_service=project_service,
    memory_admin_service=memory_admin_service,
    embedding_dim=settings.embedding_dim,
)
ingestion_service = DocumentIngestionService(
    embedding_dim=settings.embedding_dim,
    max_file_bytes=settings.ingestion_max_file_bytes,
    max_text_chars=settings.ingestion_max_text_chars,
)
mcp_service = McpService(
    dependencies=McpServiceDependencies(
        chat_service=chat_service,
        project_service=project_service,
        memory_admin_service=memory_admin_service,
        embedding_dim=settings.embedding_dim,
        ingestion_service=ingestion_service,
    ),
    server_version=settings.app_semantic_version,
)
# FastAPI dependency object kept at module scope to satisfy lint rule B008.
MCP_TOKEN_SCOPE_DEFAULT = McpTokenScope.read


@dataclass(frozen=True)
class _McpTokenFormPayload:
    name: str
    scope: McpTokenScope
    allowed_tools: str
    allowed_project_ids: str
    expires_in_days: int
    csrf_token: str


async def _parse_mcp_token_form_payload(
    request: Request,
) -> _McpTokenFormPayload:
    form = await request.form()
    scope_raw = str(form.get("scope", MCP_TOKEN_SCOPE_DEFAULT.value))
    try:
        scope = McpTokenScope(scope_raw)
    except ValueError as exc:
        raise HTTPException(status_code=422, detail="Invalid scope") from exc
    try:
        expires_in_days = int(str(form.get("expires_in_days", "90")))
    except ValueError as exc:
        raise HTTPException(status_code=422, detail="expires_in_days must be an integer") from exc
    return _McpTokenFormPayload(
        name=str(form.get("name", "")),
        scope=scope,
        allowed_tools=str(form.get("allowed_tools", "")),
        allowed_project_ids=str(form.get("allowed_project_ids", "")),
        expires_in_days=expires_in_days,
        csrf_token=str(form.get("csrf_token", "")),
    )


# FastAPI dependency object kept at module scope to satisfy lint rule B008.
MCP_TOKEN_FORM_PAYLOAD_DEPENDENCY = Depends(_parse_mcp_token_form_payload)


@dataclass(frozen=True)
class _LoginFormPayload:
    username: str
    password: str
    csrf_token: str
    next_path: str


async def _parse_login_form_payload(request: Request) -> _LoginFormPayload:
    form = await request.form()
    return _LoginFormPayload(
        username=str(form.get("username", "")),
        password=str(form.get("password", "")),
        csrf_token=str(form.get("csrf_token", "")),
        next_path=str(form.get("next_path", "")),
    )


# FastAPI dependency object kept at module scope to satisfy lint rule B008.
LOGIN_FORM_PAYLOAD_DEPENDENCY = Depends(_parse_login_form_payload)


def _session_user(request: Request) -> dict[str, Any] | None:
    user = request.session.get("user")
    if not isinstance(user, dict):
        return None
    if not user.get("user_id"):
        return None
    if not user.get("username"):
        return None
    if not user.get("role"):
        return None
    return user


def _resolve_session_user(request: Request) -> dict[str, Any] | None:
    user = _session_user(request)
    if not user:
        return None

    try:
        db_user = get_user_auth_record(user["username"])
    except Exception:
        db_user = None

    if not db_user or not db_user.get("is_active"):
        request.session.pop("user", None)
        return None

    canonical = {
        "user_id": str(db_user["user_id"]),
        "username": db_user["username"],
        "role": db_user["role"],
    }
    if user != canonical:
        request.session["user"] = canonical
    return canonical


def _is_authenticated(request: Request) -> bool:
    return _resolve_session_user(request) is not None


def _login_redirect() -> RedirectResponse:
    return RedirectResponse(url="/login", status_code=303)


def _safe_next_path(raw_path: str | None) -> str | None:
    candidate = unquote((raw_path or "").strip())
    if not candidate:
        return None
    if not candidate.startswith("/"):
        return None
    if candidate.startswith("//"):
        return None
    return candidate


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

    if not user:
        return None
    if not user.get("is_active"):
        return None
    password_hash = user.get("password_hash")
    if not password_hash:
        return None
    if not verify_password(password, password_hash):
        return None
    return {
        "user_id": str(user["user_id"]),
        "username": user["username"],
        "role": user["role"],
    }


def _validate_login_preconditions(*, request: Request, username: str, csrf_token: str) -> str:
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
    return attempt_key


def _require_roles_api(request: Request, allowed_roles: set[str]) -> dict[str, Any]:
    user = _resolve_session_user(request)
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


def _require_admin_api_user(request: Request) -> dict[str, Any]:
    return _require_roles_api(request, {UserRole.admin.value})


def _resolve_mcp_actor(request: Request) -> McpResolvedActor:
    return resolve_mcp_actor(
        request=request,
        settings=settings,
        require_session_actor=_require_authenticated_api_user,
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
        resolve_mcp_actor=_resolve_mcp_actor,
    )
)
app.include_router(
    create_ingestion_router(
        ingestion_service=ingestion_service,
        require_api_actor=_require_authenticated_api_user,
    )
)
app.include_router(
    create_oauth_router(
        settings=settings,
        resolve_session_user=_resolve_session_user,
    )
)
app.include_router(
    create_projects_router(
        project_service=project_service,
        require_api_actor=_require_authenticated_api_user,
    )
)
app.include_router(
    create_memory_admin_router(
        memory_admin_service=memory_admin_service,
        require_admin_actor=_require_admin_api_user,
    )
)
app.include_router(
    create_export_router(
        export_service=export_service,
        require_api_actor=_require_authenticated_api_user,
    )
)
app.include_router(
    create_main_api_router(
        dependencies=MainApiRouterDependencies(
            require_roles_api=lambda request, allowed_roles: _require_roles_api(
                request, allowed_roles
            ),
            require_authenticated_api_user=lambda request: _require_authenticated_api_user(request),
            get_settings=lambda: get_settings(),
            project_service=project_service,
            create_engram=lambda payload, **kwargs: create_engram(payload, **kwargs),
            list_engrams=lambda **kwargs: list_engrams(**kwargs),
            query_engrams=lambda payload, **kwargs: query_engrams(payload, **kwargs),
            get_rehydration_bundle=lambda engram_id, **kwargs: get_rehydration_bundle(
                engram_id, **kwargs
            ),
            get_engram_sources=lambda engram_id, **kwargs: get_engram_sources(engram_id, **kwargs),
            list_users=lambda **kwargs: list_users(**kwargs),
            create_user=lambda **kwargs: create_user(**kwargs),
            update_user=lambda **kwargs: update_user(**kwargs),
            hash_password=lambda password: hash_password(password),
            create_token_for_owner=lambda **kwargs: create_token_for_owner(**kwargs),
            list_token_summaries=lambda **kwargs: list_token_summaries(**kwargs),
            revoke_token_for_owner=lambda **kwargs: revoke_token_for_owner(**kwargs),
            mcp_token_pepper=settings.mcp_token_pepper,
            log_audit_event=lambda **kwargs: log_audit_event(**kwargs),
            agent_workflow=agent_workflow,
        )
    )
)


@app.get("/", include_in_schema=False)
def home_redirect(request: Request) -> Response:
    if _is_authenticated(request):
        return RedirectResponse(url="/ui", status_code=303)
    return _login_redirect()


@app.get("/login", response_class=HTMLResponse, include_in_schema=False)
def login_page(request: Request, next: str | None = Query(default=None)) -> Response:  # noqa: A002
    next_path = _safe_next_path(next)
    if _is_authenticated(request):
        return RedirectResponse(url=next_path or "/ui", status_code=303)
    return templates.TemplateResponse(
        request,
        "login.html",
        {
            "request": request,
            "error": None,
            "csrf_token": _csrf_token_for_request(request),
            "next_path": next_path or "",
        },
    )


@app.post("/login", response_class=HTMLResponse, include_in_schema=False)
def login_submit(
    request: Request,
    payload: _LoginFormPayload = LOGIN_FORM_PAYLOAD_DEPENDENCY,
) -> Response:
    attempt_key = _validate_login_preconditions(
        request=request,
        username=payload.username,
        csrf_token=payload.csrf_token,
    )

    authenticated_user = _authenticate_user(payload.username, payload.password)
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
        return RedirectResponse(url=_safe_next_path(payload.next_path) or "/ui", status_code=303)

    login_attempt_guard.register_failure(attempt_key)
    log_audit_event(
        request=request,
        event_type="login_failed",
        success=False,
        username=payload.username,
    )
    return templates.TemplateResponse(
        request,
        "login.html",
        {
            "request": request,
            "error": "Invalid username or password.",
            "csrf_token": _csrf_token_for_request(request),
            "next_path": _safe_next_path(payload.next_path) or "",
        },
        status_code=401,
    )


@app.post("/logout", include_in_schema=False)
def logout(request: Request, csrf_token: str = Form(...)) -> Response:
    user = _resolve_session_user(request)
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
    user = _resolve_session_user(request)
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
    user = _resolve_session_user(request)
    if not user:
        return _login_redirect()
    if user["role"] != UserRole.admin.value:
        raise HTTPException(status_code=403, detail="Admin role required")

    users: list[UserRecord] = []
    error: str | None = None
    token_error = request.session.pop("admin_mcp_error", None)
    latest_mcp_token = request.session.pop("latest_mcp_token", None)
    mcp_tokens: list[McpTokenSummary] = []
    try:
        users = list_users(limit=500, offset=0)
    except Exception as exc:  # pragma: no cover - local display fallback
        error = str(exc)
    try:
        mcp_tokens = list_token_summaries(owner_user_id=UUID(user["user_id"]), limit=500, offset=0)
    except Exception as exc:  # pragma: no cover - local display fallback
        token_error = str(exc)

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
            "mcp_tokens": mcp_tokens,
            "latest_mcp_token": latest_mcp_token,
            "token_error": token_error,
        },
    )


@app.post("/ui/admin/mcp-tokens/create", include_in_schema=False)
def ui_admin_create_mcp_token(
    request: Request,
    form: _McpTokenFormPayload = MCP_TOKEN_FORM_PAYLOAD_DEPENDENCY,
) -> Response:
    user = _require_roles_api(request, {UserRole.admin.value})
    if not _verify_csrf_token(request, form.csrf_token):
        raise HTTPException(status_code=403, detail="Invalid CSRF token")

    try:
        payload = McpTokenCreateRequest(
            name=form.name,
            scope=form.scope,
            allowed_tools=normalize_string_list(form.allowed_tools.split(",")),
            allowed_project_ids=normalize_string_list(form.allowed_project_ids.split(",")),
            expires_in_days=form.expires_in_days,
        )
        created = create_token_for_owner(
            owner_user_id=UUID(user["user_id"]),
            payload=payload,
            pepper=settings.mcp_token_pepper,
        )
    except Exception as exc:
        request.session["admin_mcp_error"] = str(exc)
        return RedirectResponse(url="/ui/admin", status_code=303)

    request.session["latest_mcp_token"] = created.model_dump(mode="json")
    log_audit_event(
        request=request,
        event_type="mcp_token_created",
        success=True,
        username=user["username"],
        metadata={
            "token_id": str(created.token_id),
            "scope": created.scope.value,
            "name": created.name,
            "allowed_tools": created.allowed_tools,
            "allowed_project_ids": created.allowed_project_ids,
            "expires_at": created.expires_at.isoformat(),
        },
    )
    return RedirectResponse(url="/ui/admin", status_code=303)


@app.post("/ui/admin/mcp-tokens/{token_id}/revoke", include_in_schema=False)
def ui_admin_revoke_mcp_token(
    request: Request, token_id: UUID, csrf_token: str = Form(...)
) -> Response:
    user = _require_roles_api(request, {UserRole.admin.value})
    if not _verify_csrf_token(request, csrf_token):
        raise HTTPException(status_code=403, detail="Invalid CSRF token")

    revoked = revoke_token_for_owner(
        token_id=token_id,
        owner_user_id=UUID(user["user_id"]),
    )
    if not revoked:
        raise HTTPException(status_code=404, detail="Token not found")

    log_audit_event(
        request=request,
        event_type="mcp_token_revoked",
        success=True,
        username=user["username"],
        metadata={"token_id": str(token_id)},
    )
    return RedirectResponse(url="/ui/admin", status_code=303)


@app.get("/healthz")
def healthz() -> dict[str, str]:
    return {"status": "ok"}


@app.get("/api/v1/version", response_model=AppVersionResponse)
def app_version() -> AppVersionResponse:
    # This endpoint is intentionally public so external MCP clients and operators
    # can verify runtime build identity without requiring an authenticated session.
    semantic_version = (settings.app_semantic_version or app.version).strip() or "0.1.0"
    commit_id = (settings.app_commit_sha or "").strip() or "unknown"
    return AppVersionResponse(
        commit_id=commit_id,
        semantic_version=semantic_version,
        release=f"v{semantic_version}",
    )
