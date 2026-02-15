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
        },
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

    pinned = client.post(
        f"/api/v1/chat/sessions/{session_id}/engrams/pin",
        json={"engram_id": engram_id},
    )
    assert pinned.status_code == 200
    assert pinned.json()["engram_id"] == engram_id

    listed_pins = client.get(f"/api/v1/chat/sessions/{session_id}/engrams")
    assert listed_pins.status_code == 200
    assert [item["engram_id"] for item in listed_pins.json()] == [engram_id]

    sent = client.post(
        f"/api/v1/chat/sessions/{session_id}/messages",
        json={"content_text": "use prior context"},
    )
    assert sent.status_code == 201
    assert engram_id in sent.json()["used_engram_ids"]
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

    continued_messages = client.get(f"/api/v1/chat/sessions/{continued_session_id}/messages")
    assert continued_messages.status_code == 200
    assert continued_messages.json() == []

    unpinned = client.delete(f"/api/v1/chat/sessions/{session_id}/engrams/{engram_id}")
    assert unpinned.status_code == 200
    assert unpinned.json() == {"removed": True}

    listed_after = client.get(f"/api/v1/chat/sessions/{session_id}/engrams")
    assert listed_after.status_code == 200
    assert listed_after.json() == []


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
