# CP140 - Phase 4 MCP Token Project-Scope Dispatch Parity

Date: 2026-02-27  
Status: Completed

## Summary

Added MCP token project-scope dispatch parity in Go compatibility mode for project-scoped tools.

Behavior now supports:

- token-driven `project_id` autofill when the token is constrained to a single allowed project
- explicit project selection guard for multi-project token policies
  - returns `-32602` with `{"missing":"project_id","reason":"token_has_multiple_allowed_projects"}`
- allowlist enforcement for explicit `project_id`
  - returns `-32003` with denied `project_id` payload when outside token policy
- dispatch-level token project normalization for both direct and `tools/call` paths

## Files

- Updated: `internal/mcp/compatibility_service.go`
- Updated: `internal/mcp/catalog.go`
- Added: `internal/mcp/token_authorization_policy.go`
- Updated: `internal/mcp/compatibility_service_chat_sessions_test.go`
- Updated: `checkpoint-go-migration.md`
- Added: `checkpoint-go-migration-cp140.md`
- Updated: `checkpoint.md`

## Tests Run

Targeted:
- `go test ./internal/mcp ./cmd/api ./internal/api -count=1`

Full:
- `go test ./... -count=1`

## Code Health

- `internal/mcp/catalog.go`: `9.68`
- `internal/mcp/token_authorization_policy.go`: `9.68`
- `internal/mcp/compatibility_service.go`: `10.0`
- `internal/mcp/compatibility_service_chat_sessions_test.go`: `10.0`

## Safeguard

- `pre_commit_code_health_safeguard`: `quality_gates=passed`, no findings.
