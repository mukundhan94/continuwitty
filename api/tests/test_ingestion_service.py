from __future__ import annotations

from datetime import UTC, datetime
from uuid import uuid4

import pytest

from app.ingestion.errors import IngestionServiceError
from app.ingestion.models import (
    DocumentIngestFileRequest,
    DocumentIngestTextRequest,
    DocumentSourceType,
)
from app.ingestion.service import DocumentIngestionService


def _service() -> DocumentIngestionService:
    return DocumentIngestionService(
        embedding_dim=256,
        max_file_bytes=1024,
        max_text_chars=20000,
    )


def test_ingest_text_rejects_overlap_not_smaller_than_chunk_size() -> None:
    service = _service()

    with pytest.raises(IngestionServiceError) as exc:
        service.ingest_text(
            actor_user_id=uuid4(),
            payload=DocumentIngestTextRequest(
                project_id="engram-vault",
                title="Bad chunk shape",
                text="some text",
                chunk_size_chars=400,
                chunk_overlap_chars=400,
            ),
        )

    assert exc.value.status_code == 422


def test_ingest_file_rejects_non_utf8_bytes() -> None:
    service = _service()

    with pytest.raises(IngestionServiceError) as exc:
        service.ingest_file(
            actor_user_id=uuid4(),
            payload=DocumentIngestFileRequest(project_id="engram-vault"),
            filename="binary.bin",
            mime_type="application/octet-stream",
            content_bytes=b"\x80\x81\x82",
        )

    assert exc.value.status_code == 415


def test_ingest_text_persists_chunked_document(monkeypatch) -> None:
    service = _service()
    actor_id = uuid4()
    captured: dict = {}

    def _fake_upsert(**kwargs):  # noqa: ANN003
        payload = kwargs["payload"]
        captured.update(kwargs)
        return {
            "document_id": payload.document_id,
            "owner_user_id": actor_id,
            "project_id": payload.project_id,
            "title": payload.title,
            "source_type": DocumentSourceType.text.value,
            "source_name": None,
            "mime_type": "text/plain",
            "visibility_scope": payload.visibility_scope,
            "content_hash": payload.content_hash,
            "chunk_count": len(payload.chunks),
            "created_at": datetime.now(UTC),
            "updated_at": datetime.now(UTC),
        }

    monkeypatch.setattr("app.ingestion.service.ensure_project_exists", lambda **_: None)
    monkeypatch.setattr("app.ingestion.service.upsert_document_with_chunks", _fake_upsert)

    response = service.ingest_text(
        actor_user_id=actor_id,
        payload=DocumentIngestTextRequest(
            project_id="engram-vault",
            title="Incident Notes",
            text=" ".join(["incident"] * 720),
            chunk_size_chars=300,
            chunk_overlap_chars=80,
        ),
    )

    assert response.document.title == "Incident Notes"
    assert response.document.chunk_count >= 2
    assert captured["payload"].project_id == "engram-vault"
