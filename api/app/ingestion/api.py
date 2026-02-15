from __future__ import annotations

from collections.abc import Callable
from typing import Any
from uuid import UUID

from fastapi import APIRouter, Form, HTTPException, Query, Request, UploadFile

from app.ingestion.errors import IngestionServiceError
from app.ingestion.models import (
    BlendedRetrievalQueryRequest,
    BlendedRetrievalQueryResponse,
    DocumentChunkQueryRequest,
    DocumentChunkQueryResult,
    DocumentIngestFileRequest,
    DocumentIngestResponse,
    DocumentIngestTextRequest,
    DocumentRecord,
)
from app.ingestion.service import DocumentIngestionService
from app.models import VisibilityScope

_FORM_DEFAULT_VISIBILITY = Form(default=VisibilityScope.private)
_FORM_DEFAULT_CHUNK_SIZE = Form(default=1000)
_FORM_DEFAULT_CHUNK_OVERLAP = Form(default=180)


def _to_http_exception(exc: IngestionServiceError) -> HTTPException:
    return HTTPException(status_code=exc.status_code, detail=exc.detail)


def create_ingestion_router(
    *,
    ingestion_service: DocumentIngestionService,
    require_api_actor: Callable[[Request], dict[str, Any]],
) -> APIRouter:
    router = APIRouter(prefix="/api/v1/ingestion", tags=["ingestion"])

    @router.post("/text", response_model=DocumentIngestResponse, status_code=201)
    def ingest_text(request: Request, payload: DocumentIngestTextRequest) -> DocumentIngestResponse:
        actor = require_api_actor(request)
        try:
            return ingestion_service.ingest_text(
                actor_user_id=UUID(actor["user_id"]),
                payload=payload,
            )
        except IngestionServiceError as exc:
            raise _to_http_exception(exc) from exc

    @router.post("/file", response_model=DocumentIngestResponse, status_code=201)
    async def ingest_file(
        request: Request,
        project_id: str = Form(...),
        title: str | None = Form(default=None),
        visibility_scope: VisibilityScope = _FORM_DEFAULT_VISIBILITY,
        chunk_size_chars: int = _FORM_DEFAULT_CHUNK_SIZE,
        chunk_overlap_chars: int = _FORM_DEFAULT_CHUNK_OVERLAP,
        metadata_json: str | None = Form(default=None),
        file: UploadFile | None = None,
    ) -> DocumentIngestResponse:
        actor = require_api_actor(request)
        if file is None:
            raise HTTPException(status_code=422, detail="file is required")

        metadata: dict[str, Any] = {}
        if metadata_json:
            import json

            try:
                parsed = json.loads(metadata_json)
            except json.JSONDecodeError as exc:
                raise HTTPException(
                    status_code=422, detail="metadata_json must be valid JSON"
                ) from exc
            if not isinstance(parsed, dict):
                raise HTTPException(status_code=422, detail="metadata_json must be a JSON object")
            metadata = parsed

        payload = DocumentIngestFileRequest(
            project_id=project_id,
            title=title,
            visibility_scope=visibility_scope,
            chunk_size_chars=chunk_size_chars,
            chunk_overlap_chars=chunk_overlap_chars,
            metadata=metadata,
        )
        content = await file.read()

        try:
            return ingestion_service.ingest_file(
                actor_user_id=UUID(actor["user_id"]),
                payload=payload,
                filename=file.filename or "uploaded.txt",
                mime_type=file.content_type,
                content_bytes=content,
            )
        except IngestionServiceError as exc:
            raise _to_http_exception(exc) from exc

    @router.get("/documents", response_model=list[DocumentRecord])
    def list_project_documents(
        request: Request,
        project_id: str | None = Query(default=None),
        limit: int = Query(default=100, ge=1, le=500),
        offset: int = Query(default=0, ge=0),
    ) -> list[DocumentRecord]:
        actor = require_api_actor(request)
        return ingestion_service.list_documents(
            actor_user_id=UUID(actor["user_id"]),
            project_id=project_id,
            limit=limit,
            offset=offset,
        )

    @router.post("/query", response_model=list[DocumentChunkQueryResult])
    def query_documents(
        request: Request,
        payload: DocumentChunkQueryRequest,
    ) -> list[DocumentChunkQueryResult]:
        actor = require_api_actor(request)
        return ingestion_service.query_document_chunks(
            actor_user_id=UUID(actor["user_id"]),
            payload=payload,
        )

    @router.post("/query/blended", response_model=BlendedRetrievalQueryResponse)
    def query_blended(
        request: Request,
        payload: BlendedRetrievalQueryRequest,
    ) -> BlendedRetrievalQueryResponse:
        actor = require_api_actor(request)
        return ingestion_service.query_blended(
            actor_user_id=UUID(actor["user_id"]),
            payload=payload,
        )

    return router
