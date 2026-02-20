from __future__ import annotations

from dataclasses import dataclass
from uuid import UUID

from app.chat_repository import list_pinned_documents, list_pinned_engram_summaries
from app.ingestion.models import DocumentChunkQueryRequest, DocumentChunkQueryResult
from app.ingestion.repository import query_document_chunks
from app.models import (
    ChatSessionRecord,
    ChatSourceReference,
    EngramQueryRequest,
    EngramQueryResult,
    EngramSummary,
    RehydrationBundle,
)
from app.repository import get_rehydration_bundle, query_engrams


@dataclass(frozen=True)
class AssembledChatContext:
    context_markdown: str
    used_engram_ids: list[UUID]
    used_document_chunk_ids: list[UUID]
    source_references: list[ChatSourceReference]


@dataclass(frozen=True)
class ChatContextRequest:
    session: ChatSessionRecord
    actor_user_id: UUID
    user_query: str
    embedding_dim: int
    max_engrams: int = 6
    retrieval_top_k: int = 4
    document_top_k: int = 4


def _dedupe_preserve_order(ids: list[UUID]) -> list[UUID]:
    seen: set[UUID] = set()
    ordered: list[UUID] = []
    for item in ids:
        if item in seen:
            continue
        seen.add(item)
        ordered.append(item)
    return ordered


def _dedupe_chunks_preserve_order(
    chunks: list[DocumentChunkQueryResult],
) -> list[DocumentChunkQueryResult]:
    seen: set[UUID] = set()
    ordered: list[DocumentChunkQueryResult] = []
    for chunk in chunks:
        if chunk.chunk_id in seen:
            continue
        seen.add(chunk.chunk_id)
        ordered.append(chunk)
    return ordered


def _collect_pinned_document_chunks(
    *,
    request: ChatContextRequest,
    pinned_document_ids: list[UUID],
) -> list[DocumentChunkQueryResult]:
    """Return one representative chunk per pinned document.

    We issue one scoped retrieval per pinned document so pinning N documents
    guarantees N document contributions (when each document has chunks).
    """

    chunks: list[DocumentChunkQueryResult] = []
    for document_id in _dedupe_preserve_order(pinned_document_ids):
        scoped = query_document_chunks(
            actor_user_id=request.actor_user_id,
            request=DocumentChunkQueryRequest(
                query=request.user_query,
                project_id=request.session.project_id,
                document_ids=[document_id],
                top_k=1,
            ),
            embedding_dim=request.embedding_dim,
        )
        if scoped:
            chunks.append(scoped[0])
    return chunks


def _format_list_section(lines: list[str], *, fallback: str = "- None") -> str:
    return "\n".join(lines) or fallback


def _truncate_detailed_excerpt(detailed_summary_markdown: str, max_chars: int = 1200) -> str:
    detailed_excerpt = detailed_summary_markdown.strip()
    if not detailed_excerpt:
        return "- None"
    if len(detailed_excerpt) > max_chars:
        return detailed_excerpt[: max_chars - 3].rstrip() + "..."
    return detailed_excerpt


def _citation_line(bundle: RehydrationBundle, index: int) -> str:
    citation = bundle.top_citations[index]
    return f"- {citation.title or citation.url} ({citation.url}): {citation.snippet[:140]}"


def _bundle_section(bundle: RehydrationBundle) -> str:
    detailed_excerpt = _truncate_detailed_excerpt(bundle.detailed_summary_markdown)
    decisions = _format_list_section(
        [f"- {item.decision}: {item.rationale}" for item in bundle.key_decisions[:3]]
    )
    questions = _format_list_section([f"- {item}" for item in bundle.open_questions[:3]])
    citations = _format_list_section(
        [_citation_line(bundle, index) for index in range(min(3, len(bundle.top_citations)))]
    )
    return (
        f"## {bundle.title} ({bundle.engram_id})\n"
        f"Summary: {bundle.compact_summary}\n\n"
        f"Detailed notes excerpt:\n{detailed_excerpt}\n\n"
        f"Key decisions:\n{decisions}\n\n"
        f"Open questions:\n{questions}\n\n"
        f"Top citations:\n{citations}"
    )


