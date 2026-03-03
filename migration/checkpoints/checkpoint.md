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
- **Phase 32 completed:** `cw>` query protocol + federated linked recall shipped with parser/runtime metadata (`cw_plan_applied`), cross-project traversal/ranking/context packing, and retrieval-audit telemetry (`retrieval_audit`) across REST/MCP send + stream payloads.
- **Phase 33 completed:** portable export/import stash workflow with REST + MCP + web transfer flows, owner/admin authorization, and source-fidelity round-trip coverage.
- **Phase 34 completed:** security audit remediation for production-safe config defaults, distributed login/MCP throttling, protected OAuth registration, and security regression coverage.
- **Phase 35 completed:** memory engagement tracking baseline (engram access counters/events, chat success-path access recording hooks, and non-blocking telemetry failure handling).
- **Phase 36 completed:** explicit feedback loop shipped (storage, API/MCP feedback submission, rerank feedback signal integration) with calibrated engagement/freshness weighting and latency benchmark notes.
- **Phase 37 completed:** freshness maintenance plus consolidation schema/repository/admin-API/MCP parity shipped (refresh/list/action API + MCP tools + tests), with deterministic acceptance + precision/recall benchmark coverage for grouping criteria.
- **Phase 38 completed:** contradiction warning baseline, contradiction-alert persistence/review workflows, and warning-quality benchmark coverage are shipped (trace metadata + chat/send/stream/MCP warning parity, contradiction-alert storage + refresh/list/resolve repository workflows, admin/API/MCP maintenance routes, and deterministic precision/recall acceptance coverage enabled in default `@mock` runs).
- **Phase 39 completed:** temporal query + cost-aware context assembly shipped with engagement filters (`access_count_min`, `freshness_score_min`), recall/timeline windows (`last_accessed_after/before`, `freshness_computed_after/before`), trace-aware constraints (`relation_type`, `trace_depth`), and bounded `context_token_budget` controls across REST/MCP chat send paths with retrieval-audit budget diagnostics (`context_token_budget`, `context_token_estimate`, `context_token_truncated`).
- **Phase 40 completed:** autonomous memory suggestions now include schema/model/repository foundations, admin REST + MCP curation maintenance parity (`engram.curation_refresh_links`, `engram.curation_list`, `engram.curation_action`), deterministic generation hooks from consolidation/contradiction refresh workflows plus scheduled and on-demand link-hygiene link-suggestion persistence, `status=applied` downstream orchestration for consolidation/contradiction/link payloads, mock acceptance coverage for curation generation/action flows, and benchmark baselines for curation action/sync latency.
- **Phase 41 completed:** explicit feedback enrichment now supports optional `relevance_score`, optional `session_id`, and optional `integration_depth` attribution across REST/MCP feedback submission with persisted `feedback_count` and `avg_relevance_feedback` aggregate tracking.
- **Phase 42 completed:** session-authority scoring baseline shipped with bounded `source_session_quality_score` schema/indexing, authority-aware rerank integration, and deterministic feedback-driven authority calibration on relevance-scored feedback events.
- **Phase 43 completed:** authority-aware query filtering now supports `source_session_quality_min` across REST/MCP/repository contracts with validation and catalog metadata parity.
- **Phases 44-56 completed:** authority fallback transparency shipped (`source_session_quality_score` in query responses) and query quality controls now include `avg_relevance_feedback_min`, `contradiction_count_max`, `contradiction_feedback_ratio_max`, `feedback_count_min`, `feedback_count_max`, `useful_count_min`, `useful_count_max`, and `useful_feedback_ratio_min` filters plus result diagnostics for `feedback_count`, `contradiction_count`, `contradiction_feedback_ratio`, `access_count`, `freshness_score`, `useful_count`, `avg_relevance_feedback`, and `useful_feedback_ratio`.
- **Technical debt follow-ups completed (2026-03-01):** OIDC rollout validation now includes centralized audit-sink regression coverage for callback failure events, and cross-provider engram reuse validation now has explicit chat fallback coverage asserting identical engram-context reuse and provenance metadata across provider/model switches.
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
- [x] Validation evidence: `make acceptance-test-mock` (2026-03-01) -> `20 passed`, including `@phase31 @memory-admin` scenarios.

