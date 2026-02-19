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
1. Keep lifecycle rules centralized in `api/app/chat/lifecycle_policy.py`.
2. Normalize incoming policy values (legacy `autosave_enabled` + strategy compatibility).
3. Evaluate trigger conditions:
   - `interval`: time since latest autosave snapshot
   - `message_count`: assistant message modulo threshold
4. Prevent low-signal writes:
   - skip low-value abstracts
   - skip duplicate snapshot abstracts
5. Enforce retention bounds:
   - remove autosave-tagged session snapshots beyond age/count policy
6. Persist timeline-friendly metadata so UI/MCP can show lifecycle events.

## Data and API Touchpoints
- Session policy fields are stored on `chat_sessions`.
- Lifecycle REST routes:
  - `GET /api/v1/chat/sessions/{session_id}/lifecycle-policy`
  - `PATCH /api/v1/chat/sessions/{session_id}/lifecycle-policy`
  - `GET /api/v1/chat/sessions/{session_id}/timeline`
- Lifecycle MCP tools:
  - `chat.get_lifecycle_policy`
  - `chat.update_lifecycle_policy`
  - `chat.list_timeline`

## Test Expectations
- Unit: `api/tests/test_chat_lifecycle_policy.py`
- Service/repo: `api/tests/test_chat_service.py`, `api/tests/test_chat_repository.py`
- API/MCP integration: `api/tests/test_chat_api_integration.py`, `api/tests/test_mcp_api_integration.py`
- Web: lifecycle controls and timeline rendering tests in session/chat components

## Debug Notes
- Timeline event type is derived from engram tags.
- Retention pruning only deletes autosave-tagged session-linked snapshots.
- When verifying behavior, compare lifecycle policy values with created snapshot tags and timeline output.
