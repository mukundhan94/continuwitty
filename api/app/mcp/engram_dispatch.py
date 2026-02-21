from __future__ import annotations

from collections.abc import Callable
from dataclasses import dataclass
from typing import Any
from uuid import UUID

from app.models import (
    AdminEngramDeleteRequest,
    EngramCollectionDeleteRequest,
    EngramCollectionItemsUpdateRequest,
    EngramCollectionUpdateRequest,
    EngramQueryRequest,
)

from .errors import McpRpcError


@dataclass(frozen=True)
class EngramDispatchContext:
    actor: dict[str, Any]
    actor_user_id: UUID
    method: str
    params: dict[str, Any]


@dataclass(frozen=True)
class EngramDispatchDependencies:
    memory_admin_service: Any
    embedding_dim: int
    parse_uuid: Callable[[dict[str, Any], str], UUID]
    parse_uuid_list: Callable[[dict[str, Any], str], list[UUID]]
    require_collection_access: Callable[..., Any]
    require_engram_access: Callable[..., Any]
    list_engrams_for_actor: Callable[..., list[Any]]
    list_collections_for_actor: Callable[..., list[Any]]
    query_engrams: Callable[..., list[Any]]
    get_rehydration_bundle: Callable[..., Any]


def dispatch_engram_collection_mutation_tool(
    *,
    dependencies: EngramDispatchDependencies,
    context: EngramDispatchContext,
) -> dict[str, Any] | None:
    collection_mutation_tools = {
        "engram.collection_update",
        "engram.collection_delete",
        "engram.collection_add_items",
        "engram.collection_remove_items",
    }
    if context.method not in collection_mutation_tools:
        return None

    collection_id = dependencies.parse_uuid(context.params, "collection_id")
    dependencies.require_collection_access(actor=context.actor, collection_id=collection_id)

    if context.method == "engram.collection_update":
        updated_collection = dependencies.memory_admin_service.update_collection(
            collection_id=collection_id,
            payload=EngramCollectionUpdateRequest(
                name=context.params.get("name"),
                description=context.params.get("description"),
                expected_updated_at=context.params.get("expected_updated_at"),
            ),
        )
        return {"collection": updated_collection.model_dump(mode="json")}

    if context.method == "engram.collection_delete":
        result = dependencies.memory_admin_service.delete_collection(
            collection_id=collection_id,
            actor_user_id=context.actor_user_id,
            payload=EngramCollectionDeleteRequest(reason=context.params.get("reason")),
        )
        return {"result": result}

    if context.method == "engram.collection_add_items":
        result = dependencies.memory_admin_service.add_collection_items(
            collection_id=collection_id,
            actor_user_id=context.actor_user_id,
            payload=EngramCollectionItemsUpdateRequest(
                engram_ids=dependencies.parse_uuid_list(context.params, "engram_ids")
            ),
        )
        return {"result": result}

    result = dependencies.memory_admin_service.remove_collection_item(
        collection_id=collection_id,
        engram_id=dependencies.parse_uuid(context.params, "engram_id"),
    )
    return {"result": result}


def dispatch_engram_state_mutation_tool(
    *,
    dependencies: EngramDispatchDependencies,
    context: EngramDispatchContext,
) -> dict[str, Any] | None:
    if context.method not in {"engram.delete", "engram.restore"}:
        return None

    engram_id = dependencies.parse_uuid(context.params, "engram_id")
    dependencies.require_engram_access(
        actor=context.actor,
        engram_id=engram_id,
        include_deleted=True,
    )

    if context.method == "engram.delete":
        deleted = dependencies.memory_admin_service.delete_engram(
            engram_id=engram_id,
            actor_user_id=context.actor_user_id,
            payload=AdminEngramDeleteRequest(reason=context.params.get("reason")),
        )
        return {"result": deleted.model_dump(mode="json")}

    restored = dependencies.memory_admin_service.restore_engram(engram_id=engram_id)
    return {"result": restored.model_dump(mode="json")}


def dispatch_engram_mutation_tool(
    *,
    dependencies: EngramDispatchDependencies,
    context: EngramDispatchContext,
) -> dict[str, Any] | None:
    collection_mutation_result = dispatch_engram_collection_mutation_tool(
        dependencies=dependencies,
        context=context,
    )
    if collection_mutation_result is not None:
        return collection_mutation_result
    return dispatch_engram_state_mutation_tool(
        dependencies=dependencies,
        context=context,
    )


def dispatch_engram_read_tool(
    *,
    dependencies: EngramDispatchDependencies,
    context: EngramDispatchContext,
) -> dict[str, Any] | None:
    def _query() -> dict[str, Any]:
        results = dependencies.query_engrams(
            request=EngramQueryRequest(**context.params),
            embedding_dim=dependencies.embedding_dim,
            actor_user_id=context.actor_user_id,
        )
        return {"results": [item.model_dump(mode="json") for item in results]}

    def _rehydrate() -> dict[str, Any]:
        engram_id = dependencies.parse_uuid(context.params, "engram_id")
        bundle = dependencies.get_rehydration_bundle(
            engram_id,
            actor_user_id=context.actor_user_id,
        )
        if not bundle:
            raise McpRpcError(
                code=-32004,
                message="Engram not found",
                data={"engram_id": str(engram_id)},
            )
        return {"bundle": bundle.model_dump(mode="json")}

    def _list_engrams() -> dict[str, Any]:
        session_id_value = context.params.get("session_id")
        session_id = (
            dependencies.parse_uuid({"session_id": session_id_value}, "session_id")
            if session_id_value is not None
            else None
        )
        listed = dependencies.list_engrams_for_actor(
            actor=context.actor,
            actor_user_id=context.actor_user_id,
            session_id=session_id,
            params=context.params,
        )
        return {"engrams": [item.model_dump(mode="json") for item in listed]}

    def _get_engram() -> dict[str, Any]:
        engram_id = dependencies.parse_uuid(context.params, "engram_id")
        include_deleted = bool(context.params.get("include_deleted", True))
        engram = dependencies.require_engram_access(
            actor=context.actor,
            engram_id=engram_id,
            include_deleted=include_deleted,
        )
        return {"engram": engram.model_dump(mode="json")}

    def _list_collections() -> dict[str, Any]:
        collections = dependencies.list_collections_for_actor(
            actor=context.actor,
            actor_user_id=context.actor_user_id,
            params=context.params,
        )
        return {"collections": [item.model_dump(mode="json") for item in collections]}

    handlers = {
        "engram.query": _query,
        "engram.rehydrate": _rehydrate,
        "engram.list": _list_engrams,
        "engram.get": _get_engram,
        "engram.collection_list": _list_collections,
    }
    handler = handlers.get(context.method)
    return handler() if handler else None
