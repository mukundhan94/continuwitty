# Engram Vault - Testing Guide

> Testing reference for backend, frontend, and acceptance suites.
> See [AGENT.md](../AGENT.md) section 9 for testing policy.

---

## Quick Commands

```bash
make test              # Run all Go backend tests
make test-unit         # Run Go unit tests
make test-integration  # Run Go integration tests
make lint              # go vet
make format-check      # Go formatting verification
make check             # lint + format-check + tests
make web-check         # frontend lint + test + build
make stack-smoke       # containerized db+api smoke check
make release-gate      # deterministic release candidate gate
make release-live-provider-gate # optional live-provider release gate
```

---

## Backend Test Suite

Go backend tests live under:

- `cmd/api/**/*_test.go`
- `internal/**/*_test.go`

Primary gate:

```bash
go test ./... -count=1
```

Notes:

- Repository semantics are primarily covered under `internal/repository`.
- API/auth/router behavior is primarily covered under `internal/api`.
- MCP transport and compatibility behavior is primarily covered under `internal/mcp`.

---

## Frontend Test Suite

Tests live under `web/src/**/*.test.ts(x)`.

Primary gate:

```bash
make web-check
```

---

## Acceptance Tests (Playwright-BDD)

### Local Runner

```bash
make acceptance-sync
make acceptance-bddgen
make acceptance-typecheck
make acceptance-test
make acceptance-test-mock
```

### Dockerized Runner

```bash
make acceptance-test-docker
make acceptance-test-mock-docker
```

### Live Provider Runners

```bash
make acceptance-test-bedrock-live
make acceptance-test-bedrock-live-docker
make acceptance-test-triage-live
make acceptance-test-triage-live-docker
```

Notes:

- Default execution filters to `not @bedrock-live` for deterministic local runs.
- `make acceptance-bddgen` regenerates Playwright specs from `.feature` files.
- Failure screenshots are persisted to `acceptance-tests/artifacts/`.
- Generated specs are written to `acceptance-tests/.features-gen/`.

---

## Release Gates

Deterministic release gate (required):

```bash
make release-gate
```

Optional live-provider gate (release candidates):

```bash
make release-live-provider-gate
```

For full process details, see:

- [release-checklist-v1.md](release-checklist-v1.md)
- [release-rollback-runbook-v1.md](release-rollback-runbook-v1.md)
