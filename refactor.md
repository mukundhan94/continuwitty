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

### 2026-02-19 (Checkpoint 3 - Chat Repository Phase 3)

- [x] Step 3.1 — Introduced `MessageMetadata` dataclass for `create_chat_message`
- [x] Reduced `create_chat_message` call shape from 8 args to 5 args (`metadata` payload object)
- [x] Step 3.2 — Eliminated pin/unpin duplication
- [x] Added `_pin_resource_to_session(...)` helper for engram/document pin flows
- [x] Added `_unpin_resource_from_session(...)` helper for engram/document unpin flows
- [x] Step 3.3 — Extracted `_CHAT_SESSION_COLUMNS` constant and reused across session SELECT/RETURNING queries
- [x] Test updates: `api/tests/test_chat_repository.py` metadata + round-trip assertions
- [x] Test updates: `api/tests/test_chat_service.py` metadata-aware create-message assertions
- [x] Validation run: `api/tests/test_chat_repository.py`, `api/tests/test_chat_service.py` (10 passed, 7 skipped)
- [x] Acceptance run: `make acceptance-test-mock-docker` (10 passed)
- [x] CodeScene health checks run (`code_health_review`, `pre_commit_code_health_safeguard`)
- [x] CodeScene score improvement: `api/app/chat_repository.py` from **6.69** to **7.10**

---

### 2026-02-19 (Checkpoint 4 - Chat Service Phase 4.1)

- [x] Step 4.1 — Replaced `_raise_provider_error` branch chain with `_PROVIDER_ERROR_STATUS_MAP` dispatch table
- [x] Fixed unknown-exception passthrough in `_raise_provider_error` (`raise exc` outside mapped provider errors)
- [x] Expanded tests: provider error mapping now covers request/rate-limit/auth/api/generic provider exceptions
- [x] Added unknown-exception passthrough test in `api/tests/test_chat_service.py`
- [x] Validation run: `api/tests/test_chat_service.py`, `api/tests/test_chat_repository.py` (15 passed, 7 skipped)
- [x] Acceptance run: `make acceptance-test-mock-docker` (10 passed)
- [x] CodeScene health checks run (`code_health_review`, `pre_commit_code_health_safeguard`)
- [x] CodeScene score improvement: `api/app/chat/service.py` from **6.77** to **6.95**

---

### 2026-02-19 (Checkpoint 5 - Chat Service Phase 4.2/4.3)

- [x] Step 4.2 — Decomposed `_build_debug_trace`
- [x] Added `_DebugBuildContext` dataclass to reduce `_build_debug_trace` argument count
- [x] Extracted `_build_provider_message_debug(...)`, `_build_embedding_call_debug(...)`, and `_resolve_token_usage(...)`
- [x] Step 4.3 — Reduced `stream_message_events` method length
- [x] Extracted `_build_stream_meta_payload(...)`, `_yield_stream_chunks(...)`, `_persist_stream_completion(...)`, `_build_stream_done_payload(...)`
- [x] Added tests in `api/tests/test_chat_service.py`:
  - token usage resolution (provider total vs estimated fallback)
  - stream event flow (`meta`/`chunk`/`done`) with persisted completion payload assertions
- [x] Validation run: `api/tests/test_chat_service.py`, `api/tests/test_chat_repository.py` (25 passed)
- [x] Acceptance run: `make acceptance-test-mock-docker` (10 passed)
- [x] CodeScene health checks run (`code_health_review`, `pre_commit_code_health_safeguard`)
- [x] CodeScene score improvement: `api/app/chat/service.py` from **6.95** to **7.90**

---

### 2026-02-19 (Checkpoint 6 - Main Module Phase 5.1/5.2/5.3)

- [x] Step 5.1 — Moved MCP token orchestration helpers out of `api/app/main.py` into `api/app/mcp_tokens/service.py`
- [x] Added high-level token service helpers:
  - `create_token_for_owner(...)`
  - `list_token_summaries(...)`
  - `revoke_token_for_owner(...)`
