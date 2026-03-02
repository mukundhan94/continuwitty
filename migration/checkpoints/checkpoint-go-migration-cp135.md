# CP135 - Phase 4 MCP Engram Move-Project Baseline

Date: 2026-02-26  
Status: Completed

## Summary

Migrated `engram.move_project` MCP dispatch path in Go compatibility mode.

Behavior now supports:

- direct method path (`engram.move_project`)
- `tools/call` path (`engram_move_project`)
- validation parity for:
  - required `engram_id` UUID
  - optional `target_project_id`, `reason`, and `expected_updated_at`
- error mapping parity for:
  - not found / inaccessible engram: `-32004` with `engram_id`
  - stale move: `-32602` with `status_code=409`
  - project resolution errors mapped to validation status payloads (`404`/`422`)

## Files

- Updated: `cmd/api/main.go`
- Added: `cmd/api/mcp_engram_admin_move_adapter.go`
- Updated: `internal/mcp/compatibility_service.go`
- Updated: `internal/mcp/compatibility_dispatch_engram_primary_handlers.go`
- Added: `internal/mcp/compatibility_dispatch_engram_move_support.go`
- Added: `internal/mcp/compatibility_service_engram_move_project_test.go`
- Updated: `checkpoint-go-migration.md`
- Added: `checkpoint-go-migration-cp135.md`
- Updated: `checkpoint.md`

## Tests Run

Targeted:
- `go test ./internal/mcp ./cmd/api -count=1`

Full:
- `go test ./... -count=1`

## Code Health

- `cmd/api/main.go`: `10.0`
- `cmd/api/mcp_engram_admin_move_adapter.go`: `10.0`
- `internal/mcp/compatibility_service.go`: `10.0`
- `internal/mcp/compatibility_dispatch_engram_primary_handlers.go`: `10.0`
- `internal/mcp/compatibility_dispatch_engram_move_support.go`: `10.0`
- `internal/mcp/compatibility_service_engram_move_project_test.go`: `9.51`

## Safeguard

- `pre_commit_code_health_safeguard`: `quality_gates=passed`, no findings.
