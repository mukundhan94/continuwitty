from __future__ import annotations

import logging
import re
from dataclasses import dataclass
from datetime import UTC, datetime
from typing import Any
from uuid import UUID, uuid4

from psycopg.types.json import Jsonb

from .db import get_conn
from .embeddings import embed_text
from .engram_enrichment import enrich_if_missing
from .models import (
    EngramCreateResponse,
    EngramQueryRequest,
    EngramQueryResult,
    EngramSourceRecord,
    EngramSummary,
    MemoryEngramCreate,
    RehydrationBundle,
    RehydrationCitation,
)

logger = logging.getLogger(__name__)

_TOKEN_PATTERN = re.compile(r"[a-z0-9]{2,}")
_ASSISTANT_SECTION_PATTERN = re.compile(
    r"^##\s*ASSISTANT[^\n]*\n(?P<body>.*?)(?=^##\s|\Z)",
    re.IGNORECASE | re.MULTILINE | re.DOTALL,
)
_GENERIC_CHAT_ABSTRACTS = {
    "",
    "snapshot from active chat session.",
    "snapshot from active chat session",
    "chat snapshot",
    "session snapshot",
}


@dataclass(frozen=True)
class _RehydrationContent:
    compact_summary: str
    detailed_summary_markdown: str
    decisions: list[dict[str, Any]]
    open_questions: list[str]
    packed_citations: list[RehydrationCitation]
    context_markdown: str


def _vector_literal(values: list[float]) -> str:
    return "[" + ",".join(f"{value:.6f}" for value in values) + "]"


def _build_retrieval_text(payload: MemoryEngramCreate) -> str:
    if payload.retrieval_text:
        return payload.retrieval_text

    decision_text = " ".join(d.decision for d in payload.decisions)
    question_text = " ".join(payload.open_questions)
    claim_text = " ".join(c.claim for c in payload.claims)
    return " ".join(
        [
            payload.title,
            payload.abstract,
            decision_text,
            question_text,
            claim_text,
        ]
    ).strip()


def _tokenize(text: str) -> set[str]:
    return {match.group(0) for match in _TOKEN_PATTERN.finditer(text.lower())}


def _normalize_spaces(value: str) -> str:
    return re.sub(r"\s+", " ", value.strip())


def _truncate_text(value: str, max_chars: int) -> str:
    if len(value) <= max_chars:
        return value
    return value[: max_chars - 3].rstrip() + "..."


def _extract_detailed_excerpt(markdown: str, max_chars: int) -> str:
    text = markdown.strip()
    if not text:
        return ""

    assistant_match = _ASSISTANT_SECTION_PATTERN.search(text)
    excerpt = assistant_match.group("body").strip() if assistant_match else text
    if not excerpt:
        return ""
    return _truncate_text(excerpt, max_chars=max_chars)


def _resolve_compact_summary(
    *,
    abstract: str,
    detailed_summary_markdown: str,
    max_chars: int = 800,
) -> str:
    abstract_clean = _normalize_spaces(abstract)
    if abstract_clean.lower() not in _GENERIC_CHAT_ABSTRACTS:
        return _truncate_text(abstract_clean, max_chars=max_chars)

    fallback = _normalize_spaces(
        _extract_detailed_excerpt(detailed_summary_markdown, max_chars=max_chars)
    )
    if fallback:
        return fallback
    if abstract_clean:
        return _truncate_text(abstract_clean, max_chars=max_chars)
    return "No summary available."


def _lexical_overlap_score(query: str, candidate_parts: list[str]) -> float:
    query_tokens = _tokenize(query)
    if not query_tokens:
        return 0.0

    doc_tokens: set[str] = set()
    for part in candidate_parts:
        doc_tokens.update(_tokenize(part))

    if not doc_tokens:
        return 0.0

    overlap = query_tokens & doc_tokens
    return len(overlap) / len(query_tokens)


def _combined_rank_score(distance: float, lexical_overlap: float) -> float:
    dense_score = 1.0 / (1.0 + max(distance, 0.0))
    return (dense_score * 0.8) + (lexical_overlap * 0.2)


def _pack_citations(
    citations: list[RehydrationCitation], limit: int = 5
) -> list[RehydrationCitation]:
    packed: list[RehydrationCitation] = []
    seen_urls: set[str] = set()
    for citation in citations:
        url_key = citation.url.strip().lower()
        if url_key in seen_urls:
            continue
        seen_urls.add(url_key)
        packed.append(citation)
        if len(packed) >= limit:
            break
    return packed


