# AGENT Guide - Engram Vault

This guide is the operational contract for contributors and coding agents.
Use it as the default workflow when adding or refactoring features.

## 1. Repository Map
- `api/app/`: FastAPI services, models, repositories, auth, provider adapters, MCP server.
- `api/app/chat/`: chat domain package (API router, service orchestration, context assembly, errors).
- `api/app/ingestion/`: document ingestion domain package (text/file intake, deterministic chunking, blended query path).
- `api/app/mcp/`: MCP domain package (SSE transport, JSON-RPC dispatch, tool errors).
- `api/app/providers/`: provider domain package (`base`, `errors`, concrete adapters, `registry`).
- `api/app/embeddings/`: embedding provider abstraction (local deterministic + optional external provider fallback).
- `api/tests/`: unit and integration tests.
- `api/evals/`: scenario-based eval harness.
- `db/init/`: SQL schema and migration-style DDL.
- `web/`: React UI (chat/session/pinning workflows, streaming UX, frontend tests).
- `web/src/styles/`: shared style system (theme tokens, global CSS vars, reusable styled shells).
- `web/tailwind.config.ts` + `web/postcss.config.cjs`: Tailwind utility pipeline for the web app.
- `acceptance-tests/`: Playwright-BDD acceptance framework (`features`, `steps`, `support`, `playwright.config.ts`).
- `api/Dockerfile`, `web/Dockerfile`, `acceptance-tests/Dockerfile`: container runtimes.
- `docker-compose.yml`: local stack orchestration for DB/API/web and acceptance profile.
- `skills/`: reusable agent workflows for this project.

## 2. Non-Negotiable Rules
- Keep all features local-first by default.
- Use `uv` for dependency and command execution.
- Update `README.md` in every implementation phase.
- Preserve backward compatibility for existing endpoints unless intentionally versioned.
- Add tests for every non-trivial behavior change.
- Keep all frontend colors/typography/shadows in `web/src/styles/theme.ts` and consume via shared primitives/utilities.

## 3. Daily Workflow
1. Pull latest and inspect `git status`.
2. Run setup: `make sync`, `make db-up`.
3. Implement one phase at a time.
4. Run checks: `make lint`, `make test`, `make eval`, `make check`.
5. If `web/` changed: run `make web-check`.
6. If `acceptance-tests/`, `web/Dockerfile`, `api/Dockerfile`, `docker-compose.yml`, or auth/session workflow changed: run `make acceptance-bddgen`, `make acceptance-typecheck`, and `make acceptance-test-docker`.
7. If Bedrock provider behavior changed, run live non-deterministic acceptance gate: `make acceptance-test-bedrock-live` (or docker equivalent).
8. For docker runtime changes: validate `docker compose config`.
9. Update docs (`README.md`, `Plan.md` progress, `docs/architecture-playbook.md` when call flows or schema semantics change, skill docs if needed).
10. Commit with phase-scoped message.

## 4. Architecture Boundaries
- Route handlers in `main.py` should orchestrate only.
- Domain routers (`api/app/chat/api.py`) should remain thin and delegate to services.
- DB access must live in repository modules.
- Context assembly logic belongs in domain context modules (`api/app/chat/context.py`), not route handlers.
- Lifecycle/autosave/retention heuristics must stay in `api/app/chat/lifecycle_policy.py` as pure functions so they remain testable and reusable across REST/MCP paths.
- Provider SDK calls must stay in `api/app/providers/`.
- Document chunking and retrieval behavior belongs in `api/app/ingestion/`.
- Embedding provider routing belongs in `api/app/embeddings/`; do not call provider endpoints directly from repositories.
- MCP tool handlers should call service/repository layers, not raw SQL.
- UI should call API/MCP contracts only, not reimplement business logic.
- UI styling should use shared tokens in `web/src/styles/theme.ts`; avoid ad-hoc hardcoded palette values in components.

## 4a. Domain Module Layout Rules
- For each new domain, create a dedicated module/package under `api/app/`:
  - data access (`*_repository.py`)
  - domain logic (`*_service.py` when logic grows)
  - transport boundary (routes/MCP handlers in separate files where feasible)
