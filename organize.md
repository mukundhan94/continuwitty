# Documentation Reorganization Plan

## Context

The project documentation has grown organically over 31+ implementation phases. README.md alone is 3,151 lines — containing implementation logs, file-by-file guides, env var matrices, curl examples, milestone trackers, MCP tool catalogs, and test suite listings alongside the actual project overview. This makes it hard for newcomers to orient and for maintainers to keep content current.

The goal is to restructure documentation into focused, interlinked files where README becomes a clean landing page, and specialized docs cover API, MCP, workflows, testing, and environment configuration.

---

## Target File Structure

```
engram/
  README.md                      (~200 lines, slim landing page)
  AGENT.md                       (~200 lines, updated with learnings)
  AGENTS.md                      (unchanged)
  Plan.md                        (cleaned up, remove duplicate refactor section)
  checkpoint.md                  (NEW — milestones + phase progress)
  todo.md                        (NEW — pending work tracker)
  refactor.md                    (unchanged)
  deep-research-report.md        (unchanged)
  Link.md                        (DELETE — fully redundant with Plan.md)
  docs/
    architecture-playbook.md     (unchanged)
    api-reference.md             (NEW — all REST endpoints + contracts + curl examples)
    mcp-guide.md                 (NEW — transport, auth, tools, token workflow, SSE framing)
    user-workflows.md            (NEW — docker setup, validation, admin runbook, CLI)
    testing-guide.md             (NEW — test commands, suites, acceptance, eval)
    env-reference.md             (NEW — all env vars organized by category)
    implementation-log.md        (NEW — full chronological log moved from README)
    mcp-client-integrations.md   (unchanged)
    user-flow-engram-workflow.md (unchanged)
```

---

## Content Migration Map

| README Section | Lines | Destination |
|----------------|-------|-------------|
| Unified Plan Status | 95–109 | `checkpoint.md` (Current State Summary) |
| Phase 31 Progress Tracker | 110–165 | `checkpoint.md` (Active Phase Progress) |
| Agent Guide | 166–170 | 1-line pointer in README Docs index |
| Agents & Skills Catalog | 172–199 | 1-line pointer to `AGENTS.md` |
| Repository Layout tree | 246–464 | **REMOVE** (too granular; `tree`/`ls` is better) |
| File-by-File Guide | 466–618 | **REMOVE** (maintenance burden; architecture-playbook covers) |
| Admin Console Test Runbook | 711–753 | `docs/user-workflows.md` |
| Env vars (security/debug/provider) | 747–799 | `docs/env-reference.md` |
| Backend/web test commands | 800–837 | `docs/testing-guide.md` |
| Dockerized Stack | 849–892 | `docs/user-workflows.md` |
| Docker Env Matrix | 884–891 | `docs/env-reference.md` |
| Acceptance Tests | 893–959 | `docs/testing-guide.md` |
| Workflow Notes | 960–1089 | `docs/user-workflows.md` |
| High-Impact Test Workflow | 1090–1130 | `docs/testing-guide.md` |
| Implementation Log | 1131–2245 | `docs/implementation-log.md` (verbatim) |
| Next Immediate Steps | 2238–2245 | `todo.md` |
| MVP API Surface | 2246–2314 | `docs/api-reference.md` |
| MCP Stream Usage | 2315–2329 | `docs/mcp-guide.md` |
| MCP Token Workflow | 2330–2431 | `docs/mcp-guide.md` |
| MCP examples/tools/framing | 2432–2766 | `docs/mcp-guide.md` |
| Auto Metadata Enrichment | 2768–2776 | `docs/api-reference.md` |
| CLI Workflow | 2777–2821 | `docs/user-workflows.md` |
| Test Suite listing | 2822–2879 | `docs/testing-guide.md` |
| MemoryEngram Contract | 2880–2896 | `docs/api-reference.md` |
| Local Embedding Strategy | 2897–2904 | `docs/api-reference.md` |
| Security Baseline | 2906–2912 | `docs/api-reference.md` |
| Implementation Milestones 1–29 | 2913–3071 | `checkpoint.md` |
| Example curl blocks | 3072–3134 | `docs/api-reference.md` |

---

## Execution Steps

### Step 1: Create `docs/implementation-log.md`
- **Source:** README.md lines 1131–2245 (all `### 2026-02-*` entries)
- **Action:** Copy verbatim into new file with header and back-link to README
- No dependencies on other new files

