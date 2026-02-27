# Engram Vault (Local-First MVP)

Long LLM research runs produce valuable context that disappears between sessions. Engram Vault turns that context into durable, queryable "memory engrams" with full provenance so nothing is lost.

## Quick Start

Preferred local run mode (single terminal, starts DB + API + Web)

```bash
cp .env.example .env
make dev                 # starts DB + API + web in one terminal
```

Optional split-terminal mode

```bash
make api
make web
```

Use `make stack-up` as an occasional debugging mode when you specifically need the full containerized stack.

Open [localhost:5173](http://localhost:5173) (React chat) or [localhost:8000/healthz](http://localhost:8000/healthz) (API health).

Default credentials: `admin` / `admin123`

Prerequisites: Docker, Go 1.25+, Node 18+ (`uv` is optional for legacy Python-only flows like `make py-api` and `make eval`). See [docs/env-reference.md](docs/env-reference.md) for all env vars.

## Documentation

| Guide | Covers |
|-------|--------|
| [Plan.md](Plan.md) | Canonical roadmap and phase status |
| [AGENT.md](AGENT.md) | Contribution and maintenance contract |
| [checkpoint.md](checkpoint.md) | Milestone tracking and phase progress |
| [todo.md](todo.md) | Forward-looking work items |
| [docs/api-reference.md](docs/api-reference.md) | REST endpoints, request/response schemas |
| [docs/mcp-guide.md](docs/mcp-guide.md) | MCP transport, auth, tools, and typed clients |
| [docs/testing-guide.md](docs/testing-guide.md) | Test suites, acceptance tests, eval harness |
| [docs/user-workflows.md](docs/user-workflows.md) | Docker stack, local validation, admin runbook, CLI |
| [docs/env-reference.md](docs/env-reference.md) | Environment variables and configuration |
| [docs/implementation-log.md](docs/implementation-log.md) | Detailed per-phase implementation history |
| [docs/architecture-playbook.md](docs/architecture-playbook.md) | Architecture diagrams and multi-model runbooks |
| [docs/mcp-client-integrations.md](docs/mcp-client-integrations.md) | LibreChat, Copilot, and Codex MCP config |
| [docs/go-migration-test-matrix.md](docs/go-migration-test-matrix.md) | Go migration parity gate mapping |
| [docs/go-rollout-playbook.md](docs/go-rollout-playbook.md) | Staged Go traffic rollout + rollback criteria |

## Architecture

```text
+---------------------------+         +-------------------------------+
| Research Agent / User     |         | React Chat Workbench          |
| (chat + ingest workflows) |         | (session + engram management) |
+-------------+-------------+         +---------------+---------------+
              |                                         |
              | REST API + MCP (JSON-RPC/SSE)           |
              v                                         v
       +-------------------------------------------------------+
       | Go Engram Vault API                                   |
       | - Provider adapters (OpenAI, Anthropic, Bedrock)      |
       | - Context assembly (engrams + document chunks)        |
       | - Embedding, retrieval, reranking                     |
       | - Memory lifecycle + admin management                 |
       +---------------------------+---------------------------+
                                   |
                                   | SQL + pgvector
                                   v
                    +-------------------------------------+
                    | Postgres + pgvector (local Docker)  |
                    +-------------------------------------+
```

## Data Flow

1. Capture run output (summary, decisions, claims, assumptions, questions).
2. Attach provenance (source URLs, timestamps, snippets).
3. Persist as `MemoryEngram` with retrieval embedding.
4. Query by semantic intent + metadata filters.
5. Build a compact rehydration bundle for future LLM sessions.

## Current Status

Phases 0-18, 29-34 implemented. Go migration cutover checklist completed through CP181. See [checkpoint.md](checkpoint.md) for details.

**Completed:** foundation, schema, retrieval, durability, chat continuity, providers (OpenAI/Anthropic/Bedrock), MCP stream, React UI, acceptance testing, theme/UX, MCP developer experience, document ingestion (RAG), memory lifecycle, auto-metadata enrichment, MCP tokens, enterprise memory management.

**Current focus:** post-migration operational hardening and forward roadmap phases (19+).

## Repository Layout

```text
engram/
  README.md          # this file
  AGENT.md           # contribution contract
  Plan.md            # roadmap
  checkpoint.md      # milestone tracking
  todo.md            # work items
  Makefile           # dev commands (make help)
  docker-compose.yml # local stack
  .env.example       # env template
  docs/              # guides (see table above)
  agents/            # agent definitions (sentinel)
  skills/            # workflow skills for AI agents
  cmd/api/           # Go API entrypoint
  internal/          # Go backend packages
  db/init/           # schema SQL
  api/               # legacy Python backend artifacts
    app/             # domain modules
    evals/           # eval harness
    tests/           # backend tests
  web/               # React frontend (Vite/TS)
    src/             # components, API clients, styles
  acceptance-tests/  # Playwright-BDD features
```

## Agents & Skills

Agent definitions: `agents/sentinel.md` (full-codebase code health sweep).

Skills (under `skills/`): engram-lifecycle, chat-rag-operator, mcp-http-stream-tools, mcp-token-authz, provider-openai, provider-anthropic, provider-bedrock, schema-migrations-and-backfill, testing-and-evals, release-and-maintenance, domain-module-layout, react-chat-ui-operator, frontend-style-system, dockerized-acceptance-testing, document-ingestion-rag, memory-lifecycle-policies, engram-auto-metadata-enrichment, codescene.

Skills are accessible to Claude Code (`.claude/` symlinks), OpenAI Codex (`AGENTS.md`), and GitHub Copilot (`.github/copilot-instructions.md`).

## Key Commands

```bash
make help          # list all targets
make dev           # DB + API + web (single terminal)
make test          # backend tests
make lint          # go vet
make web-check     # frontend tests
make eval          # legacy Python eval harness
make acceptance-test-mock   # acceptance tests (dockerized)
make stack-up      # full containerized stack
make stack-down    # tear down
make openapi-check # Python OpenAPI export + Go route contract validation
make shadow-compare # Go/Python shadow status-family parity
make benchmark-compare # Go/Python benchmark report (findings/)
```

## Success Checklist

- [x] Save a full research result as an engram in one API call
- [x] Each claim has URL + snippet + timestamp provenance
- [x] Retrieve engrams by semantic query + metadata filters
- [x] Generate LLM-ready rehydration bundles
- [x] Resume checkpointed agent runs by `thread_id`
- [x] Local UI login workflow for testing endpoints
- [x] Inspect stored provenance sources via API/UI
- [x] Multi-user auth with role-based access controls
- [x] Local eval scenarios (fact/cross/temporal/abstention)
- [x] CLI upload/search/rehydrate from terminal
- [x] Reranking and citation-packing for retrieval quality
- [x] Background consolidation maintenance jobs
- [x] Login rate limits and audit event logging
- [x] MCP JSON-RPC/SSE with chat/engram/user tools
- [x] Multi-provider chat (OpenAI, Anthropic, Bedrock)
- [x] Document ingestion with session-level pinning
- [x] Memory lifecycle policies (autosave, retention, timeline)
- [x] Auto-metadata enrichment across all engram create paths
- [x] MCP personal access tokens with scoped authorization
- [x] Enterprise memory management (projects, collections, admin)
- [ ] Cross-provider engram reuse validation
