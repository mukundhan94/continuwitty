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
from app.ingestion.service import DocumentIngestionService, FileIngestRequest


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


@pytest.mark.parametrize(
    ("file_request", "expected_status", "expected_detail"),
    [
        (
            FileIngestRequest(
                filename="binary.bin",
                mime_type="application/octet-stream",
                content_bytes=b"\x80\x81\x82",
            ),
            415,
            "Only UTF-8 text files are supported in this phase",
        ),
        (
            FileIngestRequest(
                filename="notes.txt",
                mime_type="text/plain",
                content_bytes=b"",
            ),
            400,
            "Uploaded file is empty",
        ),
        (
            FileIngestRequest(
                filename="big.txt",
                mime_type="text/plain",
                content_bytes=b"x" * 1025,
            ),
            413,
            "Uploaded file exceeds max allowed size of 1024 bytes",
        ),
    ],
)
def test_ingest_file_validation_errors(
    *,
    file_request: FileIngestRequest,
    expected_status: int,
    expected_detail: str,
) -> None:
    service = _service()

    with pytest.raises(IngestionServiceError) as exc:
        service.ingest_file(
            actor_user_id=uuid4(),
            payload=DocumentIngestFileRequest(project_id="engram-vault"),
            file_request=file_request,
        )

    assert exc.value.status_code == expected_status
    assert exc.value.detail == expected_detail


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


def test_ingest_file_persists_file_metadata_and_fallback_title(monkeypatch) -> None:
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
            "source_type": DocumentSourceType.file.value,
            "source_name": payload.source_name,
            "mime_type": payload.mime_type,
            "visibility_scope": payload.visibility_scope,
            "content_hash": payload.content_hash,
            "chunk_count": len(payload.chunks),
            "created_at": datetime.now(UTC),
            "updated_at": datetime.now(UTC),
        }

    monkeypatch.setattr("app.ingestion.service.ensure_project_exists", lambda **_: None)
    monkeypatch.setattr("app.ingestion.service.upsert_document_with_chunks", _fake_upsert)

    content = "# On-call Notes\n\n" + " ".join(["queue-depth"] * 40)
    response = service.ingest_file(
        actor_user_id=actor_id,
        payload=DocumentIngestFileRequest(
            project_id="engram-vault",
            metadata={"domain": "incident"},
            chunk_size_chars=320,
            chunk_overlap_chars=80,
        ),
        file_request=FileIngestRequest(
            filename="runbook.md",
            mime_type="text/markdown",
            content_bytes=content.encode("utf-8"),
        ),
    )

    assert response.document.title == "runbook"
    assert response.document.chunk_count >= 1
    assert captured["payload"].source_name == "runbook.md"
    assert captured["payload"].mime_type == "text/markdown"
    assert captured["payload"].metadata["domain"] == "incident"
    assert captured["payload"].metadata["filename"] == "runbook.md"
    assert captured["payload"].metadata["mime_type"] == "text/markdown"
    assert captured["payload"].metadata["byte_size"] == len(content.encode("utf-8"))
