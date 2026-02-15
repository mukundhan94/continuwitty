# Plan Extension: Engram Link Graph and Traceable Memory Continuity (Phases 24–28)

## Summary
This plan extends the current roadmap **after Phase 23** and keeps **Phases 16–20 unchanged** (as requested).  
It adds a memory-linking graph so agents/users can trace where knowledge came from, traverse related engrams like associative memory, and avoid stuffing full history into prompt context.

The plan is built around your chosen defaults:
- phase ordering: **append after Phase 23**
- link creation mode: **hybrid** (manual + system suggestions with user confirmation)
- recall policy: **depth-1 default**, user-configurable higher depth
- link scope v1: **same project only**
- edge model: **directed + weighted + temporal**

---

## Phase Integrity (Unchanged Scope)
1. Keep existing phases exactly as-is:
   - Phase 16: MCP DX/tooling
   - Phase 17: RAG-ready ingestion
   - Phase 18: lifecycle policies
   - Phase 19: collaboration/sharing
   - Phase 20: production security hardening
2. Keep existing phases 21–23 intact.
3. Add new work as **Phases 24–28**.

---

## New Phases

## Phase 24 — Engram Graph Foundations (Schema + Repository)
### Goals
- Introduce first-class engram-to-engram links with traceability and temporal behavior.
- Preserve current APIs and behavior when no links exist.

### Deliverables
1. Add `engram_links` table (directed edges):
   - `link_id` UUID PK
   - `project_id`
   - `source_engram_id` FK -> `engrams`
   - `target_engram_id` FK -> `engrams`
   - `relation_type` (typed relation)
   - `weight` (0..1)
   - `temporal_weight` (computed/stored score for recency decay)
   - `confidence` (0..1)
   - `evidence_json` (citations/snippets/rationale)
   - `created_by_user_id`
   - `origin` (`manual`, `suggested`, `accepted_suggestion`, `system`)
   - `status` (`active`, `rejected`, `archived`)
   - `last_reinforced_at`, `created_at`, `updated_at`
2. Add optional `engram_link_events` audit/history table for lifecycle transitions.
3. Indexes:
   - `(project_id, source_engram_id)`, `(project_id, target_engram_id)`, `(status)`, `(relation_type)`, `(last_reinforced_at)`
4. Repository methods:
   - create/list/update/archive links
   - traverse neighbors by depth with visibility checks
   - score links by `weight * temporal_weight * confidence`
5. Migration strategy:
   - forward-only/idempotent
   - no breaking changes to existing engram/chat paths

### Exit Criteria
- Link graph persisted and queryable with same-project, role/visibility-safe enforcement.

---

## Phase 25 — Linking APIs + MCP Tools + Suggestion Pipeline
### Goals
- Expose graph operations via REST and MCP.
- Add suggestion workflow while keeping user confirmation in control.

### Deliverables
1. REST endpoints (new):
   - `POST /api/v1/engrams/{engram_id}/links`
   - `GET /api/v1/engrams/{engram_id}/links`
   - `PATCH /api/v1/engrams/links/{link_id}`
   - `DELETE /api/v1/engrams/links/{link_id}` (soft archive)
   - `POST /api/v1/engrams/{engram_id}/links/suggest`
   - `POST /api/v1/engrams/{engram_id}/trace`
2. MCP tools (new):
   - `engram.link_create`
   - `engram.link_list`
   - `engram.link_update`
   - `engram.link_archive`
   - `engram.link_suggest`
   - `engram.trace_path`
3. Suggestion service (hybrid mode):
   - candidate generation from semantic overlap + shared sources + session continuity hints
   - returns ranked suggestions requiring accept/reject action
4. Validation rules:
   - same-project links only (v1)
   - no self-loop unless relation type explicitly allows it
   - cycle-safe traversal guards and max-depth caps

### Exit Criteria
- Users and agents can create/inspect/curate links through REST and MCP with full auth/visibility parity.

---

## Phase 26 — Graph-Aware Context Assembly (Configurable Recall)
### Goals
- Make linked memory useful in chat without exploding context size.
- Keep depth-1 default and allow user-configurable expansion.

### Deliverables
1. Context assembler upgrade:
   - include direct linked neighbors (depth 1) by default
   - optional depth override (session/user setting) with hard max
   - rank/prune by relevance + link score + token budget
2. New session-level recall settings:
   - `link_recall_enabled` (default true)
   - `link_recall_depth` (default 1)
   - `link_recall_max_neighbors` (budget cap)
3. Config knobs:
   - `LINK_RECALL_DEFAULT_DEPTH=1`
   - `LINK_RECALL_MAX_DEPTH` (e.g., 3)
   - `LINK_RECALL_MAX_NEIGHBORS`
   - `LINK_TEMPORAL_DECAY_HALF_LIFE_DAYS`
4. Response metadata expansion:
   - `used_engram_link_ids`
   - `engram_trace_paths` (compact provenance chain summary)
5. Backward compatibility:
   - existing `used_engram_ids` remains unchanged
   - clients ignoring new fields continue to work

### Exit Criteria
- Chat responses can cite/trace linked memory paths while staying within token limits and visibility boundaries.

---

## Phase 27 — Memory Graph UX (“Where did this come from?”)
### Goals
- Provide user-facing graph navigation and provenance traceability.

