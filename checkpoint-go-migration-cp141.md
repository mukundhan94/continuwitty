# CP141 - Phase 4 MCP Token Project-Scope Parity Extension

Date: 2026-02-27  
Status: Completed

## Summary

Extended MCP token project-scope enforcement to include session-scoped and engram-scoped tool calls.

Behavior now supports:

- resolving token project scope from `session_id` for session-scoped tools
- resolving token project scope from `engram_id` for engram-scoped tools
- denying tool calls when resolved project is outside token `allowed_project_ids`
- preserving existing token normalization/allowlist behavior for explicit `project_id` tools

## Files

- Updated: `internal/mcp/compatibility_service.go`
- Added: `internal/mcp/token_project_scope_policy.go`
- Added: `internal/mcp/compatibility_service_token_project_scope_test.go`
- Updated: `checkpoint-go-migration.md`
- Added: `checkpoint-go-migration-cp141.md`
- Updated: `checkpoint.md`

## Tests Run

Targeted:
- `go test ./internal/mcp ./cmd/api ./internal/api -count=1`

Full:
- `go test ./... -count=1`

## Code Health

- `internal/mcp/compatibility_service.go`: `10.0`
- `internal/mcp/token_project_scope_policy.go`: `9.68`
- `internal/mcp/compatibility_service_token_project_scope_test.go`: `10.0`

## Safeguard

- `pre_commit_code_health_safeguard`: `quality_gates=passed`, no findings.
