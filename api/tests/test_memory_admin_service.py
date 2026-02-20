from __future__ import annotations

from datetime import UTC, datetime
from types import SimpleNamespace
from uuid import uuid4

import pytest
from fastapi import HTTPException

from app.memory_admin.repository import AdminEngramUpdateRepositoryRequest
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


@pytest.mark.parametrize(
    ("repository_method", "service_method"),
    [
        ("list_admin_sessions", "list_sessions"),
        ("list_collections", "list_collections"),
    ],
)
def test_list_methods_forward_shared_request_object(
    monkeypatch,
    repository_method: str,
    service_method: str,
) -> None:
    captured: dict[str, object] = {}

    def _fake_list_records(**kwargs):  # noqa: ANN003
        captured.update(kwargs)
        return []

    monkeypatch.setattr(f"app.memory_admin.service.{repository_method}", _fake_list_records)

    service = _service()
    request = MemoryAdminListRequest(
        project_id="engram-vault",
        owner_user_id=uuid4(),
        include_deleted=False,
        limit=50,
        offset=10,
    )

    listed = getattr(service, service_method)(request=request)
    assert listed == []
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


def test_update_engram_uses_repository_request_object(monkeypatch) -> None:
    service = _service()
    now = datetime.now(UTC)
    engram_id = uuid4()
    actor_user_id = uuid4()
    captured: dict[str, object] = {}

    current = SimpleNamespace(
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
        sources=[],
        created_at=now,
        updated_at=now,
        deleted_at=None,
        deleted_by_user_id=None,
        delete_reason=None,
    )

    monkeypatch.setattr("app.memory_admin.service.get_admin_engram", lambda **kwargs: current)

    def _fake_update_admin_engram(**kwargs):  # noqa: ANN003
        captured.update(kwargs)
        return current

    monkeypatch.setattr("app.memory_admin.service.update_admin_engram", _fake_update_admin_engram)

    payload = SimpleNamespace(
        expected_updated_at=now,
        title="Updated",
        abstract="updated",
        detailed_summary_markdown="updated",
        tags=["tag-1"],
        keywords=["keyword-1"],
        visibility_scope="private",
        sources=[],
    )

    updated = service.update_engram(
        engram_id=engram_id,
        actor_user_id=actor_user_id,
        payload=payload,
    )

    assert updated is current
    assert captured["engram_id"] == engram_id
    request = captured["request"]
    assert isinstance(request, AdminEngramUpdateRepositoryRequest)
    assert request.actor_user_id == actor_user_id
    assert request.title == "Updated"
    assert request.tags == ["tag-1"]
    assert request.embedding_dim == 256
