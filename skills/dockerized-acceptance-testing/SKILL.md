# Skill: Dockerized Acceptance Testing

Use this skill when changing login, session lifecycle, chat layout, API proxy wiring, or docker runtime setup.

## Goal

Keep end-to-end UX regressions detectable through Gherkin scenarios running against containerized API + web services.

## Workflow

1. Validate compose config:
   - `docker compose config`
2. Run baseline quality gates:
   - `make check`
   - `make web-check` (if `web/` changed)
3. Validate acceptance code:
   - `make acceptance-bddgen`
   - `make acceptance-typecheck`
4. Run dockerized acceptance:
   - `make acceptance-test-docker`
5. If Bedrock integration changed, run live acceptance:
   - `make acceptance-test-bedrock-live`
   - `make acceptance-test-triage-live`
   - or `make acceptance-test-bedrock-live-docker` when validating compose-network behavior.
   - or `make acceptance-test-triage-live-docker` for the triage continuity workflow.
6. If failures occur:
   - check `acceptance-tests/src/support/fixtures.ts` screenshot artifacts behavior
   - verify `VITE_ALLOWED_HOSTS` includes `web` for compose-network browser access
   - verify `VITE_API_PROXY_TARGET` points to `http://api:8000` in dockerized runs

## Required Files to Update

- `acceptance-tests/features/*.feature` for new behavior specs
- `acceptance-tests/src/steps/*.ts` for step coverage
- `acceptance-tests/src/support/*.ts` for shared setup/env/timeouts
- `README.md` test matrix and workflow notes
- `AGENT.md` quality gate expectations

## Exit Criteria

- `make check` passes
- `make web-check` passes (when frontend changed)
- `make acceptance-bddgen` passes
- `make acceptance-typecheck` passes
- `make acceptance-test-docker` passes
- If Bedrock changed: `make acceptance-test-bedrock-live` passes
- If triage continuity behavior changed: `make acceptance-test-triage-live` passes
- README contains updated workflow + env notes for newcomers
