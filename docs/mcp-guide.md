# Engram Vault - MCP Guide

> Complete guide to Model Context Protocol (MCP) integration.
> See [MCP Client Integrations](mcp-client-integrations.md) for external client setup (LibreChat, VS Code Copilot, Codex).
> See [API Reference](api-reference.md) for REST endpoints.

---

## Transport

### Endpoint

- `POST /api/v1/mcp/stream` — JSON-RPC over SSE
- `GET /api/v1/mcp/stream` — probe metadata
- `HEAD /api/v1/mcp/stream` — probe health
- `POST /api/v1/mcp/stream` is transport rate-limited and returns `429` with `Retry-After` when exceeded.

### Content Negotiation

- `POST /api/v1/mcp/stream` returns JSON when `Accept` includes `application/json`.
- It returns SSE when `Accept` explicitly prefers `text/event-stream`.
- If both are sent with equal priority (common in editor MCP clients), JSON is returned for better `initialize` compatibility.
- JSON-RPC notifications without `id` (for example `notifications/initialized`) are accepted with `202` and no body.

### SSE Framing

- `event: jsonrpc`
- `data: <JSON-RPC frame>`

### Frame Types

- **Success:** `{"jsonrpc":"2.0","id":"...","result":{...}}`
- **Error:** `{"jsonrpc":"2.0","id":"...","error":{"code":...,"message":"...","data":{...}}}`
- **Progress:** `{"jsonrpc":"2.0","method":"mcp.event","params":{"id":"...","tool":"chat.send_message","event":"chunk|meta|done","data":{...}}}`

### Error Code Mapping (Tool Calls)

- `-32602` Invalid params:
  - missing required argument (for example `{"missing":"name"}`)
  - invalid UUID input (for example `collection_id` set to a name string)
- `-32003` Forbidden:
  - token scope/policy or resource authorization denied
- `-32004` Not found:
  - referenced resource/project does not exist or is not visible
- `-32009` Conflict:
  - write conflict such as duplicate collection name in a project
- `error.data.suggested_action` is included for authorization/not-found/conflict cases to help clients auto-remediate.

Example conflict payload (duplicate collection name):

```json
{
  "jsonrpc": "2.0",
  "id": "collection-create-2",
  "error": {
    "code": -32009,
    "message": "Collection name already exists for this project",
    "data": {
      "status_code": 409,
      "detail": "Collection name already exists for this project",
      "suggested_action": "use_unique_collection_name_or_update_existing_collection"
    }
  }
}
```

---

## Authentication

### Session Cookie Auth

Preferred for external tools: authenticated UI session cookies (sign in via `/login`).

### MCP Bearer Token Auth

Preferred for external tools: MCP bearer token via `Authorization: Bearer <token>`.

Bearer and session auth resolve the same actor model and visibility checks.
Some MCP clients preflight with `GET`/`HEAD`; these return `200` to avoid noisy `405` logs.

### OAuth Authorization

For MCP clients that support dynamic registration (e.g., VS Code Copilot):
- `/.well-known/oauth-authorization-server` discovery
- `POST /oauth/register` dynamic client registration (local default allows automatic registration)
- `GET /oauth/authorize` + `POST /oauth/token` PKCE authorization code flow (`S256` only)
- Token exchange issues short-lived MCP bearer tokens

When `OAUTH_REQUIRE_PROTECTED_REGISTRATION=true`, registration requires an authenticated admin session.

---

## MCP Token Workflow

`ENGRAM_MCP_TOKEN` is first-class and backed by persisted token records. Create token credentials as admin, store only the plaintext token client-side, and send it in the `Authorization` header for MCP calls.

For client-specific setup commands, see [MCP Client Integrations](mcp-client-integrations.md).

### Token Policy Model

- **Scope:** `read` or `write`
- **Optional `allowed_tools`:** when empty, all tools in scope are allowed
- **Optional `allowed_project_ids`:** when empty, no additional project restriction; when provided, calls are limited to those projects
- **Default token expiry:** 90 days

### Create a Read Token (Admin Session)

