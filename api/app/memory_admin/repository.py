from __future__ import annotations

import uuid
from dataclasses import dataclass
from datetime import UTC, datetime
from typing import TypedDict
from uuid import UUID

from psycopg.types.json import Jsonb

from app.db import get_conn
from app.embeddings import embed_text
from app.models import (
    AdminChatSessionRecord,
    AdminEngramRecord,
    AdminEngramSourceInput,
    AdminEngramSourceRecord,
    EngramCollectionRecord,
    VisibilityScope,
)

_ADMIN_SESSION_COLUMNS = """
                session_id,
                owner_user_id,
                project_id,
                title,
                provider,
                model_id,
                system_prompt,
                visibility_scope,
                autosave_enabled,
                autosave_strategy,
                autosave_interval_minutes,
                autosave_min_messages,
                retention_days,
                retention_max_snapshots,
                created_at,
                updated_at,
                deleted_at,
                deleted_by_user_id,
                delete_reason
"""


@dataclass(frozen=True)
class _AdminEngramUpdatePayload:
    title: str | None
    abstract: str | None
    detailed_summary_markdown: str | None
    tags: list[str] | None
    keywords: list[str] | None
    visibility_scope: VisibilityScope | None


class _EngramUpdateFields(TypedDict):
    title: str
    abstract: str
    detailed_summary_markdown: str
    tags: list[str]
    keywords: list[str]
    visibility_scope: str


@dataclass(frozen=True)
class AdminEngramUpdateRepositoryRequest:
    actor_user_id: UUID
    title: str | None
    abstract: str | None
    detailed_summary_markdown: str | None
    tags: list[str] | None
    keywords: list[str] | None
    visibility_scope: VisibilityScope | None
    sources: list[AdminEngramSourceInput] | None
    embedding_dim: int


@dataclass(frozen=True)
class _EngramPersistPayload:
    engram_id: UUID
    actor_user_id: UUID
    update_fields: _EngramUpdateFields
    retrieval_text: str
    embedding_provider_id: str
    embedding_literal: str
    engram_json: dict[str, object]


def list_admin_sessions(
    *,
    project_id: str | None = None,
    owner_user_id: UUID | None = None,
    include_deleted: bool = False,
    limit: int = 200,
    offset: int = 0,
) -> list[AdminChatSessionRecord]:
    where_clauses = ["1=1"]
    params: list[object] = []
    if not include_deleted:
        where_clauses.append("deleted_at IS NULL")
    if project_id:
        where_clauses.append("project_id = %s")
        params.append(project_id)
    if owner_user_id:
        where_clauses.append("owner_user_id = %s")
        params.append(owner_user_id)

    sql = f"""
        SELECT
            session_id,
            owner_user_id,
            project_id,
            title,
            provider,
            model_id,
            system_prompt,
            visibility_scope,
            autosave_enabled,
            autosave_strategy,
            autosave_interval_minutes,
            autosave_min_messages,
            retention_days,
            retention_max_snapshots,
            created_at,
            updated_at,
            deleted_at,
            deleted_by_user_id,
            delete_reason
        FROM chat_sessions
        WHERE {" AND ".join(where_clauses)}
        ORDER BY created_at DESC
        LIMIT %s OFFSET %s
    """
    params.extend([limit, offset])

    with get_conn() as conn, conn.cursor() as cur:
        cur.execute(sql, params)
        rows = cur.fetchall()
    return [AdminChatSessionRecord(**row) for row in rows]


def get_admin_session(
    *, session_id: UUID, include_deleted: bool = True
) -> AdminChatSessionRecord | None:
    where_deleted = "" if include_deleted else "AND deleted_at IS NULL"
    with get_conn() as conn, conn.cursor() as cur:
        cur.execute(
            f"""
            SELECT
                session_id,
                owner_user_id,
                project_id,
                title,
                provider,
                model_id,
                system_prompt,
                visibility_scope,
                autosave_enabled,
                autosave_strategy,
                autosave_interval_minutes,
                autosave_min_messages,
                retention_days,
                retention_max_snapshots,
                created_at,
                updated_at,
                deleted_at,
                deleted_by_user_id,
                delete_reason
            FROM chat_sessions
            WHERE session_id = %s
              {where_deleted}
            LIMIT 1
            """,
            (session_id,),
        )
        row = cur.fetchone()
    return AdminChatSessionRecord(**row) if row else None


