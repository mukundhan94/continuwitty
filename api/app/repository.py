from __future__ import annotations

from datetime import UTC, datetime
from uuid import UUID, uuid4

from psycopg.types.json import Jsonb

from .db import get_conn
from .embedding import embed_text_local
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


def create_engram(payload: MemoryEngramCreate, embedding_dim: int) -> EngramCreateResponse:
    engram_id = uuid4()
    now = datetime.now(UTC)
    retrieval_text = _build_retrieval_text(payload)
    embedding = embed_text_local(retrieval_text, embedding_dim)
    embedding_literal = _vector_literal(embedding)

    engram_json = {
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
        "created_at": now.isoformat(),
    }

    with get_conn() as conn, conn.cursor() as cur:
        cur.execute(
            """
                INSERT INTO engrams (
                    engram_id, project_id, thread_id, created_at, updated_at, schema_version,
                    title, abstract, engram_json, engram_markdown, tags, keywords,
                    retrieval_text, embedding_model, embed
                )
                VALUES (
                    %(engram_id)s, %(project_id)s, %(thread_id)s, %(created_at)s, %(updated_at)s, '1.0',
                    %(title)s, %(abstract)s, %(engram_json)s, %(engram_markdown)s, %(tags)s, %(keywords)s,
                    %(retrieval_text)s, 'local-deterministic-v1', %(embed)s::vector
                )
                """,
            {
                "engram_id": engram_id,
                "project_id": payload.project_id,
                "thread_id": payload.thread_id,
                "created_at": now,
                "updated_at": now,
                "title": payload.title,
                "abstract": payload.abstract,
                "engram_json": Jsonb(engram_json),
                "engram_markdown": payload.detailed_summary_markdown,
                "tags": payload.tags,
                "keywords": payload.keywords,
                "retrieval_text": retrieval_text,
                "embed": embedding_literal,
            },
        )

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

    return EngramCreateResponse(engram_id=engram_id, created_at=now)


def list_engrams(
    project_id: str | None = None,
    limit: int = 25,
    offset: int = 0,
) -> list[EngramSummary]:
    query = """
        SELECT
            engram_id, project_id, thread_id, title, abstract, created_at, tags, keywords
        FROM engrams
    """
    params: list = []
    if project_id:
        query += " WHERE project_id = %s"
        params.append(project_id)
    query += " ORDER BY created_at DESC LIMIT %s OFFSET %s"
    params.extend([limit, offset])

    with get_conn() as conn, conn.cursor() as cur:
        cur.execute(query, params)
        rows = cur.fetchall()

    return [EngramSummary(**row) for row in rows]


def query_engrams(request: EngramQueryRequest, embedding_dim: int) -> list[EngramQueryResult]:
    query_embedding = embed_text_local(request.query, embedding_dim)
    query_literal = _vector_literal(query_embedding)

    where_clauses: list[str] = []
    params: list = [query_literal]

    if request.project_id:
        where_clauses.append("project_id = %s")
        params.append(request.project_id)
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

    sql = f"""
        SELECT
            engram_id,
            project_id,
            title,
            abstract,
            created_at,
            tags,
            keywords,
            embed <=> %s::vector AS distance
        FROM engrams
        {where_sql}
        ORDER BY distance ASC, created_at DESC
        LIMIT %s
    """
    params.append(request.top_k)

    with get_conn() as conn, conn.cursor() as cur:
        cur.execute(sql, params)
        rows = cur.fetchall()

    return [EngramQueryResult(**row) for row in rows]


def get_rehydration_bundle(engram_id: UUID) -> RehydrationBundle | None:
    with get_conn() as conn, conn.cursor() as cur:
        cur.execute(
            """
                SELECT
                    engram_id, project_id, title, abstract, engram_json, engram_markdown
                FROM engrams
                WHERE engram_id = %s
                """,
            (engram_id,),
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
            LIMIT 10
            """,
            (engram_id,),
        )
        source_rows = cur.fetchall()

    engram_json = row["engram_json"] or {}
    decisions = engram_json.get("decisions", [])
    open_questions = engram_json.get("open_questions", [])
    citations = [RehydrationCitation(**source_row) for source_row in source_rows]

    compact_summary = row["abstract"]
    if len(compact_summary) > 800:
        compact_summary = compact_summary[:797] + "..."

    citation_lines = []
    for citation in citations[:5]:
        label = citation.title or citation.url
        citation_lines.append(f"- {label} ({citation.url})")

    citation_text = "\n".join(citation_lines) if citation_lines else "- No citations available"
    context_markdown = (
        f"# Rehydration Context: {row['title']}\n\n"
        f"## Compact Summary\n{compact_summary}\n\n"
        f"## Key Decisions\n"
        + (
            "\n".join(
                f"- {item.get('decision', '')}: {item.get('rationale', '')}" for item in decisions
            )
            or "- None"
        )
        + "\n\n## Open Questions\n"
        + ("\n".join(f"- {question}" for question in open_questions) or "- None")
        + "\n\n## Top Citations\n"
        + citation_text
    )

    return RehydrationBundle(
        engram_id=row["engram_id"],
        project_id=row["project_id"],
        title=row["title"],
        compact_summary=compact_summary,
        key_decisions=decisions,
        open_questions=open_questions,
        top_citations=citations[:5],
        context_markdown=context_markdown,
    )


def get_engram_sources(engram_id: UUID, limit: int = 100) -> list[EngramSourceRecord]:
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
