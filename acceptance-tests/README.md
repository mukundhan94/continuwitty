# Acceptance Tests (Playwright + Gherkin)

This folder contains behavior-level acceptance tests for the local chat workbench.

## Stack

- `playwright-bdd` for Gherkin step binding with Playwright runner.
- `bddgen` for generating runnable Playwright tests from feature files.
- `@playwright/test` for browser automation, traces, and artifacts.

## Layout

- `features/*.feature`: Gherkin acceptance scenarios.
- `src/steps/*.ts`: Step definitions.
- `src/support/fixtures.ts`: shared fixtures, BDD bindings, login helpers, failure screenshots.
- `src/support/env.ts`: Env parsing and defaults.
- `playwright.config.ts`: Playwright + `defineBddConfig` wiring.
- `.features-gen/`: generated tests from `bddgen` (do not edit directly).

## Environment

Copy the example:

```bash
cp acceptance-tests/.env.example acceptance-tests/.env
```

Supported env keys:

- `WEB_BASE_URL`: base URL for the UI under test.
- `API_BASE_URL`: base URL for API utilities (reserved for future steps).
- `UI_USERNAME`: login username.
- `UI_PASSWORD`: login password.
- `PW_HEADLESS`: set to `false` for headed runs.
- `PW_TIMEOUT_MS`: default per-action timeout.

## Commands

Local run:

```bash
make acceptance-sync
make acceptance-bddgen
make acceptance-typecheck
make acceptance-test
```

Dockerized run (uses `docker compose` services):

```bash
make acceptance-test-docker
```

When dockerized scenarios fail, screenshots are written to `acceptance-tests/artifacts/`.