- Keep shared contracts in `models.py` only when broadly reused.
- Prefer domain-local helper modules over adding unrelated helpers to `main.py`.
- Add tests in matching domain-focused files (`test_<domain>*.py`).

## 5. Schema and Migration Policy
- Prefer additive, forward-only SQL changes.
- Never silently drop columns or rewrite semantics without migration notes.
- Keep DDL idempotent (`IF NOT EXISTS`, compatible alters where possible).
- Add indexes for new query paths.
- Document schema impact in README under implementation log.

## 6. API and Interface Policy
- For breaking changes, add new fields/routes and keep old behavior until deprecated.
- Validate all external payloads with Pydantic models.
- Include stable identifiers in responses (`user_id`, `session_id`, `engram_id`).
- For chat responses, return `used_engram_ids` whenever context retrieval is used.
- For chat responses, return `used_document_chunk_ids` when document chunks are used for context.
- For chat responses, return `source_references` whenever retrieval context includes citations or document chunk evidence.

## 7. Provider Adapter Contract
- Implement common methods across providers:
  - `generate(messages, ...)`
  - `stream_generate(messages, ...)`
  - `healthcheck()`
- Normalize errors to app-level exceptions.
- Keep provider-specific payload differences internal.
- Never call providers directly from routes or templates.
- Keep adapter selection centralized in `api/app/providers/registry.py`.

## 8. MCP Tool Authoring Standards
- Tool names are verb/object and stable (`chat.create_session`).
- Inputs/outputs are JSON schema friendly.
- Enforce auth/visibility checks exactly as API routes do.
- Stream responses using JSON-RPC framed SSE events.
- Return structured error payloads with machine-parseable codes.
- Keep all tool routing in `api/app/mcp/service.py`; avoid embedding tool logic in route handlers.
- Keep external MCP compatibility (`initialize`, `tools/list`, `tools/call`) aligned with direct tool methods.
- When MCP contracts change, update both typed clients in the same phase:
  - `api/app/mcp/client.py`
  - `web/src/api/mcpClient.ts`

## 9. Testing Requirements
- Unit tests for pure logic and adapters.
- Integration tests for API + DB behavior.
- Eval tests for memory quality scenarios.
- Frontend tests for UI helpers/components and session workflow logic.
- Acceptance tests for end-to-end workflow regressions (`acceptance-tests/features/*.feature`).
- Lifecycle policy changes require regression coverage across:
  - `api/tests/test_chat_lifecycle_policy.py`
  - `api/tests/test_chat_api_integration.py`
  - `api/tests/test_mcp_api_integration.py`
- `@bedrock-live` acceptance tests are optional for deterministic local runs, but required when Bedrock adapter/runtime behavior changes.
- Add regression tests when fixing bugs.
- No phase is complete unless `make check` passes. If web files changed, `make web-check` is also required.
- If dockerized UX or auth/session behavior changed, `make acceptance-test-docker` is required before merge.

## 10. Refactor Checklist
Before merging refactors:
- No behavior regressions in existing endpoints.
- Docs and examples still match implementation.
- Provider adapters still respect shared interface.
- SQL query plans remain index-supported for main paths.
- New abstractions reduce duplication and improve readability.

## 11. Operations and Incident Handling (Local)
- If API is unhealthy: check `/healthz`, DB container status, and env vars.
- For startup/debug issues: use `APP_ENV=development` and `LOG_CONFIG_IN_DEV=true` to print a redacted parsed config snapshot.
- If retrieval quality drops: run `make eval`, inspect reranking and citation packing paths.
- If auth fails unexpectedly: inspect `data/audit_events.jsonl` and session settings.
- If MCP stream fails: verify SSE endpoint wiring and JSON-RPC event framing.
- If acceptance tests fail to render UI in Docker: verify Vite host allow list (`VITE_ALLOWED_HOSTS`) and proxy target (`VITE_API_PROXY_TARGET`).

## 12. Long-Run Maintenance Cadence
- Weekly: run full quality gates and evals.
- Monthly: review dependency updates and provider API changes.
- Each feature cycle: refresh `README.md`, `Plan.md`, and affected skill docs; update `docs/architecture-playbook.md` for call-flow or data-semantics changes.
- Keep skill instructions short, precise, and executable.
