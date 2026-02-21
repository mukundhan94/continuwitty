from __future__ import annotations

from dataclasses import dataclass
from uuid import UUID

from app.db import get_conn

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
class _SoftDeleteRecordRequest:
    table: str
    id_column: str
    id_value: UUID
    deleted_by_user_id: UUID
    reason: str | None
    returning_columns: str
    updated_by_column: str | None = None


def _soft_delete_record(
    *,
    request: _SoftDeleteRecordRequest,
):
    assignments = [
        "deleted_at = COALESCE(deleted_at, now())",
        "deleted_by_user_id = %s",
        "delete_reason = %s",
        "updated_at = now()",
    ]
    params: list[object] = [request.deleted_by_user_id, request.reason]
    if request.updated_by_column is not None:
        assignments.append(f"{request.updated_by_column} = %s")
        params.append(request.deleted_by_user_id)
    params.append(request.id_value)

    with get_conn() as conn, conn.cursor() as cur:
        cur.execute(
            f"""
            UPDATE {request.table}
            SET
                {", ".join(assignments)}
            WHERE {request.id_column} = %s
            RETURNING {request.returning_columns}
            """,
            tuple(params),
        )
        return cur.fetchone()


def _soft_delete_record_exists(*, request: _SoftDeleteRecordRequest) -> bool:
    return bool(_soft_delete_record(request=request))


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
