from __future__ import annotations

from uuid import uuid4

import pytest

from app.auth import hash_password
from app.chat_repository import (
    create_chat_message,
    create_chat_session,
    list_chat_messages,
    list_chat_sessions,
    list_pinned_documents,
    list_pinned_engram_summaries,
    pin_document_to_session,
    pin_engram_to_session,
    unpin_document_from_session,
    unpin_engram_from_session,
    update_chat_session,
)
from app.config import get_settings
from app.ingestion.chunking import build_content_hash, build_document_id, chunk_document_text
from app.ingestion.repository import upsert_document_with_chunks
from app.models import (
    ChatSessionCreateRequest,
    ChatSessionUpdateRequest,
    MemoryEngramCreate,
    UserRole,
)
from app.repository import create_engram
from app.user_repository import create_user, get_user_auth_record


@pytest.mark.integration
def test_chat_session_create_list_update(clean_db) -> None:
    admin = get_user_auth_record(get_settings().ui_demo_username)
    assert admin is not None

    created = create_chat_session(
        owner_user_id=admin["user_id"],
        payload=ChatSessionCreateRequest(
            project_id="project-chat",
            title="Session A",
            provider="openai",
            model_id="gpt-4o-mini",
            system_prompt="You are helpful.",
            visibility_scope="private",
        ),
    )
    assert created.title == "Session A"

    listed = list_chat_sessions(actor_user_id=admin["user_id"], project_id="project-chat")
    assert len(listed) == 1
    assert listed[0].session_id == created.session_id

    updated = update_chat_session(
        session_id=created.session_id,
        actor_user_id=admin["user_id"],
        payload=ChatSessionUpdateRequest(title="Session A Updated", autosave_enabled=True),
    )
    assert updated is not None
    assert updated.title == "Session A Updated"
    assert updated.autosave_enabled is True


@pytest.mark.integration
def test_chat_message_create_and_list(clean_db) -> None:
    admin = get_user_auth_record(get_settings().ui_demo_username)
    assert admin is not None

    session = create_chat_session(
        owner_user_id=admin["user_id"],
        payload=ChatSessionCreateRequest(
            project_id="project-msg",
            title="Session M",
            provider="openai",
            model_id="gpt-4o-mini",
            visibility_scope="private",
        ),
    )

    msg = create_chat_message(
        session_id=session.session_id,
        actor_user_id=admin["user_id"],
        role="user",
        content_text="Hello",
        provider="openai",
        model_id="gpt-4o-mini",
        token_usage_json={"input": 3, "output": 0},
    )
    assert msg is not None
    assert msg.content_text == "Hello"

    rows = list_chat_messages(session_id=session.session_id, actor_user_id=admin["user_id"])
    assert len(rows) == 1
    assert rows[0].message_id == msg.message_id


@pytest.mark.integration
def test_pin_and_unpin_engram(clean_db) -> None:
    admin = get_user_auth_record(get_settings().ui_demo_username)
    assert admin is not None

    session = create_chat_session(
        owner_user_id=admin["user_id"],
        payload=ChatSessionCreateRequest(
            project_id="project-pin",
            title="Session P",
            provider="openai",
            model_id="gpt-4o-mini",
            visibility_scope="private",
        ),
    )

    payload = MemoryEngramCreate(
        project_id="project-pin",
        thread_id="thread-pin",
        title="Pinned Engram",
        abstract="Pinned for context",
        detailed_summary_markdown="Summary",
    )
    created = create_engram(
        payload, embedding_dim=get_settings().embedding_dim, owner_user_id=admin["user_id"]
    )

    pinned = pin_engram_to_session(
        session_id=session.session_id,
        engram_id=created.engram_id,
        actor_user_id=admin["user_id"],
    )
    assert pinned is not None

    summaries = list_pinned_engram_summaries(session.session_id, actor_user_id=admin["user_id"])
    assert len(summaries) == 1
    assert summaries[0].engram_id == created.engram_id

    removed = unpin_engram_from_session(
        session_id=session.session_id,
        engram_id=created.engram_id,
        actor_user_id=admin["user_id"],
    )
    assert removed is True

    summaries_after = list_pinned_engram_summaries(
        session.session_id, actor_user_id=admin["user_id"]
    )
    assert summaries_after == []