#### Planned Phase 31 User-Facing Areas

- **Project defaults:** set default project once and reuse it when `project_id` is omitted; preserve explicit `project_id` when provided.
- **Memory management page:** separate admin route for listing, editing, moving, deleting, and restoring sessions/engrams; project-bounded collections.
- **MCP organization tools:** agent-driven memory organization from chat/tool calls with scoped authorization.

#### Phase 31 Implemented So Far

1. Schema: `projects` table, `users.default_project_id` FK, soft-delete metadata, `engram_collections`/`engram_collection_items`, idempotent project backfill.
2. Backend: `internal/projects/`, `internal/admin/`, and `internal/api/admin_memory*.go` wired through `cmd/api/main.go`.
3. API: engrams require auth, default-project resolution, `resolved_project_id`/`used_default_project` in responses.
4. MCP: project/engram/collection/session tools with dotted aliases for backward compatibility.
5. Web: routing (`/`, `/admin/memory`), default-project controls, admin page list/filter/edit/move/delete/restore workflows.
6. Tests: backend integration, MCP extensions, web unit/integration, acceptance mock coverage.

### Phase 32 Progress Tracker

- [x] Added `cw>` directive parser and normalized query-plan contract (`internal/chat/query_protocol.go`).
- [x] Integrated parser into runtime preparation so directive prefixes are removed from persisted/context query text.
- [x] Added additive `cw_plan_applied` metadata in chat send/stream payloads and MCP send responses.
- [x] Added parser/runtime tests for directive forms and metadata propagation.
- [x] Added access-aware federated cross-project graph traversal/read baseline in link repository visibility and traversal SQL.
- [x] Added ranking/context-packing fusion tuning for federated traversal candidates with resilient rehydration backfill in retrieval assembly.
- [x] Added blocked-node filtering telemetry/audit signals and cross-project integration coverage.

### Phase 35 Progress Tracker

- [x] Updated roadmap alignment in `Plan.md` with explicit mapping from `docs/memory-improvements.md` into Phases 35-40.
- [x] Added schema baseline for engagement telemetry:
  - `engrams.access_count`, `engrams.last_accessed_at`, `engrams.useful_count`, `engrams.contradiction_count`
  - `engram_access_events` table + access-focused indexes.
- [x] Added repository write path (`RecordEngramAccessEvents`) to append events and update aggregate counters.
- [x] Added chat runtime integration for successful send/stream flows with non-blocking failure behavior:
  - records used engram accesses post-success lifecycle completion.
  - emits lifecycle trace metadata for recorder success/failure.
- [x] Added tests:
  - repository tests for event normalization/persistence and aggregate update invocation.
  - chat service tests for send/stream access recording, dedupe behavior, and side-effect failure isolation.
- [x] Validation:
  - `go test ./...` passed.
  - CodeScene `pre_commit_code_health_safeguard` quality gates passed.
- [x] Code health uplift:
  - refactored active chat service surface into focused helper modules and validated `10.0` scores for all touched chat service code files.
- [x] Added target-state guide:
  - `docs/continuwitty-target-state.md` documents full-phase ContinuWitty value, role workflows, KPI framework, and rollout model.

### Phase 36 Progress Tracker

- [x] Added explicit feedback storage + aggregation:
  - `engram_feedback` table.
  - aggregate updates for `useful_count` and `contradiction_count`.
- [x] Added feedback submission surfaces:
  - REST: `POST /api/v1/engrams/{engram_id}/feedback`.
  - MCP: `engram.feedback` / `engram_feedback`.
