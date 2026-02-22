from __future__ import annotations

import io
import zipfile
from collections.abc import Callable
from uuid import UUID

from fastapi import APIRouter, File, Query, Request, UploadFile
from fastapi.responses import Response, StreamingResponse

from .models import ProjectExportFormat, ProjectImportConflictPolicy, ProjectImportResponse

from .service import ExportProjectRequest, ExportService, ImportProjectRequest

# FastAPI dependency objects kept at module scope to satisfy lint rule B008.
EXPORT_FORMAT_QUERY = Query(default=ProjectExportFormat.json)
EXPORT_COLLECTION_IDS_QUERY = Query(default=[])
EXPORT_INCLUDE_EMBEDDINGS_QUERY = Query(default=False)
IMPORT_CONFLICT_POLICY_QUERY = Query(default=ProjectImportConflictPolicy.skip)
IMPORT_FILE_DEPENDENCY = File(...)


def _build_export_filename(*, project_id: str, export_format: ProjectExportFormat) -> str:
    suffix = "zip" if export_format == ProjectExportFormat.zip else "json"
    safe_project = project_id.replace("/", "-").replace(" ", "-")
    return f"engram-export-{safe_project}.{suffix}"


def _build_zip_payload(*, json_payload: bytes) -> bytes:
    buffer = io.BytesIO()
    with zipfile.ZipFile(buffer, mode="w", compression=zipfile.ZIP_DEFLATED) as archive:
        archive.writestr("export.json", json_payload)
    return buffer.getvalue()


def create_export_router(
    *,
    export_service: ExportService,
    require_api_actor: Callable[[Request], dict[str, str]],
) -> APIRouter:
    router = APIRouter(prefix="/api/v1/projects", tags=["export"])

    @router.get("/{project_id}/export")
    def export_project_bundle(
        request: Request,
        project_id: str,
        format: ProjectExportFormat = EXPORT_FORMAT_QUERY,  # noqa: A002
        collection_ids: list[UUID] = EXPORT_COLLECTION_IDS_QUERY,
        include_embeddings: bool = EXPORT_INCLUDE_EMBEDDINGS_QUERY,
    ) -> Response:
        actor = require_api_actor(request)
        bundle = export_service.build_project_export_bundle(
            request=ExportProjectRequest(
                actor_user_id=UUID(actor["user_id"]),
                actor_role=actor["role"],
                project_id=project_id,
                collection_ids=collection_ids,
                include_embeddings=include_embeddings,
            )
        )

        payload = bundle.model_dump_json(indent=2).encode("utf-8")
        filename = _build_export_filename(project_id=project_id, export_format=format)
        headers = {"Content-Disposition": f'attachment; filename="{filename}"'}

        if format == ProjectExportFormat.zip:
            zip_payload = _build_zip_payload(json_payload=payload)
            return StreamingResponse(
                io.BytesIO(zip_payload),
                media_type="application/zip",
                headers=headers,
            )

        return Response(content=payload, media_type="application/json", headers=headers)

    @router.post("/{project_id}/import", response_model=ProjectImportResponse)
    async def import_project_bundle(
        request: Request,
        project_id: str,
        conflict_policy: ProjectImportConflictPolicy = IMPORT_CONFLICT_POLICY_QUERY,
        file: UploadFile = IMPORT_FILE_DEPENDENCY,
    ) -> ProjectImportResponse:
        actor = require_api_actor(request)
        file_bytes = await file.read()
        return export_service.import_project_bundle(
            request=ImportProjectRequest(
                actor_user_id=UUID(actor["user_id"]),
                actor_role=actor["role"],
                target_project_id=project_id,
                file_bytes=file_bytes,
                conflict_policy=conflict_policy,
            )
        )

    return router
