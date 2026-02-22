from __future__ import annotations

import json
from collections.abc import Callable
from dataclasses import dataclass
from typing import Any, Protocol
from uuid import UUID

from app.export import ExportService
from app.export.models import ProjectImportConflictPolicy
from app.export.service import ExportProjectRequest, ImportProjectRequest

from .errors import McpRpcError


class _ProjectTransferDispatchContext(Protocol):
    actor: dict[str, Any]
    actor_user_id: UUID
    method: str
    params: dict[str, Any]


@dataclass(frozen=True)
class ProjectTransferDispatchDependencies:
    export_service: ExportService | None
    parse_uuid_list: Callable[[dict[str, Any], str], list[UUID]]


def _required_project_id(params: dict[str, Any]) -> str:
    project_id = str(params.get("project_id", "")).strip()
    if project_id:
        return project_id
    raise McpRpcError(
        code=-32602,
        message="Invalid params",
        data={"missing": "project_id"},
    )


def _import_bundle_bytes(params: dict[str, Any]) -> bytes:
    bundle = params.get("bundle")
    if bundle is not None:
        if not isinstance(bundle, dict):
            raise McpRpcError(
                code=-32602,
                message="Invalid params",
                data={"invalid": "bundle"},
            )
        return json.dumps(bundle).encode("utf-8")

    bundle_json = params.get("bundle_json")
    if bundle_json is None:
        raise McpRpcError(
            code=-32602,
            message="Invalid params",
            data={"missing": "bundle"},
        )
    if not isinstance(bundle_json, str):
        raise McpRpcError(
            code=-32602,
            message="Invalid params",
            data={"invalid": "bundle_json"},
        )
    return bundle_json.encode("utf-8")


def _conflict_policy(params: dict[str, Any]) -> ProjectImportConflictPolicy:
    raw_policy = params.get("conflict_policy", ProjectImportConflictPolicy.skip.value)
    try:
        return ProjectImportConflictPolicy(str(raw_policy))
    except ValueError as exc:
        raise McpRpcError(
            code=-32602,
            message="Invalid params",
            data={"invalid": "conflict_policy"},
        ) from exc


def dispatch_project_transfer_tool(
    *,
    dependencies: ProjectTransferDispatchDependencies,
    context: _ProjectTransferDispatchContext,
) -> dict[str, Any] | None:
    if context.method not in {"project.export_bundle", "project.import_bundle"}:
        return None

    export_service = dependencies.export_service
    if export_service is None:
        raise McpRpcError(
            code=-32601,
            message="Method not found",
            data={"method": context.method},
        )

    actor_role = str(context.actor.get("role", ""))
    project_id = _required_project_id(context.params)

    if context.method == "project.export_bundle":
        bundle = export_service.build_project_export_bundle(
            request=ExportProjectRequest(
                actor_user_id=context.actor_user_id,
                actor_role=actor_role,
                project_id=project_id,
                collection_ids=dependencies.parse_uuid_list(context.params, "collection_ids"),
                include_embeddings=bool(context.params.get("include_embeddings", False)),
            )
        )
        return {"bundle": bundle.model_dump(mode="json")}

    result = export_service.import_project_bundle(
        request=ImportProjectRequest(
            actor_user_id=context.actor_user_id,
            actor_role=actor_role,
            target_project_id=project_id,
            file_bytes=_import_bundle_bytes(context.params),
            conflict_policy=_conflict_policy(context.params),
        )
    )
    return {"summary": result.model_dump(mode="json")}
