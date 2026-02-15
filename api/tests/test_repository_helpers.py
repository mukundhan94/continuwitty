from app.models import MemoryEngramCreate
from app.repository import _build_retrieval_text, _vector_literal


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
