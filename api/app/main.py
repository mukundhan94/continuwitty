from pathlib import Path
from uuid import UUID

from fastapi import FastAPI, Form, HTTPException, Query, Request
from fastapi.responses import HTMLResponse, RedirectResponse, Response
from fastapi.templating import Jinja2Templates
from starlette.middleware.sessions import SessionMiddleware

from .agent_models import AgentResumeRequest, AgentRunRequest, AgentRunResponse
from .agent_workflow import AgentWorkflowService
from .auth import generate_csrf_token, verify_password
from .config import get_settings
from .models import (
    EngramCreateResponse,
    EngramQueryRequest,
    EngramQueryResult,
    EngramSourceRecord,
    EngramSummary,
    MemoryEngramCreate,
    RehydrationBundle,
)
from .repository import (
    create_engram,
    get_engram_sources,
    get_rehydration_bundle,
    list_engrams,
    query_engrams,
)

settings = get_settings()
app = FastAPI(
    title="Engram Vault API",
    version="0.1.0",
    description="Local-first memory engram store with semantic query and rehydration.",
)
app.add_middleware(
    SessionMiddleware,
    secret_key=settings.app_session_secret,
    same_site="lax",
    https_only=False,
)
templates = Jinja2Templates(directory=str(Path(__file__).parent / "templates"))
agent_workflow = AgentWorkflowService()


def _is_authenticated(request: Request) -> bool:
    return bool(request.session.get("user"))


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
        raise HTTPException(status_code=403, detail="Invalid CSRF token")

    current_settings = get_settings()
    password_ok = False
    if current_settings.ui_demo_password_hash:
        password_ok = verify_password(password, current_settings.ui_demo_password_hash)
    else:
        password_ok = password == current_settings.ui_demo_password

    if username == current_settings.ui_demo_username and password_ok:
        request.session["user"] = username
        request.session["csrf_token"] = generate_csrf_token()
        return RedirectResponse(url="/ui", status_code=303)
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
    if not _verify_csrf_token(request, csrf_token):
        raise HTTPException(status_code=403, detail="Invalid CSRF token")
    request.session.clear()
    return _login_redirect()


@app.get("/ui", response_class=HTMLResponse, include_in_schema=False)
def ui_dashboard(request: Request) -> Response:
    if not _is_authenticated(request):
        return _login_redirect()
    return templates.TemplateResponse(
        request,
        "dashboard.html",
        {
            "request": request,
            "username": request.session.get("user", "unknown"),
            "csrf_token": _csrf_token_for_request(request),
        },
    )


@app.get("/healthz")
def healthz() -> dict[str, str]:
    return {"status": "ok"}


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
    settings = get_settings()
    return create_engram(payload, embedding_dim=settings.embedding_dim)


@app.get("/api/v1/engrams", response_model=list[EngramSummary])
def list_engrams_endpoint(
    project_id: str | None = Query(default=None),
    limit: int = Query(default=25, ge=1, le=200),
    offset: int = Query(default=0, ge=0),
) -> list[EngramSummary]:
    return list_engrams(project_id=project_id, limit=limit, offset=offset)


@app.post("/api/v1/engrams/query", response_model=list[EngramQueryResult])
def query_engrams_endpoint(payload: EngramQueryRequest) -> list[EngramQueryResult]:
    settings = get_settings()
    return query_engrams(payload, embedding_dim=settings.embedding_dim)


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
