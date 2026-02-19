---
name: mcp-token-authz
description: Use this skill when implementing or modifying MCP personal access token issuance, bearer auth, and scoped tool authorization.
---

# MCP Token Authz

## Use This Skill When
- Adding or changing MCP token APIs, storage, or UI management flows.
- Debugging bearer-token failures on `/api/v1/mcp/stream`.
- Updating read/write scope, tool allowlist, or project allowlist authorization behavior.

## Core Rules
- Never store plaintext token secrets at rest.
- Generate token as `engram_mcp_<token_id_hex>_<secret>` and store only hashed secret + hint.
- Keep token auth additive: bearer-token preferred, session-cookie fallback retained.
- Enforce authorization in MCP service before tool dispatch.
- Return JSON-RPC `-32003` for scope/allowlist/project policy violations.
- Keep `initialize` and `tools/list` usable for token-authenticated clients.

## Scope Model
- `read`: read-only tools (`chat.list_*`, `chat.get_*`, `engram.query`, `engram.rehydrate`, `user.*`).
- `write`: includes all read tools + write tools (`chat.create_session`, `chat.send_message`, `engram.create`, etc.).
- If `allowed_tools` is empty: allow all tools in scope.
- If `allowed_tools` is non-empty: allow only listed tools (canonical aliases included).

## Project Allowlist Model
- If `allowed_project_ids` is empty: no token-level project restriction.
- If non-empty: resolve project from request (`project_id` or derived from `session_id`/`engram_id`) and enforce membership.
- For optional-project tools (`engram.query`, `chat.list_sessions`, `chat.list_project_documents`):
  - if token has one allowed project and caller omits `project_id`, auto-fill it.
  - if token has multiple allowed projects and caller omits `project_id`, return invalid params.

## Files to Update Together
- `api/app/mcp_tokens/models.py`
- `api/app/mcp_tokens/repository.py`
- `api/app/mcp_tokens/service.py`
- `api/app/mcp/auth.py`
- `api/app/mcp/service.py`
- `api/app/main.py`
- `api/app/models.py`
- `api/app/templates/admin.html`
- `db/init/001_schema.sql`
- `web/src/api/mcpTokens.ts`
- `web/src/components/AdminMcpTokenPanel.tsx`
- `web/src/App.tsx`

## Required Tests
- `api/tests/test_mcp_token_service.py`
- `api/tests/test_mcp_token_api_integration.py`
- `api/tests/test_mcp_api_integration.py`
- `api/tests/test_admin_mcp_tokens_ui.py`
- `acceptance-tests/features/mcp-token-auth-mock.feature`
- `web/src/components/AdminMcpTokenPanel.test.tsx`

## Validation Commands
- `make -C /Users/mukundhan/Projects/engram check`
- `make -C /Users/mukundhan/Projects/engram acceptance-bddgen`
- `make -C /Users/mukundhan/Projects/engram acceptance-typecheck`
- `make -C /Users/mukundhan/Projects/engram acceptance-test-mock`
