# Engram Vault - Project Checkpoints

> Milestone tracking and phase progress.
> See [Plan.md](../../Plan.md) for full roadmap. See [todo.md](../../todo.md) for pending work. See [refactor.md](../../refactor.md) for refactoring checkpoints.

---

## Current State Summary

- **Phases 0-15 completed:** foundation, schema/storage, retrieval/rehydration, durability, chat continuity, providers, MCP, UI, acceptance, theme/UX hardening.
- **Phase 16 completed:** MCP developer tooling and typed clients (compatibility surface, typed clients, and CLI smoke utility complete).
- **Phase 17 completed:** document ingestion and RAG-ready retrieval, including session-level document pinning.
- **Phase 18 completed:** memory lifecycle policies (autosave/retention controls, timeline APIs/UI, and explicit consolidation merge/group semantics).
- **Phase 19 completed:** collaboration + sharing model (project memberships, membership-enforced visibility, share/unshare APIs + MCP tools, and DB-backed audit trail + admin UI controls).
- **Phase 20 completed:** OIDC login/session hardening plus centralized audit sink baseline, callback-state consumption hardening, and expanded OIDC/provider negative-path audit coverage.
- **Phase 21 completed:** observability/reliability foundation with request telemetry middleware, expanded `/api/v1/metrics` health categories (provider failures, stream outcomes, lifecycle traces), and provider fallback/circuit strategy.
- **Phase 22 completed:** release automation and deployment profile hardening with staged CI gates, compose profile matrix (`dev`/`acceptance`/`release-smoke`), and versioned release checklist/rollback runbook.
- **Phase 23 completed:** EvalOps + governance hardening with versioned prompt/tool/eval metadata, deterministic continuity/citation/memory-drift suite (extended in Phase 28 with `graph_trace`), delta regression gates, and historical trend artifacts.
- **Phase 24 completed:** graph foundation with `engram_links`/`engram_link_events`, indexed traversal paths, and repository baseline for create/list/update/archive/traversal.
- **Phase 25 completed:** link REST + MCP operations (`create/list/update/archive/suggest/trace`) and hybrid link suggestion pipeline.
- **Phase 26 completed (backend baseline):** graph-aware chat context assembly with bounded linked recall and trace metadata (`used_engram_link_ids`, `engram_trace_paths`).
- **Phase 27 completed:** web graph traceability UX with linked-recall controls, linked-memory panel (relation/weight/age), and suggestion accept/reject workflow.
- **Phase 28 completed:** temporal decay/reinforcement, graph hygiene recommendation API, graph-trace EvalOps coverage, scheduled hygiene auto-archival execution, and configurable noisy-link suppression controls.
- **Phase 29 completed:** optional deterministic auto-metadata enrichment and MCP conversation-only persistence path.
- **Phase 30 completed:** MCP PAT lifecycle APIs/UI plus scoped bearer authorization for external agents.
- **Phase 31 completed:** project defaults + enterprise memory management + MCP organization (backend/API/MCP/web/tests/docs/skills complete with acceptance mock validation).
- **Phase 33 completed:** portable export/import stash workflow with REST + MCP + web transfer flows, owner/admin authorization, and source-fidelity round-trip coverage.
- **Phase 34 completed:** security audit remediation for production-safe config defaults, distributed login/MCP throttling, protected OAuth registration, and security regression coverage.
- **Go migration initiated (2026-02-22):** phased Python/FastAPI → Go migration started with dedicated progress tracker in `checkpoint-go-migration.md` (CP1 complete: module scaffold + config parity tests; CP2 complete: DB bootstrap/transaction parity tests; CP3 complete: API scaffold + health/version route parity tests; CP4 complete: auth hashing/CSRF parity tests; CP5 complete: user models/repository parity tests; CP6 complete: embeddings local/fallback parity tests; CP7 complete: engram repository helper/query parity tests; CP8 complete: DB read-path parity for `list_engrams`/`query_engrams`; CP9 complete: rehydration/source read-path parity; CP10 complete: engram write-path repository flows; CP11 complete: chat session repository baseline; CP12 complete: project repository baseline; CP13 complete: chat message/session pinning repository continuation; CP14 complete: document repository baseline; CP15 complete: MCP token repository baseline; CP16 complete: OAuth repository baseline; CP17 complete: collection repository baseline; CP18 complete: memory-admin session repository baseline; CP19 complete: memory-admin engram repository baseline; CP20 complete: memory-admin engram update/source-replacement repository parity; CP21 complete: memory-admin service baseline; CP22 complete: memory-admin API route baseline; CP23 complete: dependency-aware memory-admin API integration baseline; CP24 complete: runtime dependency wiring with migration-time actor/project resolver bridges; CP25 complete: context-first admin actor hardening baseline; CP26 complete: session-cookie actor middleware baseline with DB-backed canonicalization; CP27 complete: session login/logout/csrf route baseline with `/api/v1/me`; CP28 complete: session hardening baseline for TTL/issued-at validation and secure cookie attributes; CP29 complete: UI/login parity baseline (`/`, `/login`, `/logout`, `/ui`) on hardened sessions; CP30 complete: login guard + auth audit parity baseline in Go runtime/UI flow; CP31 complete: distributed limiter parity baseline with `rate_limit_state` store wiring and local fallback hardening; CP32 complete: UI/admin auth integration hardening with `/ui/admin` role-gating parity and auth/session code-health uplift; CP33 complete: role-aware `/api/v1/users` API parity + auth/session route health uplift; CP34 complete: session-auth engram route parity (`/api/v1/engrams*`) with strict >9.5 code-health gate; CP35 complete: non-checkpoint code-health uplift for `internal/auth/session.go` from 9.38 to 9.68; CP36 complete: non-checkpoint code-health uplift for `internal/repository/user.go` from 9.38 to 10.0; CP37 complete: non-checkpoint code-health uplift for `internal/embeddings/service.go` from 9.09 to 9.68; CP38 complete: non-checkpoint code-health uplift for `internal/repository/oauth.go` from 9.38 to 10.0; CP39 complete: non-checkpoint code-health uplift for `internal/repository/chat_test.go` from 9.09 to 10.0; CP40 complete: non-checkpoint code-health uplift for `internal/repository/chat.go` from 9.02 to 9.68; CP41 complete: non-checkpoint code-health uplift for `internal/repository/oauth_test.go` from 9.38 to 10.0; CP42 complete: non-checkpoint code-health uplift for `internal/repository/collection_test.go` from 9.09 to 10.0; CP43 complete: non-checkpoint code-health uplift for `internal/repository/document_test.go` from 8.72 to 9.68; CP44 complete: non-checkpoint code-health uplift for `internal/repository/document.go` from 8.81 to 9.68; CP45 complete: non-checkpoint code-health uplift for `internal/repository/chat_pinning.go` from 8.81 to 9.68; CP46 complete: non-checkpoint code-health uplift for `internal/repository/engram_write_test.go` from 9.26 to 10.0; CP47 complete: non-checkpoint code-health uplift for `internal/repository/admin_engram_update_test.go` from 9.25 to 10.0; CP48 complete: non-checkpoint code-health uplift for `internal/repository/admin_engram_test.go` from 9.38 to 10.0; CP49 complete: non-checkpoint code-health uplift for `internal/admin/service_test.go` from 9.38 to 10.0; CP50 complete: non-checkpoint code-health uplift for `internal/admin/service.go` from 8.54 to 9.68; CP51 complete: non-checkpoint code-health uplift for `internal/repository/chat_pinning_test.go` from 9.38 to 9.51).
- **Go migration current checkpoint (2026-03-01):** CP189 completed in `checkpoint-go-migration.md` with Phase 22 release automation closure (CI backend/web/acceptance/release-smoke gates, optional workflow-dispatched live-provider release gate, compose profile matrix, and versioned release checklist/rollback docs) validated with compose config checks plus `go test ./...`.

