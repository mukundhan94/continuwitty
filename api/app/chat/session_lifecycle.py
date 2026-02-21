from __future__ import annotations

from collections.abc import Callable
from dataclasses import dataclass
from datetime import UTC, datetime
from uuid import UUID

from app.models import (
    ChatAutosaveStrategy,
    ChatMessageRecord,
    ChatSessionRecord,
    EngramCreateResponse,
    EngramSummary,
    MemoryEngramCreate,
)

from .lifecycle_policy import (
    duplicate_snapshot_exists,
    is_low_value_snapshot_abstract,
    select_retention_prune_ids,
    should_take_interval_snapshot,
    should_take_message_count_snapshot,
)

_GENERIC_SNAPSHOT_ABSTRACTS = {
    "",
    "snapshot from active chat session.",
    "snapshot from active chat session",
    "chat snapshot",
    "session snapshot",
}
_AUTOSAVE_SNAPSHOT_TAGS = [
    "autosave_snapshot",
    "session-lifecycle",
    "chat",
]


@dataclass(frozen=True)
class SessionLifecycleDependencies:
    list_session_linked_engrams: Callable[..., list[EngramSummary]]
    list_chat_messages: Callable[..., list[ChatMessageRecord]]
    count_session_messages_by_role: Callable[..., int]
    delete_session_autosave_engrams: Callable[..., list[UUID]]
    create_engram: Callable[..., EngramCreateResponse]


@dataclass(frozen=True)
class LifecycleMaintenanceResult:
    snapshot_engram_id: UUID | None
    pruned_engram_ids: list[UUID]
    skipped_reason: str | None = None


@dataclass(frozen=True)
class AutosaveSnapshotCreateRequest:
    actor_user_id: UUID
    session: ChatSessionRecord
    existing_snapshots: list[EngramSummary]
    embedding_dim: int


@dataclass(frozen=True)
class SnapshotResolutionRequest:
    actor_user_id: UUID
    session: ChatSessionRecord
    autosave_snapshots: list[EngramSummary]
    now: datetime


@dataclass(frozen=True)
class SnapshotCreationRequest:
    actor_user_id: UUID
    session: ChatSessionRecord
    autosave_snapshots: list[EngramSummary]
    should_create: bool
    skipped_reason: str | None
    embedding_dim: int


def _normalize_spaces(value: str) -> str:
    return " ".join(value.strip().split())


def _truncate_text(value: str, max_chars: int) -> str:
    if len(value) <= max_chars:
        return value
    return value[: max_chars - 3].rstrip() + "..."


def is_generic_snapshot_abstract(value: str) -> bool:
    return _normalize_spaces(value).lower() in _GENERIC_SNAPSHOT_ABSTRACTS


def derive_chat_snapshot_abstract(
    messages: list[ChatMessageRecord], max_chars: int = 320
) -> str:
    for role in ("assistant", "user"):
        for message in reversed(messages):
            if message.role != role:
                continue
            normalized = _normalize_spaces(message.content_text)
            if not normalized:
                continue
            return _truncate_text(normalized, max_chars=max_chars)
    return ""


def transcript_markdown(session: ChatSessionRecord, messages: list[ChatMessageRecord]) -> str:
    lines = [
        f"# Chat Session Snapshot: {session.title}",
        "",
        f"- Session ID: {session.session_id}",
        f"- Provider/Model: {session.provider.value}/{session.model_id}",
        "",
    ]
    for message in messages:
        lines.extend(
            [
                f"## {message.role.upper()} ({message.created_at.isoformat()})",
                message.content_text.strip() or "(empty)",
                "",
            ]
        )
    return "\n".join(lines).strip()


def retrieval_text_from_messages(
    messages: list[ChatMessageRecord], tail_count: int = 8
) -> str:
    tail = messages[-tail_count:]
    return " ".join(item.content_text.strip() for item in tail if item.content_text.strip())


def _is_autosave_snapshot(summary: EngramSummary) -> bool:
    return "autosave_snapshot" in {item.strip().lower() for item in summary.tags}


def _list_autosave_snapshots(
    *,
    actor_user_id: UUID,
    session_id: UUID,
    dependencies: SessionLifecycleDependencies,
    limit: int = 500,
) -> list[EngramSummary]:
    linked = dependencies.list_session_linked_engrams(
        session_id=session_id,
        actor_user_id=actor_user_id,
        limit=limit,
        offset=0,
    )
    return [item for item in linked if _is_autosave_snapshot(item)]


