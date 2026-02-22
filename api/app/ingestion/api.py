from __future__ import annotations

import json
from collections.abc import Callable
from dataclasses import dataclass
from typing import Any
from uuid import UUID

from fastapi import APIRouter, Depends, HTTPException, Query, Request, UploadFile
from starlette.datastructures import UploadFile as StarletteUploadFile

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
from app.ingestion.service import DocumentIngestionService, FileIngestRequest
from app.models import VisibilityScope


@dataclass(frozen=True)
class _IngestFileFormPayload:
    project_id: str
    title: str | None
    visibility_scope: VisibilityScope
    chunk_size_chars: int
    chunk_overlap_chars: int
    metadata_json: str | None
    file: UploadFile | None


def _parse_int_form_field(value: str | None, *, default: int, field_name: str) -> int:
    if value is None or value.strip() == "":
        return default
    try:
        return int(value.strip())
    except ValueError as exc:
        raise HTTPException(status_code=422, detail=f"{field_name} must be an integer") from exc


async def _parse_ingest_file_form_payload(request: Request) -> _IngestFileFormPayload:
    form = await request.form()
    project_id = str(form.get("project_id", "")).strip()
    if not project_id:
        raise HTTPException(status_code=422, detail="project_id is required")

    visibility_raw = str(form.get("visibility_scope", VisibilityScope.private.value)).strip()
    try:
        visibility_scope = VisibilityScope(visibility_raw)
    except ValueError as exc:
        raise HTTPException(status_code=422, detail="visibility_scope is invalid") from exc

    upload = form.get("file")
    file = upload if isinstance(upload, UploadFile | StarletteUploadFile) else None
    return _IngestFileFormPayload(
        project_id=project_id,
        title=str(form.get("title", "")).strip() or None,
        visibility_scope=visibility_scope,
        chunk_size_chars=_parse_int_form_field(
            str(form.get("chunk_size_chars", "")).strip() or None,
            default=1000,
            field_name="chunk_size_chars",
        ),
        chunk_overlap_chars=_parse_int_form_field(
            str(form.get("chunk_overlap_chars", "")).strip() or None,
            default=180,
            field_name="chunk_overlap_chars",
        ),
        metadata_json=str(form.get("metadata_json", "")).strip() or None,
        file=file,
    )


# FastAPI dependency object kept at module scope to satisfy lint rule B008.
INGEST_FILE_FORM_PAYLOAD_DEPENDENCY = Depends(_parse_ingest_file_form_payload)


def _to_http_exception(exc: IngestionServiceError) -> HTTPException:
    return HTTPException(status_code=exc.status_code, detail=exc.detail)


def _parse_metadata_json(metadata_json: str | None, *, max_bytes: int) -> dict[str, Any]:
    if not metadata_json:
        return {}
    if len(metadata_json.encode("utf-8")) > max_bytes:
        raise HTTPException(
            status_code=413,
            detail=f"metadata_json exceeds max allowed size of {max_bytes} bytes",
        )

    try:
        parsed = json.loads(metadata_json)
    except json.JSONDecodeError as exc:
        raise HTTPException(status_code=422, detail="metadata_json must be valid JSON") from exc
    if not isinstance(parsed, dict):
        raise HTTPException(status_code=422, detail="metadata_json must be a JSON object")
    return parsed


def _build_file_ingest_request(*, upload: UploadFile, content_bytes: bytes) -> FileIngestRequest:
    return FileIngestRequest(
        filename=upload.filename or "uploaded.txt",
        mime_type=upload.content_type,
        content_bytes=content_bytes,
    )


def create_ingestion_router(
    *,
    ingestion_service: DocumentIngestionService,
    require_api_actor: Callable[[Request], dict[str, Any]],
    max_metadata_json_bytes: int = 20_000,
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
        form: _IngestFileFormPayload = INGEST_FILE_FORM_PAYLOAD_DEPENDENCY,
    ) -> DocumentIngestResponse:
        actor = require_api_actor(request)
        if form.file is None:
            raise HTTPException(status_code=422, detail="file is required")

        metadata = _parse_metadata_json(form.metadata_json, max_bytes=max_metadata_json_bytes)
        payload = DocumentIngestFileRequest(
            project_id=form.project_id,
            title=form.title,
            visibility_scope=form.visibility_scope,
            chunk_size_chars=form.chunk_size_chars,
            chunk_overlap_chars=form.chunk_overlap_chars,
            metadata=metadata,
        )
        content = await form.file.read()

        try:
            return ingestion_service.ingest_file(
                actor_user_id=UUID(actor["user_id"]),
                payload=payload,
                file_request=_build_file_ingest_request(upload=form.file, content_bytes=content),
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
