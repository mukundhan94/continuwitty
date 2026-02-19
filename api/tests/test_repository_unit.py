from __future__ import annotations

from datetime import UTC, datetime
from uuid import uuid4

from app.models import EngramQueryRequest, RehydrationCitation
from app.repository import (
    _build_engram_query_where,
    _format_citations,
    _format_decisions,
    _rerank_by_combined_score,
)


def test_build_engram_query_where_includes_all_filters() -> None:
    actor_user_id = uuid4()
    created_after = datetime(2026, 2, 1, tzinfo=UTC)
    created_before = datetime(2026, 2, 19, tzinfo=UTC)
    request = EngramQueryRequest(
        query="durable memory",
        top_k=5,
        project_id="engram-vault",
        tags=["memory"],
        keywords=["checkpoint"],
        created_after=created_after,
        created_before=created_before,
    )

    where_sql, params = _build_engram_query_where(
        request=request,
        actor_user_id=actor_user_id,
        query_literal="[0.1,0.2,0.3]",
    )

    assert "WHERE deleted_at IS NULL" in where_sql
    assert "project_id = %s" in where_sql
    assert "tags && %s" in where_sql
    assert "keywords && %s" in where_sql
    assert "created_at >= %s" in where_sql
    assert "created_at <= %s" in where_sql
    assert "visibility_scope = 'project'" in where_sql
    assert params == [
        "[0.1,0.2,0.3]",
        "engram-vault",
        actor_user_id,
        ["memory"],
        ["checkpoint"],
        created_after,
        created_before,
    ]


def test_rerank_by_combined_score_prefers_lexical_overlap() -> None:
    rows = [
        {
            "engram_id": uuid4(),
            "project_id": "engram-vault",
            "title": "Dense Match",
            "abstract": "",
            "created_at": datetime(2026, 2, 18, tzinfo=UTC),
            "tags": [],
            "keywords": [],
            "retrieval_text": "unrelated text",
            "distance": 0.2,
        },
        {
            "engram_id": uuid4(),
            "project_id": "engram-vault",
            "title": "Lexical Match",
            "abstract": "",
            "created_at": datetime(2026, 2, 17, tzinfo=UTC),
            "tags": [],
            "keywords": ["durable", "checkpoint"],
            "retrieval_text": "durable checkpoint lifecycle",
            "distance": 0.25,
        },
    ]

    reranked = _rerank_by_combined_score(
        rows=rows,
        query="durable checkpoint",
        top_k=1,
    )

    assert len(reranked) == 1
    assert reranked[0]["title"] == "Lexical Match"


def test_format_citations_truncates_and_strips_newlines() -> None:
    citation_text = _format_citations(
        [
            RehydrationCitation(
                url="https://example.com/source",
                title="Primary source",
                snippet="line1\n" + ("x" * 200),
                captured_at=datetime(2026, 2, 19, tzinfo=UTC),
            )
        ]
    )

    assert citation_text.startswith("- Primary source (https://example.com/source):")
    assert "\n" not in citation_text.split(":", maxsplit=1)[1]
    assert citation_text.endswith("...")


def test_format_citations_returns_default_for_empty_list() -> None:
    assert _format_citations([]) == "- No citations available"


def test_format_decisions_formats_entries_and_defaults() -> None:
    decisions_text = _format_decisions(
        [
            {
                "decision": "Use durable checkpoints",
                "rationale": "Prevents context loss",
            }
        ]
    )
    assert decisions_text == "- Use durable checkpoints: Prevents context loss"
    assert _format_decisions([]) == "- None"
