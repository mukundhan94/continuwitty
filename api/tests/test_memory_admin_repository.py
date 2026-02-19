from __future__ import annotations

from datetime import UTC, datetime
from types import SimpleNamespace
from uuid import UUID, uuid4

import pytest

from app.chat_repository import create_chat_session
from app.config import get_settings
from app.memory_admin import repository as memory_repo
from app.models import (
    AdminEngramRecord,
    AdminEngramSourceInput,
    ChatSessionCreateRequest,
    MemoryEngramCreate,
    VisibilityScope,
)
from app.repository import create_engram
from app.user_repository import get_user_auth_record


def _sample_admin_engram() -> AdminEngramRecord:
    now = datetime(2026, 2, 19, tzinfo=UTC)
    return AdminEngramRecord(
        engram_id=uuid4(),
        project_id="engram-vault",
        thread_id="thread-1",
        title="Current title",
        abstract="Current abstract",
        detailed_summary_markdown="Current markdown",
        tags=["existing"],
        keywords=["keyword"],
        owner_user_id=uuid4(),
        visibility_scope="private",
        source_session_id=None,
        created_at=now,
        updated_at=now,
        sources=[],
    )


def test_build_engram_update_fields_uses_payload_values_and_defaults() -> None:
    current = _sample_admin_engram()
    payload = memory_repo._AdminEngramUpdatePayload(
        title="  Updated title  ",
        abstract=None,
        detailed_summary_markdown="Updated markdown",
        tags=["new-tag"],
        keywords=None,
        visibility_scope=VisibilityScope.project,
    )

    fields = memory_repo._build_engram_update_fields(current=current, payload=payload)

    assert fields["title"] == "Updated title"
    assert fields["abstract"] == current.abstract
    assert fields["detailed_summary_markdown"] == "Updated markdown"
    assert fields["tags"] == ["new-tag"]
    assert fields["keywords"] == current.keywords
    assert fields["visibility_scope"] == "project"


def test_compute_engram_embedding_formats_vector_literal(monkeypatch) -> None:
    called_with: dict[str, object] = {}

    def _fake_embed_text(text: str, *, dim: int) -> SimpleNamespace:
        called_with["text"] = text
        called_with["dim"] = dim
        return SimpleNamespace(provider_id="deterministic-local", vector=[1.0, -2.3456789])

    monkeypatch.setattr(memory_repo, "embed_text", _fake_embed_text)

    provider_id, vector_literal = memory_repo._compute_engram_embedding(
        retrieval_text="embedding text",
        embedding_dim=384,
    )

    assert called_with == {"text": "embedding text", "dim": 384}
    assert provider_id == "deterministic-local"
    assert vector_literal == "[1.000000,-2.345679]"


def test_replace_engram_sources_replaces_all_rows() -> None:
    class _CursorSpy:
        def __init__(self) -> None:
            self.calls: list[tuple[str, tuple[object, ...]]] = []

        def execute(self, sql: str, params: tuple[object, ...]) -> None:
            self.calls.append((sql, params))

    engram_id = uuid4()
    captured_at = datetime(2026, 2, 19, tzinfo=UTC)
    sources = [
        AdminEngramSourceInput(
            captured_at=captured_at,
            url="https://example.com/1",
            title="one",
            snippet="snippet one",
        ),
        AdminEngramSourceInput(
            captured_at=captured_at,
            url="https://example.com/2",
            title="two",
            snippet="snippet two",
            content_text="body",
            content_hash="hash-2",
        ),
    ]
    cursor = _CursorSpy()

    memory_repo._replace_engram_sources(cur=cursor, engram_id=engram_id, sources=sources)

    assert len(cursor.calls) == 3
    delete_sql, delete_params = cursor.calls[0]
    assert "DELETE FROM sources" in delete_sql
    assert delete_params == (engram_id,)
    for insert_sql, insert_params in cursor.calls[1:]:
        assert "INSERT INTO sources" in insert_sql
        assert isinstance(insert_params[0], UUID)
        assert insert_params[1] == engram_id


@pytest.mark.integration
def test_soft_delete_and_restore_lifecycle(clean_db) -> None:
    admin = get_user_auth_record(get_settings().ui_demo_username)
    assert admin is not None
    actor_user_id = admin["user_id"]

    session = create_chat_session(
        owner_user_id=actor_user_id,
        payload=ChatSessionCreateRequest(
            project_id="engram-vault",
            title="Repository lifecycle session",
            provider="openai",
            model_id="gpt-4o-mini",
            visibility_scope="private",
        ),
    )
    deleted_session = memory_repo.soft_delete_session(
        session_id=session.session_id,
        deleted_by_user_id=actor_user_id,
        reason="session cleanup",
    )
    assert deleted_session is not None
    assert deleted_session.deleted_at is not None
    assert deleted_session.deleted_by_user_id == actor_user_id
    assert memory_repo.restore_session(session_id=session.session_id) is True
    restored_session = memory_repo.get_admin_session(
        session_id=session.session_id,
        include_deleted=False,
    )
    assert restored_session is not None
    assert restored_session.deleted_at is None

    created_engram = create_engram(
        payload=MemoryEngramCreate(
            project_id="engram-vault",
            title="Repository lifecycle engram",
            abstract="Lifecycle",
            detailed_summary_markdown="Lifecycle markdown",
        ),
        embedding_dim=get_settings().embedding_dim,
        owner_user_id=actor_user_id,
    )
    assert memory_repo.soft_delete_engram(
        engram_id=created_engram.engram_id,
        deleted_by_user_id=actor_user_id,
        reason="engram cleanup",
    )
    deleted_engram = memory_repo.get_admin_engram(
        engram_id=created_engram.engram_id,
        include_deleted=True,
    )
    assert deleted_engram is not None
    assert deleted_engram.deleted_at is not None
    assert memory_repo.restore_engram(engram_id=created_engram.engram_id) is True
    restored_engram = memory_repo.get_admin_engram(
        engram_id=created_engram.engram_id,
        include_deleted=False,
    )
    assert restored_engram is not None
    assert restored_engram.deleted_at is None

    collection = memory_repo.create_collection(
        project_id="engram-vault",
        owner_user_id=actor_user_id,
        name="Lifecycle collection",
        description="Collection lifecycle",
    )
    assert memory_repo.soft_delete_collection(
        collection_id=collection.collection_id,
        deleted_by_user_id=actor_user_id,
        reason="collection cleanup",
    )
    deleted_collection = memory_repo.get_collection(
        collection_id=collection.collection_id,
        include_deleted=True,
    )
    assert deleted_collection is not None
    assert deleted_collection.deleted_at is not None
