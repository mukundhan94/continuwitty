package mcp

var toolCatalogEntries = map[string]toolCatalogEntry{
	"chat.create_session": {
		description: "Create a chat session in a project.",
		inputSchema: map[string]any{
			"type": "object",
			"required": []any{
				"project_id",
				"title",
			},
			"properties": map[string]any{
				"project_id": map[string]any{
					"type": "string",
				},
				"title": map[string]any{
					"type": "string",
				},
				"provider": map[string]any{
					"type": "string",
				},
				"model_id": map[string]any{
					"type": "string",
				},
				"system_prompt": map[string]any{
					"type": "string",
				},
				"visibility_scope": map[string]any{
					"type": "string",
					"enum": []any{
						"private",
						"project",
					},
				},
				"autosave_enabled": map[string]any{
					"type": "boolean",
				},
				"autosave_strategy": map[string]any{
					"type": "string",
					"enum": []any{
						"off",
						"interval",
						"message_count",
					},
				},
				"autosave_interval_minutes": map[string]any{
					"type":    "integer",
					"minimum": 1,
				},
				"autosave_min_messages": map[string]any{
					"type":    "integer",
					"minimum": 1,
				},
				"retention_days": map[string]any{
					"type":    "integer",
					"minimum": 1,
				},
				"retention_max_snapshots": map[string]any{
					"type":    "integer",
					"minimum": 1,
				},
			},
		},
	},
	"chat.list_sessions": {
		description: "List chat sessions for the authenticated user.",
		inputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"project_id": map[string]any{
					"type": "string",
				},
				"limit": map[string]any{
					"type":    "integer",
					"minimum": 1,
				},
				"offset": map[string]any{
					"type":    "integer",
					"minimum": 0,
				},
			},
		},
	},
	"chat.get_session": {
		description: "Get chat session metadata by session_id.",
		inputSchema: map[string]any{
			"type": "object",
			"required": []any{
				"session_id",
			},
			"properties": map[string]any{
				"session_id": map[string]any{
					"type":   "string",
					"format": "uuid",
				},
			},
		},
	},
	"chat.get_lifecycle_policy": {
		description: "Get autosave/retention lifecycle policy for a session.",
		inputSchema: map[string]any{
			"type": "object",
			"required": []any{
				"session_id",
			},
			"properties": map[string]any{
				"session_id": map[string]any{
					"type":   "string",
					"format": "uuid",
				},
			},
		},
	},
	"chat.update_lifecycle_policy": {
		description: "Update autosave/retention lifecycle policy for a session.",
		inputSchema: map[string]any{
			"type": "object",
			"required": []any{
				"session_id",
			},
			"properties": map[string]any{
				"session_id": map[string]any{
					"type":   "string",
					"format": "uuid",
				},
				"autosave_enabled": map[string]any{
					"type": "boolean",
				},
				"autosave_strategy": map[string]any{
					"type": "string",
					"enum": []any{
						"off",
						"interval",
						"message_count",
					},
				},
				"autosave_interval_minutes": map[string]any{
					"type":    "integer",
					"minimum": 1,
				},
				"autosave_min_messages": map[string]any{
					"type":    "integer",
					"minimum": 1,
				},
				"retention_days": map[string]any{
					"type":    "integer",
					"minimum": 1,
				},
				"retention_max_snapshots": map[string]any{
					"type":    "integer",
					"minimum": 1,
				},
			},
		},
	},
	"chat.list_messages": {
		description: "List persisted messages for a chat session.",
		inputSchema: map[string]any{
			"type": "object",
			"required": []any{
				"session_id",
			},
			"properties": map[string]any{
				"session_id": map[string]any{
					"type":   "string",
					"format": "uuid",
				},
				"limit": map[string]any{
					"type":    "integer",
					"minimum": 1,
				},
				"offset": map[string]any{
					"type":    "integer",
					"minimum": 0,
				},
			},
		},
	},
	"chat.list_timeline": {
		description: "List timeline events (manual saves, autosaves, consolidation) for a session.",
		inputSchema: map[string]any{
			"type": "object",
			"required": []any{
				"session_id",
			},
			"properties": map[string]any{
				"session_id": map[string]any{
					"type":   "string",
					"format": "uuid",
				},
				"limit": map[string]any{
					"type":    "integer",
					"minimum": 1,
				},
				"offset": map[string]any{
					"type":    "integer",
					"minimum": 0,
				},
			},
		},
	},
	"chat.send_message": {
		description: "Send a user message in a chat session (streaming supported).",
		inputSchema: map[string]any{
			"type": "object",
			"required": []any{
				"session_id",
				"content_text",
			},
			"properties": map[string]any{
				"session_id": map[string]any{
					"type":   "string",
					"format": "uuid",
				},
				"content_text": map[string]any{
					"type": "string",
				},
				"stream": map[string]any{
					"type": "boolean",
				},
			},
		},
	},
	"chat.list_pinned_engrams": {
		description: "List engrams pinned to a chat session.",
		inputSchema: map[string]any{
			"type": "object",
			"required": []any{
				"session_id",
			},
			"properties": map[string]any{
				"session_id": map[string]any{
					"type":   "string",
					"format": "uuid",
				},
			},
		},
	},
	"chat.pin_engram": {
		description: "Pin an engram to a chat session context chain.",
		inputSchema: map[string]any{
			"type": "object",
			"required": []any{
				"session_id",
				"engram_id",
			},
			"properties": map[string]any{
				"session_id": map[string]any{
					"type":   "string",
					"format": "uuid",
				},
				"engram_id": map[string]any{
					"type":   "string",
					"format": "uuid",
				},
			},
		},
	},
	"chat.unpin_engram": {
		description: "Remove a pinned engram from a chat session.",
		inputSchema: map[string]any{
			"type": "object",
			"required": []any{
				"session_id",
				"engram_id",
			},
			"properties": map[string]any{
				"session_id": map[string]any{
					"type":   "string",
					"format": "uuid",
				},
				"engram_id": map[string]any{
					"type":   "string",
					"format": "uuid",
				},
			},
		},
	},
	"chat.list_pinned_documents": {
		description: "List uploaded documents pinned to a chat session.",
		inputSchema: map[string]any{
			"type": "object",
			"required": []any{
				"session_id",
			},
			"properties": map[string]any{
				"session_id": map[string]any{
					"type":   "string",
					"format": "uuid",
				},
			},
		},
	},
	"chat.pin_document": {
		description: "Pin an uploaded document into chat context.",
		inputSchema: map[string]any{
			"type": "object",
			"required": []any{
				"session_id",
				"document_id",
			},
			"properties": map[string]any{
				"session_id": map[string]any{
					"type":   "string",
					"format": "uuid",
				},
				"document_id": map[string]any{
					"type":   "string",
					"format": "uuid",
				},
			},
		},
	},
	"chat.unpin_document": {
		description: "Remove a pinned document from chat context.",
		inputSchema: map[string]any{
			"type": "object",
			"required": []any{
				"session_id",
				"document_id",
			},
			"properties": map[string]any{
				"session_id": map[string]any{
					"type":   "string",
					"format": "uuid",
				},
				"document_id": map[string]any{
					"type":   "string",
					"format": "uuid",
				},
			},
		},
	},
	"chat.list_project_documents": {
		description: "List ingested documents visible in a project scope.",
		inputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"project_id": map[string]any{
					"type": "string",
				},
				"limit": map[string]any{
					"type":    "integer",
					"minimum": 1,
				},
				"offset": map[string]any{
					"type":    "integer",
					"minimum": 0,
				},
			},
		},
	},
	"chat.save_as_engram": {
		description: "Save as engram. Supports either a chat session snapshot or direct conversation markdown when no session_id exists.",
		inputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"session_id": map[string]any{
					"type":   "string",
					"format": "uuid",
				},
				"project_id": map[string]any{
					"type": "string",
				},
				"conversation_markdown": map[string]any{
					"type": "string",
				},
				"thread_id": map[string]any{
					"type": "string",
				},
				"title": map[string]any{
					"type": "string",
				},
				"abstract": map[string]any{
					"type": "string",
				},
				"visibility_scope": map[string]any{
					"type": "string",
					"enum": []any{
						"private",
						"project",
					},
				},
				"tags": map[string]any{
					"type": "array",
					"items": map[string]any{
						"type": "string",
					},
				},
				"keywords": map[string]any{
					"type": "array",
					"items": map[string]any{
						"type": "string",
					},
				},
				"retrieval_text": map[string]any{
					"type": "string",
				},
				"source_session_id": map[string]any{
					"type":   "string",
					"format": "uuid",
				},
			},
		},
	},
	"chat.continue_session": {
		description: "Create a continued session carrying pinned engram context.",
		inputSchema: map[string]any{
			"type": "object",
			"required": []any{
				"session_id",
			},
			"properties": map[string]any{
				"session_id": map[string]any{
					"type":   "string",
					"format": "uuid",
				},
				"title": map[string]any{
					"type": "string",
				},
			},
		},
	},
	"engram.create": {
		description: "Create a new memory engram with summary metadata.",
		inputSchema: map[string]any{
			"type": "object",
			"required": []any{
				"title",
				"detailed_summary_markdown",
			},
			"properties": map[string]any{
				"project_id": map[string]any{
					"type": "string",
				},
				"thread_id": map[string]any{
					"type": "string",
				},
				"title": map[string]any{
					"type": "string",
				},
				"abstract": map[string]any{
					"type": "string",
				},
				"detailed_summary_markdown": map[string]any{
					"type": "string",
				},
				"visibility_scope": map[string]any{
					"type": "string",
					"enum": []any{
						"private",
						"project",
					},
				},
				"tags": map[string]any{
					"type": "array",
					"items": map[string]any{
						"type": "string",
					},
				},
				"keywords": map[string]any{
					"type": "array",
					"items": map[string]any{
						"type": "string",
					},
				},
			},
		},
	},
	"engram.create_from_conversation": {
		description: "Create an engram from conversation markdown with optional fill-empty metadata enrichment.",
		inputSchema: map[string]any{
			"type": "object",
			"required": []any{
				"conversation_markdown",
			},
			"properties": map[string]any{
				"project_id": map[string]any{
					"type": "string",
				},
				"conversation_markdown": map[string]any{
					"type": "string",
				},
				"thread_id": map[string]any{
					"type": "string",
				},
				"title": map[string]any{
					"type": "string",
				},
				"abstract": map[string]any{
					"type": "string",
				},
				"visibility_scope": map[string]any{
					"type": "string",
					"enum": []any{
						"private",
						"project",
					},
				},
				"tags": map[string]any{
					"type": "array",
					"items": map[string]any{
						"type": "string",
					},
				},
				"keywords": map[string]any{
					"type": "array",
					"items": map[string]any{
						"type": "string",
					},
				},
				"retrieval_text": map[string]any{
					"type": "string",
				},
				"source_session_id": map[string]any{
					"type":   "string",
					"format": "uuid",
				},
			},
		},
	},
	"engram.query": {
		description: "Query engrams by semantic text plus metadata filters.",
		inputSchema: map[string]any{
			"type": "object",
			"required": []any{
				"query",
			},
			"properties": map[string]any{
				"query": map[string]any{
					"type": "string",
				},
				"top_k": map[string]any{
					"type":    "integer",
					"minimum": 1,
					"maximum": 50,
				},
				"project_id": map[string]any{
					"type": "string",
				},
				"tags": map[string]any{
					"type": "array",
					"items": map[string]any{
						"type": "string",
					},
				},
				"keywords": map[string]any{
					"type": "array",
					"items": map[string]any{
						"type": "string",
					},
				},
				"created_after": map[string]any{
					"type":   "string",
					"format": "date-time",
				},
				"created_before": map[string]any{
					"type":   "string",
					"format": "date-time",
				},
			},
		},
	},
	"engram.rehydrate": {
		description: "Return a compact and citation-packed rehydration bundle.",
		inputSchema: map[string]any{
			"type": "object",
			"required": []any{
				"engram_id",
			},
			"properties": map[string]any{
				"engram_id": map[string]any{
					"type":   "string",
					"format": "uuid",
				},
			},
		},
	},
	"engram.trace_path": {
		description: "Trace depth-limited link paths from a source engram.",
		inputSchema: map[string]any{
			"type": "object",
			"required": []any{
				"engram_id",
			},
			"properties": map[string]any{
				"engram_id": map[string]any{
					"type":   "string",
					"format": "uuid",
				},
				"max_depth": map[string]any{
					"type":    "integer",
					"minimum": 1,
				},
				"max_neighbors": map[string]any{
					"type":    "integer",
					"minimum": 1,
				},
				"include_archived": map[string]any{
					"type": "boolean",
				},
			},
		},
	},
	"engram.link_suggest": {
		description: "Suggest source->target links using semantic/source-overlap heuristics.",
		inputSchema: map[string]any{
			"type": "object",
			"required": []any{
				"engram_id",
			},
			"properties": map[string]any{
				"engram_id": map[string]any{
					"type":   "string",
					"format": "uuid",
				},
				"limit": map[string]any{
					"type":    "integer",
					"minimum": 1,
				},
				"max_candidates": map[string]any{
					"type":    "integer",
					"minimum": 1,
				},
				"minimum_score": map[string]any{
					"type":    "number",
					"minimum": 0,
					"maximum": 1,
				},
				"include_archived": map[string]any{
					"type": "boolean",
				},
			},
		},
	},
	"engram.pin_to_session": {
		description: "Pin an engram to a chat session context chain.",
		inputSchema: map[string]any{
			"type": "object",
			"required": []any{
				"session_id",
				"engram_id",
			},
			"properties": map[string]any{
				"session_id": map[string]any{
					"type":   "string",
					"format": "uuid",
				},
				"engram_id": map[string]any{
					"type":   "string",
					"format": "uuid",
				},
			},
		},
	},
	"chat.delete_session": {
		description: "Soft-delete a chat session with optional linked-engram deletion.",
		inputSchema: map[string]any{
			"type": "object",
			"required": []any{
				"session_id",
			},
			"properties": map[string]any{
				"session_id": map[string]any{
					"type":   "string",
					"format": "uuid",
				},
				"delete_linked_engrams": map[string]any{
					"type": "boolean",
				},
				"reason": map[string]any{
					"type": "string",
				},
			},
		},
	},
	"chat.restore_session": {
		description: "Restore a previously soft-deleted chat session.",
		inputSchema: map[string]any{
			"type": "object",
			"required": []any{
				"session_id",
			},
			"properties": map[string]any{
				"session_id": map[string]any{
					"type":   "string",
					"format": "uuid",
				},
			},
		},
	},
	"project.list": {
		description: "List visible projects for the authenticated actor.",
		inputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"include_archived": map[string]any{
					"type": "boolean",
				},
				"limit": map[string]any{
					"type":    "integer",
					"minimum": 1,
				},
				"offset": map[string]any{
					"type":    "integer",
					"minimum": 0,
				},
			},
		},
	},
	"project.create": {
		description: "Create a project; admins can optionally set owner_user_id.",
		inputSchema: map[string]any{
			"type": "object",
			"required": []any{
				"project_id",
				"name",
			},
			"properties": map[string]any{
				"project_id": map[string]any{
					"type": "string",
				},
				"name": map[string]any{
					"type": "string",
				},
				"description": map[string]any{
					"type": "string",
				},
				"owner_user_id": map[string]any{
					"type":   "string",
					"format": "uuid",
				},
			},
		},
	},
	"project.get_default": {
		description: "Get the authenticated actor's default project id.",
		inputSchema: map[string]any{
			"type":       "object",
			"properties": map[string]any{},
		},
	},
	"project.set_default": {
		description: "Set the authenticated actor's default project id.",
		inputSchema: map[string]any{
			"type": "object",
			"required": []any{
				"project_id",
			},
			"properties": map[string]any{
				"project_id": map[string]any{
					"type": "string",
				},
			},
		},
	},
	"project.member_list": {
		description: "List project members (owner/admin only).",
		inputSchema: map[string]any{
			"type": "object",
			"required": []any{
				"project_id",
			},
			"properties": map[string]any{
				"project_id": map[string]any{
					"type": "string",
				},
				"include_revoked": map[string]any{
					"type": "boolean",
				},
				"limit": map[string]any{
					"type":    "integer",
					"minimum": 1,
				},
				"offset": map[string]any{
					"type":    "integer",
					"minimum": 0,
				},
			},
		},
	},
	"project.member_add": {
		description: "Add a project member role (owner/admin only).",
		inputSchema: map[string]any{
			"type": "object",
			"required": []any{
				"project_id",
				"user_id",
				"role",
			},
			"properties": map[string]any{
				"project_id": map[string]any{
					"type": "string",
				},
				"user_id": map[string]any{
					"type":   "string",
					"format": "uuid",
				},
				"role": map[string]any{
					"type": "string",
					"enum": []any{
						"editor",
						"viewer",
					},
				},
			},
		},
	},
	"project.member_update": {
		description: "Update a project member role (owner/admin only).",
		inputSchema: map[string]any{
			"type": "object",
			"required": []any{
				"project_id",
				"user_id",
				"role",
			},
			"properties": map[string]any{
				"project_id": map[string]any{
					"type": "string",
				},
				"user_id": map[string]any{
					"type":   "string",
					"format": "uuid",
				},
				"role": map[string]any{
					"type": "string",
					"enum": []any{
						"editor",
						"viewer",
					},
				},
			},
		},
	},
	"project.member_remove": {
		description: "Remove/revoke a project member (owner/admin only).",
		inputSchema: map[string]any{
			"type": "object",
			"required": []any{
				"project_id",
				"user_id",
			},
			"properties": map[string]any{
				"project_id": map[string]any{
					"type": "string",
				},
				"user_id": map[string]any{
					"type":   "string",
					"format": "uuid",
				},
			},
		},
	},
	"project.export_bundle": {
		description: "Export a project memory bundle with optional collection filters and optional embedding inclusion flag.",
		inputSchema: map[string]any{
			"type": "object",
			"required": []any{
				"project_id",
			},
			"properties": map[string]any{
				"project_id": map[string]any{
					"type": "string",
				},
				"collection_ids": map[string]any{
					"type": "array",
					"items": map[string]any{
						"type":   "string",
						"format": "uuid",
					},
				},
				"include_embeddings": map[string]any{
					"type": "boolean",
				},
			},
		},
	},
	"project.import_bundle": {
		description: "Import a project bundle into a target project with deterministic conflict handling. Provide either bundle (object) or bundle_json (string).",
		inputSchema: map[string]any{
			"type": "object",
			"required": []any{
				"project_id",
			},
			"properties": map[string]any{
				"project_id": map[string]any{
					"type": "string",
				},
				"conflict_policy": map[string]any{
					"type": "string",
					"enum": []any{
						"skip",
						"overwrite",
						"rename",
					},
				},
				"bundle": map[string]any{
					"type": "object",
				},
				"bundle_json": map[string]any{
					"type": "string",
				},
			},
			"anyOf": []any{
				map[string]any{
					"required": []any{
						"bundle",
					},
				},
				map[string]any{
					"required": []any{
						"bundle_json",
					},
				},
			},
		},
	},
	"engram.list": {
		description: "List engrams with project/session/query filters.",
		inputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"project_id": map[string]any{
					"type": "string",
				},
				"session_id": map[string]any{
					"type":   "string",
					"format": "uuid",
				},
				"q": map[string]any{
					"type": "string",
				},
				"include_deleted": map[string]any{
					"type": "boolean",
				},
				"limit": map[string]any{
					"type":    "integer",
					"minimum": 1,
				},
				"offset": map[string]any{
					"type":    "integer",
					"minimum": 0,
				},
			},
		},
	},
	"engram.get": {
		description: "Get an engram in management format with editable source payload.",
		inputSchema: map[string]any{
			"type": "object",
			"required": []any{
				"engram_id",
			},
			"properties": map[string]any{
				"engram_id": map[string]any{
					"type":   "string",
					"format": "uuid",
				},
				"include_deleted": map[string]any{
					"type": "boolean",
				},
			},
		},
	},
	"engram.update": {
		description: "Update engram metadata/markdown/sources with optimistic concurrency.",
		inputSchema: map[string]any{
			"type": "object",
			"required": []any{
				"engram_id",
			},
			"properties": map[string]any{
				"engram_id": map[string]any{
					"type":   "string",
					"format": "uuid",
				},
				"title": map[string]any{
					"type": "string",
				},
				"abstract": map[string]any{
					"type": "string",
				},
				"detailed_summary_markdown": map[string]any{
					"type": "string",
				},
				"tags": map[string]any{
					"type": "array",
					"items": map[string]any{
						"type": "string",
					},
				},
				"keywords": map[string]any{
					"type": "array",
					"items": map[string]any{
						"type": "string",
					},
				},
				"visibility_scope": map[string]any{
					"type": "string",
					"enum": []any{
						"private",
						"project",
					},
				},
				"expected_updated_at": map[string]any{
					"type":   "string",
					"format": "date-time",
				},
				"sources": map[string]any{
					"type": "array",
					"items": map[string]any{
						"type": "object",
						"required": []any{
							"captured_at",
							"url",
							"title",
							"snippet",
						},
						"properties": map[string]any{
							"captured_at": map[string]any{
								"type":   "string",
								"format": "date-time",
							},
							"url": map[string]any{
								"type": "string",
							},
							"title": map[string]any{
								"type": "string",
							},
							"snippet": map[string]any{
								"type": "string",
							},
							"content_text": map[string]any{
								"type": "string",
							},
							"content_hash": map[string]any{
								"type": "string",
							},
						},
					},
				},
			},
		},
	},
	"engram.link_list": {
		description: "List outgoing links for one source engram.",
		inputSchema: map[string]any{
			"type": "object",
			"required": []any{
				"engram_id",
			},
			"properties": map[string]any{
				"engram_id": map[string]any{
					"type":   "string",
					"format": "uuid",
				},
				"relation_type": map[string]any{
					"type": "string",
					"enum": []any{
						"supports",
						"depends_on",
						"contradicts",
						"related_to",
						"derived_from",
					},
				},
				"include_archived": map[string]any{
					"type": "boolean",
				},
				"limit": map[string]any{
					"type":    "integer",
					"minimum": 1,
				},
				"offset": map[string]any{
					"type":    "integer",
					"minimum": 0,
				},
			},
		},
	},
	"engram.link_create": {
		description: "Create an engram-to-engram directed link.",
		inputSchema: map[string]any{
			"type": "object",
			"required": []any{
				"engram_id",
				"target_engram_id",
			},
			"properties": map[string]any{
				"engram_id": map[string]any{
					"type":   "string",
					"format": "uuid",
				},
				"target_engram_id": map[string]any{
					"type":   "string",
					"format": "uuid",
				},
				"relation_type": map[string]any{
					"type": "string",
					"enum": []any{
						"supports",
						"depends_on",
						"contradicts",
						"related_to",
						"derived_from",
					},
				},
				"weight": map[string]any{
					"type":    "number",
					"minimum": 0,
					"maximum": 1,
				},
				"temporal_weight": map[string]any{
					"type":    "number",
					"minimum": 0,
					"maximum": 1,
				},
				"confidence": map[string]any{
					"type":    "number",
					"minimum": 0,
					"maximum": 1,
				},
				"origin": map[string]any{
					"type": "string",
					"enum": []any{
						"manual",
						"suggested",
						"inferred",
						"system",
					},
				},
				"status": map[string]any{
					"type": "string",
					"enum": []any{
						"active",
						"suggested",
						"archived",
						"rejected",
					},
				},
				"evidence_json": map[string]any{
					"type": "object",
				},
				"last_reinforced_at": map[string]any{
					"type":   "string",
					"format": "date-time",
				},
			},
		},
	},
	"engram.link_update": {
		description: "Update mutable link fields for one link id.",
		inputSchema: map[string]any{
			"type": "object",
			"required": []any{
				"link_id",
			},
			"properties": map[string]any{
				"link_id": map[string]any{
					"type":   "string",
					"format": "uuid",
				},
				"weight": map[string]any{
					"type":    "number",
					"minimum": 0,
					"maximum": 1,
				},
				"temporal_weight": map[string]any{
					"type":    "number",
					"minimum": 0,
					"maximum": 1,
				},
				"confidence": map[string]any{
					"type":    "number",
					"minimum": 0,
					"maximum": 1,
				},
				"status": map[string]any{
					"type": "string",
					"enum": []any{
						"active",
						"suggested",
						"archived",
						"rejected",
					},
				},
				"evidence_json": map[string]any{
					"type": "object",
				},
				"last_reinforced_at": map[string]any{
					"type":   "string",
					"format": "date-time",
				},
			},
		},
	},
	"engram.link_archive": {
		description: "Archive (soft-delete) one engram link.",
		inputSchema: map[string]any{
			"type": "object",
			"required": []any{
				"link_id",
			},
			"properties": map[string]any{
				"link_id": map[string]any{
					"type":   "string",
					"format": "uuid",
				},
			},
		},
	},
	"engram.share": {
		description: "Set an engram visibility scope to project.",
		inputSchema: map[string]any{
			"type": "object",
			"required": []any{
				"engram_id",
			},
			"properties": map[string]any{
				"engram_id": map[string]any{
					"type":   "string",
					"format": "uuid",
				},
			},
		},
	},
	"engram.unshare": {
		description: "Set an engram visibility scope to private.",
		inputSchema: map[string]any{
			"type": "object",
			"required": []any{
				"engram_id",
			},
			"properties": map[string]any{
				"engram_id": map[string]any{
					"type":   "string",
					"format": "uuid",
				},
			},
		},
	},
	"engram.move_project": {
		description: "Move an engram to another project and auto-detach invalid collections.",
		inputSchema: map[string]any{
			"type": "object",
			"required": []any{
				"engram_id",
				"target_project_id",
			},
			"properties": map[string]any{
				"engram_id": map[string]any{
					"type":   "string",
					"format": "uuid",
				},
				"target_project_id": map[string]any{
					"type": "string",
				},
				"reason": map[string]any{
					"type": "string",
				},
				"expected_updated_at": map[string]any{
					"type":   "string",
					"format": "date-time",
				},
			},
		},
	},
	"engram.delete": {
		description: "Soft-delete an engram.",
		inputSchema: map[string]any{
			"type": "object",
			"required": []any{
				"engram_id",
			},
			"properties": map[string]any{
				"engram_id": map[string]any{
					"type":   "string",
					"format": "uuid",
				},
				"reason": map[string]any{
					"type": "string",
				},
			},
		},
	},
	"engram.restore": {
		description: "Restore a soft-deleted engram.",
		inputSchema: map[string]any{
			"type": "object",
			"required": []any{
				"engram_id",
			},
			"properties": map[string]any{
				"engram_id": map[string]any{
					"type":   "string",
					"format": "uuid",
				},
			},
		},
	},
	"engram.collection_list": {
		description: "List engram collections (project-bounded groups).",
		inputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"project_id": map[string]any{
					"type": "string",
				},
				"include_deleted": map[string]any{
					"type": "boolean",
				},
				"limit": map[string]any{
					"type":    "integer",
					"minimum": 1,
				},
				"offset": map[string]any{
					"type":    "integer",
					"minimum": 0,
				},
			},
		},
	},
	"engram.collection_create": {
		description: "Create an engram collection for a project.",
		inputSchema: map[string]any{
			"type": "object",
			"required": []any{
				"name",
			},
			"properties": map[string]any{
				"project_id": map[string]any{
					"type": "string",
				},
				"name": map[string]any{
					"type": "string",
				},
				"description": map[string]any{
					"type": "string",
				},
			},
		},
	},
	"engram.collection_update": {
		description: "Update collection metadata with optimistic concurrency.",
		inputSchema: map[string]any{
			"type": "object",
			"required": []any{
				"collection_id",
			},
			"properties": map[string]any{
				"collection_id": map[string]any{
					"type":   "string",
					"format": "uuid",
				},
				"name": map[string]any{
					"type": "string",
				},
				"description": map[string]any{
					"type": "string",
				},
				"expected_updated_at": map[string]any{
					"type":   "string",
					"format": "date-time",
				},
			},
		},
	},
	"engram.collection_delete": {
		description: "Soft-delete a collection.",
		inputSchema: map[string]any{
			"type": "object",
			"required": []any{
				"collection_id",
			},
			"properties": map[string]any{
				"collection_id": map[string]any{
					"type":   "string",
					"format": "uuid",
				},
				"reason": map[string]any{
					"type": "string",
				},
			},
		},
	},
	"engram.collection_add_items": {
		description: "Add one or more engrams to a collection.",
		inputSchema: map[string]any{
			"type": "object",
			"required": []any{
				"collection_id",
				"engram_ids",
			},
			"properties": map[string]any{
				"collection_id": map[string]any{
					"type":   "string",
					"format": "uuid",
				},
				"engram_ids": map[string]any{
					"type": "array",
					"items": map[string]any{
						"type":   "string",
						"format": "uuid",
					},
				},
			},
		},
	},
	"engram.collection_remove_items": {
		description: "Remove one engram from a collection.",
		inputSchema: map[string]any{
			"type": "object",
			"required": []any{
				"collection_id",
				"engram_id",
			},
			"properties": map[string]any{
				"collection_id": map[string]any{
					"type":   "string",
					"format": "uuid",
				},
				"engram_id": map[string]any{
					"type":   "string",
					"format": "uuid",
				},
			},
		},
	},
	"user.get_profile": {
		description: "Get the authenticated user profile.",
		inputSchema: map[string]any{
			"type":       "object",
			"properties": map[string]any{},
		},
	},
	"user.list_projects": {
		description: "List project IDs visible to the authenticated user.",
		inputSchema: map[string]any{
			"type":       "object",
			"properties": map[string]any{},
		},
	},
}
