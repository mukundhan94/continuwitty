# Engram Vault Unified Roadmap

## Merge Note

This file is now the canonical roadmap for the project.

- Historical baseline phases from the original `Plan.md` are preserved.
- Chat/MCP/multi-provider roadmap phases are folded into the same timeline.
- Upcoming work is defined below as the next implementation phases.

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

- Selected FastAPI + Postgres/pgvector + LangGraph + React.
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

## Next Phases (Planned)

### Phase 16 - MCP Developer Experience and Tooling

### Goals

- Make MCP integration easy for external agents/clients.

### Deliverables

1. Typed MCP client helpers (Python + TypeScript) for JSON-RPC/SSE transport.
2. MCP smoke CLI (`engram-cli mcp-call`) for local tool debugging.
3. Contract tests for request/response/error envelopes.
4. README examples for each core tool group (`chat.*`, `engram.*`, `user.*`).

### Exit Criteria

- New users can run MCP tool calls end-to-end in under 5 minutes.

---

### Phase 17 - Engram Ingestion V2 (RAG-Ready Documents)

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

## Cross-Phase Working Rules

1. Keep local-first default behavior and deterministic fallback paths.
2. Update README in every completed phase (progress, file map, validation log).
3. Add/extend tests for every non-trivial behavior change.
4. Keep schema migrations forward-only and idempotent.
5. Preserve stable API contracts or version intentionally.
6. Keep `AGENT.md` + `skills/` aligned with major architecture changes.

---

## Near-Term Execution Order

1. Phase 16 (MCP developer tooling)
2. Phase 17 (RAG-ready ingestion)
3. Phase 18 (memory lifecycle policies)
4. Phase 20 (security hardening) in parallel design track with Phase 19
5. Phase 21 onward after security and data-sharing model stabilize
