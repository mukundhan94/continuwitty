# Engram Vault - Full Codebase Refactoring Plan

## Context

The engram codebase has accumulated significant technical debt, particularly in the MCP service layer (`mcp/service.py` scoring 3.0/10 — "Red/severe"). CodeScene analysis reveals 15 files below the 9.5 threshold, with code smells including brain-class monoliths, duplicated patterns, excess function arguments, bumpy roads, and low cohesion. This refactoring will systematically improve every file below 9.5 using the sentinel agent with CodeScene MCP tools, adding tests at every step and committing atomically.

**Business case for the worst file (`mcp/service.py`):** Improving from 3.0 to 5.15 (industry average) predicts 24–48% defect reduction and 3–20% development speed improvement (90% confidence interval).

## Execution Checkpoints

### 2026-02-19 (Checkpoint 1 - MCP Service Phase 1)

- [x] Step 1.1 — Extract tool catalog into `api/app/mcp/catalog.py`
- [x] Step 1.2 — Split `_dispatch_tool` into domain dispatchers
- [x] Step 1.3 — Extract authorization helpers for session/engram/collection ownership checks
- [x] Step 1.4 — Reduce `_project_id_for_tool` bumps with scoped-tool constants and resolver helpers
- [x] Step 1.5 — Reduce `stream_call` duplication by extracting `_authorize_tool_call`
- [x] New test added: `api/tests/test_mcp_tool_catalog.py`
- [x] New test added: `api/tests/test_mcp_service_unit.py`
- [x] Integration expanded: `api/tests/test_mcp_api_integration.py` (`tools/call` round-trip checks)
- [x] CodeScene health checks run (`code_health_review`, `pre_commit_code_health_safeguard`)

---

### 2026-02-19 (Checkpoint 2 - Memory Admin Repository Phase 2)

- [x] Step 2.1 — Decompose `update_admin_engram`
- [x] Added `_build_engram_update_fields(current, payload)` extraction
- [x] Added `_compute_engram_embedding(retrieval_text, dim)` extraction
- [x] Added `_replace_engram_sources(cur, engram_id, sources)` extraction
- [x] Step 2.2 — Eliminate soft delete/restore duplication
- [x] Added `_soft_delete_record(...)` and `_restore_record(...)`
- [x] Refactored session/engram/collection delete+restore wrappers to thin delegates
- [x] New test added: `api/tests/test_memory_admin_repository.py`
- [x] Validation run: `api/tests/test_memory_admin_repository.py`, `api/tests/test_memory_admin_api_integration.py`, `api/tests/test_mcp_service_unit.py` (14 passed)
- [x] Acceptance run: `make acceptance-test-mock-docker` (10 passed)
- [x] CodeScene health checks run (`code_health_review`, `pre_commit_code_health_safeguard`)
- [x] CodeScene score improvement: `api/app/memory_admin/repository.py` from **6.29** to **7.42**

---

## CodeScene Health Scorecard (Current State)

| File | Score | Severity |
|------|-------|----------|
| `api/app/mcp/service.py` | **3.0** | RED |
| `api/app/memory_admin/repository.py` | 7.42 | YELLOW |
| `api/app/chat_repository.py` | 6.69 | YELLOW |
| `api/app/chat/service.py` | 6.77 | YELLOW |
| `api/app/oauth/api.py` | 7.48 | YELLOW |
| `api/app/main.py` | 7.66 | YELLOW |
| `api/app/repository.py` | 7.88 | YELLOW |
| `api/app/providers/bedrock_provider.py` | 8.17 | YELLOW |
| `web/src/App.tsx` | 8.28 | YELLOW |
| `api/app/chat/context.py` | 8.38 | YELLOW |
| `api/app/ingestion/repository.py` | 8.64 | YELLOW |
| `api/app/engram_enrichment/service.py` | 8.93 | YELLOW |
| `api/app/memory_admin/service.py` | 9.09 | GREEN |
| `api/app/ingestion/service.py` | 9.33 | GREEN |
| `api/app/mcp/auth.py` | 9.66 | GREEN |

---

## Execution Model

