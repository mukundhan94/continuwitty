from __future__ import annotations

import uuid
from datetime import UTC, datetime
from uuid import UUID

from psycopg.types.json import Jsonb

from app.db import get_conn
from app.embeddings import embed_text
from app.models import (
    AdminEngramRecord,
    AdminEngramSourceInput,
    AdminEngramSourceRecord,
)

from .repository_common import (
    _restore_record,
    _soft_delete_record_exists,
    _SoftDeleteRecordRequest,
)
from .repository_types import (
    AdminEngramListRepositoryRequest,
    AdminEngramUpdateRepositoryRequest,
    _AdminEngramUpdatePayload,
    _EngramPersistPayload,
    _EngramUpdateFields,
)


def list_admin_engrams(
    *,
    request: AdminEngramListRepositoryRequest,
) -> list[AdminEngramRecord]:
    where_clauses = ["1=1"]
    params: list[object] = []
    if not request.include_deleted:
        where_clauses.append("e.deleted_at IS NULL")
    if request.project_id:
        where_clauses.append("e.project_id = %s")
        params.append(request.project_id)
    if request.session_id:
        where_clauses.append("e.source_session_id = %s")
        params.append(request.session_id)
    if request.owner_user_id:
        where_clauses.append("e.owner_user_id = %s")
        params.append(request.owner_user_id)
    if request.query_text:
        where_clauses.append(
            "(e.title ILIKE %s OR e.abstract ILIKE %s OR e.engram_markdown ILIKE %s)"
        )
        like = f"%{request.query_text}%"
        params.extend([like, like, like])

    sql = f"""
        SELECT
            e.engram_id,
            e.project_id,
            e.thread_id,
            e.title,
            e.abstract,
            e.engram_markdown AS detailed_summary_markdown,
            e.tags,
            e.keywords,
            e.owner_user_id,
            e.visibility_scope,
            e.source_session_id,
            e.created_at,
            e.updated_at,
            e.deleted_at,
            e.deleted_by_user_id,
            e.delete_reason
        FROM engrams e
        WHERE {" AND ".join(where_clauses)}
        ORDER BY e.created_at DESC
        LIMIT %s OFFSET %s
    """
    params.extend([request.limit, request.offset])
    with get_conn() as conn, conn.cursor() as cur:
        cur.execute(sql, params)
        rows = cur.fetchall()
    return [
        AdminEngramRecord(
            **row,
            sources=[],
        )
        for row in rows
    ]


def list_engram_sources(*, engram_id: UUID) -> list[AdminEngramSourceRecord]:
    with get_conn() as conn, conn.cursor() as cur:
        cur.execute(
            """
            SELECT
                source_id,
                captured_at,
                url,
                title,
                snippet,
                content_text,
                content_hash
            FROM sources
            WHERE engram_id = %s
            ORDER BY captured_at DESC
            """,
            (engram_id,),
        )
        rows = cur.fetchall()
    return [AdminEngramSourceRecord(**row) for row in rows]


def get_admin_engram(*, engram_id: UUID, include_deleted: bool = True) -> AdminEngramRecord | None:
    where_deleted = "" if include_deleted else "AND e.deleted_at IS NULL"
    with get_conn() as conn, conn.cursor() as cur:
        cur.execute(
            f"""
            SELECT
                e.engram_id,
                e.project_id,
                e.thread_id,
                e.title,
                e.abstract,
                e.engram_markdown AS detailed_summary_markdown,
                e.tags,
                e.keywords,
                e.owner_user_id,
                e.visibility_scope,
                e.source_session_id,
                e.created_at,
                e.updated_at,
                e.deleted_at,
                e.deleted_by_user_id,
                e.delete_reason
            FROM engrams e
            WHERE e.engram_id = %s
              {where_deleted}
            LIMIT 1
            """,
            (engram_id,),
        )
        row = cur.fetchone()
    if not row:
        return None
    return AdminEngramRecord(**row, sources=list_engram_sources(engram_id=engram_id))


def _build_retrieval_text(
    *,
    fields: _EngramUpdateFields,
) -> str:
    return " ".join(
        part
        for part in [
            fields["title"].strip(),
            fields["abstract"].strip(),
            fields["detailed_summary_markdown"].strip(),
            " ".join(fields["tags"]),
            " ".join(fields["keywords"]),
        ]
        if part
    )


def _build_engram_update_fields(
    *,
    current: AdminEngramRecord,
    payload: _AdminEngramUpdatePayload,
) -> _EngramUpdateFields:
    return {
        "title": payload.title.strip() if payload.title is not None else current.title,
        "abstract": payload.abstract.strip() if payload.abstract is not None else current.abstract,
        "detailed_summary_markdown": (
            payload.detailed_summary_markdown
            if payload.detailed_summary_markdown is not None
            else current.detailed_summary_markdown
        ),
        "tags": payload.tags if payload.tags is not None else current.tags,
        "keywords": payload.keywords if payload.keywords is not None else current.keywords,
        "visibility_scope": (
            payload.visibility_scope.value
            if payload.visibility_scope is not None
            else current.visibility_scope
        ),
    }


def _compute_engram_embedding(*, retrieval_text: str, embedding_dim: int) -> tuple[str, str]:
    embedding = embed_text(retrieval_text, dim=embedding_dim)
    embedding_literal = "[" + ",".join(f"{value:.6f}" for value in embedding.vector) + "]"
    return embedding.provider_id, embedding_literal


