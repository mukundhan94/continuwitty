import uuid

import pytest


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


def test_agent_state_not_found(client) -> None:
    response = client.get("/api/v1/agent-runs/non-existent-thread")
    assert response.status_code == 404