def _build_engram_query_where(
    *,
    request: EngramQueryRequest,
    actor_user_id: UUID | None,
    query_literal: str,
) -> tuple[str, list[Any]]:
    where_clauses: list[str] = ["deleted_at IS NULL"]
    params: list[Any] = [query_literal]

    if request.project_id:
        where_clauses.append("project_id = %s")
        params.append(request.project_id)
    if actor_user_id:
        where_clauses.append(
            "(owner_user_id = %s OR visibility_scope = 'project' OR owner_user_id IS NULL)"
        )
        params.append(actor_user_id)
    if request.tags:
        where_clauses.append("tags && %s")
        params.append(request.tags)
    if request.keywords:
        where_clauses.append("keywords && %s")
        params.append(request.keywords)
    if request.created_after:
        where_clauses.append("created_at >= %s")
        params.append(request.created_after)
    if request.created_before:
        where_clauses.append("created_at <= %s")
        params.append(request.created_before)

    where_sql = "WHERE " + " AND ".join(where_clauses)
    return where_sql, params


def _rerank_by_combined_score(
    *,
    rows: list[dict[str, Any]],
    query: str,
    top_k: int,
) -> list[dict[str, Any]]:
    reranked_rows: list[tuple[float, dict[str, Any]]] = []
    for row in rows:
        lexical_overlap = _lexical_overlap_score(
            query,
            [
                row.get("title") or "",
                row.get("abstract") or "",
                row.get("retrieval_text") or "",
                " ".join(row.get("tags") or []),
                " ".join(row.get("keywords") or []),
            ],
        )
        combined_score = _combined_rank_score(float(row["distance"]), lexical_overlap)
        reranked_rows.append((combined_score, row))

    reranked_rows.sort(
        key=lambda item: (
            item[0],
            item[1]["created_at"],
        ),
        reverse=True,
    )
    return [row for _, row in reranked_rows[:top_k]]


def _fetch_engram_row(
    *,
    cur: Any,
    engram_id: UUID,
    actor_user_id: UUID | None,
) -> dict[str, Any] | None:
    where_clause = "WHERE engram_id = %s AND deleted_at IS NULL"
    where_params: list[Any] = [engram_id]
    if actor_user_id:
        where_clause += (
            " AND (owner_user_id = %s OR visibility_scope = 'project' OR owner_user_id IS NULL)"
        )
        where_params.append(actor_user_id)

    cur.execute(
        f"""
            SELECT
                engram_id, project_id, title, abstract, engram_json, engram_markdown,
                owner_user_id, visibility_scope
            FROM engrams
            {where_clause}
            """,
        where_params,
    )
    return cur.fetchone()


def _fetch_source_rows(
    *,
    cur: Any,
    engram_id: UUID,
    limit: int = 25,
) -> list[dict[str, Any]]:
    cur.execute(
        """
        SELECT url, title, snippet, captured_at
        FROM sources
        WHERE engram_id = %s
        ORDER BY captured_at DESC
        LIMIT %s
        """,
        (engram_id, limit),
    )
    return cur.fetchall()


def _format_citations(citations: list[RehydrationCitation]) -> str:
    citation_lines = []
    for citation in citations:
        label = citation.title or citation.url
        snippet = (citation.snippet or "").replace("\n", " ").strip()
        if len(snippet) > 140:
            snippet = snippet[:137] + "..."
        line = f"- {label} ({citation.url})"
        if snippet:
            line += f": {snippet}"
        citation_lines.append(line)
    return "\n".join(citation_lines) if citation_lines else "- No citations available"


def _format_decisions(decisions: list[dict[str, Any]]) -> str:
    formatted = "\n".join(
        f"- {item.get('decision', '')}: {item.get('rationale', '')}" for item in decisions
    )
    return formatted or "- None"


def _format_open_questions(open_questions: list[str]) -> str:
    return "\n".join(f"- {question}" for question in open_questions) or "- None"


def _build_rehydration_context_markdown(
    *,
    title: str,
    compact_summary: str,
    detailed_excerpt: str,
    decisions: list[dict[str, Any]],
    open_questions: list[str],
    citations: list[RehydrationCitation],
) -> str:
    sections = [
        f"# Rehydration Context: {title}",
        f"## Compact Summary\n{compact_summary}",
    ]
    if detailed_excerpt:
        sections.append(f"## Detailed Notes Excerpt\n{detailed_excerpt}")
    sections.extend(
        [
            f"## Key Decisions\n{_format_decisions(decisions)}",
            f"## Open Questions\n{_format_open_questions(open_questions)}",
            f"## Top Citations\n{_format_citations(citations)}",
        ]
    )
    return "\n\n".join(sections)


