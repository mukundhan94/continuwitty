# Go Migration Checkpoint CP52

Date: 2026-02-22
Branch: `migrate`

## Scope

Memory-admin API route refactor and parser normalization for Go migration checkpoints.

## Completed Work

- Refactored memory-admin API route wiring into focused files:
  - `internal/api/admin_memory.go`
  - `internal/api/admin_memory_sessions.go`
  - `internal/api/admin_memory_engrams.go`
  - `internal/api/admin_memory_collections.go`
- Preserved all `/api/v1/admin/memory` route behavior while reducing complexity and duplication pressure.
- Kept shared request parsing and error semantics centralized in `internal/api/admin_memory.go`.
- Normalized optional int query parsing call sites to the new typed spec form in:
  - `internal/api/session_engrams.go`
  - `internal/api/session_users.go`

## One-by-One Tests Executed

- `TestMountMemoryAdminRoutesListSessionsForwardsQueryAndActorCheck`
- `TestMountMemoryAdminRoutesDeleteSessionUsesActorAndPayload`
- `TestMountMemoryAdminRoutesListEngramsUsesRequestObject`
- `TestMountMemoryAdminRoutesUpdateEngramMapsStaleTo409`
- `TestMountMemoryAdminRoutesCreateCollectionReturns201`
- `TestMountMemoryAdminRoutesCreateCollectionMapsMissingProjectTo400`
- `TestMountMemoryAdminRoutesReturnsForbiddenWhenActorCheckFails`
- `TestMountSessionAuthRoutesListEngramSourcesUsesRepository`
- `TestMountSessionAuthRoutesEngramCollectionRoutesUseRepository`
- `TestMountSessionAuthRoutesAdminCanManageUsers`
- `TestMountSessionAuthRoutesNonAdminCannotAccessAdminRoutes`

## Full Verification

- `/usr/local/go/bin/go test ./...` passed.

## Code Health (CodeScene)

- `internal/api/admin_memory.go`: `7.78` -> `10.0`
- `internal/api/admin_memory_sessions.go`: `10.0`
- `internal/api/admin_memory_engrams.go`: `10.0`
- `internal/api/admin_memory_collections.go`: `10.0`
- `internal/api/session_engrams.go`: `10.0`
- `internal/api/session_users.go`: `10.0`

Pre-commit safeguard:

- `pre_commit_code_health_safeguard(git_repository_path=/Users/mukundhan/Projects/engram)` -> `quality_gates=passed`
- Findings: none.