```bash
COOKIE_JAR=/tmp/engram-admin.cookies
BASE_URL=http://localhost:8000
USERNAME=admin
PASSWORD=admin123

CSRF_TOKEN=$(
  curl -s -c "$COOKIE_JAR" "$BASE_URL/login" \
  | sed -n 's/.*name="csrf_token" value="\([^"]*\)".*/\1/p' \
  | head -n 1
)

curl -s -b "$COOKIE_JAR" -c "$COOKIE_JAR" \
  -X POST "$BASE_URL/login" \
  -H "Content-Type: application/x-www-form-urlencoded" \
  --data-urlencode "username=$USERNAME" \
  --data-urlencode "password=$PASSWORD" \
  --data-urlencode "csrf_token=$CSRF_TOKEN" >/dev/null

TOKEN_JSON=$(
  curl -s -b "$COOKIE_JAR" \
    -H "Content-Type: application/json" \
    -X POST "$BASE_URL/api/v1/mcp/tokens" \
    -d '{
      "name": "librechat-read-token",
      "scope": "read",
      "allowed_tools": [],
      "allowed_project_ids": ["engram-vault"],
      "expires_in_days": 90
    }'
)

echo "$TOKEN_JSON" | jq
MCP_TOKEN=$(echo "$TOKEN_JSON" | jq -r '.token')
```

### Use Bearer Token

```bash
curl -sN \
  -H "Accept: text/event-stream" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $MCP_TOKEN" \
  -X POST "$BASE_URL/api/v1/mcp/stream" \
  -d '{
    "jsonrpc": "2.0",
    "id": "tools-list-bearer",
    "method": "tools/list",
    "params": {}
  }' \
  | sed -n 's/^data: //p' \
  | jq
```

### Revoke Token

```bash
TOKEN_ID=$(echo "$TOKEN_JSON" | jq -r '.token_id')
curl -s -b "$COOKIE_JAR" \
  -H "Content-Type: application/json" \
  -X POST "$BASE_URL/api/v1/mcp/tokens/$TOKEN_ID/revoke" \
  -d '{"reason":"rotation"}' | jq
```

### LibreChat Bearer Config Example

```json
{
  "type": "sse",
  "url": "https://your-host.example.com/api/v1/mcp/stream",
  "headers": {
    "Authorization": "Bearer engram_mcp_<token_id_hex>_<secret>",
    "Accept": "text/event-stream"
  }
}
```

---

## Compatibility Methods

- `initialize`
- `tools/list`
- `tools/call`

`initialize` includes a `policy` block with governance metadata:

- `tool_policy_version`
- `eval_suite_version`

---

## Tool Naming

- `tools/list` exposes client-safe tool names with underscores (e.g., `chat_send_message`)
- For backward compatibility, dotted names (`chat.send_message`) are still accepted in direct calls and `tools/call`
- For any argument ending with `_id` (for example `collection_id`, `engram_id`, `session_id`), pass UUID values, not human-readable names.
- `chat_save_as_engram` supports two modes:
  - Session snapshot mode with `session_id`
  - Conversation-only mode with `project_id` + `conversation_markdown` (no session required)

---

## Tool Catalog

### Chat Tools

- `chat.create_session`
- `chat.list_sessions`
- `chat.get_session`
- `chat.list_messages`
- `chat.get_lifecycle_policy`
- `chat.update_lifecycle_policy`
- `chat.list_timeline`
- `chat.send_message`
- `chat.list_pinned_engrams`
- `chat.pin_engram`
- `chat.unpin_engram`
- `chat.list_pinned_documents`
- `chat.pin_document`
- `chat.unpin_document`
- `chat.list_project_documents`
- `chat.save_as_engram`
- `chat.continue_session`

### Engram Tools