def _default_enrichment_report(*, enrichment_origin: str) -> dict[str, Any]:
    return {
        "schema_version": "1.0",
        "origin": enrichment_origin,
        "enrichment_applied": False,
        "abstract_derived": False,
        "tags_derived": False,
        "keywords_derived": False,
        "auto_tags": [],
        "auto_keywords": [],
        "abstract_source": None,
    }


def _resolve_enriched_payload(
    *,
    payload: MemoryEngramCreate,
    enrichment_origin: str,
) -> tuple[MemoryEngramCreate, dict[str, Any]]:
    enrichment_report = _default_enrichment_report(enrichment_origin=enrichment_origin)
    try:
        enrichment_result = enrich_if_missing(payload=payload, origin=enrichment_origin)
    except Exception:
        # Enrichment must remain best-effort and never block writes.
        logger.warning(
            "engram auto metadata enrichment failed; persisting original caller payload",
            exc_info=True,
        )
        return payload, enrichment_report
    return enrichment_result.payload, enrichment_result.report.model_dump(mode="json")


def _build_engram_json_payload(
    *,
    payload: MemoryEngramCreate,
    enrichment_report: dict[str, Any],
    created_at: datetime,
) -> dict[str, Any]:
    return {
        "schema_version": "1.0",
        "project_id": payload.project_id,
        "thread_id": payload.thread_id,
        "title": payload.title,
        "abstract": payload.abstract,
        "detailed_summary_markdown": payload.detailed_summary_markdown,
        "decisions": [item.model_dump(mode="json") for item in payload.decisions],
        "assumptions": payload.assumptions,
        "open_questions": payload.open_questions,
        "claims": [item.model_dump(mode="json") for item in payload.claims],
        "tags": payload.tags,
        "keywords": payload.keywords,
        "artifacts": [item.model_dump(mode="json") for item in payload.artifacts],
        "visibility_scope": payload.visibility_scope,
        "source_session_id": str(payload.source_session_id) if payload.source_session_id else None,
        "auto_metadata": enrichment_report,
        "created_at": created_at.isoformat(),
    }


def _insert_engram_row(
    *,
    cur: Any,
    engram_id: UUID,
    payload: MemoryEngramCreate,
    created_at: datetime,
    owner_user_id: UUID | None,
    retrieval_text: str,
    embedding_model: str,
    embedding_literal: str,
    engram_json: dict[str, Any],
) -> None:
    cur.execute(
        """
            INSERT INTO engrams (
                engram_id, project_id, thread_id, created_at, updated_at, schema_version,
                title, abstract, engram_json, engram_markdown, tags, keywords, owner_user_id,
                visibility_scope, source_session_id, retrieval_text, embedding_model, embed
            )
            VALUES (
                %(engram_id)s, %(project_id)s, %(thread_id)s, %(created_at)s, %(updated_at)s, '1.0',
                %(title)s, %(abstract)s, %(engram_json)s, %(engram_markdown)s, %(tags)s, %(keywords)s, %(owner_user_id)s,
                %(visibility_scope)s, %(source_session_id)s, %(retrieval_text)s, %(embedding_model)s, %(embed)s::vector
            )
            """,
        {
            "engram_id": engram_id,
            "project_id": payload.project_id,
            "thread_id": payload.thread_id,
            "created_at": created_at,
            "updated_at": created_at,
            "title": payload.title,
            "abstract": payload.abstract,
            "engram_json": Jsonb(engram_json),
            "engram_markdown": payload.detailed_summary_markdown,
            "tags": payload.tags,
            "keywords": payload.keywords,
            "owner_user_id": owner_user_id,
            "visibility_scope": payload.visibility_scope,
            "source_session_id": payload.source_session_id,
            "retrieval_text": retrieval_text,
            "embedding_model": embedding_model,
            "embed": embedding_literal,
        },
    )


def _insert_claim_sources(
    *,
    cur: Any,
    engram_id: UUID,
    payload: MemoryEngramCreate,
) -> None:
    for claim in payload.claims:
        for source in claim.supporting_sources:
            cur.execute(
                """
                    INSERT INTO sources (
                        source_id, engram_id, captured_at, url, title, snippet, content_text, content_hash
                    )
                    VALUES (%s, %s, %s, %s, %s, %s, NULL, NULL)
                    """,
                (
                    uuid4(),
                    engram_id,
                    source.captured_at,
                    source.url,
                    source.title,
                    source.snippet,
                ),
            )


