# Engram Vault - API Reference

> Complete REST API endpoint reference.
> See [MCP Guide](mcp-guide.md) for MCP tool endpoints. See [Environment Variables](env-reference.md) for configuration.

---

## Authentication

All API endpoints (except `/healthz`, `/api/v1/metrics`, `/login`, `/login/oidc*`, and `/.well-known/*`) require an authenticated session.
Sign in via `POST /api/v1/session/login` (JSON) or `/login` (UI form) to obtain a session cookie.

---

## Endpoints by Domain

### User Management

| Method | Path | Description |
|--------|------|-------------|
| `GET` | `/api/v1/me` | Current user profile |
| `GET` | `/api/v1/users` | List users (admin) |
| `POST` | `/api/v1/users` | Create user (admin) |
| `PATCH` | `/api/v1/users/{user_id}` | Update user (admin) |

### Session Auth

| Method | Path | Description |
|--------|------|-------------|
| `GET` | `/api/v1/session/csrf` | Issue/refresh CSRF token and session cookie |
| `POST` | `/api/v1/session/login` | Session login (JSON username/password) |
| `POST` | `/api/v1/session/logout` | Session logout (JSON + CSRF) |
| `GET` | `/api/v1/session/oidc/start` | Start OIDC auth flow (redirect) |
| `GET` | `/api/v1/session/oidc/callback` | Complete OIDC auth flow (redirect/login mapping) |

### Observability

| Method | Path | Description |
|--------|------|-------------|
| `GET` | `/api/v1/metrics` | Process-local observability snapshot (request, provider, stream, and lifecycle counters) |

### Engrams

| Method | Path | Description |
|--------|------|-------------|
| `POST` | `/api/v1/engrams` | Create engram |
| `GET` | `/api/v1/engrams` | List engrams |
| `POST` | `/api/v1/engrams/query` | Semantic query |
| `GET` | `/api/v1/engrams/{engram_id}/sources` | Inspect provenance sources |
| `GET` | `/api/v1/engrams/{engram_id}/rehydrate` | Get rehydration bundle |
| `POST` | `/api/v1/engrams/{engram_id}/share` | Share engram to project-visible scope |
| `POST` | `/api/v1/engrams/{engram_id}/unshare` | Revert engram visibility to private |

### Chat Sessions

| Method | Path | Description |
|--------|------|-------------|
| `POST` | `/api/v1/chat/sessions` | Create session |
| `GET` | `/api/v1/chat/sessions` | List sessions |
| `GET` | `/api/v1/chat/sessions/{session_id}` | Get session |
| `PATCH` | `/api/v1/chat/sessions/{session_id}` | Update session |
| `GET` | `/api/v1/chat/sessions/{session_id}/messages` | List messages |
| `POST` | `/api/v1/chat/sessions/{session_id}/messages` | Send message |
| `POST` | `/api/v1/chat/sessions/{session_id}/messages/stream` | Send message (streaming) |

### Chat Lifecycle

| Method | Path | Description |
|--------|------|-------------|
| `GET` | `/api/v1/chat/sessions/{session_id}/lifecycle-policy` | Get lifecycle policy |
| `PATCH` | `/api/v1/chat/sessions/{session_id}/lifecycle-policy` | Update lifecycle policy |
| `GET` | `/api/v1/chat/sessions/{session_id}/timeline` | Get timeline events |

### Chat Pinning

| Method | Path | Description |
|--------|------|-------------|
| `GET` | `/api/v1/chat/sessions/{session_id}/engrams` | List pinned engrams |
| `POST` | `/api/v1/chat/sessions/{session_id}/engrams/pin` | Pin engram |
| `DELETE` | `/api/v1/chat/sessions/{session_id}/engrams/{engram_id}` | Unpin engram |
| `GET` | `/api/v1/chat/sessions/{session_id}/documents` | List pinned documents |
| `POST` | `/api/v1/chat/sessions/{session_id}/documents/pin` | Pin document |
| `DELETE` | `/api/v1/chat/sessions/{session_id}/documents/{document_id}` | Unpin document |

### Chat Continuity

| Method | Path | Description |
|--------|------|-------------|
| `POST` | `/api/v1/chat/sessions/{session_id}/save-engram` | Save session as engram |
| `POST` | `/api/v1/chat/sessions/{session_id}/continue` | Continue in new session |

### Document Ingestion

| Method | Path | Description |
|--------|------|-------------|
| `POST` | `/api/v1/ingestion/text` | Ingest text |
| `POST` | `/api/v1/ingestion/file` | Upload file |
| `GET` | `/api/v1/ingestion/documents` | List documents |
| `POST` | `/api/v1/ingestion/query` | Query documents |
| `POST` | `/api/v1/ingestion/query/blended` | Blended query (engrams + documents) |

### Projects

| Method | Path | Description |
|--------|------|-------------|
| `GET` | `/api/v1/projects` | List projects |
| `POST` | `/api/v1/projects` | Create project |
| `GET` | `/api/v1/projects/default` | Get default project |
| `PATCH` | `/api/v1/projects/default` | Set default project |
| `GET` | `/api/v1/projects/{project_id}/members` | List project members (owner/admin) |
| `POST` | `/api/v1/projects/{project_id}/members` | Add/restore project member (owner/admin) |
| `PATCH` | `/api/v1/projects/{project_id}/members/{user_id}` | Update project member role (owner/admin) |
| `DELETE` | `/api/v1/projects/{project_id}/members/{user_id}` | Remove project member (owner/admin) |
| `GET` | `/api/v1/projects/{project_id}/audit-events` | List project audit events (owner/admin) |
| `GET` | `/api/v1/projects/{project_id}/export` | Export project bundle (JSON/ZIP, optional collection filter) |
| `POST` | `/api/v1/projects/{project_id}/import` | Import project bundle with conflict policy (`skip`/`overwrite`/`rename`) |

