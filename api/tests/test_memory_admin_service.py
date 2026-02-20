from __future__ import annotations

from datetime import UTC, datetime
from types import SimpleNamespace
from uuid import uuid4

import pytest
from fastapi import HTTPException

from app.memory_admin.service import (
    MemoryAdminEngramListRequest,
    MemoryAdminListRequest,
    MemoryAdminService,
)
from app.models import AdminEngramSourceRecord


def _service() -> MemoryAdminService:
    return MemoryAdminService(
        embedding_dim=256,
        project_service=SimpleNamespace(
            resolve_project_id_for_write=lambda **kwargs: SimpleNamespace(
                project_id=kwargs["project_id"],
                used_default_project=False,
            )
        ),
    )


def test_list_sessions_uses_request_object(monkeypatch) -> None:
    captured: dict[str, object] = {}

    def _fake_list_admin_sessions(**kwargs):  # noqa: ANN003
        captured.update(kwargs)
        return []

    monkeypatch.setattr("app.memory_admin.service.list_admin_sessions", _fake_list_admin_sessions)

    service = _service()
    request = MemoryAdminListRequest(
        project_id="engram-vault",
        owner_user_id=uuid4(),
        include_deleted=False,
        limit=50,
        offset=10,
    )

    assert service.list_sessions(request=request) == []
    assert captured == {
        "project_id": request.project_id,
        "owner_user_id": request.owner_user_id,
        "include_deleted": False,
        "limit": 50,
        "offset": 10,
    }


def test_list_engrams_uses_request_object(monkeypatch) -> None:
    captured: dict[str, object] = {}

    def _fake_list_admin_engrams(**kwargs):  # noqa: ANN003
        captured.update(kwargs)
        return []

    monkeypatch.setattr("app.memory_admin.service.list_admin_engrams", _fake_list_admin_engrams)

    service = _service()
    request = MemoryAdminEngramListRequest(
        project_id="engram-vault",
        owner_user_id=None,
        include_deleted=True,
        limit=25,
        offset=5,
        session_id=uuid4(),
        query_text="incident",
    )

    assert service.list_engrams(request=request) == []
    assert captured["project_id"] == "engram-vault"
    assert captured["session_id"] == request.session_id
    assert captured["query_text"] == "incident"
    assert captured["include_deleted"] is True


def test_list_collections_uses_shared_list_helper(monkeypatch) -> None:
    captured: dict[str, object] = {}

    def _fake_list_collections(**kwargs):  # noqa: ANN003
        captured.update(kwargs)
        return []

    monkeypatch.setattr("app.memory_admin.service.list_collections", _fake_list_collections)

    service = _service()
    request = MemoryAdminListRequest(
        project_id="engram-vault",
        owner_user_id=uuid4(),
        include_deleted=True,
        limit=100,
        offset=0,
    )

    assert service.list_collections(request=request) == []
    assert captured["project_id"] == "engram-vault"
    assert captured["include_deleted"] is True


def test_update_collection_rejects_stale_expected_updated_at(monkeypatch) -> None:
    service = _service()
    now = datetime.now(UTC)
    collection_id = uuid4()

    monkeypatch.setattr(
        "app.memory_admin.service.get_collection",
        lambda **kwargs: SimpleNamespace(
            collection_id=collection_id,
            updated_at=now,
            name="Collection",
            description="desc",
            project_id="engram-vault",
            owner_user_id=uuid4(),
            created_at=now,
            deleted_at=None,
            deleted_by_user_id=None,
            delete_reason=None,
            item_count=0,
        ),
    )

    with pytest.raises(HTTPException) as exc_info:
        service.update_collection(
            collection_id=collection_id,
            payload=SimpleNamespace(
                expected_updated_at=datetime(2026, 1, 1, tzinfo=UTC),
                name="Updated",
                description="updated",
            ),
        )

    assert exc_info.value.status_code == 409


def test_update_engram_rejects_stale_expected_updated_at(monkeypatch) -> None:
    service = _service()
    now = datetime.now(UTC)
    engram_id = uuid4()

    monkeypatch.setattr(
        "app.memory_admin.service.get_admin_engram",
        lambda **kwargs: SimpleNamespace(
            engram_id=engram_id,
            project_id="engram-vault",
            owner_user_id=uuid4(),
            thread_id=None,
            title="Engram",
            abstract="",
            detailed_summary_markdown="",
            tags=[],
            keywords=[],
            visibility_scope="private",
            source_session_id=None,
            sources=[
                AdminEngramSourceRecord(
                    source_id=uuid4(),
                    captured_at=now,
                    url="https://example.com",
                    title="Source",
                    snippet="Snippet",
                )
            ],
            created_at=now,
            updated_at=now,
            deleted_at=None,
            deleted_by_user_id=None,
            delete_reason=None,
        ),
    )

    with pytest.raises(HTTPException) as exc_info:
        service.update_engram(
            engram_id=engram_id,
            actor_user_id=uuid4(),
            payload=SimpleNamespace(
                expected_updated_at=datetime(2026, 1, 1, tzinfo=UTC),
                title="Updated",
                abstract="updated",
                detailed_summary_markdown="updated",
                tags=[],
                keywords=[],
                visibility_scope="private",
                sources=[],
            ),
        )

    assert exc_info.value.status_code == 409