- [x] Added feedback-aware retrieval rerank signal.
- [x] Added engagement/freshness weighting calibration and phase latency benchmark notes:
  - composite rerank now includes engagement (`access_count`) and freshness (`freshness_score`) factors with deterministic weights.
  - latency benchmark notes captured in `docs/phase36-relevance-calibration.md`.

### Phase 37 Progress Tracker

- [x] Added schema baseline for freshness maintenance:
  - `engrams.freshness_score`.
  - `engrams.freshness_last_computed_at`.
- [x] Added repository maintenance path:
  - `RefreshEngramFreshnessScores` with half-life decay and optional project scope.
- [x] Added admin API trigger route:
  - `POST /api/v1/admin/memory/engrams/freshness/refresh`.
- [x] Added MCP maintenance trigger tool:
  - `engram.refresh_freshness` / `engram_refresh_freshness` (admin-only).
- [x] Added consolidation suggestion schema/services baseline:
  - `engram_consolidation_suggestions` schema + indexes.
  - repository refresh/list routines for exact-duplicate suggestions.
  - repository/model regression coverage.
- [x] Added admin memory REST consolidation tooling baseline:
  - `POST /api/v1/admin/memory/engrams/consolidation/refresh`.
  - `GET /api/v1/admin/memory/engrams/consolidation/suggestions`.
- [x] Added MCP consolidation tooling parity baseline:
  - `engram.refresh_consolidation` / `engram_refresh_consolidation`.
  - `engram.consolidation_list` / `engram_consolidation_list`.
- [x] Added admin memory REST consolidation action baseline:
  - `POST /api/v1/admin/memory/engrams/consolidation/suggestions/{suggestion_id}/action`.
- [x] Added MCP consolidation action parity:
  - `engram.consolidation_action` / `engram_consolidation_action`.
- [x] Added acceptance/benchmark coverage for grouping criteria:
  - deterministic `@phase37` acceptance scenario for refresh/list/action quality checks.
  - documented precision/recall benchmark fixture and thresholds in `docs/phase37-consolidation-benchmark.md`.

### Phase 38 Progress Tracker

- [x] Added contradiction-trace metadata in link recall assembly:
  - `engram_trace_paths[].has_contradiction`
  - `engram_trace_paths[].contradicting_link_ids`
- [x] Added contradiction warning synthesis in chat context assembly:
  - `contradiction_warnings` with severity guidance (`high`/`medium`/`low`).
- [x] Added send/stream/MCP payload parity for contradiction warnings:
  - chat send response.
  - stream `meta` + `done` payloads.
  - MCP `chat.send_message` response parity adapter.
- [x] Added regression coverage for contradiction warning generation and propagation.
- [x] Added contradiction alert persistence repository baseline:
  - schema + indexes in `engram_contradiction_alerts`.
  - deterministic refresh/list/resolve repository primitives with project scoping.
  - repository/model test coverage for refresh/list/resolve + status parsing.
- [x] Added contradiction review tooling via API/MCP:
  - admin routes for refresh/list/resolve contradiction alerts.
  - MCP tool parity:
    - `engram.refresh_contradictions`
    - `engram.contradiction_list`
    - `engram.contradiction_resolve`
  - regression coverage across admin routes, adapters, MCP parser/dispatch paths.
- [x] Added contradiction warning synthesis latency benchmark baseline:
  - chat benchmark coverage for `buildContradictionWarnings` at 50/200 path workloads.
  - benchmark results + runbook documented in `docs/phase38-contradiction-benchmark.md`.
- [x] Added warning-quality benchmark coverage:
  - deterministic acceptance precision/recall scenario for contradiction refresh/list/resolve flow in `acceptance-tests/features/phase38-contradiction-mock.feature`.
  - contradiction-link create path fixed by aliasing source/target CTE relations in `internal/repository/engram_links.go`, enabling scenario inclusion in default `@mock` suite.

### Phase 39 Progress Tracker

