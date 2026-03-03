# Engram Vault Unified Roadmap

## Merge Note

This file is the canonical roadmap for the project.

- Historical baseline phases from the original `Plan.md` are preserved.
- Chat/MCP/multi-provider roadmap phases are folded into the same timeline.
- Upcoming work is defined below as the next implementation phases.

> See [migration/checkpoints/checkpoint.md](migration/checkpoints/checkpoint.md) for milestone tracking and [todo.md](todo.md) for pending work.

---

## Product Mission

Build a local-first memory system where agents and humans can:

1. Capture durable research/chat knowledge as engrams.
2. Reuse that memory across sessions and providers.
3. Operate through API, chat UI, CLI, and MCP tools.
4. Maintain traceable provenance, role-based visibility, and high-quality retrieval.

---

## Completed Phases (0-15)

### Phase 0 - Foundations and Success Criteria

- Scope, terminology, and success criteria established.
- Local-first architecture and threat model defined.

### Phase 1 - Framework and Storage Decisions

- Selected Go + Postgres/pgvector + LangGraph + React (initial FastAPI baseline was later migrated to Go).
- Chose metadata+vector retrieval as baseline.

### Phase 2 - Engram Schema and Contracts

- Versioned `MemoryEngram` structure implemented.
- Provenance-rich fields added (claims, sources, assumptions, decisions).

### Phase 3 - Storage Layer

- Core tables implemented (`engrams`, `sources`, artifacts + chat tables later).
- Hashing/indexing/versioning support added.

### Phase 4 - Ingestion Pipeline

- Engram creation flow from structured payload to persisted record shipped.

### Phase 5 - Retrieval and Rehydration

- Query + rehydration APIs shipped.
- Citation packing and rerank improvements implemented.

### Phase 6 - Agent Durability

- LangGraph checkpoint/resume workflow shipped.
- Auto-persist and periodic snapshot engram support added.

### Phase 7 - Local Auth + Test UI

- Login/session UX shipped.
- CSRF and route-protected UI flows implemented.

### Phase 8 - UI/CLI Expansion

- Source inspection and admin visibility delivered.
- CLI workflows (`upload/search/rehydrate/consolidate`) delivered.

### Phase 9 - Evaluation and Security Baseline

- Eval harness delivered (fact/cross/temporal/abstention).
- Login lockout/rate-limit/audit baseline delivered.

### Phase 10 - Chat Schema and Repository Layer

- Sessions/messages/pinned engrams persisted with visibility constraints.

### Phase 11 - Provider Adapter Layer

- OpenAI, Anthropic, Bedrock adapters with shared registry contract shipped.

### Phase 12 - Chat APIs and Continuity Flows

- Session/message CRUD + streaming.
- Save-as-engram, pin/unpin, continue-session workflows.

### Phase 13 - MCP Streaming Layer

- JSON-RPC over SSE endpoint and tool routing shipped.

### Phase 14 - React Chat Workbench

- Session/sidebar/chat/pinning/save/continue UX shipped.
- Streaming UX, retry flows, and tests added.

### Phase 15 - UX + Acceptance Hardening

- Playwright-BDD acceptance baseline and live Bedrock/triage tags added.
- Markdown rendering, stream parser fix, theme mode toggle, sticky sidebar refinements shipped.

---

## Next Phases (Active/Planned)

### Phase 16 - MCP Developer Experience and Tooling

### Status

- Completed: compatibility surface (`initialize`, `tools/list`, `tools/call`) and typed clients.
- Completed: CLI smoke command (`engram-cli mcp-call`) for local MCP debugging.

### Goals

- Make MCP integration easy for external agents/clients.

### Deliverables

1. Typed MCP client helpers (Python + TypeScript) for JSON-RPC/SSE transport.
2. MCP smoke CLI (`engram-cli mcp-call`) for local tool debugging.
3. Contract tests for request/response/error envelopes.
4. README examples for each core tool group (`chat.*`, `engram.*`, `user.*`).
5. Future-consideration note for optional `FastMCP` adapter pilot (non-breaking) after baseline DX is stable.

### Exit Criteria

- New users can run MCP tool calls end-to-end in under 5 minutes.

---

### Phase 17 - Engram Ingestion V2 (RAG-Ready Documents)

### Status

- Implemented and validated in current cycle.
- Delivered: ingestion routes, deterministic chunking, embedding abstraction/fallback, chat context blending, and UI upload workflow.

### Goals

- Support document uploads and high-signal retrieval artifacts.

### Deliverables

1. Ingestion endpoints for file/text submissions.
2. Chunking pipeline with deterministic chunk IDs and metadata.
3. Optional embedding provider abstraction for real embeddings (keep deterministic fallback).
4. UI upload surface with ingestion status and error visibility.
5. Query path capable of blending chat snapshots + document chunks.

### Exit Criteria

- Users can ingest docs and retrieve chunk-grounded evidence in chat and query APIs.

---

### Phase 18 - Memory Lifecycle Policies (Autosave, Retention, Consolidation)

### Status

- Completed: lifecycle policy schema/models/service/API/MCP/UI timeline controls are implemented and validated.
- Completed: explicit consolidation merge/grouping semantics in timeline presentation.

### Goals

- Control memory growth and quality over long-lived usage.

### Deliverables

1. Session-level autosave policy options (off/interval/event-based).
2. Snapshot retention and pruning controls.
3. Consolidation heuristics for reducing duplicate/low-value engrams.
4. UI timeline showing snapshots, merges, and consolidations.

### Exit Criteria

- Memory store growth is bounded with no critical context loss in eval scenarios.

---

### Phase 19 - Collaboration and Sharing Model

### Status

- Completed: project membership model (`owner`/`editor`/`viewer`) with canonical owner membership backfill.
- Completed: membership-gated visibility enforcement across engram/chat/document read paths.
- Completed: explicit engram share/unshare REST + MCP workflows with role-aware authorization.
- Completed: DB-backed project audit trail for membership changes, share/unshare, and chat pin/unpin events.
- Completed: admin memory UI panels for project member management and project audit timeline.

### Goals

- Safely share memory across users/projects.

### Deliverables

1. Project membership model (owner/editor/viewer).
2. Scoped engram sharing and revocation workflow.
3. Audit trail on share/unshare/pin actions.
4. UI affordances for shared-vs-private memory visibility.

### Exit Criteria

- Cross-user continuity works with explicit access boundaries and auditability.

---

### Phase 20 - Production Security Hardening

### Status

- Completed (2026-03-01).
- Completed deliverables:
  - OIDC login/session mapping baseline (`/login/oidc`, `/login/oidc/callback`) with ID-token verification and session-cookie integration.
  - OIDC config validation + redaction wiring (`OIDC_*` settings).
  - OIDC login route coverage in API/session UI tests, including negative abuse-path callback cases.
  - centralized audit sink rollout baseline with configurable sink URL/token/timeout/required mode and fail-open vs required delivery behavior.
  - callback hardening to consume pending OIDC state once callback validation succeeds.
  - OIDC failure audit coverage for provider verification failures and identity-to-user mapping failures.

### Goals

- Move from local auth baseline to production-grade identity and controls.

### Deliverables

1. OIDC integration (login/session mapping).
2. Secret-handling hardening and config validation.
3. Distributed rate-limiting strategy and lockout policy.
4. Security-focused regression and abuse-path tests.

### Exit Criteria

- Security review checklist passes for production deployment gate.

---

### Phase 21 - Observability and Reliability

### Status

- Completed (2026-03-01).
- Completed deliverables:
  - structured request telemetry middleware with domain/route/status/duration logging.
  - process-local request metrics aggregation with `/api/v1/metrics` operator endpoint.
  - stream-health and provider-failure category metrics in `/api/v1/metrics`.
  - lifecycle trace hooks for chat prepare/provider/persist/lifecycle stages.
  - provider fallback strategy with transient-error circuit-breaker protection.
  - regression coverage for fallback, circuit-open behavior, stream outcomes, and telemetry recording.

### Goals

- Ensure maintainable operations under load/provider failures.

### Deliverables

1. Structured logs for API/chat/MCP/provider paths.
2. Metrics for latency, stream health, provider failure categories.
3. Tracing hooks for session/message lifecycle.
4. Provider fallback/circuit-breaker strategy for transient failure classes.

### Exit Criteria

- Operators can diagnose session/provider/MCP failures using logs+metrics without code inspection.

---

### Phase 22 - Release Automation and Deployment Profiles

### Status

- Completed (2026-03-01).
- Completed deliverables:
  - CI split into explicit backend/web/deterministic-acceptance/release-smoke stages.
  - optional gated live-provider suite for release candidates (`workflow_dispatch` + `run_live_provider`).
  - compose profiles standardized for `dev`, `acceptance`, and `release-smoke`.
  - versioned release checklist + rollback runbook docs and Makefile release-gate targets.

### Goals

- Standardize repeatable release workflows.

### Deliverables

1. CI pipeline stages: backend checks, web checks, acceptance deterministic suite.
2. Optional gated live-provider suite for release candidates.
3. Docker Compose profiles for `dev`, `acceptance`, and `release-smoke`.
4. Versioned release checklist and rollback runbook.

### Exit Criteria

- A tagged release can be validated and rolled back with documented one-command workflows.

---

### Phase 23 - EvalOps and Prompt/Policy Governance

### Status

- Completed (2026-03-01).
- Completed deliverables:
  - versioned governance metadata for chat/MCP/eval surfaces (`chat_prompt_policy_version`, `mcp_tool_policy_version`, `eval_suite_version`).
  - deterministic EvalOps suite in Go for `continuity`, `citation_trust`, and `memory_drift` (extended in Phase 28 with `graph_trace` coverage).
  - release/CI regression delta gate with baseline + previous-run threshold enforcement.
  - historical trend artifacts (`latest.json`, `history.jsonl`, markdown trend report) and operational runbook.

### Goals

- Keep memory quality stable as features evolve.

### Deliverables

1. Expanded eval suite for continuity, citation trust, and memory drift.
2. Prompt/policy versioning for chat and MCP tool behavior.
3. Regression gates tied to eval deltas.
4. Dashboard/report for historical eval trends.

### Exit Criteria

- No release ships with significant eval regression in continuity/citation correctness.

---

### Phase 24 - Engram Graph Foundations (Link Schema + Repository)

### Status

- Completed (2026-03-01).
- Completed deliverables:
  - `engram_links` + `engram_link_events` schema foundation in `db/init/001_schema.sql`.
  - repository baseline for create/list/update/archive and depth-limited traversal with visibility-safe filtering.
  - typed link models (`relation_type`, `origin`, `status`) and unit tests.
  - runtime integration proof completed via Phase 25/26 REST + MCP + chat-context wiring.

### Goals

- Add first-class engram-to-engram links to support traceable, brain-like associative memory.

### Deliverables

1. Add `engram_links` table with directed edges and lifecycle metadata:
   - `source_engram_id`, `target_engram_id`, `relation_type`
   - `weight`, `temporal_weight`, `confidence`
   - `origin`, `status`, `evidence_json`
   - `created_by_user_id`, `last_reinforced_at`, timestamps
