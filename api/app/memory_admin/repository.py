from __future__ import annotations

from app.embeddings import embed_text

from .repository_collections import (
    add_collection_items,
    create_collection,
    get_collection,
    list_collections,
    remove_collection_item,
    soft_delete_collection,
    update_collection,
)
from .repository_engrams import (
    _build_engram_json_payload,
    _build_engram_update_fields,
    _build_retrieval_text,
    _replace_engram_sources,
    get_admin_engram,
    list_admin_engrams,
    list_engram_sources,
    move_admin_engram_project,
    restore_engram,
    soft_delete_engram,
    update_admin_engram,
)
from .repository_sessions import (
    get_admin_session,
    list_admin_sessions,
    restore_session,
    soft_delete_linked_engrams,
    soft_delete_session,
)
from .repository_types import (
    AdminEngramListRepositoryRequest,
    AdminEngramUpdateRepositoryRequest,
    AdminSessionListRepositoryRequest,
    CollectionListRepositoryRequest,
    _AdminEngramUpdatePayload,
)


def _compute_engram_embedding(*, retrieval_text: str, embedding_dim: int) -> tuple[str, str]:
    embedding = embed_text(retrieval_text, dim=embedding_dim)
    embedding_literal = "[" + ",".join(f"{value:.6f}" for value in embedding.vector) + "]"
    return embedding.provider_id, embedding_literal


__all__ = [
    "_AdminEngramUpdatePayload",
    "_build_engram_json_payload",
    "_build_engram_update_fields",
    "_build_retrieval_text",
    "_compute_engram_embedding",
    "_replace_engram_sources",
    "AdminEngramListRepositoryRequest",
    "AdminEngramUpdateRepositoryRequest",
    "AdminSessionListRepositoryRequest",
    "CollectionListRepositoryRequest",
    "add_collection_items",
    "create_collection",
    "get_admin_engram",
    "get_admin_session",
    "get_collection",
    "list_admin_engrams",
    "list_admin_sessions",
    "list_collections",
    "list_engram_sources",
    "move_admin_engram_project",
    "remove_collection_item",
    "restore_engram",
    "restore_session",
    "soft_delete_collection",
    "soft_delete_engram",
    "soft_delete_linked_engrams",
    "soft_delete_session",
    "update_admin_engram",
    "update_collection",
]
