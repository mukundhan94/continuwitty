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

Live triage continuity run (focused non-deterministic scenario):

```bash
make acceptance-test-triage-live
```

Deterministic mocked lifecycle run (no live provider dependency):

```bash
make acceptance-test-mock
```

Dockerized deterministic mocked lifecycle run:

```bash
make acceptance-test-mock-docker
```

Dockerized live Bedrock run:

```bash
make acceptance-test-bedrock-live-docker
```

Dockerized live triage continuity run:

```bash
make acceptance-test-triage-live-docker
```

The `@bedrock-live` scenario intentionally validates non-deterministic behavior by asserting only:

- Bedrock provider session is created with the default model field populated.
- assistant response is received and exceeds a minimum character threshold.
- response does not match known provider error text.

The `@triage-live` scenario adds an incident-command workflow:

- generates a live triage memo with structured incident signals,
- saves that response as a project-visible engram,
- continues into a new chat, pins the saved engram,
- requests a commander handoff brief and validates continuity-oriented signals.

The `@mock` lifecycle scenario validates autosave behavior deterministically by mocking chat/session endpoints:

- autosave `message_count` triggers a timeline snapshot only at threshold,
- autosave `off` keeps timeline empty after repeated sends,
- create-session payload carries expected autosave policy fields.

The `@phase37` mock scenario validates consolidation maintenance quality deterministically via admin APIs:

- seeds duplicate and non-duplicate engrams inside an isolated project,
- refreshes/list consolidation suggestions and computes grouping precision/recall,
- asserts threshold compliance and verifies merge-action workflow (`suggested` -> `merged`).

When dockerized scenarios fail, screenshots are written to `acceptance-tests/artifacts/`.
