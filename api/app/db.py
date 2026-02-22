from collections.abc import Iterator
from contextlib import contextmanager
from pathlib import Path

import psycopg
from psycopg.rows import dict_row

from .auth import hash_password
from .config import get_settings, is_production_env

_DEFAULT_BOOTSTRAP_ADMIN_HASH = (
    "pbkdf2_sha256$390000$00112233445566778899aabbccddeeff$"
    "45c0bdc16f1609a69d14a4e8d89974fd24556c45af75ee01c34bdb9de1c1832a"
)


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
            _harden_bootstrap_admin_credentials(cur=cur, settings=settings)
        conn.commit()


def _harden_bootstrap_admin_credentials(*, cur, settings) -> None:
    if not is_production_env(settings):
        return

    username = (settings.ui_demo_username or "").strip() or "admin"
    cur.execute(
        """
        SELECT password_hash
        FROM users
        WHERE username = %s
        """,
        (username,),
    )
    row = cur.fetchone()
    if row is None:
        return

    current_hash = row[0] if isinstance(row, tuple) else row.get("password_hash")
    if current_hash != _DEFAULT_BOOTSTRAP_ADMIN_HASH:
        return

    replacement_hash = (settings.ui_demo_password_hash or "").strip()
    if not replacement_hash:
        replacement_hash = hash_password(settings.ui_demo_password)
    cur.execute(
        """
        UPDATE users
        SET password_hash = %s
        WHERE username = %s
        """,
        (replacement_hash, username),
    )
