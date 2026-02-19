from __future__ import annotations

import os
from pathlib import Path

import psycopg
import pytest
from fastapi.testclient import TestClient

from app.config import get_settings
from app.main import app, login_attempt_guard

DEFAULT_DB_URL = "postgresql://engram:engram@localhost:5432/engram_vault"


@pytest.fixture(autouse=True)
def clear_settings_cache() -> None:
    get_settings.cache_clear()
    yield
    get_settings.cache_clear()


@pytest.fixture(autouse=True)
def clear_login_guard_state() -> None:
    login_attempt_guard.reset()
    yield
    login_attempt_guard.reset()


@pytest.fixture
def client() -> TestClient:
    return TestClient(app)


@pytest.fixture(scope="session")
def db_url() -> str:
    return os.getenv("DATABASE_URL", DEFAULT_DB_URL)


@pytest.fixture
def db_conn(db_url: str):
    try:
        conn = psycopg.connect(db_url)
    except Exception as exc:
        pytest.skip(f"integration DB unavailable: {exc}")

    try:
        yield conn
    finally:
        conn.close()


@pytest.fixture
def ensure_schema(db_conn) -> None:
    schema_path = Path(__file__).resolve().parents[2] / "db" / "init" / "001_schema.sql"
    schema_sql = schema_path.read_text(encoding="utf-8")
    with db_conn.cursor() as cur:
        cur.execute(schema_sql)
    db_conn.commit()


@pytest.fixture
def clean_db(db_conn, ensure_schema) -> None:
    with db_conn.cursor() as cur:
        cur.execute(
            """
            TRUNCATE TABLE
                engram_collection_items,
                engram_collections,
                session_pinned_engrams,
                chat_messages,
                chat_sessions,
                document_chunks,
                documents,
                oauth_authorization_codes,
                oauth_clients,
                mcp_tokens,
                engrams,
                projects
            CASCADE
            """
        )
        cur.execute(
            """
            INSERT INTO users (user_id, username, password_hash, role, is_active)
            VALUES (
              '00000000-0000-0000-0000-000000000001'::UUID,
              'admin',
              'pbkdf2_sha256$390000$00112233445566778899aabbccddeeff$45c0bdc16f1609a69d14a4e8d89974fd24556c45af75ee01c34bdb9de1c1832a',
              'admin',
              TRUE
            )
            ON CONFLICT (username) DO NOTHING
            """
        )
        cur.execute(
            """
            INSERT INTO projects (project_id, name, description, owner_user_id, is_archived)
            SELECT
              'engram-vault',
              'Engram Vault',
              'Default local workspace project.',
              COALESCE(
                (SELECT user_id FROM users WHERE username = 'admin' LIMIT 1),
                (SELECT user_id FROM users ORDER BY created_at ASC LIMIT 1)
              ),
              FALSE
            ON CONFLICT (project_id) DO NOTHING
            """
        )
        cur.execute(
            """
            UPDATE users
            SET default_project_id = 'engram-vault'
            WHERE username = 'admin'
            """
        )
    db_conn.commit()
