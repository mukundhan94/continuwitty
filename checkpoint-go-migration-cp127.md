# CP127 - Phase 4 MCP Chat Send-Message Baseline

Date: 2026-02-26  
Status: Completed

## Summary

Migrated `chat.send_message` MCP dispatch path in Go compatibility mode.

Behavior now supports:

- direct method path (`chat.send_message`)
- `tools/call` path (`chat_send_message`)
- required `session_id`
- `content_text` default (`""`) when omitted
- structured chat/provider error mapping parity (`-32010` / `-32020`) and internal-error parity (`-32603`).

## Files

- Updated: `cmd/api/main.go`
- Added: `cmd/api/mcp_message_send_adapter.go`
- Updated: `internal/mcp/compatibility_service.go`
- Updated: `internal/mcp/compatibility_dispatch_chat_primary_handlers.go`
- Added: `internal/mcp/compatibility_dispatch_chat_send_message_support.go`
- Added: `internal/mcp/compatibility_service_chat_send_message_test.go`
- Updated: `checkpoint-go-migration.md`
- Updated: `checkpoint.md`

## Tests Run

Full:
- `go test ./... -count=1`

## Code Health

- `cmd/api/main.go`: `10.0`
- `cmd/api/mcp_message_send_adapter.go`: `10.0`
- `internal/mcp/compatibility_service.go`: `10.0`
- `internal/mcp/compatibility_dispatch_chat_primary_handlers.go`: `10.0`
- `internal/mcp/compatibility_dispatch_chat_send_message_support.go`: `10.0`
- `internal/mcp/compatibility_service_chat_send_message_test.go`: `10.0`

## Safeguard

- `pre_commit_code_health_safeguard`: `quality_gates=passed`, no findings.
