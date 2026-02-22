from __future__ import annotations

from datetime import datetime
from enum import Enum
from uuid import UUID

from pydantic import BaseModel, Field

from app.models import AdminEngramRecord, EngramCollectionRecord, ProjectRecord


class ProjectExportFormat(str, Enum):
    json = "json"
    zip = "zip"


class ProjectExportCollectionItemsRecord(BaseModel):
    collection_id: UUID
    engram_ids: list[UUID] = Field(default_factory=list)


class ProjectExportBundle(BaseModel):
    schema_version: str = "1.0"
    exported_at: datetime
    project: ProjectRecord
    collections: list[EngramCollectionRecord] = Field(default_factory=list)
    collection_items: list[ProjectExportCollectionItemsRecord] = Field(default_factory=list)
    engrams: list[AdminEngramRecord] = Field(default_factory=list)
    selected_collection_ids: list[UUID] = Field(default_factory=list)
    include_embeddings: bool = False


class ProjectImportConflictPolicy(str, Enum):
    skip = "skip"
    overwrite = "overwrite"
    rename = "rename"


class ProjectImportResponse(BaseModel):
    target_project_id: str
    imported_engrams: int
    skipped_engrams: int
    overwritten_engrams: int
    imported_collections: int
    reused_collections: int
    imported_collection_items: int
    conflict_policy: ProjectImportConflictPolicy
