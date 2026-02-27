# Engram Backend Migration: Python/FastAPI → Go

## Context

Engram is shipping to enterprise customers who expect high performance. The current Python/FastAPI backend (~19K LOC, 80+ endpoints, 40+ MCP tools) works well functionally but carries Python's runtime overhead — higher memory consumption, slower cold starts, GIL-limited concurrency. Enterprise deployments span both local-first (memory/startup matter) and cloud (throughput matters), requiring a runtime that excels at both.

The team has **Python + Go experience** and a **3-4 month window**. This rules out Rust (learning curve too steep for timeline) and Java/Spring (JVM memory overhead hurts local-first; no existing team experience). Go is the clear fit: the team already knows it, it produces tiny static binaries with instant startup, and the ecosystem covers every requirement.

## Decision: Go with Chi router

### Why Go over alternatives

| Factor | Go | Rust | Java/Spring |
|--------|-----|------|-------------|
| Team experience | **Have it** | No | No |
| Memory (idle) | **~10-30 MB** | ~5-20 MB | 180-500 MB (JVM) |
| Startup | **~10-50ms** | ~10-50ms | 3-4s (JVM) / 100ms (GraalVM) |
| Container image | **~20-50 MB** | ~15-40 MB | 217-361 MB |
| Build time | **2-5s** | 30-120s | 60-120s |
| MCP SDK | **mcp-go (1,307 importers) + official go-sdk** | Official rust-sdk (newer) | Spring AI MCP (newest) |
| LLM SDKs | **Official OpenAI + Anthropic + AWS** | Community only for Anthropic | Spring AI abstraction |
| Time to productivity | **Immediate** | 2-4 months | 4-6 weeks |
| 3-4 month feasibility | **Yes** | No | Tight |

### Why not Rust?

- **Learning curve**: Ownership, lifetimes, borrow checker — 2-4 months for Python developers to become productive
- **Build times**: 30-120s incremental vs Go's 2-5s — kills iteration speed during a migration
- **Anthropic SDK**: No official Rust SDK; community-maintained only
- **Timeline**: 3-4 months is infeasible for a team learning Rust while migrating ~19K LOC
- **Raw performance delta vs Go is minimal**: The bottleneck is I/O (database, LLM API calls), not CPU. Go's goroutine model handles this efficiently

### Why not Java/Spring AI?

- **JVM memory**: 180-500 MB idle baseline — unacceptable for local-first deployments where users run this on their machines
- **Startup**: 3-4 seconds on JVM (GraalVM native brings it to ~100ms but adds build complexity: 5-15 min compile, 4GB RAM required)
- **Container images**: 217-361 MB vs Go's 20-50 MB
- **Over-engineering risk**: Current codebase uses raw SQL + thin service layer — Spring's annotation-driven DI adds indirection the architecture doesn't need
- **No team experience**: Python → Java transition takes 4-6 weeks to productivity

