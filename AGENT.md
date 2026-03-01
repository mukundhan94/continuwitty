# AGENT Guide - Engram Vault

This guide is the operational contract for contributors and coding agents.
Use it as the default workflow when adding or refactoring features.

## 1. Repository Map
- `cmd/api/`: Go API runtime wiring and MCP adapters.
- `internal/`: Go domain modules (API handlers, repositories, auth, providers, MCP, admin, ingestion, export).
- `db/init/`: SQL schema and migration-style DDL.
- `web/`: React UI (chat/session/pinning workflows, streaming UX, frontend tests).
- `web/src/styles/`: shared style system (theme tokens, global CSS vars, reusable styled shells).
- `web/tailwind.config.ts` + `web/postcss.config.cjs`: Tailwind utility pipeline for the web app.
- `acceptance-tests/`: Playwright-BDD acceptance framework (`features`, `steps`, `support`, `playwright.config.ts`).
- `Dockerfile`, `web/Dockerfile`, `acceptance-tests/Dockerfile`: container runtimes.
- `docker-compose.yml`: local stack orchestration for DB/API/web and acceptance profile.
- `skills/`: reusable agent workflows for this project.

## 2. Non-Negotiable Rules
- Keep all features local-first by default.
- Update `docs/implementation-log.md` and `migration/checkpoints/checkpoint.md` in every implementation phase.
- Preserve backward compatibility for existing endpoints unless intentionally versioned.
- Add tests for every non-trivial behavior change.
- Keep all frontend colors/typography/shadows in `web/src/styles/theme.ts` and consume via shared primitives/utilities.
- Keep engram auto-metadata enrichment fill-empty-only: derive `abstract/tags/keywords` only when empty and never overwrite non-empty caller values.
- Keep project write resolution centralized in `ProjectService.resolve_project_id_for_write`: explicit `project_id` wins, otherwise use actor default project, otherwise fail with 422.
- Keep soft-delete semantics for sessions/engrams/collections (`deleted_at`, `deleted_by_user_id`, `delete_reason`) and restore by clearing those fields; never hard-delete from admin APIs/tools.
- Keep collections project-bounded: engrams can only belong to collections in the same project, and cross-project engram moves must auto-detach invalid collection links.
- Keep project visibility membership-safe: project-visible engrams/chats/documents must only resolve for admin, owner, or active project members (including ownerless legacy rows via active membership).
- Keep project ownership canonical: `projects.owner_user_id` remains source-of-truth and must have a mirrored active owner row in `project_members`.
- Keep project member management restricted to owner/admin; never allow assigning/demoting/removing `owner` via member CRUD APIs.
- Keep project audit events DB-backed in `project_audit_events` for member add/update/remove, engram share/unshare, and engram pin/unpin actions.
- Before every commit, follow `skills/codescene/SKILL.md` and run a CodeScene MCP pre-commit health check (`pre_commit_code_health_safeguard`) on the current change set; record the outcome in the implementation log.
- Keep skill docs aligned with the Go runtime layout (`internal/`, `cmd/api/`, `web/src/`); do not leave deprecated `api/app`/`api/tests` path references in active skills.

## 3. Daily Workflow
1. Pull latest and inspect `git status`.
2. Run setup: `make sync`, `make web-sync`.
3. Preferred local run mode: `make dev` (starts Docker DB + local `make api` and `make web` together in one terminal).
4. Use split terminals only when needed: run `make api` and `make web` separately.
5. Use `make stack-up` sparingly (containerized stack debugging only, not default day-to-day development).
6. Implement one phase at a time.
7. Run checks: `make lint`, `make test`, `make eval`, `make check`.
8. If `web/` changed: run `make web-check`.
9. If `acceptance-tests/`, `web/Dockerfile`, `Dockerfile`, `docker-compose.yml`, or auth/session workflow changed: run `make acceptance-bddgen`, `make acceptance-typecheck`, and `make acceptance-test-docker`.
10. If Bedrock provider behavior changed, run live non-deterministic acceptance gate: `make acceptance-test-bedrock-live` (or docker equivalent).
11. For docker runtime changes: validate `docker compose config`.
12. Update docs (`docs/implementation-log.md`, `migration/checkpoints/checkpoint.md`, `Plan.md` progress, `docs/architecture-playbook.md` when call flows or schema semantics change, skill docs if needed).
13. Commit with phase-scoped message.

