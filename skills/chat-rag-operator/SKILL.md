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

## Workflow
1. Persist session and message state in DB-backed repositories.
2. Build context assembler using:
   - pinned engrams
   - dynamic retrieval hits
   - short citation pack
   - document chunk evidence (when available)
3. Call provider adapter through registry.
4. Save assistant/user messages and metadata.
5. Expose `used_engram_ids`, `used_document_chunk_ids`, and `source_references` in chat response payloads.
6. Emit debug trace payloads for developer inspection:
   - embedding timings
   - provider call duration and token usage
   - request/response snapshots
7. Add save-as-engram endpoint to snapshot useful chat state.

## Module Layout (Current)
- `api/app/chat/api.py`: chat route transport layer.
- `api/app/chat/service.py`: orchestration and continuity flows.
- `api/app/chat/context.py`: retrieval + pinned context assembly.
- `api/app/chat/errors.py`: domain errors for HTTP mapping.

## Continuity Pattern
- Start a new session by cloning provider/system settings and pinned engrams.
- Keep sessions immutable by ID: continuation always creates a new session ID.

## Validation
- Integration test: create -> message -> pin -> continue -> save-as-engram.
- Unit test: debug trace contains timings/tokens and remains non-blocking when telemetry sinks fail.
