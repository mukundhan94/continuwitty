from __future__ import annotations

from collections.abc import Callable
from dataclasses import dataclass
from typing import TypeVar
from uuid import UUID

from fastapi import HTTPException

from app.models import (
    AdminChatSessionRecord,
    AdminEngramDeleteRequest,
    AdminEngramDeleteResponse,
    AdminEngramMoveRequest,
    AdminEngramRecord,
    AdminEngramRestoreResponse,
    AdminEngramUpdateRequest,
    AdminSessionDeleteRequest,
    AdminSessionDeleteResponse,
    AdminSessionRestoreResponse,
    EngramCollectionCreateRequest,
    EngramCollectionDeleteRequest,
    EngramCollectionItemsUpdateRequest,
    EngramCollectionRecord,
    EngramCollectionUpdateRequest,
)
from app.projects import ProjectService

from .repository import (
    AdminEngramUpdateRepositoryRequest,
    add_collection_items,
    create_collection,
    get_admin_engram,
    get_admin_session,
    get_collection,
    list_admin_engrams,
    list_admin_sessions,
    list_collections,
    move_admin_engram_project,
    remove_collection_item,
    restore_engram,
    restore_session,
    soft_delete_collection,
    soft_delete_engram,
    soft_delete_linked_engrams,
    soft_delete_session,
    update_admin_engram,
    update_collection,
)

TRecord = TypeVar("TRecord")


@dataclass(frozen=True)
class MemoryAdminListRequest:
    project_id: str | None
    owner_user_id: UUID | None
    include_deleted: bool
    limit: int
    offset: int


@dataclass(frozen=True)
class MemoryAdminEngramListRequest(MemoryAdminListRequest):
    session_id: UUID | None = None
    query_text: str | None = None


