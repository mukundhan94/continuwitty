# ContinuWitty Use Cases (Engram Vault)

## What This Document Is

This guide explains who ContinuWitty is for, what pain points it solves, and how to use it in real workflows.

It is written for:
- Enterprise teams
- Product and engineering leaders
- AI platform teams
- Individual developers building agents

It covers:
- What is available now in this repository
- How MCP integration makes agents "not forget"
- Where upcoming memory-intelligence phases expand value further

---

## ContinuWitty in One Sentence

ContinuWitty is a durable memory operating layer for humans and AI agents: it captures important context as reusable engrams, retrieves the right context later with provenance, and keeps usage secure and auditable.

---

## The Core Problem It Solves

Without durable memory, teams and agents repeatedly lose context.

Common failures:
- Same decisions are re-discussed every sprint.
- Incident learnings are buried in old chats and docs.
- Agents answer with partial context or stale assumptions.
- Teams cannot explain "why this answer" during reviews.
- Cross-project work causes data leakage risk if access is not scoped.

ContinuWitty addresses this by combining:
- Memory capture (`engram.create`, `chat.save_as_engram`)
- Retrieval (`engram.query`, `engram.rehydrate`)
- Continuity workflows (`chat.continue_session`, pinned engrams/documents)
- Collaboration controls (projects, memberships, sharing, audit events)
- Agent integration over MCP (`/api/v1/mcp/stream`)

---

## Pain Point to Solution Map

| Pain Point | Why It Hurts | ContinuWitty Workflow | Outcome |
|---|---|---|---|
| Context disappears after a session ends | Teams and agents restart from zero | Save session to engram + continue session (`chat.save_as_engram`, `chat.continue_session`) | Reduced rework and faster restart |
| Valuable knowledge is unsearchable in long chat history | Time spent re-reading old logs | Query + rehydrate (`engram.query`, `engram.rehydrate`) | Faster retrieval of prior reasoning |
| Answers cannot be trusted in review | Missing provenance and evidence chain | Source inspection + retrieval metadata (`/sources`, `used_engram_ids`, `source_references`) | Better auditability and trust |
| Shared memory is unsafe across teams | Overexposure and policy violations | Project scoping + membership + MCP token policy | Safe reuse with least privilege |
| Agents forget decisions over long-running tasks | Stateless loops and no durable store | MCP tool loop with save/query/continue pattern | More consistent multi-step agent behavior |
| Duplicate or stale memory grows over time | Retrieval quality degrades | Link graph + hygiene recommendations (`engram.link_suggest`, `links/hygiene`) | Cleaner memory graph and better recall |

---

## Customer Segments and Use Cases

## 1) Enterprise Engineering Teams

### Use Case: Architecture Decision Continuity

**Pain:** Architecture choices are debated repeatedly because rationale is scattered.

**Workflow:**
1. Team discusses options in chat session.
2. Decision is persisted as engram with sources.
3. Next sprint, team queries engrams before new proposal.
4. Relevant engrams are pinned into the active session.

**ContinuWitty Features Used:**
- Chat sessions and message flows
- `chat.save_as_engram`
- `engram.query` and `engram.rehydrate`
- Pinned engrams

**Business Value:**
- Fewer repeated debates
- Better onboarding for new engineers
- Faster design reviews with evidence

---

### Use Case: Incident Response Memory

**Pain:** Postmortem lessons are not reused when similar incidents happen again.

**Workflow:**
1. During incident, ingest runbooks and logs as documents.
2. Pin relevant docs/engrams in response session.
3. After resolution, save incident summary as engram with provenance.
4. Future incidents query similar engrams and trace linked mitigations.

**ContinuWitty Features Used:**
- Document ingestion and pinning
- Chat send/stream with retrieval metadata
- `engram.link_create`, `engram.trace_path`

**Business Value:**
- Faster mean-time-to-diagnosis
- Better consistency in remediation steps

---

## 2) Enterprise Product and Program Teams

### Use Case: Requirement and Decision Traceability

**Pain:** Product intent is lost across meetings, tickets, and changing ownership.

**Workflow:**
1. Capture roadmap discussions as engrams.
2. Tag by theme, customer segment, and release cycle.
3. Use query + rehydrate when planning next quarter.
4. Share project-visible engrams for cross-team alignment.

**ContinuWitty Features Used:**
- Engram creation with tags/keywords
- Project defaults and project-level visibility
- Share/unshare controls

**Business Value:**
- Better continuity across quarters
- Less context fragmentation between squads

---

## 3) Security, Compliance, and Governance Teams

### Use Case: Controlled AI Knowledge Sharing

**Pain:** AI integrations are blocked when teams cannot enforce access boundaries and audit trails.

