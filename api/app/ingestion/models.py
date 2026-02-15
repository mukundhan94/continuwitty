from __future__ import annotations

from datetime import datetime
from enum import Enum
from uuid import UUID

from pydantic import BaseModel, Field

from app.models import EngramQueryResult, VisibilityScope


class DocumentSourceType(str, Enum):
    text = "text"
    file = "file"


class DocumentIngestTextRequest(BaseModel):
    project_id: str = Field(min_length=1, max_length=128)
    title: str = Field(min_length=1, max_length=256)
    text: str = Field(min_length=1)
    visibility_scope: VisibilityScope = VisibilityScope.private
    chunk_size_chars: int = Field(default=1000, ge=300, le=4000)
    chunk_overlap_chars: int = Field(default=180, ge=0, le=1600)
    metadata: dict = Field(default_factory=dict)


class DocumentIngestFileRequest(BaseModel):
    project_id: str = Field(min_length=1, max_length=128)
    title: str | None = Field(default=None, max_length=256)
    visibility_scope: VisibilityScope = VisibilityScope.private
    chunk_size_chars: int = Field(default=1000, ge=300, le=4000)
    chunk_overlap_chars: int = Field(default=180, ge=0, le=1600)
    metadata: dict = Field(default_factory=dict)


class DocumentRecord(BaseModel):
    document_id: UUID
    owner_user_id: UUID
    project_id: str
    title: str
    source_type: DocumentSourceType
    source_name: str | None = None
    mime_type: str | None = None
    visibility_scope: VisibilityScope
    content_hash: str
    chunk_count: int
    created_at: datetime
    updated_at: datetime


class DocumentIngestResponse(BaseModel):
    document: DocumentRecord


class DocumentChunkQueryRequest(BaseModel):
    query: str = Field(min_length=1)
    project_id: str | None = None
    top_k: int = Field(default=6, ge=1, le=50)


class DocumentChunkQueryResult(BaseModel):
    chunk_id: UUID
    document_id: UUID
    project_id: str
    title: str
    source_name: str | None = None
    chunk_index: int
    snippet: str
    created_at: datetime
    visibility_scope: VisibilityScope
    distance: float


class BlendedRetrievalQueryRequest(BaseModel):
    query: str = Field(min_length=1)
    project_id: str | None = None
    top_k_engrams: int = Field(default=4, ge=1, le=25)
    top_k_document_chunks: int = Field(default=6, ge=1, le=50)


class BlendedRetrievalQueryResponse(BaseModel):
    engrams: list[EngramQueryResult] = Field(default_factory=list)
    document_chunks: list[DocumentChunkQueryResult] = Field(default_factory=list)