def _create_autosave_snapshot(
    *,
    request: AutosaveSnapshotCreateRequest,
    dependencies: SessionLifecycleDependencies,
) -> UUID | None:
    messages = dependencies.list_chat_messages(
        session_id=request.session.session_id,
        actor_user_id=request.actor_user_id,
        limit=500,
        offset=0,
    )
    if not messages:
        return None

    abstract = derive_chat_snapshot_abstract(messages)
    if is_low_value_snapshot_abstract(abstract):
        return None
    if duplicate_snapshot_exists(abstract=abstract, existing_snapshots=request.existing_snapshots):
        return None

    now = datetime.now(UTC)
    created = dependencies.create_engram(
        payload=MemoryEngramCreate(
            project_id=request.session.project_id,
            thread_id=f"chat-session:{request.session.session_id}:autosave",
            title=f"{request.session.title} Autosave {now.strftime('%Y-%m-%d %H:%M:%S')}",
            abstract=abstract,
            detailed_summary_markdown=transcript_markdown(request.session, messages),
            tags=[
                *_AUTOSAVE_SNAPSHOT_TAGS,
                f"autosave_strategy:{request.session.autosave_strategy.value}",
            ],
            keywords=[
                "autosave",
                "snapshot",
                request.session.provider.value,
                request.session.model_id,
            ],
            visibility_scope=request.session.visibility_scope.value,
            source_session_id=request.session.session_id,
            retrieval_text=retrieval_text_from_messages(messages),
        ),
        embedding_dim=request.embedding_dim,
        owner_user_id=request.actor_user_id,
        enrichment_origin="chat.autosave_snapshot",
    )
    return created.engram_id


def _resolve_snapshot_creation(
    *,
    request: SnapshotResolutionRequest,
    dependencies: SessionLifecycleDependencies,
) -> tuple[bool, str | None]:
    if request.session.autosave_strategy == ChatAutosaveStrategy.interval:
        latest_created_at = (
            request.autosave_snapshots[0].created_at
            if request.autosave_snapshots
            else None
        )
        should_create = should_take_interval_snapshot(
            now=request.now,
            latest_snapshot_created_at=latest_created_at,
            interval_minutes=request.session.autosave_interval_minutes,
        )
        return should_create, None if should_create else "interval_not_elapsed"

    if request.session.autosave_strategy == ChatAutosaveStrategy.message_count:
        assistant_message_count = dependencies.count_session_messages_by_role(
            session_id=request.session.session_id,
            actor_user_id=request.actor_user_id,
            role="assistant",
        )
        should_create = should_take_message_count_snapshot(
            assistant_message_count=assistant_message_count,
            min_messages=request.session.autosave_min_messages,
        )
        return should_create, None if should_create else "message_count_threshold_not_met"

    return False, None


def _maybe_create_snapshot(
    *,
    request: SnapshotCreationRequest,
    dependencies: SessionLifecycleDependencies,
) -> tuple[UUID | None, list[EngramSummary], str | None]:
    if not request.should_create:
        return None, request.autosave_snapshots, request.skipped_reason

    created_snapshot_id = _create_autosave_snapshot(
        request=AutosaveSnapshotCreateRequest(
            actor_user_id=request.actor_user_id,
            session=request.session,
            existing_snapshots=request.autosave_snapshots,
            embedding_dim=request.embedding_dim,
        ),
        dependencies=dependencies,
    )
    if created_snapshot_id is None:
        return None, request.autosave_snapshots, "duplicate_or_low_value_snapshot"

    refreshed_snapshots = _list_autosave_snapshots(
        actor_user_id=request.actor_user_id,
        session_id=request.session.session_id,
        dependencies=dependencies,
    )
    return created_snapshot_id, refreshed_snapshots, request.skipped_reason


def run_session_lifecycle_maintenance(
    *,
    actor_user_id: UUID,
    session: ChatSessionRecord,
    embedding_dim: int,
    dependencies: SessionLifecycleDependencies,
) -> LifecycleMaintenanceResult:
    if not session.autosave_enabled or session.autosave_strategy == ChatAutosaveStrategy.off:
        return LifecycleMaintenanceResult(
            snapshot_engram_id=None,
            pruned_engram_ids=[],
            skipped_reason="autosave_disabled",
        )

    now = datetime.now(UTC)
    autosave_snapshots = _list_autosave_snapshots(
        actor_user_id=actor_user_id,
        session_id=session.session_id,
        dependencies=dependencies,
    )
    should_create, skipped_reason = _resolve_snapshot_creation(
        request=SnapshotResolutionRequest(
            actor_user_id=actor_user_id,
            session=session,
            autosave_snapshots=autosave_snapshots,
            now=now,
        ),
        dependencies=dependencies,
    )
    created_snapshot_id, autosave_snapshots, skipped_reason = _maybe_create_snapshot(
        request=SnapshotCreationRequest(
            actor_user_id=actor_user_id,
            session=session,
            autosave_snapshots=autosave_snapshots,
            should_create=should_create,
            skipped_reason=skipped_reason,
            embedding_dim=embedding_dim,
        ),
        dependencies=dependencies,
    )

    prune_ids = select_retention_prune_ids(
        snapshots=autosave_snapshots,
        retention_days=session.retention_days,
        retention_max_snapshots=session.retention_max_snapshots,
        now=now,
    )
    pruned_ids = dependencies.delete_session_autosave_engrams(
        session_id=session.session_id,
        actor_user_id=actor_user_id,
        engram_ids=prune_ids,
    )

    return LifecycleMaintenanceResult(
        snapshot_engram_id=created_snapshot_id,
        pruned_engram_ids=pruned_ids,
        skipped_reason=skipped_reason,
    )