---

## Active Phase Progress

### Phase 20 Progress Tracker

- [x] Added optional OIDC login/session mapping (`/login/oidc`, `/login/oidc/callback`) with verified ID-token flow in Go runtime.
- [x] Added OIDC runtime config validation + redaction behavior (`OIDC_*` settings).
- [x] Added/updated tests for OIDC login start/callback routes and config validation.
- [x] Added centralized audit sink strategy and implementation path beyond JSONL/stdout baseline.
- [x] Extend security regression and acceptance coverage for OIDC/provider negative paths.

#### Phase 20 Implemented So Far

1. OIDC runtime provider module with discovery, auth URL generation, code exchange, ID-token validation, nonce enforcement, and claim-to-username mapping.
2. Session UI OIDC routes with pending state/nonce storage in signed session cookies and final session-authenticated login mapping.
3. Config and env updates for `OIDC_ENABLED`, issuer/client/redirect/scopes/username-claim controls with validation/redaction coverage.
4. Audit runtime supports optional centralized sink delivery with `AUDIT_SINK_URL`, `AUDIT_SINK_AUTH_TOKEN`, `AUDIT_SINK_REQUIRED`, and `AUDIT_SINK_TIMEOUT_SECONDS`.
5. Route and config test coverage for OIDC happy-path + not-enabled behavior, plus audit sink validation and delivery semantics.
6. OIDC callback hardening now consumes pending nonce/state on validated callback attempts and records explicit failure audits for provider verification and identity-to-user mapping failures.

