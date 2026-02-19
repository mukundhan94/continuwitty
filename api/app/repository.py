from __future__ import annotations

import logging
import re
from datetime import UTC, datetime
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


def create_engram_with_report(
    payload: MemoryEngramCreate,
    embedding_dim: int,
    owner_user_id: UUID | None = None,
    enrichment_origin: str = "unknown",
) -> tuple[EngramCreateResponse, dict]:
    # Fill-empty-only enrichment keeps explicit caller metadata untouched while
    # enabling low-friction persistence workflows (for example MCP conversation
    # snapshots that omit tags/keywords/abstract).
    resolved_payload = payload
    enrichment_report: dict = {
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
    try:
        enrichment_result = enrich_if_missing(payload=payload, origin=enrichment_origin)
        resolved_payload = enrichment_result.payload
        enrichment_report = enrichment_result.report.model_dump(mode="json")
    except Exception:
        # Enrichment must remain best-effort and never block writes.
        logger.warning(
            "engram auto metadata enrichment failed; persisting original caller payload",
            exc_info=True,
        )

    engram_id = uuid4()
    now = datetime.now(UTC)
    retrieval_text = _build_retrieval_text(resolved_payload)
    embedding_result = embed_text(retrieval_text, dim=embedding_dim)
    embedding_literal = _vector_literal(embedding_result.vector)

    engram_json = {
        "schema_version": "1.0",
        "project_id": resolved_payload.project_id,
        "thread_id": resolved_payload.thread_id,
        "title": resolved_payload.title,
        "abstract": resolved_payload.abstract,
        "detailed_summary_markdown": resolved_payload.detailed_summary_markdown,
        "decisions": [item.model_dump(mode="json") for item in resolved_payload.decisions],
        "assumptions": resolved_payload.assumptions,
        "open_questions": resolved_payload.open_questions,
        "claims": [item.model_dump(mode="json") for item in resolved_payload.claims],
        "tags": resolved_payload.tags,
        "keywords": resolved_payload.keywords,
        "artifacts": [item.model_dump(mode="json") for item in resolved_payload.artifacts],
        "visibility_scope": resolved_payload.visibility_scope,
        "source_session_id": (
            str(resolved_payload.source_session_id) if resolved_payload.source_session_id else None
        ),
        "auto_metadata": enrichment_report,
        "created_at": now.isoformat(),
    }

    with get_conn() as conn, conn.cursor() as cur:
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
                "project_id": resolved_payload.project_id,
                "thread_id": resolved_payload.thread_id,
                "created_at": now,
                "updated_at": now,
                "title": resolved_payload.title,
                "abstract": resolved_payload.abstract,
                "engram_json": Jsonb(engram_json),
                "engram_markdown": resolved_payload.detailed_summary_markdown,
                "tags": resolved_payload.tags,
                "keywords": resolved_payload.keywords,
                "owner_user_id": owner_user_id,
                "visibility_scope": resolved_payload.visibility_scope,
                "source_session_id": resolved_payload.source_session_id,
                "retrieval_text": retrieval_text,
                "embedding_model": embedding_result.provider_id,
                "embed": embedding_literal,
            },
        )

        for claim in resolved_payload.claims:
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

        for artifact in resolved_payload.artifacts:
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
    where_clauses: list[str] = []
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

    where_clauses: list[str] = []
    params: list = [query_literal]

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

    where_sql = ""
    if where_clauses:
        where_sql = "WHERE " + " AND ".join(where_clauses)

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

    reranked_rows = []
    for row in rows:
        lexical_overlap = _lexical_overlap_score(
            request.query,
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
    trimmed = [row for _, row in reranked_rows[: request.top_k]]
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
    where_clause = "WHERE engram_id = %s"
    where_params: list = [engram_id]
    if actor_user_id:
        where_clause += (
            " AND (owner_user_id = %s OR visibility_scope = 'project' OR owner_user_id IS NULL)"
        )
        where_params.append(actor_user_id)

    with get_conn() as conn, conn.cursor() as cur:
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
        row = cur.fetchone()
        if not row:
            return None

        cur.execute(
            """
            SELECT url, title, snippet, captured_at
            FROM sources
            WHERE engram_id = %s
            ORDER BY captured_at DESC
            LIMIT 25
            """,
            (engram_id,),
        )
        source_rows = cur.fetchall()

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

    citation_lines = []
    for citation in packed_citations:
        label = citation.title or citation.url
        snippet = (citation.snippet or "").replace("\n", " ").strip()
        if len(snippet) > 140:
            snippet = snippet[:137] + "..."
        line = f"- {label} ({citation.url})"
        if snippet:
            line += f": {snippet}"
        citation_lines.append(line)

    citation_text = "\n".join(citation_lines) if citation_lines else "- No citations available"
    decisions_text = (
        "\n".join(
            f"- {item.get('decision', '')}: {item.get('rationale', '')}" for item in decisions
        )
        or "- None"
    )
    questions_text = "\n".join(f"- {question}" for question in open_questions) or "- None"
    sections = [
        f"# Rehydration Context: {row['title']}",
        f"## Compact Summary\n{compact_summary}",
    ]
    if detailed_excerpt:
        sections.append(f"## Detailed Notes Excerpt\n{detailed_excerpt}")
    sections.extend(
        [
            f"## Key Decisions\n{decisions_text}",
            f"## Open Questions\n{questions_text}",
            f"## Top Citations\n{citation_text}",
        ]
    )
    context_markdown = "\n\n".join(sections)

    return RehydrationBundle(
        engram_id=row["engram_id"],
        project_id=row["project_id"],
        title=row["title"],
        compact_summary=compact_summary,
        detailed_summary_markdown=detailed_summary_markdown,
        key_decisions=decisions,
        open_questions=open_questions,
        top_citations=packed_citations,
        context_markdown=context_markdown,
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
