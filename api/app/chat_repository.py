from __future__ import annotations

from datetime import UTC, datetime
from uuid import UUID, uuid4

from psycopg.types.json import Jsonb

from .db import get_conn
from .models import (
    ChatMessageRecord,
    ChatSessionCreateRequest,
    ChatSessionRecord,
    ChatSessionUpdateRequest,
    EngramSummary,
    PinnedEngramRecord,
)


def create_chat_session(
    owner_user_id: UUID,
    payload: ChatSessionCreateRequest,
) -> ChatSessionRecord:
    now = datetime.now(UTC)
    session_id = uuid4()

    with get_conn() as conn, conn.cursor() as cur:
        cur.execute(
            """
            INSERT INTO chat_sessions (
                session_id,
                owner_user_id,
                project_id,
                title,
                provider,
                model_id,
                system_prompt,
                visibility_scope,
                autosave_enabled,
                created_at,
                updated_at
            ) VALUES (%s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s)
            RETURNING
                session_id,
                owner_user_id,
                project_id,
                title,
                provider,
                model_id,
                system_prompt,
                visibility_scope,
                autosave_enabled,
                created_at,
                updated_at
            """,
            (
                session_id,
                owner_user_id,
                payload.project_id,
                payload.title,
                payload.provider.value,
                payload.model_id,
                payload.system_prompt,
                payload.visibility_scope.value,
                payload.autosave_enabled,
                now,
                now,
            ),
        )
        row = cur.fetchone()
    return ChatSessionRecord(**row)


