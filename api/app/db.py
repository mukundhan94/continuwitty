from collections.abc import Iterator
from contextlib import contextmanager
from pathlib import Path

import psycopg
from psycopg.rows import dict_row

from .config import get_settings


@contextmanager
def get_conn() -> Iterator[psycopg.Connection]:
    settings = get_settings()
    conn = psycopg.connect(settings.database_url, row_factory=dict_row)
    try:
        yield conn
        conn.commit()
    except Exception:
        conn.rollback()
        raise
    finally:
        conn.close()


def ensure_schema_initialized() -> None:
    settings = get_settings()
    schema_path = Path(__file__).resolve().parents[2] / "db" / "init" / "001_schema.sql"
    schema_sql = schema_path.read_text(encoding="utf-8")
    with psycopg.connect(settings.database_url) as conn:
        with conn.cursor() as cur:
            cur.execute(schema_sql)
        conn.commit()
