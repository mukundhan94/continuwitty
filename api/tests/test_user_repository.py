from __future__ import annotations

from contextlib import contextmanager
from datetime import UTC, datetime
from uuid import UUID

import pytest
from psycopg.errors import UniqueViolation

from app import user_repository
from app.models import UserRole


class _FakeCursor:
    def __init__(
        self,
        *,
        fetchone_value: dict | None = None,
        fetchall_value: list[dict] | None = None,
        execute_error: Exception | None = None,
    ) -> None:
        self.fetchone_value = fetchone_value
        self.fetchall_value = fetchall_value or []
        self.execute_error = execute_error
        self.executed: list[tuple[str, tuple | None]] = []

    def __enter__(self) -> _FakeCursor:
        return self

    def __exit__(self, exc_type, exc, tb) -> None:
        return None

    def execute(self, query: str, params: tuple | None = None) -> None:
        self.executed.append((query, params))
        if self.execute_error:
            raise self.execute_error

    def fetchone(self) -> dict | None:
        return self.fetchone_value

    def fetchall(self) -> list[dict]:
        return self.fetchall_value


class _FakeConnection:
    def __init__(self, cursor: _FakeCursor) -> None:
        self._cursor = cursor

    def cursor(self) -> _FakeCursor:
        return self._cursor


def _install_fake_connection(monkeypatch, cursor: _FakeCursor) -> None:
    connection = _FakeConnection(cursor)

    @contextmanager
    def _fake_get_conn():
        yield connection

    monkeypatch.setattr(user_repository, "get_conn", _fake_get_conn)


def test_get_user_auth_record_looks_up_username(monkeypatch) -> None:
    row = {
        "user_id": UUID("00000000-0000-0000-0000-000000000001"),
        "username": "admin",
        "password_hash": "hash",
        "role": "admin",
        "is_active": True,
        "default_project_id": "engram-vault",
        "created_at": datetime(2026, 2, 20, tzinfo=UTC),
    }
    cursor = _FakeCursor(fetchone_value=row)
    _install_fake_connection(monkeypatch, cursor)

    result = user_repository.get_user_auth_record("admin")

    assert result == row
    _, params = cursor.executed[0]
    assert params == ("admin",)


def test_list_users_returns_records(monkeypatch) -> None:
    rows = [
        {
            "user_id": UUID("00000000-0000-0000-0000-000000000011"),
            "username": "viewer_1",
            "role": "viewer",
            "is_active": True,
            "default_project_id": None,
            "created_at": datetime(2026, 2, 20, tzinfo=UTC),
        }
    ]
    cursor = _FakeCursor(fetchall_value=rows)
    _install_fake_connection(monkeypatch, cursor)

    result = user_repository.list_users(limit=1, offset=4)

    assert len(result) == 1
    assert result[0].username == "viewer_1"
    assert result[0].role == UserRole.viewer
    _, params = cursor.executed[0]
    assert params == (1, 4)


def test_create_user_returns_created_user(monkeypatch) -> None:
    created_user_id = UUID("00000000-0000-0000-0000-000000000111")
    row = {
        "user_id": created_user_id,
        "username": "analyst_1",
        "role": "analyst",
        "is_active": True,
        "default_project_id": "engram-vault",
        "created_at": datetime(2026, 2, 20, tzinfo=UTC),
    }
    cursor = _FakeCursor(fetchone_value=row)
    _install_fake_connection(monkeypatch, cursor)
    monkeypatch.setattr(user_repository, "uuid4", lambda: created_user_id)

    result = user_repository.create_user(
        username="analyst_1",
        password_hash="hash-value",
        role=UserRole.analyst,
        is_active=True,
    )

    assert result.user_id == created_user_id
    assert result.role == UserRole.analyst
    _, params = cursor.executed[0]
    assert params == (created_user_id, "analyst_1", "hash-value", "analyst", True)


def test_create_user_raises_value_error_for_duplicate_username(monkeypatch) -> None:
    cursor = _FakeCursor(execute_error=UniqueViolation("duplicate username"))
    _install_fake_connection(monkeypatch, cursor)

    with pytest.raises(ValueError, match="username already exists"):
        user_repository.create_user(
            username="admin",
            password_hash="hash-value",
            role=UserRole.admin,
            is_active=True,
        )


def test_update_user_returns_none_when_not_found(monkeypatch) -> None:
    cursor = _FakeCursor(fetchone_value=None)
    _install_fake_connection(monkeypatch, cursor)

    result = user_repository.update_user(
        user_id=UUID("00000000-0000-0000-0000-000000000999"),
        role=None,
        is_active=None,
        password_hash=None,
    )

    assert result is None


def test_update_user_applies_provided_fields(monkeypatch) -> None:
    target_user_id = UUID("00000000-0000-0000-0000-000000000212")
    row = {
        "user_id": target_user_id,
        "username": "viewer_2",
        "role": "viewer",
        "is_active": False,
        "default_project_id": None,
        "created_at": datetime(2026, 2, 20, tzinfo=UTC),
    }
    cursor = _FakeCursor(fetchone_value=row)
    _install_fake_connection(monkeypatch, cursor)

    result = user_repository.update_user(
        user_id=target_user_id,
        role=UserRole.viewer,
        is_active=False,
        password_hash="new-hash",
    )

    assert result is not None
    assert result.user_id == target_user_id
    assert result.role == UserRole.viewer
    _, params = cursor.executed[0]
    assert params == ("viewer", False, "new-hash", target_user_id)
