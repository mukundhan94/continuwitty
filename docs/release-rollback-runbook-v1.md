# Release Rollback Runbook v1

Versioned rollback procedure for production incidents during or after release promotion.

## Trigger Conditions

Rollback when one or more conditions persist beyond the agreed hold window:

1. HTTP `5xx` error rate exceeds stage SLO.
2. p95/p99 latency regression breaches stage threshold.
3. Login/session or MCP stream critical paths fail smoke checks.
4. Data integrity anomalies are observed in write paths.

## Immediate Actions (First 5 Minutes)

1. Freeze further traffic promotion.
2. Page on-call owner and incident commander.
3. Capture initial evidence:
   - failing route/tool names,
   - timestamp window,
   - error samples,
   - current rollout stage percentage.

## Rollback Steps

1. Shift traffic to the previous known-good stage/version.
2. Verify health:
   - `/healthz`
   - `/api/v1/version`
   - `/api/v1/metrics` (errors trending down)
3. Re-run deterministic smoke checks:

```bash
make release-smoke-docker
```

4. Confirm recovery in monitoring:
   - 5xx rate normalizes
   - latency returns to baseline
   - auth + MCP probes green

## Data Safety Verification

1. Validate no in-flight migration or schema drift occurred.
2. Validate latest write operations for:
   - session/message persistence
   - engram create/update
   - MCP token and OAuth flows (if touched by release)
3. Record any remediation required before re-promotion.

## Communication Template

1. Incident start time and rollback completion time.
2. Impacted workflows (API/UI/MCP).
3. Recovery status and customer-facing impact summary.
4. Next update SLA and owner.

## Re-Promotion Gate

Do not retry promotion until:

1. Root cause hypothesis is documented.
2. Fix is merged and validated with:

```bash
make release-gate
```

3. Rollout window is re-approved.
