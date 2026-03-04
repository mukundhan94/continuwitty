# Phase 77 Query Score-Band Benchmark

This benchmark captures repository query/rerank/score-band filtering performance after introducing full score-band filter support and hardening tests.

## Command

```bash
make benchmark-query-score-bands
```

Equivalent direct command:

```bash
go test ./internal/repository -run '^$' -bench 'Benchmark(FilterByRankScoreBandsFullMatrix(50Candidates|200Candidates)|RerankByCombinedScore(50Candidates|200Candidates)|BuildEngramQueryWhereComplexFilterMatrix)$' -benchmem
```

## Environment

- Date: 2026-03-03
- OS/Arch: darwin/arm64
- CPU: Apple M4 Max

## Baseline Results

| Benchmark | ns/op | B/op | allocs/op |
|---|---:|---:|---:|
| `BenchmarkRerankByCombinedScore50Candidates` | `227555` | `193321` | `2753` |
| `BenchmarkRerankByCombinedScore200Candidates` | `834953` | `772807` | `11000` |
| `BenchmarkBuildEngramQueryWhereComplexFilterMatrix` | `2527` | `5952` | `81` |
| `BenchmarkFilterByRankScoreBandsFullMatrix50Candidates` | `2067` | `416` | `1` |
| `BenchmarkFilterByRankScoreBandsFullMatrix200Candidates` | `9440` | `1792` | `1` |

## Notes

- Full score-band filtering remains low-allocation (`1 alloc/op`) and scales linearly with candidate-set size.
- Filter-matrix cost remains materially lower than rerank cost at both benchmark sizes.