- [x] Updated `api/app/main.py` UI + API token endpoints to use service-layer helpers
- [x] Step 5.2 — Reduced `ui_admin_create_mcp_token` argument footprint using `_McpTokenFormPayload` dependency parser
- [x] Step 5.3 — Added `_validate_login_preconditions(...)` and simplified `login_submit(...)` flow
- [x] New test coverage: expanded `api/tests/test_mcp_token_service.py` for create/list/revoke service helpers
- [x] Validation run:
  - `api/tests/test_mcp_token_service.py`
  - `api/tests/test_admin_mcp_tokens_ui.py`
  - `api/tests/test_mcp_token_api_integration.py`
  - `api/tests/test_ui_auth.py::test_login_rejects_invalid_credentials`
  - `api/tests/test_ui_auth.py::test_login_rate_limit_after_repeated_failures`
  - `api/tests/test_ui_auth.py::test_login_rejects_invalid_csrf`
  - Aggregate: `7 passed, 4 skipped` plus `3 passed` (precondition-focused auth tests)
- [x] Acceptance run: `make acceptance-test-mock-docker` (10 passed)
- [x] CodeScene health checks run (`code_health_review`, `pre_commit_code_health_safeguard`)
- [x] CodeScene notes:
  - `api/app/main.py` score remains **7.66** (stable)
  - `ui_admin_create_mcp_token` excess-arguments smell fixed
  - quality gate passed

---

### 2026-02-19 (Checkpoint 7 - OAuth API Phase 6.1/6.2/6.3)

- [x] Step 6.1 — Extracted OAuth route bodies from `create_oauth_router(...)` into top-level handlers:
  - `_handle_oauth_register(...)`
  - `_handle_oauth_authorize(...)`
  - `_handle_oauth_token(...)`
- [x] Reduced router closure responsibilities; `create_oauth_router(...)` now primarily wires endpoints to extracted handlers
- [x] Step 6.2 — Added `_validate_oauth_client_and_redirect(...)` for shared client + redirect validation logic in authorize flow
- [x] Step 6.3 — Added `_dedup_string_list(...)` and reused it for metadata/redirect URI normalization
- [x] New unit test file: `api/tests/test_oauth_api_unit.py`
- [x] Integration expanded: `api/tests/test_mcp_oauth_integration.py` confidential-client token secret requirement
- [x] Validation run:
  - `api/tests/test_oauth_api_unit.py`
  - `api/tests/test_oauth_service.py`
  - `api/tests/test_mcp_oauth_integration.py`
  - Aggregate: `9 passed, 6 skipped`
- [x] Acceptance run: `make acceptance-test-mock-docker` (10 passed)
- [x] CodeScene health checks run (`code_health_review`, `pre_commit_code_health_safeguard`)
- [x] CodeScene notes:
  - `api/app/oauth/api.py` score improved from **7.48** to **7.54**
  - `create_oauth_router` complex-method and bumpy-road findings resolved
  - pre-commit quality gate passed

---

### 2026-02-19 (Checkpoint 8 - Repository Phase 7.1/7.2)

- [x] Step 7.1 — Decomposed `get_rehydration_bundle(...)`:
  - Added `_fetch_engram_row(...)` for scoped engram lookup
  - Added `_fetch_source_rows(...)` for citations source retrieval
  - Added `_format_citations(...)` for citation markdown rendering
  - Added `_format_decisions(...)` for key-decision markdown rendering
- [x] Step 7.2 — Decomposed `query_engrams(...)`:
  - Added `_build_engram_query_where(...)` to isolate WHERE clause + params assembly
  - Added `_rerank_by_combined_score(...)` to isolate lexical/dense reranking logic
- [x] New unit test file: `api/tests/test_repository_unit.py`
- [x] Validation run:
  - `api/tests/test_repository_helpers.py`
  - `api/tests/test_repository_unit.py`
  - `api/tests/test_engram_visibility.py`
  - `api/tests/test_api_integration.py::test_roundtrip_create_list_query_rehydrate`
  - `api/tests/test_api_integration.py::test_rehydrate_packs_unique_citations`
  - `api/tests/test_api_integration.py::test_rehydrate_uses_detailed_summary_when_abstract_is_generic`
  - Aggregate: `13 passed, 4 skipped`
