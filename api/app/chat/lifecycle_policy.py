from __future__ import annotations

from collections.abc import Iterable
from dataclasses import dataclass
from datetime import UTC, datetime, timedelta
from uuid import UUID

from app.models import ChatAutosaveStrategy, EngramSummary


@dataclass(frozen=True)
class SessionLifecyclePolicy:
    autosave_enabled: bool
    autosave_strategy: ChatAutosaveStrategy
    autosave_interval_minutes: int
    autosave_min_messages: int
    retention_days: int
    retention_max_snapshots: int


@dataclass(frozen=True)
class TimelineEventSemantics:
    event_type: str
    consolidation_group_key: str | None = None
    consolidation_merged_count: int | None = None


_CONSOLIDATION_GROUP_KEY_TAG_PREFIX = "consolidation_group_key:"
_CONSOLIDATION_MERGED_COUNT_TAG_PREFIX = "consolidation_merged_count:"


def normalize_autosave_policy(
    *,
    autosave_enabled: bool,
    autosave_strategy: ChatAutosaveStrategy,
) -> tuple[bool, ChatAutosaveStrategy]:
    """Keep legacy `autosave_enabled` and strategy-based controls coherent.

    Compatibility rules:
    - disabled autosave always forces strategy `off`.
    - enabling autosave with `off` strategy upgrades to `interval`.
    """

    if not autosave_enabled:
        return False, ChatAutosaveStrategy.off
    if autosave_strategy == ChatAutosaveStrategy.off:
        return True, ChatAutosaveStrategy.interval
    return True, autosave_strategy


def _normalize_tags(tags: list[str]) -> list[str]:
    normalized: list[str] = []
    for item in tags:
        candidate = item.strip()
        if candidate:
            normalized.append(candidate)
    return normalized


def _extract_tag_value(tags: list[str], *, prefix: str) -> str | None:
    for tag in tags:
        if tag.lower().startswith(prefix):
            value = tag[len(prefix) :].strip()
            if value:
                return value
    return None


def _parse_positive_int(value: str | None) -> int | None:
    if value is None:
        return None
    try:
        parsed = int(value)
    except ValueError:
        return None
    return parsed if parsed > 0 else None


def classify_timeline_event(tags: list[str]) -> TimelineEventSemantics:
    normalized_tags = _normalize_tags(tags)
    tag_set = {item.lower() for item in normalized_tags}
    if "autosave_snapshot" in tag_set:
        return TimelineEventSemantics(event_type="autosave_snapshot")
    if "consolidated" in tag_set:
        group_key = _extract_tag_value(
            normalized_tags,
            prefix=_CONSOLIDATION_GROUP_KEY_TAG_PREFIX,
        )
        merged_count = _parse_positive_int(
            _extract_tag_value(
                normalized_tags,
                prefix=_CONSOLIDATION_MERGED_COUNT_TAG_PREFIX,
            )
        )
        event_type = "consolidation"
        if merged_count and merged_count > 1:
            event_type = "consolidation_merge"
        elif group_key:
            event_type = "consolidation_group"
        return TimelineEventSemantics(
            event_type=event_type,
            consolidation_group_key=group_key,
            consolidation_merged_count=merged_count,
        )
    return TimelineEventSemantics(event_type="manual_snapshot")


def classify_timeline_event_type(tags: list[str]) -> str:
    return classify_timeline_event(tags).event_type


def is_low_value_snapshot_abstract(value: str) -> bool:
    normalized = " ".join(value.strip().split())
    if not normalized:
        return True
    return len(normalized) < 36 or len(normalized.split()) < 7


def should_take_interval_snapshot(
    *,
    now: datetime,
    latest_snapshot_created_at: datetime | None,
    interval_minutes: int,
) -> bool:
    if latest_snapshot_created_at is None:
        return True
    threshold = now - timedelta(minutes=max(interval_minutes, 1))
    return latest_snapshot_created_at <= threshold


def should_take_message_count_snapshot(
    *,
    assistant_message_count: int,
    min_messages: int,
) -> bool:
    if assistant_message_count <= 0:
        return False
    window = max(min_messages, 1)
    return assistant_message_count % window == 0


def duplicate_snapshot_exists(
    *,
    abstract: str,
    existing_snapshots: Iterable[EngramSummary],
) -> bool:
    normalized = " ".join(abstract.strip().split()).lower()
    if not normalized:
        return False
    for item in existing_snapshots:
        candidate = " ".join(item.abstract.strip().split()).lower()
        if candidate == normalized:
            return True
    return False


def select_retention_prune_ids(
    *,
    snapshots: list[EngramSummary],
    retention_days: int,
    retention_max_snapshots: int,
    now: datetime | None = None,
) -> list[UUID]:
    if not snapshots:
        return []

    reference = now or datetime.now(UTC)
    cutoff = reference - timedelta(days=max(retention_days, 1))
    max_count = max(retention_max_snapshots, 1)

    keep_ids: set[UUID] = set()
    for idx, snapshot in enumerate(snapshots):
        within_count = idx < max_count
        within_time = snapshot.created_at >= cutoff
        if within_count and within_time:
            keep_ids.add(snapshot.engram_id)

    prune_ids: list[UUID] = []
    for snapshot in snapshots:
        if snapshot.engram_id not in keep_ids:
            prune_ids.append(snapshot.engram_id)
    return prune_ids
