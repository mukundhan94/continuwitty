from __future__ import annotations

import json
import zipfile
from dataclasses import dataclass
from datetime import UTC, datetime
from io import BytesIO
from uuid import UUID

from fastapi import HTTPException

from app.db import get_conn
from app.memory_admin.service import (
    MemoryAdminEngramListRequest,
    MemoryAdminListRequest,
    MemoryAdminService,
)
from app.models import (
    AdminEngramRecord,
    AdminEngramSourceInput,
    AdminEngramSourceRecord,
    EngramCollectionCreateRequest,
    EngramCollectionItemsUpdateRequest,
    MemoryEngramCreate,
)
from app.projects import ProjectService
from app.repository import create_engram

from .models import (
    ProjectExportBundle,
    ProjectExportCollectionItemsRecord,
    ProjectImportConflictPolicy,
    ProjectImportResponse,
)


@dataclass(frozen=True)
class ExportProjectRequest:
    actor_user_id: UUID
    actor_role: str
    project_id: str
    collection_ids: list[UUID]
    include_embeddings: bool = False


@dataclass(frozen=True)
class ImportProjectRequest:
    actor_user_id: UUID
    actor_role: str
    target_project_id: str
    file_bytes: bytes
    conflict_policy: ProjectImportConflictPolicy