- [x] Acceptance run: `make acceptance-test-mock-docker` (10 passed)
- [x] CodeScene health checks run (`code_health_review`, `pre_commit_code_health_safeguard`)
- [x] CodeScene notes:
  - `api/app/repository.py` score improved from **7.88** to **9.05**
  - remaining primary smell in file is `create_engram_with_report` method size
  - pre-commit quality gate passed

---

### 2026-02-20 (Checkpoint 9 - Bedrock Provider Phase 8.1/8.2/8.3)

- [x] Step 8.1 — Reduced `_raise_invocation_error(...)` complexity with explicit AWS error-code groups and helper dispatch:
  - Added `_AWS_AUTH_ERROR_CODES`, `_AWS_RATE_LIMIT_ERROR_CODES`, `_AWS_REQUEST_ERROR_CODES`
  - Added `_invocation_error_from_code(...)`, `_is_missing_credentials_error(...)`, `_is_partial_credentials_error(...)`
- [x] Step 8.2 — Simplified `_extract_text(...)` nesting depth:
  - Added `_extract_text_part(...)` and flattened content extraction flow
- [x] Step 8.3 — Introduced `AwsCredentials` dataclass and reduced `BedrockProvider.__init__` argument footprint
- [x] Updated provider registry wiring to pass `AwsCredentials(...)`
- [x] Test updates:
  - Expanded `api/tests/test_provider_adapters.py` with throttling/auth/empty-content Bedrock coverage
  - Updated `api/tests/test_provider_registry.py` Bedrock credential assertions
- [x] Validation run:
  - `api/tests/test_provider_adapters.py`
  - `api/tests/test_provider_registry.py`
  - Aggregate: `13 passed`
- [x] Acceptance run: `make acceptance-test-mock-docker` (10 passed)
- [x] CodeScene health checks run (`code_health_review`, `pre_commit_code_health_safeguard`)
- [x] CodeScene notes:
  - `api/app/providers/bedrock_provider.py` score improved from **8.17** to **10.0**
  - fixed: complex method, deep nested complexity, excess arguments, and overall complexity thresholds
  - pre-commit quality gate passed

---

### 2026-02-20 (Checkpoint 10 - Chat Context Phase 9.1/9.2)

- [x] Step 9.1 — Introduced `ChatContextRequest` dataclass and extracted context assembly helpers:
  - Added `_select_context_engram_ids(...)`
  - Added `_collect_rehydration_bundles(...)`
  - Added `_split_document_chunks(...)`
  - Added `_build_context_sections(...)`
  - Updated call sites to pass request object (`chat/service.py`, `test_chat_context.py`)
- [x] Step 9.2 — Reduced `_bundle_section(...)` complexity:
  - Added `_format_list_section(...)`
  - Added `_truncate_detailed_excerpt(...)`
  - Added `_citation_line(...)`
- [x] Test updates:
  - Expanded `api/tests/test_chat_context.py` with truncation and empty-source behavior coverage
  - Reduced oversized test-method growth by introducing shared `_context_request(...)` helper
- [x] Validation run:
  - `api/tests/test_chat_context.py`
  - `api/tests/test_chat_service.py`
  - Aggregate: `24 passed`
- [x] Acceptance run: `make acceptance-test-mock-docker` (10 passed)
- [x] CodeScene health checks run (`code_health_review`, `pre_commit_code_health_safeguard`)
- [x] CodeScene notes:
  - `api/app/chat/context.py` score improved from **8.38** to **10.0**
  - fixed: `assemble_chat_context` complexity + excess-args, `_bundle_section` complexity
  - pre-commit quality gate passed

---

### 2026-02-20 (Checkpoint 11 - Ingestion Repository Phase 11.1)

