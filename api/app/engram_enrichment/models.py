from __future__ import annotations

from pydantic import BaseModel, Field

from app.models import MemoryEngramCreate


class EngramEnrichmentReport(BaseModel):
    """Trace payload persisted per engram for enrichment observability."""

    schema_version: str = "1.0"
    origin: str = "unknown"
    enrichment_applied: bool = False
    abstract_derived: bool = False
    tags_derived: bool = False
    keywords_derived: bool = False
    auto_tags: list[str] = Field(default_factory=list)
    auto_keywords: list[str] = Field(default_factory=list)
    abstract_source: str | None = None


class EngramEnrichmentResult(BaseModel):
    payload: MemoryEngramCreate
    report: EngramEnrichmentReport
