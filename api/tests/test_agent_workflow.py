import uuid

import pytest

from app.config import get_settings


def _login(client) -> None:  # noqa: ANN001
    login_page = client.get("/login")
    csrf_token = login_page.text.split('name="csrf_token" value="', 1)[1].split('"', 1)[0]
    settings = get_settings()
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
def test_agent_run_creates_engram_and_checkpoint(client, clean_db) -> None:
    thread_id = f"thread-{uuid.uuid4()}"

    run_response = client.post(
        "/api/v1/agent-runs",
        json={
            "project_id": "project-agent",
            "thread_id": thread_id,
            "objective": "Evaluate durable memory checkpointing",
            "notes": ["Use LangGraph for resumable threads."],
            "assumptions": ["Local single-user mode"],
            "tags": ["langgraph"],
            "keywords": ["checkpoint"],
            "sources": [
                {
                    "url": "https://example.com/langgraph",
                    "title": "LangGraph docs",
                    "snippet": "Checkpointed runs can resume by thread id.",
                    "captured_at": "2026-02-15T10:00:00Z",
                }
            ],
            "auto_persist_engram": True,
        },
    )
    assert run_response.status_code == 200
    run_body = run_response.json()
    assert run_body["thread_id"] == thread_id
    assert run_body["status"] == "completed"
    assert run_body["engram_id"] is not None
    assert run_body["snapshot_engram_ids"] == []

    state_response = client.get(f"/api/v1/agent-runs/{thread_id}")
    assert state_response.status_code == 200
    state_body = state_response.json()
    assert state_body["thread_id"] == thread_id
    assert state_body["state"]["objective"] == "Evaluate durable memory checkpointing"
    assert state_body["engram_id"] == run_body["engram_id"]


@pytest.mark.integration
def test_agent_resume_appends_notes(client, clean_db) -> None:
    thread_id = f"thread-{uuid.uuid4()}"

    first = client.post(
        "/api/v1/agent-runs",
        json={
            "project_id": "project-resume",
            "thread_id": thread_id,
            "objective": "Resume test objective",
            "notes": ["Initial note"],
            "auto_persist_engram": False,
        },
    )
    assert first.status_code == 200
    assert first.json()["status"] == "completed"
    assert first.json()["engram_id"] is None

    resumed = client.post(
        f"/api/v1/agent-runs/{thread_id}/resume",
        json={
            "notes": ["Follow-up note from resumed run"],
            "assumptions": ["Assume continuous context"],
            "auto_persist_engram": False,
        },
    )
    assert resumed.status_code == 200
    body = resumed.json()
    assert body["status"] == "completed"
    assert "Initial note" in body["state"]["notes"]
    assert "Follow-up note from resumed run" in body["state"]["notes"]


@pytest.mark.integration
def test_snapshot_creation_on_note_threshold(client, clean_db) -> None:
    _login(client)
    thread_id = f"thread-{uuid.uuid4()}"

    run_response = client.post(
        "/api/v1/agent-runs",
        json={
            "project_id": "project-snapshot",
            "thread_id": thread_id,
            "objective": "Capture periodic snapshots",
            "notes": ["note-1", "note-2", "note-3"],
            "snapshot_enabled": True,
            "snapshot_every_n_notes": 2,
            "auto_persist_engram": False,
        },
    )
    assert run_response.status_code == 200
    run_body = run_response.json()
    assert run_body["engram_id"] is None
    assert len(run_body["snapshot_engram_ids"]) == 1

    snapshot_id = run_body["snapshot_engram_ids"][0]
    rehydrate_response = client.get(f"/api/v1/engrams/{snapshot_id}/rehydrate")
    assert rehydrate_response.status_code == 200
    assert "Snapshot" in rehydrate_response.json()["title"]


@pytest.mark.integration
def test_snapshot_creation_continues_across_resume(client, clean_db) -> None:
    thread_id = f"thread-{uuid.uuid4()}"

    first = client.post(
        "/api/v1/agent-runs",
        json={
            "project_id": "project-snapshot-resume",
            "thread_id": thread_id,
            "objective": "Resume with snapshots",
            "notes": ["note-1"],
            "snapshot_enabled": True,
            "snapshot_every_n_notes": 2,
            "auto_persist_engram": False,
        },
    )
    assert first.status_code == 200
    assert first.json()["snapshot_engram_ids"] == []

    resumed_one = client.post(
        f"/api/v1/agent-runs/{thread_id}/resume",
        json={
            "notes": ["note-2"],
            "auto_persist_engram": False,
        },
    )
    assert resumed_one.status_code == 200
    first_snapshots = resumed_one.json()["snapshot_engram_ids"]
    assert len(first_snapshots) == 1

    resumed_two = client.post(
        f"/api/v1/agent-runs/{thread_id}/resume",
        json={
            "notes": ["note-3", "note-4"],
            "auto_persist_engram": False,
        },
    )
    assert resumed_two.status_code == 200
    second_snapshots = resumed_two.json()["snapshot_engram_ids"]
    assert len(second_snapshots) == 2
    assert set(first_snapshots).issubset(set(second_snapshots))


def test_agent_state_not_found(client) -> None:
    response = client.get("/api/v1/agent-runs/non-existent-thread")
    assert response.status_code == 404


def test_agent_resume_state_not_found(client) -> None:
    response = client.post(
        "/api/v1/agent-runs/non-existent-thread/resume",
        json={"notes": ["note"]},
    )
    assert response.status_code == 404
