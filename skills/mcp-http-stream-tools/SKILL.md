---
name: mcp-http-stream-tools
description: Use this skill when building or changing MCP tool handlers, JSON-RPC contracts, and SSE transport behavior.
---

# MCP HTTP Stream Tools

## Use This Skill When
- Adding MCP server endpoints.
- Adding new MCP tools for chat/engram workflows.
- Debugging JSON-RPC over SSE behavior.

## Protocol Rules
- Use JSON-RPC-shaped request and response frames.
- Stream events as SSE `data:` lines containing JSON payloads.
- Include deterministic `id` correlation for every tool call.
- Return structured errors with code/message/data.
- Emit progress notifications as `mcp.event` frames for long-running tools.
- Keep auth dual-path support:
  - bearer token via `Authorization: Bearer engram_mcp_<token_id_hex>_<secret>`
  - session-cookie fallback for local/UI compatibility
- Keep interoperability paths for external MCP clients:
  - `initialize`
  - `tools/list`
  - `tools/call`
- Keep direct tool methods for backward compatibility (`chat.*`, `engram.*`, `user.*`).
- For token-authenticated calls, enforce in service layer before dispatch:
  - scope check (`read` / `write`)
  - optional per-tool allowlist
  - optional project allowlist

## Module Layout (Current)
- `api/app/mcp/api.py`: HTTP transport and SSE writer.
- `api/app/mcp/auth.py`: bearer/session actor resolution and token validation.
- `api/app/mcp/service.py`: tool dispatch and JSON-RPC frame generation.
- `api/app/mcp/errors.py`: RPC error object and codes.
- `api/app/mcp/client.py`: typed Python MCP client helper for JSON-RPC/SSE transport.
- `api/app/mcp_tokens/*`: token issue/hash/parse/revoke persistence and auth context helpers.
- `web/src/api/mcpClient.ts`: typed TypeScript MCP client helper for JSON-RPC/SSE transport.

## Tool Implementation Sequence
1. Define input/output schema.
2. Validate auth and visibility.
3. Call service/repository layer.
4. Emit success frame or error frame.
5. Add test coverage for success, invalid params, unauthorized, forbidden.
6. If `tools/list` schemas change, update typed clients and docs in the same phase.
7. If tool scope/project semantics change, update token policy tests and docs in the same phase.

## Initial Tool Set
- `chat.create_session`
- `chat.list_sessions`
- `chat.get_session`
- `chat.list_messages`
- `chat.get_lifecycle_policy`
- `chat.update_lifecycle_policy`
- `chat.list_timeline`
- `chat.send_message`
- `chat.list_pinned_engrams`
- `chat.pin_engram`
- `chat.unpin_engram`
- `chat.list_pinned_documents`
- `chat.pin_document`
- `chat.unpin_document`
- `chat.list_project_documents`
- `chat.save_as_engram`
- `chat.continue_session`
- `engram.create`
- `engram.create_from_conversation`
- `engram.query`
- `engram.rehydrate`
- `engram.pin_to_session`
- `user.get_profile`
- `user.list_projects`

## Contract Test Minimum
- `api/tests/test_mcp_api_integration.py`: transport + tool success/error contracts.
- `api/tests/test_mcp_client.py`: Python typed client frame parsing and protocol handling.
- `api/tests/test_mcp_token_service.py`: token format/hash/expiry/revocation behavior.
- `api/tests/test_mcp_token_api_integration.py`: token lifecycle APIs and role restrictions.
- `web/src/api/mcpClient.test.ts`: TypeScript typed client frame parsing and SSE handling.

## Future Consideration (Do Not Auto-Migrate)
- Evaluate `FastMCP` as an adapter/facade for external MCP ecosystem clients.
- Keep current FastAPI/MCP service as source of truth unless an explicit migration phase is approved.
- If evaluated:
  1. Pilot adapter layer first (no API/tool contract breakage).
  2. Measure implementation simplification vs. migration risk.
  3. Preserve existing auth/session and visibility semantics.
