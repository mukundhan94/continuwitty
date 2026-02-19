from __future__ import annotations

import re

import pytest

from app.config import get_settings


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


@pytest.mark.integration
def test_projects_list_and_default_endpoints(client, clean_db) -> None:
    _login(client)

    list_response = client.get("/api/v1/projects")
    assert list_response.status_code == 200
    projects = list_response.json()
    assert any(item["project_id"] == "engram-vault" for item in projects)

    default_response = client.get("/api/v1/projects/default")
    assert default_response.status_code == 200
    assert default_response.json()["default_project_id"] == "engram-vault"


@pytest.mark.integration
def test_create_project_and_set_default(client, clean_db) -> None:
    _login(client)

    created = client.post(
        "/api/v1/projects",
        json={
            "project_id": "phase31-alpha",
            "name": "Phase 31 Alpha",
            "description": "Project for default selection tests.",
        },
    )
    assert created.status_code == 201
    assert created.json()["project_id"] == "phase31-alpha"

    updated = client.patch(
        "/api/v1/projects/default",
        json={"project_id": "phase31-alpha"},
    )
    assert updated.status_code == 200
    assert updated.json()["default_project_id"] == "phase31-alpha"

    current = client.get("/api/v1/projects/default")
    assert current.status_code == 200
    assert current.json()["default_project_id"] == "phase31-alpha"


@pytest.mark.integration
def test_create_engram_uses_default_project_when_missing(client, clean_db) -> None:
    _login(client)

    response = client.post(
        "/api/v1/engrams",
        json={
            "title": "Default project fallback engram",
            "abstract": "",
            "detailed_summary_markdown": "Fallback behavior should resolve to default project.",
            "tags": [],
            "keywords": [],
        },
    )
    assert response.status_code == 200
    payload = response.json()
    assert payload["resolved_project_id"] == "engram-vault"
    assert payload["used_default_project"] is True

    listed = client.get("/api/v1/engrams", params={"project_id": "engram-vault"})
    assert listed.status_code == 200
    assert any(item["engram_id"] == payload["engram_id"] for item in listed.json())


@pytest.mark.integration
def test_create_engram_without_project_or_default_returns_422(client, clean_db, db_conn) -> None:
    _login(client)

    with db_conn.cursor() as cur:
        cur.execute("UPDATE users SET default_project_id = NULL WHERE username = 'admin'")
    db_conn.commit()

    response = client.post(
        "/api/v1/engrams",
        json={
            "title": "No fallback",
            "abstract": "Should fail without explicit project and no default.",
            "detailed_summary_markdown": "No project context.",
        },
    )
    assert response.status_code == 422
    assert "project_id is required" in response.json()["detail"]
