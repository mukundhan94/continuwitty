# Go Migration Checkpoint CP37

Date: 2026-02-22
Branch: `migrate`

## Scope

Non-checkpoint code-health uplift for legacy embeddings fallback flow.

## Completed Work

- Refactored `internal/embeddings/service.go` to remove duplicated fallback execution paths.
- Added shared helper: `callWithFallback[T any]`.
- Updated both `embedWithFallback` and `embedManyWithFallback` to use the same fallback control flow.

## One-by-One Tests Executed

- `TestEmbeddingServiceFallsBackToLocalProvider`
- `TestEmbeddingServiceEmbedManyUsesProviderIDFromActiveProvider`

## Full Verification

- `/usr/local/go/bin/go test ./...` passed.

## Code Health (CodeScene)

- `internal/embeddings/service.go` improved from `9.09` to `9.68`.
- Remaining note is non-blocking module-level `String Heavy Function Arguments`.

Pre-commit safeguard:

- `pre_commit_code_health_safeguard(git_repository_path=/Users/mukundhan/Projects/engram)` -> `quality_gates=passed`
- Findings: none.
