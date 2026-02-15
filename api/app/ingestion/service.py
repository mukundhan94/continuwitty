from __future__ import annotations

from pathlib import Path
from uuid import UUID

from app.ingestion.chunking import (
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
    list_documents,
    query_document_chunks,
    upsert_document_with_chunks,
)
from app.models import EngramQueryRequest
from app.repository import query_engrams


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

        content_hash = build_content_hash(normalized_text)
        document_id = build_document_id(
            owner_user_id=actor_user_id,
            project_id=payload.project_id,
            title=payload.title,
            content_hash=content_hash,
        )

        chunks = chunk_document_text(
            content_hash=content_hash,
            text=normalized_text,
            chunk_size_chars=payload.chunk_size_chars,
            chunk_overlap_chars=payload.chunk_overlap_chars,
        )
        if not chunks:
            raise IngestionServiceError("No chunks were produced from document text")

        document = upsert_document_with_chunks(
            document_id=document_id,
            actor_user_id=actor_user_id,
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
            embedding_dim=self._embedding_dim,
        )
        return DocumentIngestResponse(document=document)

    def ingest_file(
        self,
        *,
        actor_user_id: UUID,
        payload: DocumentIngestFileRequest,
        filename: str,
        mime_type: str | None,
        content_bytes: bytes,
    ) -> DocumentIngestResponse:
        if not content_bytes:
            raise IngestionServiceError("Uploaded file is empty")
        if len(content_bytes) > self._max_file_bytes:
            raise IngestionServiceError(
                f"Uploaded file exceeds max allowed size of {self._max_file_bytes} bytes",
                status_code=413,
            )

        self._validate_chunk_shape(payload.chunk_size_chars, payload.chunk_overlap_chars)

        try:
            decoded = content_bytes.decode("utf-8")
        except UnicodeDecodeError as exc:
            raise IngestionServiceError(
                "Only UTF-8 text files are supported in this phase",
                status_code=415,
            ) from exc

        normalized_text = normalize_document_text(decoded)
        if not normalized_text:
            raise IngestionServiceError("Uploaded file does not contain readable text")
        self._validate_text_size(normalized_text)

        resolved_title = (payload.title or Path(filename).stem or "Untitled Document").strip()
        content_hash = build_content_hash(normalized_text)
        document_id = build_document_id(
            owner_user_id=actor_user_id,
            project_id=payload.project_id,
            title=resolved_title,
            content_hash=content_hash,
        )

        chunks = chunk_document_text(
            content_hash=content_hash,
            text=normalized_text,
            chunk_size_chars=payload.chunk_size_chars,
            chunk_overlap_chars=payload.chunk_overlap_chars,
        )
        if not chunks:
            raise IngestionServiceError("No chunks were produced from file contents")

        merged_metadata = {
            **payload.metadata,
            "filename": filename,
            "mime_type": mime_type,
            "byte_size": len(content_bytes),
        }

        document = upsert_document_with_chunks(
            document_id=document_id,
            actor_user_id=actor_user_id,
            project_id=payload.project_id,
            title=resolved_title,
            source_type=DocumentSourceType.file.value,
            source_name=filename,
            mime_type=mime_type,
            visibility_scope=payload.visibility_scope.value,
            content_text=normalized_text,
            content_hash=content_hash,
            metadata=merged_metadata,
            chunks=chunks,
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