2. Add optional `engram_link_events` table for lifecycle/audit transitions.
3. Add indexes for project/source/target/status/relation/recency paths.
4. Add repository layer methods for:
   - create/list/update/archive links
   - depth-limited traversal
   - score-aware ordering with visibility checks
5. Keep migration style forward-only and idempotent.

### Exit Criteria

- Link graph records are persisted and queryable with same-project and visibility-safe enforcement.

---

### Phase 25 - Link APIs, MCP Tools, and Suggestion Pipeline

### Status

- Completed (2026-03-01).
- Completed deliverables:
  - REST link APIs (`create/list/update/archive/suggest/trace`) under `/api/v1/engrams/*`.
  - MCP link tools (`engram.link_*`, `engram.trace_path`) with scoped token policy integration.
  - hybrid suggestion service combining semantic overlap, source overlap, lexical continuity, and recency components.
  - same-project + visibility parity enforcement aligned with existing engram access model.

### Goals

- Expose link operations to UI/agents and add suggestion-driven linking with user confirmation.

### Deliverables

1. Add REST link APIs:
   - `POST /api/v1/engrams/{engram_id}/links`
   - `GET /api/v1/engrams/{engram_id}/links`
   - `PATCH /api/v1/engrams/links/{link_id}`
   - `DELETE /api/v1/engrams/links/{link_id}` (archive/soft delete)
   - `POST /api/v1/engrams/{engram_id}/links/suggest`
   - `POST /api/v1/engrams/{engram_id}/trace`
2. Add MCP link tools:
   - `engram.link_create`, `engram.link_list`, `engram.link_update`
   - `engram.link_archive`, `engram.link_suggest`, `engram.trace_path`
3. Add hybrid suggestion service:
   - candidate generation from semantic overlap + shared sources + continuity hints
   - accept/reject workflow for suggestions
4. Enforce v1 scope constraints:
   - same-project links only
   - role/visibility parity with existing engram access model

### Exit Criteria

- Users and agents can create/manage/trace links through REST and MCP with auth parity.

---

### Phase 26 - Graph-Aware Context Assembly (Configurable Recall Depth)

### Status

- Completed backend baseline (2026-03-01).
- Completed deliverables:
  - chat context assembler now includes linked-neighbor recall (default depth `1`) with bounded controls:
    - `link_recall_enabled`
    - `link_recall_depth`
    - `link_recall_max_neighbors`
  - pruning/ranking policy combines seed relevance, link quality (weight/confidence/temporal), recency, and depth-aware penalties.
  - chat response + stream metadata now include:
    - `used_engram_link_ids`
    - `engram_trace_paths`
  - REST + MCP `chat.send_message` paths accept optional per-message link recall overrides.

### Goals

- Use linked engrams in response context without overloading prompt budgets.

### Deliverables

1. Upgrade context assembler to include linked neighbors:
   - default traversal depth = 1
   - optional user/session override (bounded max depth)
2. Add configurable recall controls:
   - `link_recall_enabled`
   - `link_recall_depth` (default 1)
   - `link_recall_max_neighbors`
3. Add pruning/scoring policy:
   - relevance + link score + temporal score + token budget caps
4. Extend response metadata:
   - `used_engram_link_ids`
   - compact trace chain metadata (`engram_trace_paths`)
5. Keep existing `used_engram_ids` behavior fully backward compatible.

### Exit Criteria

- Linked memory is used by default (depth 1) and higher-depth traversal is optional and bounded.

---

### Phase 27 - Memory Graph UX and Traceability

### Status

- Completed (2026-03-01).
- Completed deliverables in this phase:
  - chat-composer recall controls for linked-memory context (`link recall`, `depth`, `max neighbors`).
  - transcript trace strip and compact provenance chain rendering from `used_engram_link_ids` and `engram_trace_paths`.
  - linked-memory panel showing relation type, weight/confidence, status/origin, and freshness/age cues.
  - suggestion queue UX with accept/reject actions backed by link create mutations (`active`/`rejected` status paths).
  - explainability summary cards unifying seed-engram count, trace-path count, linked-edge count, and citation count.

### Goals

- Make link graph navigation clear in UI so users can understand where responses come from.

### Deliverables

1. Add linked-memory UI surfaces:
   - linked engram panel (relation type, weight, age)
   - trace view for response provenance chains
2. Add suggestion workflow UX:
   - accept/reject link suggestions
3. Add session-level controls:
   - toggle graph recall on/off
   - configure traversal depth (within limits)
4. Add explainability affordances:
   - "this answer used" chain across engram links and citations

### Exit Criteria

- Newcomers can answer "where did this answer come from?" directly in UI without API inspection.

---

### Phase 28 - Temporal Dynamics and Graph Quality Controls

### Status

- Completed (2026-03-01).
- Completed in this checkpoint:
  - temporal decay now affects effective link quality scoring during graph-aware context assembly.
  - successful chat send/stream paths now reinforce used links (`last_reinforced_at` + boosted decayed `temporal_weight`).
  - suggested links that are reinforced through successful usage are promoted to `active`.
  - graph hygiene recommendation engine now detects duplicate-target links, relation conflicts, and stale low-value links.
  - authenticated hygiene API route is available at `POST /api/v1/engrams/{engram_id}/links/hygiene`.
  - graph-focused EvalOps coverage now includes `graph_trace` dimension checks (`used_engram_link_count`, `required_trace_targets`) with updated baseline fixtures (8 deterministic cases across 4 dimensions).
  - scheduled hygiene execution now runs on successful link reinforcement with per-source cadence, and stale/low-value recommendations are auto-archived from hygiene outputs.
  - configurable noisy-link suppression controls are available via policy settings and bounded request overrides (`link_noise_suppression_enabled`, `link_noise_score_threshold`).

### Goals

- Keep the memory graph high-signal over time via reinforcement/decay and quality maintenance.

### Deliverables

1. Add temporal weighting rules:
   - decay stale links over time
   - reinforce links reused in successful sessions
2. Add graph hygiene jobs:
   - duplicate/conflict detection
   - stale/low-value link archival recommendations
3. Add graph-focused eval coverage:
   - trace correctness
   - retrieval relevance impact
   - drift and stale-link tolerance
4. Add operational guardrails:
   - traversal caps
   - cycle-safe path expansion
   - noisy-link suppression thresholds

### Exit Criteria

- Graph quality stays stable at scale with measurable improvements in continuity and traceability.

---

### Phase 29 - Auto Metadata Enrichment for Conversation Persistence (MCP-First)

### Status

- Implemented in current cycle.
- Delivered across REST, chat save/autosave paths, MCP tools, and repository create centralization.

### Goals

- Allow agents to persist conversation memory without manually crafting tags and metadata.
- Keep metadata generation deterministic and local-first while preserving caller intent.

### Deliverables

1. Add fill-empty-only enrichment for engram metadata (`abstract`, `tags`, `keywords`):
   - only derive values when incoming fields are empty.
   - never overwrite non-empty caller-provided values.
2. Centralize enrichment in repository create flow so all create paths share behavior:
   - REST `/api/v1/engrams`
   - chat save/autosave flows
   - MCP `engram.create`
   - agent/consolidation/CLI creates
3. Add MCP tool:
   - `engram.create_from_conversation`
   - accepts conversation markdown and returns an enrichment report.
4. Persist enrichment trace in `engram_json.auto_metadata` for debugging and auditability.
5. Add deterministic/acceptance test coverage for enrichment + no-overwrite contract.

### Exit Criteria

- Empty metadata is auto-derived consistently across create paths.
- Explicit metadata remains untouched.
- MCP clients can persist conversation-only payloads and receive enrichment report details.

---

### Phase 30 - MCP Token Auth and Scoped Authorization

### Status

- Implemented in current cycle.
- Delivered: MCP personal access tokens, bearer-token auth for `/api/v1/mcp/stream`, scope/allowlist/project guards, admin token lifecycle UI (server + React), and automated coverage.
- Extended: OAuth compatibility layer for external MCP clients (authorization-server metadata, dynamic client registration, authorization-code PKCE exchange, and protected-resource metadata for auto-discovery).

### Goals

- Enable secure MCP access for external agents (for example LibreChat) without session-cookie coupling.
- Enforce least privilege using token scopes, optional tool allowlists, and optional project allowlists.

### Deliverables

1. MCP token lifecycle APIs:
   - `POST /api/v1/mcp/tokens`
   - `GET /api/v1/mcp/tokens`
   - `POST /api/v1/mcp/tokens/{token_id}/revoke`
2. Add bearer token auth path to `/api/v1/mcp/stream`:
   - parse/validate `Authorization: Bearer ...`
   - resolve actor from token owner
   - keep session-cookie fallback for backward compatibility
3. Add hybrid authorization policy:
   - base scope (`read` / `write`)
   - optional per-tool allowlist
   - optional project allowlist
4. Add admin token management UI:
   - create/list/revoke
   - one-time token plaintext reveal at creation
   - include both server-admin console and React admin workspace access
   - React admin workspace loads discoverable tool/project options and applies restrictions as removable chips
5. Add automated quality gates:
   - token service unit tests
   - token REST/API integration tests
   - MCP bearer scope/project/allowlist integration tests
   - acceptance mock scenarios for read-vs-write behavior and admin UI chip-flow token creation

### Exit Criteria

- Admins can issue/revoke multiple MCP tokens with distinct privilege policies.
- Read-scoped tokens cannot execute write tools.
- Project-scoped tokens cannot access out-of-scope projects.
- Revoked/expired tokens are denied authentication.

---

### Phase 31 - Project Defaults and Enterprise Memory Management

### Status

- Completed (2026-03-01).
- Completed subphases:
  - 31.0 roadmap scaffolding in `Plan.md` and `README.md`
  - 31.1 schema/repository foundation (`projects`, default-project persistence, soft-delete metadata, collections)
  - 31.2 project APIs + default-project fallback integration + `/api/v1/engrams*` auth hardening
  - 31.3 admin memory router/service for session/engram/collection lifecycle management
  - 31.4 MCP organization tools with scope/owner/admin/project-policy enforcement
  - 31.5 web routing + admin memory management page + workspace default-project control
  - 31.6 backend/web/acceptance test additions for phase behavior
  - 31.7 docs/skills closeout (`AGENT.md`, skills updates, final acceptance-test-mock rerun and log sync)

### Goals

- Add first-class project registry and per-user default project selection.
- Add enterprise-grade memory administration workflows for sessions and engrams.
- Expose project/memory organization operations through REST, MCP, and admin UI.

### Deliverables

1. Add `projects` registry and per-user default project persistence.
2. Add default-project fallback for engram create paths when `project_id` is omitted.
3. Add admin memory router for session/engram/collection lifecycle operations:
   - list/filter by project/session
   - soft delete/restore
   - move engrams across projects
   - edit engram metadata/body/sources
