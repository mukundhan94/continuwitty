from __future__ import annotations

from dataclasses import dataclass
from uuid import UUID

from app.chat_repository import list_pinned_engram_summaries
from app.models import ChatSessionRecord, ChatSourceReference, EngramQueryRequest, RehydrationBundle
from app.repository import get_rehydration_bundle, query_engrams


@dataclass(frozen=True)
class AssembledChatContext:
    context_markdown: str
    used_engram_ids: list[UUID]
    source_references: list[ChatSourceReference]


def _dedupe_preserve_order(ids: list[UUID]) -> list[UUID]:
    seen: set[UUID] = set()
    ordered: list[UUID] = []
    for item in ids:
        if item in seen:
            continue
        seen.add(item)
        ordered.append(item)
    return ordered


def _bundle_section(bundle: RehydrationBundle) -> str:
    detailed_excerpt = bundle.detailed_summary_markdown.strip()
    if detailed_excerpt and len(detailed_excerpt) > 1200:
        detailed_excerpt = detailed_excerpt[:1197].rstrip() + "..."
    if not detailed_excerpt:
        detailed_excerpt = "- None"
    decisions = (
        "\n".join(f"- {item.decision}: {item.rationale}" for item in bundle.key_decisions[:3])
        or "- None"
    )
    questions = "\n".join(f"- {item}" for item in bundle.open_questions[:3]) or "- None"
    citations = (
        "\n".join(
            f"- {item.title or item.url} ({item.url}): {item.snippet[:140]}"
            for item in bundle.top_citations[:3]
        )
        or "- None"
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
    seen: set[tuple[UUID, str]] = set()
    for bundle in bundles:
        for citation in bundle.top_citations[:per_engram_limit]:
            key = (bundle.engram_id, citation.url.lower().strip())
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


def assemble_chat_context(
    *,
    session: ChatSessionRecord,
    actor_user_id: UUID,
    user_query: str,
    embedding_dim: int,
    max_engrams: int = 6,
    retrieval_top_k: int = 4,
) -> AssembledChatContext:
    pinned = list_pinned_engram_summaries(session.session_id, actor_user_id=actor_user_id)
    retrieved = query_engrams(
        EngramQueryRequest(
            query=user_query,
            project_id=session.project_id,
            top_k=retrieval_top_k,
        ),
        embedding_dim=embedding_dim,
        actor_user_id=actor_user_id,
    )

    candidate_ids = [item.engram_id for item in pinned] + [item.engram_id for item in retrieved]
    selected_ids = _dedupe_preserve_order(candidate_ids)[:max_engrams]

    bundles: list[RehydrationBundle] = []
    for engram_id in selected_ids:
        bundle = get_rehydration_bundle(engram_id, actor_user_id=actor_user_id)
        if bundle is not None:
            bundles.append(bundle)

    used_engram_ids = [item.engram_id for item in bundles]
    if not bundles:
        return AssembledChatContext(
            context_markdown="",
            used_engram_ids=used_engram_ids,
            source_references=[],
        )

    context_sections = ["# Engram Retrieval Context", *[_bundle_section(item) for item in bundles]]
    return AssembledChatContext(
        context_markdown="\n\n".join(context_sections),
        used_engram_ids=used_engram_ids,
        source_references=_collect_source_references(bundles),
    )
