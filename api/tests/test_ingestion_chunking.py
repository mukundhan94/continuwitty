from __future__ import annotations

from uuid import UUID

from app.ingestion.chunking import (
    build_content_hash,
    build_document_id,
    chunk_document_text,
    normalize_document_text,
)


def test_chunk_document_text_is_deterministic_for_same_input() -> None:
    text = "\n\n".join(
        [
            "Incident started at 14:32 UTC with elevated latency across write path.",
            "Mitigation introduced read-only cache and connection pool cap.",
            "Service recovered by 17:02 UTC after query rollback and index hint.",
        ]
    )
    content_hash = build_content_hash(text)

    first = chunk_document_text(
        content_hash=content_hash,
        text=text,
        chunk_size_chars=90,
        chunk_overlap_chars=18,
    )
    second = chunk_document_text(
        content_hash=content_hash,
        text=text,
        chunk_size_chars=90,
        chunk_overlap_chars=18,
    )

    assert len(first) >= 2
    assert [item.chunk_id for item in first] == [item.chunk_id for item in second]
    assert [item.chunk_text for item in first] == [item.chunk_text for item in second]


def test_build_document_id_is_stable_for_same_fingerprint() -> None:
    owner_id = UUID("00000000-0000-0000-0000-000000000001")
    content_hash = build_content_hash("same content")

    first = build_document_id(owner_id, "engram-vault", "Runbook", content_hash)
    second = build_document_id(owner_id, "engram-vault", "Runbook", content_hash)

    assert first == second


def test_normalize_document_text_flattens_mixed_newlines() -> None:
    source = "Line A\r\nLine B\rLine C\n"
    normalized = normalize_document_text(source)
    assert normalized == "Line A\nLine B\nLine C"