def _collect_source_references(
    bundles: list[RehydrationBundle],
    per_engram_limit: int = 3,
    global_limit: int = 12,
) -> list[ChatSourceReference]:
    references: list[ChatSourceReference] = []
    seen: set[str] = set()
    for bundle in bundles:
        for citation in bundle.top_citations[:per_engram_limit]:
            # Dedupe by canonical URL across all bundles so the UI/source strip
            # does not show the same source repeatedly when multiple engrams
            # cite identical material.
            key = citation.url.lower().strip()
            if key in seen:
                continue
            seen.add(key)
            snippet = citation.snippet.strip().replace("\n", " ")
            if len(snippet) > 240:
                snippet = snippet[:237] + "..."
            references.append(
                ChatSourceReference(
                    engram_id=bundle.engram_id,
                    engram_title=bundle.title,
                    url=citation.url,
                    title=citation.title,
                    snippet=snippet,
                    captured_at=citation.captured_at,
                )
            )
            if len(references) >= global_limit:
                return references
    return references


def _dedupe_source_references(
    references: list[ChatSourceReference],
    limit: int = 16,
) -> list[ChatSourceReference]:
    seen: set[tuple[str, str]] = set()
    deduped: list[ChatSourceReference] = []
    for reference in references:
        key = (
            reference.source_type.strip().lower(),
            reference.url.strip().lower(),
        )
        if key in seen:
            continue
        seen.add(key)
        deduped.append(reference)
        if len(deduped) >= limit:
            break
    return deduped


def _collect_document_source_references(
    chunks: list[DocumentChunkQueryResult],
    limit: int = 8,
) -> list[ChatSourceReference]:
    references: list[ChatSourceReference] = []
    seen_documents: set[UUID] = set()
    for chunk in chunks:
        if chunk.document_id in seen_documents:
            continue
        seen_documents.add(chunk.document_id)
        references.append(
            ChatSourceReference(
                source_type="document_chunk",
                engram_id=chunk.document_id,
                engram_title=chunk.title,
                url=f"document://{chunk.document_id}",
                title=chunk.source_name or f"{chunk.title} · chunk {chunk.chunk_index}",
                snippet=chunk.snippet,
                captured_at=chunk.created_at,
                document_id=chunk.document_id,
                chunk_id=chunk.chunk_id,
                chunk_index=chunk.chunk_index,
            )
        )
        if len(references) >= limit:
            break
    return references


def _document_chunk_section(chunk: DocumentChunkQueryResult) -> str:
    title = chunk.source_name or chunk.title
    return (
        f"## Document Chunk: {title} (chunk {chunk.chunk_index})\n"
        f"Document ID: {chunk.document_id}\n"
        f"Snippet: {chunk.snippet}\n"
        f"Reference: document://{chunk.document_id}#chunk={chunk.chunk_index}"
    )


def _select_context_engram_ids(
    *,
    request: ChatContextRequest,
    pinned: list[EngramSummary],
    retrieved: list[EngramQueryResult],
) -> list[UUID]:
    candidate_ids = [item.engram_id for item in pinned] + [item.engram_id for item in retrieved]
    return _dedupe_preserve_order(candidate_ids)[: request.max_engrams]


def _collect_rehydration_bundles(
    *,
    engram_ids: list[UUID],
    actor_user_id: UUID,
) -> list[RehydrationBundle]:
    bundles: list[RehydrationBundle] = []
    for engram_id in engram_ids:
        bundle = get_rehydration_bundle(engram_id, actor_user_id=actor_user_id)
        if bundle is not None:
            bundles.append(bundle)
    return bundles


