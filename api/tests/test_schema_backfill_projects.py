from __future__ import annotations

from pathlib import Path
from uuid import uuid4

import pytest


@pytest.mark.integration
def test_schema_adds_phase31_tables_and_columns(db_conn, clean_db) -> None:
    with db_conn.cursor() as cur:
        cur.execute(
            """
            SELECT column_name
            FROM information_schema.columns
            WHERE table_name = 'users'
            """
        )
        user_columns = {row[0] for row in cur.fetchall()}
        assert "default_project_id" in user_columns

        cur.execute(
            """
            SELECT column_name
            FROM information_schema.columns
            WHERE table_name = 'chat_sessions'
            """
        )
        session_columns = {row[0] for row in cur.fetchall()}
        assert "deleted_at" in session_columns
        assert "deleted_by_user_id" in session_columns
        assert "delete_reason" in session_columns

        cur.execute(
            """
            SELECT column_name
            FROM information_schema.columns
            WHERE table_name = 'engrams'
            """
        )
        engram_columns = {row[0] for row in cur.fetchall()}
        assert "deleted_at" in engram_columns
        assert "deleted_by_user_id" in engram_columns
        assert "updated_by_user_id" in engram_columns

        cur.execute("SELECT to_regclass('projects') AS exists")
        assert cur.fetchone()[0] == "projects"
        cur.execute("SELECT to_regclass('engram_collections') AS exists")
        assert cur.fetchone()[0] == "engram_collections"
        cur.execute("SELECT to_regclass('engram_collection_items') AS exists")
        assert cur.fetchone()[0] == "engram_collection_items"


@pytest.mark.integration
def test_schema_backfills_projects_from_existing_records(db_conn, clean_db) -> None:
    session_id = uuid4()
    engram_id = uuid4()
    with db_conn.cursor() as cur:
        cur.execute("DELETE FROM projects")
        cur.execute("UPDATE users SET default_project_id = NULL WHERE username = 'admin'")
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
                autosave_strategy,
                autosave_interval_minutes,
                autosave_min_messages,
                retention_days,
                retention_max_snapshots
            )
            VALUES (
                %s,
                '00000000-0000-0000-0000-000000000001'::UUID,
                'backfill-project',
                'Backfill seed',
                'openai',
                'gpt-4o-mini',
                '',
                'private',
                FALSE,
                'off',
                30,
                6,
                30,
                60
            )
            """,
            (session_id,),
        )
        cur.execute(
            """
            INSERT INTO engrams (
                engram_id,
                project_id,
                thread_id,
                title,
                abstract,
                engram_json,
                engram_markdown,
                tags,
                keywords,
                retrieval_text,
                embedding_model,
                embed,
                owner_user_id,
                visibility_scope
            )
            VALUES (
                %s,
                'backfill-project',
                'thread',
                'Backfill engram',
                'Backfill abstract',
                '{}'::jsonb,
                'Backfill markdown',
                '{}'::text[],
                '{}'::text[],
                'Backfill retrieval',
                'local-deterministic-v1',
                %s::vector,
                '00000000-0000-0000-0000-000000000001'::UUID,
                'private'
            )
            """,
            (engram_id, "[" + ",".join(["0"] * 256) + "]"),
        )
    db_conn.commit()

    schema_path = Path(__file__).resolve().parents[2] / "db" / "init" / "001_schema.sql"
    with db_conn.cursor() as cur:
        cur.execute(schema_path.read_text(encoding="utf-8"))
    db_conn.commit()

    with db_conn.cursor() as cur:
        cur.execute("SELECT project_id FROM projects WHERE project_id = 'backfill-project' LIMIT 1")
        assert cur.fetchone() is not None
        cur.execute("SELECT default_project_id FROM users WHERE username = 'admin' LIMIT 1")
        default_project_id = cur.fetchone()[0]
        assert default_project_id in {"engram-vault", "backfill-project"}
