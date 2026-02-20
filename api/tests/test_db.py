from __future__ import annotations

import pytest

from app import db


class _SettingsStub:
    def __init__(self, database_url: str) -> None:
        self.database_url = database_url


class _FakeCursor:
    def __init__(self) -> None:
        self.executed_sql: list[str] = []

    def __enter__(self) -> _FakeCursor:
        return self

    def __exit__(self, exc_type, exc, tb) -> None:
        return None

    def execute(self, sql: str) -> None:
        self.executed_sql.append(sql)


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
