from __future__ import annotations

from datetime import UTC, datetime, timedelta
from uuid import UUID, uuid4

import pytest

from app.chat.context import AssembledChatContext
from app.chat.errors import ChatProviderExecutionError, ChatValidationError
from app.chat.service import ChatService
from app.models import (
    ChatAutosaveStrategy,
    ChatLifecyclePolicyUpdateRequest,
    ChatMessageCreateRequest,
    ChatMessageRecord,
    ChatProvider,
    ChatSessionCreateRequest,
    ChatSessionRecord,
    ContinueSessionRequest,
    EngramCreateResponse,
    EngramSummary,
    PinEngramRequest,
    PinnedDocumentRecord,
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


def _context_with_source(referenced_engram_id: UUID) -> AssembledChatContext:
    return AssembledChatContext(
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
    )


class _FixedReplyAdapter:
    def generate(self, request):  # noqa: ANN001
        return ProviderGenerateResult(
            provider=ChatProvider.openai,
            model_id=request.model_id,
            text="Proceed with option A.",
            token_usage={"input_tokens": 9, "output_tokens": 4, "total_tokens": 13},
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
        lambda **kwargs: _context_with_source(referenced_engram_id),
    )
    monkeypatch.setattr("app.chat.service.list_chat_messages", lambda **kwargs: [user_message])

    def _fake_create_chat_message(**kwargs):
        if kwargs["role"] == "user":
            return user_message
        assert kwargs["metadata"] and kwargs["metadata"].used_engram_ids == [referenced_engram_id]
        return assistant_message

    monkeypatch.setattr("app.chat.service.create_chat_message", _fake_create_chat_message)

    monkeypatch.setattr("app.chat.service.get_provider_adapter", lambda provider: _FixedReplyAdapter())

    response = service.send_message(
        actor_user_id=actor_id,
        session_id=session.session_id,
        payload=ChatMessageCreateRequest(content_text="How should we proceed?"),
    )

    assert (
        response.session_id,
        response.message_id,
        response.reply_message_id,
        response.assistant_text,
        response.used_engram_ids,
        response.used_document_chunk_ids,
    ) == (session.session_id, user_message_id, reply_message_id, "Proceed with option A.", [referenced_engram_id], [])
    assert response.source_references[0].url == "https://example.com/source"
    assert response.debug_trace is not None
    assert response.debug_trace.llm_calls[0].provider == "openai"
    assert response.debug_trace.llm_calls[0].token_usage["total_tokens"] == 13
    assert response.debug_trace.request_input_text == "How should we proceed?"


def test_continue_session_copies_pinned_engrams(monkeypatch) -> None:
    actor_id = uuid4()
    session = _session(actor_id)
    continued = _session(actor_id)
    engram_a = uuid4()
    engram_b = uuid4()
    document_a = uuid4()
    copied_document_ids: list[UUID] = []
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
    monkeypatch.setattr(
        "app.chat.service.list_pinned_documents",
        lambda session_id, actor_user_id: [
            PinnedDocumentRecord(
                session_id=session.session_id,
                document_id=document_a,
                pinned_by_user_id=actor_id,
                created_at=datetime.now(UTC),
            )
        ],
    )

    def _fake_pin_document_to_session(session_id, document_id, actor_user_id):  # noqa: ANN001
        copied_document_ids.append(document_id)
        return PinnedDocumentRecord(
            session_id=session_id,
            document_id=document_id,
            pinned_by_user_id=actor_user_id,
            created_at=datetime.now(UTC),
        )

    monkeypatch.setattr("app.chat.service.pin_document_to_session", _fake_pin_document_to_session)

    response = service.continue_session(
        actor_user_id=actor_id,
        session_id=session.session_id,
        payload=ContinueSessionRequest(title="Continued Session"),
    )

    assert response.session.title == "Continued Session"
    assert response.carried_engram_ids == [engram_a, engram_b]
    assert copied_document_ids == [document_a]


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

    def _fake_create_engram(  # noqa: ANN001
        payload,
        embedding_dim: int,
        owner_user_id,
        enrichment_origin: str = "unknown",
    ):
        captured["payload"] = payload
        captured["embedding_dim"] = embedding_dim
        captured["owner_user_id"] = owner_user_id
        captured["enrichment_origin"] = enrichment_origin
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
    assert captured["enrichment_origin"] == "chat.save_as_engram"
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

    def _fake_create_engram(  # noqa: ANN001
        payload,
        embedding_dim: int,
        owner_user_id,
        enrichment_origin: str = "unknown",
    ):
        captured["payload"] = payload
        captured["embedding_dim"] = embedding_dim
        captured["owner_user_id"] = owner_user_id
        captured["enrichment_origin"] = enrichment_origin
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
    assert captured["enrichment_origin"] == "chat.save_as_engram"


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
        "app.chat.service.ensure_project_exists", lambda project_id, owner_user_id: None
    )
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


