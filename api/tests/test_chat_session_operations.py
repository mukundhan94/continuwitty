from __future__ import annotations

from dataclasses import dataclass
from datetime import UTC, datetime, timedelta
from uuid import UUID, uuid4

import pytest

from app.chat.errors import ChatValidationError
from app.chat.service import ChatService
from app.models import (
    ChatAutosaveStrategy,
    ChatLifecyclePolicyUpdateRequest,
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


def _session(owner_user_id: UUID) -> ChatSessionRecord:
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
    message_id: UUID,
    session_id: UUID,
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


def _pinned_record_fields(*, session_id: UUID, actor_id: UUID) -> dict[str, object]:
    return {
        "session_id": session_id,
        "pinned_by_user_id": actor_id,
        "created_at": datetime.now(UTC),
    }


@dataclass
class _CreateEngramCapture:
    payload: object | None = None
    embedding_dim: int | None = None
    owner_user_id: UUID | None = None
    enrichment_origin: str | None = None


@dataclass(frozen=True)
class _SaveSessionSetup:
    actor_id: UUID
    session: ChatSessionRecord
    messages: list[ChatMessageRecord]


@dataclass(frozen=True)
class _ContinueSessionSetup:
    actor_id: UUID
    session: ChatSessionRecord
    continued: ChatSessionRecord
    pinned_engram_ids: list[UUID]
    pinned_document_ids: list[UUID]


def _pinned_record(
    *,
    session_id: UUID,
    actor_id: UUID,
    engram_id: UUID | None = None,
    document_id: UUID | None = None,
) -> PinnedEngramRecord | PinnedDocumentRecord:
    base = _pinned_record_fields(session_id=session_id, actor_id=actor_id)
    if engram_id is not None:
        return PinnedEngramRecord(**base, engram_id=engram_id)
    assert document_id is not None
    return PinnedDocumentRecord(**base, document_id=document_id)


def _install_create_engram_capture(monkeypatch, capture: _CreateEngramCapture) -> None:
    def _fake_create_engram(  # noqa: ANN001
        payload,
        embedding_dim: int,
        owner_user_id,
        enrichment_origin: str = "unknown",
    ):
        capture.payload = payload
        capture.embedding_dim = embedding_dim
        capture.owner_user_id = owner_user_id
        capture.enrichment_origin = enrichment_origin
        return EngramCreateResponse(engram_id=uuid4(), created_at=datetime.now(UTC))

    monkeypatch.setattr("app.chat.session_operations.create_engram", _fake_create_engram)


def _run_save_session_as_engram(
    *,
    monkeypatch,
    setup: _SaveSessionSetup,
    payload: SaveSessionAsEngramRequest,
) -> tuple[object, _CreateEngramCapture]:
    service = ChatService(embedding_dim=256)
    capture = _CreateEngramCapture()
    monkeypatch.setattr(
        "app.chat.session_operations.get_chat_session",
        lambda session_id, actor_user_id: setup.session,
    )
    monkeypatch.setattr(
        "app.chat.session_operations.list_chat_messages",
        lambda session_id, actor_user_id, limit, offset: setup.messages,
    )
    _install_create_engram_capture(monkeypatch, capture)

    response = service.save_session_as_engram(
        actor_user_id=setup.actor_id,
        session_id=setup.session.session_id,
        payload=payload,
    )
    return response, capture


def _install_continue_session_dependencies(
    *,
    monkeypatch,
    setup: _ContinueSessionSetup,
    copied_document_ids: list[UUID],
) -> None:
    monkeypatch.setattr(
        "app.chat.session_operations.get_chat_session",
        lambda session_id, actor_user_id: setup.session,
    )
    monkeypatch.setattr(
        "app.chat.session_operations.create_chat_session",
        lambda owner_user_id, payload: ChatSessionRecord(
            **(
                setup.continued.model_dump()
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
        "app.chat.session_operations.list_pinned_engrams",
        lambda session_id, actor_user_id: [
            _pinned_record(
                session_id=setup.session.session_id,
                engram_id=engram_id,
                actor_id=setup.actor_id,
            )
            for engram_id in setup.pinned_engram_ids
        ],
    )
    monkeypatch.setattr(
        "app.chat.session_operations.pin_engram_to_session",
        lambda session_id, engram_id, actor_user_id: _pinned_record(
            session_id=session_id,
            engram_id=engram_id,
            actor_id=actor_user_id,
        ),
    )
    monkeypatch.setattr(
        "app.chat.session_operations.list_pinned_documents",
        lambda session_id, actor_user_id: [
            _pinned_record(
                session_id=setup.session.session_id,
                document_id=document_id,
                actor_id=setup.actor_id,
            )
            for document_id in setup.pinned_document_ids
        ],
    )

    def _fake_pin_document_to_session(session_id, document_id, actor_user_id):  # noqa: ANN001
        copied_document_ids.append(document_id)
        return _pinned_record(
            session_id=session_id,
            document_id=document_id,
            actor_id=actor_user_id,
        )

    monkeypatch.setattr(
        "app.chat.session_operations.pin_document_to_session",
        _fake_pin_document_to_session,
    )


def test_continue_session_copies_pinned_engrams(monkeypatch) -> None:
    actor_id = uuid4()
    session = _session(actor_id)
    continued = _session(actor_id)
    engram_a = uuid4()
    engram_b = uuid4()
    document_a = uuid4()
    copied_document_ids: list[UUID] = []
    service = ChatService(embedding_dim=256)

    _install_continue_session_dependencies(
        monkeypatch=monkeypatch,
        setup=_ContinueSessionSetup(
            actor_id=actor_id,
            session=session,
            continued=continued,
            pinned_engram_ids=[engram_a, engram_b],
            pinned_document_ids=[document_a],
        ),
        copied_document_ids=copied_document_ids,
    )

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
    message_a = _message(message_id=uuid4(), session_id=session.session_id, role="user", content_text="Question")
    message_b = _message(message_id=uuid4(), session_id=session.session_id, role="assistant", content_text="Answer")

    response, capture = _run_save_session_as_engram(
        monkeypatch=monkeypatch,
        setup=_SaveSessionSetup(actor_id=actor_id, session=session, messages=[message_a, message_b]),
        payload=SaveSessionAsEngramRequest(
            title="Saved Session",
            abstract="Session summary",
            visibility_scope=VisibilityScope.project,
            tags=["chat"],
            keywords=["continuity"],
        ),
    )

    assert response.session_id == session.session_id
    assert capture.embedding_dim == 256
    assert capture.owner_user_id == actor_id
    assert capture.enrichment_origin == "chat.save_as_engram"
    assert capture.payload is not None
    assert capture.payload.source_session_id == session.session_id
    assert capture.payload.visibility_scope == VisibilityScope.project.value
    assert capture.payload.thread_id == f"chat-session:{session.session_id}"


def test_save_session_as_engram_derives_abstract_from_latest_assistant(monkeypatch) -> None:
    actor_id = uuid4()
    session = _session(actor_id)
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

    _, capture = _run_save_session_as_engram(
        monkeypatch=monkeypatch,
        setup=_SaveSessionSetup(actor_id=actor_id, session=session, messages=[message_a, message_b]),
        payload=SaveSessionAsEngramRequest(
            title="Saved Session",
            abstract="Snapshot from active chat session.",
            visibility_scope=VisibilityScope.project,
            tags=["chat"],
            keywords=["continuity"],
        ),
    )

    assert capture.payload is not None
    assert capture.payload.abstract.startswith("Payment outage caused by DB saturation")
    assert capture.enrichment_origin == "chat.save_as_engram"


def test_save_session_as_engram_rejects_empty_session(monkeypatch) -> None:
    actor_id = uuid4()
    session = _session(actor_id)
    service = ChatService(embedding_dim=256)
    monkeypatch.setattr(
        "app.chat.session_operations.get_chat_session",
        lambda session_id, actor_user_id: session,
    )
    monkeypatch.setattr(
        "app.chat.session_operations.list_chat_messages",
        lambda session_id, actor_user_id, limit, offset: [],
    )

    with pytest.raises(ChatValidationError, match="empty chat session"):
        service.save_session_as_engram(
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


def test_pin_engram_raises_for_inaccessible_resources(monkeypatch) -> None:
    actor_id = uuid4()
    service = ChatService(embedding_dim=256)
    monkeypatch.setattr("app.chat.session_operations.pin_engram_to_session", lambda *args: None)

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
        "app.chat.session_operations.ensure_project_exists", lambda project_id, owner_user_id: None
    )
    monkeypatch.setattr(
        "app.chat.session_operations.create_chat_session", lambda owner_user_id, payload: expected
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

    monkeypatch.setattr("app.chat.session_operations.update_chat_session", _fake_update_chat_session)

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
    capture = _CreateEngramCapture()

    monkeypatch.setattr("app.chat.session_operations.list_session_linked_engrams", lambda **kwargs: [])
    monkeypatch.setattr(
        "app.chat.session_operations.list_chat_messages",
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
    _install_create_engram_capture(monkeypatch, capture)
    monkeypatch.setattr(
        "app.chat.session_operations.delete_session_autosave_engrams",
        lambda **kwargs: [],
    )

    result = service._run_session_lifecycle_maintenance(
        actor_user_id=actor_id,
        session=session,
    )

    assert result.snapshot_engram_id is not None
    assert result.pruned_engram_ids == []
    assert capture.payload is not None
    assert "autosave_snapshot" in capture.payload.tags
    assert capture.payload.source_session_id == session.session_id
    assert capture.enrichment_origin == "chat.autosave_snapshot"


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
        "app.chat.session_operations.list_session_linked_engrams",
        lambda **kwargs: snapshots,
    )
    monkeypatch.setattr(
        "app.chat.session_operations.delete_session_autosave_engrams",
        lambda **kwargs: kwargs["engram_ids"],
    )

    result = service._run_session_lifecycle_maintenance(
        actor_user_id=actor_id,
        session=session,
    )

    assert result.snapshot_engram_id is None
    assert len(result.pruned_engram_ids) == 2
