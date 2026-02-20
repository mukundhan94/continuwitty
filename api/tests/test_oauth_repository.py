from __future__ import annotations

from contextlib import contextmanager
from datetime import UTC, datetime
from uuid import UUID

from psycopg.types.json import Jsonb

from app.oauth import repository as oauth_repository


class _FakeCursor:
    def __init__(
        self,
        *,
        fetchone_value: dict | None = None,
    ) -> None:
        self.fetchone_value = fetchone_value
        self.executed: list[tuple[str, tuple | None]] = []

    def __enter__(self) -> _FakeCursor:
        return self

    def __exit__(self, exc_type, exc, tb) -> None:
        return None

    def execute(self, query: str, params: tuple | None = None) -> None:
        self.executed.append((query, params))

    def fetchone(self) -> dict | None:
        return self.fetchone_value


class _FakeConnection:
    def __init__(self, cursor: _FakeCursor) -> None:
        self._cursor = cursor

    def cursor(self) -> _FakeCursor:
        return self._cursor


def _install_fake_connection(monkeypatch, cursor: _FakeCursor) -> _FakeCursor:
    connection = _FakeConnection(cursor)

    @contextmanager
    def _fake_get_conn():
        yield connection

    monkeypatch.setattr(oauth_repository, "get_conn", _fake_get_conn)
    return cursor


def _client_row() -> dict:
    return {
        "client_id": "client-1",
        "client_name": "Local MCP Client",
        "redirect_uris": ["https://example.test/callback"],
        "grant_types": ["authorization_code"],
        "response_types": ["code"],
        "token_endpoint_auth_method": "client_secret_post",
        "client_secret_hash": "hash-1",
        "metadata_json": {"owner": "engram"},
        "created_at": datetime(2026, 2, 20, tzinfo=UTC),
    }


def _auth_code_row() -> dict:
    return {
        "code_id": UUID("00000000-0000-0000-0000-000000000111"),
        "code_hash": "code-hash",
        "client_id": "client-1",
        "user_id": UUID("00000000-0000-0000-0000-000000000222"),
        "redirect_uri": "https://example.test/callback",
        "code_challenge": "challenge",
        "code_challenge_method": "S256",
        "requested_scope": "mcp:read",
        "resource": "engram-vault",
        "expires_at": datetime(2026, 2, 20, 12, 5, tzinfo=UTC),
        "consumed_at": None,
        "created_at": datetime(2026, 2, 20, 12, 0, tzinfo=UTC),
    }


def test_create_oauth_client_persists_and_returns_record(monkeypatch) -> None:
    cursor = _install_fake_connection(monkeypatch, _FakeCursor(fetchone_value=_client_row()))

    result = oauth_repository.create_oauth_client(
        client_id="client-1",
        client_name="Local MCP Client",
        redirect_uris=["https://example.test/callback"],
        grant_types=["authorization_code"],
        response_types=["code"],
        token_endpoint_auth_method="client_secret_post",
        client_secret_hash="hash-1",
        metadata_json={"owner": "engram"},
    )

    assert result.client_id == "client-1"
    assert result.redirect_uris == ["https://example.test/callback"]
    _, params = cursor.executed[0]
    assert params is not None
    assert params[0:7] == (
        "client-1",
        "Local MCP Client",
        ["https://example.test/callback"],
        ["authorization_code"],
        ["code"],
        "client_secret_post",
        "hash-1",
    )
    assert isinstance(params[7], Jsonb)


def test_get_oauth_client_returns_none_when_missing(monkeypatch) -> None:
    _install_fake_connection(monkeypatch, _FakeCursor(fetchone_value=None))

    result = oauth_repository.get_oauth_client(client_id="missing-client")

    assert result is None


def test_get_oauth_client_maps_nullable_columns(monkeypatch) -> None:
    row = _client_row() | {
        "redirect_uris": None,
        "grant_types": None,
        "response_types": None,
        "metadata_json": None,
    }
    _install_fake_connection(monkeypatch, _FakeCursor(fetchone_value=row))

    result = oauth_repository.get_oauth_client(client_id="client-1")

    assert result is not None
    assert result.redirect_uris == []
    assert result.grant_types == []
    assert result.response_types == []
    assert result.metadata_json == {}


def test_create_oauth_authorization_code_returns_record(monkeypatch) -> None:
    row = _auth_code_row()
    cursor = _install_fake_connection(monkeypatch, _FakeCursor(fetchone_value=row))
    expires_at = datetime(2026, 2, 20, 12, 5, tzinfo=UTC)

    result = oauth_repository.create_oauth_authorization_code(
        code_id=row["code_id"],
        code_hash="code-hash",
        client_id="client-1",
        user_id=row["user_id"],
        redirect_uri="https://example.test/callback",
        code_challenge="challenge",
        code_challenge_method="S256",
        requested_scope="mcp:read",
        resource="engram-vault",
        expires_at=expires_at,
    )

    assert result.code_id == row["code_id"]
    assert result.requested_scope == "mcp:read"
    _, params = cursor.executed[0]
    assert params is not None
    assert params[0:4] == (row["code_id"], "code-hash", "client-1", row["user_id"])
    assert params[9] == expires_at


def test_get_oauth_authorization_code_by_hash_uses_default_scope(monkeypatch) -> None:
    row = _auth_code_row() | {"requested_scope": None}
    _install_fake_connection(monkeypatch, _FakeCursor(fetchone_value=row))

    result = oauth_repository.get_oauth_authorization_code_by_hash(code_hash="code-hash")

    assert result is not None
    assert result.requested_scope == ""


def test_get_oauth_authorization_code_by_hash_returns_none_when_missing(monkeypatch) -> None:
    _install_fake_connection(monkeypatch, _FakeCursor(fetchone_value=None))

    result = oauth_repository.get_oauth_authorization_code_by_hash(code_hash="missing")

    assert result is None


def test_consume_oauth_authorization_code_returns_none_when_already_consumed(monkeypatch) -> None:
    _install_fake_connection(monkeypatch, _FakeCursor(fetchone_value=None))

    result = oauth_repository.consume_oauth_authorization_code(
        code_id=UUID("00000000-0000-0000-0000-000000000111"),
        consumed_at=datetime(2026, 2, 20, 12, 10, tzinfo=UTC),
    )

    assert result is None


def test_consume_oauth_authorization_code_returns_updated_record(monkeypatch) -> None:
    consumed_at = datetime(2026, 2, 20, 12, 10, tzinfo=UTC)
    row = _auth_code_row() | {"consumed_at": consumed_at}
    cursor = _install_fake_connection(monkeypatch, _FakeCursor(fetchone_value=row))
    code_id = UUID("00000000-0000-0000-0000-000000000111")

    result = oauth_repository.consume_oauth_authorization_code(
        code_id=code_id,
        consumed_at=consumed_at,
    )

    assert result is not None
    assert result.consumed_at == consumed_at
    _, params = cursor.executed[0]
    assert params == (consumed_at, code_id)
