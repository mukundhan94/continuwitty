# CP118 - Phase 4 MCP Chat Unpin-Engram Mutation Baseline

Date: 2026-02-26  
Status: Completed

## Summary

Migrated `chat.unpin_engram` MCP dispatch path in Go compatibility mode.

Behavior now supports:

- direct method path (`chat.unpin_engram`)
- `tools/call` path (`chat_unpin_engram`)
- required `session_id` and `engram_id` UUID validation
- deterministic unpin response payload (`{"removed": true}`)
- not-found mapping with `-32602` and `{"status_code": 404, "detail": "Pinned engram not found for session"}`.

## Files

- Updated: `cmd/api/main.go`
- Added: `cmd/api/mcp_unpin_engram_adapter.go`
- Updated: `internal/mcp/compatibility_service.go`
- Updated: `internal/mcp/compatibility_dispatch_chat_mutation_handlers.go`
- Added: `internal/mcp/compatibility_dispatch_chat_unpin_support.go`
- Added: `internal/mcp/compatibility_service_chat_unpin_engram_test.go`
- Updated: `checkpoint-go-migration.md`
- Updated: `checkpoint.md`

## Tests Run

Full:
- `go test ./... -count=1`

## Code Health

- `cmd/api/main.go`: `10.0`
- `cmd/api/mcp_unpin_engram_adapter.go`: `10.0`
- `internal/mcp/compatibility_service.go`: `10.0`
- `internal/mcp/compatibility_dispatch_chat_mutation_handlers.go`: `10.0`
- `internal/mcp/compatibility_dispatch_chat_unpin_support.go`: `10.0`
- `internal/mcp/compatibility_service_chat_unpin_engram_test.go`: `10.0`

## Safeguard

- `pre_commit_code_health_safeguard`: `quality_gates=passed`, no findings.