**Where Java/Spring AI shines (but doesn't apply here)**:
- Spring AI's built-in RAG abstractions (VectorStore, document chunking) are excellent
- Spring Security is more complete than Go's manual auth
- MCP SDK is officially maintained by Anthropic + Spring team
- Largest enterprise hiring pool
- Best choice if team already knows Java or deploys server-side only

### Why port MCP to Go (not keep in Python)

The engram MCP server is **custom JSON-RPC 2.0** — it doesn't use the Python MCP SDK. The 40+ tools directly call the same services (ChatService, repository, etc.) as the REST API. Keeping MCP in Python means:
- Two runtimes to deploy and monitor
- Duplicated database connection management
- Cross-process communication overhead for shared business logic
- Double the operational surface area

Go's `mcp-go` library (1,307 importers, protocol version 2025-11-25) provides `StreamableHTTPServer` with tool registration, making the port straightforward.

---

## Ecosystem: Key Go Packages

| Component | Package | Maturity |
|-----------|---------|----------|
| Router | `go-chi/chi/v5` | Stable since 2017, minimal, idiomatic |
| PostgreSQL | `jackc/pgx/v5` + `pgxpool` | De facto standard, async-capable |
| pgvector | `pgvector/pgvector-go` | Native pgx integration |
| MCP server | `mark3labs/mcp-go` or official `go-sdk` | 1,307+ importers |
| OpenAI | `openai/openai-go` (official) | Official SDK |
| Anthropic | `anthropics/anthropic-sdk-go` (official) | Official SDK |
| AWS Bedrock | `aws/aws-sdk-go-v2` | Official AWS SDK |
| Config | `kelseyhightower/envconfig` | Battle-tested |
| Validation | `go-playground/validator/v10` | Standard |
| Logging | `log/slog` (stdlib, Go 1.21+) | Built-in |
| Testing | `testing` + `stretchr/testify` | Built-in |
| Password hash | `golang.org/x/crypto/pbkdf2` | Official |
| HMAC/SHA256 | `crypto/hmac` + `crypto/sha256` | Stdlib |
| JWT | `golang-jwt/jwt/v5` | Standard |
| Sessions | `alexedwards/scs` | Mature |
| HTTP client | `net/http` (stdlib) | Built-in |

---

## Project Structure

```
cmd/
  api/main.go                    # Entry point
internal/
  config/config.go               # envconfig settings + production validation
  db/db.go                       # pgxpool connection + health check
  models/                        # Go structs with JSON tags (split by domain)
    engram.go
    chat.go
    document.go
    user.go
    mcp.go
    project.go
    oauth.go
  auth/
    password.go                  # PBKDF2 hashing
    csrf.go                      # CSRF token generation
    session.go                   # Session middleware
    ratelimit.go                 # Login + MCP rate limiting
  repository/
    engram.go                    # Engram CRUD + vector search + reranking
    user.go                      # User CRUD
    chat.go                      # Session + message persistence
    chat_pinning.go              # Pin/unpin engrams/documents
    document.go                  # Document + chunk persistence
    project.go                   # Project CRUD
    collection.go                # Engram collections
    oauth.go                     # OAuth clients + auth codes
    mcp_token.go                 # MCP token CRUD
  embeddings/
    service.go                   # Provider orchestration + fallback
    local.go                     # SHA256 deterministic embeddings
    openai.go                    # OpenAI API embeddings
  providers/
    provider.go                  # ChatProvider interface
    openai.go                    # OpenAI chat completion
    anthropic.go                 # Claude API
    bedrock.go                   # AWS Bedrock
    registry.go                  # Provider lookup
  chat/
    service.go                   # Message generation + streaming
    context.go                   # Context assembly (pinned engrams/docs)
    lifecycle.go                 # Autosave + retention policies
    streaming.go                 # SSE response helpers
  mcp/
    server.go                    # MCP server setup (mcp-go)
    transport.go                 # SSE/JSON transport
    catalog.go                   # 40+ tool definitions
    auth.go                      # Token resolution
    authorization.go             # Per-tool scope enforcement
    dispatch_engram.go           # Engram tool handlers
    dispatch_chat.go             # Chat tool handlers
    dispatch_project.go          # Project tool handlers
    dispatch_document.go         # Document tool handlers
    dispatch_admin.go            # Admin tool handlers
    dispatch_transfer.go         # Export/import tool handlers
  ingestion/
    service.go                   # Document processing
    chunking.go                  # Text chunking with overlap
  projects/
    service.go                   # Project CRUD + defaults
  admin/
    service.go                   # Soft delete/restore + collections
  export/
    service.go                   # JSON/ZIP export + import
  workflow/
    agent.go                     # Simple state machine (replaces LangGraph)
  oauth/
    service.go                   # OAuth 2.0 PKCE flow
    registration.go              # Client registration
  audit/
    audit.go                     # JSONL event logging
  api/
    router.go                    # Chi router setup + middleware
    engram.go                    # Engram endpoints
    chat.go                      # Chat endpoints
    ingestion.go                 # Document endpoints
    projects.go                  # Project endpoints
    admin.go                     # Admin endpoints
    oauth.go                     # OAuth endpoints
    mcp.go                       # MCP streaming endpoint
    user.go                      # User management endpoints
    health.go                    # /healthz + /api/v1/version
    ui.go                        # Login/admin HTML pages
  templates/                     # Jinja2 → Go html/template
Dockerfile
go.mod
go.sum
```

---

## Phased Migration Plan (14-16 weeks)

### Phase 0: Scaffolding & Side-by-Side Infrastructure (Week 1)

**Goal**: Go project compiles, connects to the same PostgreSQL, runs alongside Python.

- [x] `go mod init`, install all dependencies
- [x] Port `config.py` → `internal/config/config.go` (envconfig + validation)
- [x] Port `db.py` → `internal/db/db.go` (pgxpool + schema version check)
- [x] Docker Compose Go API cutover completed (initial side-by-side migration path superseded by final Go-only `api` service)
- [x] Reverse proxy (nginx/traefik) for path-based routing during migration
- [x] Generate OpenAPI spec from Python FastAPI (`/api/v1/openapi.json`) — this is the compatibility contract
- [x] CI: Go backend checks pipeline (`gofmt` + `go vet` + `go test`) via GitHub Actions

**Key files to reference**:
- `api/app/config.py` (208 LOC)
- `api/app/db.py` (74 LOC)
- `docker-compose.yml`

### Phase 1: Models + Auth + Embeddings (Weeks 2-3)

**Goal**: Core types, authentication, and embedding generation working.

- [x] Port `models.py` (611 LOC) → `internal/models/` — Go structs with `json` tags and `validate` tags
- [x] Port `auth.py` (50 LOC) → `internal/auth/password.go` — PBKDF2 with 390K iterations
- [x] Port `login_guard.py` → `internal/auth/ratelimit.go` — sliding window rate limiter
- [x] Port `audit.py` → `internal/audit/audit.go` — JSONL file writer
- [x] Port `embeddings/` → `internal/embeddings/` — local SHA256 provider + OpenAI provider + service with fallback
- [x] Port `user_repository.py` → `internal/repository/user.go`
- [x] Tests: config loading, password hashing round-trip, embedding dimension validation, rate limit behavior

**Key files to reference**:
- `api/app/models.py` (611 LOC)
- `api/app/auth.py` (50 LOC)
- `api/app/embeddings/service.py`, `local_provider.py`, `openai_provider.py`
- `api/app/login_guard.py`

### Phase 2: Repository Layer (Weeks 4-5)

**Goal**: All database operations working — the foundation for everything above.

- [x] Port `repository.py` (743 LOC) → `internal/repository/engram.go` — **most critical file**
  - `create_engram()` with vector embedding INSERT
  - `list_engrams()` with pagination + visibility filtering
  - `query_engrams()` with `embed <=> $1::vector` + lexical reranking
  - `get_rehydration_bundle()` with source assembly
  - `_rerank_by_combined_score()` — CPU-intensive, benefits most from Go
- [x] Port `chat_repository.py` + `chat_repository_pinning.py` → `internal/repository/chat.go`
- [x] Port `ingestion/repository.py` → `internal/repository/document.go`
- [x] Port `projects/repository.py` → `internal/repository/project.go`
- [x] Port `memory_admin/repository_*.py` → `internal/repository/collection.go`
- [x] Port `mcp_tokens/repository.py` → `internal/repository/mcp_token.go`
- [x] Port `oauth/repository.py` → `internal/repository/oauth.go`
- [x] Integration tests against shared PostgreSQL (same data as Python)

**Key files to reference**:
- `api/app/repository.py` (743 LOC) — vector search + reranking logic
- `api/app/chat_repository.py`
- `api/app/ingestion/repository.py`
- `db/init/001_schema.sql` (636 LOC) — no changes needed

### Phase 3: Services + LLM Providers (Weeks 6-8)

**Goal**: Business logic layer complete — chat, ingestion, providers, admin.

- [x] Port `providers/base.py` → `internal/providers/provider.go` — Go interface:
  ```go
  type ChatProvider interface {
      Generate(ctx context.Context, req ChatRequest) (ChatResponse, error)
      StreamGenerate(ctx context.Context, req ChatRequest) (<-chan ChatChunk, error)
  }
  ```
- [x] Port `providers/openai_provider.py` (105 LOC) → `internal/providers/openai.go` (official openai-go SDK)
- [x] Port `providers/anthropic_provider.py` (104 LOC) → `internal/providers/anthropic.go` (official anthropic-sdk-go)
- [x] Port `providers/bedrock_provider.py` (197 LOC) → `internal/providers/bedrock.go` (aws-sdk-go-v2)
- [x] Port `chat/service.py` → `internal/chat/service.go` — message prep, provider calls, streaming
- [x] Port `chat/context.py` → `internal/chat/context.go` — pinned engram/document assembly
- [x] Port `chat/session_lifecycle.py` → `internal/chat/lifecycle.go` — autosave + retention
- [x] Port `ingestion/service.py` + `chunking.py` → `internal/ingestion/` — document processing
- [x] Port `projects/service.py` → `internal/projects/service.go`
- [x] Port `memory_admin/service.py` → `internal/admin/service.go`
- [x] Port `export/service.py` → `internal/export/service.go` — JSON/ZIP
- [x] Port `oauth/service.py` + `registration.py` → `internal/oauth/`
- [x] Port `agent_workflow.py` (319 LOC) → `internal/workflow/agent.go` — replace LangGraph with simple state machine; store checkpoints in PostgreSQL instead of SQLite

**Key files to reference**:
- `api/app/chat/service.py` — most complex service (streaming, context, lifecycle)
- `api/app/providers/*.py` — thin adapters, straightforward port
- `api/app/ingestion/chunking.py` — deterministic chunking algorithm
- `api/app/agent_workflow.py` — simple linear pipeline (collect→synthesize→persist)

### Phase 4: REST API + MCP Server (Weeks 9-12)

**Goal**: Full API surface ported, MCP server operational.

**REST API** (Weeks 9-10):
- [x] Port `main.py` (686 LOC) → `cmd/api/main.go` + `internal/api/router.go` — Chi router, middleware chain
- [x] Port all route handlers from `main_api_router.py`, `chat/api.py`, `ingestion/api.py`, `projects/api.py`, `memory_admin/api.py`
- [x] Port session middleware + CSRF protection
- [x] Port OAuth routes from `oauth/router.py`
- [x] Port Jinja2 templates → Go `html/template`
- [x] Validate against OpenAPI spec from Phase 0

**MCP Server** (Weeks 11-12):
- [x] Set up mcp-go `StreamableHTTPServer` in `internal/mcp/server.go`
- [x] Port `mcp/catalog.py` (750 LOC) → `internal/mcp/catalog.go` — register 40+ tools using mcp-go tool registration API
- [x] Port `mcp/service.py` (752 LOC) → `internal/mcp/` dispatch modules — route tool calls to service layer
- [x] Port `mcp/auth.py` + `token_authorization.py` → `internal/mcp/auth.go` — token resolution + scope enforcement
- [x] Port `mcp/streaming.py` → `internal/mcp/streaming.go` — SSE for chat streaming tools
- [x] Port `mcp_tokens/service.py` → included in service layer
- [x] MCP integration tests: tool discovery, read/write tool execution, token auth, rate limiting

**Key files to reference**:
- `api/app/mcp/catalog.py` (750 LOC) — 40+ tool definitions
- `api/app/mcp/service.py` (752 LOC) — JSON-RPC dispatch
- `api/app/mcp/api.py` (158 LOC) — SSE transport

### Phase 5: Testing + Cutover (Weeks 13-14)

**Goal**: Feature parity validated, traffic switched to Go.

- [x] Port remaining behavior coverage from legacy pytest surface into Go table-driven tests plus acceptance/contract parity gates
- [x] Run acceptance tests (`acceptance-tests/`) against Go API
- [x] Shadow traffic: run both Python and Go behind load balancer, compare responses
- [x] Performance benchmarks: measure requests/sec, P99 latency, memory under load
- [x] Gradual traffic shift runbook: 10% → 25% → 50% → 100% to Go
- [x] Update Dockerfile to Go binary container build
- [x] Update Docker Compose to remove Python `api` service
- [x] Update CI/CD to Go build + test pipeline
- [x] Archive Python `api/` code (keep for reference, don't delete)

---

## Risk Mitigation

| Risk | Mitigation |
|------|------------|
| API contract drift | OpenAPI spec generated from Python; Go validated against it in CI |
| Database compatibility | Zero risk — same PostgreSQL, same schema, same raw SQL |
| MCP client breakage | Run existing MCP integration tests against Go implementation |
| Timeline slip | Phases are independent; can ship Go REST API while MCP stays Python temporarily |
| LangGraph replacement | The workflow is a simple linear pipeline — state machine replacement is <200 LOC |
| Team unfamiliarity with mcp-go | Fallback: hand-roll JSON-RPC 2.0 (same as current Python approach) |

## Fallback: Interim Split Architecture

If MCP porting (Phase 4, weeks 11-12) creates timeline pressure, the MCP server can temporarily remain in Python behind the reverse proxy while the REST API runs in Go. Both share PostgreSQL. This buys 4-6 weeks but adds operational complexity — treat as temporary.

---

## Verification Plan

1. **Unit tests**: `go test ./internal/...` — covers config, auth, embeddings, repository, services
2. **Integration tests**: `go test -tags=integration ./...` — requires PostgreSQL with pgvector
3. **API contract**: Automated OpenAPI diff between Python spec and Go responses
4. **MCP smoke test**: Connect Claude Desktop / VS Code to Go MCP endpoint, verify all 40+ tools
5. **Acceptance tests**: Run existing Playwright-BDD suite against Go API
6. **Performance**: `wrk` / `hey` benchmarks comparing Python vs Go on key endpoints (engram query, chat stream, MCP tool call)
7. **Memory profile**: Compare container memory usage under sustained load

---

## Expected Outcomes

| Metric | Python (current) | Go (target) |
|--------|-------------------|-------------|
| Memory (idle) | ~100-150 MB | ~10-30 MB |
| Memory (loaded) | ~300-600 MB | ~50-150 MB |
| Container image | ~400-800 MB | ~20-50 MB |
| Startup | ~1-3 seconds | ~10-50ms |
| Requests/sec (simple) | ~3,000-8,000 | ~50,000-100,000 |
| P99 latency | ~5-20ms | ~0.5-2ms |
| Binary | Python runtime + deps | Single static binary |