4. Add MCP organization tools:
   - project tools (`project.*`)
   - engram organization tools (`engram.list/get/update/move/delete/restore`)
   - collection tools (`engram.collection_*`)
   - session lifecycle tools (`chat.delete_session`, `chat.restore_session`)
5. Harden `/api/v1/engrams*` auth and visibility checks.
6. Add routing-based admin memory page in web UI.
7. Add backend/web/acceptance coverage and roadmap/docs updates.

### Deliverables Progress

- [x] Projects registry and per-user default project persistence are active in schema + services.
- [x] Default-project fallback is active for engram create paths in REST + MCP.
- [x] Admin memory APIs are available under `/api/v1/admin/memory`.
- [x] MCP organization toolset is implemented (`project_*`, `engram_*`, `chat_delete_session`, `chat_restore_session`).
- [x] `/api/v1/engrams*` endpoints are authenticated and actor-scoped.
- [x] Web admin memory route is implemented (`/admin/memory`) with management workflows.
- [x] Backend/web test coverage added for project defaults, memory admin APIs, schema backfill, MCP organization, and admin UI route behavior.
- [x] Phase closeout docs synced (`AGENT.md`/skills final pass + final acceptance mock evidence appended).

### Completed In Current Checkpoint

1. Schema and persistence:
   - `db/init/001_schema.sql` now includes first-class `projects`, user `default_project_id`, soft-delete metadata for sessions/engrams, and collection tables.
   - Added idempotent backfill logic so existing project IDs are promoted into `projects`.
2. Backend module layout for maintainability:
   - Added `internal/projects/` (service + model boundaries) and `internal/admin/` for memory-admin workflows.
   - Added `internal/api/admin_memory*.go` route modules plus runtime dependency wiring in `cmd/api/main.go`.
3. Behavior and security changes:
   - `/api/v1/engrams*` now requires authenticated actors.
   - Missing `project_id` on engram creation now resolves via caller default project with explicit response metadata.
   - Soft-deleted sessions/engrams are excluded from standard retrieval paths by default.
4. MCP organization layer:
   - Added project, engram, collection, and session-management tool handlers.
   - Added read/write/project-policy enforcement for the new toolset with owner/admin checks on mutating operations.
5. Web/admin UX:
   - Added app routing with dedicated admin memory page (`/admin/memory`).
   - Added project-default display/set flow in session sidebar.
   - Added admin workflows for listing/editing/moving/deleting/restoring sessions and engrams plus collection membership management.
6. Validation completed in this checkpoint:
   - `make -C /Users/mukundhan/Projects/engram check`
   - `make -C /Users/mukundhan/Projects/engram web-check`
   - `make -C /Users/mukundhan/Projects/engram acceptance-bddgen`
   - `make -C /Users/mukundhan/Projects/engram acceptance-typecheck`
7. Developer workflow UX:
   - Upgraded `Makefile` help output with colorized grouped commands, quick-start guidance, and a `print-config` diagnostics target.
8. MCP token panel reliability:
   - Updated frontend MCP client to accept both SSE and JSON-RPC fallback responses from `/api/v1/mcp/stream` so tool/project option loading remains stable.
   - Updated acceptance MCP step parsers to support both transports and removed race conditions in admin token option assertions.

### Exit Criteria

- Users can set a default project and engram create operations resolve it when `project_id` is missing.
- Admins can manage sessions and engrams from a dedicated admin page.
- MCP clients can organize memory with scope/ownership checks and project policy enforcement.
- Soft-delete defaults are safe, auditable, and restorable.

---

### Phase 32 - ContinuWitty Query Prefix (`cw>`) + Federated Linked Engram Recall

### Status

- Completed (2026-03-01).
- Completed in this phase:
  - Added `cw>` query protocol parsing and normalization in `internal/chat/query_protocol.go`.
  - Added runtime integration to strip directives from persisted/context query text while preserving a normalized `cw_plan_applied` plan in send/stream metadata.
  - Extended REST + MCP send-message payload surfaces with additive `cw_plan_applied` metadata in responses.
  - Added parser/runtime regression coverage for directive forms and payload metadata propagation.
  - Enabled federated cross-project link traversal/read SQL paths with per-node access filters (same-project restriction removed from link visibility/traversal queries).
  - Added fused semantic + trace ranking for federated engram candidates with resilient context packing backfill when higher-ranked candidates cannot be rehydrated.
  - Added retrieval audit telemetry signals (`retrieval_audit`) for blocked candidate filtering, suppressed/filtered/truncated trace paths, and cross-project path usage counts.
- This phase introduces a lightweight query protocol for users/agents (`cw>`) and extends linked-memory retrieval to cross-project associations with strict access-aware filtering.

### Why This Phase

- Users need a simple way to tell agents: "use ContinuWitty MCP memory workflows first".
- Linked memory should not stop at project boundaries when the actor legitimately has access to related engrams.
- Retrieval quality should improve by combining semantic search with graph/tree traversal and provenance-safe ranking.

### Goals

1. Add a stable `cw>` query-prefix protocol that nudges agents to use MCP + memory-aware workflows deterministically.
2. Support cross-project engram association and retrieval when authorization allows it.
3. Add access-aware tree/graph search in backend retrieval path for better context assembly.

### Locked Decisions

1. Prefix contract:
   - `cw>` at start of message enables ContinuWitty protocol mode.
   - If no directives are present, default mode is `auto` (retrieve-first, suggest-save).
2. Directive styles:
   - line-1 prefix (`cw>`)
   - optional intent form (`cw> retrieve`)
   - optional key-value form (`cw> mode=analyze project=engram-vault citations=required`).
3. Cross-project retrieval:
   - only if actor has visibility rights to each candidate engram.
   - never leak inaccessible engram existence in responses or metadata.
4. Search strategy:
   - hybrid semantic seed + bounded graph/tree expansion.
   - default depth bounded, cycle-safe traversal, token-budget aware packing.

### Deliverables

1. Query protocol parser and planner:
   - add parser module for `cw>` directives and normalized execution intents.
   - add execution-plan object used by chat + MCP entry paths.
2. Backend retrieval upgrades:
   - add hybrid retrieval mode:
     - semantic seed retrieval
     - access-aware graph expansion (BFS/beam bounded traversal)
     - relevance + link score + temporal decay ranking.
3. Cross-project link support:
   - allow link traversal across projects where actor has access.
   - maintain strict per-node visibility filtering before scoring and packing.
4. MCP and API integration:
   - ensure `chat.send_message` and MCP message paths accept protocol-derived retrieval hints.
   - return enriched provenance metadata:
     - `used_engram_ids`
     - `used_engram_link_ids`
     - `engram_trace_paths`
     - `retrieval_audit`
     - optional `cw_plan_applied` summary.
5. UI affordances:
   - add optional helper/hint near composer describing `cw>` usage patterns.
   - add response panel badge when `cw>` plan is active.
6. Documentation:
   - add `cw>` protocol reference with examples.
   - update architecture docs for hybrid semantic+graph recall flow.

### API / Interface Additions (Planned)

1. Chat message request extension (optional):
   - `query_protocol` or parsed `query_options` (server may auto-derive from message prefix).
2. Chat/MCP response extension (non-breaking additive fields):
   - `cw_plan_applied`
   - `used_engram_link_ids`
   - `engram_trace_paths`.
3. MCP tool additions or extensions:
   - extend existing query tools with graph-depth/retrieval hints and protocol metadata.
   - add optional explicit graph query tool if needed for admin/debug clients.

### Data and Policy Changes (Planned)

1. Link policy upgrade:
   - move from same-project-only v1 constraint to access-allowed federated traversal.
2. Optional indexing/backfill:
   - optimize cross-project traversal with additional source/target project indexes.
3. Audit:
   - add retrieval audit signals for cross-project path use and blocked-node filtering counts.

### Security and Guardrails

1. Access checks executed at every candidate node and edge expansion step.
2. Hidden-node suppression:
   - inaccessible nodes are dropped silently without existence leakage.
3. Traversal limits:
   - max depth, max visited nodes, max neighbors per node, max context tokens.
4. Safe defaults:
   - `cw>` without explicit directives cannot escalate permissions or override auth.

### Test Plan

1. Unit tests:
   - `cw>` parser normalization and directive precedence.
   - retrieval rank fusion and traversal bounds.
2. Integration tests:
   - cross-project graph recall with mixed-access fixtures.
   - verify inaccessible nodes never appear in outputs/metadata.
3. MCP tests:
   - `cw>` messages through MCP paths produce protocol metadata and valid retrieval behavior.
4. Acceptance tests:
   - end-to-end scenario:
     - save related engrams in different projects
     - grant actor access to subset
     - query with `cw>`
     - verify only authorized linked context is used.

### Exit Criteria

1. `cw>` messages consistently trigger memory-aware retrieval planning.
2. Cross-project linked recall works only for authorized memory.
3. Responses include clear provenance over semantic + linked context usage.
4. No auth leakage regressions in REST/MCP/UI flows.

---

### Phase 33 - Portable Memory Export/Import (Project + Collections + Engrams)

### Status

- Completed.
- Portable stash workflow is available across REST, MCP, and web transfer surfaces.

### Goals

1. Export project memory as a portable bundle including project metadata, collections, collection membership, engrams, and sources.
2. Support full-project export by default and selective collection export (Option C).
3. Import bundles with deterministic conflict handling and owner/admin-safe authorization.

### Locked Decisions

1. Scope: Option C
   - default: full project export
   - optional: export subset by `collection_ids[]`.
2. Embeddings:
   - exclude by default for portability and bundle size
   - optional include flag for same-instance fast migrations.
3. Authorization:
   - export allowed for project owner and admin
   - import allowed for project owner and admin on target project.

### Deliverables

1. Backend export/import domain module under `internal/export/` with typed bundle contracts and services.
2. REST endpoints for export and import bound to project context.
3. MCP tools for export/import with scope + ownership checks.
4. Web actions for project export/import with optional collection subset selection.
5. Unit, integration, and acceptance coverage for round-trip stash migration.

### Exit Criteria

1. A project owner can export a project and import it into another instance.
2. Collection-scoped export preserves item membership and source fidelity.
3. Import conflict policies are deterministic and test-covered.
4. Owner/admin authorization is enforced across REST, MCP, and web flows.

---

### Phase 34 - Security Audit Remediation Program

### Status

- Completed (2026-02-22).
- Remediation work shipped across config/session hardening, distributed throttling, OAuth protections, ingestion/logging controls, and regression coverage.

### Goals

1. Remove critical insecure defaults and deployment footguns.
2. Harden auth/session/OAuth/MCP paths for production operation.
3. Add regression coverage for identified abuse paths.

### Deliverables

1. Secret/config hardening with fail-fast production validation.
2. Distributed rate limiting for login and MCP transport.
3. OAuth hardening (PKCE S256-only and protected registration).
4. Ingestion and logging hardening from audit findings.
5. Security-focused regression and acceptance tests.

### Exit Criteria

1. Critical and high-priority findings are remediated with tests.
2. Production startup fails when unsafe defaults are configured.
3. Security regression suite guards remediated threat paths.

---

