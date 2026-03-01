---
name: chat-rag-operator
description: Use this skill when implementing chat sessions, context assembly with engrams/document chunks, continuity across sessions, and save-as-engram flows.
---

# Chat RAG Operator

## Use This Skill When
- Building chat session APIs/UI.
- Wiring pinned engrams into model context.
- Blending ingested document chunk evidence with snapshot memory.
- Implementing continue-in-new-chat flows.
- Adding session lifecycle policy controls (autosave cadence, retention, timeline events).

## Workflow
1. Persist session and message state through repository modules.
2. Build context assembler using:
   - pinned engrams
   - dynamic retrieval hits
   - link-aware recall (when enabled)
   - document chunk evidence (when available)
3. Call provider adapters via `internal/providers` registry.
4. Persist assistant/user messages and structured metadata.
5. Expose `used_engram_ids`, `used_engram_link_ids`, `used_document_chunk_ids`, and `source_references` in chat payloads.
6. Run lifecycle maintenance after successful assistant output:
   - autosave trigger (interval or message-count)
   - duplicate/low-value snapshot guards
   - retention pruning and timeline emission

## Module Layout (Current)
- `internal/chat/service.go`: orchestration and continuity flows.
- `internal/chat/context*.go`: retrieval + pinned context assembly.
- `internal/chat/lifecycle_policy.go`: autosave/retention/timeline pure-policy helpers.
- `internal/api/chat_api*.go`: REST transport handlers.
- `internal/mcp/compatibility_dispatch_chat*.go`: MCP tool routing.
- `cmd/api/chat_runtime.go`: runtime dependency wiring.
- `web/src/api/chat*.ts`: typed frontend API client contracts.

## Continuity Pattern
- Continuation creates a new session and clones selected context/settings.
- Session IDs are immutable; do not mutate historical sessions in place.

## Validation
- Unit tests: `internal/chat/*_test.go`.
- API/MCP tests: `internal/api/chat_api*_test.go`, `internal/mcp/compatibility_service_chat*_test.go`.
- Frontend tests: `web/src/api/chat*.test.ts`, `web/src/utils/chat.test.ts`.
- Run `go test ./... -count=1` and `make eval`.
