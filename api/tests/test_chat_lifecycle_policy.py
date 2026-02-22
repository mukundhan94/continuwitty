from __future__ import annotations

from datetime import UTC, datetime, timedelta
from uuid import uuid4

from app.chat.lifecycle_policy import (
    classify_timeline_event,
    classify_timeline_event_type,
    duplicate_snapshot_exists,
    is_low_value_snapshot_abstract,
    normalize_autosave_policy,
    select_retention_prune_ids,
    should_take_interval_snapshot,
    should_take_message_count_snapshot,
)
from app.models import ChatAutosaveStrategy, EngramSummary


def _snapshot(
    minutes_ago: int, title: str = "Snapshot", abstract: str = "Useful summary"
) -> EngramSummary:
    now = datetime.now(UTC)
    return EngramSummary(
        engram_id=uuid4(),
        project_id="project-a",
        thread_id="chat-session:abc:autosave",
        title=title,
        abstract=abstract,
        created_at=now - timedelta(minutes=minutes_ago),
        tags=["autosave_snapshot"],
        keywords=["snapshot"],
    )


def test_normalize_autosave_policy_keeps_backward_compatibility() -> None:
    enabled, strategy = normalize_autosave_policy(
        autosave_enabled=True,
        autosave_strategy=ChatAutosaveStrategy.off,
    )
    assert enabled is True
    assert strategy == ChatAutosaveStrategy.interval

    enabled_disabled, strategy_disabled = normalize_autosave_policy(
        autosave_enabled=False,
        autosave_strategy=ChatAutosaveStrategy.message_count,
    )
    assert enabled_disabled is False
    assert strategy_disabled == ChatAutosaveStrategy.off


def test_interval_snapshot_trigger_rules() -> None:
    now = datetime.now(UTC)
    assert should_take_interval_snapshot(
        now=now,
        latest_snapshot_created_at=None,
        interval_minutes=15,
    )
    assert not should_take_interval_snapshot(
        now=now,
        latest_snapshot_created_at=now - timedelta(minutes=5),
        interval_minutes=15,
    )
    assert should_take_interval_snapshot(
        now=now,
        latest_snapshot_created_at=now - timedelta(minutes=20),
        interval_minutes=15,
    )


def test_message_count_snapshot_trigger_rules() -> None:
    assert should_take_message_count_snapshot(assistant_message_count=6, min_messages=3)
    assert not should_take_message_count_snapshot(assistant_message_count=5, min_messages=3)


def test_duplicate_and_low_value_guards() -> None:
    snapshots = [_snapshot(minutes_ago=10, abstract="Primary issue is cache invalidation lag.")]
    assert duplicate_snapshot_exists(
        abstract="Primary issue is cache invalidation lag.",
        existing_snapshots=snapshots,
    )
    assert is_low_value_snapshot_abstract("too short")
    assert not is_low_value_snapshot_abstract(
        "Primary issue is cache invalidation lag causing stale reads in checkout flow."
    )


def test_retention_pruning_respects_age_and_max_count() -> None:
    now = datetime.now(UTC)
    snapshots = [
        EngramSummary(
            engram_id=uuid4(),
            project_id="project-a",
            thread_id="chat-session:abc:autosave",
            title=f"Snapshot {idx}",
            abstract="Useful long summary for retention checks.",
            created_at=now - timedelta(days=idx),
            tags=["autosave_snapshot"],
            keywords=["snapshot"],
        )
        for idx in range(6)
    ]
    prune_ids = select_retention_prune_ids(
        snapshots=snapshots,
        retention_days=2,
        retention_max_snapshots=3,
        now=now,
    )
    assert len(prune_ids) >= 3


def test_timeline_event_classification() -> None:
    assert classify_timeline_event_type(["autosave_snapshot", "chat"]) == "autosave_snapshot"
    assert classify_timeline_event_type(["consolidated", "maintenance"]) == "consolidation"
    assert (
        classify_timeline_event_type(
            ["consolidated", "consolidation_group_key:incident-42"]
        )
        == "consolidation_group"
    )
    assert (
        classify_timeline_event_type(
            [
                "consolidated",
                "consolidation_group_key:incident-42",
                "consolidation_merged_count:3",
            ]
        )
        == "consolidation_merge"
    )
    assert classify_timeline_event_type(["incident", "handoff"]) == "manual_snapshot"

    semantics = classify_timeline_event(
        [
            "consolidated",
            "consolidation_group_key:incident-42",
            "consolidation_merged_count:3",
        ]
    )
    assert semantics.consolidation_group_key == "incident-42"
    assert semantics.consolidation_merged_count == 3