### Step 2: Create `docs/env-reference.md`
- **Sources:**
  - README lines 747–799 (security/debug/provider/LangGraph env vars)
  - README lines 884–891 (Docker Env Matrix)
  - README lines 887–891 (web/acceptance env vars)
- **Structure:** Categorized by: Database, API Runtime, Security, Debug/Observability, Providers, Embedding, Ingestion, Web Runtime, Acceptance Tests, LangGraph

### Step 3: Create `docs/api-reference.md`
- **Sources:**
  - README lines 2246–2314 (MVP API Surface — full endpoint list)
  - README lines 2768–2776 (Auto Metadata Enrichment)
  - README lines 2880–2912 (MemoryEngram Contract, Embedding Strategy, Security Baseline)
  - README lines 3072–3134 (Example curl blocks for create/query/agent-run/sources)
- **Structure:** Grouped by domain (Engram, Chat, Ingestion, Admin Memory, Projects, Agent Runs, OAuth, MCP Tokens, Health), with curl examples inline
- **Cross-links:** → `docs/mcp-guide.md`, `docs/env-reference.md`

### Step 4: Create `docs/mcp-guide.md`
- **Sources:**
  - README lines 2315–2329 (MCP Stream Usage: endpoint, auth)
  - README lines 2330–2431 (MCP Token Workflow: create/use/revoke, policy model)
  - README lines 2398–2766 (remaining MCP sections: SSE framing, tools/list, tools/call, direct tool examples, typed client helpers)
- **Structure:** Transport → Authentication (session/bearer/OAuth) → Token Workflow → Compatibility Methods → Tool Catalog → Invocation Styles → Typed Clients
- **Cross-links:** → `docs/mcp-client-integrations.md`, `docs/api-reference.md`

### Step 5: Create `docs/testing-guide.md`
- **Sources:**
  - README lines 800–837 (make test/web-check/test-unit/lint/check/eval commands)
  - README lines 893–959 (Acceptance Tests: local/docker/live runners, feature coverage)
  - README lines 1090–1130 (High-Impact Test Workflow)
  - README lines 2822–2879 (Test Suite: backend + frontend file listings)
- **Structure:** Quick Commands → Backend Test Suite → Frontend Test Suite → Acceptance Tests → Evaluation Harness → Test Policy (from AGENT.md §9)
- **Cross-links:** → `AGENT.md` §9, `docs/user-workflows.md`

### Step 6: Create `docs/user-workflows.md`
- **Sources:**
  - README lines 711–753 (Admin Console Test Runbook)
  - README lines 849–892 (Dockerized Stack setup)
  - README lines 960–1089 (Workflow Notes: validation sequence)
  - README lines 2777–2821 (CLI Workflows)
- **Structure:** Dockerized Stack Setup → Local Validation Sequence → High-Impact Multi-Provider Test → Admin Console Runbook → CLI Workflows
- **Cross-links:** → `docs/user-flow-engram-workflow.md`, `docs/testing-guide.md`, `docs/env-reference.md`

### Step 7: Create `checkpoint.md`
- **Sources:**
  - README lines 95–109 (Unified Plan Status summary)
  - README lines 110–165 (Phase 31 Progress Tracker)
  - README lines 2913–3071 (Implementation Milestones 1–29)
- **Structure:** Current State Summary → Active Phase Progress (Phase 31 tracker) → Completed Milestones (1–29) → Phase Completion History (summary table)
- **Cross-links:** → `Plan.md`, `todo.md`, `refactor.md`

### Step 8: Create `todo.md`
- **Sources:** Derived from:
  - README lines 2238–2245 (Next Immediate Steps)
  - Plan.md lines 755–771 (Near-Term Execution Order)
  - README unchecked items from Current Progress (lines 92–94)
- **Structure:** Immediate (current sprint) → Near-Term (next phases) → Future (post-Phase 23) → Documentation Debt → Technical Debt
- **Cross-links:** → `Plan.md`, `checkpoint.md`

### Step 9: Clean up `Plan.md`
- **Action:** Remove the duplicate "2026-02-19 Refactor Checkpoint" section (Plan.md lines 708–742) — this content already lives in `refactor.md`
- **Action:** Add note at top: "See [checkpoint.md](checkpoint.md) for milestone tracking and [todo.md](todo.md) for pending work"
- **Action:** Update cross-phase working rule #2 to reference `docs/implementation-log.md` instead of just README

### Step 10: Slim `README.md`
This is the core transformation — README goes from ~3,151 lines to ~200 lines.

