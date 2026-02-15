# Engram Vault (Local-First MVP)

This project turns long LLM research runs into durable, queryable "memory engrams" so context does not disappear between sessions.

This README is written for a newcomer and follows an implementation sequence based on `deep-research-report.md`.

## Current Progress

- [x] Read and convert research report into a local-first implementation plan.
- [x] Define local architecture and data flow.
- [x] Create local development scaffold (`FastAPI` + `Postgres` + `pgvector`).
- [x] Add base database schema for engrams, sources, artifacts.
- [x] Add MVP API endpoints: create, list, query, rehydrate.
- [x] Add deterministic local embedding function (no external model dependency).
- [x] Add full local test suite (unit + API + integration).
- [x] Migrate setup to `uv` with lockfile-based dependencies.
- [x] Add lint/format best-practice gates with `ruff`.
- [ ] Add LangGraph durable run pipeline integration.
- [ ] Add CLI/UI for upload/search/rehydrate workflows.
- [ ] Add automated evaluation harness (temporal + multi-engram reasoning).
- [ ] Add security hardening and RBAC.

## Why This Exists

Long research threads lose useful context once a session ends. The goal here is to store:

- What was concluded
- Why it was concluded
- Which sources supported each claim
- How to rehydrate those conclusions into any future LLM session

## Local-First Architecture

```text
+---------------------------+         +-------------------------------+
| Research Agent / User     |         | Optional Local UI / CLI       |
| (today: manual ingestion) |         | (next milestone)              |
+-------------+-------------+         +---------------+---------------+
              |                                         |
              | POST /api/v1/engrams                   | POST /api/v1/engrams/query
              v                                         v
       +-------------------------------------------------------+
       | FastAPI Engram Vault API                              |
       | - Validation (engram schema)                          |
       | - Local embedding generation (deterministic)          |
       | - Retrieval + metadata filtering                      |
       | - Rehydration bundle builder                          |
       +---------------------------+---------------------------+
                                   |
                                   | SQL + pgvector
                                   v
                    +-------------------------------------+
                    | Postgres + pgvector (local Docker) |
                    | - engrams                          |
                    | - sources                          |
                    | - artifacts                        |
                    +-------------------------------------+
```

## Data Flow (One By One)

1. Capture run output (summary, decisions, claims, assumptions, questions).
2. Attach provenance (source URLs, timestamps, snippets).
3. Persist as `MemoryEngram` JSON + markdown + retrieval embedding.
4. Query by semantic intent + metadata filters.
5. Build a compact rehydration bundle for future LLM sessions.

## Repository Layout

```text
engram/
  README.md
  deep-research-report.md
  Engram_Vault_Plan.md
  .env.example
  docker-compose.yml
  Makefile
  db/
    init/
      001_schema.sql
  api/
    pyproject.toml
    uv.lock
    app/
      __init__.py
      config.py
      db.py
      embedding.py
      main.py
      models.py
      repository.py
```

## File-by-File Guide

- `docker-compose.yml`: Local Postgres + pgvector service.
- `db/init/001_schema.sql`: Database extension, tables, and indexes.
- `api/app/main.py`: FastAPI routes and API surface.
- `api/app/models.py`: Request/response and engram schema models.
- `api/app/repository.py`: SQL persistence, semantic query, rehydration builder.
- `api/app/embedding.py`: Deterministic local embedding helper.
- `api/app/db.py`: DB connection lifecycle.
- `api/app/config.py`: Environment-backed settings.
- `api/pyproject.toml`: project dependencies, pytest config, and ruff config.
- `api/uv.lock`: locked dependency graph for reproducible local runs.
- `api/tests/conftest.py`: API client, schema bootstrap, DB cleanup fixtures.
- `api/tests/test_api_unit.py`: unit tests for API routing behavior.
- `api/tests/test_api_integration.py`: full API + DB integration tests.
- `api/tests/test_embedding.py`: embedding utility tests.
- `api/tests/test_repository_helpers.py`: repository helper tests.
- `Makefile`: Local run shortcuts.
- `.env.example`: Starter configuration for local setup.

## Prerequisites

