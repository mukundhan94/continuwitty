from __future__ import annotations

from datetime import UTC, datetime
from uuid import uuid4

from app.chat.context import assemble_chat_context
from app.ingestion.models import DocumentChunkQueryResult
from app.models import ChatSessionRecord, EngramQueryResult, EngramSummary, RehydrationBundle


def _session() -> ChatSessionRecord:
    now = datetime.now(UTC)
    return ChatSessionRecord(
        session_id=uuid4(),
        owner_user_id=uuid4(),
        project_id="project-chat",
        title="Session",
        provider="openai",
        model_id="gpt-4o-mini",
        system_prompt="",
        visibility_scope="private",
        autosave_enabled=False,
        created_at=now,
        updated_at=now,
    )


def _bundle(engram_id, title: str) -> RehydrationBundle:
    now = datetime.now(UTC)
    return RehydrationBundle(
        engram_id=engram_id,
        project_id="project-chat",
        title=title,
        compact_summary=f"{title} summary",
        detailed_summary_markdown=f"{title} detailed notes",
        key_decisions=[{"decision": f"{title} decision", "rationale": "because"}],
        open_questions=[f"{title} question"],
        top_citations=[
            {
                "url": f"https://example.com/{title.lower()}",
                "title": f"{title} Source",
                "snippet": f"{title} snippet",
                "captured_at": now,
            }
        ],
        context_markdown="ctx",
        owner_user_id=uuid4(),
        visibility_scope="private",
    )


def test_assemble_chat_context_merges_pinned_and_retrieved(monkeypatch) -> None:
    session = _session()
    pinned_id = uuid4()
    retrieved_id = uuid4()

    monkeypatch.setattr(
        "app.chat.context.list_pinned_engram_summaries",
        lambda session_id, actor_user_id: [
            EngramSummary(
                engram_id=pinned_id,
                project_id="project-chat",
                thread_id="thread-1",
                title="Pinned",
                abstract="Pinned abstract",
                created_at=datetime.now(UTC),
                tags=[],
                keywords=[],
                owner_user_id=actor_user_id,
                visibility_scope="private",
            )
        ],
    )
    monkeypatch.setattr(
        "app.chat.context.query_engrams",
        lambda request, embedding_dim, actor_user_id: [
            EngramQueryResult(
                engram_id=retrieved_id,
                project_id="project-chat",
                title="Retrieved",
                abstract="Retrieved abstract",
                created_at=datetime.now(UTC),
                tags=[],
                keywords=[],
                owner_user_id=actor_user_id,
                visibility_scope="private",
                distance=0.1,
            ),
            EngramQueryResult(
                engram_id=pinned_id,
                project_id="project-chat",
                title="Pinned",
                abstract="Pinned abstract",
                created_at=datetime.now(UTC),
                tags=[],
                keywords=[],
                owner_user_id=actor_user_id,
                visibility_scope="private",
                distance=0.2,
            ),
        ],
    )
    monkeypatch.setattr(
        "app.chat.context.get_rehydration_bundle",
        lambda engram_id, actor_user_id: _bundle(
            engram_id=engram_id,
            title="Pinned" if engram_id == pinned_id else "Retrieved",
        ),
    )
    document_chunk_id = uuid4()
    monkeypatch.setattr(
        "app.chat.context.query_document_chunks",
        lambda actor_user_id, request, embedding_dim: [
            DocumentChunkQueryResult(
                chunk_id=document_chunk_id,
                document_id=uuid4(),
                project_id="project-chat",
                title="Runbook Notes",
                source_name="runbook.md",
                chunk_index=0,
                snippet="Escalate when queue depth remains high for 20 minutes.",
                created_at=datetime.now(UTC),
                visibility_scope="project",
                distance=0.2,
            )
        ],
    )

    assembled = assemble_chat_context(
        session=session,
        actor_user_id=session.owner_user_id,
        user_query="what should we do",
        embedding_dim=256,
    )

    assert assembled.used_engram_ids == [pinned_id, retrieved_id]
    assert assembled.used_document_chunk_ids == [document_chunk_id]
    assert len(assembled.source_references) == 3
    assert "Engram Retrieval Context" in assembled.context_markdown
    assert "Pinned summary" in assembled.context_markdown
    assert "Pinned detailed notes" in assembled.context_markdown
    assert "Retrieved summary" in assembled.context_markdown
    assert "Document Retrieval Context" in assembled.context_markdown
