# Phase 39 Query Benchmark

## Scope

Benchmark coverage for complex temporal/engagement/trace query filter combinations in `internal/repository`:

- dense all-filter query assembly path (`buildEngramQueryWhere`)
- mixed filter-matrix query assembly path (`buildEngramQueryWhere`)

## Command

```bash
go test ./internal/repository \
  -run '^$' \
  -bench 'Benchmark(BuildEngramQueryWhereComplexTemporalEngagementTrace|BuildEngramQueryWhereComplexFilterMatrix)$' \
  -benchmem
```

## Environment

- Date: 2026-03-03
- OS/Arch: `darwin/arm64`
- CPU: `Apple M4 Max`

## Results

| Benchmark | ns/op | B/op | allocs/op |
|---|---:|---:|---:|
| `BenchmarkBuildEngramQueryWhereComplexTemporalEngagementTrace` | 3,501 | 6,021 | 83 |
| `BenchmarkBuildEngramQueryWhereComplexFilterMatrix` | 2,669 | 5,952 | 81 |

## Notes

- Both scenarios remain sub-4 microseconds for SQL/parameter assembly under dense filter combinations.
- Allocation profile is stable between dense single-shape and mixed matrix workloads.
- Coverage targets the previous Phase 39 follow-up gap for complex temporal/engagement/trace combinations.
