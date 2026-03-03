# Phase 40 Curation Benchmark

## Scope

Latency baseline for Phase 40 curation workflows in `internal/admin`:

- curation action transition path (`ActionMemoryCurationSuggestion`)
- curation applied transition with downstream orchestration:
  - consolidation side effect (`ActionMemoryCurationSuggestionAppliedConsolidate`)
  - contradiction side effect (`ActionMemoryCurationSuggestionAppliedContradiction`)
- deterministic consolidation-to-curation sync generation path

## Command

```bash
go test ./internal/admin \
  -run '^$' \
  -bench 'Benchmark(ActionMemoryCurationSuggestion|ActionMemoryCurationSuggestionAppliedConsolidate|ActionMemoryCurationSuggestionAppliedContradiction|SyncConsolidationCurationSuggestions(50Candidates|200Candidates))$' \
  -benchmem
```

## Environment

- Date: 2026-03-03
- OS/Arch: `darwin/arm64`
- CPU: `Apple M4 Max`

## Results

| Benchmark | ns/op | B/op | allocs/op |
|---|---:|---:|---:|
| `BenchmarkActionMemoryCurationSuggestion` | 63.90 | 272 | 2 |
| `BenchmarkActionMemoryCurationSuggestionAppliedConsolidate` | 305.3 | 1,072 | 9 |
| `BenchmarkActionMemoryCurationSuggestionAppliedContradiction` | 334.8 | 1,088 | 9 |
| `BenchmarkSyncConsolidationCurationSuggestions50Candidates` | 19,299 | 34,442 | 501 |
| `BenchmarkSyncConsolidationCurationSuggestions200Candidates` | 57,363 | 137,721 | 2,001 |

## Notes

- Action transition latency remains sub-microsecond, including applied-side-effect dispatch paths.
- Consolidation sync scales linearly with candidate count in the deterministic in-process benchmark.
- This benchmark intentionally isolates service-layer behavior (repository dependencies mocked) so regression tracking remains stable in CI.
