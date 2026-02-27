# CP136 - Phase 4 MCP Engram Collection Mutation Baseline

Date: 2026-02-27  
Status: Completed

## Summary

Migrated MCP engram collection mutation dispatch in Go compatibility mode:

- `engram.collection_create`
- `engram.collection_update`
- `engram.collection_delete`
- `engram.collection_add_items`
- `engram.collection_remove_items`

Behavior now supports:

- direct method paths (dotted names)
- `tools/call` alias paths (underscored names)
- runtime owner/admin visibility parity for collection mutations
- project resolution/default-project parity for collection create
- error mapping parity for:
  - not found/inaccessible collection: `-32004` + `collection_id`
  - stale/duplicate updates: `-32602` with `status_code=409`
  - project resolution failures on create (`404`/`422` + invalid `project_id`)

## Files

- Updated: `cmd/api/main.go`
- Added: `cmd/api/mcp_engram_admin_collection_create_adapter.go`
- Added: `cmd/api/mcp_engram_admin_collection_mutation_support.go`
- Added: `cmd/api/mcp_engram_admin_collection_update_adapter.go`
- Added: `cmd/api/mcp_engram_admin_collection_delete_adapter.go`
- Added: `cmd/api/mcp_engram_admin_collection_add_items_adapter.go`
- Added: `cmd/api/mcp_engram_admin_collection_remove_item_adapter.go`
- Updated: `internal/mcp/compatibility_service.go`
- Updated: `internal/mcp/compatibility_dispatch_engram_primary_handlers.go`
- Updated: `internal/mcp/compatibility_dispatch_engram_mutation_handlers.go`
- Added: `internal/mcp/compatibility_dispatch_engram_collection_create_support.go`
- Added: `internal/mcp/compatibility_dispatch_engram_collection_mutation_dispatch_support.go`
- Added: `internal/mcp/compatibility_dispatch_engram_collection_update_support.go`
- Added: `internal/mcp/compatibility_dispatch_engram_collection_delete_support.go`
- Added: `internal/mcp/compatibility_dispatch_engram_collection_add_items_support.go`
- Added: `internal/mcp/compatibility_dispatch_engram_collection_remove_item_support.go`
- Added: `internal/mcp/compatibility_service_engram_collection_create_test.go`
- Added: `internal/mcp/compatibility_service_engram_collection_update_test.go`
- Added: `internal/mcp/compatibility_service_engram_collection_delete_test.go`
- Added: `internal/mcp/compatibility_service_engram_collection_add_items_test.go`
- Added: `internal/mcp/compatibility_service_engram_collection_remove_item_test.go`
- Updated: `checkpoint-go-migration.md`
- Added: `checkpoint-go-migration-cp136.md`
- Updated: `checkpoint.md`

## Tests Run

Targeted:
- `go test ./internal/mcp ./cmd/api -count=1`

Full:
- `go test ./... -count=1`

## Code Health

- All checkpoint-touched Go files: `10.0`

## Safeguard

- `pre_commit_code_health_safeguard`: `quality_gates=passed`, no findings.
