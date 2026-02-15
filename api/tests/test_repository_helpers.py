from datetime import UTC, datetime

from app.models import MemoryEngramCreate, RehydrationCitation
from app.repository import (
    _build_retrieval_text,
    _combined_rank_score,
    _lexical_overlap_score,
    _pack_citations,
    _vector_literal,
)


def test_vector_literal_format() -> None:
    literal = _vector_literal([0.1, -0.2, 0.0])
    assert literal.startswith("[")
    assert literal.endswith("]")
    assert "," in literal


def test_build_retrieval_text_uses_override() -> None:
    payload = MemoryEngramCreate(
        project_id="p1",
        title="T",
        abstract="A",
        detailed_summary_markdown="D",
        retrieval_text="override retrieval",
    )
    assert _build_retrieval_text(payload) == "override retrieval"


def test_build_retrieval_text_fallback_composes_fields() -> None:
    payload = MemoryEngramCreate(
        project_id="p1",
        title="LangGraph Choice",
        abstract="Selected for durable checkpoints",
        detailed_summary_markdown="details",
        open_questions=["When to rerank?"],
    )
    value = _build_retrieval_text(payload)
    assert "LangGraph Choice" in value
    assert "Selected for durable checkpoints" in value
    assert "When to rerank?" in value


def test_lexical_overlap_score_prefers_matching_terms() -> None:
    score = _lexical_overlap_score(
        "durable checkpoint workflow",
        ["LangGraph enables durable checkpoint workflow execution"],
    )
    assert score > 0.6


def test_combined_rank_score_uses_dense_and_lexical_signals() -> None:
    weak_dense_strong_lexical = _combined_rank_score(distance=0.8, lexical_overlap=1.0)
    strong_dense_weak_lexical = _combined_rank_score(distance=0.1, lexical_overlap=0.0)
    assert weak_dense_strong_lexical > 0.0
    assert strong_dense_weak_lexical > weak_dense_strong_lexical


def test_pack_citations_deduplicates_urls() -> None:
    citations = [
        RehydrationCitation(
            url="https://example.com/a",
            title="A1",
            snippet="first",
            captured_at=datetime(2026, 2, 15, tzinfo=UTC),
        ),
        RehydrationCitation(
            url="https://example.com/a",
            title="A2",
            snippet="duplicate",
            captured_at=datetime(2026, 2, 15, tzinfo=UTC),
        ),
        RehydrationCitation(
            url="https://example.com/b",
            title="B",
            snippet="second",
            captured_at=datetime(2026, 2, 15, tzinfo=UTC),
        ),
    ]
    packed = _pack_citations(citations, limit=5)
    assert len(packed) == 2
    assert packed[0].url == "https://example.com/a"
    assert packed[1].url == "https://example.com/b"