def list_chat_sessions(
    actor_user_id: UUID,
    project_id: str | None = None,
    limit: int = 50,
    offset: int = 0,
) -> list[ChatSessionRecord]:
    query = """
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
            created_at,
            updated_at
        FROM chat_sessions
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
    return [ChatSessionRecord(**row) for row in rows]


def get_chat_session(session_id: UUID, actor_user_id: UUID) -> ChatSessionRecord | None:
    with get_conn() as conn, conn.cursor() as cur:
        cur.execute(
            """
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
                created_at,
                updated_at
            FROM chat_sessions
            WHERE
                session_id = %s
                AND (owner_user_id = %s OR visibility_scope = 'project')
            """,
            (session_id, actor_user_id),
        )
        row = cur.fetchone()
    if not row:
        return None
    return ChatSessionRecord(**row)


def update_chat_session(
    session_id: UUID,
    actor_user_id: UUID,
    payload: ChatSessionUpdateRequest,
) -> ChatSessionRecord | None:
    with get_conn() as conn, conn.cursor() as cur:
        cur.execute(
            """
            UPDATE chat_sessions
            SET
                title = COALESCE(%s, title),
                provider = COALESCE(%s, provider),
                model_id = COALESCE(%s, model_id),
                system_prompt = COALESCE(%s, system_prompt),
                visibility_scope = COALESCE(%s, visibility_scope),
                autosave_enabled = COALESCE(%s, autosave_enabled),
                updated_at = %s
            WHERE
                session_id = %s
                AND owner_user_id = %s
            RETURNING
                session_id,
                owner_user_id,
                project_id,
                title,
                provider,
                model_id,
                system_prompt,
                visibility_scope,
                autosave_enabled,
                created_at,
                updated_at
            """,
            (
                payload.title,
                payload.provider.value if payload.provider else None,
                payload.model_id,
                payload.system_prompt,
                payload.visibility_scope.value if payload.visibility_scope else None,
                payload.autosave_enabled,
                datetime.now(UTC),
                session_id,
                actor_user_id,
            ),
        )
        row = cur.fetchone()
    if not row:
        return None
    return ChatSessionRecord(**row)


def create_chat_message(
    session_id: UUID,
    actor_user_id: UUID,
    role: str,
    content_text: str,
    provider: str | None = None,
    model_id: str | None = None,
    token_usage_json: dict | None = None,
    used_engram_ids: list[UUID] | None = None,
) -> ChatMessageRecord | None:
    message_id = uuid4()
    with get_conn() as conn, conn.cursor() as cur:
        cur.execute(
            """
            INSERT INTO chat_messages (
                message_id,
                session_id,
                role,
                content_text,
                provider,
                model_id,
                token_usage_json,
                used_engram_ids,
                created_at
            )
            SELECT
                %s,
                s.session_id,
                %s,
                %s,
                %s,
                %s,
                %s,
                %s,
                %s
            FROM chat_sessions s
            WHERE
                s.session_id = %s
                AND (s.owner_user_id = %s OR s.visibility_scope = 'project')
            RETURNING
                message_id,
                session_id,
                role,
                content_text,
                provider,
                model_id,
                token_usage_json,
                used_engram_ids,
                created_at
            """,
            (
                message_id,
                role,
                content_text,
                provider,
                model_id,
                Jsonb(token_usage_json or {}),
                used_engram_ids or [],
                datetime.now(UTC),
                session_id,
                actor_user_id,
            ),
        )
        row = cur.fetchone()
    if not row:
        return None
    return ChatMessageRecord(**row)


def list_chat_messages(
    session_id: UUID,
    actor_user_id: UUID,
    limit: int = 200,
    offset: int = 0,
) -> list[ChatMessageRecord]:
    with get_conn() as conn, conn.cursor() as cur:
        cur.execute(
            """
            SELECT
                m.message_id,
                m.session_id,
                m.role,
                m.content_text,
                m.provider,
                m.model_id,
                m.token_usage_json,
                m.used_engram_ids,
                m.created_at
            FROM chat_messages m
            JOIN chat_sessions s
              ON s.session_id = m.session_id
            WHERE
                m.session_id = %s
                AND (s.owner_user_id = %s OR s.visibility_scope = 'project')
            ORDER BY m.created_at ASC
            LIMIT %s OFFSET %s
            """,
            (session_id, actor_user_id, limit, offset),
        )
        rows = cur.fetchall()
    return [ChatMessageRecord(**row) for row in rows]


def pin_engram_to_session(
    session_id: UUID,
    engram_id: UUID,
    actor_user_id: UUID,
) -> PinnedEngramRecord | None:
    with get_conn() as conn, conn.cursor() as cur:
        cur.execute(
            """
            WITH accessible_session AS (
                SELECT s.session_id
                FROM chat_sessions s
                WHERE
                    s.session_id = %s
                    AND (s.owner_user_id = %s OR s.visibility_scope = 'project')
            ),
            accessible_engram AS (
                SELECT e.engram_id
                FROM engrams e
                WHERE
                    e.engram_id = %s
                    AND (e.owner_user_id = %s OR e.visibility_scope = 'project' OR e.owner_user_id IS NULL)
            )
            INSERT INTO session_pinned_engrams (
                session_id,
                engram_id,
                pinned_by_user_id,
                created_at
            )
            SELECT
                s.session_id,
                e.engram_id,
                %s,
                %s
            FROM accessible_session s
            CROSS JOIN accessible_engram e
            ON CONFLICT (session_id, engram_id) DO UPDATE
                SET pinned_by_user_id = EXCLUDED.pinned_by_user_id
            RETURNING
                session_id,
                engram_id,
                pinned_by_user_id,
                created_at
            """,
            (
                session_id,
                actor_user_id,
                engram_id,
                actor_user_id,
                actor_user_id,
                datetime.now(UTC),
            ),
        )
        row = cur.fetchone()
    if not row:
        return None
    return PinnedEngramRecord(**row)


def unpin_engram_from_session(
    session_id: UUID,
    engram_id: UUID,
    actor_user_id: UUID,
) -> bool:
    with get_conn() as conn, conn.cursor() as cur:
        cur.execute(
            """
            DELETE FROM session_pinned_engrams p
            USING chat_sessions s
            WHERE
                p.session_id = s.session_id
                AND p.session_id = %s
                AND p.engram_id = %s
                AND (s.owner_user_id = %s OR s.visibility_scope = 'project')
            RETURNING p.session_id
            """,
            (session_id, engram_id, actor_user_id),
        )
        row = cur.fetchone()
    return row is not None


def list_pinned_engrams(session_id: UUID, actor_user_id: UUID) -> list[PinnedEngramRecord]:
    with get_conn() as conn, conn.cursor() as cur:
        cur.execute(
            """
            SELECT
                p.session_id,
                p.engram_id,
                p.pinned_by_user_id,
                p.created_at
            FROM session_pinned_engrams p
            JOIN chat_sessions s
              ON s.session_id = p.session_id
            WHERE
                p.session_id = %s
                AND (s.owner_user_id = %s OR s.visibility_scope = 'project')
            ORDER BY p.created_at ASC
            """,
            (session_id, actor_user_id),
        )
        rows = cur.fetchall()
    return [PinnedEngramRecord(**row) for row in rows]


def list_pinned_engram_summaries(
    session_id: UUID,
    actor_user_id: UUID,
) -> list[EngramSummary]:
    with get_conn() as conn, conn.cursor() as cur:
        cur.execute(
            """
            SELECT
                e.engram_id,
                e.project_id,
                e.thread_id,
                e.title,
                e.abstract,
                e.created_at,
                e.tags,
                e.keywords,
                e.owner_user_id,
                e.visibility_scope
            FROM session_pinned_engrams p
            JOIN chat_sessions s
              ON s.session_id = p.session_id
            JOIN engrams e
              ON e.engram_id = p.engram_id
            WHERE
                p.session_id = %s
                AND (s.owner_user_id = %s OR s.visibility_scope = 'project')
                AND (e.owner_user_id = %s OR e.visibility_scope = 'project' OR e.owner_user_id IS NULL)
            ORDER BY p.created_at ASC
            """,
            (session_id, actor_user_id, actor_user_id),
        )
        rows = cur.fetchall()
    return [EngramSummary(**row) for row in rows]