- [x] Step 11.1 — Introduced `DocumentUpsertPayload` dataclass for document persistence input
- [x] Refactored `upsert_document_with_chunks(...)` call shape from many scalar args to:
  - `actor_user_id`
  - `payload: DocumentUpsertPayload`
  - `embedding_dim`
- [x] Extracted persistence helpers:
  - `_upsert_document_record(...)`
  - `_replace_document_chunks(...)`
  - `_build_document_chunk_where(...)`
  - `_rerank_document_chunk_rows(...)`
- [x] Moved SQL literals to module constants to reduce method size and improve scanability:
  - `_UPSERT_DOCUMENT_SQL`
  - `_INSERT_DOCUMENT_CHUNK_SQL`
  - `_QUERY_DOCUMENT_CHUNKS_SELECT_TEMPLATE`
- [x] Updated service + integration call sites for new payload contract:
  - `api/app/ingestion/service.py`
  - `api/tests/test_ingestion_service.py`
  - `api/tests/test_chat_repository.py`
- [x] Validation run:
  - `api/tests/test_ingestion_service.py`
  - `api/tests/test_chat_repository.py::test_pin_and_unpin_document`
  - `api/tests/test_ingestion_api_integration.py::test_ingestion_text_and_query_flow`
  - Aggregate: `3 passed, 2 skipped`
- [x] Acceptance run: `make acceptance-test-mock-docker` (10 passed)
- [x] CodeScene health checks run (`code_health_review`, `pre_commit_code_health_safeguard`)
- [x] CodeScene notes:
  - `api/app/ingestion/repository.py` score improved from **8.64** to **10.0**
  - fixed: `upsert_document_with_chunks` argument-count and large-method smells
  - pre-commit quality gate passed

---

### 2026-02-20 (Checkpoint 12 - Engram Enrichment Phase 12.1)

- [x] Step 12.1 — Decomposed `extract_keywords(...)` bumpy-road logic
- [x] Added helper extractions:
  - `_extract_from_assistant_sections(...)`
  - `_extract_from_tag_keyword_map(...)`
  - `_deduplicate_keywords(...)`
  - `_ordered_keyword_candidates(...)`
  - `_token_frequency(...)`
  - `_is_keyword_candidate(...)`
- [x] Preserved deterministic ranking behavior while layering assistant-section and tag-map keyword seeds through helper pipeline
- [x] Test updates in `api/tests/test_engram_enrichment.py`:
  - assistant-section keyword extraction coverage
  - deduplication + max-keyword limit coverage
- [x] Validation run:
  - `api/tests/test_engram_enrichment.py`
  - Aggregate: `7 passed`
- [x] Acceptance run: `make acceptance-test-mock-docker` (10 passed)
- [x] CodeScene health checks run (`code_health_review`, `pre_commit_code_health_safeguard`)
- [x] CodeScene notes:
  - `api/app/engram_enrichment/service.py` score improved from **8.93** to **10.0**
  - fixed: `extract_keywords` bumpy-road and complex-method findings
  - pre-commit quality gate passed

---

### 2026-02-20 (Checkpoint 13 - Memory Admin Service Phase 13a)

- [x] Step 13a — Refactored list APIs in `api/app/memory_admin/service.py` to use parameter objects:
  - added `MemoryAdminListRequest`
  - added `MemoryAdminEngramListRequest`
  - updated `list_sessions(...)`, `list_engrams(...)`, and `list_collections(...)` to accept request objects
- [x] Deduplicated list-query lifecycle pattern:
  - added `_list_project_scoped_records(...)`
  - reused shared helper for sessions + collections list paths
- [x] Consolidated stale-write conflict guard:
  - added `_raise_if_stale_update(...)`
  - reused in `update_engram(...)`, `move_engram(...)`, and `update_collection(...)`
- [x] API + MCP call sites updated for new request-object contract:
  - `api/app/memory_admin/api.py`
  - `api/app/mcp/service.py`
  - `api/app/memory_admin/__init__.py` export surface
