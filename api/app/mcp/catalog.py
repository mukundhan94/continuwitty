from __future__ import annotations

from copy import deepcopy
from typing import Any

_READ_TOOL_NAMES = {
    "chat.list_sessions",
    "chat.get_session",
    "chat.get_lifecycle_policy",
    "chat.list_messages",
    "chat.list_timeline",
    "chat.list_pinned_engrams",
    "chat.list_pinned_documents",
    "chat.list_project_documents",
    "project.list",
    "project.export_bundle",
    "project.get_default",
    "engram.list",
    "engram.get",
    "engram.collection_list",
    "engram.query",
    "engram.rehydrate",
    "user.get_profile",
    "user.list_projects",
}

_WRITE_TOOL_NAMES = {
    "chat.create_session",
    "chat.update_lifecycle_policy",
    "chat.send_message",
    "chat.pin_engram",
    "chat.unpin_engram",
    "chat.pin_document",
    "chat.unpin_document",
    "chat.save_as_engram",
    "chat.continue_session",
    "chat.delete_session",
    "chat.restore_session",
    "project.create",
    "project.import_bundle",
    "project.set_default",
    "engram.create",
    "engram.create_from_conversation",
    "engram.pin_to_session",
    "engram.update",
    "engram.move_project",
    "engram.delete",
    "engram.restore",
    "engram.collection_create",
    "engram.collection_update",
    "engram.collection_delete",
    "engram.collection_add_items",
    "engram.collection_remove_items",
}

# Alias maps to existing implementation branch but should obey the same scope semantics.
_TOOL_ALIASES = {"engram.pin_to_session": "chat.pin_engram"}

_OPTIONAL_PROJECT_TOOLS = {
    "engram.query",
    "chat.list_sessions",
    "chat.list_project_documents",
    "engram.list",
    "engram.collection_list",
}

_PROJECT_FALLBACK_TOOLS = {
    "chat.save_as_engram",
    "engram.create",
    "engram.create_from_conversation",
    "engram.collection_create",
}

_SESSION_SCOPED_TOOLS = {
    "chat.get_session",
    "chat.get_lifecycle_policy",
    "chat.update_lifecycle_policy",
    "chat.list_messages",
    "chat.list_timeline",
    "chat.send_message",
    "chat.list_pinned_engrams",
    "chat.pin_engram",
    "chat.unpin_engram",
    "chat.list_pinned_documents",
    "chat.pin_document",
    "chat.unpin_document",
    "chat.continue_session",
    "engram.pin_to_session",
    "chat.delete_session",
    "chat.restore_session",
}

_ENGRAM_SCOPED_TOOLS = {
    "engram.get",
    "engram.update",
    "engram.move_project",
    "engram.delete",
    "engram.restore",
}

_TOOL_NAMESPACE_PREFIXES = ("chat", "engram", "project", "user")


def _to_public_tool_name(canonical_name: str) -> str:
    """Expose VS Code-compatible tool names (no dots)."""
    return canonical_name.replace(".", "_")


def _to_dotted_tool_name(tool_name: str) -> str:
    """Convert external tool names back to dotted canonical method names.

    We intentionally support both forms:
    - dotted (`chat.send_message`) for backward compatibility
    - underscore (`chat_send_message`) for strict MCP clients.
    """
    if "." in tool_name:
        return tool_name
    for namespace in _TOOL_NAMESPACE_PREFIXES:
        prefix = f"{namespace}_"
        if tool_name.startswith(prefix):
            return f"{namespace}.{tool_name[len(prefix) :]}"
    return tool_name


