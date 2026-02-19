from __future__ import annotations

from datetime import UTC, datetime
from uuid import UUID, uuid4

from ..db import get_conn
from .models import McpTokenRecord


def _row_to_record(row: dict) -> McpTokenRecord:
    return McpTokenRecord(
        token_id=row["token_id"],
        owner_user_id=row["owner_user_id"],
        name=row["name"],
        scope=row["scope"],
        allowed_tools=row["allowed_tools"] or [],
        allowed_project_ids=row["allowed_project_ids"] or [],
        token_secret_hash=row["token_secret_hash"],
        token_secret_hint=row["token_secret_hint"],
        expires_at=row["expires_at"],
        last_used_at=row["last_used_at"],
        revoked_at=row["revoked_at"],
        created_at=row["created_at"],
    )


def create_mcp_token(
    *,
    token_id: UUID | None = None,
    owner_user_id: UUID,
    name: str,
    scope: str,
    allowed_tools: list[str],
    allowed_project_ids: list[str],
    token_secret_hash: str,
    token_secret_hint: str,
    expires_at: datetime,
) -> McpTokenRecord:
    with get_conn() as conn, conn.cursor() as cur:
        # Token IDs are generated application-side so the plaintext token can embed the UUID.
        created_token_id = token_id or uuid4()
        cur.execute(
            """
            INSERT INTO mcp_tokens (
                token_id,
                owner_user_id,
                name,
                scope,
                allowed_tools,
                allowed_project_ids,
                token_secret_hash,
                token_secret_hint,
                expires_at
            )
            VALUES (%s, %s, %s, %s, %s, %s, %s, %s, %s)
            RETURNING
                token_id,
                owner_user_id,
                name,
                scope,
                allowed_tools,
                allowed_project_ids,
                token_secret_hash,
                token_secret_hint,
                expires_at,
                last_used_at,
                revoked_at,
                created_at
            """,
            (
                created_token_id,
                owner_user_id,
                name,
                scope,
                allowed_tools,
                allowed_project_ids,
                token_secret_hash,
                token_secret_hint,
                expires_at,
            ),
        )
        row = cur.fetchone()
    return _row_to_record(row)


def list_mcp_tokens(
    *,
    owner_user_id: UUID,
    limit: int = 200,
    offset: int = 0,
) -> list[McpTokenRecord]:
    with get_conn() as conn, conn.cursor() as cur:
        cur.execute(
            """
            SELECT
                token_id,
                owner_user_id,
                name,
                scope,
                allowed_tools,
                allowed_project_ids,
                token_secret_hash,
                token_secret_hint,
                expires_at,
                last_used_at,
                revoked_at,
                created_at
            FROM mcp_tokens
            WHERE owner_user_id = %s
            ORDER BY created_at DESC
            LIMIT %s OFFSET %s
            """,
            (owner_user_id, limit, offset),
        )
        rows = cur.fetchall()
    return [_row_to_record(row) for row in rows]


def get_mcp_token_by_id(*, token_id: UUID) -> McpTokenRecord | None:
    with get_conn() as conn, conn.cursor() as cur:
        cur.execute(
            """
            SELECT
                token_id,
                owner_user_id,
                name,
                scope,
                allowed_tools,
                allowed_project_ids,
                token_secret_hash,
                token_secret_hint,
                expires_at,
                last_used_at,
                revoked_at,
                created_at
            FROM mcp_tokens
            WHERE token_id = %s
            """,
            (token_id,),
        )
        row = cur.fetchone()
    if not row:
        return None
    return _row_to_record(row)


def revoke_mcp_token(*, token_id: UUID, owner_user_id: UUID) -> McpTokenRecord | None:
    with get_conn() as conn, conn.cursor() as cur:
        cur.execute(
            """
            UPDATE mcp_tokens
            SET revoked_at = COALESCE(revoked_at, now())
            WHERE token_id = %s
              AND owner_user_id = %s
            RETURNING
                token_id,
                owner_user_id,
                name,
                scope,
                allowed_tools,
                allowed_project_ids,
                token_secret_hash,
                token_secret_hint,
                expires_at,
                last_used_at,
                revoked_at,
                created_at
            """,
            (token_id, owner_user_id),
        )
        row = cur.fetchone()
    if not row:
        return None
    return _row_to_record(row)


def touch_mcp_token_last_used(*, token_id: UUID) -> None:
    with get_conn() as conn, conn.cursor() as cur:
        cur.execute(
            """
            UPDATE mcp_tokens
            SET last_used_at = %s
            WHERE token_id = %s
            """,
            (datetime.now(UTC), token_id),
        )
