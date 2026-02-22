from __future__ import annotations

import re
import uuid
from typing import Any

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


def _create_chat_session(client, *, title: str) -> str:  # noqa: ANN001
    response = client.post(
        "/api/v1/chat/sessions",
        json={
            "project_id": "engram-vault",
            "title": title,
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
    assert response.status_code == 201
    return response.json()["session_id"]


def _delete_admin_session(
    client,
    *,
    session_id: str,
    delete_linked_engrams: bool,
    reason: str,
) -> dict[str, Any]:  # noqa: ANN001
    response = client.request(
        "DELETE",
        f"/api/v1/admin/memory/sessions/{session_id}",
        json={"delete_linked_engrams": delete_linked_engrams, "reason": reason},
    )
    assert response.status_code == 200
    return response.json()


def _list_session_engrams(
    client,
    *,
    session_id: str,
    include_deleted: bool,
) -> list[dict[str, Any]]:  # noqa: ANN001
    response = client.get(
        "/api/v1/admin/memory/engrams",
        params={
            "session_id": session_id,
            "include_deleted": str(include_deleted).lower(),
        },
    )
    assert response.status_code == 200
    return response.json()


def _create_project(client, *, project_id: str) -> None:  # noqa: ANN001
    response = client.post(
        "/api/v1/projects",
        json={
            "project_id": project_id,
            "name": f"Project {project_id}",
            "description": "Move target project.",
        },
    )
    assert response.status_code == 201


def _create_engram(client, *, project_id: str, title: str) -> str:  # noqa: ANN001
    return _create_engram_record(
        client,
        project_id=project_id,
        title=title,
        source_session_id=None,
    )


def _create_engram_record(
    client,
    *,
    project_id: str,
    title: str,
    source_session_id: str | None,
) -> str:  # noqa: ANN001
    payload: dict[str, Any] = {
        "project_id": project_id,
        "title": title,
        "abstract": "old abstract",
        "detailed_summary_markdown": "old markdown",
    }
    if source_session_id is not None:
        payload["source_session_id"] = source_session_id

    response = client.post(
        "/api/v1/engrams",
        json=payload,
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


def _assert_collection_detached(db_conn, *, engram_id: str) -> None:  # noqa: ANN001
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
@pytest.mark.parametrize("delete_linked_engrams", [False, True])
def test_admin_memory_session_delete_applies_linked_engram_policy(
    client,
    clean_db,
    delete_linked_engrams: bool,
) -> None:
    _login_admin(client)
    suffix = "delete" if delete_linked_engrams else "keep"
    session_id = _create_chat_session(client, title=f"Delete policy session {suffix}")
    linked_engram_id = _create_engram_record(
        client,
        project_id="engram-vault",
        source_session_id=session_id,
        title=f"Linked snapshot {suffix}",
    )

    deleted = _delete_admin_session(
        client,
        session_id=session_id,
        delete_linked_engrams=delete_linked_engrams,
        reason="deep cleanup" if delete_linked_engrams else "session cleanup",
    )

    if delete_linked_engrams:
        assert deleted["linked_engrams_deleted"] >= 1
        deleted_list = _list_session_engrams(
            client,
            session_id=session_id,
            include_deleted=True,
        )
        target = next(item for item in deleted_list if item["engram_id"] == linked_engram_id)
        assert target["deleted_at"] is not None
        return

    assert deleted["linked_engrams_deleted"] == 0
    visible = _list_session_engrams(
        client,
        session_id=session_id,
        include_deleted=False,
    )
    assert any(item["engram_id"] == linked_engram_id for item in visible)


@pytest.mark.integration
def test_admin_memory_engram_edit_enforces_optimistic_lock(client, clean_db) -> None:
    _login_admin(client)
    engram_id = _create_engram(
        client,
        project_id="engram-vault",
        title="Editable engram",
    )

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


@pytest.mark.integration
def test_admin_memory_engram_move_detaches_collection_items(client, clean_db, db_conn) -> None:
    _login_admin(client)
    _create_project(client, project_id="phase31-target")
    engram_id = _create_engram(
        client,
        project_id="engram-vault",
        title="Editable engram",
    )
    collection_id = _create_collection(
        client,
        project_id="engram-vault",
        name="Ops Collection",
    )

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

    _assert_collection_detached(db_conn, engram_id=engram_id)


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