def _soft_delete_record(
    *,
    conn,
    table: str,
    id_column: str,
    id_value: UUID,
    deleted_by_user_id: UUID,
    reason: str | None,
    returning_columns: str,
    updated_by_column: str | None = None,
):
    assignments = [
        "deleted_at = COALESCE(deleted_at, now())",
        "deleted_by_user_id = %s",
        "delete_reason = %s",
        "updated_at = now()",
    ]
    params: list[object] = [deleted_by_user_id, reason]
    if updated_by_column is not None:
        assignments.append(f"{updated_by_column} = %s")
        params.append(deleted_by_user_id)
    params.append(id_value)

    with conn.cursor() as cur:
        cur.execute(
            f"""
            UPDATE {table}
            SET
                {", ".join(assignments)}
            WHERE {id_column} = %s
            RETURNING {returning_columns}
            """,
            tuple(params),
        )
        return cur.fetchone()


def _restore_record(*, conn, table: str, id_column: str, id_value: UUID) -> bool:
    with conn.cursor() as cur:
        cur.execute(
            f"""
            UPDATE {table}
            SET
                deleted_at = NULL,
                deleted_by_user_id = NULL,
                delete_reason = NULL,
                updated_at = now()
            WHERE {id_column} = %s
            RETURNING {id_column}
            """,
            (id_value,),
        )
        row = cur.fetchone()
    return bool(row)


def soft_delete_session(
    *,
    session_id: UUID,
    deleted_by_user_id: UUID,
    reason: str | None,
) -> AdminChatSessionRecord | None:
    with get_conn() as conn:
        row = _soft_delete_record(
            conn=conn,
            table="chat_sessions",
            id_column="session_id",
            id_value=session_id,
            deleted_by_user_id=deleted_by_user_id,
            reason=reason,
            returning_columns=_ADMIN_SESSION_COLUMNS,
        )
    return AdminChatSessionRecord(**row) if row else None


def restore_session(*, session_id: UUID) -> bool:
    with get_conn() as conn:
        return _restore_record(
            conn=conn,
            table="chat_sessions",
            id_column="session_id",
            id_value=session_id,
        )


def soft_delete_linked_engrams(
    *,
    session_id: UUID,
    deleted_by_user_id: UUID,
    reason: str | None,
) -> int:
    with get_conn() as conn, conn.cursor() as cur:
        cur.execute(
            """
            UPDATE engrams
            SET
                deleted_at = COALESCE(deleted_at, now()),
                deleted_by_user_id = %s,
                delete_reason = %s,
                updated_at = now(),
                updated_by_user_id = %s
            WHERE
                source_session_id = %s
                AND deleted_at IS NULL
            RETURNING engram_id
            """,
            (deleted_by_user_id, reason, deleted_by_user_id, session_id),
        )
        rows = cur.fetchall()
    return len(rows)


def list_admin_engrams(
    *,
    project_id: str | None = None,
    session_id: UUID | None = None,
    owner_user_id: UUID | None = None,
    query_text: str | None = None,
    include_deleted: bool = False,
    limit: int = 200,
    offset: int = 0,
) -> list[AdminEngramRecord]:
    where_clauses = ["1=1"]
    params: list[object] = []
    if not include_deleted:
        where_clauses.append("e.deleted_at IS NULL")
    if project_id:
        where_clauses.append("e.project_id = %s")
        params.append(project_id)
    if session_id:
        where_clauses.append("e.source_session_id = %s")
        params.append(session_id)
    if owner_user_id:
        where_clauses.append("e.owner_user_id = %s")
        params.append(owner_user_id)
    if query_text:
        where_clauses.append(
            "(e.title ILIKE %s OR e.abstract ILIKE %s OR e.engram_markdown ILIKE %s)"
        )
        like = f"%{query_text}%"
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
    params.extend([limit, offset])
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
    with get_conn() as conn:
        row = _soft_delete_record(
            conn=conn,
            table="engrams",
            id_column="engram_id",
            id_value=engram_id,
            deleted_by_user_id=deleted_by_user_id,
            reason=reason,
            updated_by_column="updated_by_user_id",
            returning_columns="engram_id",
        )
    return bool(row)


def restore_engram(*, engram_id: UUID) -> bool:
    with get_conn() as conn:
        return _restore_record(
            conn=conn,
            table="engrams",
            id_column="engram_id",
            id_value=engram_id,
        )


def list_collections(
    *,
    project_id: str | None = None,
    owner_user_id: UUID | None = None,
    include_deleted: bool = False,
    limit: int = 200,
    offset: int = 0,
) -> list[EngramCollectionRecord]:
    where_clauses = ["1=1"]
    params: list[object] = []
    if project_id:
        where_clauses.append("project_id = %s")
        params.append(project_id)
    if owner_user_id:
        where_clauses.append("owner_user_id = %s")
        params.append(owner_user_id)
    if not include_deleted:
        where_clauses.append("deleted_at IS NULL")

    sql = f"""
        SELECT
            collection_id,
            project_id,
            owner_user_id,
            name,
            description,
            created_at,
            updated_at,
            deleted_at,
            deleted_by_user_id,
            delete_reason
        FROM engram_collections
        WHERE {" AND ".join(where_clauses)}
        ORDER BY created_at DESC
        LIMIT %s OFFSET %s
    """
    params.extend([limit, offset])
    with get_conn() as conn, conn.cursor() as cur:
        cur.execute(sql, params)
        rows = cur.fetchall()
    return [EngramCollectionRecord(**row) for row in rows]


