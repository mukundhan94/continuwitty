# AGENT Guide - Engram Vault

This guide is the operational contract for contributors and coding agents.
Use it as the default workflow when adding or refactoring features.

## 1. Repository Map
- `api/app/`: FastAPI services, models, repositories, auth, provider adapters, MCP server.
- `api/app/providers/`: provider domain package (`base`, `errors`, concrete adapters, `registry`).
- `api/tests/`: unit and integration tests.
- `api/evals/`: scenario-based eval harness.
- `db/init/`: SQL schema and migration-style DDL.
- `web/`: React UI (chat/session/pinning workflows).
- `skills/`: reusable agent workflows for this project.

## 2. Non-Negotiable Rules
- Keep all features local-first by default.
- Use `uv` for dependency and command execution.
- Update `README.md` in every implementation phase.
- Preserve backward compatibility for existing endpoints unless intentionally versioned.
- Add tests for every non-trivial behavior change.

## 3. Daily Workflow
1. Pull latest and inspect `git status`.
2. Run setup: `make sync`, `make db-up`.
3. Implement one phase at a time.
4. Run checks: `make lint`, `make test`, `make eval`, `make check`.
5. Update docs (`README.md`, `Plan.Next.md` progress, skill docs if needed).
6. Commit with phase-scoped message.

## 4. Architecture Boundaries
- Route handlers in `main.py` should orchestrate only.
- DB access must live in repository modules.
- Provider SDK calls must stay in `api/app/providers/`.
- MCP tool handlers should call service/repository layers, not raw SQL.
- UI should call API/MCP contracts only, not reimplement business logic.

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

## 9. Testing Requirements
- Unit tests for pure logic and adapters.
- Integration tests for API + DB behavior.
- Eval tests for memory quality scenarios.
- Add regression tests when fixing bugs.
- No phase is complete unless `make check` passes.

## 10. Refactor Checklist
Before merging refactors:
- No behavior regressions in existing endpoints.
- Docs and examples still match implementation.
- Provider adapters still respect shared interface.
- SQL query plans remain index-supported for main paths.
- New abstractions reduce duplication and improve readability.

## 11. Operations and Incident Handling (Local)
- If API is unhealthy: check `/healthz`, DB container status, and env vars.
- If retrieval quality drops: run `make eval`, inspect reranking and citation packing paths.
- If auth fails unexpectedly: inspect `data/audit_events.jsonl` and session settings.
- If MCP stream fails: verify SSE endpoint wiring and JSON-RPC event framing.

## 12. Long-Run Maintenance Cadence
- Weekly: run full quality gates and evals.
- Monthly: review dependency updates and provider API changes.
- Each feature cycle: refresh `README.md`, `Plan.Next.md`, and affected skill docs.
- Keep skill instructions short, precise, and executable.