- [x] Added temporal query filters beyond created-at bounds:
  - `access_count_min`
  - `freshness_score_min`
  - repository + MCP parser/catalog/test coverage updated for new filters.
- [x] Added recall/timeline temporal windows:
  - `last_accessed_after` / `last_accessed_before`
  - `freshness_computed_after` / `freshness_computed_before`
  - REST decode validation + MCP parser/catalog parity + repository predicates.
- [x] Added trace-aware query constraints:
  - `relation_type`
  - `trace_depth` (depth-1 trace constraint baseline)
  - REST/MCP parser/catalog parity + repository `engram_links` filter support.
- [x] Added bounded cost-aware context controls in chat send paths:
  - optional `context_token_budget` in REST/MCP send-message payloads.
  - bounded context-section assembly with deterministic estimated-token truncation.
- [x] Added retrieval-audit budget diagnostics:
  - `context_token_budget`
  - `context_token_estimate`
  - `context_token_truncated`
- [x] Added regression coverage for context-budget normalization/truncation and REST/MCP forwarding.
- [x] Add trace-aware temporal filters and expanded contract coverage.
- [x] Expand benchmark coverage for complex temporal/engagement/trace filter combinations.
  - repository benchmark suite now includes:
    - `BenchmarkBuildEngramQueryWhereComplexTemporalEngagementTrace`
    - `BenchmarkBuildEngramQueryWhereComplexFilterMatrix`
  - benchmark artifact: `docs/phase39-query-benchmark.md`.

### Phase 40 Progress Tracker

- [x] Added `memory_curation_suggestions` schema baseline:
  - new table + status/type checks.
  - project/status and session/type indexes.
- [x] Added curation suggestion model contracts:
  - suggestion type/status enums + parse helpers.
  - persisted record payload shape.
- [x] Added repository persistence baseline:
  - `CreateMemoryCurationSuggestion`
  - `ListMemoryCurationSuggestions`
  - `ApplyMemoryCurationSuggestionAction`
- [x] Added repository regression coverage for create/list/action and invalid status paths.
- [x] Added API + MCP routes/tools for memory curation suggestion workflows.
  - REST admin routes:
    - `POST /api/v1/admin/memory/engrams/{engram_id}/links/curation/refresh`
    - `GET /api/v1/admin/memory/engrams/curation/suggestions`
    - `POST /api/v1/admin/memory/engrams/curation/suggestions/{suggestion_id}/action`
  - MCP tools:
    - `engram.curation_refresh_links`
    - `engram.curation_list`
    - `engram.curation_action`
  - Added route/service/dispatch/catalog/token-policy regression coverage.
  - `engram.curation_refresh_links` catalog metadata now exposes optional scoped `project_id` in `tools/list` schema.
- [x] Added deterministic suggestion-generation workflow hooks.
  - consolidation refresh now rebuilds `consolidate` curation suggestions.
  - contradiction refresh now rebuilds `contradiction` curation suggestions.
  - admin link-curation refresh now persists deduped `link` curation suggestions from on-demand hygiene recommendations.
  - scheduled link-hygiene execution now persists `link` curation suggestions for non-auto-archived recommendations.
  - scheduled link-hygiene suggestion persistence dedupes pending rows by payload identity (`link_id`, `target_engram_id`, `suggested_action`).
  - stale `suggested` rows are reset per type/project before regeneration.
- [x] Added curation apply-action orchestration for actionable suggestion types.
  - `status=applied` now dispatches downstream operations before curation status update.
  - `consolidate` payloads trigger consolidation `merged` transitions.
  - `contradiction` payloads trigger contradiction-alert `resolved` transitions.
  - `link` payloads trigger link archival for archive-oriented hygiene actions and `review_relation_conflict`.
  - payload parse failures now return explicit bad-request errors in REST/MCP action flows.