def get_collection(
    *, collection_id: UUID, include_deleted: bool = True
) -> EngramCollectionRecord | None:
    where_deleted = "" if include_deleted else "AND deleted_at IS NULL"
    with get_conn() as conn, conn.cursor() as cur:
        cur.execute(
            f"""
            SELECT
                collection_id,
                project_id,
                owner_user_id,
                name,
                description,
                created_at,
                updated_at,
                deleted_at,
                deleted_by_user_id,
                delete_reason
            FROM engram_collections
            WHERE collection_id = %s
              {where_deleted}
            LIMIT 1
            """,
            (collection_id,),
        )
        row = cur.fetchone()
    return EngramCollectionRecord(**row) if row else None


def create_collection(
    *,
    project_id: str,
    owner_user_id: UUID,
    name: str,
    description: str,
) -> EngramCollectionRecord:
    collection_id = uuid.uuid4()
    with get_conn() as conn, conn.cursor() as cur:
        cur.execute(
            """
            INSERT INTO engram_collections (
                collection_id,
                project_id,
                owner_user_id,
                name,
                description,
                created_at,
                updated_at
            )
            VALUES (%s, %s, %s, %s, %s, now(), now())
            RETURNING
                collection_id,
                project_id,
                owner_user_id,
                name,
                description,
                created_at,
                updated_at,
                deleted_at,
                deleted_by_user_id,
                delete_reason
            """,
            (collection_id, project_id, owner_user_id, name, description),
        )
        row = cur.fetchone()
    return EngramCollectionRecord(**row)


def update_collection(
    *,
    collection_id: UUID,
    name: str | None,
    description: str | None,
) -> EngramCollectionRecord | None:
    with get_conn() as conn, conn.cursor() as cur:
        cur.execute(
            """
            UPDATE engram_collections
            SET
                name = COALESCE(%s, name),
                description = COALESCE(%s, description),
                updated_at = now()
            WHERE
                collection_id = %s
                AND deleted_at IS NULL
            RETURNING
                collection_id,
                project_id,
                owner_user_id,
                name,
                description,
                created_at,
                updated_at,
                deleted_at,
                deleted_by_user_id,
                delete_reason
            """,
            (name, description, collection_id),
        )
        row = cur.fetchone()
    return EngramCollectionRecord(**row) if row else None


def soft_delete_collection(
    *,
    collection_id: UUID,
    deleted_by_user_id: UUID,
    reason: str | None,
) -> bool:
    with get_conn() as conn:
        row = _soft_delete_record(
            conn=conn,
            table="engram_collections",
            id_column="collection_id",
            id_value=collection_id,
            deleted_by_user_id=deleted_by_user_id,
            reason=reason,
            returning_columns="collection_id",
        )
    return bool(row)


def add_collection_items(
    *,
    collection_id: UUID,
    actor_user_id: UUID,
    engram_ids: list[UUID],
) -> int:
    if not engram_ids:
        return 0
    with get_conn() as conn, conn.cursor() as cur:
        cur.execute(
            """
            INSERT INTO engram_collection_items (
                collection_id,
                engram_id,
                added_by_user_id,
                created_at
            )
            SELECT
                c.collection_id,
                e.engram_id,
                %s,
                now()
            FROM engram_collections c
            JOIN engrams e
              ON e.engram_id = ANY(%s::UUID[])
             AND e.project_id = c.project_id
             AND e.deleted_at IS NULL
            WHERE
                c.collection_id = %s
                AND c.deleted_at IS NULL
            ON CONFLICT (collection_id, engram_id) DO NOTHING
            RETURNING engram_id
            """,
            (actor_user_id, engram_ids, collection_id),
        )
        rows = cur.fetchall()
    return len(rows)


def remove_collection_item(*, collection_id: UUID, engram_id: UUID) -> bool:
    with get_conn() as conn, conn.cursor() as cur:
        cur.execute(
            """
            DELETE FROM engram_collection_items
            WHERE collection_id = %s AND engram_id = %s
            RETURNING engram_id
            """,
            (collection_id, engram_id),
        )
        row = cur.fetchone()
    return bool(row)