### Deliverables
1. UI additions:
   - **Linked Engrams panel** (neighbors, relation type, weight, age)
   - **Trace view** for response provenance paths
   - **Accept/Reject suggestion** actions
   - **Depth control** (default 1, optional increase)
2. Session UX:
   - toggle graph recall on/off per session
   - quick “pin from linked neighbors”
3. Explainability UX:
   - “This answer used: Engram A -> Engram B -> Engram C” chain display
4. Accessibility + performance:
   - lazy-load graph nodes
   - responsive layout and keyboard navigability

### Exit Criteria
- Newcomer can answer “where did this answer come from?” directly from UI without API inspection.

---

## Phase 28 — Temporal Memory Dynamics + Graph Quality Controls
### Goals
- Mimic memory reinforcement/forgetting behavior with safe controls.
- Keep graph quality high over time.

### Deliverables
1. Temporal weighting engine:
   - decay stale links over time
   - reinforce links when reused successfully in sessions
2. Link hygiene jobs:
   - detect duplicates/conflicts
   - recommend consolidation/archival
3. Graph quality evals:
   - trace correctness
   - stale-link penalty behavior
   - hallucination reduction impact
4. Guardrails:
   - avoid over-traversal
   - relation-type conflict detection
   - noisy-link suppression thresholding

### Exit Criteria
- Graph remains interpretable and high-signal as data grows; eval gates prevent degradation.

---

## Important API / Interface / Type Additions

## REST Additions
1. `POST /api/v1/engrams/{engram_id}/links`
2. `GET /api/v1/engrams/{engram_id}/links`
3. `PATCH /api/v1/engrams/links/{link_id}`
4. `DELETE /api/v1/engrams/links/{link_id}`
5. `POST /api/v1/engrams/{engram_id}/links/suggest`
6. `POST /api/v1/engrams/{engram_id}/trace`

## MCP Additions
1. `engram.link_create`
2. `engram.link_list`
3. `engram.link_update`
4. `engram.link_archive`
5. `engram.link_suggest`
6. `engram.trace_path`

## Type Additions
1. `EngramLinkRecord`
2. `EngramLinkCreateRequest`
3. `EngramLinkUpdateRequest`
4. `EngramLinkSuggestion`
5. `EngramTraceRequest`
6. `EngramTraceResponse`
7. `ChatResponseMetadata` extension:
   - `used_engram_link_ids`
   - `engram_trace_paths`
8. Enums:
   - `EngramRelationType`
   - `EngramLinkStatus`
   - `EngramLinkOrigin`

---

## Data Flow (Graph-Aware Recall)
1. User/agent sends message.
2. System gathers pinned + retrieved engrams.
3. For each seed engram, traverse links (depth default=1, configurable).
4. Rank candidates by semantic relevance + link score + temporal weighting.
5. Prune by token budget and visibility.
6. Call provider with compact context pack.
7. Return response with `used_engram_ids` + link trace metadata.

---

## Test Cases and Scenarios

## Unit Tests
1. Link scoring (weight/confidence/temporal decay composition)
2. Traversal depth and cycle handling
3. Relation-type validation and self-loop constraints
4. Suggestion ranking determinism (with seeded fixtures)

## Integration Tests
1. CRUD for links under role/visibility constraints
2. Same-project enforcement for link creation
3. Chat response includes `used_engram_link_ids` when graph recall enabled
4. Depth setting changes retrieved neighbors as expected
5. Archive/reject behavior removes links from active traversal

## MCP Tests
1. Tool success/error envelopes for all `engram.link_*` tools
2. Authorization failures return structured RPC errors
3. Trace-path streaming metadata consistency

## UI Tests
1. Link suggestions accept/reject flow
2. Trace view rendering and path expansion
3. Session recall depth control behavior
4. “Where did this come from?” chain visibility for assistant responses

## E2E / Acceptance
1. Multi-session model switch:
   - Session A save -> Session B link/pin -> Session C synthesis with trace paths
2. Non-deterministic live-provider check:
   - verify structural trace metadata (not exact text)
3. Regression:
   - existing chat/pin/save/continue flows remain intact with link recall disabled/enabled

---

## Documentation and Skills Updates (per phase)
1. Update `README.md` progress, file map, and implementation log after each phase.
2. Update `AGENT.md` for graph conventions and traversal safety rules.
3. Add/update skills:
   - `skills/engram-graph-linking/SKILL.md`
   - `skills/graph-aware-context-assembly/SKILL.md`
   - `skills/traceability-and-provenance-ux/SKILL.md`

---

## Rollout Strategy
1. Phase 24 ships schema/repo behind feature flag.
2. Phase 25 enables APIs/MCP with suggestion mode disabled by default in production-like profiles.
3. Phase 26 enables depth-1 recall default after eval baseline passes.
4. Phase 27 exposes UI controls progressively.
5. Phase 28 enables temporal reinforcement/decay jobs with monitoring and rollback toggles.

---

## Explicit Assumptions and Defaults
1. Phases 16–20 remain unchanged.
2. Existing phases 21–23 remain unchanged; new work starts at Phase 24.
3. V1 link scope is same-project only.
4. Link creation is hybrid (manual + suggestions requiring confirmation).
5. Link model is directed, weighted, and temporal.
6. Default graph recall depth is 1; user can increase (bounded by max-depth config).
7. No breaking change to existing APIs; additive fields/endpoints only.
8. Visibility and authorization semantics remain consistent with current engram/session model.
