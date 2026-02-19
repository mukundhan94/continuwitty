from __future__ import annotations

import json
from datetime import UTC, datetime
from types import SimpleNamespace

from fastapi import FastAPI
from fastapi.testclient import TestClient

from app.oauth import api as oauth_api
from app.oauth.models import OAuthClientRecord


def _settings() -> SimpleNamespace:
    return SimpleNamespace(
        oauth_enabled=True,
        oauth_client_secret_pepper="oauth-pepper",
        oauth_issuer_url="",
        mcp_token_pepper="mcp-pepper",
        oauth_authorization_code_ttl_seconds=300,
        oauth_access_token_ttl_seconds=3600,
    )


def _oauth_client_record(
    *,
    token_endpoint_auth_method: str = "none",
) -> OAuthClientRecord:
    return OAuthClientRecord(
        client_id="engram_client_test",
        client_name="Test Client",
        redirect_uris=["https://example.com/callback"],
        grant_types=["authorization_code"],
        response_types=["code"],
        token_endpoint_auth_method=token_endpoint_auth_method,
        client_secret_hash="secret-hash"
        if token_endpoint_auth_method == "client_secret_post"
        else None,
        metadata_json={},
        created_at=datetime.now(UTC),
    )


def test_dedup_string_list_trims_and_deduplicates() -> None:
    assert oauth_api._dedup_string_list(["  one  ", "", "one", " two ", "two"]) == [
        "one",
        "two",
    ]


def test_validate_oauth_client_and_redirect_rejects_unknown_client(monkeypatch) -> None:
    monkeypatch.setattr("app.oauth.api.get_oauth_client", lambda *, client_id: None)

    client, error = oauth_api._validate_oauth_client_and_redirect(
        client_id="missing-client",
        redirect_uri="https://example.com/callback",
    )

    assert client is None
    assert error is not None
    assert error.status_code == 401
    payload = json.loads(error.body)
    assert payload["error"] == "invalid_client"


def test_validate_oauth_client_and_redirect_rejects_unregistered_redirect(monkeypatch) -> None:
    monkeypatch.setattr(
        "app.oauth.api.get_oauth_client",
        lambda *, client_id: _oauth_client_record(),
    )

    client, error = oauth_api._validate_oauth_client_and_redirect(
        client_id="engram_client_test",
        redirect_uri="https://evil.example.com/callback",
    )

    assert client is None
    assert error is not None
    assert error.status_code == 400
    payload = json.loads(error.body)
    assert payload["error"] == "invalid_request"


def test_handle_oauth_register_normalizes_lists_and_issues_secret(monkeypatch) -> None:
    captured_args: dict[str, object] = {}

    def _fake_create_oauth_client(**kwargs):  # noqa: ANN003
        captured_args.update(kwargs)
        return OAuthClientRecord(
            client_id=kwargs["client_id"],
            client_name=kwargs["client_name"],
            redirect_uris=kwargs["redirect_uris"],
            grant_types=kwargs["grant_types"],
            response_types=kwargs["response_types"],
            token_endpoint_auth_method=kwargs["token_endpoint_auth_method"],
            client_secret_hash=kwargs["client_secret_hash"],
            metadata_json=kwargs["metadata_json"],
            created_at=datetime(2026, 2, 19, tzinfo=UTC),
        )

    monkeypatch.setattr("app.oauth.api.build_oauth_client_id", lambda: "engram_client_static")
    monkeypatch.setattr("app.oauth.api.issue_oauth_client_secret", lambda: "secret-value")
    monkeypatch.setattr("app.oauth.api.oauth_secret_hash", lambda **_: "hashed-secret")
    monkeypatch.setattr("app.oauth.api.create_oauth_client", _fake_create_oauth_client)

    payload = oauth_api.OAuthClientRegistrationRequest(
        client_name="  Test Client  ",
        redirect_uris=[" https://example.com/callback ", "", "https://example.com/callback"],
        grant_types=[],
        response_types=[],
        token_endpoint_auth_method="client_secret_post",
    )

    response = oauth_api._handle_oauth_register(settings=_settings(), payload=payload)

    assert response.status_code == 201
    response_payload = json.loads(response.body)
    assert response_payload["client_id"] == "engram_client_static"
    assert response_payload["client_secret"] == "secret-value"
    assert response_payload["token_endpoint_auth_method"] == "client_secret_post"

    assert captured_args["client_name"] == "Test Client"
    assert captured_args["redirect_uris"] == ["https://example.com/callback"]
    assert captured_args["grant_types"] == ["authorization_code"]
    assert captured_args["response_types"] == ["code"]
    assert captured_args["client_secret_hash"] == "hashed-secret"


def test_oauth_authorize_redirects_to_login_when_session_missing(monkeypatch) -> None:
    monkeypatch.setattr(
        "app.oauth.api.get_oauth_client",
        lambda *, client_id: _oauth_client_record(),
    )

    app = FastAPI()
    app.include_router(
        oauth_api.create_oauth_router(
            settings=_settings(),
            resolve_session_user=lambda _request: None,
        )
    )
    client = TestClient(app)

    response = client.get(
        "/oauth/authorize",
        params={
            "response_type": "code",
            "client_id": "engram_client_test",
            "redirect_uri": "https://example.com/callback",
            "scope": "mcp:read",
            "state": "state-1",
            "code_challenge": "pkce-challenge",
            "code_challenge_method": "S256",
        },
        follow_redirects=False,
    )

    assert response.status_code == 303
    assert response.headers["location"].startswith("/login?next=")
