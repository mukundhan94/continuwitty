from __future__ import annotations

from datetime import UTC, datetime
from uuid import UUID

from app.db import get_conn
from app.models import ProjectRecord


def _project_from_row(row: dict) -> ProjectRecord:
    return ProjectRecord(
        project_id=row["project_id"],
        name=row["name"],
        description=row.get("description") or "",
        owner_user_id=row["owner_user_id"],
        is_archived=bool(row["is_archived"]),
        created_at=row["created_at"],
        updated_at=row["updated_at"],
    )


def list_projects_for_actor(
    *,
    actor_user_id: UUID,
    actor_role: str,
    include_archived: bool = False,
    limit: int = 500,
    offset: int = 0,
) -> list[ProjectRecord]:
    where = ["1=1"]
    params: list[object] = []
    if actor_role != "admin":
        where.append("owner_user_id = %s")
        params.append(actor_user_id)
    if not include_archived:
        where.append("is_archived = FALSE")

    sql = f"""
        SELECT
            project_id,
            name,
            description,
            owner_user_id,
            is_archived,
            created_at,
            updated_at
        FROM projects
        WHERE {" AND ".join(where)}
        ORDER BY created_at ASC
        LIMIT %s OFFSET %s
    """
    params.extend([limit, offset])
    with get_conn() as conn, conn.cursor() as cur:
        cur.execute(sql, params)
        rows = cur.fetchall()
    return [_project_from_row(row) for row in rows]


def get_project_for_actor(
    *,
    project_id: str,
    actor_user_id: UUID,
    actor_role: str,
    include_archived: bool = False,
) -> ProjectRecord | None:
    where = ["project_id = %s"]
    params: list[object] = [project_id]
    if actor_role != "admin":
        where.append("owner_user_id = %s")
        params.append(actor_user_id)
    if not include_archived:
        where.append("is_archived = FALSE")

    sql = f"""
        SELECT
            project_id,
            name,
            description,
            owner_user_id,
            is_archived,
            created_at,
            updated_at
        FROM projects
        WHERE {" AND ".join(where)}
        LIMIT 1
    """
    with get_conn() as conn, conn.cursor() as cur:
        cur.execute(sql, params)
        row = cur.fetchone()
    return _project_from_row(row) if row else None


def create_project(
    *,
    project_id: str,
    name: str,
    description: str,
    owner_user_id: UUID,
) -> ProjectRecord:
    now = datetime.now(UTC)
    with get_conn() as conn, conn.cursor() as cur:
        cur.execute(
            """
            INSERT INTO projects (
                project_id,
                name,
                description,
                owner_user_id,
                is_archived,
                created_at,
                updated_at
            )
            VALUES (%s, %s, %s, %s, FALSE, %s, %s)
            ON CONFLICT (project_id) DO UPDATE
              SET
                name = EXCLUDED.name,
                description = EXCLUDED.description,
                updated_at = EXCLUDED.updated_at
            RETURNING
                project_id,
                name,
                description,
                owner_user_id,
                is_archived,
                created_at,
                updated_at
            """,
            (project_id, name, description, owner_user_id, now, now),
        )
        row = cur.fetchone()
    return _project_from_row(row)


def ensure_project_exists(
    *,
    project_id: str,
    owner_user_id: UUID,
) -> ProjectRecord:
    existing = get_project_for_actor(
        project_id=project_id,
        actor_user_id=owner_user_id,
        actor_role="admin",
        include_archived=True,
    )
    if existing:
        return existing
    return create_project(
        project_id=project_id,
        name=project_id,
        description="Autocreated from workflow input.",
        owner_user_id=owner_user_id,
    )


def get_user_default_project_id(*, user_id: UUID) -> str | None:
    with get_conn() as conn, conn.cursor() as cur:
        cur.execute(
            """
            SELECT default_project_id
            FROM users
            WHERE user_id = %s
            LIMIT 1
            """,
            (user_id,),
        )
        row = cur.fetchone()
    if not row:
        return None
    raw = row.get("default_project_id")
    return str(raw) if raw else None


def set_user_default_project_id(*, user_id: UUID, project_id: str) -> bool:
    with get_conn() as conn, conn.cursor() as cur:
        cur.execute(
            """
            UPDATE users
            SET default_project_id = %s
            WHERE user_id = %s
            RETURNING user_id
            """,
            (project_id, user_id),
        )
        row = cur.fetchone()
    return bool(row)
