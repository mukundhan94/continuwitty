---
name: memory-lifecycle-policies
description: Use this skill when implementing or modifying chat memory lifecycle controls such as autosave cadence, retention pruning, and lifecycle timeline events.
---

# Memory Lifecycle Policies

## Use This Skill When
- Adding or changing session autosave options.
- Tuning retention windows and snapshot pruning behavior.
- Exposing lifecycle policy controls in REST, MCP, or UI.
- Debugging why snapshots were created, skipped, or pruned.

## Core Workflow
1. Keep lifecycle rules centralized in `internal/chat/lifecycle_policy.go`.
2. Normalize incoming policy values (legacy compatibility included).
3. Evaluate trigger conditions:
   - `interval`: time since latest autosave snapshot
   - `message_count`: assistant message modulo threshold
4. Prevent low-signal writes:
   - skip low-value abstracts
   - skip duplicate snapshot abstracts
5. Enforce retention bounds:
   - prune autosave-tagged session snapshots beyond age/count policy
6. Persist timeline-friendly metadata for REST/MCP/UI consumers.

## Data and API Touchpoints
- Lifecycle fields are stored in chat session records/repository layer.
- REST routes:
  - `GET /api/v1/chat/sessions/{session_id}/lifecycle-policy`
  - `PATCH /api/v1/chat/sessions/{session_id}/lifecycle-policy`
  - `GET /api/v1/chat/sessions/{session_id}/timeline`
- MCP tools:
  - `chat.get_lifecycle_policy`
  - `chat.update_lifecycle_policy`
  - `chat.list_timeline`

## Test Expectations
- Policy unit tests: `internal/chat/lifecycle_policy_test.go`.
- Chat API tests: `internal/api/chat_api_lifecycle_test.go`.
- MCP tests: `internal/mcp/compatibility_service_chat_lifecycle_policy*_test.go`, `internal/mcp/compatibility_service_chat_timeline_test.go`.
- Frontend tests: lifecycle/timeline clients under `web/src/api/chat*.test.ts`.

## Debug Notes
- Timeline event type is derived from lifecycle semantics and snapshot metadata.
- Retention pruning only applies to autosave-tagged session-linked snapshots.