- [x] Added acceptance coverage for curation generation/action quality.
  - deterministic `@phase40 @mock` scenarios validate:
    - curation generation via consolidation + contradiction refresh flows.
    - curation type coverage (`consolidate`, `contradiction`).
    - curation action transitions (`suggested` -> `accepted`, `suggested` -> `applied`) and audit fields.
    - applied-status downstream effects:
      - consolidation suggestion status transitions to `merged`.
      - contradiction alert status transitions to `resolved`.
    - link-hygiene refresh generates actionable `link` suggestions and `status=applied` action succeeds.
    - link-hygiene refresh rejects mismatched scoped `project_id` with explicit bad-request behavior.
- [x] Added scoped project hardening for link-curation refresh.
  - REST + MCP refresh paths now accept optional `project_id`.
  - refresh rejects mismatched scope (`project_id` != source engram project) with explicit bad-request mapping.
- [x] Added benchmark coverage for curation suggestion action latency, applied-side-effect orchestration, and sync scaling.
  - benchmark suite:
    - `BenchmarkActionMemoryCurationSuggestion`
    - `BenchmarkActionMemoryCurationSuggestionAppliedConsolidate`
    - `BenchmarkActionMemoryCurationSuggestionAppliedContradiction`
    - `BenchmarkSyncConsolidationCurationSuggestions50Candidates`
    - `BenchmarkSyncConsolidationCurationSuggestions200Candidates`
  - benchmark artifact: `docs/phase40-curation-benchmark.md`

### Phase 41 Progress Tracker

- [x] Added schema/model support for richer feedback signals:
  - `engrams.feedback_count`.
  - `engrams.avg_relevance_feedback`.
  - `engram_feedback.relevance_score` (optional, 1-5).
  - `engram_feedback.session_id` (optional, FK to `chat_sessions`).
  - `engram_feedback.integration_depth` (optional enum: `mentioned|elaborated|contradicted|ignored`).
- [x] Extended feedback repository write path:
  - feedback events now persist optional `relevance_score`, optional `session_id`, and optional `integration_depth`.
  - engram aggregates now update `feedback_count` and `avg_relevance_feedback` deterministically.
- [x] Added REST + MCP feedback payload parity:
  - REST and MCP feedback requests now accept optional `relevance_score` with `1-5` validation, optional `session_id` UUID, and optional `integration_depth`.
  - MCP `engram.feedback` catalog schema now documents `relevance_score`, `session_id`, and `integration_depth`.
- [x] Added regression coverage:
  - repository tests for relevance score/session/integration-depth persistence + validation and aggregate returns.
  - API route tests for relevance-score/session/integration-depth forwarding and validation.
  - MCP compatibility tests for relevance-score/session/integration-depth forwarding and validation.

### Phase 42 Progress Tracker

- [x] Added schema support for source-session authority scoring:
  - `engrams.source_session_quality_score` (`0.0-1.0`, default `0.5`).
  - idempotent check constraint + `engrams_source_session_quality_idx`.
- [x] Added retrieval authority signal integration:
  - engram query candidate selection now reads `source_session_quality_score`.
  - composite rerank now includes authority weighting.
- [x] Added deterministic feedback-driven authority calibration:
  - feedback write path updates `source_session_quality_score` when `relevance_score` is provided.
- [x] Added regression coverage:
  - rerank unit test for authority signal ordering.
  - query-shape assertion for `source_session_quality_score` selection.
  - feedback SQL regression assertion for authority-score update path.

### Phase 43 Progress Tracker

- [x] Added authority-threshold query contract:
  - `source_session_quality_min` on `models.EngramQueryRequest`.
- [x] Added REST validation parity:
  - `source_session_quality_min` now validates as bounded `0..1`.
- [x] Added repository query predicate support:
  - `COALESCE(source_session_quality_score, 0.5) >= ...` filter path.
- [x] Added MCP parser/catalog parity:
  - `engram.query` now accepts and validates `source_session_quality_min`.
  - tool schema now documents `source_session_quality_min`.