class MemoryAdminService:
    def __init__(self, *, embedding_dim: int, project_service: ProjectService) -> None:
        self._embedding_dim = embedding_dim
        self._project_service = project_service

    @staticmethod
    def _list_project_scoped_records(
        *,
        request: MemoryAdminListRequest,
        fetcher: Callable[..., list[TRecord]],
    ) -> list[TRecord]:
        return fetcher(
            project_id=request.project_id,
            owner_user_id=request.owner_user_id,
            include_deleted=request.include_deleted,
            limit=request.limit,
            offset=request.offset,
        )

    @staticmethod
    def _raise_if_stale_update(
        *,
        expected_updated_at,
        current_updated_at,
        detail: str,
    ) -> None:
        if expected_updated_at and expected_updated_at != current_updated_at:
            raise HTTPException(status_code=409, detail=detail)

    def list_sessions(
        self,
        *,
        request: MemoryAdminListRequest,
    ) -> list[AdminChatSessionRecord]:
        return self._list_project_scoped_records(
            request=request,
            fetcher=list_admin_sessions,
        )

    def get_session(
        self,
        *,
        session_id: UUID,
        include_deleted: bool = True,
    ) -> AdminChatSessionRecord | None:
        return get_admin_session(session_id=session_id, include_deleted=include_deleted)

    def delete_session(
        self,
        *,
        session_id: UUID,
        actor_user_id: UUID,
        payload: AdminSessionDeleteRequest,
    ) -> AdminSessionDeleteResponse:
        deleted = soft_delete_session(
            session_id=session_id,
            deleted_by_user_id=actor_user_id,
            reason=payload.reason,
        )
        if not deleted:
            raise HTTPException(status_code=404, detail="Session not found")
        linked_deleted = 0
        if payload.delete_linked_engrams:
            linked_deleted = soft_delete_linked_engrams(
                session_id=session_id,
                deleted_by_user_id=actor_user_id,
                reason=payload.reason,
            )
        return AdminSessionDeleteResponse(
            session_id=session_id,
            deleted=True,
            linked_engrams_deleted=linked_deleted,
        )

    def restore_session(self, *, session_id: UUID) -> AdminSessionRestoreResponse:
        restored = restore_session(session_id=session_id)
        if not restored:
            raise HTTPException(status_code=404, detail="Session not found")
        return AdminSessionRestoreResponse(session_id=session_id, restored=True)

    def list_engrams(
        self,
        *,
        request: MemoryAdminEngramListRequest,
    ) -> list[AdminEngramRecord]:
        return list_admin_engrams(
            project_id=request.project_id,
            session_id=request.session_id,
            owner_user_id=request.owner_user_id,
            query_text=request.query_text,
            include_deleted=request.include_deleted,
            limit=request.limit,
            offset=request.offset,
        )

    def find_engram(
        self, *, engram_id: UUID, include_deleted: bool = True
    ) -> AdminEngramRecord | None:
        return get_admin_engram(engram_id=engram_id, include_deleted=include_deleted)

    def get_engram(self, *, engram_id: UUID, include_deleted: bool = True) -> AdminEngramRecord:
        engram = self.find_engram(engram_id=engram_id, include_deleted=include_deleted)
        if not engram:
            raise HTTPException(status_code=404, detail="Engram not found")
        return engram

    def update_engram(
        self,
        *,
        engram_id: UUID,
        actor_user_id: UUID,
        payload: AdminEngramUpdateRequest,
    ) -> AdminEngramRecord:
        current = self.get_engram(engram_id=engram_id, include_deleted=False)
        self._raise_if_stale_update(
            expected_updated_at=payload.expected_updated_at,
            current_updated_at=current.updated_at,
            detail="Engram was updated by another operation",
        )

        updated = update_admin_engram(
            engram_id=engram_id,
            request=AdminEngramUpdateRepositoryRequest(
                actor_user_id=actor_user_id,
                title=payload.title,
                abstract=payload.abstract,
                detailed_summary_markdown=payload.detailed_summary_markdown,
                tags=payload.tags,
                keywords=payload.keywords,
                visibility_scope=payload.visibility_scope,
                sources=payload.sources,
                embedding_dim=self._embedding_dim,
            ),
        )
        if not updated:
            raise HTTPException(status_code=404, detail="Engram not found")
        return updated

    def move_engram(
        self,
        *,
        engram_id: UUID,
        actor_user_id: UUID,
        actor_role: str,
        payload: AdminEngramMoveRequest,
    ) -> AdminEngramRecord:
        current = self.get_engram(engram_id=engram_id, include_deleted=False)
        self._raise_if_stale_update(
            expected_updated_at=payload.expected_updated_at,
            current_updated_at=current.updated_at,
            detail="Engram was updated by another operation",
        )

        resolution = self._project_service.resolve_project_id_for_write(
            actor_user_id=actor_user_id,
            actor_role=actor_role,
            project_id=payload.target_project_id,
        )
        moved = move_admin_engram_project(
            engram_id=engram_id,
            target_project_id=resolution.project_id,
            actor_user_id=actor_user_id,
        )
        if not moved:
            raise HTTPException(status_code=404, detail="Engram not found")
        return moved

    def delete_engram(
        self,
        *,
        engram_id: UUID,
        actor_user_id: UUID,
        payload: AdminEngramDeleteRequest,
    ) -> AdminEngramDeleteResponse:
        deleted = soft_delete_engram(
            engram_id=engram_id,
            deleted_by_user_id=actor_user_id,
            reason=payload.reason,
        )
        if not deleted:
            raise HTTPException(status_code=404, detail="Engram not found")
        return AdminEngramDeleteResponse(engram_id=engram_id, deleted=True)

    def restore_engram(self, *, engram_id: UUID) -> AdminEngramRestoreResponse:
        restored = restore_engram(engram_id=engram_id)
        if not restored:
            raise HTTPException(status_code=404, detail="Engram not found")
        return AdminEngramRestoreResponse(engram_id=engram_id, restored=True)

    def list_collections(
        self,
        *,
        request: MemoryAdminListRequest,
    ) -> list[EngramCollectionRecord]:
        return self._list_project_scoped_records(
            request=request,
            fetcher=list_collections,
        )

    def find_collection(
        self,
        *,
        collection_id: UUID,
        include_deleted: bool = True,
    ) -> EngramCollectionRecord | None:
        return get_collection(collection_id=collection_id, include_deleted=include_deleted)

    def create_collection(
        self,
        *,
        actor_user_id: UUID,
        actor_role: str,
        payload: EngramCollectionCreateRequest,
    ) -> EngramCollectionRecord:
        resolution = self._project_service.resolve_project_id_for_write(
            actor_user_id=actor_user_id,
            actor_role=actor_role,
            project_id=payload.project_id,
        )
        return create_collection(
            project_id=resolution.project_id,
            owner_user_id=actor_user_id,
            name=payload.name.strip(),
            description=payload.description.strip(),
        )

    def update_collection(
        self,
        *,
        collection_id: UUID,
        payload: EngramCollectionUpdateRequest,
    ) -> EngramCollectionRecord:
        current = self.find_collection(collection_id=collection_id, include_deleted=False)
        if current is None:
            raise HTTPException(status_code=404, detail="Collection not found")
        self._raise_if_stale_update(
            expected_updated_at=payload.expected_updated_at,
            current_updated_at=current.updated_at,
            detail="Collection was updated by another operation",
        )
        updated = update_collection(
            collection_id=collection_id,
            name=payload.name.strip() if payload.name else None,
            description=payload.description.strip() if payload.description is not None else None,
        )
        if not updated:
            raise HTTPException(status_code=404, detail="Collection not found")
        return updated

    def delete_collection(
        self,
        *,
        collection_id: UUID,
        actor_user_id: UUID,
        payload: EngramCollectionDeleteRequest,
    ) -> dict[str, bool]:
        deleted = soft_delete_collection(
            collection_id=collection_id,
            deleted_by_user_id=actor_user_id,
            reason=payload.reason,
        )
        if not deleted:
            raise HTTPException(status_code=404, detail="Collection not found")
        return {"deleted": True}

    def add_collection_items(
        self,
        *,
        collection_id: UUID,
        actor_user_id: UUID,
        payload: EngramCollectionItemsUpdateRequest,
    ) -> dict[str, int]:
        added_count = add_collection_items(
            collection_id=collection_id,
            actor_user_id=actor_user_id,
            engram_ids=payload.engram_ids,
        )
        return {"added": added_count}

    def remove_collection_item(self, *, collection_id: UUID, engram_id: UUID) -> dict[str, bool]:
        removed = remove_collection_item(collection_id=collection_id, engram_id=engram_id)
        if not removed:
            raise HTTPException(status_code=404, detail="Collection item not found")
        return {"removed": True}
