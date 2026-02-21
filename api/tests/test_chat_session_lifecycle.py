from __future__ import annotations

from datetime import UTC, datetime
from uuid import UUID, uuid4

from app.chat.session_lifecycle import (
    SessionLifecycleDependencies,
    derive_chat_snapshot_abstract,
    run_session_lifecycle_maintenance,
)
from app.models import (
    ChatAutosaveStrategy,
    ChatMessageRecord,
    ChatProvider,
    ChatSessionRecord,
    EngramCreateResponse,
    EngramSummary,
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
    session_id: UUID,
    role: str,
    content_text: str,
) -> ChatMessageRecord:
    return ChatMessageRecord(
        message_id=uuid4(),
        session_id=session_id,
        role=role,
        content_text=content_text,
        provider=None,
        model_id=None,
        token_usage_json={},
        used_engram_ids=[],
        created_at=datetime.now(UTC),
    )


def test_derive_chat_snapshot_abstract_prefers_latest_assistant() -> None:
    session_id = uuid4()
    messages = [
        _message(
            session_id=session_id,
            role="user",
            content_text="First question",
        ),
        _message(
            session_id=session_id,
            role="assistant",
            content_text="Older summary",
        ),
        _message(
            session_id=session_id,
            role="assistant",
            content_text="Latest assistant summary",
        ),
    ]

    assert derive_chat_snapshot_abstract(messages) == "Latest assistant summary"


def test_run_session_lifecycle_returns_disabled_without_side_effects() -> None:
    actor_id = uuid4()
    session = _session(actor_id).model_copy(
        update={
            "autosave_enabled": False,
            "autosave_strategy": ChatAutosaveStrategy.off,
        }
    )
    calls = {"linked": 0, "messages": 0, "count": 0, "delete": 0, "create": 0}
    dependencies = SessionLifecycleDependencies(
        list_session_linked_engrams=lambda **kwargs: calls.__setitem__("linked", calls["linked"] + 1) or [],
        list_chat_messages=lambda **kwargs: calls.__setitem__("messages", calls["messages"] + 1) or [],
        count_session_messages_by_role=lambda **kwargs: calls.__setitem__("count", calls["count"] + 1) or 0,
        delete_session_autosave_engrams=lambda **kwargs: calls.__setitem__("delete", calls["delete"] + 1) or [],
        create_engram=lambda **kwargs: calls.__setitem__("create", calls["create"] + 1)
        or EngramCreateResponse(engram_id=uuid4(), created_at=datetime.now(UTC)),
    )

    result = run_session_lifecycle_maintenance(
        actor_user_id=actor_id,
        session=session,
        embedding_dim=256,
        dependencies=dependencies,
    )

    assert result.snapshot_engram_id is None
    assert result.pruned_engram_ids == []
    assert result.skipped_reason == "autosave_disabled"
    assert calls == {"linked": 0, "messages": 0, "count": 0, "delete": 0, "create": 0}


def test_run_session_lifecycle_skips_message_count_threshold() -> None:
    actor_id = uuid4()
    session = _session(actor_id).model_copy(
        update={
            "autosave_enabled": True,
            "autosave_strategy": ChatAutosaveStrategy.message_count,
            "autosave_min_messages": 5,
            "retention_days": 30,
            "retention_max_snapshots": 10,
        }
    )
    existing_snapshot = EngramSummary(
        engram_id=uuid4(),
        project_id=session.project_id,
        thread_id=f"chat-session:{session.session_id}:autosave",
        title="Snapshot",
        abstract="Useful summary",
        created_at=datetime.now(UTC),
        tags=["autosave_snapshot"],
        keywords=[],
    )
    delete_calls: list[list[UUID]] = []
    create_calls: list[int] = []
    dependencies = SessionLifecycleDependencies(
        list_session_linked_engrams=lambda **kwargs: [existing_snapshot],
        list_chat_messages=lambda **kwargs: [],
        count_session_messages_by_role=lambda **kwargs: 2,
        delete_session_autosave_engrams=lambda **kwargs: delete_calls.append(kwargs["engram_ids"]) or [],
        create_engram=lambda **kwargs: create_calls.append(1)
        or EngramCreateResponse(engram_id=uuid4(), created_at=datetime.now(UTC)),
    )

    result = run_session_lifecycle_maintenance(
        actor_user_id=actor_id,
        session=session,
        embedding_dim=256,
        dependencies=dependencies,
    )

    assert result.snapshot_engram_id is None
    assert result.skipped_reason == "message_count_threshold_not_met"
    assert delete_calls == [[]]
    assert create_calls == []


def test_run_session_lifecycle_creates_snapshot_when_threshold_is_met() -> None:
    actor_id = uuid4()
    session = _session(actor_id).model_copy(
        update={
            "autosave_enabled": True,
            "autosave_strategy": ChatAutosaveStrategy.message_count,
            "autosave_min_messages": 1,
            "retention_days": 30,
            "retention_max_snapshots": 10,
        }
    )
    created_id = uuid4()
    created_payload = None

    def create_engram(**kwargs):
        nonlocal created_payload
        created_payload = kwargs["payload"]
        return EngramCreateResponse(engram_id=created_id, created_at=datetime.now(UTC))

    dependencies = SessionLifecycleDependencies(
        list_session_linked_engrams=lambda **kwargs: [],
        list_chat_messages=lambda **kwargs: [
            _message(
                session_id=session.session_id,
                role="user",
                content_text="Need help with a query",
            ),
            _message(
                session_id=session.session_id,
                role="assistant",
                content_text=(
                    "Detailed assistant summary with concrete decisions, actions, and next steps."
                ),
            ),
        ],
        count_session_messages_by_role=lambda **kwargs: 1,
        delete_session_autosave_engrams=lambda **kwargs: [],
        create_engram=create_engram,
    )

    result = run_session_lifecycle_maintenance(
        actor_user_id=actor_id,
        session=session,
        embedding_dim=256,
        dependencies=dependencies,
    )

    assert result.snapshot_engram_id == created_id
    assert result.pruned_engram_ids == []
    assert result.skipped_reason is None
    assert created_payload is not None
    assert created_payload.project_id == session.project_id
    assert created_payload.source_session_id == session.session_id
    assert "autosave_snapshot" in created_payload.tags