### Phase 21 Progress Tracker

- [x] Added structured request telemetry middleware with per-request domain/route/status/duration logging.
- [x] Added process-local in-memory request metrics recorder and `/api/v1/metrics` route.
- [x] Added router/middleware tests for metrics aggregation and route pattern tracking.
- [x] Add stream-health and provider-failure category metrics.
- [x] Add trace hooks for chat/session/message lifecycle.
- [x] Add provider fallback/circuit-breaker strategy and regression coverage.

### Phase 23 Progress Tracker

- [x] Added deterministic EvalOps suite for `continuity`, `citation_trust`, and `memory_drift`.
- [x] Added governance version metadata in chat responses, stream payloads, MCP initialize policy, and version endpoint.
- [x] Added baseline + previous-run delta regression gate for eval scores.
- [x] Added run history persistence and markdown trend reporting.
- [x] Integrated eval gate into `make check`, `make release-gate`, and CI Go backend checks.
- [x] Extended EvalOps in Phase 28 with `graph_trace` dimension coverage while preserving v1 governance contracts.

### Phase 24 Progress Tracker

- [x] Added `engram_links` table with lifecycle metadata and guardrail constraints.
- [x] Added optional `engram_link_events` table for link lifecycle/audit transitions.
- [x] Added project/source/target/status/relation/recency indexes for graph retrieval paths.
- [x] Added repository baseline methods for create/list/update/archive and depth-limited traversal.
- [x] Added visibility-safe traversal/list filters and score-aware ordering.
- [x] Complete runtime integration proof and phase closeout docs.

### Phase 25 Progress Tracker

- [x] Added session-auth REST link routes (`create/list/update/archive/suggest/trace`).
- [x] Added MCP link tools (`engram.link_*`, `engram.trace_path`) with token-policy project scoping.
- [x] Added hybrid suggestion service (semantic + source overlap + lexical continuity + recency).
- [x] Added regression coverage for REST, MCP dispatch, repository access, and runtime wiring.

### Phase 26 Progress Tracker

- [x] Added graph-aware recall in chat context assembler (default depth `1`, bounded traversal).
- [x] Added configurable recall controls: `link_recall_enabled`, `link_recall_depth`, `link_recall_max_neighbors`.
- [x] Added linked-context response metadata: `used_engram_link_ids`, `engram_trace_paths`.
- [x] Added REST + MCP send-message override support for link-recall controls.
- [x] Added unit/integration coverage and CodeScene-safe refactors for new graph recall paths.

### Phase 27 Progress Tracker

- [x] Added chat-composer linked-recall controls (enable toggle, depth, max neighbors).
- [x] Added transcript traceability strip from `used_engram_link_ids` + `engram_trace_paths`.
- [x] Added web API/client/hook typing and state propagation for linked trace metadata in send/stream flows.
- [x] Added linked engram relationship panel with relation type, weight, and age.
- [x] Added link suggestion accept/reject workflow in the web UI.
- [x] Added richer "this answer used" explainability chain unifying links and citations.

### Phase 28 Progress Tracker

- [x] Added temporal decay to effective link-quality scoring in graph-aware context assembly.
- [x] Added successful-session link reinforcement (`temporal_weight` boost + `last_reinforced_at` updates).
- [x] Added suggested-link promotion to `active` when reinforced through successful usage.
- [x] Added graph hygiene recommendation engine + route (`/api/v1/engrams/{engram_id}/links/hygiene`) for duplicate/conflict/stale-low-value detection.
- [x] Added graph-focused eval coverage (`trace correctness`, `relevance impact`, `drift tolerance`) via `graph_trace` fixtures and checks (`used_engram_link_count`, `required_trace_targets`).
- [x] Added scheduled hygiene execution/auto-archival policy around recommendation outputs (per-source cadence with stale-low-value recommendation archival on successful reinforcement paths).
- [x] Added configurable noisy-link suppression thresholds (policy defaults + bounded request overrides) with chat context suppression filtering.