## 4. Architecture Boundaries
- Route handlers in `internal/api` should orchestrate only.
- Domain routers should remain thin and delegate to services.
- DB access must live in repository modules.
- Context assembly logic belongs in domain context modules (`internal/chat/context.go`), not route handlers.
- Lifecycle/autosave/retention heuristics must stay in `internal/chat/lifecycle_policy.go` as pure functions so they remain testable and reusable across REST/MCP paths.
- Provider SDK calls must stay in `internal/providers/`.
- Document chunking and retrieval behavior belongs in `internal/ingestion/`.
- Embedding provider routing belongs in `internal/embeddings/`; do not call provider endpoints directly from repositories.
- MCP tool handlers should call service/repository layers, not raw SQL.
- UI should call API/MCP contracts only, not reimplement business logic.
- UI styling should use shared tokens in `web/src/styles/theme.ts`; avoid ad-hoc hardcoded palette values in components.

## 4a. Domain Module Layout Rules
- For each new domain, create a dedicated package under `internal/`:
  - data access (`internal/repository/*`)
  - domain logic (`internal/<domain>/*.go`)
  - transport boundary (`internal/api`, `internal/mcp`, or `cmd/api` adapters as needed)
- Keep shared contracts in `internal/models` only when broadly reused.
- Prefer domain-local helper modules over adding unrelated helpers to `cmd/api/main.go`.
- Add tests in matching domain-focused files (`*_test.go`).

## 5. Schema and Migration Policy
- Prefer additive, forward-only SQL changes.
- Never silently drop columns or rewrite semantics without migration notes.
- Keep DDL idempotent (`IF NOT EXISTS`, compatible alters where possible).
- Add indexes for new query paths.
- Document schema impact in `docs/implementation-log.md`.

## 6. API and Interface Policy
- For breaking changes, add new fields/routes and keep old behavior until deprecated.
- Validate all external payloads with explicit request decoding and validation helpers.
- Include stable identifiers in responses (`user_id`, `session_id`, `engram_id`).
- For write flows that allow omitted `project_id` (`/api/v1/engrams`, admin collection create, MCP organization write tools), always resolve project via `ProjectService` and keep response-level project resolution metadata where defined (`resolved_project_id`, `used_default_project`).
- Keep admin memory lists default-hidden for soft-deleted rows; expose deleted rows only with explicit `include_deleted=true`.
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
- Keep adapter selection centralized in `internal/providers/registry.go`.

## 8. MCP Tool Authoring Standards
- Tool names are verb/object and stable (`chat.create_session`).
- Inputs/outputs are JSON schema friendly.
- Enforce auth/visibility checks exactly as API routes do.
- Stream responses using JSON-RPC framed SSE events.
- Return structured error payloads with machine-parseable codes.
- Keep all tool routing in `internal/mcp`; avoid embedding tool logic in route handlers.
- Keep external MCP compatibility (`initialize`, `tools/list`, `tools/call`) aligned with direct tool methods.
- Keep MCP organization contracts stable:
  - project: `project.list`, `project.create`, `project.get_default`, `project.set_default`
  - engram admin lifecycle: `engram.list`, `engram.get`, `engram.update`, `engram.move_project`, `engram.delete`, `engram.restore`
  - collection lifecycle: `engram.collection_list`, `engram.collection_create`, `engram.collection_update`, `engram.collection_delete`, `engram.collection_add_items`, `engram.collection_remove_items`
  - session lifecycle: `chat.delete_session`, `chat.restore_session`
- Keep dotted canonical tool names as source-of-truth; continue exposing underscore aliases in `tools/list` for strict MCP client compatibility and accept both forms in `tools/call`.
- When MCP contracts change, update `web/src/api/mcpClient.ts` in the same phase.

## 9. Testing Requirements
- Unit tests for pure logic and adapters.
- Integration tests for API + DB behavior.
- Eval tests for memory quality scenarios.
- Frontend tests for UI helpers/components and session workflow logic.
- Acceptance tests for end-to-end workflow regressions (`acceptance-tests/features/*.feature`).
- Lifecycle policy changes require regression coverage across:
  - `internal/chat/lifecycle_policy_test.go`
  - `internal/api/chat_api_lifecycle_test.go`
  - `internal/mcp/compatibility_service_chat_timeline_test.go`
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
- Default dev startup: use `make dev`.
- If you need manual server control: use `make db-up`, then `make api` and `make web` in separate terminals.
- Keep `make stack-up` for occasional full-container checks only.
- For startup/debug issues: use `APP_ENV=development` and `LOG_CONFIG_IN_DEV=true` to print a redacted parsed config snapshot.
- If retrieval quality drops: run targeted Go tests for retrieval and chat context paths.
- If auth fails unexpectedly: inspect `data/audit_events.jsonl` and session settings.
- If MCP stream fails: verify SSE endpoint wiring and JSON-RPC event framing.
- If acceptance tests fail to render UI in Docker: verify Vite host allow list (`VITE_ALLOWED_HOSTS`) and proxy target (`VITE_API_PROXY_TARGET`).