- `engram.create`
- `engram.create_from_conversation`
- `engram.query`
- `engram.rehydrate`
- `engram.link_create`
- `engram.link_list`
- `engram.link_update`
- `engram.link_archive`
- `engram.link_suggest`
- `engram.trace_path`
- `engram.feedback` (optional `session_id`, optional `integration_depth` [`mentioned`,`elaborated`,`contradicted`,`ignored`], optional `relevance_score` 1-5)
- `engram.refresh_freshness` (admin maintenance)
- `engram.consolidation_list` (admin maintenance)
- `engram.refresh_consolidation` (admin maintenance)
- `engram.consolidation_action` (admin maintenance)
- `engram.contradiction_list` (admin maintenance)
- `engram.refresh_contradictions` (admin maintenance)
- `engram.contradiction_resolve` (admin maintenance)
- `engram.curation_list` (admin maintenance)
- `engram.curation_refresh_links` (admin maintenance; refreshes link-hygiene recommendations into curation suggestions for one source engram, optional `project_id` scope must match source engram project)
- `engram.curation_action` (admin maintenance; `status=applied` triggers linked consolidation/contradiction downstream actions and payload-linked link archival actions (`archive_*`, `review_relation_conflict`))
- `engram.pin_to_session`
- `engram.share`
- `engram.unshare`

### Project Tools

- `project.list`
- `project.create`
- `project.get_default`
- `project.set_default`
- `project.member_list`
- `project.member_add`
- `project.member_update`
- `project.member_remove`
- `project.export_bundle`
- `project.import_bundle`

Note: project audit-event listing is currently REST-only (`GET /api/v1/projects/{project_id}/audit-events`) and is not exposed as an MCP tool in this phase.

### User Tools

- `user.get_profile`
- `user.list_projects`

---

## Invocation Styles

### 1. Direct Tool Method (Backward-Compatible)

```json
{
  "jsonrpc": "2.0",
  "id": "tool-call-1",
  "method": "chat.send_message",
  "params": {
    "session_id": "00000000-0000-0000-0000-000000000000",
    "content_text": "Summarize the pinned engrams",
    "stream": true
  }
}
```

### 2. MCP-Compatible `tools/call`

```json
{
  "jsonrpc": "2.0",
  "id": "tool-call-2",
  "method": "tools/call",
  "params": {
    "name": "chat.send_message",
      "arguments": {
        "session_id": "00000000-0000-0000-0000-000000000000",
        "content_text": "Summarize the pinned engrams",
        "context_token_budget": 1200,
        "link_recall_enabled": true,
        "link_recall_depth": 1,
        "link_recall_max_neighbors": 8,
        "link_noise_suppression_enabled": true,
        "link_noise_score_threshold": 0.30,
        "stream": true
      }
    }
  }
```

---

## Quick Start (curl)

```bash
# 1) Login and persist session cookie
COOKIE_JAR=/tmp/engram-mcp.cookies
BASE_URL=http://localhost:8000
USERNAME=admin
PASSWORD=admin123

CSRF_TOKEN=$(
  curl -s -c "$COOKIE_JAR" "$BASE_URL/login" \
  | sed -n 's/.*name="csrf_token" value="\([^"]*\)".*/\1/p' \
  | head -n 1
)

curl -s -i -b "$COOKIE_JAR" -c "$COOKIE_JAR" \
  -X POST "$BASE_URL/login" \
  -H "Content-Type: application/x-www-form-urlencoded" \
  --data-urlencode "username=$USERNAME" \
  --data-urlencode "password=$PASSWORD" \
  --data-urlencode "csrf_token=$CSRF_TOKEN" \
  | head -n 1

# Expect: HTTP/1.1 303 See Other
```

```bash
# 2) Verify session auth is active
curl -s -b "$COOKIE_JAR" "$BASE_URL/api/v1/me" | jq
```

```bash
# 3) MCP initialize
curl -sN -b "$COOKIE_JAR" \
  -H "Accept: text/event-stream" \
  -H "Content-Type: application/json" \
  -X POST "$BASE_URL/api/v1/mcp/stream" \
  -d '{
    "jsonrpc": "2.0",
    "id": "init-1",
    "method": "initialize",
    "params": {
      "protocolVersion": "2025-03-26",
      "clientInfo": {"name": "curl-client", "version": "0.1.0"}
    }
  }' \
  | sed -n 's/^data: //p' \
  | jq
```

