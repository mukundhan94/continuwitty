from __future__ import annotations

from datetime import UTC, datetime
from uuid import uuid4

from app.chat.context import ChatContextRequest, _bundle_section, assemble_chat_context
from app.ingestion.models import DocumentChunkQueryResult
from app.models import (
    ChatSessionRecord,
    EngramQueryResult,
    EngramSummary,
    PinnedDocumentRecord,
    RehydrationBundle,
)


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


def _context_request(
    session: ChatSessionRecord,
    *,
    query: str,
    embedding_dim: int = 256,
    document_top_k: int = 4,
) -> ChatContextRequest:
    return ChatContextRequest(
        session=session,
        actor_user_id=session.owner_user_id,
        user_query=query,
        embedding_dim=embedding_dim,
        document_top_k=document_top_k,
    )


def _bundle(engram_id, title: str, citation_url: str | None = None) -> RehydrationBundle:
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
                "url": citation_url or f"https://example.com/{title.lower()}",
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
    pinned_document_id = uuid4()

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
        "app.chat.context.list_pinned_documents",
        lambda session_id, actor_user_id: [
            PinnedDocumentRecord(
                session_id=session_id,
                document_id=pinned_document_id,
                pinned_by_user_id=actor_user_id,
                created_at=datetime.now(UTC),
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
    pinned_chunk_id = uuid4()
    retrieved_chunk_id = uuid4()

    def _fake_query_document_chunks(actor_user_id, request, embedding_dim):  # noqa: ANN001
        if request.document_ids:
            assert request.document_ids == [pinned_document_id]
            return [
                DocumentChunkQueryResult(
                    chunk_id=pinned_chunk_id,
                    document_id=pinned_document_id,
                    project_id="project-chat",
                    title="Pinned Runbook",
                    source_name="runbook.md",
                    chunk_index=0,
                    snippet="Pinned runbook excerpt for immediate response context.",
                    created_at=datetime.now(UTC),
                    visibility_scope="project",
                    distance=0.1,
                )
            ]
        return [
            DocumentChunkQueryResult(
                chunk_id=retrieved_chunk_id,
                document_id=uuid4(),
                project_id="project-chat",
                title="Retrieved Incident Notes",
                source_name="incident.md",
                chunk_index=2,
                snippet="Escalate when queue depth remains high for 20 minutes.",
                created_at=datetime.now(UTC),
                visibility_scope="project",
                distance=0.2,
            )
        ]

    monkeypatch.setattr("app.chat.context.query_document_chunks", _fake_query_document_chunks)

    assembled = assemble_chat_context(
        request=_context_request(session, query="what should we do"),
    )

    assert assembled.used_engram_ids == [pinned_id, retrieved_id]
    assert assembled.used_document_chunk_ids == [pinned_chunk_id, retrieved_chunk_id]
    assert len(assembled.source_references) == 4
    assert "Engram Retrieval Context" in assembled.context_markdown
    assert "Pinned summary" in assembled.context_markdown
    assert "Pinned detailed notes" in assembled.context_markdown
    assert "Retrieved summary" in assembled.context_markdown
    assert "Pinned Document Context" in assembled.context_markdown
    assert "Document Retrieval Context" in assembled.context_markdown


def test_assemble_chat_context_dedupes_duplicate_source_urls(monkeypatch) -> None:
    session = _session()
    pinned_id = uuid4()
    retrieved_id = uuid4()
    shared_url = "https://example.com/shared-runbook"

    monkeypatch.setattr("app.chat.context.list_pinned_documents", lambda *args, **kwargs: [])
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
            )
        ],
    )

    def _fake_bundle(engram_id, actor_user_id):  # noqa: ANN001
        title = "Pinned" if engram_id == pinned_id else "Retrieved"
        return _bundle(engram_id=engram_id, title=title, citation_url=shared_url)

    monkeypatch.setattr("app.chat.context.get_rehydration_bundle", _fake_bundle)
    monkeypatch.setattr("app.chat.context.query_document_chunks", lambda *args, **kwargs: [])

    assembled = assemble_chat_context(
        request=_context_request(session, query="what should we do"),
    )

    assert assembled.used_engram_ids == [pinned_id, retrieved_id]
    assert len(assembled.source_references) == 1
    assert assembled.source_references[0].url == shared_url


