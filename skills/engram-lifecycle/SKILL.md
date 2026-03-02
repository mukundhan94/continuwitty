---
name: engram-lifecycle
description: Use this skill when implementing or operating engram creation, query, rehydration, ownership, visibility, and cross-session reuse flows.
---

# Engram Lifecycle

## Use This Skill When
- Adding or changing engram create/query/rehydrate behavior.
- Adding ownership/visibility rules for engrams.
- Building save-from-chat or pin-to-session workflows.

## Workflow
1. Confirm schema fields needed (`owner_user_id`, `visibility_scope`, `source_session_id`, soft-delete metadata).
2. Update contracts in `internal/models`.
3. Update persistence in `internal/repository/engram*.go` and related repository modules.
4. Route write-time project fallback through project service resolution helpers.
5. Update REST handlers in `internal/api/session_engrams*.go` and MCP dispatch in `internal/mcp`.
6. Update runtime adapters in `cmd/api` when dependency wiring changes.
7. Add/extend tests for auth, visibility, and behavior.

## Rules
- `private` visibility is default.
- Enforce ownership/actor checks before returning engrams.
- Keep soft-delete/recovery additive; never hard-delete in admin flows.
- For create paths that omit `project_id`, preserve `resolved_project_id` + `used_default_project` metadata where contracted.
- Preserve collection/project integrity on engram project moves.
- Rehydration responses must include compact context + citations/source references.

## Validation
- `go test ./internal/repository ./internal/api ./internal/mcp -count=1`
- `go test ./... -count=1`
- `make eval`