### Memory Improvements Alignment (docs/memory-improvements.md)

### Status

- Aligned on 2026-03-01 so execution tracking lives in one canonical roadmap.
- `docs/memory-improvements.md` remains the detailed blueprint.
- `Plan.md` is the implementation contract and phase gate source.

### Mapping

1. Blueprint Phase 1 (Foundation) maps to:
   - Phase 35: engagement/access tracking baseline.
   - Phase 36: feedback loop + relevance/freshness scoring.
   - Phase 37: time-decay and consolidation suggestions.
2. Blueprint Phase 2 (LLM safety/intelligence) maps to:
   - Phase 38: contradiction detection and warning flows.
   - Phase 39: temporal query extensions + cost-aware context assembly.
3. Blueprint Phase 3 (autonomous intelligence) maps to:
   - Phase 40: autonomous memory suggestions and action workflows.

---

### Phase 35 - Memory Engagement Tracking Baseline

### Status

- Completed (2026-03-02).
- Delivered scope: telemetry + counters integrated into existing chat send/stream success paths with non-blocking failure behavior.

### Goals

1. Track when engrams are used during successful chat responses.
2. Persist durable access events suitable for future scoring/feedback features.
3. Add forward-compatible schema fields required by the memory-improvements foundation.

### Deliverables

1. Schema:
   - add `engram` engagement counters (`access_count`, `last_accessed_at`) and forward-compatible aggregates (`useful_count`, `contradiction_count`).
   - add `engram_access_events` with retrieval-safe indexes.
2. Repository:
   - add write path to append access events and update engram aggregate counters atomically per event.
3. Chat runtime:
   - record used engram access on successful `chat.send_message` and successful stream completion.
   - preserve non-blocking behavior (access telemetry failures must not fail chat response generation).
4. Tests:
   - repository tests for event persistence + aggregate updates.
   - chat service tests for send/stream access recording and failure isolation.

### Exit Criteria

1. Every successful chat response with `used_engram_ids` records access events.
2. Access counters are monotonically updated and queryable for ranking inputs.
3. Regression tests cover send, stream, and recorder-failure cases.

---

### Phase 36 - Feedback Loop + Relevance Scoring

### Status

- Completed (2026-03-03).
- Delivered in this cycle:
  - explicit `engram_feedback` storage and aggregate updates (`useful_count`, `contradiction_count`).
  - REST endpoint `POST /api/v1/engrams/{engram_id}/feedback`.
  - MCP tool `engram.feedback` / `engram_feedback`.
  - feedback signal integration in retrieval reranking.
  - engagement/freshness weighting calibration in composite rerank scoring.
  - deterministic ranking tests for feedback + engagement + freshness signal effects.
  - latency benchmark notes captured in `docs/phase36-relevance-calibration.md`.

### Goals

1. Capture explicit memory usefulness/contradiction feedback.
2. Extend retrieval ranking beyond dense+lexical with engagement + freshness factors.

### Deliverables

1. `engram_feedback` storage and aggregation paths.
2. feedback MCP/API surfaces.
3. composite scoring rollout with deterministic weighting + tests.

### Exit Criteria

1. Feedback updates aggregate counters deterministically.
2. Relevance scoring remains stable and benchmarked under current latency targets.

---

### Phase 37 - Time-Decay + Consolidation Suggestions

### Status

- Completed (2026-03-03).
- Delivered in this checkpoint:
  - schema baseline for freshness scoring (`engrams.freshness_score`, `engrams.freshness_last_computed_at`).
  - repository freshness refresh routine with half-life decay and optional project scoping.
  - admin memory API route to trigger freshness refresh (`POST /api/v1/admin/memory/engrams/freshness/refresh`).
  - MCP maintenance tool to trigger freshness refresh (`engram.refresh_freshness` / `engram_refresh_freshness`, admin-only).
  - consolidation suggestion schema baseline (`engram_consolidation_suggestions` + indexes).
  - repository consolidation suggestion routines for deterministic exact-duplicate refresh/list operations.
  - admin service + REST routes for consolidation maintenance/listing:
    - `POST /api/v1/admin/memory/engrams/consolidation/refresh`
    - `GET /api/v1/admin/memory/engrams/consolidation/suggestions`
    - `POST /api/v1/admin/memory/engrams/consolidation/suggestions/{suggestion_id}/action`
  - MCP consolidation parity tooling:
    - `engram.refresh_consolidation` / `engram_refresh_consolidation` (admin-only)
    - `engram.consolidation_list` / `engram_consolidation_list` (admin-only)
    - `engram.consolidation_action` / `engram_consolidation_action` (admin-only)
  - repository/model tests for consolidation refresh/list defaulting, filtering, and status parsing.
  - deterministic acceptance scenario for consolidation refresh/list/action quality checks (`@phase37`, mock path).
  - benchmark coverage doc for grouping quality thresholds (`docs/phase37-consolidation-benchmark.md`).

### Goals

1. Introduce freshness decay and maintenance heuristics for stale/redundant memory.
2. Generate deterministic consolidation suggestions with operator-safe review workflows.

### Deliverables

1. freshness score maintenance job + repository updates.
2. consolidation suggestion schema/services + MCP/API tooling.
3. acceptance and precision/recall benchmark coverage for duplicate/theme grouping.

### Exit Criteria

1. Stale memory receives updated freshness scores over time.
2. Consolidation candidates are generated with test-covered deterministic criteria.

---

### Phase 38 - Contradiction Detection + Warning Flows

### Status

- Completed (2026-03-03).
- Delivered in this checkpoint:
  - contradiction-trace metadata (`has_contradiction`, `contradicting_link_ids`) propagated on recalled trace paths when relation type is `contradicts`.
  - chat context now emits `contradiction_warnings` for contradiction-bearing trace paths with severity guidance.
  - chat send responses and stream `meta`/`done` payloads now include `contradiction_warnings`.
  - MCP `chat.send_message` parity now forwards `contradiction_warnings`.
  - regression tests added for contradiction-warning generation and payload propagation.
  - contradiction alert persistence baseline added with deterministic schema + repository workflows:
    - `engram_contradiction_alerts` storage + indexes.
    - repository refresh/list/resolve primitives with project scoping and deterministic alert hashing.
    - repository/model unit tests for refresh/list/resolve behavior and status parsing.
  - contradiction alert review tooling shipped across admin REST + MCP:
    - admin REST endpoints for contradiction maintenance:
      - `POST /api/v1/admin/memory/engrams/contradictions/refresh`
      - `GET /api/v1/admin/memory/engrams/contradictions/alerts`
      - `POST /api/v1/admin/memory/engrams/contradictions/alerts/{alert_id}/resolve`
    - MCP tool parity for contradiction maintenance:
      - `engram.refresh_contradictions`
      - `engram.contradiction_list`
      - `engram.contradiction_resolve`
    - regression coverage for admin routes, service adapters, MCP parse/dispatch, and compatibility response parity.
  - contradiction warning synthesis benchmark baseline documented:
    - microbenchmarks for `buildContradictionWarnings` at 50/200 trace-path workloads.
    - benchmark artifact captured in `docs/phase38-contradiction-benchmark.md`.
  - contradiction warning quality benchmark coverage delivered:
    - deterministic acceptance precision/recall scenario:
      - `acceptance-tests/features/phase38-contradiction-mock.feature`
      - `acceptance-tests/src/steps/phase38-contradiction-mock.steps.ts`
    - scenario now included in default `@mock` suite after fixing contradiction-link create SQL CTE aliasing in `internal/repository/engram_links.go`.

### Goals

1. Detect contradictory memory traces before they are silently reused.
2. Surface contradiction risk consistently across chat and MCP response paths.
3. Establish a foundation for operator-assisted contradiction resolution.

### Deliverables

1. contradiction-warning metadata in chat context assembly.
2. send/stream/MCP payload parity for contradiction warnings.
3. contradiction alert persistence + resolution paths and benchmark notes.

### Exit Criteria

1. Contradiction risks are surfaced deterministically in chat and MCP outputs.
2. Contradiction warnings are test-covered and benchmarked for precision/recall.

---

### Phase 39 - Temporal Query Extensions + Cost-Aware Context Assembly

### Status

- Completed (2026-03-03).
- Delivered in this checkpoint:
  - trace-aware query constraints added for engram query paths:
    - `relation_type` and `trace_depth` filters in `models.EngramQueryRequest`.
    - repository trace filter support (`EXISTS` on active `engram_links`) with optional relation-type filtering.
    - REST decode validation/defaulting for relation + trace depth (`trace_depth` defaults to `1` when relation is set).
    - MCP parser/validation + catalog metadata parity for trace filters.
  - temporal recall-window extensions added for engram query paths:
    - `last_accessed_after` / `last_accessed_before`
    - `freshness_computed_after` / `freshness_computed_before`
    - repository query support using:
      - `COALESCE(last_accessed_at, created_at)` window predicates.
      - `COALESCE(freshness_last_computed_at, created_at)` window predicates.
    - REST decode validation for temporal windows + range ordering.
    - MCP parser/validation + catalog metadata parity for temporal windows.
  - temporal query extensions added for engram query paths:
    - `access_count_min` and `freshness_score_min` filters in `models.EngramQueryRequest`.
    - repository query builder support via `COALESCE(access_count, 0)` and `COALESCE(freshness_score, 1.0)` predicates.
    - MCP `engram.query` parser/validation + catalog metadata support for the new filters.
  - bounded context-budget controls added for chat send paths:
    - REST/MCP send-message payloads now accept `context_token_budget`.
    - context assembly now enforces bounded estimated-token budgets with deterministic section truncation.
    - retrieval audit now includes context-budget diagnostics:
      - `context_token_budget`
      - `context_token_estimate`
      - `context_token_truncated`
  - regression coverage added for:
    - context-budget normalization and truncation behavior.
    - context-budget audit metadata for empty/non-empty context assembly.
    - REST/MCP forwarding of `context_token_budget` overrides.
  - benchmark coverage expanded for complex temporal/engagement/trace query filters:
    - repository microbenchmarks:
      - `BenchmarkBuildEngramQueryWhereComplexTemporalEngagementTrace`
      - `BenchmarkBuildEngramQueryWhereComplexFilterMatrix`
    - benchmark artifact: `docs/phase39-query-benchmark.md`.

### Goals

1. Add explicit temporal controls so recall can prioritize the right time horizon.
2. Keep chat context assembly budget-aware to reduce token waste.
3. Preserve deterministic, traceable retrieval behavior across REST and MCP.

### Deliverables

1. Temporal query/filter extensions in engram query contracts.
2. Cost-aware context assembly controls with retrieval audit metadata.
3. Test and benchmark coverage for budget adherence and temporal filter correctness.

### Exit Criteria

1. Temporal filters can be applied consistently in REST and MCP query paths.
2. Context assembly honors bounded budgets with deterministic audit metadata.
3. Deterministic acceptance and unit coverage protects temporal + budget behavior.

---

### Phase 40 - Autonomous Memory Suggestions + Action Workflows

### Status

