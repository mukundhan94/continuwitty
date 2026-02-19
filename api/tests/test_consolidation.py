from __future__ import annotations

from datetime import UTC, datetime, timedelta
from types import SimpleNamespace
from uuid import UUID

from app.consolidation import run_project_consolidation
from app.models import EngramCreateResponse, EngramSummary


def _summary(
    engram_id: str,
    created_at: datetime,
    tags: list[str] | None = None,
    keywords: list[str] | None = None,
) -> EngramSummary:
    return EngramSummary(
        engram_id=UUID(engram_id),
        project_id="proj-a",
        thread_id=None,
        title=f"Title-{engram_id[-4:]}",
        abstract="Abstract",
        created_at=created_at,
        tags=tags or [],
        keywords=keywords or [],
    )


def test_consolidation_dry_run(monkeypatch) -> None:
    now = datetime(2026, 2, 15, tzinfo=UTC)
    rows = [
        _summary("11111111-1111-1111-1111-111111111111", now),
        _summary("22222222-2222-2222-2222-222222222222", now - timedelta(hours=1)),
        _summary("33333333-3333-3333-3333-333333333333", now - timedelta(hours=2)),
    ]

    monkeypatch.setattr("app.consolidation.list_engrams", lambda **_kwargs: rows)

    result = run_project_consolidation(project_id="proj-a", dry_run=True)

    assert result["created"] is False
    assert result["reason"] == "dry_run"
    assert result["source_count"] == 3
    assert result["proposed_title"].startswith("Consolidated Memory Snapshot")


def test_consolidation_persists_engram(monkeypatch) -> None:
    now = datetime(2026, 2, 15, tzinfo=UTC)
    rows = [
        _summary("44444444-4444-4444-4444-444444444444", now),
        _summary("55555555-5555-5555-5555-555555555555", now - timedelta(hours=1)),
        _summary("66666666-6666-6666-6666-666666666666", now - timedelta(hours=2)),
    ]

    captured: dict[str, object] = {}

    def fake_create(payload, embedding_dim, enrichment_origin="unknown"):  # noqa: ANN001
        captured["payload"] = payload
        captured["embedding_dim"] = embedding_dim
        captured["enrichment_origin"] = enrichment_origin
        return EngramCreateResponse(
            engram_id=UUID("77777777-7777-7777-7777-777777777777"),
            created_at=now,
        )

    monkeypatch.setattr("app.consolidation.list_engrams", lambda **_kwargs: rows)
    monkeypatch.setattr("app.consolidation.create_engram", fake_create)
    monkeypatch.setattr("app.consolidation.get_settings", lambda: SimpleNamespace(embedding_dim=12))

    result = run_project_consolidation(project_id="proj-a")

    assert result["created"] is True
    assert result["engram_id"] == "77777777-7777-7777-7777-777777777777"
    assert captured["embedding_dim"] == 12
    assert captured["enrichment_origin"] == "consolidation.auto"
    assert "consolidated" in captured["payload"].tags


def test_consolidation_skips_recent_consolidation(monkeypatch) -> None:
    now = datetime.now(UTC)
    rows = [
        _summary(
            "88888888-8888-8888-8888-888888888888",
            now - timedelta(minutes=30),
            tags=["consolidated"],
        ),
        _summary("99999999-9999-9999-9999-999999999999", now - timedelta(hours=1)),
        _summary("aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa", now - timedelta(hours=2)),
    ]

    monkeypatch.setattr("app.consolidation.list_engrams", lambda **_kwargs: rows)

    result = run_project_consolidation(project_id="proj-a")

    assert result["created"] is False
    assert result["reason"] == "recent_consolidation_exists"
