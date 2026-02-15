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

## Module Layout (Current)
- `api/app/mcp/api.py`: HTTP transport and SSE writer.
- `api/app/mcp/service.py`: tool dispatch and JSON-RPC frame generation.
- `api/app/mcp/errors.py`: RPC error object and codes.

## Tool Implementation Sequence
1. Define input/output schema.
2. Validate auth and visibility.
3. Call service/repository layer.
4. Emit success frame or error frame.
5. Add test coverage for success, invalid params, unauthorized, forbidden.

## Initial Tool Set
- `chat.create_session`
- `chat.list_sessions`
- `chat.get_session`
- `chat.send_message`
- `chat.save_as_engram`
- `engram.create`
- `engram.query`
- `engram.rehydrate`
- `engram.pin_to_session`
- `user.get_profile`
- `user.list_projects`