def test_update_lifecycle_policy_normalizes_disabled_autosave(monkeypatch) -> None:
    actor_id = uuid4()
    service = ChatService(embedding_dim=256)
    captured: dict = {}

    def _fake_update_chat_session(session_id, actor_user_id, payload):  # noqa: ANN001
        _ = session_id, actor_user_id
        captured["payload"] = payload
        return _session(actor_id).model_copy(
            update={
                "autosave_enabled": payload.autosave_enabled,
                "autosave_strategy": payload.autosave_strategy,
            }
        )

    monkeypatch.setattr("app.chat.service.update_chat_session", _fake_update_chat_session)

    policy = service.update_lifecycle_policy(
        actor_user_id=actor_id,
        session_id=uuid4(),
        payload=ChatLifecyclePolicyUpdateRequest(
            autosave_enabled=False,
            autosave_strategy=ChatAutosaveStrategy.interval,
        ),
    )

    assert captured["payload"].autosave_enabled is False
    assert captured["payload"].autosave_strategy == ChatAutosaveStrategy.off
    assert policy.autosave_enabled is False
    assert policy.autosave_strategy == ChatAutosaveStrategy.off


def test_run_session_lifecycle_creates_autosave_snapshot(monkeypatch) -> None:
    actor_id = uuid4()
    session = _session(actor_id).model_copy(
        update={
            "autosave_enabled": True,
            "autosave_strategy": ChatAutosaveStrategy.interval,
            "autosave_interval_minutes": 1,
            "retention_days": 30,
            "retention_max_snapshots": 10,
        }
    )
    service = ChatService(embedding_dim=256)
    captured: dict = {}

    monkeypatch.setattr("app.chat.service.list_session_linked_engrams", lambda **kwargs: [])
    monkeypatch.setattr(
        "app.chat.service.list_chat_messages",
        lambda **kwargs: [
            _message(
                message_id=uuid4(),
                session_id=session.session_id,
                role="user",
                content_text="Summarize incident details for stakeholders.",
            ),
            _message(
                message_id=uuid4(),
                session_id=session.session_id,
                role="assistant",
                content_text=(
                    "Incident timeline confirmed. Primary issue was cache invalidation lag "
                    "causing stale checkout reads and payment retries."
                ),
            ),
        ],
    )

    def _fake_create_engram(  # noqa: ANN001
        payload,
        embedding_dim: int,
        owner_user_id,
        enrichment_origin: str = "unknown",
    ):
        captured["payload"] = payload
        captured["embedding_dim"] = embedding_dim
        captured["owner_user_id"] = owner_user_id
        captured["enrichment_origin"] = enrichment_origin
        return EngramCreateResponse(engram_id=uuid4(), created_at=datetime.now(UTC))

    monkeypatch.setattr("app.chat.service.create_engram", _fake_create_engram)
    monkeypatch.setattr(
        "app.chat.service.delete_session_autosave_engrams",
        lambda **kwargs: [],
    )

    result = service._run_session_lifecycle_maintenance(
        actor_user_id=actor_id,
        session=session,
    )

    assert result.snapshot_engram_id is not None
    assert result.pruned_engram_ids == []
    assert "autosave_snapshot" in captured["payload"].tags
    assert captured["payload"].source_session_id == session.session_id
    assert captured["enrichment_origin"] == "chat.autosave_snapshot"


def test_run_session_lifecycle_prunes_retention_excess(monkeypatch) -> None:
    actor_id = uuid4()
    now = datetime.now(UTC)
    session = _session(actor_id).model_copy(
        update={
            "autosave_enabled": True,
            "autosave_strategy": ChatAutosaveStrategy.interval,
            "autosave_interval_minutes": 60,
            "retention_days": 365,
            "retention_max_snapshots": 1,
        }
    )
    service = ChatService(embedding_dim=256)

    snapshots = [
        EngramSummary(
            engram_id=uuid4(),
            project_id=session.project_id,
            thread_id=f"chat-session:{session.session_id}:autosave",
            title=f"Snapshot {idx}",
            abstract="High-value summary for lifecycle retention policy.",
            created_at=now - timedelta(minutes=idx),
            tags=["autosave_snapshot"],
            keywords=[],
        )
        for idx in range(3)
    ]

    monkeypatch.setattr(
        "app.chat.service.list_session_linked_engrams",
        lambda **kwargs: snapshots,
    )
    monkeypatch.setattr(
        "app.chat.service.delete_session_autosave_engrams",
        lambda **kwargs: kwargs["engram_ids"],
    )

    result = service._run_session_lifecycle_maintenance(
        actor_user_id=actor_id,
        session=session,
    )

    assert result.snapshot_engram_id is None
    assert len(result.pruned_engram_ids) == 2
