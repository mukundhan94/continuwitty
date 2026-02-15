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
- `ACCEPTANCE_BDD_TAGS`: tag expression for `playwright-bdd` selection. Default is `not @bedrock-live`.
- `BEDROCK_LIVE_EXPECTED_MODEL`: optional strict model-id assertion for live Bedrock scenario.
- `BEDROCK_LIVE_PROMPT`: prompt used by the `@bedrock-live` scenario.
- `BEDROCK_LIVE_MIN_RESPONSE_CHARS`: minimum assistant response length for non-deterministic Bedrock validation.
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

Single-command local UI full suite (includes `@bedrock-live`):

```bash
cd acceptance-tests && npm run test:ui:all
```

Open Playwright interactive UI for manual debugging:

```bash
cd acceptance-tests && npm run test:ui
```

Dockerized run (uses `docker compose` services):

```bash
make acceptance-test-docker
```

Live Bedrock run (excluded from default deterministic acceptance suite):

```bash
make acceptance-test-bedrock-live
```

Dockerized live Bedrock run:

```bash
make acceptance-test-bedrock-live-docker
```

The `@bedrock-live` scenario intentionally validates non-deterministic behavior by asserting only:

- Bedrock provider session is created with the default model field populated.
- assistant response is received and exceeds a minimum character threshold.
- response does not match known provider error text.

When dockerized scenarios fail, screenshots are written to `acceptance-tests/artifacts/`.