_TOOL_CATALOG: tuple[dict[str, Any], ...] = (
    {
        "name": "chat.create_session",
        "description": "Create a chat session in a project.",
        "inputSchema": {
            "type": "object",
            "required": ["project_id", "title"],
            "properties": {
                "project_id": {"type": "string"},
                "title": {"type": "string"},
                "provider": {"type": "string"},
                "model_id": {"type": "string"},
                "system_prompt": {"type": "string"},
                "visibility_scope": {"type": "string", "enum": ["private", "project"]},
                "autosave_enabled": {"type": "boolean"},
                "autosave_strategy": {
                    "type": "string",
                    "enum": ["off", "interval", "message_count"],
                },
                "autosave_interval_minutes": {"type": "integer", "minimum": 1},
                "autosave_min_messages": {"type": "integer", "minimum": 1},
                "retention_days": {"type": "integer", "minimum": 1},
                "retention_max_snapshots": {"type": "integer", "minimum": 1},
            },
        },
    },
    {
        "name": "chat.list_sessions",
        "description": "List chat sessions for the authenticated user.",
        "inputSchema": {
            "type": "object",
            "properties": {
                "project_id": {"type": "string"},
                "limit": {"type": "integer", "minimum": 1},
                "offset": {"type": "integer", "minimum": 0},
            },
        },
    },
    {
        "name": "chat.get_session",
        "description": "Get chat session metadata by session_id.",
        "inputSchema": {
            "type": "object",
            "required": ["session_id"],
            "properties": {"session_id": {"type": "string", "format": "uuid"}},
        },
    },
    {
        "name": "chat.get_lifecycle_policy",
        "description": "Get autosave/retention lifecycle policy for a session.",
        "inputSchema": {
            "type": "object",
            "required": ["session_id"],
            "properties": {"session_id": {"type": "string", "format": "uuid"}},
        },
    },
    {
        "name": "chat.update_lifecycle_policy",
        "description": "Update autosave/retention lifecycle policy for a session.",
        "inputSchema": {
            "type": "object",
            "required": ["session_id"],
            "properties": {
                "session_id": {"type": "string", "format": "uuid"},
                "autosave_enabled": {"type": "boolean"},
                "autosave_strategy": {
                    "type": "string",
                    "enum": ["off", "interval", "message_count"],
                },
                "autosave_interval_minutes": {"type": "integer", "minimum": 1},
                "autosave_min_messages": {"type": "integer", "minimum": 1},
                "retention_days": {"type": "integer", "minimum": 1},
                "retention_max_snapshots": {"type": "integer", "minimum": 1},
            },
        },
    },
    {
        "name": "chat.list_messages",
        "description": "List persisted messages for a chat session.",
        "inputSchema": {
            "type": "object",
            "required": ["session_id"],
            "properties": {
                "session_id": {"type": "string", "format": "uuid"},
                "limit": {"type": "integer", "minimum": 1},
                "offset": {"type": "integer", "minimum": 0},
            },
        },
    },
    {
        "name": "chat.list_timeline",
        "description": "List timeline events (manual saves, autosaves, consolidation) for a session.",
        "inputSchema": {
            "type": "object",
            "required": ["session_id"],
            "properties": {
                "session_id": {"type": "string", "format": "uuid"},
                "limit": {"type": "integer", "minimum": 1},
                "offset": {"type": "integer", "minimum": 0},
            },
        },
    },
    {
        "name": "chat.send_message",
        "description": "Send a user message in a chat session (streaming supported).",
        "inputSchema": {
            "type": "object",
            "required": ["session_id", "content_text"],
            "properties": {
                "session_id": {"type": "string", "format": "uuid"},
                "content_text": {"type": "string"},
                "stream": {"type": "boolean"},
            },
        },
    },
    {
        "name": "chat.list_pinned_engrams",
        "description": "List engrams pinned to a chat session.",
        "inputSchema": {
            "type": "object",
            "required": ["session_id"],
            "properties": {"session_id": {"type": "string", "format": "uuid"}},
        },
    },
    {
        "name": "chat.pin_engram",
        "description": "Pin an engram to a chat session context chain.",
        "inputSchema": {
            "type": "object",
            "required": ["session_id", "engram_id"],
            "properties": {
                "session_id": {"type": "string", "format": "uuid"},
                "engram_id": {"type": "string", "format": "uuid"},
            },
        },
    },
    {
        "name": "chat.unpin_engram",
        "description": "Remove a pinned engram from a chat session.",
        "inputSchema": {
            "type": "object",
            "required": ["session_id", "engram_id"],
            "properties": {
                "session_id": {"type": "string", "format": "uuid"},
                "engram_id": {"type": "string", "format": "uuid"},
            },
        },
    },
    {
        "name": "chat.list_pinned_documents",
        "description": "List uploaded documents pinned to a chat session.",
        "inputSchema": {
            "type": "object",
            "required": ["session_id"],
            "properties": {"session_id": {"type": "string", "format": "uuid"}},
        },
    },
    {
        "name": "chat.pin_document",
        "description": "Pin an uploaded document into chat context.",
        "inputSchema": {
            "type": "object",
            "required": ["session_id", "document_id"],
            "properties": {
                "session_id": {"type": "string", "format": "uuid"},
                "document_id": {"type": "string", "format": "uuid"},
            },
        },
    },
    {
        "name": "chat.unpin_document",
        "description": "Remove a pinned document from chat context.",
        "inputSchema": {
            "type": "object",
            "required": ["session_id", "document_id"],
            "properties": {
                "session_id": {"type": "string", "format": "uuid"},
                "document_id": {"type": "string", "format": "uuid"},
            },
        },
    },
    {
        "name": "chat.list_project_documents",
        "description": "List ingested documents visible in a project scope.",
        "inputSchema": {
            "type": "object",
            "properties": {
                "project_id": {"type": "string"},
                "limit": {"type": "integer", "minimum": 1},
                "offset": {"type": "integer", "minimum": 0},
            },
        },
    },
    {
        "name": "chat.save_as_engram",
        "description": (
            "Save as engram. Supports either a chat session snapshot "
            "or direct conversation markdown when no session_id exists."
        ),
        "inputSchema": {
            "type": "object",
            "properties": {
                "session_id": {"type": "string", "format": "uuid"},
                "project_id": {"type": "string"},
                "conversation_markdown": {"type": "string"},
                "thread_id": {"type": "string"},
                "title": {"type": "string"},
                "abstract": {"type": "string"},
                "visibility_scope": {"type": "string", "enum": ["private", "project"]},
                "tags": {"type": "array", "items": {"type": "string"}},
                "keywords": {"type": "array", "items": {"type": "string"}},
                "retrieval_text": {"type": "string"},
                "source_session_id": {"type": "string", "format": "uuid"},
            },
        },
    },
    {
        "name": "chat.continue_session",
        "description": "Create a continued session carrying pinned engram context.",
        "inputSchema": {
            "type": "object",
            "required": ["session_id"],
            "properties": {
                "session_id": {"type": "string", "format": "uuid"},
                "title": {"type": "string"},
            },
        },
    },
    {
        "name": "engram.create",
        "description": "Create a new memory engram with summary metadata.",
        "inputSchema": {
            "type": "object",
            "required": ["title", "detailed_summary_markdown"],
            "properties": {
                "project_id": {"type": "string"},
                "thread_id": {"type": "string"},
                "title": {"type": "string"},
                "abstract": {"type": "string"},
                "detailed_summary_markdown": {"type": "string"},
                "visibility_scope": {"type": "string", "enum": ["private", "project"]},
                "tags": {"type": "array", "items": {"type": "string"}},
                "keywords": {"type": "array", "items": {"type": "string"}},
            },
        },
    },
    {
        "name": "engram.create_from_conversation",
        "description": (
            "Create an engram from conversation markdown with optional "
            "fill-empty metadata enrichment."
        ),
        "inputSchema": {
            "type": "object",
            "required": ["conversation_markdown"],
            "properties": {
                "project_id": {"type": "string"},
                "conversation_markdown": {"type": "string"},
                "thread_id": {"type": "string"},
                "title": {"type": "string"},
                "abstract": {"type": "string"},
                "visibility_scope": {"type": "string", "enum": ["private", "project"]},
                "tags": {"type": "array", "items": {"type": "string"}},
                "keywords": {"type": "array", "items": {"type": "string"}},
                "retrieval_text": {"type": "string"},
                "source_session_id": {"type": "string", "format": "uuid"},
            },
        },
    },
    {
        "name": "engram.query",
        "description": "Query engrams by semantic text plus metadata filters.",
        "inputSchema": {
            "type": "object",
            "required": ["query"],
            "properties": {
                "query": {"type": "string"},
                "top_k": {"type": "integer", "minimum": 1, "maximum": 50},
                "project_id": {"type": "string"},
                "tags": {"type": "array", "items": {"type": "string"}},
                "keywords": {"type": "array", "items": {"type": "string"}},
                "created_after": {"type": "string", "format": "date-time"},
                "created_before": {"type": "string", "format": "date-time"},
            },
        },
    },
    {
        "name": "engram.rehydrate",
        "description": "Return a compact and citation-packed rehydration bundle.",
        "inputSchema": {
            "type": "object",
            "required": ["engram_id"],
            "properties": {"engram_id": {"type": "string", "format": "uuid"}},
        },
    },
    {
        "name": "engram.pin_to_session",
        "description": "Pin an engram to a chat session context chain.",
        "inputSchema": {
            "type": "object",
            "required": ["session_id", "engram_id"],
            "properties": {
                "session_id": {"type": "string", "format": "uuid"},
                "engram_id": {"type": "string", "format": "uuid"},
            },
        },
    },
    {
        "name": "chat.delete_session",
        "description": "Soft-delete a chat session with optional linked-engram deletion.",
        "inputSchema": {
            "type": "object",
            "required": ["session_id"],
            "properties": {
                "session_id": {"type": "string", "format": "uuid"},
                "delete_linked_engrams": {"type": "boolean"},
                "reason": {"type": "string"},
            },
        },
    },
    {
        "name": "chat.restore_session",
        "description": "Restore a previously soft-deleted chat session.",
        "inputSchema": {
            "type": "object",
            "required": ["session_id"],
            "properties": {"session_id": {"type": "string", "format": "uuid"}},
        },
    },
    {
        "name": "project.list",
        "description": "List visible projects for the authenticated actor.",
        "inputSchema": {
            "type": "object",
            "properties": {
                "include_archived": {"type": "boolean"},
                "limit": {"type": "integer", "minimum": 1},
                "offset": {"type": "integer", "minimum": 0},
            },
        },
    },
    {
        "name": "project.create",
        "description": "Create a project; admins can optionally set owner_user_id.",
        "inputSchema": {
            "type": "object",
            "required": ["project_id", "name"],
            "properties": {
                "project_id": {"type": "string"},
                "name": {"type": "string"},
                "description": {"type": "string"},
                "owner_user_id": {"type": "string", "format": "uuid"},
            },
        },
    },
    {
        "name": "project.get_default",
        "description": "Get the authenticated actor's default project id.",
        "inputSchema": {"type": "object", "properties": {}},
    },
    {
        "name": "project.set_default",
        "description": "Set the authenticated actor's default project id.",
        "inputSchema": {
            "type": "object",
            "required": ["project_id"],
            "properties": {"project_id": {"type": "string"}},
        },
    },
    {
        "name": "project.export_bundle",
        "description": (
            "Export a project memory bundle with optional collection filters and "
            "optional embedding inclusion flag."
        ),
        "inputSchema": {
            "type": "object",
            "required": ["project_id"],
            "properties": {
                "project_id": {"type": "string"},
                "collection_ids": {
                    "type": "array",
                    "items": {"type": "string", "format": "uuid"},
                },
                "include_embeddings": {"type": "boolean"},
            },
        },
    },
    {
        "name": "project.import_bundle",
        "description": (
            "Import a project bundle into a target project with deterministic "
            "conflict handling. Provide either bundle (object) or bundle_json (string)."
        ),
        "inputSchema": {
            "type": "object",
            "required": ["project_id"],
            "properties": {
                "project_id": {"type": "string"},
                "conflict_policy": {
                    "type": "string",
                    "enum": ["skip", "overwrite", "rename"],
                },
                "bundle": {"type": "object"},
                "bundle_json": {"type": "string"},
            },
            "anyOf": [{"required": ["bundle"]}, {"required": ["bundle_json"]}],
        },
    },
    {
        "name": "engram.list",
        "description": "List engrams with project/session/query filters.",
        "inputSchema": {
            "type": "object",
            "properties": {
                "project_id": {"type": "string"},
                "session_id": {"type": "string", "format": "uuid"},
                "q": {"type": "string"},
                "include_deleted": {"type": "boolean"},
                "limit": {"type": "integer", "minimum": 1},
                "offset": {"type": "integer", "minimum": 0},
            },
        },
    },
    {
        "name": "engram.get",
        "description": "Get an engram in management format with editable source payload.",
        "inputSchema": {
            "type": "object",
            "required": ["engram_id"],
            "properties": {
                "engram_id": {"type": "string", "format": "uuid"},
                "include_deleted": {"type": "boolean"},
            },
        },
    },
    {
        "name": "engram.update",
        "description": "Update engram metadata/markdown/sources with optimistic concurrency.",
        "inputSchema": {
            "type": "object",
            "required": ["engram_id"],
            "properties": {
                "engram_id": {"type": "string", "format": "uuid"},
                "title": {"type": "string"},
                "abstract": {"type": "string"},
                "detailed_summary_markdown": {"type": "string"},
                "tags": {"type": "array", "items": {"type": "string"}},
                "keywords": {"type": "array", "items": {"type": "string"}},
                "visibility_scope": {"type": "string", "enum": ["private", "project"]},
                "expected_updated_at": {"type": "string", "format": "date-time"},
                "sources": {
                    "type": "array",
                    "items": {
                        "type": "object",
                        "required": ["captured_at", "url", "title", "snippet"],
                        "properties": {
                            "captured_at": {"type": "string", "format": "date-time"},
                            "url": {"type": "string"},
                            "title": {"type": "string"},
                            "snippet": {"type": "string"},
                            "content_text": {"type": "string"},
                            "content_hash": {"type": "string"},
                        },
                    },
                },
            },
        },
    },
    {
        "name": "engram.move_project",
        "description": "Move an engram to another project and auto-detach invalid collections.",
        "inputSchema": {
            "type": "object",
            "required": ["engram_id", "target_project_id"],
            "properties": {
                "engram_id": {"type": "string", "format": "uuid"},
                "target_project_id": {"type": "string"},
                "reason": {"type": "string"},
                "expected_updated_at": {"type": "string", "format": "date-time"},
            },
        },
    },
    {
        "name": "engram.delete",
        "description": "Soft-delete an engram.",
        "inputSchema": {
            "type": "object",
            "required": ["engram_id"],
            "properties": {
                "engram_id": {"type": "string", "format": "uuid"},
                "reason": {"type": "string"},
            },
        },
    },
    {
        "name": "engram.restore",
        "description": "Restore a soft-deleted engram.",
        "inputSchema": {
            "type": "object",
            "required": ["engram_id"],
            "properties": {"engram_id": {"type": "string", "format": "uuid"}},
        },
    },
    {
        "name": "engram.collection_list",
        "description": "List engram collections (project-bounded groups).",
        "inputSchema": {
            "type": "object",
            "properties": {
                "project_id": {"type": "string"},
                "include_deleted": {"type": "boolean"},
                "limit": {"type": "integer", "minimum": 1},
                "offset": {"type": "integer", "minimum": 0},
            },
        },
    },
    {
        "name": "engram.collection_create",
        "description": "Create an engram collection for a project.",
        "inputSchema": {
            "type": "object",
            "required": ["name"],
            "properties": {
                "project_id": {"type": "string"},
                "name": {"type": "string"},
                "description": {"type": "string"},
            },
        },
    },
    {
        "name": "engram.collection_update",
        "description": "Update collection metadata with optimistic concurrency.",
        "inputSchema": {
            "type": "object",
            "required": ["collection_id"],
            "properties": {
                "collection_id": {"type": "string", "format": "uuid"},
                "name": {"type": "string"},
                "description": {"type": "string"},
                "expected_updated_at": {"type": "string", "format": "date-time"},
            },
        },
    },
    {
        "name": "engram.collection_delete",
        "description": "Soft-delete a collection.",
        "inputSchema": {
            "type": "object",
            "required": ["collection_id"],
            "properties": {
                "collection_id": {"type": "string", "format": "uuid"},
                "reason": {"type": "string"},
            },
        },
    },
    {
        "name": "engram.collection_add_items",
        "description": "Add one or more engrams to a collection.",
        "inputSchema": {
            "type": "object",
            "required": ["collection_id", "engram_ids"],
            "properties": {
                "collection_id": {"type": "string", "format": "uuid"},
                "engram_ids": {
                    "type": "array",
                    "items": {"type": "string", "format": "uuid"},
                },
            },
        },
    },
    {
        "name": "engram.collection_remove_items",
        "description": "Remove one engram from a collection.",
        "inputSchema": {
            "type": "object",
            "required": ["collection_id", "engram_id"],
            "properties": {
                "collection_id": {"type": "string", "format": "uuid"},
                "engram_id": {"type": "string", "format": "uuid"},
            },
        },
    },
    {
        "name": "user.get_profile",
        "description": "Get the authenticated user profile.",
        "inputSchema": {"type": "object", "properties": {}},
    },
    {
        "name": "user.list_projects",
        "description": "List project IDs visible to the authenticated user.",
        "inputSchema": {"type": "object", "properties": {}},
    },
)


def _build_namespace_tool_catalog(namespace: str) -> list[dict[str, Any]]:
    prefix = f"{namespace}."
    return [deepcopy(item) for item in _TOOL_CATALOG if item["name"].startswith(prefix)]


def _build_chat_tool_catalog() -> list[dict[str, Any]]:
    return _build_namespace_tool_catalog("chat")


def _build_engram_tool_catalog() -> list[dict[str, Any]]:
    return _build_namespace_tool_catalog("engram")


def _build_project_tool_catalog() -> list[dict[str, Any]]:
    return _build_namespace_tool_catalog("project")


def _build_user_tool_catalog() -> list[dict[str, Any]]:
    return _build_namespace_tool_catalog("user")


def build_tool_catalog() -> list[dict[str, Any]]:
    """Return the full MCP tool catalog with input schemas."""
    return [
        *_build_chat_tool_catalog(),
        *_build_engram_tool_catalog(),
        *_build_project_tool_catalog(),
        *_build_user_tool_catalog(),
    ]