```bash
# 4) Discover tools
curl -sN -b "$COOKIE_JAR" \
  -H "Accept: text/event-stream" \
  -H "Content-Type: application/json" \
  -X POST "$BASE_URL/api/v1/mcp/stream" \
  -d '{
    "jsonrpc": "2.0",
    "id": "tools-list-1",
    "method": "tools/list",
    "params": {}
  }' \
  | sed -n 's/^data: //p' \
  | jq
```

```bash
# 5) Persist conversation-only memory with auto-metadata enrichment
curl -sN -b "$COOKIE_JAR" \
  -H "Accept: text/event-stream" \
  -H "Content-Type: application/json" \
  -X POST "$BASE_URL/api/v1/mcp/stream" \
  -d '{
    "jsonrpc": "2.0",
    "id": "engram-from-conv-1",
    "method": "tools/call",
    "params": {
      "name": "engram.create_from_conversation",
      "arguments": {
        "project_id": "engram-vault",
        "conversation_markdown": "## User\nCheckout latency spiked after deploy.\n## Assistant\nLikely cache miss storm and connection pool saturation."
      }
    }
  }' \
  | sed -n 's/^data: //p' \
  | jq
```

```bash
# 6) Streaming chat tool call
curl -sN -b "$COOKIE_JAR" \
  -H "Accept: text/event-stream" \
  -H "Content-Type: application/json" \
  -X POST "$BASE_URL/api/v1/mcp/stream" \
  -d '{
    "jsonrpc": "2.0",
    "id": "chat-stream-1",
    "method": "tools/call",
    "params": {
      "name": "chat.send_message",
      "arguments": {
        "session_id": "00000000-0000-0000-0000-000000000000",
        "content_text": "Summarize pinned engrams and list action items.",
        "context_token_budget": 1200,
        "link_recall_enabled": true,
        "link_recall_depth": 1,
        "link_recall_max_neighbors": 8,
        "link_noise_suppression_enabled": true,
        "link_noise_score_threshold": 0.30,
        "stream": true
      }
    }
  }'
```

---

## Example Calls by Tool Group

### chat.*

```json
{
  "jsonrpc": "2.0",
  "id": "chat-create-1",
  "method": "chat.create_session",
  "params": {
    "project_id": "engram-vault",
    "title": "MCP chat session",
    "provider": "openai",
    "model_id": "gpt-4o-mini",
    "visibility_scope": "private",
    "autosave_enabled": true,
    "autosave_strategy": "interval",
    "autosave_interval_minutes": 30,
    "autosave_min_messages": 4,
    "retention_days": 30,
    "retention_max_snapshots": 25
  }
}
```

`chat.send_message` responses include:

- `cw_plan_applied` (optional normalized `cw>` directive plan)
- `prompt_policy_version`
- `used_engram_ids`
- `used_engram_link_ids`
- `engram_trace_paths`
- `contradiction_warnings` (trace-derived contradiction risk guidance)
- `used_document_chunk_ids`
- `source_references`
- `retrieval_audit` (blocked candidate count + trace suppression/filtering/truncation + cross-project usage signals)

`content_text` can begin with `cw>` to activate ContinuWitty query-protocol planning. The first-line directive is parsed and excluded from the stored/context query text.

`chat.send_message` request arguments can also include optional bounded recall controls:

- `context_token_budget`
- `link_recall_enabled`
- `link_recall_depth`
- `link_recall_max_neighbors`
- `link_noise_suppression_enabled`
- `link_noise_score_threshold`

Document pin/list helpers:

```json
{
  "jsonrpc": "2.0",
  "id": "chat-pin-doc-1",
  "method": "tools/call",
  "params": {
    "name": "chat.pin_document",
    "arguments": {
      "session_id": "00000000-0000-0000-0000-000000000000",
      "document_id": "11111111-1111-1111-1111-111111111111"
    }
  }
}
```

```json
{
  "jsonrpc": "2.0",
  "id": "chat-list-pins-1",
  "method": "tools/call",
  "params": {
    "name": "chat.list_pinned_documents",
    "arguments": {
      "session_id": "00000000-0000-0000-0000-000000000000"
    }
  }
}
```

### engram.*