- Completed (2026-03-03).
- Delivered in this checkpoint:
  - schema baseline for `memory_curation_suggestions` in `db/init/001_schema.sql`.
  - model contracts in `internal/models/memory_curation_suggestion.go`:
    - suggestion types (`auto_save`, `consolidate`, `contradiction`, `link`)
    - suggestion statuses (`suggested`, `accepted`, `rejected`, `applied`)
  - repository baseline in `internal/repository/memory_curation_suggestions.go`:
    - `CreateMemoryCurationSuggestion`
    - `ListMemoryCurationSuggestions`
    - `ApplyMemoryCurationSuggestionAction`
  - repository regression coverage in `internal/repository/memory_curation_suggestions_test.go`.
  - API + MCP list/action parity for memory curation suggestions:
    - admin REST routes:
      - `POST /api/v1/admin/memory/engrams/{engram_id}/links/curation/refresh`
      - `GET /api/v1/admin/memory/engrams/curation/suggestions`
      - `POST /api/v1/admin/memory/engrams/curation/suggestions/{suggestion_id}/action`
    - MCP tools:
      - `engram.curation_refresh_links`
      - `engram.curation_list`
      - `engram.curation_action`
    - compatibility catalog + token project-policy support + route/dispatch regression tests.
  - scoped refresh hardening for project safety:
    - optional `project_id` is now accepted on REST + MCP link-curation refresh flows.
    - refresh rejects mismatched project scope with explicit bad-request semantics.
  - apply-action orchestration for curation suggestions:
    - `status=applied` now executes deterministic downstream actions before persisting curation status.
    - `consolidate` suggestions dispatch `merged` action on referenced consolidation suggestions.
    - `contradiction` suggestions dispatch `resolved` action on referenced contradiction alerts.
    - `link` suggestions dispatch link archival for archive-oriented hygiene actions and `review_relation_conflict`.
    - invalid/missing curation payload identifiers fail with explicit bad-request semantics.
  - suggestion generation workflow hooks:
    - consolidation refresh now regenerates type `consolidate` curation suggestions.
    - contradiction refresh now regenerates type `contradiction` curation suggestions.
    - admin link-curation refresh now persists deduped type `link` suggestions from on-demand hygiene recommendations.
    - scheduled link-hygiene execution now generates type `link` curation suggestions for non-auto-archived recommendations.
    - scheduled link-hygiene generation dedupes against existing pending link curation payload keys (`link_id`, `target_engram_id`, `suggested_action`).
    - generation pass resets stale `suggested` curation rows per type/project before rebuilding deterministic candidates.
  - acceptance coverage for curation generation/action quality:
    - deterministic `@phase40 @mock` acceptance scenarios validate:
      - generation + accepted action transitions.
      - `applied` action cascades to downstream consolidation (`merged`) and contradiction (`resolved`) workflows.
      - on-demand link-hygiene refresh generates actionable `link` curation suggestions and supports `status=applied`.
  - benchmark coverage baseline:
    - `internal/admin` benchmark suite for curation action latency, applied-side-effect orchestration, and sync-generation scaling.
    - benchmark artifact: `docs/phase40-curation-benchmark.md`.

### Goals

1. Provide a deterministic persistence layer for autonomous memory recommendations.
2. Enable safe action workflows with explicit accepted/rejected/applied states.
3. Prepare API/MCP and UI integration on top of stable repository contracts.

### Deliverables

1. Memory curation suggestion schema + model contracts.
2. Repository create/list/action workflows with validation.
3. API/MCP workflow parity and acceptance/benchmark coverage.

### Exit Criteria

1. Suggestions can be created, listed, and actioned consistently through REST + MCP.
2. Action transitions are validated and audit-friendly.
3. Deterministic test and benchmark coverage protects suggestion quality and latency.

---

### Phase 41 - Feedback Signal Enrichment

### Status

- Completed (2026-03-03).
- Delivered in this checkpoint:
  - explicit feedback supports optional `relevance_score` (`1-5`) across REST + MCP submit paths.
  - explicit feedback supports optional `session_id` attribution across REST + MCP submit paths.
  - explicit feedback supports optional `integration_depth` (`mentioned`/`elaborated`/`contradicted`/`ignored`) across REST + MCP submit paths.
  - feedback persistence now tracks aggregate counters on `engrams`:
    - `feedback_count`
    - `avg_relevance_feedback`
  - feedback records now persist optional per-event `relevance_score`, `session_id`, and `integration_depth`.
  - regression coverage expanded across repository/API/MCP for relevance-score, session-attribution, and integration-depth validation and forwarding.

### Goals

1. Capture richer explicit feedback quality signals without breaking existing feedback flows.
2. Improve downstream retrieval calibration inputs with durable aggregate relevance metrics.
3. Preserve deterministic behavior and validation across REST, MCP, and repository write paths.

### Deliverables

1. Schema/model extensions for relevance-score, integration-depth, and aggregate counters.
2. Repository feedback-write path updates for aggregate maintenance.
3. REST/MCP payload parity plus validation and regression coverage.

### Exit Criteria

1. Feedback submissions remain backward-compatible while accepting optional `relevance_score`, `session_id`, and `integration_depth`.
2. Aggregate counters are updated deterministically for each persisted feedback event.
3. REST/MCP docs and tests fully reflect feedback contract changes.

---

### Phase 42 - Session Authority Scoring Baseline

### Status

- Completed (2026-03-03).
- Delivered in this checkpoint:
  - schema baseline for session-authority quality signal:
    - `engrams.source_session_quality_score` (`0.0-1.0`, default `0.5`) with idempotent check constraint.
    - index added: `engrams_source_session_quality_idx`.
  - retrieval query pipeline now selects `source_session_quality_score` into rerank candidates.
  - composite rerank scoring now incorporates authority weighting via `source_session_quality_score`.
  - feedback aggregation now updates `source_session_quality_score` deterministically when `relevance_score` is provided.
  - regression coverage added for:
    - authority-aware rerank ordering behavior.
    - query-shape parity for authority column selection.

### Goals

1. Introduce an authority-quality signal tied to source-session trust.
2. Improve retrieval ranking quality with a bounded authority factor.
3. Keep scoring behavior deterministic, test-covered, and backward-compatible.

### Deliverables

1. Schema + indexing support for bounded `source_session_quality_score`.
2. Rerank integration of authority signal in repository query path.
3. Feedback-write calibration path for authority updates plus tests.

### Exit Criteria

1. Authority score exists in schema with deterministic defaults and guardrails.
2. Rerank favors higher-authority memories when competing signals are otherwise equal.
3. Feedback-driven updates and rerank behavior are protected by regression tests.

---

### Phase 43 - Authority-Aware Query Filters

### Status

- Completed (2026-03-03).
- Delivered in this checkpoint:
  - added `source_session_quality_min` filter to `models.EngramQueryRequest`.
  - REST query validation now enforces `source_session_quality_min` bounds (`0..1`).
  - repository query builder now supports `COALESCE(source_session_quality_score, 0.5) >= ...` predicate.
  - MCP `engram.query` parser/catalog now accept and validate `source_session_quality_min`.
  - regression coverage expanded across repository/API/MCP for filter parsing, validation, and query-shape assertions.

### Goals

1. Let operators and agents explicitly filter recall by authority quality.
2. Keep query behavior deterministic across REST and MCP.
3. Preserve backward compatibility for existing query clients.

### Deliverables

1. Contract extension for `source_session_quality_min`.
2. REST/MCP validation and schema metadata parity.
3. Repository query-predicate support with deterministic tests.

### Exit Criteria

1. Clients can request authority-threshold filtering in both REST and MCP query paths.
2. Invalid authority thresholds are rejected with explicit validation errors.
3. Query contract/docs/tests stay synchronized.

---

### Phase 44 - Authority Signal Transparency and Fallback Calibration

### Status

- Completed (2026-03-03).
- Delivered in this checkpoint:
  - feedback authority calibration no longer depends solely on `relevance_score`; fallback mappings now apply when score is omitted:
    - `integration_depth` fallback (`elaborated`/`mentioned`/`ignored`/`contradicted`) mapped to bounded authority signal.
    - `feedback_type` fallback (`useful`/`contradiction`) used when integration depth is absent.
  - feedback write path now updates `source_session_quality_score` deterministically from the resolved authority signal, while preserving existing average-relevance aggregation semantics.
  - engram query results now expose `source_session_quality_score` in returned rows for REST + MCP clients.
  - regression coverage expanded for authority fallback normalization and query-response authority-field parity.

### Goals

1. Keep authority scoring adaptive even when explicit relevance scores are missing.
2. Expose authority diagnostics directly to clients to improve retrieval transparency.
3. Preserve deterministic scoring behavior and backward-compatible query contracts.

### Deliverables

1. Fallback authority calibration logic in feedback repository write path.
2. `source_session_quality_score` response-field parity in engram query model/repository mapping.
3. REST/MCP/repository regression coverage plus docs synchronization.

### Exit Criteria

1. Feedback events without `relevance_score` still contribute bounded authority updates.
2. Query responses include `source_session_quality_score` consistently in REST and MCP tool paths.
3. Tests and docs protect the fallback-calibration and response-contract behavior.

---

### Phase 45 - Feedback-Aware Query Filters

### Status

- Completed (2026-03-03).
- Delivered in this checkpoint:
  - added `avg_relevance_feedback_min` to `models.EngramQueryRequest`.
  - REST query validation now enforces `avg_relevance_feedback_min` bounds (`0..1`).
  - repository query builder now supports `COALESCE(avg_relevance_feedback, 0.5) >= ...` predicate.
  - MCP `engram.query` parser/catalog now accept and validate `avg_relevance_feedback_min`.
  - regression coverage expanded across repository/API/MCP for parsing, validation, and query-shape parity.

### Goals

1. Let operators and agents filter recalled memory by explicit feedback quality.
2. Keep query semantics deterministic and aligned across REST and MCP surfaces.
3. Preserve backward compatibility for existing clients while expanding filter controls.

### Deliverables

1. Contract extension for `avg_relevance_feedback_min`.
2. REST/MCP validation + schema metadata parity.
3. Repository predicate support with regression coverage.

### Exit Criteria

1. Query clients can request minimum average relevance feedback thresholds across REST and MCP.
2. Out-of-range values are rejected with explicit validation errors.
3. Query contract, parser metadata, docs, and tests remain synchronized.

---

### Phase 46 - Contradiction-Aware Query Filters

### Status

- Completed (2026-03-03).
- Delivered in this checkpoint:
  - added `contradiction_count_max` to `models.EngramQueryRequest`.
  - REST query validation now enforces non-negative `contradiction_count_max`.
  - repository query builder now supports `COALESCE(contradiction_count, 0) <= ...` predicate.
  - MCP `engram.query` parser/catalog now accept and validate `contradiction_count_max`.
  - regression coverage expanded across repository/API/MCP for filter parsing, validation, and query-shape parity.

### Goals

1. Allow operators and agents to suppress high-conflict memories during recall.
2. Keep contradiction-aware filtering deterministic and aligned across REST and MCP.
3. Preserve backward-compatible query behavior for existing clients.

