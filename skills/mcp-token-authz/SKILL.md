---
name: mcp-token-authz
description: Use this skill when implementing or modifying MCP personal access token issuance, bearer auth, and scoped tool authorization.
---

# MCP Token Authz

## Use This Skill When
- Adding or changing MCP token APIs, storage, or admin UI flows.
- Debugging bearer-token failures on `/api/v1/mcp/stream`.
- Updating read/write scope, tool allowlist, or project allowlist enforcement.
- Updating OAuth compatibility flows that mint MCP-compatible bearer tokens.

## Core Rules
- Never store plaintext token secrets at rest.
- Generate token as `engram_mcp_<token_id_hex>_<secret>` and store only hashed secret + hint.
- Keep token auth additive: bearer preferred, session-cookie fallback retained.
- Keep OAuth compatibility additive: `.well-known` metadata and `/oauth/*` flows must reuse the same MCP authorization guards.
- Enforce authorization before MCP dispatch.
- Return structured JSON-RPC errors for scope/allowlist/project policy violations.
- Keep `initialize` and `tools/list` usable for token-authenticated clients.

## Scope Model
- `read`: read-only tools (`chat.list_*`, `chat.get_*`, `engram.query`, `engram.rehydrate`, `user.*`).
- `write`: includes all read tools plus mutating tools (`chat.create_session`, `chat.send_message`, `engram.create`, etc.).
- Empty `allowed_tools`: allow all tools in scope.
- Non-empty `allowed_tools`: allow only listed tools (canonical aliases included).

## Project Allowlist Model
- Empty `allowed_project_ids`: no token-level project restriction.
- Non-empty `allowed_project_ids`: resolve project from request (`project_id` or derived entity ID) and enforce policy.
- For optional-project tools, auto-fill only when policy resolves a single project; otherwise return invalid params.

## Files to Update Together
- `internal/models/mcp_token*.go`
- `internal/repository/mcp_token*.go`
- `internal/mcptokens/service*.go`
- `internal/mcp/auth*.go`
- `internal/mcp/token_authorization_policy.go`
- `internal/mcp/token_project_scope_policy.go`
- `internal/mcp/compatibility_service*.go`
- `internal/api/session_mcp_tokens.go`
- `internal/api/mcp_stream*.go`
- `internal/oauth/*.go`
- `internal/api/oauth_*.go`
- `cmd/api/main.go` + `cmd/api/mcp_*_adapter.go`
- `db/init/001_schema.sql`
- `web/src/api/mcpTokens.ts`
- `web/src/components/AdminMcpTokenPanel.tsx`

## Required Tests
- `internal/mcptokens/service_test.go`
- `internal/repository/mcp_token_test.go`
- MCP token policy tests in `internal/mcp/*token*test.go`
- MCP stream/token route tests in `internal/api/mcp_stream*_test.go`
- OAuth tests in `internal/oauth/*_test.go` + `internal/api/oauth_*_test.go`
- Web tests: `web/src/components/AdminMcpTokenPanel.test.tsx`, `web/src/api/mcpTokens.test.ts`
- Acceptance: `acceptance-tests/features/mcp-token-auth-mock.feature`, `acceptance-tests/features/admin-mcp-token-ui.feature`

## Validation Commands
- `go test ./internal/mcptokens ./internal/mcp ./internal/api ./internal/oauth -count=1`
- `go test ./... -count=1`
- `make acceptance-bddgen`
- `make acceptance-typecheck`
- `make acceptance-test-mock`