### Admin Memory Management

| Method | Path | Description |
|--------|------|-------------|
| `GET` | `/api/v1/admin/memory/sessions` | List sessions (admin) |
| `DELETE` | `/api/v1/admin/memory/sessions/{session_id}` | Delete session (soft) |
| `POST` | `/api/v1/admin/memory/sessions/{session_id}/restore` | Restore session |
| `GET` | `/api/v1/admin/memory/engrams` | List engrams (admin) |
| `GET` | `/api/v1/admin/memory/engrams/{engram_id}` | Get engram (admin) |
| `PATCH` | `/api/v1/admin/memory/engrams/{engram_id}` | Update engram |
| `POST` | `/api/v1/admin/memory/engrams/{engram_id}/move` | Move engram to project |
| `DELETE` | `/api/v1/admin/memory/engrams/{engram_id}` | Delete engram (soft) |
| `POST` | `/api/v1/admin/memory/engrams/{engram_id}/restore` | Restore engram |
| `GET` | `/api/v1/admin/memory/collections` | List collections |
| `POST` | `/api/v1/admin/memory/collections` | Create collection |
| `PATCH` | `/api/v1/admin/memory/collections/{collection_id}` | Update collection |
| `DELETE` | `/api/v1/admin/memory/collections/{collection_id}` | Delete collection |
| `POST` | `/api/v1/admin/memory/collections/{collection_id}/items` | Add items to collection |
| `DELETE` | `/api/v1/admin/memory/collections/{collection_id}/items/{engram_id}` | Remove item from collection |

### MCP Tokens

| Method | Path | Description |
|--------|------|-------------|
| `POST` | `/api/v1/mcp/tokens` | Create MCP token (admin) |
| `GET` | `/api/v1/mcp/tokens` | List MCP tokens (admin) |
| `POST` | `/api/v1/mcp/tokens/{token_id}/revoke` | Revoke MCP token |

### MCP Stream

| Method | Path | Description |
|--------|------|-------------|
| `POST` | `/api/v1/mcp/stream` | MCP JSON-RPC over SSE (rate-limited, returns `429` with `Retry-After`) |

### OAuth

| Method | Path | Description |
|--------|------|-------------|
| `GET` | `/.well-known/oauth-authorization-server` | OAuth server metadata |
| `GET` | `/.well-known/openid-configuration` | OpenID configuration |
| `GET` | `/.well-known/oauth-protected-resource` | Protected resource metadata |
| `POST` | `/oauth/register` | Protected dynamic client registration (admin session required) |
| `GET` | `/oauth/authorize` | Authorization endpoint |
| `POST` | `/oauth/token` | Token exchange (PKCE `S256` only) |

### Agent Runs

| Method | Path | Description |
|--------|------|-------------|
| `POST` | `/api/v1/agent-runs` | Create agent run |
| `GET` | `/api/v1/agent-runs/{thread_id}` | Get agent run state |
| `POST` | `/api/v1/agent-runs/{thread_id}/resume` | Resume agent run |

### Health

| Method | Path | Description |
|--------|------|-------------|
| `GET` | `/healthz` | Health check |
| `GET` | `/api/v1/version` | Version info |

---

## Examples

### Observability Snapshot

`GET /api/v1/metrics` returns:

- `totals`, `by_status_class`, `by_domain`, `by_route` for request-level telemetry.
- `provider_failures` keyed by `operation provider error_code`.
- `stream_health` keyed by `operation provider outcome` with count, chunks, and duration aggregates.
- `lifecycle_traces` keyed by `operation stage` (or `operation stage error_code` for failure stages).

### Create Engram

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

### Query Engrams

```bash
curl -X POST http://localhost:8000/api/v1/engrams/query \
  -H "Content-Type: application/json" \
  -d '{
    "query": "Which framework was selected for durable long runs?",
    "project_id": "engram-vault",
    "top_k": 5
  }'
```

### Agent Run (With Snapshots)

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

### Inspect Engram Sources

```bash
curl http://localhost:8000/api/v1/engrams/<engram_id>/sources
```

---

## Data Contracts

### MemoryEngram Contract (MVP)

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

### Auto Metadata Enrichment (Phase 29)

- Scope: applies to all engram create paths through centralized repository logic (`internal/repository/engram_write.go`).
- Behavior: fill-empty-only for `abstract`, `tags`, and `keywords`; non-empty caller values are never overwritten.
- Strategy: deterministic local parsing in v1 (no provider call required).
- Traceability: enrichment details are stored in `engram_json.auto_metadata` with schema version and origin.
- MCP-first path: `engram.create_from_conversation` accepts transcript markdown and returns enrichment report fields.
- API compatibility: runtime endpoints remain unchanged (`POST /api/v1/engrams` keeps backward-compatible payload support).

---

## Local Embedding Strategy (Temporary)

- Pros: zero external dependencies, reproducible tests, fast setup
- Cons: weak semantic quality vs real embedding models

Planned upgrade: swap to a local embedding model (e.g. sentence-transformers) or hosted provider while preserving schema.

## Security Baseline (MVP)

- No external network dependency for memory pipeline
- DB credentials from environment variables
- Keep sensitive source text in local database only
- Add encryption, secrets manager, and RBAC before production
