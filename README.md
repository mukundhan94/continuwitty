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
- [x] Add local UI test harness with simple login workflow.
- [x] Add LangGraph checkpoint/resume workflow with automatic engram write.
- [x] Add optional periodic snapshot engrams for long-running threads.
- [x] Add provenance source-inspection endpoint and UI flow.
- [x] Add CSRF-protected UI auth with optional hashed-password support.
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
      agent_models.py
      agent_workflow.py
      config.py
      db.py
      embedding.py
      main.py
      models.py
      repository.py
      templates/
        login.html
        dashboard.html
```

## File-by-File Guide

- `docker-compose.yml`: Local Postgres + pgvector service.
- `db/init/001_schema.sql`: Database extension, tables, and indexes.
- `api/app/main.py`: FastAPI routes and API surface.
- `api/app/agent_models.py`: request/response models for agent runs.
- `api/app/agent_workflow.py`: LangGraph workflow, checkpointing, and resume logic.
- `api/app/auth.py`: password hashing/verification and CSRF token helpers.
- `api/app/models.py`: Request/response and engram schema models.
- `api/app/repository.py`: SQL persistence, semantic query, rehydration builder.
- `api/app/embedding.py`: Deterministic local embedding helper.
- `api/app/db.py`: DB connection lifecycle.
- `api/app/config.py`: Environment-backed settings.
- `api/app/templates/login.html`: local sign-in page for UI testing.
- `api/app/templates/dashboard.html`: authenticated local dashboard for API testing.
- `api/pyproject.toml`: project dependencies, pytest config, and ruff config.
- `api/uv.lock`: locked dependency graph for reproducible local runs.
- `api/tests/conftest.py`: API client, schema bootstrap, DB cleanup fixtures.
- `api/tests/test_api_unit.py`: unit tests for API routing behavior.
- `api/tests/test_api_integration.py`: full API + DB integration tests.
- `api/tests/test_ui_auth.py`: login/logout/session workflow tests.
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

UI testing entrypoints:

- [http://localhost:8000/login](http://localhost:8000/login)
- [http://localhost:8000/ui](http://localhost:8000/ui)

Default local UI credentials (override in `.env` if needed):

- username: `admin`
- password: `admin123`
- optional hash override: `UI_DEMO_PASSWORD_HASH` (if set, plain password env is ignored)

LangGraph checkpoint file (local):

- `LANGGRAPH_CHECKPOINT_PATH=./data/langgraph_checkpoints.sqlite`

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

## Workflow Notes (Before Phase 5)

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

Comment: open `http://localhost:8000/login`, sign in, then use `/ui` to run create/list/query/rehydrate from the dashboard.

Then test durable agent runs:

```bash
curl -X POST http://localhost:8000/api/v1/agent-runs \
  -H "Content-Type: application/json" \
  -d '{
    "project_id": "engram-vault",
    "thread_id": "thread-manual-001",
    "objective": "Validate checkpoint + resume flow",
    "notes": ["Initial run note"],
    "auto_persist_engram": true
  }'
```

```bash
curl -X POST http://localhost:8000/api/v1/agent-runs/thread-manual-001/resume \
  -H "Content-Type: application/json" \
  -d '{
    "notes": ["Resumed run note"],
    "auto_persist_engram": false
  }'
```

```bash
curl http://localhost:8000/api/v1/agent-runs/thread-manual-001
```

Then test periodic snapshots for longer runs:

```bash
curl -X POST http://localhost:8000/api/v1/agent-runs \
  -H "Content-Type: application/json" \
  -d '{
    "project_id": "engram-vault",
    "thread_id": "thread-snapshot-001",
    "objective": "Validate periodic snapshot behavior",
    "notes": ["note-1", "note-2", "note-3"],
    "snapshot_enabled": true,
    "snapshot_every_n_notes": 2,
    "auto_persist_engram": false
  }'
```

Phase 5 evaluation work is intentionally paused until this validation pass is complete.

If you want to use a hashed local UI password instead of plaintext in `.env`, generate one with:

```bash
cd api
uv run python -c "from app.auth import hash_password; print(hash_password('admin123'))"
```

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

### 2026-02-15 (UI login phase)

1. Added local browser UI with session-based login:
   - `/login` sign-in page
   - `/ui` authenticated dashboard for manual API testing
2. Added UI dependencies and config:
   - `jinja2`, `python-multipart`, `itsdangerous`
   - new settings: `APP_SESSION_SECRET`, `UI_DEMO_USERNAME`, `UI_DEMO_PASSWORD`
3. Added auth workflow tests:
   - unauthenticated redirect checks
   - invalid login rejection
   - login/logout session behavior
4. Verification:
   - `make lint` -> all checks passed
   - `make test` -> `17 passed`