def _insert_artifacts(
    *,
    cur: Any,
    engram_id: UUID,
    payload: MemoryEngramCreate,
) -> None:
    for artifact in payload.artifacts:
        cur.execute(
            """
                INSERT INTO artifacts (artifact_id, engram_id, artifact_type, storage_uri, metadata)
                VALUES (%s, %s, %s, %s, %s)
                """,
            (
                uuid4(),
                engram_id,
                artifact.artifact_type,
                artifact.storage_uri,
                Jsonb(artifact.metadata),
            ),
        )


def _fetch_rehydration_rows(
    *,
    engram_id: UUID,
    actor_user_id: UUID | None,
) -> tuple[dict[str, Any], list[dict[str, Any]]] | None:
    with get_conn() as conn, conn.cursor() as cur:
        row = _fetch_engram_row(
            cur=cur,
            engram_id=engram_id,
            actor_user_id=actor_user_id,
        )
        if not row:
            return None
        source_rows = _fetch_source_rows(
            cur=cur,
            engram_id=engram_id,
            limit=25,
        )
    return row, source_rows


def _build_rehydration_content(
    *,
    row: dict[str, Any],
    source_rows: list[dict[str, Any]],
) -> _RehydrationContent:
    engram_json = row["engram_json"] or {}
    decisions = engram_json.get("decisions", [])
    open_questions = engram_json.get("open_questions", [])
    citations = [RehydrationCitation(**source_row) for source_row in source_rows]
    packed_citations = _pack_citations(citations, limit=5)

    detailed_summary_markdown = str(
        engram_json.get("detailed_summary_markdown") or row.get("engram_markdown") or ""
    ).strip()
    compact_summary = _resolve_compact_summary(
        abstract=str(row.get("abstract") or ""),
        detailed_summary_markdown=detailed_summary_markdown,
    )
    detailed_excerpt = _extract_detailed_excerpt(detailed_summary_markdown, max_chars=2400)
    context_markdown = _build_rehydration_context_markdown(
        title=row["title"],
        compact_summary=compact_summary,
        detailed_excerpt=detailed_excerpt,
        decisions=decisions,
        open_questions=open_questions,
        citations=packed_citations,
    )
    return _RehydrationContent(
        compact_summary=compact_summary,
        detailed_summary_markdown=detailed_summary_markdown,
        decisions=decisions,
        open_questions=open_questions,
        packed_citations=packed_citations,
        context_markdown=context_markdown,
    )


def create_engram_with_report(
    payload: MemoryEngramCreate,
    embedding_dim: int,
    owner_user_id: UUID | None = None,
    enrichment_origin: str = "unknown",
) -> tuple[EngramCreateResponse, dict]:
    resolved_payload, enrichment_report = _resolve_enriched_payload(
        payload=payload,
        enrichment_origin=enrichment_origin,
    )
    engram_id = uuid4()
    now = datetime.now(UTC)
    retrieval_text = _build_retrieval_text(resolved_payload)
    embedding_result = embed_text(retrieval_text, dim=embedding_dim)
    embedding_literal = _vector_literal(embedding_result.vector)
    engram_json = _build_engram_json_payload(
        payload=resolved_payload,
        enrichment_report=enrichment_report,
        created_at=now,
    )

    with get_conn() as conn, conn.cursor() as cur:
        _insert_engram_row(
            cur=cur,
            engram_id=engram_id,
            payload=resolved_payload,
            created_at=now,
            owner_user_id=owner_user_id,
            retrieval_text=retrieval_text,
            embedding_model=embedding_result.provider_id,
            embedding_literal=embedding_literal,
            engram_json=engram_json,
        )
        _insert_claim_sources(cur=cur, engram_id=engram_id, payload=resolved_payload)
        _insert_artifacts(cur=cur, engram_id=engram_id, payload=resolved_payload)

    return EngramCreateResponse(engram_id=engram_id, created_at=now), enrichment_report


def create_engram(
    payload: MemoryEngramCreate,
    embedding_dim: int,
    owner_user_id: UUID | None = None,
    enrichment_origin: str = "unknown",
) -> EngramCreateResponse:
    created, _ = create_engram_with_report(
        payload=payload,
        embedding_dim=embedding_dim,
        owner_user_id=owner_user_id,
        enrichment_origin=enrichment_origin,
    )
    return created