- [x] Added regression coverage:
  - REST invalid-filter test and parsed-request assertion for authority filter.
  - repository where-clause/params assertions for authority predicate.
  - MCP parity + validation tests for authority filter handling.

### Phase 44 Progress Tracker

- [x] Added fallback authority calibration for feedback writes:
  - `source_session_quality_score` updates now use normalized `relevance_score` when present.
  - fallback authority mapping now applies from `integration_depth` when `relevance_score` is omitted.
  - feedback-type fallback now applies when both relevance and integration depth are absent.
- [x] Added authority-score visibility in query response contracts:
  - `models.EngramQueryResult` now includes `source_session_quality_score`.
  - repository query result mapping now forwards `source_session_quality_score` to API/MCP callers.
- [x] Added regression coverage:
  - repository authority-signal normalization tests for fallback precedence and bounded mappings.
  - API query route response test asserting serialized `source_session_quality_score`.
  - MCP compatibility parity test including `source_session_quality_score` in returned payloads.

### Phase 45 Progress Tracker

- [x] Added feedback-quality query filter contract:
  - `avg_relevance_feedback_min` on `models.EngramQueryRequest`.
- [x] Added REST validation parity:
  - `avg_relevance_feedback_min` now validates as bounded `0..1`.
- [x] Added repository predicate support:
  - `COALESCE(avg_relevance_feedback, 0.5) >= ...` filter path.
- [x] Added MCP parser/catalog parity:
  - `engram.query` now accepts and validates `avg_relevance_feedback_min`.
  - tool schema now documents `avg_relevance_feedback_min`.
- [x] Added regression coverage:
  - REST invalid-filter and parsed-request assertions for `avg_relevance_feedback_min`.
  - repository where-clause/params assertions for feedback-quality predicate.
  - MCP parity and validation tests for filter handling.

### Phase 46 Progress Tracker

- [x] Added contradiction-aware query filter contract:
  - `contradiction_count_max` on `models.EngramQueryRequest`.
- [x] Added REST validation parity:
  - `contradiction_count_max` now validates as non-negative.
- [x] Added repository predicate support:
  - `COALESCE(contradiction_count, 0) <= ...` filter path.
- [x] Added MCP parser/catalog parity:
  - `engram.query` now accepts and validates `contradiction_count_max`.
  - tool schema now documents `contradiction_count_max`.
- [x] Added regression coverage:
  - REST invalid-filter and parsed-request assertions for `contradiction_count_max`.
  - repository where-clause/params assertions for contradiction predicate.
  - MCP parity and validation tests for contradiction-count filter behavior.

### Phase 47 Progress Tracker

- [x] Added query-result diagnostics contract extension:
  - `feedback_count` on `models.EngramQueryResult`.
  - `contradiction_count` on `models.EngramQueryResult`.
- [x] Added repository query projection parity:
  - query SQL now projects `COALESCE(feedback_count, 0)`.
  - result mapping now forwards both feedback and contradiction counters.
- [x] Added REST/MCP parity coverage:
  - REST query route response test asserts `feedback_count` + `contradiction_count` serialization.
  - MCP compatibility query parity test includes quality counters in returned payloads.
- [x] Added repository regression coverage:
  - query fixture/mapping assertions now verify feedback and contradiction counters on query results.

### Phase 48 Progress Tracker

- [x] Added query-result engagement diagnostics contract extension:
  - `access_count` on `models.EngramQueryResult`.
  - `freshness_score` on `models.EngramQueryResult`.
- [x] Added repository projection/mapping parity:
  - query result mapping now forwards access and freshness values from candidate rows.
- [x] Added REST/MCP parity coverage:
  - REST query route response test asserts `access_count` + `freshness_score` serialization.
  - MCP compatibility query parity test includes engagement diagnostics fields.
