from __future__ import annotations

from datetime import UTC, datetime
from uuid import uuid4

import pytest

from app.chat.context import AssembledChatContext
from app.chat.errors import ChatProviderExecutionError, ChatValidationError
from app.chat.service import ChatService
from app.models import (
    ChatMessageCreateRequest,
    ChatMessageRecord,
    ChatProvider,
    ChatSessionCreateRequest,
    ChatSessionRecord,
    ContinueSessionRequest,
    EngramCreateResponse,
    PinEngramRequest,
    PinnedEngramRecord,
    SaveSessionAsEngramRequest,
    VisibilityScope,
)
from app.providers.base import ProviderGenerateResult
from app.providers.errors import ProviderRequestError


def _session(owner_user_id) -> ChatSessionRecord:
    now = datetime.now(UTC)
    return ChatSessionRecord(
        session_id=uuid4(),
        owner_user_id=owner_user_id,
        project_id="project-chat",
        title="Session",
        provider=ChatProvider.openai,
        model_id="gpt-4o-mini",
        system_prompt="be helpful",
        visibility_scope=VisibilityScope.private,
        autosave_enabled=False,
        created_at=now,
        updated_at=now,
    )


def _message(
    *,
    message_id,
    session_id,
    role: str,
    content_text: str,
) -> ChatMessageRecord:
    return ChatMessageRecord(
        message_id=message_id,
        session_id=session_id,
        role=role,
        content_text=content_text,
        provider=None,
        model_id=None,
        token_usage_json={},
        used_engram_ids=[],
        created_at=datetime.now(UTC),
    )


def test_send_message_returns_used_engram_ids_and_sources(monkeypatch) -> None:
    actor_id = uuid4()
    session = _session(actor_id)
    user_message_id = uuid4()
    reply_message_id = uuid4()
    referenced_engram_id = uuid4()

    service = ChatService(embedding_dim=256)
    user_message = _message(
        message_id=user_message_id,
        session_id=session.session_id,
        role="user",
        content_text="How should we proceed?",
    )
    assistant_message = _message(
        message_id=reply_message_id,
        session_id=session.session_id,
        role="assistant",
        content_text="Proceed with option A.",
    )

    monkeypatch.setattr(
        "app.chat.service.get_chat_session", lambda session_id, actor_user_id: session
    )
    monkeypatch.setattr(
        "app.chat.service.assemble_chat_context",
        lambda **kwargs: AssembledChatContext(
            context_markdown="ctx",
            used_engram_ids=[referenced_engram_id],
            used_document_chunk_ids=[],
            source_references=[
                {
                    "engram_id": referenced_engram_id,
                    "engram_title": "Referenced",
                    "url": "https://example.com/source",
                    "title": "Source",
                    "snippet": "Snippet",
                    "captured_at": datetime.now(UTC),
                }
            ],
        ),
    )
    monkeypatch.setattr("app.chat.service.list_chat_messages", lambda **kwargs: [user_message])

    def _fake_create_chat_message(**kwargs):
        if kwargs["role"] == "user":
            return user_message
        assert kwargs["used_engram_ids"] == [referenced_engram_id]
        return assistant_message

    monkeypatch.setattr("app.chat.service.create_chat_message", _fake_create_chat_message)

    class _FakeAdapter:
        def generate(self, request):  # noqa: ANN001
            return ProviderGenerateResult(
                provider=ChatProvider.openai,
                model_id=request.model_id,
                text="Proceed with option A.",
                token_usage={"input_tokens": 9, "output_tokens": 4, "total_tokens": 13},
            )

    monkeypatch.setattr("app.chat.service.get_provider_adapter", lambda provider: _FakeAdapter())

    response = service.send_message(
        actor_user_id=actor_id,
        session_id=session.session_id,
        payload=ChatMessageCreateRequest(content_text="How should we proceed?"),
    )

    assert response.session_id == session.session_id
    assert response.message_id == user_message_id
    assert response.reply_message_id == reply_message_id
    assert response.assistant_text == "Proceed with option A."
    assert response.used_engram_ids == [referenced_engram_id]
    assert response.used_document_chunk_ids == []
    assert response.source_references[0].url == "https://example.com/source"


def test_continue_session_copies_pinned_engrams(monkeypatch) -> None:
    actor_id = uuid4()
    session = _session(actor_id)
    continued = _session(actor_id)
    engram_a = uuid4()
    engram_b = uuid4()
    service = ChatService(embedding_dim=256)

    monkeypatch.setattr(
        "app.chat.service.get_chat_session", lambda session_id, actor_user_id: session
    )
    monkeypatch.setattr(
        "app.chat.service.create_chat_session",
        lambda owner_user_id, payload: ChatSessionRecord(
            **(
                continued.model_dump()
                | {
                    "title": payload.title,
                    "project_id": payload.project_id,
                    "provider": payload.provider,
                    "model_id": payload.model_id,
                    "system_prompt": payload.system_prompt,
                    "visibility_scope": payload.visibility_scope,
                    "autosave_enabled": payload.autosave_enabled,
                }
            )
        ),
    )
    monkeypatch.setattr(
        "app.chat.service.list_pinned_engrams",
        lambda session_id, actor_user_id: [
            PinnedEngramRecord(
                session_id=session.session_id,
                engram_id=engram_a,
                pinned_by_user_id=actor_id,
                created_at=datetime.now(UTC),
            ),
            PinnedEngramRecord(
                session_id=session.session_id,
                engram_id=engram_b,
                pinned_by_user_id=actor_id,
                created_at=datetime.now(UTC),
            ),
        ],
    )
    monkeypatch.setattr(
        "app.chat.service.pin_engram_to_session",
        lambda session_id, engram_id, actor_user_id: PinnedEngramRecord(
            session_id=session_id,
            engram_id=engram_id,
            pinned_by_user_id=actor_user_id,
            created_at=datetime.now(UTC),
        ),
    )

    response = service.continue_session(
        actor_user_id=actor_id,
        session_id=session.session_id,
        payload=ContinueSessionRequest(title="Continued Session"),
    )

    assert response.session.title == "Continued Session"
    assert response.carried_engram_ids == [engram_a, engram_b]


