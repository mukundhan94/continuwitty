from __future__ import annotations

import re
from dataclasses import dataclass
from datetime import UTC, datetime
from typing import Any
from uuid import UUID

from psycopg.types.json import Jsonb

from app.db import get_conn
from app.embeddings import embed_many, embed_text
from app.ingestion.chunking import ChunkDraft
from app.ingestion.models import DocumentChunkQueryRequest, DocumentChunkQueryResult, DocumentRecord

_TOKEN_PATTERN = re.compile(r"[a-z0-9]{2,}")
_UPSERT_DOCUMENT_SQL = """
INSERT INTO documents (
    document_id,
    owner_user_id,
    project_id,
    title,
    source_type,
    source_name,
    mime_type,
    visibility_scope,
    content_text,
    content_hash,
    metadata,
    chunk_count,
    created_at,
    updated_at
)
VALUES (
    %(document_id)s,
    %(owner_user_id)s,
    %(project_id)s,
    %(title)s,
    %(source_type)s,
    %(source_name)s,
    %(mime_type)s,
    %(visibility_scope)s,
    %(content_text)s,
    %(content_hash)s,
    %(metadata)s,
    %(chunk_count)s,
    %(created_at)s,
    %(updated_at)s
)
ON CONFLICT (document_id)
DO UPDATE SET
    title = EXCLUDED.title,
    source_type = EXCLUDED.source_type,
    source_name = EXCLUDED.source_name,
    mime_type = EXCLUDED.mime_type,
    visibility_scope = EXCLUDED.visibility_scope,
    content_text = EXCLUDED.content_text,
    content_hash = EXCLUDED.content_hash,
    metadata = EXCLUDED.metadata,
    chunk_count = EXCLUDED.chunk_count,
    updated_at = EXCLUDED.updated_at
RETURNING
    document_id,
    owner_user_id,
    project_id,
    title,
    source_type,
    source_name,
    mime_type,
    visibility_scope,
    content_hash,
    chunk_count,
    created_at,
    updated_at
"""
_INSERT_DOCUMENT_CHUNK_SQL = """
INSERT INTO document_chunks (
    chunk_id,
    document_id,
    chunk_index,
    chunk_text,
    snippet,
    char_start,
    char_end,
    token_estimate,
    metadata,
    embedding_model,
    embed,
    created_at
) VALUES (
    %(chunk_id)s,
    %(document_id)s,
    %(chunk_index)s,
    %(chunk_text)s,
    %(snippet)s,
    %(char_start)s,
    %(char_end)s,
    %(token_estimate)s,
    %(metadata)s,
    %(embedding_model)s,
    %(embed)s::vector,
    %(created_at)s
)
"""
_QUERY_DOCUMENT_CHUNKS_SELECT_TEMPLATE = """
SELECT
    dc.chunk_id,
    dc.document_id,
    d.project_id,
    d.title,
    d.source_name,
    dc.chunk_index,
    dc.snippet,
    dc.created_at,
    d.visibility_scope,
    dc.chunk_text,
    dc.embed <=> %s::vector AS distance
FROM document_chunks dc
JOIN documents d ON d.document_id = dc.document_id
WHERE {where_sql}
ORDER BY distance ASC, d.created_at DESC, dc.chunk_index ASC
LIMIT %s
"""


@dataclass(frozen=True)
class DocumentUpsertPayload:
    document_id: UUID
    project_id: str
    title: str
    source_type: str
    source_name: str | None
    mime_type: str | None
    visibility_scope: str
    content_text: str
    content_hash: str
    metadata: dict
    chunks: list[ChunkDraft]


def _vector_literal(values: list[float]) -> str:
    return "[" + ",".join(f"{value:.6f}" for value in values) + "]"


def _tokenize(text: str) -> set[str]:
    return {match.group(0) for match in _TOKEN_PATTERN.finditer(text.lower())}


def _combined_rank_score(distance: float, lexical_overlap: float) -> float:
    dense_score = 1.0 / (1.0 + max(distance, 0.0))
    return (dense_score * 0.8) + (lexical_overlap * 0.2)


def _lexical_overlap_score(query: str, candidate_parts: list[str]) -> float:
    query_tokens = _tokenize(query)
    if not query_tokens:
        return 0.0

    chunk_tokens: set[str] = set()
    for part in candidate_parts:
        chunk_tokens.update(_tokenize(part))

    if not chunk_tokens:
        return 0.0
    return len(query_tokens & chunk_tokens) / len(query_tokens)


def _upsert_document_record(
    *,
    cur: Any,
    actor_user_id: UUID,
    payload: DocumentUpsertPayload,
    now: datetime,
) -> dict[str, Any]:
    cur.execute(
        _UPSERT_DOCUMENT_SQL,
        {
            "document_id": payload.document_id,
            "owner_user_id": actor_user_id,
            "project_id": payload.project_id,
            "title": payload.title,
            "source_type": payload.source_type,
            "source_name": payload.source_name,
            "mime_type": payload.mime_type,
            "visibility_scope": payload.visibility_scope,
            "content_text": payload.content_text,
            "content_hash": payload.content_hash,
            "metadata": Jsonb(payload.metadata),
            "chunk_count": len(payload.chunks),
            "created_at": now,
            "updated_at": now,
        },
    )
    return cur.fetchone()


