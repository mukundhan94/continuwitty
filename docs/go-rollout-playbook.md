# Go Rollout Playbook

This runbook defines the staged production rollout from Python to Go after parity gates are green.

## Prerequisites

- `go test ./...` passes on the target branch.
- `make acceptance-test-mock-docker` passes.
- `make openapi-check` passes.
- `make shadow-compare` passes (status-family parity).
- `make benchmark-compare` produces `findings/go-migration-benchmark.md`.

## Traffic Shift Stages

| Stage | Go Traffic | Hold Time | Promotion Criteria |
|---|---:|---|---|
| 1 | 10% | 30 min | Error rate and p95 latency stay within ±10% of Python baseline |
| 2 | 25% | 30 min | No sustained 5xx increase, auth/MCP health checks remain green |
| 3 | 50% | 60 min | No critical incidents, acceptance smoke and canary probes green |
| 4 | 100% | 24h observation | Stable metrics; Python remains on standby during observation window |

## Gating Metrics

- HTTP 5xx rate (`go` vs `python`)
- p95 / p99 latency on key read and write endpoints
- Login success/failure ratios and lockout behavior
- MCP stream error rate and timeout rate
- Database saturation (connections, lock waits, slow query rate)

## Rollback Criteria

Rollback to previous stage (or full Python) if any are true for more than 5 minutes:

- 5xx rate exceeds baseline by >20%
- p95 latency regresses by >25% on key endpoints
- Auth or MCP critical workflows fail acceptance smoke checks
- Data integrity issues are detected in write paths

## Rollback Procedure

1. Set traffic weight back to the previous known-good stage.
2. Confirm health checks recover and error budget stabilizes.
3. Capture incident notes and exact failing routes.
4. Create a fix checkpoint commit, re-run parity gates, and retry rollout from the prior stage.
