from __future__ import annotations

import re
from collections import Counter

from app.models import MemoryEngramCreate

from .models import EngramEnrichmentReport, EngramEnrichmentResult

_DEFAULT_ABSTRACT = "Conversation snapshot."
_TOKEN_PATTERN = re.compile(r"[a-z0-9][a-z0-9_-]{2,}")
_ASSISTANT_SECTION_PATTERN = re.compile(
    r"^##\s*ASSISTANT[^\n]*\n(?P<body>.*?)(?=^##\s|\Z)",
    re.IGNORECASE | re.MULTILINE | re.DOTALL,
)
_STOPWORDS = {
    "about",
    "after",
    "again",
    "also",
    "been",
    "before",
    "being",
    "between",
    "could",
    "does",
    "doing",
    "done",
    "from",
    "have",
    "into",
    "just",
    "like",
    "make",
    "many",
    "more",
    "most",
    "much",
    "need",
    "only",
    "other",
    "over",
    "same",
    "some",
    "such",
    "than",
    "that",
    "their",
    "them",
    "there",
    "these",
    "they",
    "this",
    "those",
    "through",
    "under",
    "using",
    "very",
    "what",
    "when",
    "where",
    "which",
    "while",
    "with",
    "would",
    "your",
}

# Deterministic keyword->tag mapping for v1. This is intentionally local and
# explainable so metadata generation has zero provider dependency.
_TAG_KEYWORD_MAP: dict[str, set[str]] = {
    "incident": {
        "incident",
        "outage",
        "triage",
        "mitigation",
        "latency",
        "sev",
        "rollback",
        "blast",
        "impact",
    },
    "support": {"support", "customer", "ticket", "handoff", "escalation"},
    "product": {"product", "feature", "roadmap", "market", "pricing", "growth"},
    "research": {"research", "analysis", "experiment", "hypothesis", "benchmark"},
    "security": {"security", "vulnerability", "threat", "auth", "oidc", "compliance"},
    "mcp": {"mcp", "jsonrpc", "sse", "tool", "tools", "interop"},
    "chat": {"chat", "session", "conversation", "assistant", "prompt"},
    "ops": {"ops", "runbook", "oncall", "deploy", "deployment", "monitoring", "infra"},
    "docs": {"docs", "documentation", "readme", "playbook", "guide"},
    "release": {"release", "version", "hotfix", "changelog", "rollback"},
}


def _normalize_spaces(value: str) -> str:
    return " ".join(value.strip().split())


def _truncate_text(value: str, max_chars: int) -> str:
    if len(value) <= max_chars:
        return value
    return value[: max_chars - 3].rstrip() + "..."


def _strip_markdown_noise(value: str) -> str:
    text = re.sub(r"[`*_>#-]+", " ", value)
    return _normalize_spaces(text)


def _first_signal_paragraph(markdown: str) -> str:
    paragraphs = re.split(r"\n\s*\n", markdown.strip())
    for paragraph in paragraphs:
        cleaned = _strip_markdown_noise(paragraph)
        if len(cleaned) >= 36 and len(cleaned.split()) >= 7:
            return cleaned
    return ""


def _non_empty_trimmed(values: list[str]) -> list[str]:
    return [item.strip() for item in values if item.strip()]


def derive_abstract_from_markdown_or_retrieval_text(
    conversation_markdown: str,
    retrieval_text: str | None = None,
    max_chars: int = 320,
) -> tuple[str, str]:
    """Derive a compact abstract from the highest-signal available text source."""

    markdown = (conversation_markdown or "").strip()
    if markdown:
        assistant_match = _ASSISTANT_SECTION_PATTERN.search(markdown)
        if assistant_match:
            candidate = _strip_markdown_noise(assistant_match.group("body"))
            if candidate:
                return _truncate_text(candidate, max_chars=max_chars), "assistant_section"

        paragraph = _first_signal_paragraph(markdown)
        if paragraph:
            return _truncate_text(paragraph, max_chars=max_chars), "first_signal_paragraph"

    retrieval_candidate = _normalize_spaces(retrieval_text or "")
    if retrieval_candidate:
        return _truncate_text(retrieval_candidate, max_chars=max_chars), "retrieval_text"

    return _DEFAULT_ABSTRACT, "fallback"


def extract_keywords(text: str, max_keywords: int = 12) -> list[str]:
    """Extract deterministic keywords sorted by freq then first appearance."""

    lowered = (text or "").lower()
    if not lowered.strip():
        return []

    ordered_tokens: list[str] = []
    seen_index: dict[str, int] = {}
    for match in _TOKEN_PATTERN.finditer(lowered):
        token = match.group(0)
        if token in _STOPWORDS:
            continue
        if token.isdigit():
            continue
        if not any(ch.isalpha() for ch in token):
            continue
        if token not in seen_index:
            seen_index[token] = len(ordered_tokens)
            ordered_tokens.append(token)

    if not ordered_tokens:
        return []

    counts = Counter()
    for match in _TOKEN_PATTERN.finditer(lowered):
        token = match.group(0)
        if token in seen_index:
            counts[token] += 1

    ranked = sorted(
        ordered_tokens,
        key=lambda token: (-counts[token], seen_index[token], token),
    )
    return ranked[:max(max_keywords, 1)]


def map_keywords_to_tags(keywords: list[str], max_tags: int = 8) -> list[str]:
    normalized = {item.strip().lower() for item in keywords if item.strip()}
    selected: list[str] = []

    for tag, mapping in _TAG_KEYWORD_MAP.items():
        if normalized & mapping:
            selected.append(tag)

    if not selected:
        selected = ["conversation", "snapshot"]

    return selected[:max(max_tags, 1)]


def enrich_if_missing(payload: MemoryEngramCreate, origin: str) -> EngramEnrichmentResult:
    """Fill empty abstract/tags/keywords without mutating non-empty caller values.

    This function is intentionally deterministic for v1. Future phases can add an
    optional LLM/hybrid branch while preserving the same non-destructive contract.
    """

    resolved_abstract = payload.abstract
    resolved_tags = payload.tags
    resolved_keywords = payload.keywords

    report = EngramEnrichmentReport(origin=origin)

    abstract_missing = not payload.abstract.strip()
    keywords_missing = len(_non_empty_trimmed(payload.keywords)) == 0
    tags_missing = len(_non_empty_trimmed(payload.tags)) == 0

    source_text = "\n".join(
        [
            payload.title,
            payload.abstract,
            payload.detailed_summary_markdown,
            payload.retrieval_text or "",
        ]
    )

    if abstract_missing:
        derived_abstract, abstract_source = derive_abstract_from_markdown_or_retrieval_text(
            payload.detailed_summary_markdown,
            retrieval_text=payload.retrieval_text,
        )
        resolved_abstract = derived_abstract
        report.abstract_derived = True
        report.abstract_source = abstract_source

    if keywords_missing:
        resolved_keywords = extract_keywords(source_text, max_keywords=12)
        report.keywords_derived = True
        report.auto_keywords = resolved_keywords

    if tags_missing:
        tag_seed = resolved_keywords if report.keywords_derived else _non_empty_trimmed(payload.keywords)
        resolved_tags = map_keywords_to_tags(tag_seed, max_tags=8)
        report.tags_derived = True
        report.auto_tags = resolved_tags

    report.enrichment_applied = report.abstract_derived or report.tags_derived or report.keywords_derived

    resolved_payload = payload.model_copy(
        update={
            "abstract": resolved_abstract,
            "tags": resolved_tags,
            "keywords": resolved_keywords,
        }
    )

    return EngramEnrichmentResult(payload=resolved_payload, report=report)
