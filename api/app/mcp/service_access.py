from __future__ import annotations

from typing import Any
from uuid import UUID

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
                message="Invalid params",
                data={"invalid": key},
            ) from exc

    @staticmethod
    def _parse_uuid_list(params: dict[str, Any], key: str) -> list[UUID]:
        raw = params.get(key, [])
        if raw is None:
            return []
        if not isinstance(raw, list):
            raise McpRpcError(
                code=-32602,
                message="Invalid params",
                data={"invalid": key},
            )
        parsed: list[UUID] = []
        for index, item in enumerate(raw):
            try:
                parsed.append(UUID(str(item)))
            except ValueError as exc:
                raise McpRpcError(
                    code=-32602,
                    message="Invalid params",
                    data={"invalid": key, "index": index},
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

    def _require_session_access(self, *, actor: dict[str, Any], session_id: UUID) -> Any:
        session = self._memory_admin_service.get_session(
            session_id=session_id, include_deleted=True
        )
        return self._require_existing_owned_resource(
            actor=actor,
            resource="session",
            resource_id=session_id,
            record=session,
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
        collection = self._memory_admin_service.find_collection(
            collection_id=collection_id,
            include_deleted=True,
        )
        return self._require_existing_owned_resource(
            actor=actor,
            resource="collection",
            resource_id=collection_id,
            record=collection,
        )
