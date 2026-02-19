from __future__ import annotations

from app.engram_enrichment.service import (
    derive_abstract_from_markdown_or_retrieval_text,
    enrich_if_missing,
    extract_keywords,
    map_keywords_to_tags,
)
from app.models import MemoryEngramCreate


def _payload(**updates) -> MemoryEngramCreate:
    base = {
        "project_id": "project-enrichment",
        "thread_id": "thread-1",
        "title": "Incident commander handoff",
        "abstract": "",
        "detailed_summary_markdown": (
            "# Chat Session Snapshot\n\n"
            "## USER\nSummarize the outage and next actions.\n\n"
            "## ASSISTANT\n"
            "Payment outage impact was isolated after queue drain and targeted rollback. "
            "Support and oncall teams aligned on mitigation and customer comms."
        ),
        "tags": [],
        "keywords": [],
        "visibility_scope": "project",
    }
    base.update(updates)
    return MemoryEngramCreate(**base)


def test_derive_abstract_prefers_assistant_section() -> None:
    abstract, source = derive_abstract_from_markdown_or_retrieval_text(
        "## ASSISTANT\nRoot cause identified in cache invalidation path."
    )

    assert "Root cause identified" in abstract
    assert source == "assistant_section"


def test_derive_abstract_uses_retrieval_text_when_markdown_is_empty() -> None:
    abstract, source = derive_abstract_from_markdown_or_retrieval_text(
        "",
        retrieval_text="Incident timeline with rollback checkpoints",
    )

    assert abstract == "Incident timeline with rollback checkpoints"
    assert source == "retrieval_text"


def test_extract_keywords_and_tag_mapping_are_deterministic() -> None:
    text = (
        "incident outage incident mitigation support support "
        "rollback mcp session monitoring support"
    )
    keywords = extract_keywords(text)
    tags = map_keywords_to_tags(keywords)

    assert keywords[:3] == ["support", "incident", "outage"]
    assert "incident" in tags
    assert "support" in tags


def test_enrich_if_missing_fills_empty_fields_only() -> None:
    result = enrich_if_missing(_payload(), origin="test.enrichment")

    assert result.report.enrichment_applied is True
    assert result.report.abstract_derived is True
    assert result.report.tags_derived is True
    assert result.report.keywords_derived is True
    assert result.payload.abstract
    assert result.payload.tags == result.report.auto_tags
    assert result.payload.keywords == result.report.auto_keywords


def test_enrich_if_missing_does_not_override_caller_metadata() -> None:
    payload = _payload(
        abstract="Keep this exact abstract",
        tags=["custom-tag"],
        keywords=["custom-keyword"],
    )

    result = enrich_if_missing(payload, origin="test.enrichment")

    assert result.report.enrichment_applied is False
    assert result.payload.abstract == "Keep this exact abstract"
    assert result.payload.tags == ["custom-tag"]
    assert result.payload.keywords == ["custom-keyword"]
