# Release Checklist v1

Versioned release checklist for deterministic Go runtime validation and deploy readiness.

## Scope

- Applies to tagged releases (`v*`) and release candidates promoted from `migrate`/`main`.
- Complements [go-rollout-playbook.md](go-rollout-playbook.md) and [release-rollback-runbook-v1.md](release-rollback-runbook-v1.md).

## Deterministic Gate (Required)

Run this full gate before tagging or deployment:

```bash
make release-gate
```

This includes:

1. `make check` (backend vet + formatting + tests + EvalOps delta gate)
2. `make web-check` (frontend lint + tests + build)
3. `make acceptance-test-mock-docker` (deterministic acceptance profile)
4. `make release-smoke-docker` (compose `release-smoke` profile)

## Live-Provider Gate (Optional, Release Candidates)

Use when promoting a release candidate where live Bedrock behavior must be validated.

```bash
make release-live-provider-gate
```

Expected environment:

- `AWS_REGION`
- `AWS_ACCESS_KEY_ID`
- `AWS_SECRET_ACCESS_KEY`
- optional `AWS_SESSION_TOKEN`
- `BEDROCK_LIVE_EXPECTED_MODEL`
- optional `BEDROCK_LIVE_PROMPT`

## CI Expectations

- `Go Backend Checks` job passes.
- `Web Frontend Checks` job passes.
- `Acceptance Deterministic Suite` job passes.
- `Release Smoke (Compose Profile)` job passes.
- `Release Live-Provider Gate` only runs when manually requested (`workflow_dispatch` + `run_live_provider=true`).

## Artifact + Metadata Checks

1. Confirm `APP_SEMANTIC_VERSION` and release tag alignment.
2. Confirm `APP_COMMIT_SHA` resolves to the tagged commit.
3. Confirm release notes include:
   - phase/checkpoint deltas,
   - operator-impacting config changes,
   - rollback trigger notes.

## Sign-Off

1. Product/engineering sign-off on deterministic gate output.
2. Operator sign-off on rollout window and monitoring readiness.
3. Security sign-off if release includes auth/session/OAuth/MCP policy changes.
