from __future__ import annotations

from uuid import UUID, uuid4

from psycopg.errors import UniqueViolation

from .db import get_conn
from .models import UserRecord, UserRole


def get_user_auth_record(username: str) -> dict | None:
    with get_conn() as conn, conn.cursor() as cur:
        cur.execute(
            """
            SELECT user_id, username, password_hash, role, is_active, created_at
            FROM users
            WHERE username = %s
            """,
            (username,),
        )
        return cur.fetchone()


def get_user_auth_record_by_id(user_id: UUID) -> dict | None:
    with get_conn() as conn, conn.cursor() as cur:
        cur.execute(
            """
            SELECT user_id, username, password_hash, role, is_active, created_at
            FROM users
            WHERE user_id = %s
            """,
            (user_id,),
        )
        return cur.fetchone()


def list_users(limit: int = 200, offset: int = 0) -> list[UserRecord]:
    with get_conn() as conn, conn.cursor() as cur:
        cur.execute(
            """
            SELECT user_id, username, role, is_active, created_at
            FROM users
            ORDER BY created_at ASC
            LIMIT %s OFFSET %s
            """,
            (limit, offset),
        )
        rows = cur.fetchall()
    return [UserRecord(**row) for row in rows]


def create_user(username: str, password_hash: str, role: UserRole, is_active: bool) -> UserRecord:
    with get_conn() as conn, conn.cursor() as cur:
        try:
            cur.execute(
                """
                INSERT INTO users (user_id, username, password_hash, role, is_active)
                VALUES (%s, %s, %s, %s, %s)
                RETURNING user_id, username, role, is_active, created_at
                """,
                (uuid4(), username, password_hash, role.value, is_active),
            )
        except UniqueViolation as exc:
            raise ValueError("username already exists") from exc
        row = cur.fetchone()
    return UserRecord(**row)


def update_user(
    user_id: UUID,
    role: UserRole | None = None,
    is_active: bool | None = None,
    password_hash: str | None = None,
) -> UserRecord | None:
    with get_conn() as conn, conn.cursor() as cur:
        cur.execute(
            """
            UPDATE users
            SET
                role = COALESCE(%s, role),
                is_active = COALESCE(%s, is_active),
                password_hash = COALESCE(%s, password_hash)
            WHERE user_id = %s
            RETURNING user_id, username, role, is_active, created_at
            """,
            (role.value if role else None, is_active, password_hash, user_id),
        )
        row = cur.fetchone()
    if not row:
        return None
    return UserRecord(**row)
