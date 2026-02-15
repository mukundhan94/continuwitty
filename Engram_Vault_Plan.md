# Persistent LLM Research Memory (Engram Vault) --- Detailed Phased Plan

## Goal

Build a system that preserves long research/analysis runs (≈1 hour of
browsing + synthesis) as durable, model-agnostic "memory engrams" that
you can query days/weeks later, with citations/provenance and an
LLM-ready rehydration bundle.

------------------------------------------------------------------------

## Phase 0 --- Foundations & Success Criteria (Week 0)

### Outcomes

-   Clear scope and definitions: run, engram, artifact, rehydration
    bundle, provenance.
-   Defined success criteria and threat model.

### Deliverables

-   Success criteria checklist.
-   Privacy and retention baseline.
-   Decision memo: LangGraph + Postgres (pgvector) as MVP stack.

------------------------------------------------------------------------

## Phase 1 --- Landscape Survey & Tech Choices

### Outcomes

-   Chosen framework and storage strategy.

### Key Decisions

-   Use LangGraph for durable agents.
-   Use Postgres + pgvector for unified JSON + vector storage.
-   Begin with vector + metadata filtering retrieval.

------------------------------------------------------------------------

## Phase 2 --- Engram Schema Design

### Outcomes

-   Stable, versioned MemoryEngram object.

### Core Fields

-   Research objective
-   Abstract
-   Detailed summary (Markdown)
-   Decisions + rationale
-   Assumptions
-   Open questions
-   Claims with evidence
-   Metadata (timestamps, tags, keywords)

------------------------------------------------------------------------

## Phase 3 --- Storage Layer

### Minimum Tables

-   engrams (JSON + markdown + embedding)
-   sources (url, snippet, captured_at)
-   optional artifacts table

### Requirements

-   Embedding version tracking
-   Content hash for deduplication
-   Indexing on project_id + created_at

------------------------------------------------------------------------

## Phase 4 --- Ingestion Pipeline

### Flow

1.  Capture sources + notes
2.  Consolidate via LLM into structured engram
3.  Embed compact retrieval text
4.  Persist engram + sources

------------------------------------------------------------------------

## Phase 5 --- Retrieval & Rehydration

### API Endpoints

-   POST /engrams
-   GET /engrams
-   POST /engrams/query
-   GET /engrams/{id}/rehydrate

### Rehydration Bundle Includes

-   Context markdown
-   Top citations
-   Referenced engram IDs

------------------------------------------------------------------------

## Phase 6 --- Agent Durability

### With LangGraph

-   Checkpoint long runs
-   Resume capability
-   Automatic engram write at completion

### Local MVP Status

-   Implemented with `api/app/agent_workflow.py` using SQLite checkpoints.
-   Exposed endpoints:
    - `POST /api/v1/agent-runs`
    - `GET /api/v1/agent-runs/{thread_id}`
    - `POST /api/v1/agent-runs/{thread_id}/resume`

------------------------------------------------------------------------

## Phase 7 --- Local Test UI + Simple Login

### Features

- Login page for local operator access
- Session-based auth (local/dev baseline)
- Browser dashboard to call create/list/query/rehydrate endpoints
- Logout + route protection for UI pages

------------------------------------------------------------------------

## Phase 8 --- MVP UI Expansion

### Features

-   Engram list view
-   Search
-   Detail page
-   Rehydrate button

------------------------------------------------------------------------

## Phase 9 --- Evaluation & Hardening

### Tests

-   Fact recall accuracy
-   Cross-engram reasoning
-   Temporal correctness
-   Abstention behavior

------------------------------------------------------------------------

## Acceptance Criteria

-   Engrams are versioned and durable.
-   Each key claim has provenance.
-   Semantic + filtered search works.
-   Rehydration bundle works across models.
-   Long runs resume safely.
-   UI demo functional end-to-end.