**Workflow:**
1. Define projects and membership roles.
2. Issue scoped MCP tokens (`read` or `write`, optional tool/project allowlists).
3. Monitor memory changes through project audit events.
4. Revoke tokens quickly when needed.

**ContinuWitty Features Used:**
- Project membership APIs
- MCP token issuance/revocation
- Project audit events
- Role-aware visibility checks

**Business Value:**
- Policy-safe AI enablement
- Clear accountability for memory mutations

---

## 4) AI Platform Teams (Agent Builders)

### Use Case: Agent That Does Not Forget Work Across Runs

**Pain:** Stateless agent loops repeat questions, miss prior decisions, and lose thread context.

**Workflow Pattern (MCP-first):**
1. Agent creates or loads a session (`chat.create_session`, `chat.list_sessions`).
2. Agent sends task messages (`chat.send_message`) and consumes retrieval metadata.
3. Agent saves milestones (`chat.save_as_engram`).
4. On next run, agent queries prior memory (`engram.query`) and rehydrates context (`engram.rehydrate`).
5. Agent continues into a new bounded session while preserving memory (`chat.continue_session`).

**ContinuWitty Features Used:**
- MCP transport (`/api/v1/mcp/stream`)
- Chat + engram toolset
- Scoped token policies
- Provenance-aware response metadata

**Business Value:**
- More reliable long-horizon agent behavior
- Lower hallucination risk from missing context

---

## 5) Developers and Startups

### Use Case: Solo Builder Memory Assistant

**Pain:** A single developer context-switches constantly and loses rationale for implementation choices.

**Workflow:**
1. Use web chat for iterative planning and coding notes.
2. Save session snapshots as engrams at feature milestones.
3. Query past engrams before new implementation decisions.
4. Export/import project bundles to move environments.

**ContinuWitty Features Used:**
- Local-first API + web app
- Session continuation and save-as-engram
- Project export/import

**Business Value:**
- Personal knowledge continuity
- Less duplicated trial-and-error

---

## End-to-End Workflow Catalog

## Workflow A: Capture Knowledge

Use when a conversation produced reusable value.

Steps:
1. Start session and discuss problem.
2. Save output as engram (`chat.save_as_engram`) or direct create (`engram.create`).
3. Attach tags/keywords/sources for retrieval quality.

Output:
- Durable engram with structured and markdown-friendly content.

---

## Workflow B: Retrieve and Reuse Knowledge

Use when starting a new task that may overlap older work.

Steps:
1. Query engrams (`engram.query`).
2. Rehydrate top results (`engram.rehydrate`).
3. Send chat message with retrieved context.

Output:
- Response with `used_engram_ids`, citations, and source references.

---

## Workflow C: Multi-Session Continuity

Use when work spans multiple sessions or threads.

Steps:
1. Keep active session for current goal.
2. Continue to a new session when context window must be reset (`chat.continue_session`).
3. Carry forward pinned memory artifacts.

Output:
- Controlled continuity without losing prior context.

---

## Workflow D: Graph-Driven Recall

Use when relationship between memories matters (dependencies, contradictions, derivations).

Steps:
1. Create links between related engrams (`engram.link_create`).
2. Ask for suggestions (`engram.link_suggest`) and hygiene checks.
3. Traverse paths (`engram.trace_path`) for explainable lineage.

Output:
- Higher-recall retrieval and inspectable memory chains.

---

## Workflow E: Document + Memory Blended Context

Use when answers require both durable memory and source documents.

Steps:
1. Ingest files/text into document store.
2. Pin relevant documents to session.
3. Query blended retrieval path for answer generation.

Output:
- Responses grounded in both engrams and document chunks.

---

## Workflow F: Team Collaboration and Governance

Use when multiple users need shared but controlled memory.

Steps:
1. Create projects and set defaults.
2. Add members with roles.
3. Share/unshare engrams by project visibility.
4. Review project audit events.

Output:
- Collaboration with explicit boundaries and accountability.

---

## Workflow G: Admin Memory Operations

Use when memory lifecycle cleanup or correction is needed.

Steps:
1. Admin lists sessions/engrams/collections.
2. Move engrams across projects when ownership changes.
3. Soft-delete and restore artifacts safely.
4. Manage collection contents.

Output:
- Safe, reversible operations instead of destructive deletion.

---

## Workflow H: Portability and Environment Transfer

Use when moving memory across projects or deployment stages.

Steps:
1. Export project bundle.
2. Import bundle into target project with conflict policy.
3. Verify outcomes and continue work from imported memory.

Output:
- Reusable memory assets across environments.

---

## Workflow I: Long-Running Agent Execution with Checkpoints

Use when autonomous or semi-autonomous work spans multiple hours/days.

