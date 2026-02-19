# Engram Vault - Copilot Instructions

## Project Identity

Engram Vault is a local-first memory system that turns long LLM research runs
into durable, queryable "memory engrams." Stack: FastAPI + PostgreSQL/pgvector
backend, React frontend, MCP server for agent integration.

## Operational Contract

Read `AGENT.md` at the project root before making architectural, API, or schema
changes. It is the canonical contribution contract for all contributors and agents.

## Agents

| Agent | File | Description |
|-------|------|-------------|
| sentinel | `agents/sentinel.md` | Full-codebase code health sweep using CodeScene MCP tools. |

## Skills Catalog

Skills are reusable agent workflow definitions in `skills/<name>/SKILL.md`:

| Skill | Description |
|-------|-------------|
| `engram-lifecycle` | Engram creation, query, rehydration, ownership/visibility. |
| `chat-rag-operator` | Chat session continuity and context assembly with engrams. |
| `mcp-http-stream-tools` | JSON-RPC over SSE tool contracts and handlers. |
| `mcp-token-authz` | MCP PAT lifecycle, scope/allowlist/project authorization. |
| `provider-openai` | OpenAI adapter implementation conventions. |
| `provider-anthropic` | Anthropic adapter implementation conventions. |
| `provider-bedrock` | Bedrock adapter implementation conventions. |
| `schema-migrations-and-backfill` | Forward-only schema evolution and compatibility. |
| `testing-and-evals` | Test matrix and eval harness extension guidance. |
| `release-and-maintenance` | Release checklist and long-run maintenance cadence. |
| `domain-module-layout` | Module/package conventions for long-run maintainability. |
| `react-chat-ui-operator` | Frontend workflow conventions for chat/session/engram UX. |
| `frontend-style-system` | Token-driven styled-components + Tailwind workflow rules. |
| `dockerized-acceptance-testing` | Playwright-BDD dockerized quality-gate workflow. |
| `document-ingestion-rag` | Deterministic document chunking, ingestion, blended retrieval. |
| `memory-lifecycle-policies` | Autosave, retention, consolidation, timeline events. |
| `engram-auto-metadata-enrichment` | Fill-empty metadata derivation rules. |
| `codescene` | Code health analysis and improvement using CodeScene MCP tools. |

## Key Directories

- `api/app/` - FastAPI backend (routes, services, repositories, providers, MCP)
- `web/` - React frontend (chat UI, session management, styled-components + Tailwind)
- `db/init/` - SQL schema and migrations
- `acceptance-tests/` - Playwright-BDD end-to-end tests
- `docs/` - Architecture playbook, user flow, MCP integration guides
- `skills/` - Reusable agent workflow definitions
- `agents/` - Agent definitions
