from __future__ import annotations

from datetime import UTC, datetime, timedelta
from typing import Any

from .config import get_settings
from .models import Decision, EngramSummary, MemoryEngramCreate
from .repository import create_engram, list_engrams


def _build_consolidated_payload(
    project_id: str,
    source_rows: list[EngramSummary],
    now: datetime,
) -> MemoryEngramCreate:
    oldest = source_rows[-1].created_at
    newest = source_rows[0].created_at

    summary_lines = [
        f"- {row.created_at.isoformat()} | {row.title}: {row.abstract}" for row in source_rows
    ]
    detailed_summary = (
        "# Consolidated Memory Snapshot\n\n"
        f"Project: {project_id}\n\n"
        f"Coverage window: {oldest.isoformat()} -> {newest.isoformat()}\n\n"
        "## Included Engrams\n" + "\n".join(summary_lines)
    )

    keyword_pool: list[str] = []
    for row in source_rows:
        keyword_pool.extend(row.tags)
        keyword_pool.extend(row.keywords)

    unique_keywords: list[str] = []
    for keyword in keyword_pool:
        if keyword and keyword not in unique_keywords:
            unique_keywords.append(keyword)
        if len(unique_keywords) >= 16:
            break

    return MemoryEngramCreate(
        project_id=project_id,
        thread_id=f"consolidation-{now.strftime('%Y%m%d%H%M%S')}",
        title=f"Consolidated Memory Snapshot ({now.date().isoformat()})",
        abstract=(
            f"Auto-consolidated {len(source_rows)} engrams "
            f"from {oldest.date().isoformat()} to {newest.date().isoformat()}."
        ),
        detailed_summary_markdown=detailed_summary,
        decisions=[
            Decision(
                decision="Create periodic memory consolidation snapshot",
                rationale="Improve long-term retrieval and reduce context fragmentation.",
            )
        ],
        assumptions=["Consolidation rows are already quality-filtered by project scope."],
        open_questions=["Should consolidation frequency vary by project activity level?"],
        tags=["consolidated", "maintenance", "auto"],
        keywords=unique_keywords,
    )


def run_project_consolidation(
    project_id: str,
    source_limit: int = 20,
    min_items: int = 3,
    dry_run: bool = False,
) -> dict[str, Any]:
    now = datetime.now(UTC)
    source_rows = list_engrams(project_id=project_id, limit=source_limit, offset=0)
    required_items = max(min_items, 1)

    if not source_rows:
        return {
            "project_id": project_id,
            "created": False,
            "reason": "no_engrams",
            "source_count": 0,
            "source_engram_ids": [],
        }

    latest = source_rows[0]
    latest_is_consolidated = "consolidated" in latest.tags
    latest_is_recent = (now - latest.created_at) < timedelta(hours=6)
    if latest_is_consolidated and latest_is_recent:
        return {
            "project_id": project_id,
            "created": False,
            "reason": "recent_consolidation_exists",
            "source_count": 0,
            "source_engram_ids": [],
        }

    candidates = [row for row in source_rows if "consolidated" not in row.tags]
    if len(candidates) < required_items:
        return {
            "project_id": project_id,
            "created": False,
            "reason": "insufficient_non_consolidated_engrams",
            "source_count": len(candidates),
            "source_engram_ids": [str(row.engram_id) for row in candidates],
        }

    selected = candidates[:source_limit]
    payload = _build_consolidated_payload(project_id=project_id, source_rows=selected, now=now)

    if dry_run:
        return {
            "project_id": project_id,
            "created": False,
            "reason": "dry_run",
            "source_count": len(selected),
            "source_engram_ids": [str(row.engram_id) for row in selected],
            "proposed_title": payload.title,
        }

    created = create_engram(payload, get_settings().embedding_dim)
    return {
        "project_id": project_id,
        "created": True,
        "reason": "created",
        "engram_id": str(created.engram_id),
        "source_count": len(selected),
        "source_engram_ids": [str(row.engram_id) for row in selected],
    }