@pytest.mark.integration
def test_pin_and_unpin_document(clean_db) -> None:
    admin = get_user_auth_record(get_settings().ui_demo_username)
    assert admin is not None

    settings = get_settings()
    session = create_chat_session(
        owner_user_id=admin["user_id"],
        payload=ChatSessionCreateRequest(
            project_id="project-doc-pin",
            title="Session D",
            provider="openai",
            model_id="gpt-4o-mini",
            visibility_scope="private",
        ),
    )

    text = "Queue depth exceeded threshold. Start dequeue worker burst and monitor retries."
    content_hash = build_content_hash(text)
    document_id = build_document_id(
        owner_user_id=admin["user_id"],
        project_id="project-doc-pin",
        title="Runbook Document",
        content_hash=content_hash,
    )
    chunks = chunk_document_text(
        content_hash=content_hash,
        text=text,
        chunk_size_chars=500,
        chunk_overlap_chars=80,
    )
    upsert_document_with_chunks(
        document_id=document_id,
        actor_user_id=admin["user_id"],
        project_id="project-doc-pin",
        title="Runbook Document",
        source_type="text",
        source_name=None,
        mime_type="text/plain",
        visibility_scope="project",
        content_text=text,
        content_hash=content_hash,
        metadata={},
        chunks=chunks,
        embedding_dim=settings.embedding_dim,
    )

    pinned = pin_document_to_session(
        session_id=session.session_id,
        document_id=document_id,
        actor_user_id=admin["user_id"],
    )
    assert pinned is not None
    assert pinned.document_id == document_id

    listed = list_pinned_documents(
        session_id=session.session_id,
        actor_user_id=admin["user_id"],
    )
    assert len(listed) == 1
    assert listed[0].document_id == document_id

    removed = unpin_document_from_session(
        session_id=session.session_id,
        document_id=document_id,
        actor_user_id=admin["user_id"],
    )
    assert removed is True

    listed_after = list_pinned_documents(
        session_id=session.session_id,
        actor_user_id=admin["user_id"],
    )
    assert listed_after == []


@pytest.mark.integration
def test_visibility_filter_private_vs_project(clean_db) -> None:
    admin = get_user_auth_record(get_settings().ui_demo_username)
    assert admin is not None

    analyst = create_user(
        username=f"analyst-{uuid4().hex[:8]}",
        password_hash=hash_password("analyst-pass-123"),
        role=UserRole.analyst,
        is_active=True,
    )

    _private_session = create_chat_session(
        owner_user_id=admin["user_id"],
        payload=ChatSessionCreateRequest(
            project_id="project-visibility",
            title="Private Session",
            provider="openai",
            model_id="gpt-4o-mini",
            visibility_scope="private",
        ),
    )

    shared_session = create_chat_session(
        owner_user_id=admin["user_id"],
        payload=ChatSessionCreateRequest(
            project_id="project-visibility",
            title="Project Session",
            provider="openai",
            model_id="gpt-4o-mini",
            visibility_scope="project",
        ),
    )

    visible_to_analyst = list_chat_sessions(
        actor_user_id=analyst.user_id,
        project_id="project-visibility",
    )

    ids = {item.session_id for item in visible_to_analyst}
    assert shared_session.session_id in ids
    assert len(ids) == 1


@pytest.mark.integration
def test_cannot_pin_private_engram_of_other_user(clean_db) -> None:
    admin = get_user_auth_record(get_settings().ui_demo_username)
    assert admin is not None

    analyst = create_user(
        username=f"viewer-{uuid4().hex[:8]}",
        password_hash=hash_password("viewer-pass-123"),
        role=UserRole.viewer,
        is_active=True,
    )

    shared_session = create_chat_session(
        owner_user_id=analyst.user_id,
        payload=ChatSessionCreateRequest(
            project_id="project-enforce",
            title="Shared Session",
            provider="openai",
            model_id="gpt-4o-mini",
            visibility_scope="project",
        ),
    )

    private_engram = create_engram(
        MemoryEngramCreate(
            project_id="project-enforce",
            thread_id="thread-private",
            title="Private Engram",
            abstract="private",
            detailed_summary_markdown="private",
            visibility_scope="private",
        ),
        embedding_dim=get_settings().embedding_dim,
        owner_user_id=admin["user_id"],
    )

    pinned = pin_engram_to_session(
        session_id=shared_session.session_id,
        engram_id=private_engram.engram_id,
        actor_user_id=analyst.user_id,
    )
    assert pinned is None
