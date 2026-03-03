# Phase 40 Curation Benchmark

## Scope

Latency baseline for Phase 40 curation workflows in `internal/admin`:

- curation action transition path (`ActionMemoryCurationSuggestion`)
- deterministic consolidation-to-curation sync generation path

## Command

```bash
go test ./internal/admin \
  -run '^$' \
  -bench 'Benchmark(ActionMemoryCurationSuggestion|SyncConsolidationCurationSuggestions(50Candidates|200Candidates))$' \
  -benchmem
```

## Environment

- Date: 2026-03-03
- OS/Arch: `darwin/arm64`
- CPU: `Apple M4 Max`

## Results

| Benchmark | ns/op | B/op | allocs/op |
|---|---:|---:|---:|
| `BenchmarkActionMemoryCurationSuggestion` | 48.66 | 272 | 2 |
| `BenchmarkSyncConsolidationCurationSuggestions50Candidates` | 13,125 | 34,442 | 501 |
| `BenchmarkSyncConsolidationCurationSuggestions200Candidates` | 51,607 | 137,721 | 2,001 |

## Notes

- Action transition latency is well below 1 microsecond in the service-layer benchmark baseline.
- Consolidation sync scales linearly with candidate count in the deterministic in-process benchmark.
- This benchmark intentionally isolates service-layer behavior (repository dependencies mocked) so regression tracking remains stable in CI.