### 2026-02-15 (LangGraph durability phase)

1. Added LangGraph-backed agent workflow service:
   - state graph nodes: collect -> synthesize -> snapshot -> persist
   - SQLite checkpointing for thread persistence
   - resume support by `thread_id`
2. Added agent-run API endpoints:
   - `POST /api/v1/agent-runs`
   - `GET /api/v1/agent-runs/{thread_id}`
   - `POST /api/v1/agent-runs/{thread_id}/resume`
3. Added automatic engram persistence at workflow completion (`auto_persist_engram` toggle).
4. Added optional periodic snapshot engrams for long runs:
   - `snapshot_enabled`
   - `snapshot_every_n_notes`
   - tracked `snapshot_engram_ids` and `snapshot_count`
5. Added integration tests for run/create/resume/state retrieval and snapshot behavior.
6. Verification:
   - `make lint` -> all checks passed
   - `make test` -> `23 passed`

### 2026-02-15 (Auth hardening + provenance inspection phase)

1. Added UI auth hardening:
   - CSRF token validation for `/login` and `/logout`
   - optional hashed password support via `UI_DEMO_PASSWORD_HASH`
2. Added provenance inspection support:
   - endpoint `GET /api/v1/engrams/{engram_id}/sources`
   - dashboard `Inspect Sources` action in `/ui`
3. Added tests:
   - CSRF and logout protection tests
   - sources endpoint integration tests
4. Verification:
   - `make lint` -> all checks passed
   - `make test` -> `27 passed`

### Next Immediate Steps (One By One)

1. Run the workflow notes section and complete your manual validation pass.
2. Add multi-user auth model + role-based access controls.
3. Build evaluation harness for temporal and cross-engram reasoning.
4. Add reranking/citation-packing improvements for retrieval quality.
5. Add background consolidation jobs for long-run memory maintenance.

## MVP API Surface

- `POST /api/v1/engrams`
- `GET /api/v1/engrams`
- `POST /api/v1/engrams/query`
- `GET /api/v1/engrams/{engram_id}/sources`
- `GET /api/v1/engrams/{engram_id}/rehydrate`
- `POST /api/v1/agent-runs`
- `GET /api/v1/agent-runs/{thread_id}`
- `POST /api/v1/agent-runs/{thread_id}/resume`
- `GET /healthz`

## Test Suite

Test files live under `api/tests`:

- `test_embedding.py`: deterministic embedding behavior.
- `test_repository_helpers.py`: retrieval text and vector literal helpers.
- `test_api_unit.py`: endpoint behavior with repository function mocking.
- `test_api_integration.py`: end-to-end API roundtrip + sources endpoint checks.
- `test_ui_auth.py`: login/logout/session-protected UI + CSRF checks.
- `test_agent_workflow.py`: LangGraph checkpoint/resume + auto-persist + snapshot checks.
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

### Milestone 2 (Completed)

- Added local UI login workflow for manual testing
- Added authenticated dashboard for create/list/query/rehydrate API calls
- Added UI auth tests for login/logout/session redirects

### Milestone 3 (Completed)

- LangGraph run checkpointing integrated
- Automatic engram write at end of each research run integrated
- Optional periodic snapshot engrams integrated for long-running threads

### Milestone 4 (In Progress)

- Added CSRF protection for login/logout UI forms
- Added optional hashed-password authentication path (`UI_DEMO_PASSWORD_HASH`)
- Added source-inspection endpoint and dashboard workflow
- Pending: multi-user auth model and role-based access controls

### Milestone 5 (Next)

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

## Example: Agent Run (With Snapshots)

```bash
curl -X POST http://localhost:8000/api/v1/agent-runs \
  -H "Content-Type: application/json" \
  -d '{
    "project_id": "engram-vault",
    "thread_id": "thread-demo-001",
    "objective": "Track long run with periodic snapshots",
    "notes": ["note-1", "note-2", "note-3", "note-4"],
    "snapshot_enabled": true,
    "snapshot_every_n_notes": 2,
    "auto_persist_engram": true
  }'
```

## Example: Inspect Engram Sources

```bash
curl http://localhost:8000/api/v1/engrams/<engram_id>/sources
```

## "Successful If" Checklist

- [x] I can save a full research result as an engram in one API call.
- [x] Each major claim has URL + snippet + timestamp provenance.
- [x] I can retrieve relevant engrams by semantic query + metadata filters.
- [x] I can generate an LLM-ready rehydration bundle from any engram.
- [x] I can resume a checkpointed agent run by `thread_id`.
- [x] I can access a local UI using a simple login workflow to test endpoints.
- [x] I can inspect stored provenance sources for an engram via API/UI.
- [x] A newcomer can run the system locally using this README alone.
- [ ] The same stored engram can be reused with different LLM providers later.
