from __future__ import annotations

import re
import uuid

import pytest

from app.config import get_settings
from app.models import ChatProvider
from app.providers.base import ProviderGenerateResult


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


def _install_fake_provider(monkeypatch) -> None:
    class _FakeAdapter:
        def generate(self, request):  # noqa: ANN001
            last_user = ""
            for message in reversed(request.messages):
                if message.role == "user":
                    last_user = message.content
                    break
            return ProviderGenerateResult(
                provider=ChatProvider.openai,
                model_id=request.model_id,
                text=f"assistant:{last_user}",
                token_usage={"input_tokens": 5, "output_tokens": 3, "total_tokens": 8},
            )

        def stream_generate(self, request):  # noqa: ANN001
            _ = request
            yield "assistant:"
            yield "streamed"

    monkeypatch.setattr("app.chat.service.get_provider_adapter", lambda provider: _FakeAdapter())


def _create_session(client, project_id: str = "project-chat") -> dict:  # noqa: ANN001
    response = client.post(
        "/api/v1/chat/sessions",
        json={
            "project_id": project_id,
            "title": "Session A",
            "provider": "openai",
            "model_id": "gpt-4o-mini",
            "system_prompt": "You are concise.",
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
    return response.json()


def _send_long_message(client, session_id: str, content_text: str) -> dict:  # noqa: ANN001
    response = client.post(
        f"/api/v1/chat/sessions/{session_id}/messages",
        json={"content_text": content_text},
    )
    assert response.status_code == 201
    return response.json()


@pytest.mark.integration
def test_chat_api_requires_authentication(client, clean_db) -> None:
    response = client.get("/api/v1/chat/sessions")
    assert response.status_code == 401


@pytest.mark.integration
def test_chat_api_session_message_and_stream_flow(client, clean_db, monkeypatch) -> None:
    _login(client)
    _install_fake_provider(monkeypatch)
    created = _create_session(client)
    session_id = created["session_id"]

    listed = client.get("/api/v1/chat/sessions", params={"project_id": "project-chat"})
    assert listed.status_code == 200
    assert any(item["session_id"] == session_id for item in listed.json())

    fetched = client.get(f"/api/v1/chat/sessions/{session_id}")
    assert fetched.status_code == 200
    assert fetched.json()["title"] == "Session A"

    updated = client.patch(
        f"/api/v1/chat/sessions/{session_id}", json={"title": "Session A Updated"}
    )
    assert updated.status_code == 200
    assert updated.json()["title"] == "Session A Updated"

    sent = client.post(
        f"/api/v1/chat/sessions/{session_id}/messages",
        json={"content_text": "hello"},
    )
    assert sent.status_code == 201
    sent_body = sent.json()
    assert sent_body["assistant_text"] == "assistant:hello"
    assert sent_body["used_engram_ids"] == []

    messages = client.get(f"/api/v1/chat/sessions/{session_id}/messages")
    assert messages.status_code == 200
    roles = [item["role"] for item in messages.json()]
    assert roles == ["user", "assistant"]

    streamed = client.post(
        f"/api/v1/chat/sessions/{session_id}/messages/stream",
        json={"content_text": "stream"},
    )
    assert streamed.status_code == 200
    assert "text/event-stream" in streamed.headers.get("content-type", "")
    assert "event: meta" in streamed.text
    assert "event: chunk" in streamed.text
    assert "event: done" in streamed.text
    assert "assistant:streamed" in streamed.text

    final_messages = client.get(f"/api/v1/chat/sessions/{session_id}/messages")
    assert final_messages.status_code == 200
    assert len(final_messages.json()) == 4


@pytest.mark.integration
def test_chat_api_pin_save_and_continue_flow(client, clean_db, monkeypatch) -> None:
    _login(client)
    _install_fake_provider(monkeypatch)
    session = _create_session(client)
    session_id = session["session_id"]

    engram_response = client.post(
        "/api/v1/engrams",
        json={
            "project_id": "project-chat",
            "thread_id": "seed-thread",
            "title": "Pinned context",
            "abstract": "Important context",
            "detailed_summary_markdown": "Pinned context details",
        },
    )
    assert engram_response.status_code == 200
    engram_id = engram_response.json()["engram_id"]

    document_response = client.post(
        "/api/v1/ingestion/text",
        json={
            "project_id": "project-chat",
            "title": "Incident Runbook",
            "text": "Queue depth crossed 5k for 20 minutes. Drain policy: prioritize webhook retries.",
            "visibility_scope": "project",
        },
    )
    assert document_response.status_code == 201
    document_id = document_response.json()["document"]["document_id"]
    second_document_response = client.post(
        "/api/v1/ingestion/text",
        json={
            "project_id": "project-chat",
            "title": "Escalation Matrix",
            "text": "Escalate to DB oncall after 15 minutes and notify support incident commander.",
            "visibility_scope": "project",
        },
    )
    assert second_document_response.status_code == 201
    second_document_id = second_document_response.json()["document"]["document_id"]

    pinned = client.post(
        f"/api/v1/chat/sessions/{session_id}/engrams/pin",
        json={"engram_id": engram_id},
    )
    assert pinned.status_code == 200
    assert pinned.json()["engram_id"] == engram_id

    listed_pins = client.get(f"/api/v1/chat/sessions/{session_id}/engrams")
    assert listed_pins.status_code == 200
    assert [item["engram_id"] for item in listed_pins.json()] == [engram_id]

    pinned_document = client.post(
        f"/api/v1/chat/sessions/{session_id}/documents/pin",
        json={"document_id": document_id},
    )
    assert pinned_document.status_code == 200
    assert pinned_document.json()["document_id"] == document_id
    pinned_second_document = client.post(
        f"/api/v1/chat/sessions/{session_id}/documents/pin",
        json={"document_id": second_document_id},
    )
    assert pinned_second_document.status_code == 200
    assert pinned_second_document.json()["document_id"] == second_document_id

    listed_documents = client.get(f"/api/v1/chat/sessions/{session_id}/documents")
    assert listed_documents.status_code == 200
    assert {item["document_id"] for item in listed_documents.json()} == {
        document_id,
        second_document_id,
    }

    sent = client.post(
        f"/api/v1/chat/sessions/{session_id}/messages",
        json={"content_text": "use prior context"},
    )
    assert sent.status_code == 201
    assert engram_id in sent.json()["used_engram_ids"]
    assert len(sent.json()["used_document_chunk_ids"]) >= 2
    assert any(
        ref.get("source_type") == "document_chunk" for ref in sent.json()["source_references"]
    )
    assert isinstance(sent.json()["source_references"], list)

    saved = client.post(
        f"/api/v1/chat/sessions/{session_id}/save-engram",
        json={
            "title": "Saved session snapshot",
            "abstract": "Captured from chat flow",
            "visibility_scope": "private",
            "tags": ["chat"],
            "keywords": ["continuity"],
        },
    )
    assert saved.status_code == 201
    saved_body = saved.json()
    assert saved_body["session_id"] == session_id

    continued = client.post(
        f"/api/v1/chat/sessions/{session_id}/continue",
        json={"title": "Session A Continued"},
    )
    assert continued.status_code == 201
    body = continued.json()
    continued_session_id = body["session"]["session_id"]
    assert continued_session_id != session_id
    assert engram_id in body["carried_engram_ids"]

    continued_documents = client.get(f"/api/v1/chat/sessions/{continued_session_id}/documents")
    assert continued_documents.status_code == 200
    assert {item["document_id"] for item in continued_documents.json()} == {
        document_id,
        second_document_id,
    }

    continued_messages = client.get(f"/api/v1/chat/sessions/{continued_session_id}/messages")
    assert continued_messages.status_code == 200
    assert continued_messages.json() == []

    unpinned = client.delete(f"/api/v1/chat/sessions/{session_id}/engrams/{engram_id}")
    assert unpinned.status_code == 200
    assert unpinned.json() == {"removed": True}

    listed_after = client.get(f"/api/v1/chat/sessions/{session_id}/engrams")
    assert listed_after.status_code == 200
    assert listed_after.json() == []

    unpinned_document = client.delete(f"/api/v1/chat/sessions/{session_id}/documents/{document_id}")
    assert unpinned_document.status_code == 200
    assert unpinned_document.json() == {"removed": True}
    unpinned_second_document = client.delete(
        f"/api/v1/chat/sessions/{session_id}/documents/{second_document_id}"
    )
    assert unpinned_second_document.status_code == 200
    assert unpinned_second_document.json() == {"removed": True}

    listed_documents_after = client.get(f"/api/v1/chat/sessions/{session_id}/documents")
    assert listed_documents_after.status_code == 200
    assert listed_documents_after.json() == []


@pytest.mark.integration
def test_chat_save_engram_auto_enriches_when_metadata_is_empty(
    client,
    clean_db,
    monkeypatch,
) -> None:
    _login(client)
    _install_fake_provider(monkeypatch)
    session = _create_session(client, project_id="project-chat-auto-meta")
    session_id = session["session_id"]

    sent = client.post(
        f"/api/v1/chat/sessions/{session_id}/messages",
        json={
            "content_text": (
                "Prepare a detailed incident update with outage impact, mitigation status, "
                "rollback outcome, and support handoff actions."
            )
        },
    )
    assert sent.status_code == 201

    title = f"Auto metadata save {uuid.uuid4()}"
    saved = client.post(
        f"/api/v1/chat/sessions/{session_id}/save-engram",
        json={
            "title": title,
            "abstract": "",
            "visibility_scope": "project",
            "tags": [],
            "keywords": [],
        },
    )
    assert saved.status_code == 201
    engram_id = saved.json()["engram_id"]

    listed = client.get("/api/v1/engrams", params={"project_id": "project-chat-auto-meta"})
    assert listed.status_code == 200
    created = next(item for item in listed.json() if item["engram_id"] == engram_id)
    assert created["title"] == title
    assert created["abstract"].strip() != ""
    assert len(created["tags"]) > 0
    assert len(created["keywords"]) > 0


@pytest.mark.integration
def test_chat_api_reconciles_stale_session_user_id_after_db_reset(
    client,
    clean_db,
    db_conn,
) -> None:
    _login(client)

    settings = get_settings()
    replacement_user_id = uuid.uuid4()
    with db_conn.cursor() as cur:
        cur.execute(
            """
            UPDATE users
            SET user_id = %s
            WHERE username = %s
            """,
            (replacement_user_id, settings.ui_demo_username),
        )
    db_conn.commit()

    created = _create_session(client, project_id="project-session-reconcile")
    assert created["project_id"] == "project-session-reconcile"

    me = client.get("/api/v1/me")
    assert me.status_code == 200
    assert me.json()["user_id"] == str(replacement_user_id)


@pytest.mark.integration
def test_chat_lifecycle_policy_and_timeline_flow(client, clean_db, monkeypatch) -> None:
    _login(client)
    _install_fake_provider(monkeypatch)
    created = _create_session(client, project_id="project-lifecycle")
    session_id = created["session_id"]

    policy = client.get(f"/api/v1/chat/sessions/{session_id}/lifecycle-policy")
    assert policy.status_code == 200
    assert policy.json()["autosave_strategy"] == "off"
    assert policy.json()["autosave_enabled"] is False

    updated_policy = client.patch(
        f"/api/v1/chat/sessions/{session_id}/lifecycle-policy",
        json={
            "autosave_enabled": True,
            "autosave_strategy": "interval",
            "autosave_interval_minutes": 1,
            "retention_days": 7,
            "retention_max_snapshots": 10,
        },
    )
    assert updated_policy.status_code == 200
    assert updated_policy.json()["autosave_enabled"] is True
    assert updated_policy.json()["autosave_strategy"] == "interval"
    assert updated_policy.json()["autosave_interval_minutes"] == 1

    sent = client.post(
        f"/api/v1/chat/sessions/{session_id}/messages",
        json={
            "content_text": (
                "Create a detailed status summary covering impact, timeline, root cause, and next steps."
            )
        },
    )
    assert sent.status_code == 201

    timeline = client.get(f"/api/v1/chat/sessions/{session_id}/timeline")
    assert timeline.status_code == 200
    events = timeline.json()
    assert any(item["event_type"] == "autosave_snapshot" for item in events)


@pytest.mark.integration
def test_chat_lifecycle_policy_off_does_not_autosave(client, clean_db, monkeypatch) -> None:
    _login(client)
    _install_fake_provider(monkeypatch)
    created = _create_session(client, project_id="project-lifecycle-off")
    session_id = created["session_id"]

    _send_long_message(
        client,
        session_id,
        (
            "Draft a full incident summary with impact, timeline, root cause, mitigation, "
            "and follow-up actions for tomorrow's executive review."
        ),
    )

    timeline = client.get(f"/api/v1/chat/sessions/{session_id}/timeline")
    assert timeline.status_code == 200
    autosave_events = [
        item for item in timeline.json() if item["event_type"] == "autosave_snapshot"
    ]
    assert autosave_events == []


@pytest.mark.integration
def test_chat_lifecycle_policy_message_count_autosaves_on_threshold(
    client,
    clean_db,
    monkeypatch,
) -> None:
    _login(client)
    _install_fake_provider(monkeypatch)
    created = _create_session(client, project_id="project-lifecycle-message-count")
    session_id = created["session_id"]

    updated_policy = client.patch(
        f"/api/v1/chat/sessions/{session_id}/lifecycle-policy",
        json={
            "autosave_enabled": True,
            "autosave_strategy": "message_count",
            "autosave_min_messages": 2,
            "retention_days": 7,
            "retention_max_snapshots": 10,
        },
    )
    assert updated_policy.status_code == 200

    _send_long_message(
        client,
        session_id,
        (
            "Produce a detailed customer impact summary with observed failure patterns, "
            "regional spread, and first-pass mitigation notes."
        ),
    )
    timeline_after_first = client.get(f"/api/v1/chat/sessions/{session_id}/timeline")
    assert timeline_after_first.status_code == 200
    first_autosave_events = [
        item for item in timeline_after_first.json() if item["event_type"] == "autosave_snapshot"
    ]
    assert first_autosave_events == []

    _send_long_message(
        client,
        session_id,
        (
            "Now generate the escalation-ready remediation plan with ownership, rollback checkpoints, "
            "and monitoring verification criteria."
        ),
    )
    timeline_after_second = client.get(f"/api/v1/chat/sessions/{session_id}/timeline")
    assert timeline_after_second.status_code == 200
    second_autosave_events = [
        item for item in timeline_after_second.json() if item["event_type"] == "autosave_snapshot"
    ]
    assert len(second_autosave_events) >= 1


@pytest.mark.integration
def test_chat_lifecycle_policy_interval_autosaves_then_waits(client, clean_db, monkeypatch) -> None:
    _login(client)
    _install_fake_provider(monkeypatch)
    created = _create_session(client, project_id="project-lifecycle-interval")
    session_id = created["session_id"]

    updated_policy = client.patch(
        f"/api/v1/chat/sessions/{session_id}/lifecycle-policy",
        json={
            "autosave_enabled": True,
            "autosave_strategy": "interval",
            "autosave_interval_minutes": 60,
            "retention_days": 7,
            "retention_max_snapshots": 10,
        },
    )
    assert updated_policy.status_code == 200

    _send_long_message(
        client,
        session_id,
        (
            "Summarize the current incident with timeline checkpoints, blast radius, and "
            "validation tasks for on-call handoff."
        ),
    )
    timeline_after_first = client.get(f"/api/v1/chat/sessions/{session_id}/timeline")
    assert timeline_after_first.status_code == 200
    first_autosave_events = [
        item for item in timeline_after_first.json() if item["event_type"] == "autosave_snapshot"
    ]
    assert len(first_autosave_events) == 1

    _send_long_message(
        client,
        session_id,
        (
            "Prepare the next status update in a different tone but keep the same factual grounding "
            "for incident communication."
        ),
    )
    timeline_after_second = client.get(f"/api/v1/chat/sessions/{session_id}/timeline")
    assert timeline_after_second.status_code == 200
    second_autosave_events = [
        item for item in timeline_after_second.json() if item["event_type"] == "autosave_snapshot"
    ]
    assert len(second_autosave_events) == 1