def _replace_engram_sources(*, cur, engram_id: UUID, sources: list[AdminEngramSourceInput]) -> None:
    cur.execute("DELETE FROM sources WHERE engram_id = %s", (engram_id,))
    for source in sources:
        cur.execute(
            """
            INSERT INTO sources (
                source_id,
                engram_id,
                captured_at,
                url,
                title,
                snippet,
                content_text,
                content_hash
            )
            VALUES (%s, %s, %s, %s, %s, %s, %s, %s)
            """,
            (
                uuid.uuid4(),
                engram_id,
                source.captured_at,
                source.url,
                source.title,
                source.snippet,
                source.content_text,
                source.content_hash,
            ),
        )


def _build_engram_json_payload(
    *,
    current: AdminEngramRecord,
    update_fields: _EngramUpdateFields,
) -> dict[str, object]:
    engram_json = dict(current.model_dump(mode="json"))
    engram_json["title"] = update_fields["title"]
    engram_json["abstract"] = update_fields["abstract"]
    engram_json["detailed_summary_markdown"] = update_fields["detailed_summary_markdown"]
    engram_json["tags"] = update_fields["tags"]
    engram_json["keywords"] = update_fields["keywords"]
    engram_json["visibility_scope"] = update_fields["visibility_scope"]
    engram_json["updated_at"] = datetime.now(UTC).isoformat()
    return engram_json


def _persist_engram_update(*, cur, payload: _EngramPersistPayload) -> bool:
    cur.execute(
        """
        UPDATE engrams
        SET
            title = %s,
            abstract = %s,
            engram_markdown = %s,
            tags = %s,
            keywords = %s,
            visibility_scope = %s,
            retrieval_text = %s,
            embedding_model = %s,
            embed = %s::vector,
            engram_json = %s,
            updated_at = now(),
            updated_by_user_id = %s
        WHERE
            engram_id = %s
            AND deleted_at IS NULL
        RETURNING engram_id
        """,
        (
            payload.update_fields["title"],
            payload.update_fields["abstract"],
            payload.update_fields["detailed_summary_markdown"],
            payload.update_fields["tags"],
            payload.update_fields["keywords"],
            payload.update_fields["visibility_scope"],
            payload.retrieval_text,
            payload.embedding_provider_id,
            payload.embedding_literal,
            Jsonb(payload.engram_json),
            payload.actor_user_id,
            payload.engram_id,
        ),
    )
    return bool(cur.fetchone())


def update_admin_engram(
    *,
    engram_id: UUID,
    request: AdminEngramUpdateRepositoryRequest,
) -> AdminEngramRecord | None:
    current = get_admin_engram(engram_id=engram_id, include_deleted=False)
    if not current:
        return None

    update_payload = _AdminEngramUpdatePayload(
        title=request.title,
        abstract=request.abstract,
        detailed_summary_markdown=request.detailed_summary_markdown,
        tags=request.tags,
        keywords=request.keywords,
        visibility_scope=request.visibility_scope,
    )
    update_fields = _build_engram_update_fields(current=current, payload=update_payload)

    retrieval_text = _build_retrieval_text(fields=update_fields)
    embedding_provider_id, embedding_literal = _compute_engram_embedding(
        retrieval_text=retrieval_text,
        embedding_dim=request.embedding_dim,
    )
    engram_json = _build_engram_json_payload(current=current, update_fields=update_fields)

    with get_conn() as conn, conn.cursor() as cur:
        updated = _persist_engram_update(
            cur=cur,
            payload=_EngramPersistPayload(
                engram_id=engram_id,
                actor_user_id=request.actor_user_id,
                update_fields=update_fields,
                retrieval_text=retrieval_text,
                embedding_provider_id=embedding_provider_id,
                embedding_literal=embedding_literal,
                engram_json=engram_json,
            ),
        )
        if not updated:
            return None

        if request.sources is not None:
            _replace_engram_sources(cur=cur, engram_id=engram_id, sources=request.sources)

    return get_admin_engram(engram_id=engram_id, include_deleted=False)


def move_admin_engram_project(
    *,
    engram_id: UUID,
    target_project_id: str,
    actor_user_id: UUID,
) -> AdminEngramRecord | None:
    with get_conn() as conn, conn.cursor() as cur:
        cur.execute(
            """
            UPDATE engrams
            SET
                project_id = %s,
                updated_at = now(),
                updated_by_user_id = %s
            WHERE
                engram_id = %s
                AND deleted_at IS NULL
            RETURNING engram_id
            """,
            (target_project_id, actor_user_id, engram_id),
        )
        updated = cur.fetchone()
        if not updated:
            return None
        # Enforce project-bounded collections by detaching moved engrams from
        # collections that belong to a different project.
        cur.execute(
            """
            DELETE FROM engram_collection_items item
            USING engram_collections collection, engrams e
            WHERE
                item.collection_id = collection.collection_id
                AND item.engram_id = e.engram_id
                AND item.engram_id = %s
                AND collection.project_id <> e.project_id
            """,
            (engram_id,),
        )
    return get_admin_engram(engram_id=engram_id, include_deleted=False)


def soft_delete_engram(*, engram_id: UUID, deleted_by_user_id: UUID, reason: str | None) -> bool:
    return _soft_delete_record_exists(
        request=_SoftDeleteRecordRequest(
            table="engrams",
            id_column="engram_id",
            id_value=engram_id,
            deleted_by_user_id=deleted_by_user_id,
            reason=reason,
            updated_by_column="updated_by_user_id",
            returning_columns="engram_id",
        )
    )


def restore_engram(*, engram_id: UUID) -> bool:
    with get_conn() as conn:
        return _restore_record(
            conn=conn,
            table="engrams",
            id_column="engram_id",
            id_value=engram_id,
        )
