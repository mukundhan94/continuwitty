"""Deterministic engram metadata enrichment helpers.

This module provides a fill-empty-only enrichment workflow so callers can omit
metadata fields (for example in MCP conversation persistence) while preserving
explicit caller-provided values unchanged.
"""

from .models import EngramEnrichmentReport, EngramEnrichmentResult
from .service import (
    derive_abstract_from_markdown_or_retrieval_text,
    enrich_if_missing,
    extract_keywords,
    map_keywords_to_tags,
)

__all__ = [
    "EngramEnrichmentReport",
    "EngramEnrichmentResult",
    "derive_abstract_from_markdown_or_retrieval_text",
    "enrich_if_missing",
    "extract_keywords",
    "map_keywords_to_tags",
]