Steps:
1. Start run with objective and initial notes (`POST /api/v1/agent-runs`).
2. Persist run state by `thread_id` for crash-safe continuation.
3. Resume the same run later (`POST /api/v1/agent-runs/{thread_id}/resume`).
4. Optionally persist final or periodic engram snapshots for durable reuse.

Output:
- Recoverable long-run execution state plus durable memory artifacts.

---

## MCP Integration Blueprint: "Agent That Does Not Forget"

## Why MCP Here

MCP gives agent runtimes a stable tool contract over JSON-RPC/SSE. ContinuWitty exposes memory operations as tools so the agent can persist and retrieve context instead of relying on short-lived prompt history.

## Integration Architecture

1. Agent runtime connects to `/api/v1/mcp/stream`.
2. Agent authenticates via session or bearer MCP token.
3. Agent discovers capabilities via `tools/list`.
4. Agent executes chat + engram tools in a durable loop.

## Minimal Durable Loop for Any Agent

1. `chat.create_session`
2. `chat.send_message`
3. `chat.save_as_engram` at milestones
4. `engram.query` at start of future tasks
5. `engram.rehydrate` for high-fidelity context
6. `chat.continue_session` when starting a fresh session boundary

## Reliability Rules for Agent Authors

- Always save milestones, not just final output.
- Always query before creating new engrams on similar topics.
- Use scoped tokens with least-privilege tool/project allowlists.
- Consume `used_engram_ids` and `source_references` for explainability.
- Keep memory operations deterministic and idempotent where possible.

## Example MCP Calls (Conceptual)

`tools/list`:
```json
{"jsonrpc":"2.0","id":"1","method":"tools/list","params":{}}
```

`tools/call` `chat.send_message`:
```json
{
  "jsonrpc": "2.0",
  "id": "2",
  "method": "tools/call",
  "params": {
    "name": "chat.send_message",
    "arguments": {
      "session_id": "<uuid>",
      "content_text": "Summarize prior incident learnings for this service",
      "stream": false
    }
  }
}
```

`tools/call` `chat.save_as_engram`:
```json
{
  "jsonrpc": "2.0",
  "id": "3",
  "method": "tools/call",
  "params": {
    "name": "chat.save_as_engram",
    "arguments": {
      "session_id": "<uuid>",
      "title": "Incident mitigation pattern",
      "project_id": "engram-vault"
    }
  }
}
```

---

## Current vs Upcoming Capability Notes

## Available Now (in this repository)

- Durable engram lifecycle (create/list/query/rehydrate)
- Chat continuity (send/stream/save-as-engram/continue)
- Link graph workflows (create/list/suggest/trace/hygiene)
- Project collaboration and admin memory operations
- MCP token auth with scoped authorization
- Export/import portability
- Observability endpoints and release/test gates

## Upcoming (Memory Intelligence Phases 36-40)

These are roadmap-aligned extensions that increase adaptive memory quality:
- Explicit feedback-driven relevance scoring
- Time-decay and consolidation suggestions
- Contradiction detection and warning flows
- Temporal and cost-aware context assembly
- Autonomous memory curation suggestions

Use this document for implementation planning, but treat roadmap items as planned unless explicitly marked complete in `Plan.md` and checkpoint logs.

---

## KPI Suggestions by Customer Type

| Customer Type | Suggested KPI | Why It Matters |
|---|---|---|
| Enterprise Engineering | % of sessions reusing prior engrams | Measures memory reuse over rework |
| Product/Program | Time to recover decision rationale | Measures planning continuity |
| AI Platform Team | Agent task success after restart | Measures anti-forgetting behavior |
| Compliance/Security | % tool calls blocked by policy as expected | Measures governance effectiveness |
| Solo Developer | Time to resume paused work | Measures personal continuity value |

---

## Quick Adoption Playbooks

## Enterprise Team Rollout (4 Weeks)

1. Week 1: Start with one high-context workflow (incident response or architecture reviews).
2. Week 2: Enforce project roles and scoped MCP tokens.
3. Week 3: Require save-as-engram at milestone boundaries.
4. Week 4: Track reuse and retrieval quality KPIs.

## Developer/Startup Rollout (1 Week)

1. Day 1: Stand up local stack (`make dev`).
2. Day 2: Use session + save-as-engram on one real feature.
3. Day 3: Add query-before-create rule for memory hygiene.
4. Day 4+: Integrate MCP into coding/research agent loop.

---

## Final Takeaway

ContinuWitty is most valuable when treated as an operating workflow, not just a storage feature.

If teams and agents consistently follow capture -> query -> rehydrate -> continue, they stop losing context, make better decisions with provenance, and scale AI-assisted work without sacrificing control.
