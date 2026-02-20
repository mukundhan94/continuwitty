from __future__ import annotations

from dataclasses import dataclass
from pathlib import Path
from uuid import UUID

from app.ingestion.chunking import (
    ChunkDraft,
    build_content_hash,
    build_document_id,
    chunk_document_text,
    normalize_document_text,
)
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
    DocumentSourceType,
)
from app.ingestion.repository import (
    DocumentUpsertPayload,
    list_documents,
    query_document_chunks,
    upsert_document_with_chunks,
)
from app.models import EngramQueryRequest
from app.projects.repository import ensure_project_exists
from app.repository import query_engrams


@dataclass(frozen=True)
class FileIngestRequest:
    filename: str
    mime_type: str | None
    content_bytes: bytes


@dataclass(frozen=True)
class ChunkBuildRequest:
    actor_user_id: UUID
    project_id: str
    title: str
    normalized_text: str
    chunk_size_chars: int
    chunk_overlap_chars: int
    empty_chunks_detail: str


class DocumentIngestionService:
    """Coordinates document ingestion, chunking, and blended retrieval responses."""

    def __init__(
        self,
        *,
        embedding_dim: int,
        max_file_bytes: int,
        max_text_chars: int,
    ) -> None:
        self._embedding_dim = embedding_dim
        self._max_file_bytes = max_file_bytes
        self._max_text_chars = max_text_chars

    @staticmethod
    def _validate_chunk_shape(chunk_size_chars: int, chunk_overlap_chars: int) -> None:
        if chunk_overlap_chars >= chunk_size_chars:
            raise IngestionServiceError(
                "chunk_overlap_chars must be smaller than chunk_size_chars",
                status_code=422,
            )

    def _validate_text_size(self, text: str) -> None:
        if len(text) > self._max_text_chars:
            raise IngestionServiceError(
                f"Document text exceeds max allowed size of {self._max_text_chars} characters",
                status_code=413,
            )

    def _validate_file_size(self, *, content_bytes: bytes) -> None:
        if not content_bytes:
            raise IngestionServiceError("Uploaded file is empty")
        if len(content_bytes) > self._max_file_bytes:
            raise IngestionServiceError(
                f"Uploaded file exceeds max allowed size of {self._max_file_bytes} bytes",
                status_code=413,
            )

    @staticmethod
    def _decode_file_text(*, content_bytes: bytes) -> str:
        try:
            return content_bytes.decode("utf-8")
        except UnicodeDecodeError as exc:
            raise IngestionServiceError(
                "Only UTF-8 text files are supported in this phase",
                status_code=415,
            ) from exc

    def _validate_file_input(self, *, request: FileIngestRequest) -> str:
        self._validate_file_size(content_bytes=request.content_bytes)
        decoded = self._decode_file_text(content_bytes=request.content_bytes)
        normalized_text = normalize_document_text(decoded)
        if not normalized_text:
            raise IngestionServiceError("Uploaded file does not contain readable text")
        self._validate_text_size(normalized_text)
        return normalized_text

    @staticmethod
    def _resolve_file_title(*, requested_title: str | None, filename: str) -> str:
        return (requested_title or Path(filename).stem or "Untitled Document").strip()

    @staticmethod
    def _merge_file_metadata(
        *,
        metadata: dict,
        filename: str,
        mime_type: str | None,
        byte_size: int,
    ) -> dict:
        return {
            **metadata,
            "filename": filename,
            "mime_type": mime_type,
            "byte_size": byte_size,
        }

    def _build_chunks_for_document(
        self,
        *,
        request: ChunkBuildRequest,
    ) -> tuple[str, UUID, list[ChunkDraft]]:
        content_hash = build_content_hash(request.normalized_text)
        ensure_project_exists(
            project_id=request.project_id,
            owner_user_id=request.actor_user_id,
        )
        document_id = build_document_id(
            owner_user_id=request.actor_user_id,
            project_id=request.project_id,
            title=request.title,
            content_hash=content_hash,
        )
        chunks = chunk_document_text(
            content_hash=content_hash,
            text=request.normalized_text,
            chunk_size_chars=request.chunk_size_chars,
            chunk_overlap_chars=request.chunk_overlap_chars,
        )
        if not chunks:
            raise IngestionServiceError(request.empty_chunks_detail)
        return content_hash, document_id, chunks

    def ingest_text(
        self,
        *,
        actor_user_id: UUID,
        payload: DocumentIngestTextRequest,
    ) -> DocumentIngestResponse:
        self._validate_chunk_shape(payload.chunk_size_chars, payload.chunk_overlap_chars)

        normalized_text = normalize_document_text(payload.text)
        if not normalized_text:
            raise IngestionServiceError("Document text cannot be empty")
        self._validate_text_size(normalized_text)

        content_hash, document_id, chunks = self._build_chunks_for_document(
            request=ChunkBuildRequest(
                actor_user_id=actor_user_id,
                project_id=payload.project_id,
                title=payload.title,
                normalized_text=normalized_text,
                chunk_size_chars=payload.chunk_size_chars,
                chunk_overlap_chars=payload.chunk_overlap_chars,
                empty_chunks_detail="No chunks were produced from document text",
            )
        )

        document = upsert_document_with_chunks(
            actor_user_id=actor_user_id,
            payload=DocumentUpsertPayload(
                document_id=document_id,
                project_id=payload.project_id,
                title=payload.title.strip(),
                source_type=DocumentSourceType.text.value,
                source_name=None,
                mime_type="text/plain",
                visibility_scope=payload.visibility_scope.value,
                content_text=normalized_text,
                content_hash=content_hash,
                metadata=payload.metadata,
                chunks=chunks,
            ),
            embedding_dim=self._embedding_dim,
        )
        return DocumentIngestResponse(document=document)

    def ingest_file(
        self,
        *,
        actor_user_id: UUID,
        payload: DocumentIngestFileRequest,
        file_request: FileIngestRequest,
    ) -> DocumentIngestResponse:
        self._validate_chunk_shape(payload.chunk_size_chars, payload.chunk_overlap_chars)
        normalized_text = self._validate_file_input(request=file_request)
        resolved_title = self._resolve_file_title(
            requested_title=payload.title,
            filename=file_request.filename,
        )
        content_hash, document_id, chunks = self._build_chunks_for_document(
            request=ChunkBuildRequest(
                actor_user_id=actor_user_id,
                project_id=payload.project_id,
                title=resolved_title,
                normalized_text=normalized_text,
                chunk_size_chars=payload.chunk_size_chars,
                chunk_overlap_chars=payload.chunk_overlap_chars,
                empty_chunks_detail="No chunks were produced from file contents",
            )
        )

        merged_metadata = self._merge_file_metadata(
            metadata=payload.metadata,
            filename=file_request.filename,
            mime_type=file_request.mime_type,
            byte_size=len(file_request.content_bytes),
        )

        document = upsert_document_with_chunks(
            actor_user_id=actor_user_id,
            payload=DocumentUpsertPayload(
                document_id=document_id,
                project_id=payload.project_id,
                title=resolved_title,
                source_type=DocumentSourceType.file.value,
                source_name=file_request.filename,
                mime_type=file_request.mime_type,
                visibility_scope=payload.visibility_scope.value,
                content_text=normalized_text,
                content_hash=content_hash,
                metadata=merged_metadata,
                chunks=chunks,
            ),
            embedding_dim=self._embedding_dim,
        )
        return DocumentIngestResponse(document=document)

    def list_documents(
        self,
        *,
        actor_user_id: UUID,
        project_id: str | None,
        limit: int,
        offset: int,
    ) -> list[DocumentRecord]:
        return list_documents(
            actor_user_id=actor_user_id,
            project_id=project_id,
            limit=limit,
            offset=offset,
        )

    def query_document_chunks(
        self,
        *,
        actor_user_id: UUID,
        payload: DocumentChunkQueryRequest,
    ) -> list[DocumentChunkQueryResult]:
        return query_document_chunks(
            actor_user_id=actor_user_id,
            request=payload,
            embedding_dim=self._embedding_dim,
        )

    def query_blended(
        self,
        *,
        actor_user_id: UUID,
        payload: BlendedRetrievalQueryRequest,
    ) -> BlendedRetrievalQueryResponse:
        engram_results = query_engrams(
            request=EngramQueryRequest(
                query=payload.query,
                project_id=payload.project_id,
                top_k=payload.top_k_engrams,
            ),
            embedding_dim=self._embedding_dim,
            actor_user_id=actor_user_id,
        )
        chunk_results = query_document_chunks(
            actor_user_id=actor_user_id,
            request=DocumentChunkQueryRequest(
                query=payload.query,
                project_id=payload.project_id,
                top_k=payload.top_k_document_chunks,
            ),
            embedding_dim=self._embedding_dim,
        )
        return BlendedRetrievalQueryResponse(
            engrams=engram_results,
            document_chunks=chunk_results,
        )