The sentinel agent processes each phase by:
1. Scoring the target file with `code_health_review`
2. Applying refactoring (manual extractions + `code_health_auto_refactor` where supported)
3. Running `pre_commit_code_health_safeguard` to verify improvement
4. Running `make test` (backend) or `make web-check` (frontend)
5. Committing atomically with a descriptive message
6. Moving to the next step

---

## Naming & Commenting Standards (Applied Throughout)

**Python:** `verb_noun` snake_case functions, `_private_helper` prefix, `is_`/`has_` booleans, `SCREAMING_SNAKE_CASE` constants. Every public function gets a one-line docstring. Complex SQL gets inline comments.

**TypeScript:** `handle<Action>` event handlers, `load<Resource>` data loaders, `use<Domain><Feature>` custom hooks. Document non-obvious async side effects.

---

## Phase 1: `api/app/mcp/service.py` (3.0 -> 5.5+) — CRITICAL

Key issues: 2152 lines, `_dispatch_tool` CC=79 (573 LoC), `_tool_catalog` 555 LoC, low cohesion, 11 bumps

### Step 1.1 — Extract Tool Catalog to `api/app/mcp/catalog.py` (new file)
- Move `_tool_catalog()` (555 lines) as `build_tool_catalog()` module-level function
- Move constants: `_READ_TOOL_NAMES`, `_WRITE_TOOL_NAMES`, `_TOOL_ALIASES`, `_OPTIONAL_PROJECT_TOOLS`, `_PROJECT_FALLBACK_TOOLS`, `_TOOL_NAMESPACE_PREFIXES`
- Move helpers: `_to_public_tool_name()`, `_to_dotted_tool_name()`
- **New test:** `api/tests/test_mcp_tool_catalog.py` — verify all READ/WRITE tool names have catalog entries, all entries have `inputSchema`

### Step 1.2 — Split `_dispatch_tool` into Domain Dispatchers
- Extract `_dispatch_chat_tool()` — all `chat.*` branches
- Extract `_dispatch_engram_tool()` — all `engram.*` branches
- Extract `_dispatch_project_tool()` — all `project.*` branches
- Extract `_dispatch_user_tool()` — all `user.*` branches
- `_dispatch_tool` becomes a thin router (~30 lines)

### Step 1.3 — Extract Authorization Helpers
- `_require_collection_access(actor, collection_id)` — eliminates 4-copy duplication
- `_require_session_access(actor, session_id)` — eliminates 2-copy duplication
- `_require_engram_access(actor, engram_id)` — eliminates 5+ copy duplication

### Step 1.4 — Reduce `_project_id_for_tool` Bumps
- Group tool names into named constants in `catalog.py` (`_SESSION_SCOPED_TOOLS`, `_ENGRAM_SCOPED_TOOLS`)
- Extract `_resolve_project_from_session()` and `_resolve_project_from_engram()`

### Step 1.5 — Reduce `stream_call` Bumpy Road
- Extract `_authorize_tool_call()` to deduplicate shared auth+error pattern

### New Tests
- `api/tests/test_mcp_service_unit.py` — dispatcher routing, authorization helpers, unknown method error
- Expand `test_mcp_api_integration.py` — round-trip `tools/call` for engram.list, project.list, unknown tool

---

## Phase 2: `api/app/memory_admin/repository.py` (6.29 -> 8.0+)

Key issues: Low cohesion, code duplication (soft_delete/restore), `update_admin_engram` 114 LoC + 10 args

### Step 2.1 — Decompose `update_admin_engram`
- Extract `_build_engram_update_fields(current, payload) -> dict`
- Extract `_replace_engram_sources(cur, engram_id, sources)`
- Extract `_compute_engram_embedding(retrieval_text, dim) -> tuple`

### Step 2.2 — Eliminate Soft Delete/Restore Duplication
- Extract `_soft_delete_record(conn, table, id_column, id_value, deleted_by, reason)`
- Extract `_restore_record(conn, table, id_column, id_value)`
- Session/engram/collection delete+restore become thin wrappers

### New Tests
- `api/tests/test_memory_admin_repository.py` — update field building, soft-delete/restore lifecycle

---

## Phase 3: `api/app/chat_repository.py` (6.69 -> 8.0+)

Key issues: Code duplication (pin/unpin mirror pairs), `create_chat_message` 8 args

