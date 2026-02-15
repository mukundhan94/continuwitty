from __future__ import annotations

from uuid import UUID, uuid4

from psycopg.errors import UniqueViolation

from .auth import hash_password
from .config import get_settings
from .db import get_conn
from .models import UserRecord, UserRole

USERS_TABLE_SQL = """
CREATE TABLE IF NOT EXISTS users (
    user_id UUID PRIMARY KEY,
    username TEXT NOT NULL UNIQUE,
    password_hash TEXT NOT NULL,
    role TEXT NOT NULL CHECK (role IN ('admin', 'analyst', 'viewer')),
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS users_role_idx ON users (role);
CREATE INDEX IF NOT EXISTS users_active_idx ON users (is_active);
"""


def ensure_user_store() -> None:
    settings = get_settings()
    password_hash = settings.ui_demo_password_hash or hash_password(settings.ui_demo_password)
    with get_conn() as conn, conn.cursor() as cur:
        cur.execute(USERS_TABLE_SQL)
        cur.execute(
            """
            INSERT INTO users (user_id, username, password_hash, role, is_active)
            VALUES (%s, %s, %s, %s, %s)
            ON CONFLICT (username) DO NOTHING
            """,
            (
                uuid4(),
                settings.ui_demo_username,
                password_hash,
                UserRole.admin.value,
                True,
            ),
        )


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