- Docker + Docker Compose
- `uv` (`uv --version` should work)
- `psql` client optional (for manual inspection)

## Quick Start (Local Only)

1. Copy environment file:

```bash
cp .env.example .env
```

2. Start database:

```bash
docker compose up -d db
```

If this fails with "Cannot connect to the Docker daemon", start Docker Desktop first, wait until it is healthy, then rerun:

```bash
docker compose up -d db
docker compose ps
```

3. Sync dependencies with `uv`:

```bash
cd api
uv sync --group dev
cd ..
```

4. Run API:

```bash
make api
```

5. Open API docs:

- [http://localhost:8000/docs](http://localhost:8000/docs)

6. Run tests:

```bash
make test
```

Use these for targeted runs:

```bash
make test-unit
make test-integration
```

7. Run lint/format checks:

```bash
make lint
make format-check
```

8. Run all quality gates together:

```bash
make check
```

## Workflow Notes (Before Phase 2)

Use this exact flow while you validate Milestone 1 behavior.

1. Start infra:

```bash
make db-up
docker compose ps
```

Comment: confirms local Postgres/pgvector is healthy for integration tests.

2. Sync environment:

```bash
make sync
```

Comment: enforces reproducible dependency versions from `api/uv.lock`.

3. Run static quality checks:

```bash
make lint
make format-check
```

Comment: catches style/import/bug-prone patterns before runtime testing.

4. Run automated tests:

```bash
make test
```

Comment: verifies API and DB behavior end-to-end.

5. Manual API smoke test:

```bash
make api
```

Comment: open `http://localhost:8000/docs`, run create/list/query/rehydrate once.

Phase 2 is intentionally paused until this validation pass is complete.

## Implementation Log

### 2026-02-15 (Completed in this pass)

1. Read `deep-research-report.md` and extracted MVP scope into this README.
2. Created local-first architecture and data flow.
3. Added `docker-compose.yml` with local `pgvector/pgvector:pg16`.
4. Added base SQL schema: `engrams`, `sources`, `artifacts`.
5. Added FastAPI project with endpoints for create/list/query/rehydrate.
6. Added deterministic local embedding to keep everything provider-free.
7. Performed validation:
   - `python3 -m compileall api/app` passed.
   - `docker compose config` passed.
   - `docker compose up -d db` required Docker daemon startup.

### 2026-02-15 (Testing pass)

1. Added full test suite under `api/tests`:
   - unit tests for embedding and repository helper logic
   - API unit tests using repository function mocks
   - integration tests with local Postgres/pgvector
2. Added test markers and commands:
   - `api/pyproject.toml` (`tool.pytest.ini_options`)
   - `make test`
   - `make test-unit`
   - `make test-integration`
3. Ran the full suite locally:
   - `make test` -> `13 passed`
   - `make test-unit` -> `11 passed, 2 deselected`
   - `make test-integration` -> `2 passed, 11 deselected`

### 2026-02-15 (uv + lint hardening pass)

1. Migrated to `uv`:
   - added `api/pyproject.toml`
   - generated `api/uv.lock`
   - removed legacy `api/requirements.txt` and standalone `api/pytest.ini`
2. Added quality gates:
   - `make lint` (`ruff check`)
   - `make format` / `make format-check` (`ruff format`)
   - `make check` (lint + format-check + tests)
3. Applied lint-driven code cleanups:
   - modern typing imports
   - `datetime.UTC` usage
   - single combined context managers for DB operations
4. Re-ran verification:
   - `make lint` -> all checks passed
   - `make format-check` -> all files formatted
   - `make test` -> `13 passed`

### Next Immediate Steps (One By One)

1. Run the workflow notes section and complete your manual validation pass.
2. Integrate LangGraph checkpointing and thread resume support.
3. Add automatic end-of-run engram writes from the agent workflow.
4. Add optional periodic snapshot engrams for long runs.
5. Add minimal UI for list/search/detail/rehydrate actions.

## MVP API Surface

- `POST /api/v1/engrams`
- `GET /api/v1/engrams`
- `POST /api/v1/engrams/query`
- `GET /api/v1/engrams/{engram_id}/rehydrate`
- `GET /healthz`

## Test Suite

Test files live under `api/tests`:

- `test_embedding.py`: deterministic embedding behavior.
- `test_repository_helpers.py`: retrieval text and vector literal helpers.
- `test_api_unit.py`: endpoint behavior with repository function mocking.
- `test_api_integration.py`: end-to-end API roundtrip against local Postgres.
- `conftest.py`: DB fixture, schema bootstrap, and cleanup.

Notes:

- Integration tests are marked with `@pytest.mark.integration` and configured in `api/pyproject.toml`.
- If the DB is unavailable, integration tests are skipped with a clear reason.

## MemoryEngram Contract (MVP)

Stored in two layers:

- Structured JSON (`engram_json`) for long-term machine use
- Markdown (`engram_markdown`) for human readability

Key fields:

- Identity: `engram_id`, `project_id`, optional `thread_id`
- Time: `created_at`, source `captured_at`
- Analysis: `title`, `abstract`, `detailed_summary_markdown`
- Reasoning: `decisions`, `assumptions`, `open_questions`
- Evidence: `claims[*].supporting_sources[*]`
- Retrieval hooks: `tags`, `keywords`
- Pointers: `artifacts`

## Local Embedding Strategy (Temporary)

For local-first development, this repo uses a deterministic hash-based embedding function to avoid external model calls.

- Pros: zero external dependencies, reproducible tests, fast setup
- Cons: weak semantic quality vs real embedding models

Planned upgrade: swap to a local embedding model (e.g. sentence-transformers) or hosted provider while preserving schema.

## Security Baseline (MVP)

- No external network dependency for memory pipeline
- DB credentials from environment variables
- Keep sensitive source text in local database only
- Add encryption, secrets manager, and RBAC before production

## Implementation Milestones

### Milestone 1 (Completed)

- Local DB + schema created
- API skeleton created
- CRUD + query + rehydrate endpoints wired

### Milestone 2 (Next)

- Integrate LangGraph run checkpointing
- Automatic engram write at end of each research run
- Optional periodic snapshot engrams for long runs

### Milestone 3

- Add minimal UI (list/search/detail/rehydrate copy action)
- Add source-inspection view for provenance

### Milestone 4

- Build eval harness:
  - fact recall
  - cross-engram reasoning
  - temporal updates
  - abstention checks

## Example: Create Engram

```bash
curl -X POST http://localhost:8000/api/v1/engrams \
  -H "Content-Type: application/json" \
  -d '{
    "project_id": "engram-vault",
    "thread_id": "run-001",
    "title": "RAG framework comparison",
    "abstract": "LangGraph + Postgres/pgvector is the local MVP choice.",
    "detailed_summary_markdown": "Detailed notes here.",
    "decisions": [{"decision": "Use LangGraph", "rationale": "Durable checkpoints"}],
    "assumptions": ["Single-tenant local setup"],
    "open_questions": ["When to add reranking model?"],
    "claims": [{
      "claim": "Hybrid retrieval improves robustness.",
      "supporting_sources": [{
        "url": "https://example.com/article",
        "title": "Hybrid retrieval notes",
        "snippet": "Dense+sparse outperforms single method.",
        "captured_at": "2026-02-15T10:00:00Z"
      }]
    }],
    "tags": ["rag", "memory"],
    "keywords": ["langgraph", "pgvector"],
    "artifacts": [{"artifact_type": "markdown", "storage_uri": "local://notes/run-001.md"}]
  }'
```

## Example: Query Engrams

```bash
curl -X POST http://localhost:8000/api/v1/engrams/query \
  -H "Content-Type: application/json" \
  -d '{
    "query": "Which framework was selected for durable long runs?",
    "project_id": "engram-vault",
    "top_k": 5
  }'
```

## "Successful If" Checklist

- [x] I can save a full research result as an engram in one API call.
- [x] Each major claim has URL + snippet + timestamp provenance.
- [x] I can retrieve relevant engrams by semantic query + metadata filters.
- [x] I can generate an LLM-ready rehydration bundle from any engram.
- [x] A newcomer can run the system locally using this README alone.
- [ ] The same stored engram can be reused with different LLM providers later.
