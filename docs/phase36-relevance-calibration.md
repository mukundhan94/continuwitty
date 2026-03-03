# Phase 36 Relevance Calibration Notes

This note documents the calibrated composite rerank model and benchmark measurements for retrieval latency.

## Composite Scoring Calibration

`QueryEngrams` now applies a five-factor deterministic rerank score:

`score = dense*0.55 + lexical*0.20 + feedback*0.10 + engagement*0.10 + freshness*0.05`

Signal definitions:

- `dense`: inverse pgvector distance (`1 / (1 + distance)`).
- `lexical`: token overlap ratio between query and candidate text fields.
- `feedback`: normalized from `useful_count` vs `contradiction_count`.
- `engagement`: log-scaled from `access_count` (saturates around 20 accesses).
- `freshness`: clamped `freshness_score` in `[0,1]` (defaults to `1.0` when absent).

The weighting keeps semantic relevance dominant while explicitly rewarding active/fresh memory that has positive explicit feedback.

## Regression Coverage

Added/updated tests in `internal/repository`:

- ranking still respects lexical signal boosts.
- ranking prefers positive feedback over contradicted memory when other factors are equal.
- ranking now prefers active/fresh memory over stale/inactive memory under otherwise equal inputs.
- query SQL includes calibrated input columns:
  - `COALESCE(access_count, 0) AS access_count`
  - `COALESCE(freshness_score, 1.0) AS freshness_score`

## Latency Benchmark Notes

Measured on `2026-03-03` with:

- machine: Apple M4 Max (darwin/arm64)
- command:

```bash
go test ./internal/repository -run '^$' -bench 'BenchmarkRerankByCombinedScore(50Candidates|200Candidates)$' -benchmem
```

Results:

- `BenchmarkRerankByCombinedScore50Candidates`: `206956 ns/op` (`~0.207 ms`), `190564 B/op`, `2405 allocs/op`
- `BenchmarkRerankByCombinedScore200Candidates`: `888583 ns/op` (`~0.889 ms`), `761664 B/op`, `9606 allocs/op`

Interpretation:

- rerank overhead remains sub-millisecond for candidate windows up to 200 rows.
- calibration stays within current query latency targets for deterministic local retrieval.
