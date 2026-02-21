from __future__ import annotations

import uuid
from uuid import UUID

from app.db import get_conn
from app.models import EngramCollectionRecord

from .repository_common import _soft_delete_record_exists, _SoftDeleteRecordRequest
from .repository_types import CollectionListRepositoryRequest


def list_collections(
    *,
    request: CollectionListRepositoryRequest,
) -> list[EngramCollectionRecord]:
    where_clauses = ["1=1"]
    params: list[object] = []
    if request.project_id:
        where_clauses.append("project_id = %s")
        params.append(request.project_id)
    if request.owner_user_id:
        where_clauses.append("owner_user_id = %s")
        params.append(request.owner_user_id)
    if not request.include_deleted:
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
    params.extend([request.limit, request.offset])
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
    return _soft_delete_record_exists(
        request=_SoftDeleteRecordRequest(
            table="engram_collections",
            id_column="collection_id",
            id_value=collection_id,
            deleted_by_user_id=deleted_by_user_id,
            reason=reason,
            returning_columns="collection_id",
        )
    )


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