class ExportService:
    def __init__(
        self,
        *,
        project_service: ProjectService,
        memory_admin_service: MemoryAdminService,
        embedding_dim: int,
    ) -> None:
        self._project_service = project_service
        self._memory_admin_service = memory_admin_service
        self._embedding_dim = embedding_dim

    def _resolve_project_or_404(self, *, actor_user_id: UUID, actor_role: str, project_id: str):
        project = self._project_service.get_project(
            actor_user_id=actor_user_id,
            actor_role=actor_role,
            project_id=project_id,
            include_archived=True,
        )
        if not project:
            raise HTTPException(status_code=404, detail="Project not found")
        return project

    @staticmethod
    def _decode_bundle_text(file_bytes: bytes) -> str:
        if not file_bytes:
            raise HTTPException(status_code=422, detail="Import file is empty")

        try:
            with zipfile.ZipFile(BytesIO(file_bytes), mode="r") as archive:
                return archive.read("export.json").decode("utf-8")
        except zipfile.BadZipFile:
            pass
        except KeyError as exc:
            raise HTTPException(status_code=422, detail="ZIP missing export.json") from exc

        try:
            return file_bytes.decode("utf-8")
        except UnicodeDecodeError as exc:
            raise HTTPException(status_code=422, detail="Unsupported import file format") from exc

    @classmethod
    def _parse_bundle(cls, file_bytes: bytes) -> ProjectExportBundle:
        text_payload = cls._decode_bundle_text(file_bytes)

        try:
            raw = json.loads(text_payload)
            return ProjectExportBundle.model_validate(raw)
        except Exception as exc:
            raise HTTPException(status_code=422, detail="Invalid export bundle") from exc

    @staticmethod
    def _fetch_single_uuid(*, sql: str, params: tuple[object, ...], id_key: str) -> UUID | None:
        with get_conn() as conn, conn.cursor() as cur:
            cur.execute(sql, params)
            row = cur.fetchone()
        return row[id_key] if row else None

    @staticmethod
    def _soft_delete_engram(*, engram_id: UUID, actor_user_id: UUID) -> None:
        with get_conn() as conn, conn.cursor() as cur:
            cur.execute(
                """
                UPDATE engrams
                SET deleted_at = now(), deleted_by_user_id = %s, delete_reason = 'import_overwrite'
                WHERE engram_id = %s
                """,
                (actor_user_id, engram_id),
            )

    @staticmethod
    def _build_unique_name(*, table: str, column: str, base_name: str, project_id: str) -> str:
        candidate = f"{base_name} (imported)"
        index = 2
        sql = f"""
            SELECT 1
            FROM {table}
            WHERE project_id = %s
              AND {column} = %s
              AND deleted_at IS NULL
            LIMIT 1
        """
        with get_conn() as conn, conn.cursor() as cur:
            while True:
                cur.execute(sql, (project_id, candidate))
                if not cur.fetchone():
                    return candidate
                candidate = f"{base_name} (imported {index})"
                index += 1

    def _resolve_engram_title(
        self,
        *,
        existing_id: UUID | None,
        request: ImportProjectRequest,
        base_title: str,
    ) -> str | None:
        if not existing_id:
            return base_title
        if request.conflict_policy == ProjectImportConflictPolicy.skip:
            return None
        if request.conflict_policy == ProjectImportConflictPolicy.overwrite:
            self._soft_delete_engram(engram_id=existing_id, actor_user_id=request.actor_user_id)
            return base_title
        return self._build_unique_name(
            table="engrams",
            column="title",
            base_name=base_title,
            project_id=request.target_project_id,
        )

    def _import_engrams(
        self,
        *,
        bundle: ProjectExportBundle,
        request: ImportProjectRequest,
    ) -> tuple[dict[UUID, UUID], int, int, int]:
        imported_engrams = 0
        skipped_engrams = 0
        overwritten_engrams = 0
        engram_id_map: dict[UUID, UUID] = {}

        for exported_engram in bundle.engrams:
            existing_id = self._fetch_single_uuid(
                sql="""
                SELECT engram_id
                FROM engrams
                WHERE project_id = %s
                  AND title = %s
                  AND engram_markdown = %s
                  AND deleted_at IS NULL
                LIMIT 1
                """,
                params=(
                    request.target_project_id,
                    exported_engram.title,
                    exported_engram.detailed_summary_markdown,
                ),
                id_key="engram_id",
            )
            title = self._resolve_engram_title(
                existing_id=existing_id,
                request=request,
                base_title=exported_engram.title,
            )
            if title is None:
                skipped_engrams += 1
                if existing_id is not None:
                    engram_id_map[exported_engram.engram_id] = existing_id
                continue

            if existing_id and request.conflict_policy == ProjectImportConflictPolicy.overwrite:
                overwritten_engrams += 1

            created = create_engram(
                MemoryEngramCreate(
                    project_id=request.target_project_id,
                    thread_id=exported_engram.thread_id,
                    title=title,
                    abstract=exported_engram.abstract,
                    detailed_summary_markdown=exported_engram.detailed_summary_markdown,
                    tags=exported_engram.tags,
                    keywords=exported_engram.keywords,
                    visibility_scope=exported_engram.visibility_scope,
                ),
                embedding_dim=self._embedding_dim,
                owner_user_id=request.actor_user_id,
                enrichment_origin="export.import",
            )
            self._replace_engram_sources(
                engram_id=created.engram_id,
                sources=exported_engram.sources,
            )
            engram_id_map[exported_engram.engram_id] = created.engram_id
            imported_engrams += 1

        return engram_id_map, imported_engrams, skipped_engrams, overwritten_engrams

    def _import_collections(
        self,
        *,
        bundle: ProjectExportBundle,
        request: ImportProjectRequest,
    ) -> tuple[dict[UUID, UUID], int, int]:
        imported_collections = 0
        reused_collections = 0
        collection_id_map: dict[UUID, UUID] = {}

        for exported_collection in bundle.collections:
            existing_collection_id = self._fetch_single_uuid(
                sql="""
                SELECT collection_id
                FROM engram_collections
                WHERE project_id = %s
                  AND name = %s
                  AND deleted_at IS NULL
                LIMIT 1
                """,
                params=(request.target_project_id, exported_collection.name),
                id_key="collection_id",
            )

            if existing_collection_id and request.conflict_policy in {
                ProjectImportConflictPolicy.skip,
                ProjectImportConflictPolicy.overwrite,
            }:
                collection_id_map[exported_collection.collection_id] = existing_collection_id
                reused_collections += 1
                continue

            collection_name = exported_collection.name
            if existing_collection_id and request.conflict_policy == ProjectImportConflictPolicy.rename:
                collection_name = self._build_unique_name(
                    table="engram_collections",
                    column="name",
                    base_name=exported_collection.name,
                    project_id=request.target_project_id,
                )

            created_collection = self._memory_admin_service.create_collection(
                actor_user_id=request.actor_user_id,
                actor_role=request.actor_role,
                payload=EngramCollectionCreateRequest(
                    project_id=request.target_project_id,
                    name=collection_name,
                    description=exported_collection.description,
                ),
            )
            collection_id_map[exported_collection.collection_id] = created_collection.collection_id
            imported_collections += 1

        return collection_id_map, imported_collections, reused_collections

    def _import_collection_items(
        self,
        *,
        bundle: ProjectExportBundle,
        request: ImportProjectRequest,
        engram_id_map: dict[UUID, UUID],
        collection_id_map: dict[UUID, UUID],
    ) -> int:
        imported_collection_items = 0
        for item_record in bundle.collection_items:
            target_collection_id = collection_id_map.get(item_record.collection_id)
            if not target_collection_id:
                continue

            mapped_engram_ids = [
                engram_id_map[source_engram_id]
                for source_engram_id in item_record.engram_ids
                if source_engram_id in engram_id_map
            ]
            if not mapped_engram_ids:
                continue

            result = self._memory_admin_service.add_collection_items(
                collection_id=target_collection_id,
                actor_user_id=request.actor_user_id,
                payload=EngramCollectionItemsUpdateRequest(engram_ids=mapped_engram_ids),
            )
            imported_collection_items += result["added"]

        return imported_collection_items

    def import_project_bundle(self, *, request: ImportProjectRequest) -> ProjectImportResponse:
        self._resolve_project_or_404(
            actor_user_id=request.actor_user_id,
            actor_role=request.actor_role,
            project_id=request.target_project_id,
        )

        bundle = self._parse_bundle(request.file_bytes)

        engram_id_map, imported_engrams, skipped_engrams, overwritten_engrams = (
            self._import_engrams(bundle=bundle, request=request)
        )
        collection_id_map, imported_collections, reused_collections = self._import_collections(
            bundle=bundle,
            request=request,
        )
        imported_collection_items = self._import_collection_items(
            bundle=bundle,
            request=request,
            engram_id_map=engram_id_map,
            collection_id_map=collection_id_map,
        )

        return ProjectImportResponse(
            target_project_id=request.target_project_id,
            imported_engrams=imported_engrams,
            skipped_engrams=skipped_engrams,
            overwritten_engrams=overwritten_engrams,
            imported_collections=imported_collections,
            reused_collections=reused_collections,
            imported_collection_items=imported_collection_items,
            conflict_policy=request.conflict_policy,
        )

    @staticmethod
    def _list_collection_item_map(*, collection_ids: list[UUID]) -> dict[UUID, list[UUID]]:
        if not collection_ids:
            return {}
        with get_conn() as conn, conn.cursor() as cur:
            cur.execute(
                """
                SELECT collection_id, engram_id
                FROM engram_collection_items
                WHERE collection_id = ANY(%s::UUID[])
                ORDER BY collection_id, created_at ASC
                """,
                (collection_ids,),
            )
            rows = cur.fetchall()

        result: dict[UUID, list[UUID]] = {collection_id: [] for collection_id in collection_ids}
        for row in rows:
            result[row["collection_id"]].append(row["engram_id"])
        return result

    @staticmethod
    def _replace_engram_sources(
        *,
        engram_id: UUID,
        sources: list[AdminEngramSourceInput],
    ) -> None:
        with get_conn() as conn, conn.cursor() as cur:
            cur.execute("DELETE FROM sources WHERE engram_id = %s", (engram_id,))
            for source in sources:
                cur.execute(
                    """
                    INSERT INTO sources (
                        source_id,
                        engram_id,
                        captured_at,
                        url,
                        title,
                        snippet,
                        content_text,
                        content_hash
                    )
                    VALUES (gen_random_uuid(), %s, %s, %s, %s, %s, %s, %s)
                    """,
                    (
                        engram_id,
                        source.captured_at,
                        source.url,
                        source.title,
                        source.snippet,
                        source.content_text,
                        source.content_hash,
                    ),
                )

    @staticmethod
    def _list_engram_sources_map(
        *,
        engram_ids: list[UUID],
    ) -> dict[UUID, list[AdminEngramSourceRecord]]:
        if not engram_ids:
            return {}
        with get_conn() as conn, conn.cursor() as cur:
            cur.execute(
                """
                SELECT
                    source_id,
                    engram_id,
                    captured_at,
                    url,
                    title,
                    snippet,
                    content_text,
                    content_hash
                FROM sources
                WHERE engram_id = ANY(%s::UUID[])
                ORDER BY engram_id, captured_at DESC
                """,
                (engram_ids,),
            )
            rows = cur.fetchall()
        grouped: dict[UUID, list[AdminEngramSourceRecord]] = {
            engram_id: [] for engram_id in engram_ids
        }
        for row in rows:
            grouped[row["engram_id"]].append(
                AdminEngramSourceRecord(
                    source_id=row["source_id"],
                    captured_at=row["captured_at"],
                    url=row["url"],
                    title=row["title"],
                    snippet=row["snippet"],
                    content_text=row["content_text"],
                    content_hash=row["content_hash"],
                )
            )
        return grouped

    @classmethod
    def _attach_engram_sources(
        cls,
        *,
        engrams: list[AdminEngramRecord],
    ) -> list[AdminEngramRecord]:
        source_map = cls._list_engram_sources_map(
            engram_ids=[engram.engram_id for engram in engrams]
        )
        return [
            engram.model_copy(update={"sources": source_map.get(engram.engram_id, [])})
            for engram in engrams
        ]

    def build_project_export_bundle(
        self,
        *,
        request: ExportProjectRequest,
    ) -> ProjectExportBundle:
        project = self._resolve_project_or_404(
            actor_user_id=request.actor_user_id,
            actor_role=request.actor_role,
            project_id=request.project_id,
        )

        collections = self._memory_admin_service.list_collections(
            request=MemoryAdminListRequest(
                project_id=project.project_id,
                owner_user_id=None,
                include_deleted=False,
                limit=5_000,
                offset=0,
            )
        )
        collections_by_id = {collection.collection_id: collection for collection in collections}

        selected_collection_ids = request.collection_ids
        if selected_collection_ids:
            missing = [
                collection_id
                for collection_id in selected_collection_ids
                if collection_id not in collections_by_id
            ]
            if missing:
                raise HTTPException(status_code=404, detail="Collection not found for project")
            selected_collections = [
                collections_by_id[collection_id] for collection_id in selected_collection_ids
            ]
        else:
            selected_collections = collections
            selected_collection_ids = [
                collection.collection_id for collection in selected_collections
            ]

        collection_item_map = self._list_collection_item_map(
            collection_ids=[collection.collection_id for collection in selected_collections]
        )

        if request.collection_ids:
            engram_ids: list[UUID] = []
            seen: set[UUID] = set()
            for collection in selected_collections:
                for engram_id in collection_item_map.get(collection.collection_id, []):
                    if engram_id in seen:
                        continue
                    seen.add(engram_id)
                    engram_ids.append(engram_id)
            engrams = [
                self._memory_admin_service.get_engram(engram_id=engram_id, include_deleted=False)
                for engram_id in engram_ids
            ]
        else:
            engrams = self._memory_admin_service.list_engrams(
                request=MemoryAdminEngramListRequest(
                    project_id=project.project_id,
                    owner_user_id=None,
                    include_deleted=False,
                    limit=50_000,
                    offset=0,
                    session_id=None,
                    query_text=None,
                )
            )
            engrams = self._attach_engram_sources(engrams=engrams)

        collection_items = [
            ProjectExportCollectionItemsRecord(
                collection_id=collection.collection_id,
                engram_ids=collection_item_map.get(collection.collection_id, []),
            )
            for collection in selected_collections
        ]

        return ProjectExportBundle(
            exported_at=datetime.now(UTC),
            project=project,
            collections=selected_collections,
            collection_items=collection_items,
            engrams=engrams,
            selected_collection_ids=selected_collection_ids,
            include_embeddings=bool(request.include_embeddings),
        )
