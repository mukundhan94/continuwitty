from __future__ import annotations

import re
import uuid

import pytest

from app.config import get_settings


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


def _login_admin(client) -> None:  # noqa: ANN001
    settings = get_settings()
    _login(client, settings.ui_demo_username, settings.ui_demo_password)


@pytest.mark.integration
def test_admin_memory_session_delete_keeps_or_deletes_linked_engrams(client, clean_db) -> None:
    _login_admin(client)

    session_response = client.post(
        "/api/v1/chat/sessions",
        json={
            "project_id": "engram-vault",
            "title": "Delete policy session",
            "provider": "openai",
            "model_id": "gpt-4o-mini",
            "system_prompt": "",
            "visibility_scope": "private",
            "autosave_enabled": False,
            "autosave_strategy": "off",
            "autosave_interval_minutes": 30,
            "autosave_min_messages": 6,
            "retention_days": 30,
            "retention_max_snapshots": 60,
        },
    )
    assert session_response.status_code == 201
    session_id = session_response.json()["session_id"]

    linked_engram = client.post(
        "/api/v1/engrams",
        json={
            "project_id": "engram-vault",
            "source_session_id": session_id,
            "title": "Linked snapshot",
            "abstract": "Linked engram for delete behavior.",
            "detailed_summary_markdown": "Linked markdown",
        },
    )
    assert linked_engram.status_code == 200
    linked_engram_id = linked_engram.json()["engram_id"]

    delete_keep = client.request(
        "DELETE",
        f"/api/v1/admin/memory/sessions/{session_id}",
        json={"delete_linked_engrams": False, "reason": "session cleanup"},
    )
    assert delete_keep.status_code == 200
    assert delete_keep.json()["linked_engrams_deleted"] == 0

    still_visible = client.get(
        "/api/v1/admin/memory/engrams",
        params={"session_id": session_id, "include_deleted": "false"},
    )
    assert still_visible.status_code == 200
    assert any(item["engram_id"] == linked_engram_id for item in still_visible.json())

    session_response_2 = client.post(
        "/api/v1/chat/sessions",
        json={
            "project_id": "engram-vault",
            "title": "Delete policy session 2",
            "provider": "openai",
            "model_id": "gpt-4o-mini",
            "system_prompt": "",
            "visibility_scope": "private",
            "autosave_enabled": False,
            "autosave_strategy": "off",
            "autosave_interval_minutes": 30,
            "autosave_min_messages": 6,
            "retention_days": 30,
            "retention_max_snapshots": 60,
        },
    )
    assert session_response_2.status_code == 201
    session_id_2 = session_response_2.json()["session_id"]
    linked_engram_2 = client.post(
        "/api/v1/engrams",
        json={
            "project_id": "engram-vault",
            "source_session_id": session_id_2,
            "title": "Linked snapshot delete-all",
            "abstract": "Linked engram that should be deleted.",
            "detailed_summary_markdown": "Linked markdown 2",
        },
    )
    assert linked_engram_2.status_code == 200
    linked_engram_id_2 = linked_engram_2.json()["engram_id"]

    delete_linked = client.request(
        "DELETE",
        f"/api/v1/admin/memory/sessions/{session_id_2}",
        json={"delete_linked_engrams": True, "reason": "deep cleanup"},
    )
    assert delete_linked.status_code == 200
    assert delete_linked.json()["linked_engrams_deleted"] >= 1

    deleted_list = client.get(
        "/api/v1/admin/memory/engrams",
        params={"session_id": session_id_2, "include_deleted": "true"},
    )
    assert deleted_list.status_code == 200
    target = next(item for item in deleted_list.json() if item["engram_id"] == linked_engram_id_2)
    assert target["deleted_at"] is not None


@pytest.mark.integration
def test_admin_memory_engram_edit_move_and_collection_detach(client, clean_db, db_conn) -> None:
    _login_admin(client)

    created_project = client.post(
        "/api/v1/projects",
        json={
            "project_id": "phase31-target",
            "name": "Phase 31 Target",
            "description": "Move target project.",
        },
    )
    assert created_project.status_code == 201

    created_engram = client.post(
        "/api/v1/engrams",
        json={
            "project_id": "engram-vault",
            "title": "Editable engram",
            "abstract": "old abstract",
            "detailed_summary_markdown": "old markdown",
        },
    )
    assert created_engram.status_code == 200
    engram_id = created_engram.json()["engram_id"]

    detail = client.get(f"/api/v1/admin/memory/engrams/{engram_id}")
    assert detail.status_code == 200
    expected_updated_at = detail.json()["updated_at"]

    updated = client.patch(
        f"/api/v1/admin/memory/engrams/{engram_id}",
        json={
            "title": "Editable engram updated",
            "abstract": "new abstract",
            "detailed_summary_markdown": "new markdown",
            "tags": ["phase31"],
            "keywords": ["admin"],
            "visibility_scope": "private",
            "sources": [
                {
                    "captured_at": "2026-02-19T00:00:00Z",
                    "url": "https://example.com/phase31",
                    "title": "Phase 31",
                    "snippet": "Admin edit source",
                }
            ],
            "expected_updated_at": expected_updated_at,
        },
    )
    assert updated.status_code == 200
    assert updated.json()["title"] == "Editable engram updated"
    assert len(updated.json()["sources"]) == 1

    stale_update = client.patch(
        f"/api/v1/admin/memory/engrams/{engram_id}",
        json={"title": "stale", "expected_updated_at": expected_updated_at},
    )
    assert stale_update.status_code == 409

    collection = client.post(
        "/api/v1/admin/memory/collections",
        json={"project_id": "engram-vault", "name": "Ops Collection", "description": "ops"},
    )
    assert collection.status_code == 201
    collection_id = collection.json()["collection_id"]

    add_item = client.post(
        f"/api/v1/admin/memory/collections/{collection_id}/items",
        json={"engram_ids": [engram_id]},
    )
    assert add_item.status_code == 200
    assert add_item.json()["added"] >= 1

    moved = client.post(
        f"/api/v1/admin/memory/engrams/{engram_id}/move",
        json={"target_project_id": "phase31-target"},
    )
    assert moved.status_code == 200
    assert moved.json()["project_id"] == "phase31-target"

    with db_conn.cursor() as cur:
        cur.execute(
            """
            SELECT COUNT(*)::INT
            FROM engram_collection_items
            WHERE engram_id = %s::UUID
            """,
            (engram_id,),
        )
        assert cur.fetchone()[0] == 0


@pytest.mark.integration
def test_admin_memory_routes_are_admin_only(client, clean_db) -> None:
    _login_admin(client)
    viewer_username = f"viewer_{uuid.uuid4().hex[:8]}"
    create_viewer = client.post(
        "/api/v1/users",
        json={
            "username": viewer_username,
            "password": "StrongPass123",
            "role": "viewer",
            "is_active": True,
        },
    )
    assert create_viewer.status_code == 201

    from fastapi.testclient import TestClient

    from app.main import app  # local import to avoid fixture side effects

    viewer_client = TestClient(app)
    _login(viewer_client, viewer_username, "StrongPass123")

    denied = viewer_client.get("/api/v1/admin/memory/sessions")
    assert denied.status_code == 403