- [x] Added repository regression coverage:
  - query fixture/mapping assertions now verify `access_count` and `freshness_score` on returned results.

### Phase 49 Progress Tracker

- [x] Added feedback-volume query filter contract:
  - `feedback_count_min` on `models.EngramQueryRequest`.
- [x] Added REST validation parity:
  - `feedback_count_min` now validates as non-negative.
- [x] Added repository predicate support:
  - `COALESCE(feedback_count, 0) >= ...` filter path.
- [x] Added MCP parser/catalog parity:
  - `engram.query` now accepts and validates `feedback_count_min`.
  - tool schema now documents `feedback_count_min`.
- [x] Added regression coverage:
  - REST invalid-filter and parsed-request assertions for `feedback_count_min`.
  - repository where-clause/params assertions for feedback-count predicate.
  - MCP parity and validation tests for feedback-volume filter behavior.

### Phase 50 Progress Tracker

- [x] Added useful-signal query filter contract:
  - `useful_count_min` on `models.EngramQueryRequest`.
- [x] Added REST validation parity:
  - `useful_count_min` now validates as non-negative.
- [x] Added repository predicate support:
  - `COALESCE(useful_count, 0) >= ...` filter path.
- [x] Added MCP parser/catalog parity:
  - `engram.query` now accepts and validates `useful_count_min`.
  - tool schema now documents `useful_count_min`.
- [x] Added regression coverage:
  - REST invalid-filter and parsed-request assertions for `useful_count_min`.
  - repository where-clause/params assertions for useful-count predicate.
  - MCP parity and validation tests for useful-signal filter behavior.

### Phase 51 Progress Tracker

- [x] Added useful-ratio query filter contract:
  - `useful_feedback_ratio_min` on `models.EngramQueryRequest`.
- [x] Added REST validation parity:
  - `useful_feedback_ratio_min` now validates as bounded `0..1`.
- [x] Added repository ratio-predicate support:
  - query filter now applies `useful_count / feedback_count` threshold when feedback exists.
  - deterministic fallback `0.5` is used when feedback_count is zero.
- [x] Added MCP parser/catalog parity:
  - `engram.query` now accepts and validates `useful_feedback_ratio_min`.
  - tool schema now documents `useful_feedback_ratio_min`.
- [x] Added regression coverage:
  - REST invalid-filter and parsed-request assertions for `useful_feedback_ratio_min`.
  - repository where-clause/params assertions for ratio predicate.
  - MCP parity and validation tests for useful-ratio filter behavior.

### Phase 52 Progress Tracker

- [x] Extended query-result diagnostics contract:
  - `useful_count`, `avg_relevance_feedback`, and `useful_feedback_ratio` on `models.EngramQueryResult`.
- [x] Added repository projection/mapping support:
  - query projection now selects `COALESCE(avg_relevance_feedback, 0.5)`.
  - result mapper now emits deterministic `useful_feedback_ratio` with neutral `0.5` fallback when feedback is absent.
- [x] Added REST/MCP parity coverage:
  - REST query response assertions now validate useful diagnostics fields.
  - MCP compatibility parity fixture now includes useful diagnostics in query results.
- [x] Added repository regression coverage:
  - query fixture assertions now verify useful diagnostics mapping and SQL projection clauses.
  - helper coverage ensures ratio fallback semantics when `feedback_count` is zero.

### Phase 53 Progress Tracker

- [x] Added contradiction-ratio query filter contract:
  - `contradiction_feedback_ratio_max` on `models.EngramQueryRequest`.
- [x] Added REST validation parity:
  - `contradiction_feedback_ratio_max` now validates as bounded `0..1`.
- [x] Added repository ratio-predicate support:
  - query filter now applies `contradiction_count / feedback_count` threshold when feedback exists.
  - deterministic fallback `0.0` is used when feedback_count is zero.
- [x] Added MCP parser/catalog parity:
  - `engram.query` now accepts and validates `contradiction_feedback_ratio_max`.
  - tool schema now documents `contradiction_feedback_ratio_max`.