def _replace_document_chunks(
    *,
    cur: Any,
    payload: DocumentUpsertPayload,
    now: datetime,
    chunk_embeddings: list[Any],
) -> None:
    cur.execute("DELETE FROM document_chunks WHERE document_id = %s", (payload.document_id,))
    for chunk, embedded in zip(payload.chunks, chunk_embeddings, strict=True):
        cur.execute(
            _INSERT_DOCUMENT_CHUNK_SQL,
            {
                "chunk_id": chunk.chunk_id,
                "document_id": payload.document_id,
                "chunk_index": chunk.chunk_index,
                "chunk_text": chunk.chunk_text,
                "snippet": chunk.snippet,
                "char_start": chunk.char_start,
                "char_end": chunk.char_end,
                "token_estimate": chunk.token_estimate,
                "metadata": Jsonb(chunk.metadata),
                "embedding_model": embedded.provider_id,
                "embed": _vector_literal(embedded.vector),
                "created_at": now,
            },
        )


def _build_document_chunk_where(
    *,
    actor_user_id: UUID,
    request: DocumentChunkQueryRequest,
) -> tuple[str, list[Any]]:
    where_clauses: list[str] = ["(d.owner_user_id = %s OR d.visibility_scope = 'project')"]
    where_params: list[Any] = [actor_user_id]

    if request.project_id:
        where_clauses.append("d.project_id = %s")
        where_params.append(request.project_id)
    if request.document_ids:
        # Limit retrieval to a caller-selected document subset (for session-pinned docs).
        where_clauses.append("d.document_id = ANY(%s::uuid[])")
        where_params.append(request.document_ids)

    return " AND ".join(where_clauses), where_params


def _rerank_document_chunk_rows(
    *,
    rows: list[dict[str, Any]],
    query: str,
    top_k: int,
) -> list[dict[str, Any]]:
    ranked: list[tuple[float, dict[str, Any]]] = []
    for row in rows:
        lexical = _lexical_overlap_score(
            query,
            [row.get("title") or "", row.get("snippet") or "", row.get("chunk_text") or ""],
        )
        ranked.append((_combined_rank_score(float(row["distance"]), lexical), row))

    ranked.sort(key=lambda item: item[0], reverse=True)
    return [row for _, row in ranked[:top_k]]


def upsert_document_with_chunks(
    *,
    actor_user_id: UUID,
    payload: DocumentUpsertPayload,
    embedding_dim: int,
) -> DocumentRecord:
    """Persist document metadata and atomically replace all associated chunks."""

    now = datetime.now(UTC)
    chunk_embeddings = embed_many([item.chunk_text for item in payload.chunks], dim=embedding_dim)

    with get_conn() as conn, conn.cursor() as cur:
        row = _upsert_document_record(
            cur=cur,
            actor_user_id=actor_user_id,
            payload=payload,
            now=now,
        )
        _replace_document_chunks(
            cur=cur,
            payload=payload,
            now=now,
            chunk_embeddings=chunk_embeddings,
        )

    return DocumentRecord(**row)


def list_documents(
    *,
    actor_user_id: UUID,
    project_id: str | None,
    limit: int,
    offset: int,
) -> list[DocumentRecord]:
    query = """
        SELECT
            document_id,
            owner_user_id,
            project_id,
            title,
            source_type,
            source_name,
            mime_type,
            visibility_scope,
            content_hash,
            chunk_count,
            created_at,
            updated_at
        FROM documents
        WHERE (owner_user_id = %s OR visibility_scope = 'project')
    """
    params: list = [actor_user_id]

    if project_id:
        query += " AND project_id = %s"
        params.append(project_id)

    query += " ORDER BY created_at DESC LIMIT %s OFFSET %s"
    params.extend([limit, offset])

    with get_conn() as conn, conn.cursor() as cur:
        cur.execute(query, params)
        rows = cur.fetchall()
    return [DocumentRecord(**row) for row in rows]


def query_document_chunks(
    *,
    actor_user_id: UUID,
    request: DocumentChunkQueryRequest,
    embedding_dim: int,
) -> list[DocumentChunkQueryResult]:
    query_embedding = embed_text(request.query, dim=embedding_dim)
    query_literal = _vector_literal(query_embedding.vector)
    where_sql, where_params = _build_document_chunk_where(
        actor_user_id=actor_user_id,
        request=request,
    )
    candidate_limit = min(max(request.top_k * 4, request.top_k), 200)

    with get_conn() as conn, conn.cursor() as cur:
        cur.execute(
            _QUERY_DOCUMENT_CHUNKS_SELECT_TEMPLATE.format(where_sql=where_sql),
            [query_literal, *where_params, candidate_limit],
        )
        rows = cur.fetchall()

    trimmed = _rerank_document_chunk_rows(
        rows=rows,
        query=request.query,
        top_k=request.top_k,
    )

    return [
        DocumentChunkQueryResult(
            chunk_id=row["chunk_id"],
            document_id=row["document_id"],
            project_id=row["project_id"],
            title=row["title"],
            source_name=row.get("source_name"),
            chunk_index=row["chunk_index"],
            snippet=row.get("snippet") or "",
            created_at=row["created_at"],
            visibility_scope=row["visibility_scope"],
            distance=row["distance"],
        )
        for row in trimmed
    ]