### Step 3.1 — Introduce `MessageMetadata` Dataclass
- Bundle `provider`, `model_id`, `token_usage_json`, `used_engram_ids` into `MessageMetadata | None`
- Reduces `create_chat_message` from 8 to 5 args

### Step 3.2 — Eliminate Pin/Unpin Duplication
- Extract `_pin_resource_to_session(table, id_column, resource_id, session_id, actor, conn)`
- Extract `_unpin_resource_from_session(table, id_column, resource_id, session_id, actor, conn)`
- Four public functions become thin callers (~60-80 lines eliminated)

### Step 3.3 — Extract `_CHAT_SESSION_COLUMNS` Constant
- Same 14-column SELECT repeated 4 times; extract to module constant

### New Tests
- Expand `test_chat_repository.py` — metadata tests, pin/unpin round-trips

---

## Phase 4: `api/app/chat/service.py` (6.77 -> 8.0+)

Key issues: Bumpy road, `_raise_provider_error` CC=12, `stream_message_events` 98 LoC, `_build_debug_trace` 7 args

### Step 4.1 — Replace `_raise_provider_error` With Dispatch Table
- Create `_PROVIDER_ERROR_MAP: list[tuple[type, int, str]]` mapping exception types to status codes
- Reduces CC from 12 to ~3

### Step 4.2 — Decompose `_build_debug_trace`
- Extract `_build_provider_message_debug()`, `_build_embedding_call_debug()`, `_resolve_token_usage()`
- Introduce `_DebugBuildContext` dataclass to reduce 7 args to 4

### Step 4.3 — Reduce `stream_message_events` Length
- Extract `_yield_stream_chunks()` and `_build_stream_done_payload()`

### New Tests
- Expand `test_chat_service.py` — error mapping, debug trace building, token usage estimation

---

## Phase 5: `api/app/main.py` (7.66 -> 9.0+)

Key issues: Low cohesion, complex conditionals, `ui_admin_create_mcp_token` 7 args

### Step 5.1 — Move MCP Token Helpers to `mcp_tokens/` Service
### Step 5.2 — Extract `_McpTokenFormPayload` for 7-arg Route Handler
### Step 5.3 — Extract `_validate_login_preconditions()` to Simplify Login
### New Tests — Expand `test_ui_auth.py` with CSRF, safe-path, inactive-user tests

---

## Phase 6: `api/app/oauth/api.py` (7.48 -> 9.0+)

Key issues: `create_oauth_router` CC=35, 323 LoC monolith

### Step 6.1 — Extract Route Bodies from Closure Factory
- Extract `_handle_oauth_register()`, `_handle_oauth_authorize()`, `_handle_oauth_token()`
- Factory reduces to ~40 lines of route registration

### Step 6.2 — Extract `_validate_oauth_client_and_redirect()` to Reduce Bumps
### Step 6.3 — Extract `_dedup_string_list()` Shared Helper
### New Tests — `test_oauth_api_unit.py` (new), expand `test_mcp_oauth_integration.py`

---

## Phase 7: `api/app/repository.py` (7.88 -> 9.0+)

Key issues: `get_rehydration_bundle` CC=19, `query_engrams` CC=19, `create_engram_with_report` 120 LoC

### Step 7.1 — Decompose `get_rehydration_bundle`
- Extract `_fetch_engram_row()`, `_fetch_source_rows()`, `_format_citations()`, `_format_decisions()`

### Step 7.2 — Decompose `query_engrams`
- Extract `_build_engram_query_where()` and `_rerank_by_combined_score()`

### Step 7.3 — Decompose `create_engram_with_report`
- Extract `_build_engram_persistence_payload()`, `_persist_engram_sources()`, `_persist_engram_artifacts()`

### New Tests — `api/tests/test_repository_unit.py` — WHERE builder, re-ranking, citation formatting

---

## Phase 8: `api/app/providers/bedrock_provider.py` (8.17 -> 9.5+)

### Step 8.1 — Reduce `_raise_invocation_error` CC=16 with `_AWS_AUTH_ERROR_CODES` constant + extracted helpers
### Step 8.2 — Simplify `_extract_text` Deep Nesting (ACE-eligible)
### Step 8.3 — Introduce `AwsCredentials` Dataclass to Reduce `__init__` to 2 Args
### New Tests — Expand `test_provider_adapters.py` with throttling, auth, no-credentials, empty-content tests