- [x] Added regression coverage:
  - REST invalid-filter and parsed-request assertions for `contradiction_feedback_ratio_max`.
  - repository where-clause/params assertions for contradiction-ratio predicate.
  - MCP parity and validation tests for contradiction-ratio filter behavior.

### Phase 54 Progress Tracker

- [x] Extended query-result diagnostics contract:
  - `contradiction_feedback_ratio` on `models.EngramQueryResult`.
- [x] Added repository mapping support:
  - result mapper now emits deterministic contradiction ratio (`contradiction_count / feedback_count`).
  - fallback semantics return `0.0` when `feedback_count` is zero.
- [x] Added REST/MCP parity coverage:
  - REST query response assertions now validate `contradiction_feedback_ratio`.
  - MCP compatibility parity fixture now includes contradiction-ratio diagnostics.
- [x] Added repository regression coverage:
  - query fixture assertions now verify contradiction-ratio mapping.
  - helper coverage ensures zero-feedback fallback behavior remains deterministic.

### Phase 55 Progress Tracker

- [x] Added feedback-ceiling query filter contract:
  - `feedback_count_max` on `models.EngramQueryRequest`.
- [x] Added REST validation parity:
  - `feedback_count_max` now validates as non-negative.
- [x] Added repository predicate support:
  - query builder now supports `COALESCE(feedback_count, 0) <= ...`.
- [x] Added MCP parser/catalog parity:
  - `engram.query` now accepts and validates `feedback_count_max`.
  - tool schema now documents `feedback_count_max`.
- [x] Added regression coverage:
  - REST invalid-filter and parsed-request assertions for `feedback_count_max`.
  - repository where-clause/params assertions for feedback-ceiling predicate.
  - MCP parity and validation tests for feedback-ceiling filter behavior.

### Phase 56 Progress Tracker

- [x] Added useful-ceiling query filter contract:
  - `useful_count_max` on `models.EngramQueryRequest`.
- [x] Added REST validation parity:
  - `useful_count_max` now validates as non-negative.
- [x] Added repository predicate support:
  - query builder now supports `COALESCE(useful_count, 0) <= ...`.
- [x] Added MCP parser/catalog parity:
  - `engram.query` now accepts and validates `useful_count_max`.
  - tool schema now documents `useful_count_max`.
- [x] Added regression coverage:
  - REST invalid-filter and parsed-request assertions for `useful_count_max`.
  - repository where-clause/params assertions for useful-ceiling predicate.
  - MCP parity and validation tests for useful-ceiling filter behavior.

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
| 32 | ContinuWitty Query Protocol | Completed |
| 33 | Portable Export/Import Stash | Completed |
| 34 | Security Audit Remediation Program | Completed |
| 35 | Memory Engagement Tracking Baseline | Completed |
| 36 | Feedback Loop + Relevance Scoring | Completed |
| 37 | Time-Decay + Consolidation Suggestions | Completed |
| 38 | Contradiction Detection + Warning Flows | Completed |
| 39 | Temporal Query Extensions + Cost-Aware Context Assembly | Completed |
| 40 | Autonomous Memory Suggestions + Action Workflows | Completed |
| 41 | Feedback Signal Enrichment | Completed |
| 42 | Session Authority Scoring Baseline | Completed |
| 43 | Authority-Aware Query Filters | Completed |
| 44 | Authority Signal Transparency + Fallback Calibration | Completed |
| 45 | Feedback-Aware Query Filters | Completed |
| 46 | Contradiction-Aware Query Filters | Completed |
| 47 | Query Quality Diagnostics in Results | Completed |
| 48 | Query Engagement Diagnostics in Results | Completed |
| 49 | Feedback-Volume Query Filters | Completed |
| 50 | Useful-Signal Query Filters | Completed |
| 51 | Useful-Ratio Query Filters | Completed |
