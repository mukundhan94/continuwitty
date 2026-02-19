from __future__ import annotations

import base64
import hashlib
import json
import re
from urllib.parse import parse_qs, unquote, urlparse

import pytest


def _extract_csrf_token(html: str) -> str:
    match = re.search(r'name="csrf_token" value="([^"]+)"', html)
    assert match is not None
    return match.group(1)


def _login(client) -> None:  # noqa: ANN001
    login_page = client.get("/login")
    csrf_token = _extract_csrf_token(login_page.text)
    response = client.post(
        "/login",
        data={
            "username": "admin",
            "password": "admin123",
            "csrf_token": csrf_token,
        },
        follow_redirects=False,
    )
    assert response.status_code == 303


def _s256_challenge(verifier: str) -> str:
    digest = hashlib.sha256(verifier.encode()).digest()
    return base64.urlsafe_b64encode(digest).decode().rstrip("=")


def _mcp_frames(client, *, headers: dict[str, str] | None = None) -> list[dict]:  # noqa: ANN001
    response = client.post(
        "/api/v1/mcp/stream",
        json={
            "jsonrpc": "2.0",
            "id": "oauth-tools-list",
            "method": "tools/list",
            "params": {},
        },
        headers={"Accept": "text/event-stream", **(headers or {})},
    )
    assert response.status_code == 200
    frames: list[dict] = []
    for line in response.text.splitlines():
        if line.startswith("data: "):
            frames.append(json.loads(line[6:]))
    return frames


@pytest.mark.integration
def test_oauth_metadata_exposes_dynamic_registration(client, clean_db) -> None:
    metadata = client.get("/.well-known/oauth-authorization-server")
    assert metadata.status_code == 200
    payload = metadata.json()
    assert payload["authorization_endpoint"].endswith("/oauth/authorize")
    assert payload["token_endpoint"].endswith("/oauth/token")
    assert payload["registration_endpoint"].endswith("/oauth/register")
    assert "authorization_code" in payload["grant_types_supported"]

    protected = client.get("/.well-known/oauth-protected-resource")
    assert protected.status_code == 200
    protected_payload = protected.json()
    assert protected_payload["resource"].endswith("/api/v1/mcp/stream")
    assert "mcp:read" in protected_payload["scopes_supported"]


@pytest.mark.integration
def test_oauth_authorize_redirects_to_login_with_next(client, clean_db) -> None:
    register = client.post(
        "/oauth/register",
        json={
            "client_name": "vscode-mcp-client",
            "redirect_uris": ["http://127.0.0.1:33418"],
            "grant_types": ["authorization_code"],
            "response_types": ["code"],
            "token_endpoint_auth_method": "none",
        },
    )
    assert register.status_code == 201
    registered = register.json()

    verifier = "oauth-next-verifier"
    authorize = client.get(
        "/oauth/authorize",
        params={
            "response_type": "code",
            "client_id": registered["client_id"],
            "redirect_uri": "http://127.0.0.1:33418",
            "scope": "mcp:read",
            "state": "state-next",
            "code_challenge": _s256_challenge(verifier),
            "code_challenge_method": "S256",
        },
        follow_redirects=False,
    )
    assert authorize.status_code == 303
    assert authorize.headers["location"].startswith("/login?next=")

    parsed = urlparse(authorize.headers["location"])
    next_path = unquote(parse_qs(parsed.query)["next"][0])

    login_page = client.get(authorize.headers["location"])
    csrf_token = _extract_csrf_token(login_page.text)
    login = client.post(
        "/login",
        data={
            "username": "admin",
            "password": "admin123",
            "csrf_token": csrf_token,
            "next_path": next_path,
        },
        follow_redirects=False,
    )
    assert login.status_code == 303
    assert login.headers["location"] == next_path


@pytest.mark.integration
def test_oauth_authorization_code_exchange_issues_mcp_bearer_token(client, clean_db) -> None:
    register = client.post(
        "/oauth/register",
        json={
            "client_name": "vscode-mcp-client",
            "redirect_uris": ["http://127.0.0.1:33418"],
            "grant_types": ["authorization_code"],
            "response_types": ["code"],
            "token_endpoint_auth_method": "none",
        },
    )
    assert register.status_code == 201
    registered = register.json()

    _login(client)
    verifier = "oauth-code-verifier-123"
    authorize = client.get(
        "/oauth/authorize",
        params={
            "response_type": "code",
            "client_id": registered["client_id"],
            "redirect_uri": "http://127.0.0.1:33418",
            "scope": "mcp:write",
            "state": "state-123",
            "code_challenge": _s256_challenge(verifier),
            "code_challenge_method": "S256",
        },
        follow_redirects=False,
    )
    assert authorize.status_code == 303

    parsed_redirect = urlparse(authorize.headers["location"])
    query = parse_qs(parsed_redirect.query)
    code = query["code"][0]
    assert query["state"][0] == "state-123"

    token_response = client.post(
        "/oauth/token",
        data={
            "grant_type": "authorization_code",
            "code": code,
            "redirect_uri": "http://127.0.0.1:33418",
            "client_id": registered["client_id"],
            "code_verifier": verifier,
        },
    )
    assert token_response.status_code == 200
    token_payload = token_response.json()
    assert token_payload["token_type"] == "Bearer"
    assert token_payload["scope"] == "mcp:write"
    assert token_payload["access_token"].startswith("engram_mcp_")

    frames = _mcp_frames(
        client,
        headers={"Authorization": f"Bearer {token_payload['access_token']}"},
    )
    result_frames = [item for item in frames if "result" in item]
    assert result_frames
    tool_names = {item["name"] for item in result_frames[-1]["result"]["tools"]}
    assert "chat_send_message" in tool_names


@pytest.mark.integration
def test_oauth_token_rejects_invalid_pkce_verifier(client, clean_db) -> None:
    register = client.post(
        "/oauth/register",
        json={
            "client_name": "vscode-mcp-client",
            "redirect_uris": ["http://127.0.0.1:33418"],
        },
    )
    assert register.status_code == 201
    registered = register.json()

    _login(client)
    verifier = "oauth-valid-verifier"
    authorize = client.get(
        "/oauth/authorize",
        params={
            "response_type": "code",
            "client_id": registered["client_id"],
            "redirect_uri": "http://127.0.0.1:33418",
            "scope": "mcp:read",
            "code_challenge": _s256_challenge(verifier),
            "code_challenge_method": "S256",
        },
        follow_redirects=False,
    )
    assert authorize.status_code == 303
    code = parse_qs(urlparse(authorize.headers["location"]).query)["code"][0]

    token_response = client.post(
        "/oauth/token",
        data={
            "grant_type": "authorization_code",
            "code": code,
            "redirect_uri": "http://127.0.0.1:33418",
            "client_id": registered["client_id"],
            "code_verifier": "wrong-verifier",
        },
    )
    assert token_response.status_code == 400
    assert token_response.json()["error"] == "invalid_grant"


@pytest.mark.integration
def test_mcp_unauthorized_response_includes_oauth_resource_metadata_header(
    client, clean_db
) -> None:
    response = client.post(
        "/api/v1/mcp/stream",
        json={
            "jsonrpc": "2.0",
            "id": "oauth-authz",
            "method": "tools/list",
            "params": {},
        },
        headers={"Authorization": "Bearer invalid-token"},
    )
    assert response.status_code == 401
    challenge = response.headers["www-authenticate"]
    assert "resource_metadata=" in challenge
    assert 'scope="mcp:read"' in challenge
