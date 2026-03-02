# CP138 - Phase 4 MCP Chat Send-Message Stream Parity

Date: 2026-02-27  
Status: Completed

## Summary

Migrated MCP `chat.send_message` streaming behavior in Go compatibility mode.

Behavior now supports:

- direct tool method stream paths (`chat.send_message` and underscore alias)
- `tools/call` stream paths for `chat_send_message` when `stream` is truthy
- `mcp.event` frame emission with parity payload shape:
  - `{"id": <request_id>, "tool": <tool_name>, "event": <event_name>, "data": <event_payload>}`
- terminal stream response parity:
  - success payload wraps `{"message": ...}` from `done` event payload
  - `error` stream events map to `-32020`
  - missing `done` completion maps to `-32021`
- non-stream fallback preservation:
  - direct `stream=false` path remains non-stream
  - `tools/call` with `stream=false` remains on non-stream dispatch

## Files

- Updated: `cmd/api/main.go`
- Updated: `cmd/api/mcp_message_send_adapter.go`
- Updated: `internal/mcp/compatibility_service.go`
- Added: `internal/mcp/compatibility_stream_chat_send_message_support.go`
- Added: `internal/mcp/compatibility_service_chat_send_message_stream_test.go`
- Updated: `checkpoint-go-migration.md`
- Added: `checkpoint-go-migration-cp138.md`
- Updated: `checkpoint.md`

## Tests Run

Targeted:
- `go test ./internal/mcp ./cmd/api -count=1`

Full:
- `go test ./... -count=1`

## Code Health

- `cmd/api/main.go`: `10.0`
- `cmd/api/mcp_message_send_adapter.go`: `10.0`
- `internal/mcp/compatibility_service.go`: `10.0`
- `internal/mcp/compatibility_stream_chat_send_message_support.go`: `10.0`
- `internal/mcp/compatibility_service_chat_send_message_stream_test.go`: `10.0`

## Safeguard

- `pre_commit_code_health_safeguard`: `quality_gates=passed`, no findings.