---

## Phase 9: `api/app/chat/context.py` (8.38 -> 9.5+)

### Step 9.1 — Introduce `ChatContextRequest` Dataclass, Extract Bundle Collectors
### Step 9.2 — Reduce `_bundle_section` CC=10 with `_format_list_section()` Helper
### New Tests — Expand `test_chat_context.py` with empty pinned, truncation, dedup tests

---

## Phase 10: `web/src/App.tsx` (8.28 -> 9.5+)

### Step 10.1 — Extract Custom Hooks
- `useAuth()` in `web/src/hooks/useAuth.ts`
- `useChatSessions()` in `web/src/hooks/useChatSessions.ts`
- `useEngramPin()` in `web/src/hooks/useEngramPin.ts`
- `useDocumentIngestion()` in `web/src/hooks/useDocumentIngestion.ts`

### Step 10.2 — Eliminate Pin/Unpin Handler Duplication (absorbed by hooks)
### Step 10.3 — Create `ChatSessionFormPayload` Type in `web/src/api/types.ts`
### Step 10.4 — Extract `sendPrompt` Event Dispatch to `useChatSend` Hook
### New Tests — `App.test.tsx`, `useAuth.test.ts`, `useChatSessions.test.ts`

---

## Phase 11: `api/app/ingestion/repository.py` (8.64 -> 9.5+)

### Step 11.1 — Introduce `DocumentUpsertPayload` Dataclass (13 args -> 3)
- Extract `_upsert_document_record()`, `_replace_document_chunks()`
### New Tests — Expand `test_ingestion_service.py`

---

## Phase 12: `api/app/engram_enrichment/service.py` (8.93 -> 9.5+)

### Step 12.1 — Decompose `extract_keywords` Bumpy Road (ACE-eligible)
- Extract `_extract_from_assistant_sections()`, `_extract_from_tag_keyword_map()`, `_deduplicate_keywords()`
### New Tests — Expand `test_engram_enrichment.py`

---

## Phase 13: Green-Zone Files (9.09–9.66 -> 10.0)

### 13a: `memory_admin/service.py` (9.09) — Parameter objects for 5-7 arg list methods, dedup lifecycle pattern
### 13b: `ingestion/service.py` (9.33) — Extract `_validate_file_input()`, introduce `FileIngestRequest`
### 13c: `mcp/auth.py` (9.66) — Extract `_resolve_token_auth_context()` from CC=12 `resolve_mcp_actor`
### New Tests for each

---

## Phase 14: Test Infrastructure (Parallel)

### 14.1 — Add pytest-cov
- Add to `pyproject.toml` dev deps, add `make coverage` target, start at 60% threshold

### 14.2 — Add Missing Backend Tests
- `test_audit.py` — audit event recording
- `test_login_guard.py` — lockout, window expiry, reset
- `test_user_repository.py` — CRUD with mock DB
- `test_db.py` — idempotent schema init

### 14.3 — Add Missing Frontend Tests
- `api/chat.test.ts` — stream events, session listing
- `api/http.test.ts` — ApiError, auth headers
- `api/ingestion.test.ts` — multipart form
- `api/mcpTokens.test.ts` — token CRUD

### 14.4 — Add OAuth Layer Tests
- `test_oauth_repository.py` — client CRUD, code consumption

---

## Architectural Constraints (Must Preserve)

1. DB access stays in `*_repository.py` — no SQL in services
2. Business logic stays out of `main.py` — route handlers call services only
3. `chat/lifecycle_policy.py` remains pure (no side effects)
4. MCP tool handlers call service/repository layers — never raw SQL
5. Provider SDK calls stay in `api/app/providers/` only
6. Engram enrichment remains fill-empty-only
7. Single `db/init/001_schema.sql` approach preserved
8. All public API contracts remain backward-compatible

---

## Verification

After each commit:
- `make test` passes (backend)
- `make web-check` passes (frontend, when applicable)
- `pre_commit_code_health_safeguard` confirms improvement
- No regression in `test_mcp_api_integration.py` (26 tests)
- No regression in `test_chat_api_integration.py` (9 tests)

Full quality gate at end of each phase: `make check && make web-check`