### Phase 31 Progress Tracker

- [x] `Plan.md` updated with Phase 31 status and near-term execution order.
- [x] Added first-class `projects` table and per-user default project persistence.
- [x] Added default-project fallback when `project_id` is missing in engram create paths.
- [x] Added admin memory router (`/api/v1/admin/memory`) for session/engram/collection management.
- [x] Added MCP organization tools for project, engram, collection, and session lifecycle operations.
- [x] Added dedicated admin memory UI page with routing (`/admin/memory`).
- [x] Hardened `/api/v1/engrams*` with authenticated actor-scoped visibility.
- [x] Added backend/web/acceptance tests for Phase 31 behavior.
- [x] Update `AGENT.md` + skills docs and append final phase-closeout validation evidence.
- [x] Validation evidence: `make acceptance-test-mock` (2026-02-22) -> `18 passed`, including `@phase31 @memory-admin` scenarios.

#### Planned Phase 31 User-Facing Areas

- **Project defaults:** set default project once and reuse it when `project_id` is omitted; preserve explicit `project_id` when provided.
- **Memory management page:** separate admin route for listing, editing, moving, deleting, and restoring sessions/engrams; project-bounded collections.
- **MCP organization tools:** agent-driven memory organization from chat/tool calls with scoped authorization.

#### Phase 31 Implemented So Far

1. Schema: `projects` table, `users.default_project_id` FK, soft-delete metadata, `engram_collections`/`engram_collection_items`, idempotent project backfill.
2. Backend: `api/app/projects/`, `api/app/memory_admin/`, wired into `api/app/main.py`.
3. API: engrams require auth, default-project resolution, `resolved_project_id`/`used_default_project` in responses.
4. MCP: project/engram/collection/session tools with dotted aliases for backward compatibility.
5. Web: routing (`/`, `/admin/memory`), default-project controls, admin page list/filter/edit/move/delete/restore workflows.
6. Tests: backend integration, MCP extensions, web unit/integration, acceptance mock coverage.

---

## Completed Milestones

### Milestone 1 — Local DB + Schema + API Skeleton
- Local DB + schema created
- API skeleton created
- CRUD + query + rehydrate endpoints wired

### Milestone 2 — UI Login Workflow
- Added local UI login workflow for manual testing
- Added authenticated dashboard for create/list/query/rehydrate API calls
- Added UI auth tests for login/logout/session redirects

### Milestone 3 — LangGraph Durability
- LangGraph run checkpointing integrated
- Automatic engram write at end of each research run
- Optional periodic snapshot engrams for long-running threads

### Milestone 4 — Auth Hardening + Provenance
- CSRF protection for login/logout UI forms
- Optional hashed-password authentication path
- Source-inspection endpoint and dashboard workflow
- Multi-user auth model and role-based access controls

### Milestone 5 — Evaluation Harness
- Local eval harness: fact recall, cross-engram reasoning proxy, temporal updates, abstention checks

### Milestone 6 — CLI Workflow
- Local CLI: upload engrams from JSON, query/search from terminal, rehydrate bundles

### Milestone 7 — Retrieval Quality
- Reranking (dense + lexical overlap)
- Citation-packing in rehydration context

### Milestone 8 — Memory Maintenance
- Background consolidation jobs

### Milestone 9 — Security Baseline (Completed)
- Completed: local audit event logging, login rate limiting + lockout guard
- Completed in later phases: OIDC, centralized audit pipeline, distributed auth rate limits

### Milestone 10 — Schema + Repository Layer
- Chat/session/pinning tables, engram ownership/visibility fields, visibility-aware repository filtering

### Milestone 11 — Provider Adapter Layer
- OpenAI, Anthropic, Bedrock adapters with normalized mapping
- Provider registry and configuration contract

### Milestone 12 — Chat API + Continuity
- Session/message endpoints, stream endpoint, save-as-engram, continue-session flows
- Context assembly with `used_engram_ids`, `used_engram_link_ids`, `engram_trace_paths`, and `source_references`

### Milestone 13 — MCP HTTP Stream
- JSON-RPC over SSE transport, chat/engram/user tool routing
- Auth/visibility parity with REST APIs, structured error frames