```json
{
  "jsonrpc": "2.0",
  "id": "engram-query-1",
  "method": "engram.query",
  "params": {
    "query": "incident mitigation",
    "project_id": "engram-vault",
    "top_k": 5,
    "useful_count_min": 1,
    "useful_count_max": 8,
    "access_count_min": 2,
    "access_count_max": 20,
    "feedback_count_min": 3,
    "feedback_count_max": 12,
    "contradiction_count_max": 2,
    "contradiction_feedback_ratio_max": 0.4,
    "freshness_score_min": 0.4,
    "freshness_score_max": 0.9,
    "useful_feedback_ratio_min": 0.7,
    "source_session_quality_min": 0.7,
    "last_accessed_after": "2026-02-01T00:00:00Z",
    "last_accessed_before": "2026-03-01T00:00:00Z",
    "relation_type": "supports",
    "trace_depth": 1
  }
}
```

`engram.query` also supports optional `created_after` / `created_before` and `freshness_computed_after` / `freshness_computed_before` (RFC3339), plus `useful_count_min`, `useful_count_max`, `access_count_min`, `access_count_max`, `feedback_count_min`, `feedback_count_max`, `contradiction_count_max`, `contradiction_feedback_ratio_max`, `freshness_score_min`, `freshness_score_max`, `useful_feedback_ratio_min`, `avg_relevance_feedback_min`, `source_session_quality_min`, and trace constraints (`relation_type`, `trace_depth`).
Returned rows include `source_session_quality_score` (`0..1`) plus `access_count`, `freshness_score`, `feedback_count`, `useful_count`, `avg_relevance_feedback` (`0..1`), `useful_feedback_ratio` (`0..1`), `contradiction_count`, and `contradiction_feedback_ratio` (`0..1`) for authority/quality diagnostics.

Collection add-items flow (resolve collection UUID first):

```json
{
  "jsonrpc": "2.0",
  "id": "collection-list-1",
  "method": "tools/call",
  "params": {
    "name": "engram.collection_list",
    "arguments": {
      "project_id": "engram-vault"
    }
  }
}
```

Use the returned `collections[].collection_id` UUID:

```json
{
  "jsonrpc": "2.0",
  "id": "collection-add-items-1",
  "method": "tools/call",
  "params": {
    "name": "engram.collection_add_items",
    "arguments": {
      "collection_id": "00000000-0000-0000-0000-000000000000",
      "engram_ids": ["11111111-1111-1111-1111-111111111111"]
    }
  }
}
```

```json
{
  "jsonrpc": "2.0",
  "id": "engram-create-conversation-1",
  "method": "tools/call",
  "params": {
    "name": "engram.create_from_conversation",
    "arguments": {
      "project_id": "engram-vault",
      "conversation_markdown": "## User\nDatabase latency spiked.\n## Assistant\nLikely cache miss storm after deploy."
    }
  }
}
```

Expected response highlights:

- `result.engram`: persisted engram payload
- `result.enrichment_report.enrichment_applied`: whether empty metadata fields were auto-filled
- `result.enrichment_report.auto_tags` / `auto_keywords`: deterministic derived values

### user.*

```json
{
  "jsonrpc": "2.0",
  "id": "user-profile-1",
  "method": "user.get_profile",
  "params": {}
}
```

---

## Typed Client Helpers

### Python (`McpSseClient`)

```python
from app.mcp.client import McpSseClient

# Session auth
with McpSseClient(base_url="http://localhost:8000") as client:
    client.login_with_password(username="admin", password="admin123")
    init_result = client.call_tool(method="initialize", params={}).require_result()
    tools_result = client.call_tool(method="tools/list", params={}).require_result()
    print("tool count:", len(tools_result["tools"]))
```

```python
# Bearer auth
from app.mcp.client import McpSseClient

with McpSseClient(
    base_url="http://localhost:8000",
    bearer_token="engram_mcp_<token_id_hex>_<secret>",
) as client:
    tools = client.call_tool(method="tools/list", params={}).require_result()
    print("visible tools:", [tool["name"] for tool in tools["tools"]])
```

### TypeScript (`streamMcpCall`)

- Source: `web/src/api/mcpClient.ts`
- Provides `streamMcpCall`, frame parsers, and final-frame helpers
