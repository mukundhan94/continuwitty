from __future__ import annotations

from dataclasses import dataclass
from typing import TypedDict
from uuid import UUID

from app.models import AdminEngramSourceInput, VisibilityScope


@dataclass(frozen=True)
class _AdminEngramUpdatePayload:
    title: str | None
    abstract: str | None
    detailed_summary_markdown: str | None
    tags: list[str] | None
    keywords: list[str] | None
    visibility_scope: VisibilityScope | None


class _EngramUpdateFields(TypedDict):
    title: str
    abstract: str
    detailed_summary_markdown: str
    tags: list[str]
    keywords: list[str]
    visibility_scope: str


@dataclass(frozen=True)
class AdminEngramUpdateRepositoryRequest:
    actor_user_id: UUID
    title: str | None
    abstract: str | None
    detailed_summary_markdown: str | None
    tags: list[str] | None
    keywords: list[str] | None
    visibility_scope: VisibilityScope | None
    sources: list[AdminEngramSourceInput] | None
    embedding_dim: int


@dataclass(frozen=True)
class _EngramPersistPayload:
    engram_id: UUID
    actor_user_id: UUID
    update_fields: _EngramUpdateFields
    retrieval_text: str
    embedding_provider_id: str
    embedding_literal: str
    engram_json: dict[str, object]


@dataclass(frozen=True)
class AdminSessionListRepositoryRequest:
    project_id: str | None = None
    owner_user_id: UUID | None = None
    include_deleted: bool = False
    limit: int = 200
    offset: int = 0


@dataclass(frozen=True)
class AdminEngramListRepositoryRequest(AdminSessionListRepositoryRequest):
    session_id: UUID | None = None
    query_text: str | None = None


@dataclass(frozen=True)
class CollectionListRepositoryRequest(AdminSessionListRepositoryRequest):
    pass
