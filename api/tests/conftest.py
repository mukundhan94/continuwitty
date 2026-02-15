from __future__ import annotations

import os
from pathlib import Path

import psycopg
import pytest
from fastapi.testclient import TestClient

from app.config import get_settings
from app.main import app

DEFAULT_DB_URL = "postgresql://engram:engram@localhost:5432/engram_vault"


@pytest.fixture(autouse=True)
def clear_settings_cache() -> None:
    get_settings.cache_clear()
    yield
    get_settings.cache_clear()


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
        cur.execute("TRUNCATE TABLE engrams CASCADE")
    db_conn.commit()