- [x] New tests added: `api/tests/test_memory_admin_service.py`
  - verifies request-object forwarding for list methods
  - verifies stale `expected_updated_at` conflict handling
- [x] Validation run:
  - `uv run ruff check app/memory_admin/service.py app/memory_admin/api.py app/memory_admin/__init__.py app/mcp/service.py tests/test_memory_admin_service.py`
  - `uv run pytest -q tests/test_memory_admin_service.py tests/test_memory_admin_api_integration.py tests/test_mcp_service_unit.py`
  - aggregate: `15 passed`
- [x] Acceptance run: `make acceptance-test-mock-docker` (10 passed on port `5173`)
- [x] CodeScene health checks run (`code_health_review`, `pre_commit_code_health_safeguard`)
- [x] CodeScene notes:
  - `api/app/memory_admin/service.py` score improved from **9.09** to **10.0**
  - `api/app/memory_admin/api.py` score currently **9.38**
  - `pre_commit_code_health_safeguard` quality gate: **passed**

---

### 2026-02-20 (Checkpoint 14 - Ingestion Service Phase 13b)

- [x] Step 13b — Refactored file ingestion call shape with parameter object:
  - added `FileIngestRequest` in `api/app/ingestion/service.py`
  - updated `DocumentIngestionService.ingest_file(...)` to accept `file_request`
  - updated `api/app/ingestion/api.py` call site to construct and pass `FileIngestRequest`
- [x] Extracted file validation helpers to reduce complexity and improve testability:
  - `_validate_file_size(...)`
  - `_decode_file_text(...)`
  - `_validate_file_input(...)`
  - `_resolve_file_title(...)`
  - `_merge_file_metadata(...)`
- [x] Deduplicated text/file chunking lifecycle:
  - added `ChunkBuildRequest`
  - added `_build_chunks_for_document(...)`
  - reused in both `ingest_text(...)` and `ingest_file(...)`
- [x] Simplified ingestion API route helper logic:
  - added `_parse_metadata_json(...)`
  - added `_build_file_ingest_request(...)`
- [x] Test updates in `api/tests/test_ingestion_service.py`:
  - added parametrized file-validation error coverage (UTF-8, empty file, oversized file)
  - added metadata + title fallback persistence coverage for file ingestion
- [x] Validation run:
  - `uv run ruff check app/ingestion/service.py app/ingestion/api.py tests/test_ingestion_service.py`
  - `uv run pytest -q tests/test_ingestion_service.py tests/test_ingestion_api_integration.py`
  - `make test`
  - aggregate: `8 passed` (targeted), `229 passed` (full backend suite)
- [x] Acceptance run: `make acceptance-test-mock-docker` (10 passed on port `5173`)
- [x] CodeScene health checks run (`code_health_review`, `pre_commit_code_health_safeguard`)
- [x] CodeScene notes:
  - `api/app/ingestion/service.py` score improved from **9.33** to **10.0**
  - `pre_commit_code_health_safeguard` quality gate: **passed**

---

## CodeScene Health Scorecard (Current State)

| File | Score | Severity |
|------|-------|----------|
| `api/app/mcp/service.py` | **4.56** | YELLOW |
| `api/app/memory_admin/repository.py` | 7.42 | YELLOW |
| `api/app/chat_repository.py` | 7.10 | YELLOW |
| `api/app/chat/service.py` | 7.90 | YELLOW |
| `api/app/oauth/api.py` | 7.54 | YELLOW |
| `api/app/main.py` | 7.66 | YELLOW |
| `api/app/repository.py` | 9.05 | GREEN |
| `api/app/providers/bedrock_provider.py` | 10.0 | GREEN |
| `web/src/App.tsx` | 8.28 | YELLOW |
| `api/app/chat/context.py` | 10.0 | GREEN |
| `api/app/ingestion/repository.py` | 10.0 | GREEN |
| `api/app/engram_enrichment/service.py` | 10.0 | GREEN |
| `api/app/memory_admin/service.py` | 10.0 | GREEN |
| `api/app/ingestion/service.py` | 10.0 | GREEN |
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
