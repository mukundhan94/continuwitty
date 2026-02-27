# CP121 - Phase 4 MCP Chat Project-Document Query Baseline

Date: 2026-02-26  
Status: Completed

## Summary

Migrated `chat.list_project_documents` MCP dispatch path in Go compatibility mode.

Behavior now supports:

- direct method path (`chat.list_project_documents`)
- `tools/call` path (`chat_list_project_documents`)
- optional `project_id` filter
- paging defaults (`limit=200`, `offset=0`)
- invalid paging mapping to `-32602`.

## Files

- Updated: `cmd/api/main.go`
- Added: `cmd/api/mcp_project_document_list_adapter.go`
- Updated: `internal/mcp/compatibility_service.go`
- Updated: `internal/mcp/compatibility_dispatch.go`
- Updated: `internal/mcp/compatibility_dispatch_chat_collection_handlers.go`
- Added: `internal/mcp/compatibility_dispatch_chat_project_documents_support.go`
- Added: `internal/mcp/compatibility_service_chat_project_documents_test.go`
- Added: `internal/mcp/compatibility_service_chat_project_documents_support_test.go`
- Updated: `checkpoint-go-migration.md`
- Updated: `checkpoint.md`

## Tests Run

Full:
- `go test ./... -count=1`

## Code Health

- `cmd/api/main.go`: `10.0`
- `cmd/api/mcp_project_document_list_adapter.go`: `10.0`
- `internal/mcp/compatibility_service.go`: `10.0`
- `internal/mcp/compatibility_dispatch.go`: `10.0`
- `internal/mcp/compatibility_dispatch_chat_collection_handlers.go`: `10.0`
- `internal/mcp/compatibility_dispatch_chat_project_documents_support.go`: `10.0`
- `internal/mcp/compatibility_service_chat_project_documents_test.go`: `10.0`
- `internal/mcp/compatibility_service_chat_project_documents_support_test.go`: `10.0`

## Safeguard

- `pre_commit_code_health_safeguard`: `quality_gates=passed`, no findings.
