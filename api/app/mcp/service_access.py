from __future__ import annotations

from collections.abc import Callable
from typing import Any
from uuid import UUID

from app.memory_admin import MemoryAdminEngramListRequest, MemoryAdminListRequest
from app.models import AdminEngramRecord, EngramCollectionRecord

from .errors import McpRpcError


class McpServiceAccessMixin:
    @staticmethod
    def _parse_uuid(params: dict[str, Any], key: str) -> UUID:
        raw = params.get(key)
        if raw is None:
            raise McpRpcError(
                code=-32602,
                message="Invalid params",
                data={"missing": key},
            )
        try:
            return UUID(str(raw))
        except ValueError as exc:
            raise McpRpcError(
                code=-32602,
                message=f"Invalid {key}: expected UUID string",
                data={"invalid": key, "expected": "uuid", "received": str(raw)},
            ) from exc

    @staticmethod
    def _parse_uuid_list(params: dict[str, Any], key: str) -> list[UUID]:
        raw = params.get(key, [])
        if raw is None:
            return []
        if not isinstance(raw, list):
            raise McpRpcError(
                code=-32602,
                message=f"Invalid {key}: expected array of UUID strings",
                data={"invalid": key, "expected": "uuid[]", "received_type": type(raw).__name__},
            )
        parsed: list[UUID] = []
        for index, item in enumerate(raw):
            try:
                parsed.append(UUID(str(item)))
            except ValueError as exc:
                raise McpRpcError(
                    code=-32602,
                    message=f"Invalid {key}[{index}]: expected UUID string",
                    data={"invalid": key, "index": index, "expected": "uuid", "received": str(item)},
                ) from exc
        return parsed

    @staticmethod
    def _is_admin(actor: dict[str, Any]) -> bool:
        return str(actor.get("role", "")).lower() == "admin"

    @staticmethod
    def _tool_name_and_params_for_tools_call(
        params: dict[str, Any],
    ) -> tuple[str, dict[str, Any]]:
        tool_name = params.get("name")
        if not isinstance(tool_name, str) or not tool_name:
            raise McpRpcError(code=-32602, message="Invalid params", data={"missing": "name"})

        tool_params = params.get("arguments", {})
        if tool_params is None:
            tool_params = {}
        if not isinstance(tool_params, dict):
            raise McpRpcError(code=-32602, message="Invalid params", data={"invalid": "arguments"})
        return tool_name, tool_params

    def _require_owner_or_admin(
        self,
        *,
        actor: dict[str, Any],
        owner_user_id: UUID | None,
        resource: str,
        resource_id: str,
    ) -> None:
        if self._is_admin(actor):
            return
        actor_user_id = UUID(str(actor["user_id"]))
        if owner_user_id and owner_user_id == actor_user_id:
            return
        raise McpRpcError(
            code=-32003,
            message="Resource ownership policy denied this action",
            data={"resource": resource, "resource_id": resource_id},
        )

    def _require_existing_owned_resource(
        self,
        *,
        actor: dict[str, Any],
        resource: str,
        resource_id: UUID,
        record: Any | None,
    ) -> Any:
        if not record:
            raise McpRpcError(
                code=-32004,
                message=f"{resource.capitalize()} not found",
                data={f"{resource}_id": str(resource_id)},
            )
        self._require_owner_or_admin(
            actor=actor,
            owner_user_id=record.owner_user_id,
            resource=resource,
            resource_id=str(resource_id),
        )
        return record

    def _require_owned_resource_by_lookup(
        self,
        *,
        actor: dict[str, Any],
        resource: str,
        resource_id: UUID,
        lookup: Callable[[UUID], Any | None],
    ) -> Any:
        return self._require_existing_owned_resource(
            actor=actor,
            resource=resource,
            resource_id=resource_id,
            record=lookup(resource_id),
        )

    def _lookup_session(self, session_id: UUID) -> Any | None:
        return self._memory_admin_service.get_session(
            session_id=session_id,
            include_deleted=True,
        )

    def _lookup_collection(self, collection_id: UUID) -> Any | None:
        return self._memory_admin_service.find_collection(
            collection_id=collection_id,
            include_deleted=True,
        )

    def _require_session_access(self, *, actor: dict[str, Any], session_id: UUID) -> Any:
        return self._require_owned_resource_by_lookup(
            actor=actor,
            resource="session",
            resource_id=session_id,
            lookup=self._lookup_session,
        )

    def _require_engram_access(
        self,
        *,
        actor: dict[str, Any],
        engram_id: UUID,
        include_deleted: bool = True,
    ) -> Any:
        engram = self._memory_admin_service.find_engram(
            engram_id=engram_id,
            include_deleted=include_deleted,
        )
        return self._require_existing_owned_resource(
            actor=actor,
            resource="engram",
            resource_id=engram_id,
            record=engram,
        )

    def _require_collection_access(self, *, actor: dict[str, Any], collection_id: UUID) -> Any:
        return self._require_owned_resource_by_lookup(
            actor=actor,
            resource="collection",
            resource_id=collection_id,
            lookup=self._lookup_collection,
        )

    def _admin_list_request_kwargs(
        self,
        *,
        actor: dict[str, Any],
        actor_user_id: UUID,
        params: dict[str, Any],
    ) -> dict[str, Any]:
        return {
            "project_id": params.get("project_id"),
            "owner_user_id": None if self._is_admin(actor) else actor_user_id,
            "include_deleted": bool(params.get("include_deleted", False)),
            "limit": int(params.get("limit", 200)),
            "offset": int(params.get("offset", 0)),
        }

    def _list_engrams_for_actor(
        self,
        *,
        actor: dict[str, Any],
        actor_user_id: UUID,
        session_id: UUID | None,
        params: dict[str, Any],
    ) -> list[AdminEngramRecord]:
        request_kwargs = self._admin_list_request_kwargs(
            actor=actor,
            actor_user_id=actor_user_id,
            params=params,
        )
        return self._memory_admin_service.list_engrams(
            request=MemoryAdminEngramListRequest(
                **request_kwargs,
                session_id=session_id,
                query_text=params.get("q"),
            )
        )

    def _list_collections_for_actor(
        self,
        *,
        actor: dict[str, Any],
        actor_user_id: UUID,
        params: dict[str, Any],
    ) -> list[EngramCollectionRecord]:
        request_kwargs = self._admin_list_request_kwargs(
            actor=actor,
            actor_user_id=actor_user_id,
            params=params,
        )
        return self._memory_admin_service.list_collections(
            request=MemoryAdminListRequest(**request_kwargs)
        )
