# CP109 - Phase 4 MCP User-Projects Dispatch Baseline

Date: 2026-02-26  
Status: Completed

## Summary

Migrated `user.list_projects` MCP dispatch path in Go compatibility mode.

Behavior now supports:

- direct method path (`user.list_projects`)
- `tools/call` path (`user_list_projects`)
- session-derived project IDs (deduplicated, non-empty, sorted ascending)
- runtime wiring through a session-list adapter backed by repository session queries.

## Files

- Updated: `cmd/api/main.go`
- Added: `cmd/api/mcp_session_list_adapter.go`
- Updated: `internal/mcp/compatibility_service.go`
- Updated: `internal/mcp/compatibility_dispatch.go`
- Added: `internal/mcp/compatibility_service_user_projects_test.go`
- Updated: `checkpoint-go-migration.md`
- Updated: `checkpoint.md`

## Tests Run

Full:
- `go test ./... -count=1`

## Code Health

- `cmd/api/main.go`: `10.0`
- `cmd/api/mcp_session_list_adapter.go`: `10.0`
- `internal/mcp/compatibility_service.go`: `10.0`
- `internal/mcp/compatibility_dispatch.go`: `9.68`
- `internal/mcp/compatibility_service_user_projects_test.go`: `10.0`

## Safeguard

- `pre_commit_code_health_safeguard`: `quality_gates=passed`, no findings.