def list_engrams(
    project_id: str | None = None,
    limit: int = 25,
    offset: int = 0,
    actor_user_id: UUID | None = None,
) -> list[EngramSummary]:
    query = """
        SELECT
            engram_id, project_id, thread_id, title, abstract, created_at, tags, keywords,
            owner_user_id, visibility_scope
        FROM engrams
    """
    where_clauses: list[str] = ["deleted_at IS NULL"]
    params: list = []
    if actor_user_id:
        where_clauses.append(
            "(owner_user_id = %s OR visibility_scope = 'project' OR owner_user_id IS NULL)"
        )
        params.append(actor_user_id)
    if project_id:
        where_clauses.append("project_id = %s")
        params.append(project_id)
    if where_clauses:
        query += " WHERE " + " AND ".join(where_clauses)
    query += " ORDER BY created_at DESC LIMIT %s OFFSET %s"
    params.extend([limit, offset])

    with get_conn() as conn, conn.cursor() as cur:
        cur.execute(query, params)
        rows = cur.fetchall()

    return [EngramSummary(**row) for row in rows]


def query_engrams(
    request: EngramQueryRequest,
    embedding_dim: int,
    actor_user_id: UUID | None = None,
) -> list[EngramQueryResult]:
    query_embedding = embed_text(request.query, dim=embedding_dim)
    query_literal = _vector_literal(query_embedding.vector)
    where_sql, params = _build_engram_query_where(
        request=request,
        actor_user_id=actor_user_id,
        query_literal=query_literal,
    )

    candidate_limit = min(max(request.top_k * 4, request.top_k), 200)

    sql = f"""
        SELECT
            engram_id,
            project_id,
            title,
            abstract,
            created_at,
            tags,
            keywords,
            owner_user_id,
            visibility_scope,
            retrieval_text,
            embed <=> %s::vector AS distance
        FROM engrams
        {where_sql}
        ORDER BY distance ASC, created_at DESC
        LIMIT %s
    """
    params.append(candidate_limit)

    with get_conn() as conn, conn.cursor() as cur:
        cur.execute(sql, params)
        rows = cur.fetchall()

    trimmed = _rerank_by_combined_score(
        rows=rows,
        query=request.query,
        top_k=request.top_k,
    )
    return [
        EngramQueryResult(
            engram_id=row["engram_id"],
            project_id=row["project_id"],
            title=row["title"],
            abstract=row["abstract"],
            created_at=row["created_at"],
            tags=row.get("tags") or [],
            keywords=row.get("keywords") or [],
            owner_user_id=row.get("owner_user_id"),
            visibility_scope=row.get("visibility_scope") or "private",
            distance=row["distance"],
        )
        for row in trimmed
    ]


def get_rehydration_bundle(
    engram_id: UUID,
    actor_user_id: UUID | None = None,
) -> RehydrationBundle | None:
    rehydration_rows = _fetch_rehydration_rows(
        engram_id=engram_id,
        actor_user_id=actor_user_id,
    )
    if rehydration_rows is None:
        return None
    row, source_rows = rehydration_rows
    content = _build_rehydration_content(row=row, source_rows=source_rows)

    return RehydrationBundle(
        engram_id=row["engram_id"],
        project_id=row["project_id"],
        title=row["title"],
        compact_summary=content.compact_summary,
        detailed_summary_markdown=content.detailed_summary_markdown,
        key_decisions=content.decisions,
        open_questions=content.open_questions,
        top_citations=content.packed_citations,
        context_markdown=content.context_markdown,
        owner_user_id=row.get("owner_user_id"),
        visibility_scope=row.get("visibility_scope") or "private",
    )


def get_engram_sources(
    engram_id: UUID,
    limit: int = 100,
    actor_user_id: UUID | None = None,
) -> list[EngramSourceRecord]:
    bundle = get_rehydration_bundle(engram_id, actor_user_id=actor_user_id)
    if not bundle:
        return []

    with get_conn() as conn, conn.cursor() as cur:
        cur.execute(
            """
            SELECT source_id, engram_id, captured_at, url, title, snippet
            FROM sources
            WHERE engram_id = %s
            ORDER BY captured_at DESC
            LIMIT %s
            """,
            (engram_id, limit),
        )
        rows = cur.fetchall()
    return [EngramSourceRecord(**row) for row in rows]