### Deliverables

1. Contract extension for `contradiction_count_max`.
2. REST/MCP validation and schema metadata parity.
3. Repository predicate support with regression tests.

### Exit Criteria

1. Query clients can apply contradiction-count ceilings across REST and MCP query paths.
2. Invalid contradiction ceilings are rejected with explicit validation details.
3. Query docs and tests stay synchronized with runtime parser and repository behavior.

---

### Phase 47 - Query Quality Diagnostics in Results

### Status

- Completed (2026-03-03).
- Delivered in this checkpoint:
  - `models.EngramQueryResult` now includes `feedback_count` and `contradiction_count`.
  - repository query SQL/mapping now forwards aggregate feedback diagnostics in query rows.
  - REST and MCP query paths now return quality counters alongside authority score.
  - regression coverage added for repository result mapping plus REST/MCP payload parity.

### Goals

1. Improve observability of recall quality signals in query responses.
2. Let clients reason about conflict density and feedback volume per candidate engram.
3. Preserve deterministic query payload shape across REST and MCP surfaces.

### Deliverables

1. Query-result contract extension for quality counters.
2. Repository query projection/mapping updates.
3. REST/MCP parity tests and docs synchronization.

### Exit Criteria

1. Query responses include `feedback_count` and `contradiction_count` consistently in REST and MCP.
2. Repository mapping behavior is protected by regression tests.
3. API/MCP docs reflect the new response diagnostics.

---

### Phase 48 - Query Engagement Diagnostics in Results

### Status

- Completed (2026-03-03).
- Delivered in this checkpoint:
  - `models.EngramQueryResult` now includes `access_count` and `freshness_score`.
  - repository query result mapping now forwards engagement diagnostics already used by rerank internals.
  - REST and MCP query paths now return engagement counters/scores alongside authority and feedback diagnostics.
  - regression coverage extended for repository mapping and REST/MCP payload parity.

### Goals

1. Expose engagement/recency signals in query output for downstream agent reasoning.
2. Improve transparency of rerank inputs returned to clients.
3. Keep query payload shape deterministic across REST and MCP paths.

### Deliverables

1. Query-result contract extension for `access_count` and `freshness_score`.
2. Repository mapping updates for engagement diagnostics.
3. REST/MCP parity tests and docs synchronization.

### Exit Criteria

1. Query responses include `access_count` and `freshness_score` in both REST and MCP.
2. Mapping behavior is regression-tested in repository/API/MCP suites.
3. API/MCP docs reflect engagement diagnostics in query result rows.

---

### Phase 49 - Feedback-Volume Query Filters

### Status

- Completed (2026-03-03).
- Delivered in this checkpoint:
  - added `feedback_count_min` to `models.EngramQueryRequest`.
  - REST query validation now enforces non-negative `feedback_count_min`.
  - repository query builder now supports `COALESCE(feedback_count, 0) >= ...` predicate.
  - MCP `engram.query` parser/catalog now accept and validate `feedback_count_min`.
  - regression coverage expanded across repository/API/MCP for filter parsing, validation, and query-shape assertions.

### Goals

1. Allow operators and agents to scope recall to memories with minimum explicit feedback volume.
2. Keep feedback-volume filtering deterministic and consistent across REST and MCP.
3. Preserve backward-compatible query behavior while extending quality controls.

### Deliverables

1. Contract extension for `feedback_count_min`.
2. REST/MCP validation + schema metadata parity.
3. Repository predicate support with regression tests.

### Exit Criteria

1. Query clients can apply minimum feedback-count thresholds across REST and MCP query paths.
2. Invalid threshold values are rejected with explicit validation details.
3. Docs and tests remain synchronized with parser and repository behavior.

---

### Phase 50 - Useful-Signal Query Filters

### Status

- Completed (2026-03-03).
- Delivered in this checkpoint:
  - added `useful_count_min` to `models.EngramQueryRequest`.
  - REST query validation now enforces non-negative `useful_count_min`.
  - repository query builder now supports `COALESCE(useful_count, 0) >= ...` predicate.
  - MCP `engram.query` parser/catalog now accept and validate `useful_count_min`.
  - regression coverage expanded across repository/API/MCP for parsing, validation, and query-shape parity.

### Goals

1. Allow operators and agents to favor memories with stronger explicit usefulness history.
2. Keep useful-signal filtering deterministic and aligned across REST and MCP.
3. Preserve backward compatibility while extending retrieval-quality controls.

### Deliverables

1. Contract extension for `useful_count_min`.
2. REST/MCP validation and schema metadata parity.
3. Repository predicate support with regression tests.

### Exit Criteria

1. Query clients can apply minimum useful-feedback thresholds across REST and MCP query paths.
2. Invalid values are rejected with explicit validation details.
3. Parser/repository/docs/tests remain synchronized for the new filter.

---

### Phase 51 - Useful-Ratio Query Filters

### Status

- Completed (2026-03-03).
- Delivered in this checkpoint:
  - added `useful_feedback_ratio_min` (`0..1`) to `models.EngramQueryRequest`.
  - REST query validation now enforces bounded `useful_feedback_ratio_min`.
  - repository query builder now supports ratio predicate:
    - `useful_count / feedback_count` when feedback exists.
    - deterministic neutral fallback (`0.5`) when feedback is absent.
  - MCP `engram.query` parser/catalog now accept and validate `useful_feedback_ratio_min`.
  - regression coverage expanded across repository/API/MCP for parsing, validation, and query-shape assertions.

### Goals

1. Let operators and agents filter recall by explicit useful-feedback ratio quality.
2. Keep ratio filtering deterministic and aligned across REST and MCP surfaces.
3. Preserve backward compatibility while extending retrieval-quality controls.

### Deliverables

1. Contract extension for `useful_feedback_ratio_min`.
2. REST/MCP validation and metadata parity.
3. Repository ratio predicate support with regression coverage.

### Exit Criteria

1. Query clients can apply minimum useful-feedback ratio thresholds in REST and MCP.
2. Out-of-range values are rejected with explicit validation details.
3. Parser/repository/docs/tests remain synchronized for ratio filtering behavior.

---

### Phase 52 - Useful Diagnostics in Query Results

### Status

- Completed (2026-03-03).
- Delivered in this checkpoint:
  - `models.EngramQueryResult` now includes `useful_count`, `avg_relevance_feedback`, and `useful_feedback_ratio`.
  - repository candidate projection now forwards `avg_relevance_feedback` with deterministic fallback (`0.5`).
  - repository result mapping now computes and returns `useful_feedback_ratio` with neutral fallback (`0.5`) when `feedback_count` is zero.
  - REST and MCP query flows now return useful-signal diagnostics alongside existing quality/engagement/authority signals.
  - regression coverage expanded for repository mapping/runtime query SQL and REST/MCP payload parity.

### Goals

1. Expose useful-signal diagnostics so query filters and ranking behavior are transparent to clients.
2. Keep feedback-quality diagnostics deterministic across REST and MCP query result payloads.
3. Preserve backward compatibility while extending query observability.

### Deliverables

1. Query-result contract extension for useful diagnostics.
2. Repository projection/mapping updates with deterministic ratio fallback behavior.
3. REST/MCP parity tests and docs synchronization.

### Exit Criteria

1. Query responses include `useful_count`, `avg_relevance_feedback`, and `useful_feedback_ratio` in REST and MCP.
2. Zero-feedback rows return deterministic neutral ratio fallback (`0.5`) and are regression-tested.
3. API/MCP docs and tests remain synchronized with repository output behavior.

---

### Phase 53 - Contradiction-Ratio Query Filters

### Status

- Completed (2026-03-03).
- Delivered in this checkpoint:
  - added `contradiction_feedback_ratio_max` (`0..1`) to `models.EngramQueryRequest`.
  - REST query validation now enforces bounded `contradiction_feedback_ratio_max`.
  - repository query builder now supports contradiction-ratio predicate:
    - `contradiction_count / feedback_count` when feedback exists.
    - deterministic low-risk fallback (`0.0`) when feedback is absent.
  - MCP `engram.query` parser/catalog now accept and validate `contradiction_feedback_ratio_max`.
  - regression coverage expanded across repository/API/MCP for parsing, validation, and query-shape assertions.

### Goals

1. Let operators and agents constrain recall by relative contradiction risk, not just raw contradiction counts.
2. Keep contradiction-ratio filtering deterministic and aligned across REST and MCP surfaces.
3. Preserve backward compatibility while extending retrieval-quality controls.

### Deliverables

1. Contract extension for `contradiction_feedback_ratio_max`.
2. REST/MCP validation and metadata parity.
3. Repository contradiction-ratio predicate support with regression coverage.

### Exit Criteria

1. Query clients can apply maximum contradiction-feedback ratio thresholds in REST and MCP.
2. Out-of-range values are rejected with explicit validation details.
3. Parser/repository/docs/tests remain synchronized for contradiction-ratio filtering behavior.

---

### Phase 54 - Contradiction-Ratio Diagnostics in Query Results

### Status

- Completed (2026-03-03).
- Delivered in this checkpoint:
  - `models.EngramQueryResult` now includes `contradiction_feedback_ratio`.
  - repository result mapping now computes and returns contradiction ratio using:
    - `contradiction_count / feedback_count` when feedback exists.
    - deterministic low-risk fallback (`0.0`) when feedback is absent.
  - REST and MCP query flows now return contradiction-ratio diagnostics alongside existing quality/engagement/usefulness signals.
  - regression coverage expanded for repository mapping helpers and REST/MCP payload parity.

### Goals

1. Expose contradiction-risk diagnostics so contradiction-aware filtering is transparent in retrieval output.
2. Keep contradiction-ratio diagnostics deterministic and aligned across REST and MCP.
3. Preserve backward compatibility while extending query observability.

### Deliverables

1. Query-result contract extension for `contradiction_feedback_ratio`.
2. Repository mapping updates with deterministic fallback behavior.
3. REST/MCP parity tests and docs synchronization.

### Exit Criteria

1. Query responses include `contradiction_feedback_ratio` in REST and MCP.
2. Zero-feedback rows return deterministic low-risk fallback (`0.0`) and are regression-tested.
3. API/MCP docs and tests remain synchronized with repository output behavior.

---

### Phase 55 - Feedback-Ceiling Query Filters

### Status

- Completed (2026-03-03).
- Delivered in this checkpoint:
  - added `feedback_count_max` to `models.EngramQueryRequest`.
  - REST query validation now enforces non-negative `feedback_count_max`.
  - repository query builder now supports `COALESCE(feedback_count, 0) <= ...` predicate.
  - MCP `engram.query` parser/catalog now accept and validate `feedback_count_max`.
  - regression coverage expanded across repository/API/MCP for parsing, validation, and query-shape assertions.

### Goals

1. Allow operators and agents to scope recall to low-feedback memories that need review or curation.
2. Keep feedback-ceiling filtering deterministic and aligned across REST and MCP.
3. Preserve backward compatibility while extending retrieval-quality controls.

