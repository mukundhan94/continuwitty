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
   - [ ] enable access-aware federated linked recall across projects
