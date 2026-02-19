from __future__ import annotations

import re
from uuid import UUID, uuid4

import pytest
from fastapi.testclient import TestClient

from app.config import get_settings
from app.main import app


def _extract_csrf_token(html: str) -> str:
    match = re.search(r'name="csrf_token" value="([^"]+)"', html)
    assert match is not None
    return match.group(1)


def _login(client) -> None:  # noqa: ANN001
    settings = get_settings()
    login_page = client.get("/login")
    csrf_token = _extract_csrf_token(login_page.text)
    response = client.post(
        "/login",
        data={
            "username": settings.ui_demo_username,
            "password": settings.ui_demo_password,
            "csrf_token": csrf_token,
        },
        follow_redirects=False,
    )
    assert response.status_code == 303
    assert response.headers["location"] == "/ui"


@pytest.mark.integration
def test_admin_ui_mcp_token_create_and_revoke_workflow(client, clean_db) -> None:
    _login(client)

    admin_page = client.get("/ui/admin")
    assert admin_page.status_code == 200
    assert "MCP Tokens" in admin_page.text
    csrf_token = _extract_csrf_token(admin_page.text)

    create_response = client.post(
        "/ui/admin/mcp-tokens/create",
        data={
            "csrf_token": csrf_token,
            "name": "UI token",
            "scope": "read",
            "allowed_tools": "engram.query,chat.list_sessions",
            "allowed_project_ids": "engram-vault",
            "expires_in_days": "90",
        },
        follow_redirects=False,
    )
    assert create_response.status_code == 303
    assert create_response.headers["location"] == "/ui/admin"

    refreshed_admin = client.get("/ui/admin")
    assert refreshed_admin.status_code == 200
    assert "New token created:" in refreshed_admin.text
    assert "engram_mcp_" in refreshed_admin.text

    token_list = client.get("/api/v1/mcp/tokens")
    assert token_list.status_code == 200
    token_id = UUID(token_list.json()[0]["token_id"])

    revoke_page = client.get("/ui/admin")
    revoke_csrf = _extract_csrf_token(revoke_page.text)
    revoke_response = client.post(
        f"/ui/admin/mcp-tokens/{token_id}/revoke",
        data={"csrf_token": revoke_csrf},
        follow_redirects=False,
    )
    assert revoke_response.status_code == 303
    assert revoke_response.headers["location"] == "/ui/admin"

    listed_after = client.get("/api/v1/mcp/tokens")
    assert listed_after.status_code == 200
    assert listed_after.json()[0]["revoked_at"] is not None


@pytest.mark.integration
def test_non_admin_cannot_access_or_mutate_admin_mcp_token_ui(client, clean_db) -> None:
    _login(client)

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
    _login_with_credentials(viewer_client, username=username, password="viewerpass123")

    viewer_admin_page = viewer_client.get("/ui/admin")
    assert viewer_admin_page.status_code == 403

    csrf_page = viewer_client.get("/ui")
    csrf_token = _extract_csrf_token(csrf_page.text)
    create_denied = viewer_client.post(
        "/ui/admin/mcp-tokens/create",
        data={
            "csrf_token": csrf_token,
            "name": "viewer-ui-token",
            "scope": "read",
            "allowed_tools": "",
            "allowed_project_ids": "",
            "expires_in_days": "30",
        },
        follow_redirects=False,
    )
    assert create_denied.status_code == 403


def _login_with_credentials(client, username: str, password: str) -> None:  # noqa: ANN001
    login_page = client.get("/login")
    csrf_token = _extract_csrf_token(login_page.text)
    response = client.post(
        "/login",
        data={
            "username": username,
            "password": password,
            "csrf_token": csrf_token,
        },
        follow_redirects=False,
    )
    assert response.status_code == 303
    assert response.headers["location"] == "/ui"
