# Go Migration Benchmark Report

- Generated at: 2026-02-27 10:18:00 UTC

| Runtime | Endpoint | Requests | Concurrency | Req/s | P50 (ms) | P95 (ms) | P99 (ms) | Mean (ms) | Statuses |
|---|---|---:|---:|---:|---:|---:|---:|---:|---|
| go | `/healthz` | 300 | 20 | 3704.87 | 2.90 | 21.06 | 22.21 | 5.25 | 200:300 |
| python | `/healthz` | 300 | 20 | 2452.44 | 7.88 | 9.53 | 9.91 | 7.89 | 200:300 |
| go | `/api/v1/version` | 300 | 20 | 5594.84 | 2.84 | 9.27 | 11.81 | 3.37 | 200:300 |
| python | `/api/v1/version` | 300 | 20 | 2506.23 | 7.70 | 10.07 | 11.08 | 7.74 | 200:300 |
