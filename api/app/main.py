from uuid import UUID

from fastapi import FastAPI, HTTPException, Query

from .config import get_settings
from .models import (
    EngramCreateResponse,
    EngramQueryRequest,
    EngramQueryResult,
    EngramSummary,
    MemoryEngramCreate,
    RehydrationBundle,
)
from .repository import create_engram, get_rehydration_bundle, list_engrams, query_engrams

app = FastAPI(
    title="Engram Vault API",
    version="0.1.0",
    description="Local-first memory engram store with semantic query and rehydration.",
)


@app.get("/healthz")
def healthz() -> dict[str, str]:
    return {"status": "ok"}


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


@app.get("/api/v1/engrams/{engram_id}/rehydrate", response_model=RehydrationBundle)
def rehydrate_engram_endpoint(engram_id: UUID) -> RehydrationBundle:
    result = get_rehydration_bundle(engram_id)
    if not result:
        raise HTTPException(status_code=404, detail="Engram not found")
    return result
