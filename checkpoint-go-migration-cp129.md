# CP129 - Phase 4 MCP Engram Read-Access Baseline

Date: 2026-02-26  
Status: Completed

## Summary

Migrated read-access engram MCP dispatch paths in Go compatibility mode.

Behavior now supports:

- `engram.list` (direct + `tools/call` as `engram_list`)
- `engram.get` (direct + `tools/call` as `engram_get`)
- `engram.collection_list` (direct + `tools/call` as `engram_collection_list`)
- owner/admin visibility parity in runtime adapter:
  - non-admin list + collection-list requests are owner-scoped
  - non-admin `engram.get` for non-owned records resolves as not-found
- defaults and validation parity for filters/paging and include-deleted fields.

## Files

- Updated: `cmd/api/main.go`
- Added: `cmd/api/mcp_engram_admin_adapter.go`
- Updated: `internal/mcp/compatibility_dispatch.go`
- Added: `internal/mcp/compatibility_dispatch_engram_registration.go`
- Added: `internal/mcp/compatibility_dispatch_engram_read_handlers.go`
- Added: `internal/mcp/compatibility_dispatch_engram_read_support.go`
- Updated: `internal/mcp/compatibility_service.go`
- Added: `internal/mcp/compatibility_service_engram_list_test.go`
- Added: `internal/mcp/compatibility_service_engram_get_test.go`
- Added: `internal/mcp/compatibility_service_engram_collection_list_test.go`
- Updated: `checkpoint-go-migration.md`
- Updated: `checkpoint.md`

## Tests Run

Targeted:
- `go test ./internal/mcp ./cmd/api -count=1`

Full:
- `go test ./... -count=1`

## Code Health

- `cmd/api/main.go`: `10.0`
- `cmd/api/mcp_engram_admin_adapter.go`: `10.0`
- `internal/mcp/compatibility_service.go`: `10.0`
- `internal/mcp/compatibility_dispatch.go`: `10.0`
- `internal/mcp/compatibility_dispatch_engram_registration.go`: `10.0`
- `internal/mcp/compatibility_dispatch_engram_read_handlers.go`: `10.0`
- `internal/mcp/compatibility_dispatch_engram_read_support.go`: `10.0`
- `internal/mcp/compatibility_service_engram_list_test.go`: `9.68`
- `internal/mcp/compatibility_service_engram_get_test.go`: `10.0`
- `internal/mcp/compatibility_service_engram_collection_list_test.go`: `9.55`

## Safeguard

- `pre_commit_code_health_safeguard`: `quality_gates=passed`, no findings.
