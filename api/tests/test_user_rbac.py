import re
import uuid

import pytest
from fastapi.testclient import TestClient

from app.config import get_settings
from app.main import app


def _extract_csrf_token(html: str) -> str:
    match = re.search(r'name="csrf_token" value="([^"]+)"', html)
    assert match is not None
    return match.group(1)


def _login(client: TestClient, username: str, password: str) -> None:
    login_page = client.get("/login")
    csrf_token = _extract_csrf_token(login_page.text)
    login_response = client.post(
        "/login",
        data={"username": username, "password": password, "csrf_token": csrf_token},
        follow_redirects=False,
    )
    assert login_response.status_code == 303
    assert login_response.headers["location"] == "/ui"


@pytest.mark.integration
def test_admin_can_manage_users(client, clean_db) -> None:
    settings = get_settings()
    _login(client, settings.ui_demo_username, settings.ui_demo_password)

    list_response = client.get("/api/v1/users")
    assert list_response.status_code == 200
    assert any(item["username"] == settings.ui_demo_username for item in list_response.json())

    username = f"analyst_{uuid.uuid4().hex[:8]}"
    create_response = client.post(
        "/api/v1/users",
        json={
            "username": username,
            "password": "StrongPass123",
            "role": "analyst",
            "is_active": True,
        },
    )
    assert create_response.status_code == 201
    created = create_response.json()
    assert created["username"] == username
    assert created["role"] == "analyst"

    update_response = client.patch(
        f"/api/v1/users/{created['user_id']}",
        json={"role": "viewer"},
    )
    assert update_response.status_code == 200
    updated = update_response.json()
    assert updated["role"] == "viewer"


@pytest.mark.integration
def test_non_admin_cannot_access_admin_routes(client, clean_db) -> None:
    settings = get_settings()
    _login(client, settings.ui_demo_username, settings.ui_demo_password)

    username = f"viewer_{uuid.uuid4().hex[:8]}"
    create_response = client.post(
        "/api/v1/users",
        json={
            "username": username,
            "password": "StrongPass123",
            "role": "viewer",
            "is_active": True,
        },
    )
    assert create_response.status_code == 201

    viewer_client = TestClient(app)
    _login(viewer_client, username, "StrongPass123")

    me_response = viewer_client.get("/api/v1/me")
    assert me_response.status_code == 200
    assert me_response.json()["role"] == "viewer"

    users_response = viewer_client.get("/api/v1/users")
    assert users_response.status_code == 403

    admin_ui_response = viewer_client.get("/ui/admin")
    assert admin_ui_response.status_code == 403