## 12. Long-Run Maintenance Cadence
- Weekly: run full quality gates and evals (`make check`, `make web-check`, `make eval-report`).
- Monthly: review dependency updates and provider API changes.
- Each feature cycle: refresh `docs/implementation-log.md`, `migration/checkpoints/checkpoint.md`, `Plan.md`, and affected skill docs; update `docs/architecture-playbook.md` for call-flow or data-semantics changes.
- Keep skill instructions short, precise, and executable.

## 13. Accumulated Project Learnings

### Architecture Patterns That Worked
- Domain module layout (`internal/<domain>/` with focused package boundaries) scales well for feature isolation.
- Centralized enrichment in repository create flow ensures all create paths share behavior (REST, MCP, chat save, autosave, CLI, consolidation).
- Typed MCP client helpers alongside the server ensure contract parity and catch drift early.
- Fill-empty-only metadata enrichment preserves caller intent while enabling zero-config agent persistence.
- Token scope + allowlist + project policy model provides flexible least-privilege for external MCP agents.
- Production fail-fast config validation prevents insecure session/token/OAuth defaults from booting in `APP_ENV=production`.
- MCP transport throttling plus login lockout controls should be enforced via distributed state (`rate_limit_state`) rather than process-local memory.
- Soft-delete with `deleted_at`/`delete_reason`/`deleted_by_user_id` columns enables safe reversible operations with audit trail.
- Project-default writes are contractually explicit via `resolved_project_id` + `used_default_project` fields where supported.
- Collection/project boundaries are enforced at mutation time; cross-project engram moves purge invalid collection memberships.

### Common Pitfalls to Avoid
- Never put business logic in API route handlers — always delegate to services.
- Never use raw SQL in service or MCP layers — all DB access through repositories.
- Never hardcode color/font values in React components — always use theme tokens from `web/src/styles/theme.ts`.
- Never change MCP tool names — they are stable contracts for external agents (underscored aliases for compatibility only).
- Never permit OAuth `plain` PKCE or unprotected dynamic registration in production paths.
- SSE stream parsing must handle both `\n\n` and `\r\n\r\n` framing boundaries.
- Frontend redirect detection must handle `opaqueredirect` and `status 0` from `fetch` with manual redirect mode.
- Bedrock errors must be classified specifically (`ValidationException` vs auth vs throttle) rather than treated as generic failures.
- MCP `Accept` header negotiation matters: JSON vs SSE response depends on client preference order.

### Schema Evolution Guidelines
- The project uses a single `db/init/001_schema.sql` file with `IF NOT EXISTS` guards for idempotent bootstrap.
- When adding tables, always include indexes for query paths that will be used by repositories.
- Soft-delete patterns require `deleted_at` columns and `WHERE deleted_at IS NULL` defaults in repositories.
- Project backfill logic must be idempotent for existing data promotion.
- Use compatible `ALTER TABLE ... ADD COLUMN IF NOT EXISTS` for additive changes.

### Testing Insights
- Lifecycle policy tests require regression coverage across unit (policy helpers), integration (chat API), and MCP layers.
- Provider adapter tests should cover error classification, not just happy paths.
- Acceptance tests should use deterministic mocks for CI reliability and tagged live scenarios for provider validation.
- Frontend tests must render within `ThemeProvider` context.
- MCP contract changes need transport + JSON-RPC integration coverage plus typed client parser/transport tests.

### MCP Development Notes
- MCP tools use stable verb-object names (`chat.create_session`) — underscored variants (`chat_create_session`) for external client compatibility.
- Authorization checks for MCP tools must mirror REST API authorization exactly.
- Streaming tools emit `mcp.event` progress frames with correlation IDs.
- JSON-RPC notifications without `id` get `202` response with no body.
- `tools/list` exposes underscored names; dotted names are accepted in `tools/call` for backward compatibility.

## 14. Documentation Maintenance
- `README.md` is a slim landing page — do not add detailed content to it.
- API endpoint changes go to `docs/api-reference.md`.
- MCP tool/transport changes go to `docs/mcp-guide.md`.
- Environment variable additions go to `docs/env-reference.md`.
- Implementation log entries go to `docs/implementation-log.md`.
- Milestone completions go to `migration/checkpoints/checkpoint.md`.
- When a phase completes, update `migration/checkpoints/checkpoint.md`, `todo.md`, and `Plan.md`.