**Keep (rewrite/compress as needed):**
- Title + 1-paragraph description (lines 1–5)
- New expanded Documentation Index (replace old Docs section)
- Current Progress checklist (lines 33–94) — this is the project's heartbeat
- Why This Exists (lines 201–208)
- Local-First Architecture ASCII diagram (lines 210–236)
- Data Flow (lines 238–244)
- Prerequisites (lines 619–623)
- Quick Start steps 1–9 (lines 625–699) — shortened, no runbook
- "Successful If" Checklist (lines 3135–3152)

**Remove entirely (moved to dedicated docs):**
- Unified Plan Status block (→ `checkpoint.md`)
- Phase 31 Progress Tracker (→ `checkpoint.md`)
- Agents & Skills Catalog verbose listing (→ 1-line pointer to `AGENTS.md`)
- Repository Layout tree + File-by-File Guide (→ removed, too granular)
- Admin Console Test Runbook (→ `docs/user-workflows.md`)
- All env var blocks (→ `docs/env-reference.md`)
- Dockerized Stack section (→ `docs/user-workflows.md`)
- Acceptance Tests section (→ `docs/testing-guide.md`)
- Workflow Notes (→ `docs/user-workflows.md`)
- High-Impact Test Workflow (→ `docs/testing-guide.md`)
- Implementation Log (→ `docs/implementation-log.md`)
- MVP API Surface (→ `docs/api-reference.md`)
- MCP sections (→ `docs/mcp-guide.md`)
- CLI Workflow (→ `docs/user-workflows.md`)
- Test Suite listing (→ `docs/testing-guide.md`)
- MemoryEngram Contract + Embedding + Security (→ `docs/api-reference.md`)
- Implementation Milestones (→ `checkpoint.md`)
- Example curl blocks (→ `docs/api-reference.md`)
- Next Immediate Steps (→ `todo.md`)

### Step 11: Update `AGENT.md`
Add two new sections:

**§13 — Accumulated Project Learnings:**
- Architecture patterns that worked (domain module layout, centralized enrichment, typed MCP clients, fill-empty metadata, token scope model)
- Common pitfalls (no business logic in main.py, no raw SQL in services, use theme tokens, stable MCP tool names, SSE CRLF framing, Bedrock error classification)
- Schema evolution guidelines (single bootstrap file, IF NOT EXISTS, soft-delete patterns, idempotent backfill)
- Testing insights (lifecycle policy needs 3-layer regression, provider error classification tests, deterministic mocks for CI, ThemeProvider in frontend tests)
- MCP development notes (stable verb-object names, auth parity with REST, progress frame correlation, JSON-RPC notifications)

**§14 — Documentation Maintenance:**
- README is a slim landing page — do not add detailed content
- Route API changes → `docs/api-reference.md`
- Route MCP changes → `docs/mcp-guide.md`
- Route env var additions → `docs/env-reference.md`
- Route implementation log entries → `docs/implementation-log.md`
- Route milestone completions → `checkpoint.md`
- When phase completes: update `checkpoint.md`, `todo.md`, `Plan.md`

**Update existing sections:**
- §2 rule "Update README.md in every implementation phase" → "Update `docs/implementation-log.md` and `checkpoint.md` in every implementation phase"
- §12 maintenance cadence: reference new doc file locations

### Step 12: Delete `Link.md`
- Confirmed: Plan.md already contains all phases 24–28 and Phase 32 content
- Link.md is fully redundant

### Step 13: Verification
- Check all cross-links resolve (grep for broken relative links)
- Run `make check` to ensure no runtime breakage
- Verify git status shows expected file changes

---

## Critical Files

| File | Action |
|------|--------|
| `README.md` | Major rewrite (3151 → ~200 lines) |
| `AGENT.md` | Add §13 + §14, update §2 and §12 |
| `Plan.md` | Remove duplicate refactor section, add cross-links |
| `Link.md` | Delete |
| `checkpoint.md` | Create new |
| `todo.md` | Create new |
| `docs/api-reference.md` | Create new |
| `docs/mcp-guide.md` | Create new |
| `docs/user-workflows.md` | Create new |
| `docs/testing-guide.md` | Create new |
| `docs/env-reference.md` | Create new |
| `docs/implementation-log.md` | Create new |

---

## Verification

1. All new docs have proper headers and cross-links
2. README Documentation Index links resolve to existing files
3. No content is lost — every section moved has a destination
4. `make check` still passes (no code changes, docs only)
5. `git diff --stat` shows expected file additions/modifications/deletions