def _split_document_chunks(
    *,
    selected_chunks: list[DocumentChunkQueryResult],
    pinned_chunks: list[DocumentChunkQueryResult],
) -> tuple[list[DocumentChunkQueryResult], list[DocumentChunkQueryResult]]:
    pinned_chunk_ids = {item.chunk_id for item in pinned_chunks}
    pinned_context_chunks = [item for item in selected_chunks if item.chunk_id in pinned_chunk_ids]
    retrieved_context_chunks = [
        item for item in selected_chunks if item.chunk_id not in pinned_chunk_ids
    ]
    return pinned_context_chunks, retrieved_context_chunks


def _build_context_sections(
    *,
    bundles: list[RehydrationBundle],
    selected_chunks: list[DocumentChunkQueryResult],
    pinned_chunks: list[DocumentChunkQueryResult],
) -> list[str]:
    context_sections: list[str] = []
    if bundles:
        context_sections.extend(
            ["# Engram Retrieval Context", *[_bundle_section(item) for item in bundles]]
        )

    pinned_context_chunks, retrieved_context_chunks = _split_document_chunks(
        selected_chunks=selected_chunks,
        pinned_chunks=pinned_chunks,
    )
    if pinned_context_chunks:
        context_sections.extend(
            [
                "# Pinned Document Context",
                *[_document_chunk_section(item) for item in pinned_context_chunks],
            ]
        )
    if retrieved_context_chunks:
        context_sections.extend(
            [
                "# Document Retrieval Context",
                *[_document_chunk_section(item) for item in retrieved_context_chunks],
            ]
        )
    return context_sections


def assemble_chat_context(
    *,
    request: ChatContextRequest,
) -> AssembledChatContext:
    pinned = list_pinned_engram_summaries(
        request.session.session_id,
        actor_user_id=request.actor_user_id,
    )
    pinned_documents = list_pinned_documents(
        request.session.session_id,
        actor_user_id=request.actor_user_id,
    )
    retrieved = query_engrams(
        EngramQueryRequest(
            query=request.user_query,
            project_id=request.session.project_id,
            top_k=request.retrieval_top_k,
        ),
        embedding_dim=request.embedding_dim,
        actor_user_id=request.actor_user_id,
    )

    selected_ids = _select_context_engram_ids(request=request, pinned=pinned, retrieved=retrieved)
    bundles = _collect_rehydration_bundles(
        engram_ids=selected_ids,
        actor_user_id=request.actor_user_id,
    )
    used_engram_ids = [item.engram_id for item in bundles]

    pinned_document_ids = [item.document_id for item in pinned_documents]
    pinned_chunks = _collect_pinned_document_chunks(
        request=request,
        pinned_document_ids=pinned_document_ids,
    )

    retrieved_chunks = query_document_chunks(
        actor_user_id=request.actor_user_id,
        request=DocumentChunkQueryRequest(
            query=request.user_query,
            project_id=request.session.project_id,
            top_k=request.document_top_k,
        ),
        embedding_dim=request.embedding_dim,
    )

    chunk_budget = min(max(request.document_top_k, len(pinned_chunks)), 12)
    selected_chunks = _dedupe_chunks_preserve_order(pinned_chunks + retrieved_chunks)[:chunk_budget]
    used_document_chunk_ids = [item.chunk_id for item in selected_chunks]

    if not bundles and not selected_chunks:
        return AssembledChatContext(
            context_markdown="",
            used_engram_ids=used_engram_ids,
            used_document_chunk_ids=used_document_chunk_ids,
            source_references=[],
        )

    context_sections = _build_context_sections(
        bundles=bundles,
        selected_chunks=selected_chunks,
        pinned_chunks=pinned_chunks,
    )

    source_references = _dedupe_source_references(
        _collect_source_references(bundles) + _collect_document_source_references(selected_chunks)
    )
    return AssembledChatContext(
        context_markdown="\n\n".join(context_sections),
        used_engram_ids=used_engram_ids,
        used_document_chunk_ids=used_document_chunk_ids,
        source_references=source_references,
    )
