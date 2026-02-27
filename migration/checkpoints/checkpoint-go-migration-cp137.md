# CP137 - Phase 4 MCP Project Transfer Baseline

Date: 2026-02-27  
Status: Completed

## Summary

Migrated MCP project transfer dispatch in Go compatibility mode:

- `project.export_bundle`
- `project.import_bundle`

Behavior now supports:

- direct method paths (dotted names)
- `tools/call` alias paths (underscored names)
- runtime export/import service wiring through Go compatibility dependencies
- request validation parity for:
  - required `project_id`
  - optional `collection_ids` UUID list
  - optional `include_embeddings` boolean
  - optional `conflict_policy` (`skip`/`overwrite`/`rename`, default `skip`)
  - import `bundle` object or `bundle_json` string payloads
- error mapping parity for export/import project-transfer failures:
  - `404`-class project/collection lookup errors
  - `422`-class import payload decoding/validation errors

## Files

- Updated: `cmd/api/main.go`
- Added: `cmd/api/mcp_project_transfer_adapter.go`
- Updated: `internal/mcp/compatibility_service.go`
- Updated: `internal/mcp/compatibility_dispatch.go`
- Added: `internal/mcp/compatibility_dispatch_project_transfer_handlers.go`
- Added: `internal/mcp/compatibility_dispatch_project_transfer_export_support.go`
- Added: `internal/mcp/compatibility_dispatch_project_transfer_import_support.go`
- Added: `internal/mcp/compatibility_dispatch_project_transfer_error_support.go`
- Added: `internal/mcp/compatibility_service_project_export_bundle_test.go`
- Added: `internal/mcp/compatibility_service_project_import_bundle_test.go`
- Added: `internal/mcp/compatibility_dispatch_catalog_coverage_test.go`
- Updated: `checkpoint-go-migration.md`
- Added: `checkpoint-go-migration-cp137.md`
- Updated: `checkpoint.md`

## Tests Run

Targeted:
- `go test ./internal/mcp ./cmd/api -count=1`

Full:
- `go test ./... -count=1`

## Code Health

- `cmd/api/main.go`: `10.0`
- `cmd/api/mcp_project_transfer_adapter.go`: `10.0`
- `internal/mcp/compatibility_dispatch.go`: `10.0`
- `internal/mcp/compatibility_service.go`: `10.0`
- `internal/mcp/compatibility_dispatch_project_transfer_handlers.go`: `10.0`
- `internal/mcp/compatibility_dispatch_project_transfer_export_support.go`: `10.0`
- `internal/mcp/compatibility_dispatch_project_transfer_import_support.go`: `10.0`
- `internal/mcp/compatibility_dispatch_project_transfer_error_support.go`: `10.0`
- `internal/mcp/compatibility_service_project_export_bundle_test.go`: `10.0`
- `internal/mcp/compatibility_service_project_import_bundle_test.go`: `10.0`
- `internal/mcp/compatibility_dispatch_catalog_coverage_test.go`: `10.0`

## Safeguard

- `pre_commit_code_health_safeguard`: `quality_gates=passed`, no findings.