### Deliverables

1. Contract extension for `feedback_count_max`.
2. REST/MCP validation and schema metadata parity.
3. Repository predicate support with regression tests.

### Exit Criteria

1. Query clients can apply maximum feedback-count thresholds across REST and MCP query paths.
2. Invalid threshold values are rejected with explicit validation details.
3. Parser/repository/docs/tests remain synchronized for feedback-ceiling filtering behavior.

---

### Phase 56 - Useful-Ceiling Query Filters

### Status

- Completed (2026-03-03).
- Delivered in this checkpoint:
  - added `useful_count_max` to `models.EngramQueryRequest`.
  - REST query validation now enforces non-negative `useful_count_max`.
  - repository query builder now supports `COALESCE(useful_count, 0) <= ...` predicate.
  - MCP `engram.query` parser/catalog now accept and validate `useful_count_max`.
  - regression coverage expanded across repository/API/MCP for parsing, validation, and query-shape assertions.

### Goals

1. Allow operators and agents to scope recall to lower-usefulness memories for review or curation.
2. Keep useful-ceiling filtering deterministic and aligned across REST and MCP.
3. Preserve backward compatibility while extending retrieval-quality controls.

### Deliverables

1. Contract extension for `useful_count_max`.
2. REST/MCP validation and schema metadata parity.
3. Repository predicate support with regression tests.

### Exit Criteria

1. Query clients can apply maximum useful-count thresholds across REST and MCP query paths.
2. Invalid threshold values are rejected with explicit validation details.
3. Parser/repository/docs/tests remain synchronized for useful-ceiling filtering behavior.

---

### Phase 57 - Access-Ceiling Query Filters

### Status

- Completed (2026-03-03).
- Delivered in this checkpoint:
  - added `access_count_max` to `models.EngramQueryRequest`.
  - REST query validation now enforces non-negative `access_count_max`.
  - repository query builder now supports `COALESCE(access_count, 0) <= ...` predicate.
  - MCP `engram.query` parser/catalog now accept and validate `access_count_max`.
  - regression coverage expanded across repository/API/MCP for parsing, validation, and query-shape assertions.

### Goals

1. Allow operators and agents to cap recall by engagement volume so low-touch memories can be reviewed.
2. Keep access-ceiling filtering deterministic and aligned across REST and MCP.
3. Preserve backward compatibility while extending retrieval-quality controls.

### Deliverables

1. Contract extension for `access_count_max`.
2. REST/MCP validation and schema metadata parity.
3. Repository predicate support with regression tests.

### Exit Criteria

1. Query clients can apply maximum access-count thresholds across REST and MCP query paths.
2. Invalid threshold values are rejected with explicit validation details.
3. Parser/repository/docs/tests remain synchronized for access-ceiling filtering behavior.

---

### Phase 58 - Freshness-Ceiling Query Filters

### Status

- Completed (2026-03-03).
- Delivered in this checkpoint:
  - added `freshness_score_max` to `models.EngramQueryRequest`.
  - REST query validation now enforces bounded `freshness_score_max` (`0..1`).
  - repository query builder now supports `COALESCE(freshness_score, 1.0) <= ...` predicate.
  - MCP `engram.query` parser/catalog now accept and validate `freshness_score_max`.
  - regression coverage expanded across repository/API/MCP for parsing, validation, and query-shape assertions.

### Goals

1. Allow operators and agents to constrain recall to memories below a freshness ceiling for decay-driven review workflows.
2. Keep freshness-ceiling filtering deterministic and aligned across REST and MCP.
3. Preserve backward compatibility while extending retrieval-quality controls.

### Deliverables

1. Contract extension for `freshness_score_max`.
2. REST/MCP validation and schema metadata parity.
3. Repository predicate support with regression tests.

### Exit Criteria

1. Query clients can apply maximum freshness-score thresholds across REST and MCP query paths.
2. Invalid threshold values are rejected with explicit validation details.
3. Parser/repository/docs/tests remain synchronized for freshness-ceiling filtering behavior.

---

### Phase 59 - Authority-Ceiling Query Filters

### Status

- Completed (2026-03-03).
- Delivered in this checkpoint:
  - added `source_session_quality_max` to `models.EngramQueryRequest`.
  - REST query validation now enforces bounded `source_session_quality_max` (`0..1`).
  - repository query builder now supports `COALESCE(source_session_quality_score, 0.5) <= ...` predicate.
  - MCP `engram.query` parser/catalog now accept and validate `source_session_quality_max`.
  - regression coverage expanded across repository/API/MCP for parsing, validation, and query-shape assertions.

### Goals

1. Allow operators and agents to target lower-authority memories for review and calibration workflows.
2. Keep authority-ceiling filtering deterministic and aligned across REST and MCP.
3. Preserve backward compatibility while extending retrieval-quality controls.

### Deliverables

1. Contract extension for `source_session_quality_max`.
2. REST/MCP validation and schema metadata parity.
3. Repository predicate support with regression tests.

### Exit Criteria

1. Query clients can apply maximum authority-score thresholds across REST and MCP query paths.
2. Invalid threshold values are rejected with explicit validation details.
3. Parser/repository/docs/tests remain synchronized for authority-ceiling filtering behavior.

---

### Phase 60 - Relevance-Ceiling Query Filters

### Status

- Completed (2026-03-03).
- Delivered in this checkpoint:
  - added `avg_relevance_feedback_max` to `models.EngramQueryRequest`.
  - REST query validation now enforces bounded `avg_relevance_feedback_max` (`0..1`).
  - repository query builder now supports `COALESCE(avg_relevance_feedback, 0.5) <= ...` predicate.
  - MCP `engram.query` parser/catalog now accept and validate `avg_relevance_feedback_max`.
  - regression coverage expanded across repository/API/MCP for parsing, validation, and query-shape assertions.

### Goals

1. Allow operators and agents to target lower-confidence memories by capping average relevance feedback.
2. Keep relevance-ceiling filtering deterministic and aligned across REST and MCP.
3. Preserve backward compatibility while extending retrieval-quality controls.

### Deliverables

1. Contract extension for `avg_relevance_feedback_max`.
2. REST/MCP validation and schema metadata parity.
3. Repository predicate support with regression tests.

### Exit Criteria

1. Query clients can apply maximum average relevance-feedback thresholds across REST and MCP query paths.
2. Invalid threshold values are rejected with explicit validation details.
3. Parser/repository/docs/tests remain synchronized for relevance-ceiling filtering behavior.

---

### Phase 61 - Useful-Ratio Ceiling Query Filters

### Status

- Completed (2026-03-03).
- Delivered in this checkpoint:
  - added `useful_feedback_ratio_max` to `models.EngramQueryRequest`.
  - REST query validation now enforces bounded `useful_feedback_ratio_max` (`0..1`).
  - repository query builder now supports useful-ratio ceiling predicate with deterministic fallback (`0.5` when feedback is absent).
  - MCP `engram.query` parser/catalog now accept and validate `useful_feedback_ratio_max`.
  - regression coverage expanded across repository/API/MCP for parsing, validation, and query-shape assertions.

### Goals

1. Allow operators and agents to cap retrieval by useful-feedback ratio to surface low-value memories for review.
2. Keep useful-ratio ceiling filtering deterministic and aligned across REST and MCP.
3. Preserve backward compatibility while extending retrieval-quality controls.

### Deliverables

1. Contract extension for `useful_feedback_ratio_max`.
2. REST/MCP validation and schema metadata parity.
3. Repository predicate support with regression tests.

### Exit Criteria

1. Query clients can apply maximum useful-feedback ratio thresholds across REST and MCP query paths.
2. Invalid threshold values are rejected with explicit validation details.
3. Parser/repository/docs/tests remain synchronized for useful-ratio ceiling filtering behavior.

---

### Phase 62 - Contradiction-Ratio Floor Query Filters

### Status

- Completed (2026-03-03).
- Delivered in this checkpoint:
  - added `contradiction_feedback_ratio_min` to `models.EngramQueryRequest`.
  - REST query validation now enforces bounded `contradiction_feedback_ratio_min` (`0..1`).
  - repository query builder now supports contradiction-ratio floor predicate with deterministic low-risk fallback (`0.0` when feedback is absent).
  - MCP `engram.query` parser/catalog now accept and validate `contradiction_feedback_ratio_min`.
  - regression coverage expanded across repository/API/MCP for parsing, validation, and query-shape assertions.

### Goals

1. Allow operators and agents to explicitly retrieve higher-contradiction memories for audit and remediation workflows.
2. Keep contradiction-ratio floor filtering deterministic and aligned across REST and MCP.
3. Preserve backward compatibility while extending retrieval-quality controls.

### Deliverables

1. Contract extension for `contradiction_feedback_ratio_min`.
2. REST/MCP validation and schema metadata parity.
3. Repository predicate support with regression tests.

### Exit Criteria

1. Query clients can apply minimum contradiction-feedback ratio thresholds across REST and MCP query paths.
2. Invalid threshold values are rejected with explicit validation details.
3. Parser/repository/docs/tests remain synchronized for contradiction-ratio floor filtering behavior.

---

### Phase 63 - Contradiction-Count Floor Query Filters

### Status

- Completed (2026-03-03).
- Delivered in this checkpoint:
  - added `contradiction_count_min` to `models.EngramQueryRequest`.
  - REST query validation now enforces non-negative `contradiction_count_min`.
  - repository query builder now supports `COALESCE(contradiction_count, 0) >= ...` predicate.
  - MCP `engram.query` parser/catalog now accept and validate `contradiction_count_min`.
  - regression coverage expanded across repository/API/MCP for parsing, validation, and query-shape assertions.

### Goals

1. Allow operators and agents to retrieve high-contradiction memories explicitly for triage workflows.
2. Keep contradiction-count floor filtering deterministic and aligned across REST and MCP.
3. Preserve backward compatibility while extending retrieval-quality controls.

### Deliverables

1. Contract extension for `contradiction_count_min`.
2. REST/MCP validation and schema metadata parity.
3. Repository predicate support with regression tests.

### Exit Criteria

1. Query clients can apply minimum contradiction-count thresholds across REST and MCP query paths.
2. Invalid threshold values are rejected with explicit validation details.
3. Parser/repository/docs/tests remain synchronized for contradiction-count floor filtering behavior.

---

### Phase 64 - Query Range Coherence Validation

### Status

- Completed (2026-03-03).
- Delivered in this checkpoint:
  - REST query validation now rejects inconsistent numeric ranges when both bounds are present (`*_min` must be `<= *_max`).
  - MCP `engram.query` parser now enforces the same range-coherence guardrails and returns `-32602` invalid-parameter errors for inverted ranges.
  - API/MCP docs now explicitly describe `min <= max` requirement for paired numeric filters.
  - regression coverage expanded for REST and MCP invalid-range scenarios.

### Goals

1. Prevent ambiguous or self-contradicting query payloads that can cause surprising retrieval behavior.
2. Keep numeric-range semantics deterministic and aligned across REST and MCP.
3. Preserve backward compatibility for all valid query payloads.

