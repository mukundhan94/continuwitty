# Plan.Next - MCP Streaming + Multi-Provider Chat + Engram Continuity

## Objective
Implement a complete chat and memory workflow on top of the current Engram Vault:
- Multi-provider chat (OpenAI, Anthropic, Bedrock)
- Chat session persistence and continuity
- Save/rehydrate/pin engrams across sessions
- MCP JSON-RPC over HTTP SSE
- Agent-friendly contribution standards (`AGENT.md` + `skills/`)

## Working Defaults
- Visibility model: configurable with `private` default and optional `project` scope.
- Chat persistence: full transcript + metadata + save-as-engram snapshots.
- Save policy: manual save as engram, with optional autosave.
- UI stack: Vite + React + TypeScript.
- MCP transport: JSON-RPC framed events over SSE.

## Phase Breakdown

### Phase 0 - Planning and Agent Foundation
- Add this file as active roadmap (`Plan.Next.md`).
- Add `AGENT.md` with architecture and maintenance rules.
- Add `skills/` with operator and contributor skills.
- Update README with plan status and skill catalog.

### Phase 1 - Schema and Repository Layer
- Extend engram ownership and visibility fields.
- Add chat session/message/pinned-engram tables.
- Add repository modules for sessions, messages, pinning, and visibility-safe retrieval.
- Preserve compatibility for existing users/engrams.

### Phase 2 - Provider Adapter Layer
- Implement provider abstraction contract.
- Add OpenAI, Anthropic, Bedrock adapters.
- Add provider registry and config validation.

### Phase 3 - Chat APIs and Continuity
- Add chat session/message CRUD and streaming endpoints.
- Add pin/unpin engram endpoints.
- Add save-session-as-engram endpoint.
- Add continue-session workflow that carries context safely.

### Phase 4 - MCP HTTP Streaming
- Add MCP endpoint with JSON-RPC over SSE.
- Implement core tool handlers for chat + engram operations.
- Enforce auth/visibility/role checks in MCP tools.

### Phase 5 - React Chat UI
- Create `web/` app (Vite + React + TS).
- Build session list, chat view, provider selector, pinned engram panel.
- Add save-as-engram and continue-in-new-chat workflows.

### Phase 6 - Hardening and Validation
- Add unit, integration, and UI-level tests.
- Extend eval coverage with continuity/citation scenarios.
- Ensure `make check` is green.
- Complete README operational docs for newcomers.

## New Public API Surface
- `POST /api/v1/chat/sessions`
- `GET /api/v1/chat/sessions`
- `GET /api/v1/chat/sessions/{session_id}`
- `PATCH /api/v1/chat/sessions/{session_id}`
- `GET /api/v1/chat/sessions/{session_id}/messages`
- `POST /api/v1/chat/sessions/{session_id}/messages`
- `POST /api/v1/chat/sessions/{session_id}/messages/stream`
- `POST /api/v1/chat/sessions/{session_id}/engrams/pin`
- `DELETE /api/v1/chat/sessions/{session_id}/engrams/{engram_id}`
- `POST /api/v1/chat/sessions/{session_id}/save-engram`
- `POST /api/v1/chat/sessions/{session_id}/continue`

## MCP Tools (Initial)
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

## Done Criteria
- Users can run multi-provider chat with persistent sessions.
- Users can save and reuse engrams between chat sessions.
- MCP SSE endpoint supports core chat/engram workflows.
- Skills and AGENT guide are sufficient for a fresh agent session.
- README is up to date and local quality gates pass.