def test_save_session_as_engram_sets_source_session_id(monkeypatch) -> None:
    actor_id = uuid4()
    session = _session(actor_id)
    service = ChatService(embedding_dim=256)
    message_a = _message(
        message_id=uuid4(),
        session_id=session.session_id,
        role="user",
        content_text="Question",
    )
    message_b = _message(
        message_id=uuid4(),
        session_id=session.session_id,
        role="assistant",
        content_text="Answer",
    )
    captured: dict = {}

    monkeypatch.setattr(
        "app.chat.service.get_chat_session", lambda session_id, actor_user_id: session
    )
    monkeypatch.setattr(
        "app.chat.service.list_chat_messages",
        lambda session_id, actor_user_id, limit, offset: [message_a, message_b],
    )

    def _fake_create_engram(payload, embedding_dim: int, owner_user_id):  # noqa: ANN001
        captured["payload"] = payload
        captured["embedding_dim"] = embedding_dim
        captured["owner_user_id"] = owner_user_id
        return EngramCreateResponse(engram_id=uuid4(), created_at=datetime.now(UTC))

    monkeypatch.setattr("app.chat.service.create_engram", _fake_create_engram)

    response = service.save_session_as_engram(
        actor_user_id=actor_id,
        session_id=session.session_id,
        payload=SaveSessionAsEngramRequest(
            title="Saved Session",
            abstract="Session summary",
            visibility_scope=VisibilityScope.project,
            tags=["chat"],
            keywords=["continuity"],
        ),
    )

    assert response.session_id == session.session_id
    assert captured["embedding_dim"] == 256
    assert captured["owner_user_id"] == actor_id
    assert captured["payload"].source_session_id == session.session_id
    assert captured["payload"].visibility_scope == VisibilityScope.project.value
    assert captured["payload"].thread_id == f"chat-session:{session.session_id}"


def test_save_session_as_engram_derives_abstract_from_latest_assistant(monkeypatch) -> None:
    actor_id = uuid4()
    session = _session(actor_id)
    service = ChatService(embedding_dim=256)
    message_a = _message(
        message_id=uuid4(),
        session_id=session.session_id,
        role="user",
        content_text="Need a clean incident summary",
    )
    message_b = _message(
        message_id=uuid4(),
        session_id=session.session_id,
        role="assistant",
        content_text="Payment outage caused by DB saturation; rollback ineffective due to query-plan drift.",
    )
    captured: dict = {}

    monkeypatch.setattr(
        "app.chat.service.get_chat_session", lambda session_id, actor_user_id: session
    )
    monkeypatch.setattr(
        "app.chat.service.list_chat_messages",
        lambda session_id, actor_user_id, limit, offset: [message_a, message_b],
    )

    def _fake_create_engram(payload, embedding_dim: int, owner_user_id):  # noqa: ANN001
        captured["payload"] = payload
        captured["embedding_dim"] = embedding_dim
        captured["owner_user_id"] = owner_user_id
        return EngramCreateResponse(engram_id=uuid4(), created_at=datetime.now(UTC))

    monkeypatch.setattr("app.chat.service.create_engram", _fake_create_engram)

    service.save_session_as_engram(
        actor_user_id=actor_id,
        session_id=session.session_id,
        payload=SaveSessionAsEngramRequest(
            title="Saved Session",
            abstract="Snapshot from active chat session.",
            visibility_scope=VisibilityScope.project,
            tags=["chat"],
            keywords=["continuity"],
        ),
    )

    assert captured["payload"].abstract.startswith("Payment outage caused by DB saturation")


def test_pin_engram_raises_for_inaccessible_resources(monkeypatch) -> None:
    actor_id = uuid4()
    service = ChatService(embedding_dim=256)
    monkeypatch.setattr("app.chat.service.pin_engram_to_session", lambda **kwargs: None)

    with pytest.raises(ChatValidationError, match="not accessible"):
        service.pin_engram(
            actor_user_id=actor_id,
            session_id=uuid4(),
            payload=PinEngramRequest(engram_id=uuid4()),
        )


def test_create_session_delegates_to_repository(monkeypatch) -> None:
    actor_id = uuid4()
    service = ChatService(embedding_dim=256)
    expected = _session(actor_id)
    monkeypatch.setattr(
        "app.chat.service.create_chat_session", lambda owner_user_id, payload: expected
    )

    created = service.create_session(
        actor_user_id=actor_id,
        payload=ChatSessionCreateRequest(
            project_id="project-chat",
            title="Session",
            provider=ChatProvider.openai,
            model_id="gpt-4o-mini",
            visibility_scope=VisibilityScope.private,
        ),
    )

    assert created.session_id == expected.session_id


def test_raise_provider_error_maps_provider_request_error() -> None:
    with pytest.raises(ChatProviderExecutionError) as exc_info:
        ChatService._raise_provider_error(ProviderRequestError("bad request to provider"))

    assert exc_info.value.status_code == 400
    assert exc_info.value.error_code == "provider_request_error"
