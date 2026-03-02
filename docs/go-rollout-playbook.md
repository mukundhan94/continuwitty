# Go Rollout Playbook

This runbook defines the staged production rollout for the Go API after release gates are green.

Versioned release prep and rollback references:

- [release-checklist-v1.md](release-checklist-v1.md)
- [release-rollback-runbook-v1.md](release-rollback-runbook-v1.md)

## Prerequisites

- `make release-gate` passes on the target branch.
- optional release-candidate live checks run via `make release-live-provider-gate` when required.
- Observability dashboards for API error rate, latency, and DB saturation are live.

## Traffic Shift Stages

| Stage | Traffic | Hold Time | Promotion Criteria |
|---|---:|---|---|
| 1 | 10% | 30 min | Error rate and p95 latency remain within SLO guardrails |
| 2 | 25% | 30 min | No sustained 5xx increase; auth and MCP health checks remain green |
| 3 | 50% | 60 min | No critical incidents; acceptance smoke and canary probes green |
| 4 | 100% | 24h observation | Stable metrics across all key routes and background jobs |

## Gating Metrics

- HTTP 5xx rate
- p95 / p99 latency on key read and write endpoints
- Login success/failure ratios and lockout behavior
- MCP stream error and timeout rates
- Database saturation (connections, lock waits, slow queries)

## Rollback Criteria

Rollback to the previous stage if any condition persists for more than 5 minutes:

- 5xx rate exceeds the agreed SLO threshold
- p95 latency regresses beyond the agreed stage threshold
- Auth or MCP critical workflows fail smoke checks
- Data integrity issues are detected in write paths

## Rollback Procedure

1. Shift traffic back to the last known-good stage.
2. Confirm health checks recover and error budget stabilizes.
3. Capture incident notes with failing routes and timestamps.
4. Ship a fix, re-run release gates, then retry rollout from the previous stage.
