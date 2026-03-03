# Phase 38 Contradiction Benchmark

This benchmark captures contradiction-warning generation cost in chat context assembly.

## Scope

- Target function: `buildContradictionWarnings` in `internal/chat/context_contradictions.go`.
- Workload model:
  - `50` trace paths (typical bounded linked-recall context).
  - `200` trace paths (upper-bound recall stress case).
- Contradiction density: every third trace path includes contradiction metadata.

## Command

```bash
go test ./internal/chat -run '^$' -bench 'BenchmarkBuildContradictionWarnings(50Paths|200Paths)$' -benchmem
```

## Environment

- Date: `2026-03-03`
- OS/Arch: `darwin/arm64`
- CPU: `Apple M4 Max`

## Results

| Benchmark | ns/op | B/op | allocs/op |
|---|---:|---:|---:|
| `BenchmarkBuildContradictionWarnings50Paths-16` | `1574` | `5008` | `54` |
| `BenchmarkBuildContradictionWarnings200Paths-16` | `7081` | `22544` | `246` |

## Notes

- Contradiction warning synthesis remains sub-10µs per request in both tested path counts.
- Benchmark output should be re-captured after major linked-recall or warning-shaping changes.