def test_assemble_chat_context_dedupes_multiple_chunks_from_same_document(monkeypatch) -> None:
    session = _session()
    engram_id = uuid4()
    shared_document_id = uuid4()

    monkeypatch.setattr("app.chat.context.list_pinned_documents", lambda *args, **kwargs: [])
    monkeypatch.setattr(
        "app.chat.context.list_pinned_engram_summaries",
        lambda session_id, actor_user_id: [
            EngramSummary(
                engram_id=engram_id,
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
    monkeypatch.setattr("app.chat.context.query_engrams", lambda *args, **kwargs: [])
    monkeypatch.setattr(
        "app.chat.context.get_rehydration_bundle",
        lambda engram_id, actor_user_id: _bundle(engram_id=engram_id, title="Pinned"),
    )

    def _fake_query_document_chunks(actor_user_id, request, embedding_dim):  # noqa: ANN001
        return [
            DocumentChunkQueryResult(
                chunk_id=uuid4(),
                document_id=shared_document_id,
                project_id="project-chat",
                title="AGENT",
                source_name="AGENT.md",
                chunk_index=0,
                snippet="chunk 0",
                created_at=datetime.now(UTC),
                visibility_scope="project",
                distance=0.1,
            ),
            DocumentChunkQueryResult(
                chunk_id=uuid4(),
                document_id=shared_document_id,
                project_id="project-chat",
                title="AGENT",
                source_name="AGENT.md",
                chunk_index=1,
                snippet="chunk 1",
                created_at=datetime.now(UTC),
                visibility_scope="project",
                distance=0.2,
            ),
        ]

    monkeypatch.setattr("app.chat.context.query_document_chunks", _fake_query_document_chunks)

    assembled = assemble_chat_context(
        request=_context_request(session, query="agent workflow", document_top_k=4),
    )

    document_refs = [
        ref for ref in assembled.source_references if ref.source_type == "document_chunk"
    ]
    assert len(document_refs) == 1
    assert document_refs[0].document_id == shared_document_id
    assert document_refs[0].title == "AGENT.md"


def test_assemble_chat_context_uses_all_pinned_documents(monkeypatch) -> None:
    session = _session()
    pinned_document_a = uuid4()
    pinned_document_b = uuid4()
    chunk_a = uuid4()
    chunk_b = uuid4()

    monkeypatch.setattr("app.chat.context.list_pinned_engram_summaries", lambda *args, **kwargs: [])
    monkeypatch.setattr(
        "app.chat.context.list_pinned_documents",
        lambda session_id, actor_user_id: [
            PinnedDocumentRecord(
                session_id=session_id,
                document_id=pinned_document_a,
                pinned_by_user_id=actor_user_id,
                created_at=datetime.now(UTC),
            ),
            PinnedDocumentRecord(
                session_id=session_id,
                document_id=pinned_document_b,
                pinned_by_user_id=actor_user_id,
                created_at=datetime.now(UTC),
            ),
        ],
    )
    monkeypatch.setattr("app.chat.context.query_engrams", lambda *args, **kwargs: [])
    monkeypatch.setattr("app.chat.context.get_rehydration_bundle", lambda *args, **kwargs: None)

    def _fake_query_document_chunks(actor_user_id, request, embedding_dim):  # noqa: ANN001
        if request.document_ids == [pinned_document_a]:
            return [
                DocumentChunkQueryResult(
                    chunk_id=chunk_a,
                    document_id=pinned_document_a,
                    project_id="project-chat",
                    title="Doc A",
                    source_name="doc-a.md",
                    chunk_index=0,
                    snippet="doc a",
                    created_at=datetime.now(UTC),
                    visibility_scope="project",
                    distance=0.3,
                )
            ]
        if request.document_ids == [pinned_document_b]:
            return [
                DocumentChunkQueryResult(
                    chunk_id=chunk_b,
                    document_id=pinned_document_b,
                    project_id="project-chat",
                    title="Doc B",
                    source_name="doc-b.md",
                    chunk_index=1,
                    snippet="doc b",
                    created_at=datetime.now(UTC),
                    visibility_scope="project",
                    distance=0.4,
                )
            ]
        return []

    monkeypatch.setattr("app.chat.context.query_document_chunks", _fake_query_document_chunks)

    assembled = assemble_chat_context(
        request=_context_request(session, query="run checks", document_top_k=1),
    )

    assert assembled.used_document_chunk_ids == [chunk_a, chunk_b]
    assert "Pinned Document Context" in assembled.context_markdown
    document_refs = [
        ref for ref in assembled.source_references if ref.source_type == "document_chunk"
    ]
    assert {ref.document_id for ref in document_refs} == {pinned_document_a, pinned_document_b}


def test_bundle_section_truncates_detailed_excerpt() -> None:
    engram_id = uuid4()
    bundle = _bundle(engram_id=engram_id, title="Long")
    bundle = bundle.model_copy(
        update={
            "detailed_summary_markdown": "x" * 1300,
        }
    )

    section = _bundle_section(bundle)

    assert "Detailed notes excerpt:\n" in section
    assert ("x" * 1200) not in section
    assert "..." in section


def test_assemble_chat_context_returns_empty_when_no_sources(monkeypatch) -> None:
    session = _session()
    monkeypatch.setattr("app.chat.context.list_pinned_engram_summaries", lambda *args, **kwargs: [])
    monkeypatch.setattr("app.chat.context.list_pinned_documents", lambda *args, **kwargs: [])
    monkeypatch.setattr("app.chat.context.query_engrams", lambda *args, **kwargs: [])
    monkeypatch.setattr("app.chat.context.get_rehydration_bundle", lambda *args, **kwargs: None)
    monkeypatch.setattr("app.chat.context.query_document_chunks", lambda *args, **kwargs: [])

    assembled = assemble_chat_context(request=_context_request(session, query="no data"))

    assert assembled.context_markdown == ""
    assert assembled.used_engram_ids == []
    assert assembled.used_document_chunk_ids == []
    assert assembled.source_references == []
