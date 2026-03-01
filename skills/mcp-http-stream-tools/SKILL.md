---
name: mcp-http-stream-tools
description: Use this skill when building or changing MCP tool handlers, JSON-RPC contracts, and SSE transport behavior.
---

# MCP HTTP Stream Tools

## Use This Skill When
- Adding MCP server endpoints.
- Adding or changing MCP tools for chat/engram/project flows.
- Debugging JSON-RPC over SSE behavior.

## Protocol Rules
- Use JSON-RPC shaped request/response frames.
- Stream SSE `data:` lines containing JSON payloads.
- Keep deterministic `id` correlation for tool calls.
- Return structured errors (`code`, `message`, `data`).
- Emit `mcp.event` progress frames for long-running operations.
- Keep auth dual-path support:
  - bearer token via `Authorization: Bearer engram_mcp_<token_id_hex>_<secret>`
  - session-cookie fallback for local/UI compatibility
- Preserve interoperability methods:
  - `initialize`
  - `tools/list`
  - `tools/call`
- Keep canonical dotted tool names as source-of-truth; expose underscore aliases in `tools/list` and accept both forms in `tools/call`.

## Organization Contracts
- Keep project/memory-admin contracts stable:
  - `project.*` (`list`, `create`, `get_default`, `set_default`)
  - `engram.*` lifecycle (`list/get/update/move_project/delete/restore`)
  - `engram.collection_*` lifecycle
  - `chat.delete_session`, `chat.restore_session`
- For token-authenticated calls, enforce scope/allowlist/project policy before tool dispatch.
- For write tools that allow omitted `project_id`, resolve via project policy and surface `resolved_project_id` + `used_default_project` where contracted.

## Module Layout (Current Go Runtime)
- `internal/api/mcp_stream*.go`: HTTP/SSE transport + auth gates.
- `internal/mcp/compatibility_dispatch*.go`: tool dispatch, input parsing, routing.
- `internal/mcp/compatibility_service.go`: service-layer tool execution.
- `internal/mcp/auth*.go`: bearer/session auth context and errors.
- `internal/mcp/token_*policy*.go`: token scope/project policy enforcement.
- `internal/mcptokens/service.go`: token parsing/validation helpers.
- `cmd/api/mcp_*_adapter.go`: runtime dependency adapters.
- `web/src/api/mcpClient.ts`: typed TypeScript MCP client.

## Tool Implementation Sequence
1. Define input/output contracts.
2. Validate auth and visibility.
3. Dispatch through service/repository domain helpers.
4. Emit success or structured error frame.
5. Add tests for success, invalid params, unauthorized, forbidden.
6. Update `tools/list` metadata + typed clients/docs in the same change.

## Contract Test Minimum
- MCP service/dispatch tests under `internal/mcp/compatibility_service*_test.go` and `internal/mcp/compatibility_dispatch*_test.go`.
- MCP transport/token tests under `internal/api/mcp_stream*_test.go`.
- Token policy tests under `internal/mcp/*token*test.go` and `internal/mcptokens/service_test.go`.
- Frontend transport tests: `web/src/api/mcpClient.test.ts`.

## Validation
- `go test ./internal/mcp ./internal/api ./internal/mcptokens -count=1`
- `go test ./... -count=1`
- `make acceptance-test-mock` when MCP contracts/UI flows are impacted.
