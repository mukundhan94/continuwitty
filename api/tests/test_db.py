from __future__ import annotations

import pytest

from app import db


class _SettingsStub:
    def __init__(self, database_url: str) -> None:
        self.database_url = database_url
        self.app_env = "development"
        self.ui_demo_username = "admin"
        self.ui_demo_password = "admin123"
        self.ui_demo_password_hash = None


class _FakeCursor:
    def __init__(self) -> None:
        self.executed_sql: list[str] = []
        self.executed_params: list[tuple[object, ...] | None] = []
        self.fetchone_value = None

    def __enter__(self) -> _FakeCursor:
        return self

    def __exit__(self, exc_type, exc, tb) -> None:
        return None

    def execute(self, sql: str, params=None) -> None:  # noqa: ANN001
        self.executed_sql.append(sql)
        self.executed_params.append(params)

    def fetchone(self):  # noqa: ANN201
        return self.fetchone_value


class _FakeConnection:
    def __init__(self, cursor: _FakeCursor) -> None:
        self._cursor = cursor
        self.commits = 0
        self.rollbacks = 0
        self.closes = 0

    def __enter__(self) -> _FakeConnection:
        return self

    def __exit__(self, exc_type, exc, tb) -> None:
        return None

    def cursor(self) -> _FakeCursor:
        return self._cursor

    def commit(self) -> None:
        self.commits += 1

    def rollback(self) -> None:
        self.rollbacks += 1

    def close(self) -> None:
        self.closes += 1


def test_get_conn_commits_and_closes_on_success(monkeypatch) -> None:
    fake_cursor = _FakeCursor()
    fake_connection = _FakeConnection(fake_cursor)
    connect_calls: list[tuple[str, object]] = []

    def _fake_connect(database_url: str, row_factory=None):
        connect_calls.append((database_url, row_factory))
        return fake_connection

    monkeypatch.setattr(db, "get_settings", lambda: _SettingsStub("postgresql://example/svc"))
    monkeypatch.setattr(db.psycopg, "connect", _fake_connect)

    with db.get_conn() as conn:
        assert conn is fake_connection

    assert connect_calls == [("postgresql://example/svc", db.dict_row)]
    assert fake_connection.commits == 1
    assert fake_connection.rollbacks == 0
    assert fake_connection.closes == 1


def test_get_conn_rolls_back_on_exception(monkeypatch) -> None:
    fake_connection = _FakeConnection(_FakeCursor())

    def _fake_connect(database_url: str, row_factory=None):
        return fake_connection

    monkeypatch.setattr(db, "get_settings", lambda: _SettingsStub("postgresql://example/svc"))
    monkeypatch.setattr(db.psycopg, "connect", _fake_connect)

    with pytest.raises(RuntimeError, match="boom"), db.get_conn():
        raise RuntimeError("boom")

    assert fake_connection.commits == 0
    assert fake_connection.rollbacks == 1
    assert fake_connection.closes == 1


def test_ensure_schema_initialized_executes_schema_sql(monkeypatch) -> None:
    fake_cursor = _FakeCursor()
    fake_connection = _FakeConnection(fake_cursor)
    connect_calls: list[str] = []

    def _fake_connect(database_url: str):
        connect_calls.append(database_url)
        return fake_connection

    monkeypatch.setattr(db, "get_settings", lambda: _SettingsStub("postgresql://example/svc"))
    monkeypatch.setattr(db.psycopg, "connect", _fake_connect)
    monkeypatch.setattr(db.Path, "read_text", lambda self, encoding: "SELECT 1;", raising=False)

    db.ensure_schema_initialized()

    assert connect_calls == ["postgresql://example/svc"]
    assert fake_cursor.executed_sql == ["SELECT 1;"]
    assert fake_connection.commits == 1


def test_harden_bootstrap_admin_credentials_updates_default_hash(monkeypatch) -> None:
    fake_cursor = _FakeCursor()
    fake_cursor.fetchone_value = (db._DEFAULT_BOOTSTRAP_ADMIN_HASH,)

    monkeypatch.setattr(db, "hash_password", lambda _password: "hashed-replacement")
    settings = _SettingsStub("postgresql://example/svc")
    settings.app_env = "production"
    settings.ui_demo_username = "admin"
    settings.ui_demo_password = "StrongPassword-12345"

    db._harden_bootstrap_admin_credentials(cur=fake_cursor, settings=settings)

    assert any("SELECT password_hash" in sql for sql in fake_cursor.executed_sql)
    assert any("UPDATE users" in sql for sql in fake_cursor.executed_sql)


def test_harden_bootstrap_admin_credentials_skips_non_default_hash(monkeypatch) -> None:
    fake_cursor = _FakeCursor()
    fake_cursor.fetchone_value = ("custom-password-hash",)
    settings = _SettingsStub("postgresql://example/svc")
    settings.app_env = "production"

    db._harden_bootstrap_admin_credentials(cur=fake_cursor, settings=settings)

    assert any("SELECT password_hash" in sql for sql in fake_cursor.executed_sql)
    assert not any("UPDATE users" in sql for sql in fake_cursor.executed_sql)
