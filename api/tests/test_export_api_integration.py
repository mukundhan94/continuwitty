from __future__ import annotations

import re
import uuid
import zipfile
from io import BytesIO

import pytest

from app.config import get_settings


def _extract_csrf_token(html: str) -> str:
    match = re.search(r'name="csrf_token" value="([^"]+)"', html)
    assert match is not None
    return match.group(1)


def _login(client, *, username: str, password: str) -> None:  # noqa: ANN001
    login_page = client.get("/login")
    csrf_token = _extract_csrf_token(login_page.text)
    response = client.post(
        "/login",
        data={"username": username, "password": password, "csrf_token": csrf_token},
        follow_redirects=False,
    )
    assert response.status_code == 303


def _login_admin(client) -> None:  # noqa: ANN001
    settings = get_settings()
    _login(client, username=settings.ui_demo_username, password=settings.ui_demo_password)


def _create_project(client, *, project_id: str) -> None:  # noqa: ANN001
    response = client.post(
        "/api/v1/projects",
        json={
            "project_id": project_id,
            "name": f"Project {project_id}",
            "description": "export tests",
        },
    )
    assert response.status_code == 201


def _create_engram(client, *, project_id: str, title: str) -> str:  # noqa: ANN001
    response = client.post(
        "/api/v1/engrams",
        json={
            "project_id": project_id,
            "title": title,
            "abstract": "",
            "detailed_summary_markdown": f"Summary for {title}",
        },
    )
    assert response.status_code == 200
    return response.json()["engram_id"]


def _create_collection(client, *, project_id: str, name: str) -> str:  # noqa: ANN001
    response = client.post(
        "/api/v1/admin/memory/collections",
        json={"project_id": project_id, "name": name, "description": "ops"},
    )
    assert response.status_code == 201
    return response.json()["collection_id"]


@pytest.mark.integration
def test_project_export_json_with_collection_filter_returns_subset(client, clean_db) -> None:
    _login_admin(client)
    project_id = f"phase33-{uuid.uuid4().hex[:8]}"
    _create_project(client, project_id=project_id)

    engram_a = _create_engram(client, project_id=project_id, title="A")
    engram_b = _create_engram(client, project_id=project_id, title="B")

    collection_id = _create_collection(client, project_id=project_id, name="Subset")
    added = client.post(
        f"/api/v1/admin/memory/collections/{collection_id}/items",
        json={"engram_ids": [engram_a]},
    )
    assert added.status_code == 200
    assert added.json()["added"] == 1

    exported = client.get(
        f"/api/v1/projects/{project_id}/export",
        params={"collection_ids": [collection_id]},
    )
    assert exported.status_code == 200
    assert exported.headers["content-type"].startswith("application/json")

    payload = exported.json()
    assert payload["project"]["project_id"] == project_id
    assert payload["selected_collection_ids"] == [collection_id]
    exported_engram_ids = {item["engram_id"] for item in payload["engrams"]}
    assert exported_engram_ids == {engram_a}
    assert engram_b not in exported_engram_ids


@pytest.mark.integration
def test_project_export_zip_returns_archive(client, clean_db) -> None:
    _login_admin(client)
    project_id = f"phase33-{uuid.uuid4().hex[:8]}"
    _create_project(client, project_id=project_id)
    _create_engram(client, project_id=project_id, title="Zip Engram")

    exported = client.get(
        f"/api/v1/projects/{project_id}/export",
        params={"format": "zip"},
    )
    assert exported.status_code == 200
    assert exported.headers["content-type"].startswith("application/zip")
    assert 'attachment; filename="engram-export-' in exported.headers["content-disposition"]

    with zipfile.ZipFile(BytesIO(exported.content), mode="r") as archive:
        assert archive.namelist() == ["export.json"]
        extracted = archive.read("export.json").decode("utf-8")

    assert project_id in extracted


@pytest.mark.integration
def test_project_export_denies_non_owner_non_admin(client, clean_db) -> None:
    _login_admin(client)
    project_id = f"phase33-{uuid.uuid4().hex[:8]}"
    _create_project(client, project_id=project_id)

    viewer_username = f"viewer_{uuid.uuid4().hex[:8]}"
    created = client.post(
        "/api/v1/users",
        json={
            "username": viewer_username,
            "password": "StrongPass123",
            "role": "viewer",
            "is_active": True,
        },
    )
    assert created.status_code == 201

    from fastapi.testclient import TestClient

    from app.main import app

    viewer_client = TestClient(app)
    _login(viewer_client, username=viewer_username, password="StrongPass123")

    denied = viewer_client.get(f"/api/v1/projects/{project_id}/export")
    assert denied.status_code == 404


@pytest.mark.integration
def test_project_import_json_bundle_with_skip_policy(client, clean_db) -> None:
    _login_admin(client)

    source_project_id = f"phase33-src-{uuid.uuid4().hex[:8]}"
    target_project_id = f"phase33-dst-{uuid.uuid4().hex[:8]}"
    _create_project(client, project_id=source_project_id)
    _create_project(client, project_id=target_project_id)

    source_engram_id = _create_engram(client, project_id=source_project_id, title="Imported A")
    source_collection_id = _create_collection(
        client, project_id=source_project_id, name="Collection A"
    )
    added = client.post(
        f"/api/v1/admin/memory/collections/{source_collection_id}/items",
        json={"engram_ids": [source_engram_id]},
    )
    assert added.status_code == 200

    exported = client.get(f"/api/v1/projects/{source_project_id}/export")
    assert exported.status_code == 200

    imported = client.post(
        f"/api/v1/projects/{target_project_id}/import",
        params={"conflict_policy": "skip"},
        files={"file": ("export.json", exported.content, "application/json")},
    )
    assert imported.status_code == 200
    payload = imported.json()
    assert payload["target_project_id"] == target_project_id
    assert payload["imported_engrams"] == 1
    assert payload["imported_collections"] == 1
    assert payload["imported_collection_items"] == 1
    assert payload["conflict_policy"] == "skip"


@pytest.mark.integration
def test_project_import_zip_bundle_with_rename_policy(client, clean_db) -> None:
    _login_admin(client)

    source_project_id = f"phase33-src-{uuid.uuid4().hex[:8]}"
    target_project_id = f"phase33-dst-{uuid.uuid4().hex[:8]}"
    _create_project(client, project_id=source_project_id)
    _create_project(client, project_id=target_project_id)

    _create_engram(client, project_id=source_project_id, title="Collision")
    _create_engram(client, project_id=target_project_id, title="Collision")

    exported = client.get(
        f"/api/v1/projects/{source_project_id}/export",
        params={"format": "zip"},
    )
    assert exported.status_code == 200

    imported = client.post(
        f"/api/v1/projects/{target_project_id}/import",
        params={"conflict_policy": "rename"},
        files={"file": ("export.zip", exported.content, "application/zip")},
    )
    assert imported.status_code == 200
    payload = imported.json()
    assert payload["imported_engrams"] == 1
    assert payload["conflict_policy"] == "rename"

    listed = client.get(
        "/api/v1/engrams",
        params={"project_id": target_project_id, "limit": 100, "offset": 0},
    )
    assert listed.status_code == 200
    titles = [item["title"] for item in listed.json()]
    assert any(title.startswith("Collision (imported") for title in titles)
