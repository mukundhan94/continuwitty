from datetime import datetime
from uuid import UUID

from pydantic import BaseModel, Field


class SupportingSource(BaseModel):
    url: str
    title: str | None = None
    snippet: str
    captured_at: datetime


class Claim(BaseModel):
    claim: str
    supporting_sources: list[SupportingSource] = Field(default_factory=list)


class Decision(BaseModel):
    decision: str
    rationale: str


class ArtifactIn(BaseModel):
    artifact_type: str
    storage_uri: str
    metadata: dict = Field(default_factory=dict)


class MemoryEngramCreate(BaseModel):
    project_id: str
    thread_id: str | None = None
    title: str
    abstract: str
    detailed_summary_markdown: str
    decisions: list[Decision] = Field(default_factory=list)
    assumptions: list[str] = Field(default_factory=list)
    open_questions: list[str] = Field(default_factory=list)
    claims: list[Claim] = Field(default_factory=list)
    tags: list[str] = Field(default_factory=list)
    keywords: list[str] = Field(default_factory=list)
    artifacts: list[ArtifactIn] = Field(default_factory=list)
    retrieval_text: str | None = None


class EngramSummary(BaseModel):
    engram_id: UUID
    project_id: str
    thread_id: str | None
    title: str
    abstract: str
    created_at: datetime
    tags: list[str] = Field(default_factory=list)
    keywords: list[str] = Field(default_factory=list)


class EngramCreateResponse(BaseModel):
    engram_id: UUID
    created_at: datetime


class EngramQueryRequest(BaseModel):
    query: str
    top_k: int = Field(default=5, ge=1, le=50)
    project_id: str | None = None
    tags: list[str] = Field(default_factory=list)
    keywords: list[str] = Field(default_factory=list)
    created_after: datetime | None = None
    created_before: datetime | None = None


class EngramQueryResult(BaseModel):
    engram_id: UUID
    project_id: str
    title: str
    abstract: str
    created_at: datetime
    tags: list[str] = Field(default_factory=list)
    keywords: list[str] = Field(default_factory=list)
    distance: float


class RehydrationCitation(BaseModel):
    url: str
    title: str | None = None
    snippet: str
    captured_at: datetime


class RehydrationBundle(BaseModel):
    engram_id: UUID
    project_id: str
    title: str
    compact_summary: str
    key_decisions: list[Decision] = Field(default_factory=list)
    open_questions: list[str] = Field(default_factory=list)
    top_citations: list[RehydrationCitation] = Field(default_factory=list)
    context_markdown: str
