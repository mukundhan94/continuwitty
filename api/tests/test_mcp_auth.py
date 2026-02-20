from __future__ import annotations

from types import SimpleNamespace
from uuid import uuid4

import pytest
from fastapi import HTTPException
from starlette.requests import Request

from app.config import Settings
from app.mcp.auth import resolve_mcp_actor


def _request(*, authorization: str | None = None) -> Request:
    headers: list[tuple[bytes, bytes]] = []
    if authorization is not None:
        headers.append((b"authorization", authorization.encode("utf-8")))
    scope = {
        "type": "http",
        "asgi": {"version": "3.0"},
        "http_version": "1.1",
        "method": "GET",
        "scheme": "http",
        "path": "/api/v1/mcp",
        "raw_path": b"/api/v1/mcp",
        "query_string": b"",
        "headers": headers,
        "client": ("testclient", 12345),
        "server": ("testserver", 80),
    }
    return Request(scope)


def _settings(*, oauth_enabled: bool = False) -> Settings:
    return Settings(
        oauth_enabled=oauth_enabled,
        mcp_token_pepper="test-pepper",
    )


def test_resolve_mcp_actor_uses_session_actor_without_bearer() -> None:
    expected_actor = {"user_id": str(uuid4()), "username": "admin", "role": "admin"}
    result = resolve_mcp_actor(
        request=_request(),
        settings=_settings(),
        require_session_actor=lambda _request: expected_actor,
    )
    assert result.actor == expected_actor
    assert result.token_auth is None


def test_resolve_mcp_actor_rejects_non_bearer_authorization_header() -> None:
    with pytest.raises(HTTPException) as exc_info:
        resolve_mcp_actor(
            request=_request(authorization="Basic abc"),
            settings=_settings(),
            require_session_actor=lambda _request: pytest.fail("session actor should not be used"),
        )

    assert exc_info.value.status_code == 401
    assert exc_info.value.detail == "Authentication required"


def test_resolve_mcp_actor_oauth_mode_sets_www_authenticate_header() -> None:
    with pytest.raises(HTTPException) as exc_info:
        resolve_mcp_actor(
            request=_request(authorization="Bearer"),
            settings=_settings(oauth_enabled=True),
            require_session_actor=lambda _request: pytest.fail("session actor should not be used"),
        )

    assert exc_info.value.status_code == 401
    assert "WWW-Authenticate" in exc_info.value.headers
    assert "oauth-protected-resource" in exc_info.value.headers["WWW-Authenticate"]


def test_resolve_mcp_actor_rejects_invalid_plaintext_token(monkeypatch) -> None:
    monkeypatch.setattr(
        "app.mcp.auth.parse_plaintext_token",
        lambda _token: (_ for _ in ()).throw(ValueError("invalid token")),
    )

    with pytest.raises(HTTPException) as exc_info:
        resolve_mcp_actor(
            request=_request(authorization="Bearer invalid"),
            settings=_settings(),
            require_session_actor=lambda _request: pytest.fail("session actor should not be used"),
        )

    assert exc_info.value.status_code == 401


def test_resolve_mcp_actor_rejects_inactive_owner(monkeypatch) -> None:
    token_id = uuid4()
    owner_user_id = uuid4()
    record = SimpleNamespace(
        token_id=token_id,
        owner_user_id=owner_user_id,
        token_secret_hash="expected",
    )
    monkeypatch.setattr("app.mcp.auth.parse_plaintext_token", lambda _token: (token_id, "secret"))
    monkeypatch.setattr("app.mcp.auth.get_mcp_token_by_id", lambda **_kwargs: record)
    monkeypatch.setattr("app.mcp.auth.token_is_active", lambda _record: True)
    monkeypatch.setattr("app.mcp.auth.verify_token_secret", lambda **_kwargs: True)
    monkeypatch.setattr(
        "app.mcp.auth.get_user_auth_record_by_id",
        lambda _user_id: {"user_id": owner_user_id, "username": "owner", "role": "admin"},
    )

    with pytest.raises(HTTPException) as exc_info:
        resolve_mcp_actor(
            request=_request(authorization="Bearer token"),
            settings=_settings(),
            require_session_actor=lambda _request: pytest.fail("session actor should not be used"),
        )

    assert exc_info.value.status_code == 401


def test_resolve_mcp_actor_resolves_valid_token_actor(monkeypatch) -> None:
    token_id = uuid4()
    owner_user_id = uuid4()
    record = SimpleNamespace(
        token_id=token_id,
        owner_user_id=owner_user_id,
        token_secret_hash="expected",
    )
    touched: dict[str, object] = {}
    token_auth = SimpleNamespace(scope="read", allowed_tools={"engram.query"})

    monkeypatch.setattr("app.mcp.auth.parse_plaintext_token", lambda _token: (token_id, "secret"))
    monkeypatch.setattr("app.mcp.auth.get_mcp_token_by_id", lambda **_kwargs: record)
    monkeypatch.setattr("app.mcp.auth.token_is_active", lambda _record: True)
    monkeypatch.setattr("app.mcp.auth.verify_token_secret", lambda **_kwargs: True)
    monkeypatch.setattr(
        "app.mcp.auth.get_user_auth_record_by_id",
        lambda _user_id: {
            "user_id": owner_user_id,
            "username": "owner",
            "role": "admin",
            "is_active": True,
        },
    )
    monkeypatch.setattr(
        "app.mcp.auth.touch_mcp_token_last_used",
        lambda **kwargs: touched.update(kwargs),
    )
    monkeypatch.setattr("app.mcp.auth.build_auth_context", lambda **_kwargs: token_auth)

    result = resolve_mcp_actor(
        request=_request(authorization="Bearer token"),
        settings=_settings(),
        require_session_actor=lambda _request: pytest.fail("session actor should not be used"),
    )

    assert result.actor == {
        "user_id": str(owner_user_id),
        "username": "owner",
        "role": "admin",
    }
    assert result.token_auth is token_auth
    assert touched["token_id"] == token_id