### Deliverables

1. REST range-coherence validation for numeric min/max filter pairs.
2. MCP parity validation for numeric min/max filter pairs.
3. Documentation and regression test updates.

### Exit Criteria

1. Inverted numeric ranges are rejected consistently across REST and MCP.
2. Validation error behavior is covered by regression tests.
3. Docs clearly communicate min/max ordering expectations.

---

### Phase 65 - Semantic Distance Ceiling Query Filter

### Status

- Completed (2026-03-03).
- Delivered in this checkpoint:
  - added `distance_max` to `models.EngramQueryRequest`.
  - REST query validation now enforces non-negative `distance_max`.
  - repository query builder now supports vector-distance ceiling predicate (`embed <=> $1::vector <= ...`).
  - MCP `engram.query` parser/catalog now accept and validate `distance_max`.
  - regression coverage expanded across repository/API/MCP for parsing, validation, and query-shape assertions.

### Goals

1. Allow operators and agents to bound semantic recall by absolute vector-distance ceiling.
2. Keep semantic-distance filtering deterministic and aligned across REST and MCP.
3. Preserve backward compatibility for query payloads that omit distance controls.

### Deliverables

1. Contract extension for `distance_max`.
2. REST/MCP validation and schema metadata parity.
3. Repository predicate support with regression tests.

### Exit Criteria

1. Query clients can apply maximum vector-distance thresholds across REST and MCP query paths.
2. Invalid threshold values are rejected with explicit validation details.
3. Parser/repository/docs/tests remain synchronized for distance-ceiling filtering behavior.

---

### Phase 66 - Semantic Distance Floor Query Filter

### Status

- Completed (2026-03-03).
- Delivered in this checkpoint:
  - added `distance_min` to `models.EngramQueryRequest`.
  - REST query validation now enforces non-negative `distance_min`.
  - repository query builder now supports vector-distance floor predicate (`embed <=> $1::vector >= ...`).
  - MCP `engram.query` parser/catalog now accept and validate `distance_min`.
  - range-coherence guardrails now enforce `distance_min <= distance_max` when both are provided.
  - regression coverage expanded across repository/API/MCP for parsing, validation, and query-shape assertions.

### Goals

1. Allow operators and agents to define semantic distance bands with both minimum and maximum bounds.
2. Keep semantic-distance band filtering deterministic and aligned across REST and MCP.
3. Preserve backward compatibility for payloads that omit distance-floor controls.

### Deliverables

1. Contract extension for `distance_min`.
2. REST/MCP validation and schema metadata parity.
3. Repository predicate support with regression tests.

### Exit Criteria

1. Query clients can apply minimum vector-distance thresholds across REST and MCP query paths.
2. Invalid threshold values and inverted distance windows are rejected with explicit validation details.
3. Parser/repository/docs/tests remain synchronized for distance-floor and distance-window behavior.

---

### Phase 67 - Query Rerank Diagnostics Transparency

### Status

- Completed (2026-03-03).
- Delivered in this checkpoint:
  - extended `models.EngramQueryResult` with rerank explainability diagnostics:
    - `composite_rank_score`
    - `dense_score`
    - `lexical_overlap_score`
    - `feedback_signal_score`
    - `engagement_signal_score`
    - `freshness_signal_score`
    - `authority_signal_score`
  - repository rerank pipeline now records component signals and final composite rank score per candidate row.
  - query-result mapping now forwards rerank diagnostics with deterministic fallback computation when score fields are absent.
  - REST and MCP query parity tests now assert rerank diagnostics are preserved in response payloads.
  - API and MCP docs now describe rerank explainability fields returned in query rows.

### Goals

1. Expose ranking internals so operators and agents can explain retrieval ordering without reverse-engineering repository logic.
2. Keep rerank diagnostics deterministic across repository, REST, and MCP surfaces.
3. Preserve backward compatibility while extending query-result diagnostics.

### Deliverables

1. Query-result contract extension for rerank diagnostics.
2. Repository rerank instrumentation and result mapping parity.
3. REST/MCP test and documentation updates.

### Exit Criteria

1. Query responses include composite and component rerank scores for every row.
2. Diagnostics remain deterministic and stable in regression tests.
3. API/MCP docs reflect the expanded query-result diagnostics contract.

---

### Phase 68 - Composite-Rank Score Query Band Filters

### Status

- Completed (2026-03-03).
- Delivered in this checkpoint:
  - added `composite_rank_score_min` and `composite_rank_score_max` to `models.EngramQueryRequest`.
  - REST query validation now enforces bounded composite-rank filters (`0..1`) and rejects inverted composite-rank windows.
  - repository query pipeline now supports post-rerank composite-score filtering before final top-k trimming.
  - MCP `engram.query` parser/catalog now accept and validate composite-rank filters with min/max window parity.
  - regression coverage expanded across repository/API/MCP for parsing, validation, filtering, and catalog metadata assertions.

### Goals

1. Allow operators and agents to constrain query results to a desired rerank-confidence band.
2. Keep composite-rank filtering deterministic and aligned across REST and MCP.
3. Preserve backward compatibility for clients that omit composite-rank filters.

### Deliverables

1. Contract extension for `composite_rank_score_min` and `composite_rank_score_max`.
2. REST/MCP validation and schema metadata parity.
3. Repository post-rerank filter support with regression tests.

### Exit Criteria

1. Query clients can apply composite-rank score floors/ceilings across REST and MCP query paths.
2. Invalid threshold values and inverted composite-rank windows are rejected consistently.
3. Parser/repository/docs/tests remain synchronized for composite-rank band filtering behavior.

---

## Cross-Phase Working Rules

1. Keep local-first default behavior and deterministic fallback paths.
2. Update `docs/implementation-log.md` and `migration/checkpoints/checkpoint.md` in every completed phase.
3. Add/extend tests for every non-trivial behavior change.
4. Keep schema migrations forward-only and idempotent.
5. Preserve stable API contracts or version intentionally.
6. Keep `AGENT.md` + `skills/` aligned with major architecture changes.

---

## Near-Term Execution Order

1. Phase 20 (production security hardening) completed on 2026-03-01.
2. Phase 23 (EvalOps and prompt/policy governance) completed on 2026-03-01.
3. Execute link-graph roadmap in order:
   - Phase 24 (graph foundations) completed on 2026-03-01.
   - Phase 25 (link APIs/MCP + suggestions) completed on 2026-03-01.
   - Phase 26 (graph-aware recall) backend baseline completed on 2026-03-01.
   - Phase 27 (traceability UX) completed on 2026-03-01.
   - Phase 28 (temporal dynamics + graph quality) completed on 2026-03-01.
4. Execute Phase 32 after Phase 24-28 baselines are in place:
   - [x] add `cw>` query protocol baseline (parser + runtime metadata)
   - [x] enable access-aware federated linked recall across projects
5. Execute memory-intelligence foundation in order:
   - [x] Phase 35: memory engagement tracking baseline.
   - [x] Phase 36: feedback loop + relevance/freshness scoring.
   - [x] Phase 37: time-decay + consolidation suggestions.
   - [x] Phase 38: contradiction detection + warning flows.
   - [x] Phase 39: temporal query extensions + cost-aware context assembly.
   - [x] Phase 40: autonomous memory suggestions and action workflows.
6. Execute feedback-signal enrichment increment:
   - [x] Phase 41: richer explicit feedback payloads + aggregate relevance counters.
7. Execute authority-scoring baseline increment:
   - [x] Phase 42: source-session authority scoring baseline in schema/retrieval/feedback loops.
8. Execute authority-aware query contract increment:
   - [x] Phase 43: authority-threshold query filter parity across REST/MCP/repository.
9. Execute authority signal transparency increment:
   - [x] Phase 44: fallback authority calibration + query-response authority signal exposure.
10. Execute feedback-quality query filter increment:
   - [x] Phase 45: `avg_relevance_feedback_min` parity across REST/MCP/repository.
11. Execute contradiction-aware query filter increment:
   - [x] Phase 46: `contradiction_count_max` parity across REST/MCP/repository.
12. Execute query diagnostics increment:
   - [x] Phase 47: expose `feedback_count` + `contradiction_count` in query results.
13. Execute engagement diagnostics increment:
   - [x] Phase 48: expose `access_count` + `freshness_score` in query results.
14. Execute feedback-volume query filter increment:
   - [x] Phase 49: `feedback_count_min` parity across REST/MCP/repository.
15. Execute useful-signal query filter increment:
   - [x] Phase 50: `useful_count_min` parity across REST/MCP/repository.
16. Execute useful-ratio query filter increment:
   - [x] Phase 51: `useful_feedback_ratio_min` parity across REST/MCP/repository.
17. Execute useful-diagnostics query result increment:
   - [x] Phase 52: expose `useful_count` + `avg_relevance_feedback` + `useful_feedback_ratio` in query results.
18. Execute contradiction-ratio query filter increment:
   - [x] Phase 53: `contradiction_feedback_ratio_max` parity across REST/MCP/repository.
19. Execute contradiction-ratio diagnostics increment:
   - [x] Phase 54: expose `contradiction_feedback_ratio` in query results.
20. Execute feedback-ceiling query filter increment:
   - [x] Phase 55: `feedback_count_max` parity across REST/MCP/repository.
21. Execute useful-ceiling query filter increment:
   - [x] Phase 56: `useful_count_max` parity across REST/MCP/repository.
22. Execute access-ceiling query filter increment:
   - [x] Phase 57: `access_count_max` parity across REST/MCP/repository.
23. Execute freshness-ceiling query filter increment:
   - [x] Phase 58: `freshness_score_max` parity across REST/MCP/repository.
24. Execute authority-ceiling query filter increment:
   - [x] Phase 59: `source_session_quality_max` parity across REST/MCP/repository.
25. Execute relevance-ceiling query filter increment:
   - [x] Phase 60: `avg_relevance_feedback_max` parity across REST/MCP/repository.
26. Execute useful-ratio ceiling query filter increment:
   - [x] Phase 61: `useful_feedback_ratio_max` parity across REST/MCP/repository.
27. Execute contradiction-ratio floor query filter increment:
   - [x] Phase 62: `contradiction_feedback_ratio_min` parity across REST/MCP/repository.
28. Execute contradiction-count floor query filter increment:
   - [x] Phase 63: `contradiction_count_min` parity across REST/MCP/repository.
29. Execute query range coherence validation increment:
   - [x] Phase 64: reject numeric `*_min > *_max` payloads across REST/MCP.
30. Execute semantic distance ceiling query filter increment:
   - [x] Phase 65: `distance_max` parity across REST/MCP/repository.
31. Execute semantic distance floor query filter increment:
   - [x] Phase 66: `distance_min` parity across REST/MCP/repository.
32. Execute rerank diagnostics transparency increment:
   - [x] Phase 67: expose composite/component rerank scores in query result payloads across repository/REST/MCP.
33. Execute composite-rank score filter increment:
   - [x] Phase 68: `composite_rank_score_min` + `composite_rank_score_max` parity across REST/MCP/repository.
