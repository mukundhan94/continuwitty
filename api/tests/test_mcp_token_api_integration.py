from __future__ import annotations

import re
from uuid import uuid4

import pytest
from fastapi.testclient import TestClient

from app.config import get_settings
from app.main import app


def _extract_csrf_token(html: str) -> str:
    match = re.search(r'name="csrf_token" value="([^"]+)"', html)
    assert match is not None
    return match.group(1)


def _login(client, username: str, password: str) -> None:  # noqa: ANN001
    login_page = client.get("/login")
    csrf_token = _extract_csrf_token(login_page.text)
    response = client.post(
        "/login",
        data={"username": username, "password": password, "csrf_token": csrf_token},
        follow_redirects=False,
    )
    assert response.status_code == 303
    assert response.headers["location"] == "/ui"


@pytest.mark.integration
def test_admin_can_create_list_and_revoke_mcp_tokens(client, clean_db) -> None:
    settings = get_settings()
    _login(client, settings.ui_demo_username, settings.ui_demo_password)

    create_response = client.post(
        "/api/v1/mcp/tokens",
        json={
            "name": "LibreChat read token",
            "scope": "read",
            "allowed_tools": ["engram.query", "chat.list_sessions"],
            "allowed_project_ids": ["engram-vault"],
            "expires_in_days": 90,
        },
    )
    assert create_response.status_code == 201
    created = create_response.json()
    assert created["token"].startswith("engram_mcp_")
    token_id = created["token_id"]

    list_response = client.get("/api/v1/mcp/tokens")
    assert list_response.status_code == 200
    listed = next(item for item in list_response.json() if item["token_id"] == token_id)
    assert listed["name"] == "LibreChat read token"
    assert listed["scope"] == "read"
    assert listed["is_active"] is True
    assert "token" not in listed

    revoke_response = client.post(
        f"/api/v1/mcp/tokens/{token_id}/revoke",
        json={"reason": "rotation"},
    )
    assert revoke_response.status_code == 200
    revoked = revoke_response.json()
    assert revoked["token_id"] == token_id
    assert revoked["is_active"] is False
    assert revoked["revoked_at"] is not None


@pytest.mark.integration
def test_non_admin_cannot_create_or_revoke_mcp_tokens(client, clean_db) -> None:
    settings = get_settings()
    _login(client, settings.ui_demo_username, settings.ui_demo_password)

    username = f"viewer_{uuid4().hex[:8]}"
    create_user_response = client.post(
        "/api/v1/users",
        json={
            "username": username,
            "password": "viewerpass123",
            "role": "viewer",
            "is_active": True,
        },
    )
    assert create_user_response.status_code == 201

    viewer_client = TestClient(app)
    _login(viewer_client, username, "viewerpass123")

    create_denied = viewer_client.post(
        "/api/v1/mcp/tokens",
        json={"name": "viewer token", "scope": "read", "expires_in_days": 30},
    )
    assert create_denied.status_code == 403

    admin_token = client.post(
        "/api/v1/mcp/tokens",
        json={"name": "admin token", "scope": "read", "expires_in_days": 30},
    )
    assert admin_token.status_code == 201
    token_id = admin_token.json()["token_id"]

    revoke_denied = viewer_client.post(
        f"/api/v1/mcp/tokens/{token_id}/revoke",
        json={"reason": "malicious"},
    )
    assert revoke_denied.status_code == 403
