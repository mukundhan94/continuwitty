# Engram Vault - Project Checkpoints

> Milestone tracking and phase progress.
> See [Plan.md](Plan.md) for full roadmap. See [todo.md](todo.md) for pending work. See [refactor.md](refactor.md) for refactoring checkpoints.

---

## Current State Summary

- **Phases 0-15 completed:** foundation, schema/storage, retrieval/rehydration, durability, chat continuity, providers, MCP, UI, acceptance, theme/UX hardening.
- **Phase 16 completed:** MCP developer tooling and typed clients (compatibility surface, typed clients, and CLI smoke utility complete).
- **Phase 17 completed:** document ingestion and RAG-ready retrieval, including session-level document pinning.
- **Phase 18 completed:** memory lifecycle policies (autosave/retention controls, timeline APIs/UI, and explicit consolidation merge/group semantics).
- **Phase 29 completed:** optional deterministic auto-metadata enrichment and MCP conversation-only persistence path.
- **Phase 30 completed:** MCP PAT lifecycle APIs/UI plus scoped bearer authorization for external agents.
- **Phase 31 completed:** project defaults + enterprise memory management + MCP organization (backend/API/MCP/web/tests/docs/skills complete with acceptance mock validation).
- **Phase 33 completed:** portable export/import stash workflow with REST + MCP + web transfer flows, owner/admin authorization, and source-fidelity round-trip coverage.
- **Phase 34 completed:** security audit remediation for production-safe config defaults, distributed login/MCP throttling, protected OAuth registration, and security regression coverage.
- **Go migration initiated (2026-02-22):** phased Python/FastAPI → Go migration started with dedicated progress tracker in `checkpoint-go-migration.md` (CP1 complete: module scaffold + config parity tests; CP2 complete: DB bootstrap/transaction parity tests; CP3 complete: API scaffold + health/version route parity tests; CP4 complete: auth hashing/CSRF parity tests; CP5 complete: user models/repository parity tests; CP6 complete: embeddings local/fallback parity tests; CP7 complete: engram repository helper/query parity tests; CP8 complete: DB read-path parity for `list_engrams`/`query_engrams`; CP9 complete: rehydration/source read-path parity; CP10 complete: engram write-path repository flows; CP11 complete: chat session repository baseline; CP12 complete: project repository baseline; CP13 complete: chat message/session pinning repository continuation; CP14 complete: document repository baseline; CP15 complete: MCP token repository baseline; CP16 complete: OAuth repository baseline; CP17 complete: collection repository baseline; CP18 complete: memory-admin session repository baseline; CP19 complete: memory-admin engram repository baseline; CP20 complete: memory-admin engram update/source-replacement repository parity; CP21 complete: memory-admin service baseline; CP22 complete: memory-admin API route baseline; CP23 complete: dependency-aware memory-admin API integration baseline; CP24 complete: runtime dependency wiring with migration-time actor/project resolver bridges; CP25 complete: context-first admin actor hardening baseline; CP26 complete: session-cookie actor middleware baseline with DB-backed canonicalization; CP27 complete: session login/logout/csrf route baseline with `/api/v1/me`; CP28 complete: session hardening baseline for TTL/issued-at validation and secure cookie attributes; CP29 complete: UI/login parity baseline (`/`, `/login`, `/logout`, `/ui`) on hardened sessions; CP30 complete: login guard + auth audit parity baseline in Go runtime/UI flow; CP31 complete: distributed limiter parity baseline with `rate_limit_state` store wiring and local fallback hardening; CP32 complete: UI/admin auth integration hardening with `/ui/admin` role-gating parity and auth/session code-health uplift; CP33 complete: role-aware `/api/v1/users` API parity + auth/session route health uplift; CP34 complete: session-auth engram route parity (`/api/v1/engrams*`) with strict >9.5 code-health gate; CP35 complete: non-checkpoint code-health uplift for `internal/auth/session.go` from 9.38 to 9.68; CP36 complete: non-checkpoint code-health uplift for `internal/repository/user.go` from 9.38 to 10.0; CP37 complete: non-checkpoint code-health uplift for `internal/embeddings/service.go` from 9.09 to 9.68; CP38 complete: non-checkpoint code-health uplift for `internal/repository/oauth.go` from 9.38 to 10.0; CP39 complete: non-checkpoint code-health uplift for `internal/repository/chat_test.go` from 9.09 to 10.0; CP40 complete: non-checkpoint code-health uplift for `internal/repository/chat.go` from 9.02 to 9.68; CP41 complete: non-checkpoint code-health uplift for `internal/repository/oauth_test.go` from 9.38 to 10.0; CP42 complete: non-checkpoint code-health uplift for `internal/repository/collection_test.go` from 9.09 to 10.0; CP43 complete: non-checkpoint code-health uplift for `internal/repository/document_test.go` from 8.72 to 9.68; CP44 complete: non-checkpoint code-health uplift for `internal/repository/document.go` from 8.81 to 9.68; CP45 complete: non-checkpoint code-health uplift for `internal/repository/chat_pinning.go` from 8.81 to 9.68; CP46 complete: non-checkpoint code-health uplift for `internal/repository/engram_write_test.go` from 9.26 to 10.0; CP47 complete: non-checkpoint code-health uplift for `internal/repository/admin_engram_update_test.go` from 9.25 to 10.0; CP48 complete: non-checkpoint code-health uplift for `internal/repository/admin_engram_test.go` from 9.38 to 10.0; CP49 complete: non-checkpoint code-health uplift for `internal/admin/service_test.go` from 9.38 to 10.0; CP50 complete: non-checkpoint code-health uplift for `internal/admin/service.go` from 8.54 to 9.68; CP51 complete: non-checkpoint code-health uplift for `internal/repository/chat_pinning_test.go` from 9.38 to 9.51).
- **Go migration current checkpoint (2026-02-26):** CP127 completed in `checkpoint-go-migration.md` with `chat.send_message` MCP dispatch in Go compatibility mode across direct and `tools/call` paths, runtime chat-service wiring, and structured chat/provider error mapping parity, full `go test ./...` pass, and CodeScene safeguard pass.

---

## Active Phase Progress

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

### Milestone 9 — Security Baseline (In Progress)
- Completed: local audit event logging, login rate limiting + lockout guard
- Remaining: OIDC, centralized audit pipeline, distributed auth rate limits

### Milestone 10 — Schema + Repository Layer
- Chat/session/pinning tables, engram ownership/visibility fields, visibility-aware repository filtering

### Milestone 11 — Provider Adapter Layer
- OpenAI, Anthropic, Bedrock adapters with normalized mapping
- Provider registry and configuration contract

### Milestone 12 — Chat API + Continuity
- Session/message endpoints, stream endpoint, save-as-engram, continue-session flows
- Context assembly with `used_engram_ids` and `source_references`

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

### Milestone 19 — Collaboration (Planned)
- Project membership model, scoped sharing/revocation, audit-visible events

### Milestone 20 — Production Security (Planned)
- OIDC integration, distributed auth rate limiting, centralized audit

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
| 19 | Collaboration + Sharing | Planned |
| 20 | Production Security | Planned |
| 21-23 | Observability, Release, EvalOps | Planned |
| 24-28 | Engram Link Graph | Planned |
| 32 | ContinuWitty Query Protocol | Planned |
| 33 | Portable Export/Import Stash | Completed |
| 34 | Security Audit Remediation Program | Completed |
