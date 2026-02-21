from __future__ import annotations

from uuid import UUID

from app.db import get_conn
from app.models import AdminChatSessionRecord

from .repository_common import (
    _ADMIN_SESSION_COLUMNS,
    _restore_record,
    _soft_delete_record,
    _SoftDeleteRecordRequest,
)
from .repository_types import AdminSessionListRepositoryRequest


def list_admin_sessions(
    *,
    request: AdminSessionListRepositoryRequest,
) -> list[AdminChatSessionRecord]:
    where_clauses = ["1=1"]
    params: list[object] = []
    if not request.include_deleted:
        where_clauses.append("deleted_at IS NULL")
    if request.project_id:
        where_clauses.append("project_id = %s")
        params.append(request.project_id)
    if request.owner_user_id:
        where_clauses.append("owner_user_id = %s")
        params.append(request.owner_user_id)

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
    params.extend([request.limit, request.offset])

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


def soft_delete_session(
    *,
    session_id: UUID,
    deleted_by_user_id: UUID,
    reason: str | None,
) -> AdminChatSessionRecord | None:
    row = _soft_delete_record(
        request=_SoftDeleteRecordRequest(
            table="chat_sessions",
            id_column="session_id",
            id_value=session_id,
            deleted_by_user_id=deleted_by_user_id,
            reason=reason,
            returning_columns=_ADMIN_SESSION_COLUMNS,
        ),
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