### Milestone 14 — React Chat UI
- Session list/create, streaming transcript, pin/save/continue controls
- Frontend test baseline and `make web-check`

### Milestone 15 — Acceptance + UX Hardening
- Playwright-BDD acceptance baseline for login/chat/continuation flows
- Bedrock live and triage continuity scenarios
- Markdown rendering, stream parsing, dark/light theming, sidebar UX

### Milestone 16 — MCP Developer Experience (Completed)
- MCP compatibility methods: `initialize`, `tools/list`, `tools/call`
- Typed Python and TypeScript MCP client helpers
- Completed: CLI smoke utility (`engram-cli mcp-call`) for local MCP transport/tool debugging.

### Milestone 17 — RAG-Ready Ingestion
- File/document upload, deterministic chunking, retrieval blending
- Session-level document pin/unpin and continuation carry-forward

### Milestone 18 — Memory Lifecycle (Completed)
- Session-level autosave strategies, retention windows, pruning
- Lifecycle policy + timeline APIs + MCP tools + UI
- Completed: explicit consolidation merge/group timeline semantics in lifecycle timeline rendering.

### Milestone 29 — Auto-Metadata Enrichment
- Deterministic fill-empty-only derivation for `abstract`, `tags`, `keywords`
- Repository-level centralization across all create paths
- MCP `engram.create_from_conversation` for conversation-only persistence

### Milestone 33 — Portable Export/Import Stash (Completed)
- REST project export/import endpoints with deterministic conflict handling
- MCP `project.export_bundle` and `project.import_bundle` tools
- Transfer page export/import workflow with searchable project/collection IDs
- Round-trip tests for collection membership and engram source fidelity

### Milestone 34 — Security Audit Remediation Program (Completed)
- Production fail-fast validation for insecure secrets/defaults and protected OAuth registration.
- Distributed rate limiting for login lockouts and MCP transport request bursts.
- OAuth PKCE hardening (`S256` only) plus admin-protected dynamic client registration.
- Ingestion and audit hardening (metadata size cap, file type validation, audit payload sanitization/truncation).
- Security-focused API regression coverage and acceptance lockout scenario.

### Milestone 19 — Collaboration (Completed)
- Project membership model with `owner`/`editor`/`viewer` roles and owner-membership backfill.
- Membership-enforced visibility across engram/chat/document read paths.
- Project member CRUD + project audit REST endpoints.
- MCP tool parity for member management and engram share/unshare.
- DB-backed audit events for member changes, share/unshare, and chat pin/unpin.
- Admin memory page member management + audit timeline panels.

### Milestone 20 — Production Security (Completed)
- Completed: OIDC login/session mapping baseline, OIDC config validation/redaction coverage, centralized audit sink rollout, and expanded security abuse-path coverage.

---

## Phase Completion Timeline

| Phase | Name | Status |
|-------|------|--------|
| 0-9 | Foundations through Security Baseline | Completed |
| 10 | Schema + Repository Layer | Completed |
| 11 | Provider Adapter Layer | Completed |
| 12 | Chat API + Continuity | Completed |
| 13 | MCP HTTP Stream | Completed |
| 14 | React Chat UI | Completed |
| 15 | Acceptance + UX Hardening | Completed |
| 16 | MCP Developer Experience | Completed |
| 17 | Document Ingestion (RAG) | Completed |
| 18 | Memory Lifecycle Policies | Completed |
| 29 | Auto-Metadata Enrichment | Completed |
| 30 | MCP Personal Access Tokens | Completed |
| 31 | Enterprise Memory Management | Completed |
| 19 | Collaboration + Sharing | Completed |
| 20 | Production Security | Completed |
| 21 | Observability and Reliability | Completed |
| 22 | Release Automation and Deployment Profiles | Completed |
| 23 | EvalOps and Prompt/Policy Governance | Completed |
| 24 | Engram Graph Foundations | Completed |
| 25 | Link APIs, MCP Tools, Suggestion Pipeline | Completed |
| 26 | Graph-Aware Context Assembly | Completed (backend baseline) |
| 27 | Memory Graph UX and Traceability | Completed |
| 28 | Temporal Dynamics and Graph Quality Controls | Completed |
| 32 | ContinuWitty Query Protocol | Planned |
| 33 | Portable Export/Import Stash | Completed |
| 34 | Security Audit Remediation Program | Completed |
