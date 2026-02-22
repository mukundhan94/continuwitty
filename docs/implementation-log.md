# Engram Vault - Implementation Log

> Chronological log of implementation changes by date and pass.
> See [README](../README.md) for project overview. See [checkpoint.md](../checkpoint.md) for milestone summary.

---

## Implementation Log

### 2026-02-22 (Phase 33 export service code-health pass)

1. Refactored `api/app/export/service.py` to remove duplicated lookup patterns in import flows and consolidate lookup logic through a shared helper.
2. Kept export/import behavior unchanged while reducing duplication flagged by CodeScene.
3. Verification:
   - `cd api && uv run ruff check app/export/service.py`
   - `cd api && uv run pytest -q tests/test_export_api.py tests/test_export_api_integration.py`
   - CodeScene Code Health review for `api/app/export/service.py`: `10.0`

### 2026-02-22 (Dev workflow contract test fix)

1. Fixed a single backend test failure in `tests/test_dev_workflow_contract.py::test_readme_quick_start_uses_new_local_run_defaults` by aligning `README.md` quick-start wording with the expected local workflow contract.
2. Added explicit quick-start phrases for:
   - preferred local run mode (`make dev`)
   - optional split-terminal mode (`make api`, `make web`)
   - container stack usage as occasional debugging mode (`make stack-up`).
3. Verification:
   - `cd api && uv run pytest -q tests/test_dev_workflow_contract.py`
   - `cd api && uv run pytest -q`

### 2026-02-15 (Engram panel UX pass)

1. Updated engram cards to show only titles by default and reveal full markdown abstract on hover/focus.
2. Added pinned-state visual highlighting so selected engrams are clearly distinguishable.
3. Switched pinned/unpinned layout to a fixed split with a stable dotted mid-divider.
4. Added frontend tests for split layout, pinned selected-state semantics, and markdown-link rendering.
5. Verification:
   - `make web-check` -> all lint, unit tests, and production build checks passed.

### 2026-02-15 (Engram tooltip refinement pass)

1. Replaced inline abstract expansion with a floating tooltip that keeps card height stable.
2. Kept tooltip content markdown-rendered and scrollable for long abstracts.
3. Reduced engram action button sizing to keep the panel compact and readable.
4. Switched tooltip to an opaque card surface and tightened markdown alignment/spacing for cleaner readability.
5. Added markdown normalization before tooltip rendering so mixed inline separators/headings format correctly.
6. Increased tooltip viewport size and line spacing for clearer long-form abstract previews.
7. Verified behavior in Playwright against local UI (`http://localhost:5173`) and retained panel split/divider behavior.
8. Updated tooltip sizing/positioning to be viewport-aware so it no longer overflows or breaks compact screens.
9. Verification:
   - `make web-check` -> all lint, unit tests, and production build checks passed.

### 2026-02-15 (User flow screenshot documentation pass)

1. Added a dedicated UI workflow document:
   - `docs/user-flow-engram-workflow.md`.
2. Re-captured an expanded ordered screenshot set for the full local flow under:
   - `docs/screenshots/user-flow/01-login.png`
   - ...
   - `docs/screenshots/user-flow/27-engram-4-saved-final-catalog.png`
3. Documented a complete continuity chain with the enforced pattern:
   - save engram
   - continue in new chat
   - pin all available engrams
   - continue conversation
4. The final documented flow now covers 4 engrams across 4 linked sessions using the same default model.
4. Updated README docs navigation and file-by-file guide to include the new workflow doc and screenshot assets.
5. Verification:
   - confirmed all workflow screenshot files are present and embedded in order in the user-flow document.

### 2026-02-15 (Unified roadmap phase 16: MCP interoperability + typed clients)

1. Added MCP interoperability methods in `api/app/mcp/service.py` for external MCP clients:
   - `initialize`
   - `tools/list`
   - `tools/call`
2. Kept existing direct tool methods (`chat.*`, `engram.*`, `user.*`) for backward compatibility.
3. Added streaming compatibility for `tools/call` when invoking `chat.send_message`:
   - emits `mcp.event` progress frames with stable request correlation.
   - returns tool-call style final result payload in JSON-RPC success frame.
4. Added typed Python client helper:
   - `api/app/mcp/client.py` (`McpSseClient`, typed frame models, SSE parser, login helper).
5. Added typed TypeScript client helper:
   - `web/src/api/mcpClient.ts` (`streamMcpCall`, frame parsers, final-frame helpers).
6. Added contract tests:
   - `api/tests/test_mcp_api_integration.py`: compatibility envelopes and `tools/call` stream flow.
   - `api/tests/test_mcp_client.py`: Python helper auth/transport/protocol coverage.
   - `web/src/api/mcpClient.test.ts`: TypeScript helper protocol parsing and SSE handling.
7. Updated docs and skills:
   - README MCP usage expanded with compatibility invocation paths and tool-group examples.
   - `skills/mcp-http-stream-tools/SKILL.md` updated with compatibility and client-helper expectations.
   - `AGENT.md` updated with MCP contract synchronization rule for typed clients.
8. Validation:
   - `cd api && uv run ruff check app/mcp tests/test_mcp_api_integration.py tests/test_mcp_client.py`
   - `cd api && uv run pytest -q tests/test_mcp_api_integration.py tests/test_mcp_client.py`
   - `cd web && npm run lint`
   - `cd web && npm run test -- src/api/mcpClient.test.ts`
9. Future consideration:
   - evaluate `FastMCP` as an optional adapter layer for broader third-party MCP ecosystem integration.
   - keep current FastAPI MCP implementation as the source of truth unless an explicit migration phase is approved.

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

### 2026-02-15 (Multi-user RBAC phase)

1. Added multi-user store with seeded local admin account (`users` table).
2. Added role model (`admin`, `analyst`, `viewer`) and session role propagation.
3. Added role-protected admin APIs:
   - `GET /api/v1/users`
   - `POST /api/v1/users`
   - `PATCH /api/v1/users/{user_id}`
   - `GET /api/v1/me`
4. Added admin UI route:
   - `GET /ui/admin` (admin role required)
5. Added tests for admin management and non-admin access denial.
6. Verification:
   - `make lint` -> all checks passed
   - `make test` -> `29 passed`

### 2026-02-15 (Evaluation harness phase)

1. Added a local evaluation harness under `api/evals`:
   - fact recall
   - cross-engram reasoning proxy
   - temporal correctness
   - abstention on empty results
2. Added `make eval` and JSON output write to `api/evals/last_eval.json`.
3. Added integration test coverage for harness execution:
   - `api/tests/test_eval_harness.py`
4. Verification:
   - `make lint` -> all checks passed
   - `make test` -> `30 passed`
   - `make eval` -> `4/4` cases passed (score `1.0`)

### 2026-02-15 (Local CLI workflow phase)

1. Added terminal CLI entrypoint (`api/app/cli.py`) for operator workflows:
   - `upload` from JSON file (single object or list)
   - `search` with semantic query + metadata filters
   - `rehydrate` by engram id
2. Added Makefile CLI passthrough:
   - `make cli ARGS="..."`
3. Added CLI tests:
   - `api/tests/test_cli.py`
4. Verification:
   - `make lint` -> all checks passed
   - `make test` -> `34 passed`
   - `make eval` -> `4/4` cases passed (score `1.0`)

### 2026-02-15 (Retrieval quality phase)

1. Added retrieval reranking in `query_engrams`:
   - combines dense vector distance with lexical token overlap
   - fetches a wider candidate window, then reranks locally
2. Added citation packing improvements in `get_rehydration_bundle`:
   - deduplicates citations by URL
   - preserves latest-first order
   - includes snippet previews in context markdown
3. Added/updated tests:
   - helper tests for lexical overlap, combined score, and citation dedupe
   - integration test verifying unique citation packing in rehydration output
4. Verification:
   - `make lint` -> all checks passed
   - `make test` -> `38 passed`
   - `make eval` -> `4/4` cases passed (score `1.0`)

### 2026-02-15 (Consolidation maintenance phase)

1. Added local consolidation service (`api/app/consolidation.py`) to create periodic summary engrams.
2. Added guardrails:
   - skip when a recent consolidation engram already exists
   - require a minimum number of non-consolidated source engrams
   - support dry-run mode
3. Extended CLI:
   - `engram-cli consolidate`
   - `make consolidate ARGS="--project-id <id> [--dry-run]"`
4. Added tests:
   - `api/tests/test_consolidation.py`
   - CLI coverage for `consolidate` command
5. Verification:
   - `make lint` -> all checks passed
   - `make test` -> `42 passed`
   - `make eval` -> `4/4` cases passed (score `1.0`)

### 2026-02-15 (Local security baseline phase)

1. Added local audit logging (`api/app/audit.py`):
   - append-only JSONL events with timestamp, route, IP, actor, and outcome
2. Added login protection guard (`api/app/login_guard.py`):
   - per-user+IP attempt tracking
   - rate-limit window + lockout duration controls
3. Hardened auth and admin routes:
   - login CSRF rejection events
   - login failed/success/rate-limited events
   - logout CSRF rejection/success events
   - user create/update audit events
4. Added environment toggles in `.env.example`:
   - `AUDIT_LOG_PATH`
   - `LOGIN_RATE_LIMIT_MAX_ATTEMPTS`
   - `LOGIN_RATE_LIMIT_WINDOW_SECONDS`
   - `LOGIN_LOCKOUT_SECONDS`
5. Added tests:
   - rate-limit behavior after repeated failed logins
   - audit-log write on failed login
6. Verification:
   - `make lint` -> all checks passed
   - `make test` -> `44 passed`
   - `make eval` -> `4/4` cases passed (score `1.0`)

### 2026-02-15 (Unified roadmap phase 10: schema and repository layer)

1. Added schema extensions for continuity and access control:
   - `engrams.owner_user_id`
   - `engrams.visibility_scope` (`private`/`project`)
   - `engrams.source_session_id`
   - new tables: `chat_sessions`, `chat_messages`, `session_pinned_engrams`
2. Added compatibility bootstrap:
   - `ensure_schema_initialized()` runs schema DDL at app startup for existing local databases.
3. Added chat repository module:
   - `api/app/chat_repository.py` with create/list/get/update session operations
   - message write/list operations
   - pin/unpin/list pinned engrams with visibility enforcement
4. Extended data contracts in `api/app/models.py`:
   - chat/session/message/pin request-response models
   - visibility and provider enums
   - MCP JSON-RPC request/response types
5. Added integration tests:
   - `api/tests/test_chat_repository.py`
   - `api/tests/test_engram_visibility.py`
6. Verification:
   - `make lint` -> all checks passed
   - `make test` -> `49 passed`
   - `make eval` -> `4/4` cases passed (score `1.0`)

### 2026-02-15 (Unified roadmap phase 11: provider adapter layer)

1. Added provider domain package under `api/app/providers`:
   - shared adapter contract (`base.py`)
   - provider errors (`errors.py`)
   - concrete adapters: OpenAI, Anthropic, Bedrock
   - central registry resolver (`registry.py`)
2. Added provider configuration keys:
   - `DEFAULT_CHAT_PROVIDER`, `DEFAULT_CHAT_MODEL`
   - OpenAI/Anthropic API settings
   - AWS region/credential settings for Bedrock
3. Added runtime dependencies for provider integration:
   - `httpx` (runtime)
   - `boto3`
   - updated `api/uv.lock`
4. Added unit tests:
   - `api/tests/test_provider_registry.py`
   - `api/tests/test_provider_adapters.py`
5. Updated maintainability guidance:
   - `AGENT.md` now includes explicit domain module layout rules
   - added `skills/domain-module-layout/SKILL.md`
6. Verification:
   - `make lint` -> all checks passed
   - `make test` -> `58 passed`
   - `make eval` -> `4/4` cases passed (score `1.0`)

### 2026-02-15 (Unified roadmap phase 12: chat API and continuity flow)

1. Added chat domain package under `api/app/chat`:
   - `api.py`: `/api/v1/chat/*` route layer
   - `service.py`: session/message/pin/save/continue orchestration
   - `context.py`: pinned + retrieval context assembler with source reference packing
   - `errors.py`: chat service error mapping contract
2. Implemented chat API surface:
   - session create/list/get/update
   - message send and stream (`text/event-stream`)
   - pin/unpin engram
   - save session as engram
   - continue session with pinned engram carry-forward
3. Added response metadata for continuity:
   - `used_engram_ids` in assistant replies
   - `source_references` from packed rehydration citations
4. Added compatibility fix for legacy local engrams:
   - ownerless engrams (`owner_user_id IS NULL`) remain readable for authenticated users.
5. Added tests:
   - `api/tests/test_chat_context.py`
   - `api/tests/test_chat_service.py`
   - `api/tests/test_chat_api_integration.py`
6. Verification:
   - `make lint` -> all checks passed
   - `make test` -> `67 passed`
   - `make eval` -> `4/4` cases passed (score `1.0`)

### 2026-02-15 (Unified roadmap phase 13: MCP HTTP stream server)

1. Added MCP domain package under `api/app/mcp`:
   - `api.py`: `/api/v1/mcp/stream` transport endpoint
   - `service.py`: JSON-RPC tool dispatcher and event framing
   - `errors.py`: structured RPC error model
2. Implemented JSON-RPC over SSE behavior:
   - SSE events carry JSON-RPC frames (`result`, `error`, and `mcp.event` notifications)
   - deterministic request `id` correlation in all frames
3. Implemented initial MCP tool handlers:
   - `chat.create_session`, `chat.list_sessions`, `chat.get_session`, `chat.send_message`, `chat.save_as_engram`, `chat.continue_session`
   - `engram.create`, `engram.query`, `engram.rehydrate`, `engram.pin_to_session`
   - `user.get_profile`, `user.list_projects`
4. Enforced auth/visibility parity with API contracts:
   - endpoint requires authenticated session role
   - tool calls reuse chat/engram service/repository authorization paths
5. Added integration tests:
   - `api/tests/test_mcp_api_integration.py` for success/error/auth paths and streaming behavior
6. Verification:
   - `make lint` -> all checks passed
   - `make test` -> `72 passed`
   - `make eval` -> `4/4` cases passed (score `1.0`)

### 2026-02-15 (Unified roadmap phase 14: React chat UI)

1. Initialized `web/` with Vite + React + TypeScript.
2. Implemented full local chat workbench:
   - login workflow using existing server session auth
   - session list/create (provider/model/visibility selectors)
   - streaming transcript UI with retry and error handling
   - pinned engram panel with search, pin/unpin, copy engram ID
   - save-as-engram modal and continue-in-new-chat action
3. Added frontend domain modules for long-run maintainability:
   - `web/src/api/*` for transport contracts
   - `web/src/components/*` for workflow UI modules
   - `web/src/utils/sse.ts` for stream parsing
4. Added frontend tests and quality gates:
   - Vitest + Testing Library setup
   - tests for CSRF token parsing, SSE parsing, and login submit behavior
   - `make web-check` target (`lint`, `test`, `build`)
5. Extended backend chat API for UI parity:
   - `GET /api/v1/chat/sessions/{session_id}/engrams` for pinned engram list state.
6. Verification:
   - `make check` -> all backend checks passed
   - `make web-check` -> all frontend checks passed
   - backend tests: `72 passed`
   - frontend tests: `7 passed`

### 2026-02-15 (UI style-system + backend config debug pass)

1. Migrated frontend styling to shared `styled-components` + Tailwind architecture:
   - added token source: `web/src/styles/theme.ts`
   - added global token/cssvar bridge: `web/src/styles/globalStyles.ts`
   - added reusable shells/cards/typography primitives: `web/src/styles/primitives.ts`
   - replaced legacy class-based CSS usage in `App.tsx` and all primary UI components
2. Added Tailwind pipeline for reusable utility classes:
   - `web/postcss.config.cjs`
   - `web/tailwind.config.ts`
   - simplified `web/src/index.css` to Tailwind layers
3. Added backend dev-mode config logging with safe redaction:
   - `APP_ENV` + `LOG_CONFIG_IN_DEV` settings
   - redacted config snapshot printer on app startup in dev/local environments
4. Added tests for new behavior:
   - `api/tests/test_config_settings.py` for redaction and env gating
   - updated React component tests to render with shared theme provider
5. Verification:
   - `make check` -> all backend checks passed
   - `make web-check` -> all frontend checks passed
   - backend tests: `74 passed`
   - frontend tests: `7 passed`
   - one-time debug sanity check:
     - `cd api && APP_ENV=development LOG_CONFIG_IN_DEV=true uv run python -c "...build_debug_settings_snapshot(...)"` prints a redacted config snapshot

### 2026-02-15 (UI spacing and layout stabilization pass)

1. Restored stable pane spacing and paddings with styled primitives:
   - `TopNavShell`, `GlassPane`, and grid gap defaults now enforce consistent layout without relying on utility-only classes.
2. Added reusable layout primitives for consistent section structure:
   - `PaneHeader`, `SectionDivider`, `FormGrid`, `SplitGrid`, `ScrollColumn`
3. Updated session/chat/pinned panels to use these primitives for predictable spacing on desktop and smaller breakpoints.
4. Fixed workspace growth bug during repeated session creation:
   - root cause: grid row used auto height, so growing session list expanded all panes.
   - fix: constrained workspace row with `grid-template-rows: minmax(0, 1fr)` and `flex: 1` container sizing, with pane-internal scroll.
5. Fixed frontend sign-in redirect handling in dev:
   - root cause: `fetch(..., redirect: 'manual')` can return `status=0` + `opaqueredirect`, which was incorrectly treated as login failure even when server auth succeeded.
   - fix: treat manual redirect responses (`303`/`302`/`307`/`308`, `opaqueredirect`, `status=0`) as success for login/logout flow.
6. Refined base controls:
   - button text centering and form-control line-height/min-height adjustments in global styles.
7. Verification:
   - `make web-check` -> all frontend checks passed
   - frontend tests: `10 passed`

### 2026-02-15 (Bedrock error diagnostics and request mapping pass)

1. Fixed Bedrock provider error handling to surface real AWS error code/message instead of generic credential failure text.
2. Added provider error classification:
   - `ValidationException` -> provider request error (`400`)
   - auth-signature/token credential errors -> provider auth error (`503`)
   - throttling errors -> provider rate-limit error (`429`)
   - other AWS client errors -> provider API error (`502`)
3. Added `ProviderRequestError` and mapped it in chat service to HTTP `400`.
4. Added tests:
   - Bedrock validation-message propagation and credential error mapping
   - chat service mapping for provider request errors
5. Validation:
   - `make check` passed
   - backend tests: `77 passed`

### 2026-02-15 (Unified roadmap phase 15: dockerized acceptance baseline)

1. Dockerized runtime surfaces:
   - added `api/Dockerfile` for uv-locked API container runtime
   - added `web/Dockerfile` for Vite runtime with API proxy behavior
   - expanded `docker-compose.yml` to orchestrate `db`, `api`, `web`, and `acceptance-tests` profile
   - added `.dockerignore` to keep image build contexts lean
2. Added acceptance framework in `acceptance-tests/`:
   - Playwright-BDD feature specs + step definitions
   - `bddgen` generated test pipeline + shared fixture model + failure screenshot capture
   - feature coverage for login handoff, layout drift guardrail, and continuation flow
3. Added workflow commands:
   - `make stack-up`, `make stack-down`, `make stack-logs`
   - `make acceptance-sync`, `make acceptance-bddgen`, `make acceptance-typecheck`, `make acceptance-test`, `make acceptance-test-docker`
4. Added docker/acceptance config hardening:
   - env-driven Vite proxy target (`VITE_API_PROXY_TARGET`)
   - Vite allowed host support (`VITE_ALLOWED_HOSTS`) to enable Playwright access from compose network hostnames
   - docker env matrix expanded in `.env.example`
5. Validation:
   - `make check` passed
   - `make web-check` passed
   - `make acceptance-typecheck` passed
   - `make acceptance-test-docker` passed (`3` scenarios, `13` steps)

### 2026-02-15 (Chat snapshot continuity fix: transcript-aware rehydration)

1. Root cause fixed:
   - chat snapshot engrams were saved with detailed transcript markdown, but rehydration/context assembly used only `abstract` as compact summary.
   - default snapshot abstracts (for example `Snapshot from active chat session.`) led to low-signal context in pinned engram prompts.
2. Backend improvements:
   - rehydration now derives compact summary from transcript content when abstract is generic.
   - rehydration context now includes a `Detailed Notes Excerpt` section.
   - `RehydrationBundle` now carries `detailed_summary_markdown`.
3. Save-as-engram improvements:
   - when saving with a generic snapshot abstract, backend auto-derives a stronger abstract from latest assistant/user message.
4. Frontend improvements:
   - save modal now accepts a `defaultAbstract` and pre-fills it from recent chat content.
5. Added tests:
   - repository helper tests for transcript fallback and assistant-section extraction.
   - chat service test for abstract auto-derivation.
   - integration test for generic-abstract rehydration fallback.
   - frontend test for save modal abstract prefill behavior.
6. Validation:
   - `make check` passed (`81` backend tests + eval `4/4`)
   - `make web-check` passed (`11` frontend tests)

### 2026-02-15 (Unified roadmap phase 15 extension: Bedrock live acceptance coverage)

1. Added a tagged live-provider acceptance scenario:
   - `acceptance-tests/features/bedrock-live.feature` with `@bedrock-live` tag.
   - asserts Bedrock session creation from UI default model and non-empty live assistant response.
2. Added Bedrock step bindings:
   - `acceptance-tests/src/steps/bedrock.steps.ts`.
   - non-deterministic assertion strategy uses response length threshold and explicit provider-error-text guard.
3. Added acceptance env controls:
   - `ACCEPTANCE_BDD_TAGS`, `BEDROCK_LIVE_EXPECTED_MODEL`, `BEDROCK_LIVE_PROMPT`, `BEDROCK_LIVE_MIN_RESPONSE_CHARS`.
   - wired through `acceptance-tests/src/support/env.ts`, `docker-compose.yml`, `.env.example`, and `acceptance-tests/.env.example`.
4. Added workflow commands:
   - `make acceptance-test-bedrock-live`
   - `make acceptance-test-bedrock-live-docker`
5. Validation:
   - `make acceptance-typecheck` passed
   - `make acceptance-test` passed
   - `make acceptance-test-bedrock-live` passed
   - `make acceptance-test-bedrock-live-docker` passed

### 2026-02-15 (Chat UI markdown rendering support)

1. Added markdown rendering in chat transcript bubbles:
   - integrated `react-markdown` with `remark-gfm` and `remark-breaks`.
   - links in markdown now open in a new tab with `rel="noreferrer"`.
2. Added markdown-aware transcript styling:
   - list spacing, blockquote, inline code, and code block styles in shared primitives.
3. Added frontend regression coverage:
   - `ChatPanel` test verifies markdown headings/lists/links are rendered correctly.
4. Validation:
   - `make web-check` passed (`12` frontend tests)

### 2026-02-15 (Live triage continuity acceptance scenario)

1. Added a new non-deterministic triage workflow feature:
   - `acceptance-tests/features/triage-live.feature` tagged with `@triage-live @bedrock-live`.
   - covers live triage memo generation, save-as-engram, continue-in-new-chat, pinning, and continuity handoff brief generation.
2. Added triage step bindings:
   - `acceptance-tests/src/steps/triage.steps.ts`.
   - asserts only structural continuity signals (keywords + length thresholds), not exact deterministic wording.
3. Added reusable live-response helper:
   - `acceptance-tests/src/support/chat.ts` for robust assistant text extraction compatible with markdown-rendered messages.
4. Expanded execution commands:
   - `make acceptance-test-triage-live`
   - `make acceptance-test-triage-live-docker`
   - `npm run test:triage-live`
5. Validation:
   - `make acceptance-typecheck` passed
   - `npm run test:triage-live -- --list` confirmed triage scenario generation and selection

### 2026-02-15 (Frontend save-as-engram cutoff fix)

1. Fixed frontend truncation of default save abstract:
   - removed the `320`-character truncation from chat-to-engram default abstract generation.
   - extracted logic into `web/src/utils/chat.ts` for maintainable reuse.
2. Added regression tests:
   - `web/src/utils/chat.test.ts` verifies full latest assistant message is preserved and fallback behavior to user message.
3. Validation:
   - `make web-check` passed

### 2026-02-15 (Streaming parser and visual refresh pass)

1. Restored robust SSE parsing for chat streaming:
   - updated `web/src/utils/sse.ts` to handle both `\n\n` and `\r\n\r\n` framing via normalization.
   - added end-of-stream buffer flush behavior for partial final frames.
2. Added regression coverage:
   - `web/src/utils/sse.test.ts` now includes CRLF + chunk-boundary frame parsing.
3. Refreshed UI away from green:
   - updated theme tokens in `web/src/styles/theme.ts`.
   - removed green background accents in `web/src/styles/globalStyles.ts`.
   - switched assistant bubble/notice/surface accents to blue-amber family in `web/src/styles/primitives.ts`.
4. Improved markdown readability:
   - richer heading, table, blockquote, and horizontal-rule styles in transcript message renderer.
5. Validation:
   - `make web-check` passed

### 2026-02-15 (Dark theme mode and toggle support)

1. Added dual theme definitions:
   - `web/src/styles/theme.ts` now exports `lightTheme`, `darkTheme`, and `appThemes`.
   - both themes include complete color tokens for panels, bubbles, markdown surfaces, and modal layers.
2. Added persistent theme-mode state:
   - `web/src/styles/themeMode.tsx` introduces `ThemeModeProvider`.
   - `web/src/styles/useThemeMode.ts` provides theme-mode hook access.
   - `web/src/styles/themeModeUtils.ts` centralizes mode normalization/initial-resolution logic.
   - mode is stored in `localStorage` and restored on load.
3. Wired runtime theme switching:
   - `web/src/main.tsx` + `web/src/ThemedApp.tsx` now select `ThemeProvider` theme from active mode.
   - `web/src/App.tsx` top-nav toggle switches between Light/Dark themes.
4. Updated style layers for mode-aware rendering:
   - `web/src/styles/globalStyles.ts` adds CSS variables for surfaces/background glows/markdown UI.
   - `web/src/styles/primitives.ts` consumes those variables for cards, panes, notices, bubbles, tables, blockquotes, and modals.
5. Added tests:
   - `web/src/styles/themeMode.test.ts` validates mode normalization and initial mode resolution logic.
6. Validation:
   - `make web-check` passed
   - `make acceptance-test` passed

### 2026-02-15 (Dark scrollbar contrast and sidebar scroll-layout pass)

1. Improved themed scrollbar styling:
   - added light/dark scrollbar tokens in `web/src/styles/theme.ts`.
   - applied global scrollbar styling in `web/src/styles/globalStyles.ts` for both WebKit and Firefox (`scrollbar-color`).
2. Improved session-sidebar ergonomics:
   - split sidebar into two independent scroll zones in `web/src/components/SessionSidebar.tsx`:
     - create-session panel (`session-create-panel`) remains scrollable.
     - session list panel (`session-list-panel`) gets larger relative height and independent scrolling.
3. Validation:
   - `make web-check` passed

### 2026-02-15 (Session sidebar sticky controls and active-state UX pass)

1. Made creator controls permanent and sticky:
   - `web/src/components/SessionSidebar.tsx` now keeps the create button in a sticky footer area.
   - create panel remains independently scrollable for long forms.
2. Added optional creator collapse toggle:
   - `Hide Creator` / `Show Creator` toggle in sidebar create header.
3. Added explicit session-list label:
   - bottom panel now includes sticky section heading `Previous Sessions`.
4. Improved active session highlighting:
   - stronger active card colors via theme tokens.
   - selected session now sets `aria-current="true"` for accessibility/testing hooks.
5. Added regression tests:
   - `web/src/components/SessionSidebar.test.tsx` validates list label, creator toggle, and selected-session marker.
6. Validation:
   - `make web-check` passed

### 2026-02-15 (Unified roadmap merge and next-phase design pass)

1. Merged planning sources:
   - combined historical baseline and chat/MCP expansion roadmap into one canonical unified `Plan.md`.
2. Added concrete next implementation phases in unified plan:
   - phase 16: MCP developer tooling and typed clients.
   - phase 17: document ingestion and RAG-ready retrieval.
   - phase 18: memory lifecycle policies (autosave/retention/consolidation).
   - phase 19: collaboration and sharing model.
   - phase 20+: security hardening, observability, release automation, evalops.
3. Updated README references and file maps:
   - switched roadmap references from dual-plan status to unified-plan status.
   - refreshed repository/file guides with current web theme-mode modules and acceptance artifacts.

### 2026-02-15 (Architecture playbook documentation pass)

1. Added architecture playbook with mermaid diagrams and multi-model continuity runbooks.
2. Added architecture guide:
   - created `docs/architecture-playbook.md` with dual-layer narrative (newcomer + technical deep dive).
   - added system/call-flow/data-model Mermaid diagrams and cross-model continuity visuals.
3. Added multi-model runbooks:
   - incident triage continuity flow (OpenAI -> Bedrock -> Anthropic).
   - product strategy research flow.
   - support escalation handoff flow.
4. Added explicit call-surface mapping:
   - REST, MCP JSON-RPC/SSE, and UI button-to-endpoint mappings.
   - documented where `used_engram_ids` and source references appear.
5. Added interface impact statement:
   - no new runtime APIs.
   - no schema or type changes.
   - documentation clarifies existing contracts only.
6. Updated README indexing:
   - added top-level docs pointer.
   - added file-map entry for architecture playbook.

### 2026-02-15 (PlantUML architecture/workflow map pass)

1. Added detailed PlantUML diagram:
   - created `docs/architecture-workflows.puml`.
   - diagram includes architecture layers and four core workflows (stream chat, save-as-engram, continue-in-new-chat, MCP tool flow).
2. Added color-coded flow paths:
   - REST/UI flow.
   - MCP JSON-RPC/SSE flow.
   - provider generation/stream flow.
   - persistence/retrieval continuity flow.
3. Updated README docs/file map:
   - added docs pointer for PlantUML map.
   - added file guide entry for `docs/architecture-workflows.puml`.

### 2026-02-15 (PlantUML render portability pass)

1. Added Docker-based PlantUML renderer script:
   - created `docs/render-plantuml.sh`.
   - renders all `docs/*.puml` to `docs/rendered/` as `svg` or `png`.
2. Added Makefile shortcuts:
   - `make diagram-render`
   - `make diagram-render-png`
3. Added troubleshooting guidance:
   - documented fallback for local Graphviz path errors such as `/opt/local/bin/dot` not found.

### 2026-02-15 (PlantUML use-case diagram pass)

1. Added detailed use-case diagram:
   - created `docs/model-switch-engram-usecases.puml`.
   - covers actors, model switching, save/pin/continue lifecycle, query/rehydrate, and MCP tool usage.
2. Added relationship semantics:
   - explicit `<<include>>` and `<<extend>>` relations for session, continuity, and metadata-inspection paths.
3. Updated README docs/file map:
   - added docs pointer for use-case map.
   - added file guide entry for `docs/model-switch-engram-usecases.puml`.
4. Updated render automation:
   - `make diagram-render` and `make diagram-render-png` now target both PlantUML diagram sources explicitly.

### 2026-02-15 (Fresh-reset session FK fix pass)

1. Fixed stale-session user mapping after DB reset:
   - API/UI auth now reconciles session user payload against current `users` table by username.
   - stale cookie `user_id` values are auto-corrected to the active DB user record before chat routes run.
2. Hardened demo user bootstrap for clean starts:
   - moved local demo user bootstrap into `db/init/001_schema.sql` for fresh setup.
   - removed startup-time user seeding from API lifespan path.
   - login now authenticates against DB users only (no env-only fallback identity).
3. Added reset scripts for clean local starts:
   - `make db-reset` (drop volumes + clear local runtime files + start DB)
   - `make stack-reset` (drop volumes + clear local runtime files + start DB/API/Web)
4. Added regression test:
   - integration test validates chat session creation still works when session cookie has stale `user_id` after DB identity change.
5. Validation:
   - `make test` passed (`82` tests)
   - `make lint` passed
   - `make format-check` passed

### 2026-02-15 (Phase 17 - ingestion and blended retrieval pass)

1. Added ingestion domain and schema:
   - new `documents` and `document_chunks` tables with owner/project/visibility indexes and vector index for chunk retrieval.
   - added `api/app/ingestion/*` package (models, chunking, repository, service, API router).
2. Added embedding abstraction for RAG-ready storage:
   - new `api/app/embeddings/*` package with local deterministic provider and optional OpenAI embeddings provider.
   - repository embedding calls now route through abstraction with provider metadata persisted in `embedding_model`.
3. Added query blending behavior:
   - chat context now merges engram snapshots with document chunk retrieval.
   - chat responses/stream metadata now include `used_document_chunk_ids` alongside `used_engram_ids`.
4. Added React ingestion UX:
   - `DocumentIngestionPanel` for file/text ingest with status/error feedback.
   - project-scoped recent document list surfaced in the right rail.
5. Added tests:
   - API: chunking determinism, embedding fallback, ingestion service, ingestion integration lifecycle, chat-context chunk blend assertions.
   - Web: ingestion panel interaction tests.
6. Validation:
   - `cd api && uv run ruff check app tests` passed.
   - `cd api && uv run pytest -q` passed (`99` tests).
   - `cd web && npm run lint` passed.
   - `cd web && npm run test` passed (`36` tests).
   - `cd web && npm run build` passed.

### 2026-02-16 (Phase 17 follow-up - pinned document continuity pass)

1. Added pinned-document chat APIs:
   - `GET /api/v1/chat/sessions/{session_id}/documents`
   - `POST /api/v1/chat/sessions/{session_id}/documents/pin`
   - `DELETE /api/v1/chat/sessions/{session_id}/documents/{document_id}`
2. Upgraded chat context assembly:
   - session-pinned documents are queried first and merged with normal document retrieval.
   - context now includes a dedicated `Pinned Document Context` block when pinned chunks are available.
3. Added continuation carry-forward:
   - pinned documents now carry into newly continued sessions (same behavior as pinned engrams).
4. Added UI support:
   - recent document cards now support `Pin to Chat` and `Unpin`.
   - pinned documents are visually highlighted and tagged in the ingestion panel.
   - removed top-nav docs hide toggle so the document panel is always visible.
5. Added regression tests:
   - API integration for pin/list/unpin documents and continuation carry-forward.
   - repository integration for document pin lifecycle.
   - context and service unit tests for pinned document usage and copy-on-continue behavior.
   - web interaction tests for document pin/unpin controls.
6. Validation:
   - `make check` passed (`100` API tests + eval suite).
   - `make web-check` passed (`38` web tests + build).

### 2026-02-16 (Phase 17 follow-up - multi-document pin context pass)

1. Upgraded pinned-document context assembly:
   - pinned document retrieval now guarantees one representative chunk per pinned document.
   - this prevents a single high-similarity document from crowding out other pinned docs in the same session.
2. Expanded integration coverage:
   - chat API integration now validates pinning two documents to one session, using both, carrying both into continuation, and unpinning both.
3. Expanded context coverage:
   - added unit test to ensure all pinned documents appear in `used_document_chunk_ids` and document source references.
4. Validation:
   - `make check` passed (`103` API tests + eval suite).

### 2026-02-16 (Phase 16 follow-up - MCP workflow expansion pass)

1. Expanded MCP chat workflow coverage:
   - added tool methods for message history and pin lifecycle:
     - `chat.list_messages`
     - `chat.list_pinned_engrams`, `chat.pin_engram`, `chat.unpin_engram`
     - `chat.list_pinned_documents`, `chat.pin_document`, `chat.unpin_document`
     - `chat.list_project_documents`
2. Fixed MCP non-stream `chat.send_message` path:
   - `tools/call` with `stream: false` now resolves through a direct success result instead of method fallback errors.
3. Added integration tests for new MCP flows:
   - tool catalog assertions include all newly exposed methods.
   - end-to-end MCP test validates: create session, list docs, pin 2 docs, list pinned docs, send message, and unpin.
4. Added admin-console runbook:
   - documented manual validation sequence for admin login, ingest, multi-pin, continuity, and MCP parity checks.
5. Validation:
   - `make check` passed (`104` API tests + eval suite).

### 2026-02-16 (Observability pass - debug traces + optional Langfuse)

1. Added structured chat debug trace payloads:
   - includes embedding timing, LLM call timing, token usage, input/output snapshots, and provider request previews.
   - debug trace now appears in chat send responses and stream completion payloads.
2. Added embedding observability hooks:
   - `embed` and `embed_many` now record provider, duration, text size, batch size, and fallback usage under chat context.
3. Added optional Langfuse publisher:
   - configured by `LANGFUSE_ENABLED`, `LANGFUSE_HOST`, `LANGFUSE_PUBLIC_KEY`, `LANGFUSE_SECRET_KEY`.
   - failures are non-blocking and never interrupt chat flow.
4. Added UI debug panel:
   - displays response-level timing/tokens and full JSON trace for inspection.
5. Added tests:
   - embedding observability unit tests.
   - chat service response test now validates debug trace presence and token data.
6. Validation:
   - `make check` passed (`107` API tests + eval suite).
   - `make web-check` passed (`39` web tests + build).

### 2026-02-16 (Observability hardening - Langfuse v3 API + debug UX)

1. Switched Langfuse integration to latest API only:
   - uses `start_as_current_observation` and `update_current_trace`.
   - legacy `trace()/generation()` branch removed for simpler long-term maintenance.
2. Added explicit telemetry diagnostics:
   - startup logs now clearly indicate disabled, missing dependency, missing keys, or incompatible client shape.
   - publish logs now include success and exception traces for faster debugging in dev.
3. Improved debug trace UI usability:
   - debug trace panel now uses bounded scroll containers so large JSON payloads are inspectable without breaking chat layout.
4. Expanded tests:
   - added telemetry publisher tests for success, failure, missing credentials, and incompatible client behaviors.
5. Validation:
   - `make check` passed (`111` API tests + eval suite).
   - `make web-check` passed (`39` web tests + build).

### 2026-02-18 (Phase 18 - memory lifecycle policy controls pass)

1. Added session lifecycle schema controls:
   - `autosave_strategy`, `autosave_interval_minutes`, `autosave_min_messages`.
   - `retention_days`, `retention_max_snapshots`.
   - idempotent schema alter support and value constraints in `db/init/001_schema.sql`.
2. Added lifecycle policy contracts and pure-policy module:
   - new models for policy read/update and timeline events in `api/app/models.py`.
   - new `api/app/chat/lifecycle_policy.py` for normalization, trigger guards, retention selectors, and event typing.
3. Added chat lifecycle APIs and service wiring:
   - `GET/PATCH /api/v1/chat/sessions/{session_id}/lifecycle-policy`
   - `GET /api/v1/chat/sessions/{session_id}/timeline`
   - autosave snapshot + retention pruning maintenance on assistant completion (sync + stream).
4. Added MCP lifecycle tool coverage:
   - `chat.get_lifecycle_policy`
   - `chat.update_lifecycle_policy`
   - `chat.list_timeline`
5. Added UI lifecycle controls and timeline visibility:
   - session creator now captures autosave strategy and retention settings.
   - chat panel renders lifecycle timeline entries for newcomers/operators.
6. Added tests:
   - `api/tests/test_chat_lifecycle_policy.py`
   - lifecycle policy/repository/service/API/MCP coverage expansions.
   - web updates for session sidebar and chat panel lifecycle rendering checks.
7. Validation:
   - `make check` passed (`123` API tests + eval suite).
   - `make web-check` passed (`39` web tests + build).

### 2026-02-18 (Phase 18 follow-up - autosave configuration trigger tests)

1. Expanded lifecycle integration tests to validate autosave behavior by policy configuration:
   - `off`: no autosave snapshots are created after chat messages.
   - `message_count` (threshold 2): autosave triggers on threshold and not before.
   - `interval` (60 minutes): first autosave is created, immediate second message does not create another snapshot.
2. Validation:
   - `make check` passed (`126` API tests + eval suite).

### 2026-02-18 (Acceptance follow-up - mocked autosave lifecycle scenario)

1. Added deterministic mocked acceptance scenario:
   - `acceptance-tests/features/lifecycle-autosave-mock.feature`
   - validates autosave `message_count` threshold trigger and autosave `off` no-trigger behavior.
2. Added stateful mock step bindings:
   - `acceptance-tests/src/steps/lifecycle-mock.steps.ts`
   - intercepts chat/session endpoints and emits deterministic SSE stream events.
3. Added runner commands:
   - `make acceptance-test-mock`
   - `cd acceptance-tests && npm run test:mock`
4. Validation:
   - `make acceptance-bddgen` passed
   - `make acceptance-typecheck` passed
   - `make acceptance-test-mock` passed

### 2026-02-19 (Phase 29 - optional auto-metadata enrichment for engram persistence)

1. Added deterministic fill-empty-only metadata enrichment domain:
   - `api/app/engram_enrichment/models.py`
   - `api/app/engram_enrichment/service.py`
   - derives `abstract`, `keywords`, and `tags` only when caller values are empty.
2. Centralized enrichment in repository create flow so all create paths share behavior:
   - REST `POST /api/v1/engrams`
   - chat save/autosave persistence
   - MCP `engram.create`
   - CLI/agent/consolidation create paths
3. Added MCP conversation-first tool:
   - `engram.create_from_conversation`
   - returns created engram plus enrichment report metadata.
4. Added traceability metadata:
   - `engram_json.auto_metadata` now stores derivation report and origin/source details.
5. Added/updated tests:
   - `api/tests/test_engram_enrichment.py`
   - `api/tests/test_api_integration.py`
   - `api/tests/test_chat_api_integration.py`
   - `api/tests/test_mcp_api_integration.py`
   - `acceptance-tests/features/engram-auto-metadata-mock.feature`
   - `acceptance-tests/src/steps/engram-auto-metadata-mock.steps.ts`
6. Validation:
   - `cd api && uv run ruff check app tests/test_api_integration.py tests/test_engram_enrichment.py` passed.
   - `cd api && uv run pytest -q tests/test_engram_enrichment.py tests/test_api_integration.py tests/test_chat_api_integration.py tests/test_mcp_api_integration.py tests/test_api_unit.py tests/test_chat_service.py tests/test_cli.py tests/test_consolidation.py` passed (`57 passed`).
   - `cd acceptance-tests && npm run bdd:gen` passed.
   - `cd acceptance-tests && npm run typecheck` passed.
   - `cd acceptance-tests && npm run test:mock` passed (`4 passed`).

### 2026-02-19 (Phase 30 - MCP PAT auth with scoped read/write authorization)

1. Added MCP token domain and storage:
   - `api/app/mcp_tokens/models.py`
   - `api/app/mcp_tokens/repository.py`
   - `api/app/mcp_tokens/service.py`
   - `db/init/001_schema.sql` (`mcp_tokens` table + indexes)
   - `api/app/config.py` and `.env.example` (`MCP_TOKEN_PEPPER`)
2. Added bearer-token MCP auth resolution with session fallback:
   - `api/app/mcp/auth.py`
   - `api/app/mcp/api.py`
3. Added scoped MCP authorization guard:
   - `api/app/mcp/service.py` now enforces scope (`read`/`write`), optional `allowed_tools`, and optional `allowed_project_ids`.
4. Added admin token lifecycle APIs/UI:
   - `POST /api/v1/mcp/tokens`
   - `GET /api/v1/mcp/tokens`
   - `POST /api/v1/mcp/tokens/{token_id}/revoke`
   - admin console token create/list/revoke panel in `api/app/templates/admin.html`
   - admin-only React token manager panel in `web/src/components/AdminMcpTokenPanel.tsx`
   - panel now loads available MCP tools/projects and applies restrictions via selectable chips
5. Added typed contracts and client support:
   - `api/app/models.py` token request/response models
   - `api/app/mcp/client.py` optional `bearer_token` support
6. Added automated test coverage:
   - `api/tests/test_mcp_token_service.py`
   - `api/tests/test_mcp_token_api_integration.py`
   - `api/tests/test_admin_mcp_tokens_ui.py`
   - extended `api/tests/test_mcp_api_integration.py` for bearer scope/allowlist/project/revoked/expired paths
   - extended `api/tests/test_mcp_client.py` for bearer header propagation
   - `web/src/components/AdminMcpTokenPanel.test.tsx`
   - `acceptance-tests/features/mcp-token-auth-mock.feature`
   - `acceptance-tests/src/steps/mcp-token-auth-mock.steps.ts`
   - `acceptance-tests/features/admin-mcp-token-ui.feature`
   - `acceptance-tests/src/steps/admin-mcp-token-ui.steps.ts`
7. Validation:
   - `make check` passed (`157 passed` + eval pass).
   - `make acceptance-bddgen` passed.
   - `make acceptance-typecheck` passed.
   - `make acceptance-test-mock` passed (`8 passed`).

### 2026-02-19 (MCP OAuth compatibility for VS Code dynamic registration)

1. Added OAuth-compatible discovery and registration endpoints:
   - `/.well-known/oauth-authorization-server`
   - `/.well-known/openid-configuration`
   - `/.well-known/oauth-protected-resource` (plus path-scoped variant)
   - `POST /oauth/register`
   - `GET /oauth/authorize`
   - `POST /oauth/token`
2. Implemented OAuth authorization-code + PKCE flow backed by DB persistence:
   - added `oauth_clients` and `oauth_authorization_codes` schema in `db/init/001_schema.sql`
   - added OAuth domain modules under `api/app/oauth/` (api/repository/service/models)
   - token exchange now issues short-lived MCP bearer tokens that reuse existing MCP authz controls
3. Added login continuation support for OAuth browser redirects:
   - `/login` now accepts safe `next` paths and preserves them through sign-in.
4. Added standards-based MCP auth challenge hints:
   - `WWW-Authenticate` on MCP 401 responses now includes `resource_metadata` when OAuth is enabled.
5. Added config surface for OAuth operations:
   - `.env.example` and `api/app/config.py` now include `OAUTH_ENABLED`, `OAUTH_ISSUER_URL`, `OAUTH_ACCESS_TOKEN_TTL_SECONDS`, `OAUTH_AUTHORIZATION_CODE_TTL_SECONDS`, and `OAUTH_CLIENT_SECRET_PEPPER`.
6. Added automated coverage:
   - `api/tests/test_mcp_oauth_integration.py`
   - `api/tests/test_oauth_service.py`
   - updated `api/tests/test_config_settings.py` for OAuth secret redaction
7. Validation:
   - `make check` passed (`157 passed` + eval pass).
   - focused OAuth/MCP integration suite passed (`36 passed`).

### 2026-02-19 (Phase 31 progress checkpoint - projects, memory admin, MCP organization)

1. Added project/default-project architecture and persistence semantics:
   - schema updates in `db/init/001_schema.sql` (`projects`, user default project FK, soft-delete metadata, collection tables, idempotent backfill).
   - ensured chat/ingestion/session creation paths create/resolve project records consistently.
2. Added maintainable backend module boundaries for long-term growth:
   - project domain under `api/app/projects/`
   - memory administration domain under `api/app/memory_admin/`
   - both mounted from `api/app/main.py` with actor-role checks.
3. Implemented project APIs:
   - `GET /api/v1/projects`
   - `POST /api/v1/projects`
   - `GET /api/v1/projects/default`
   - `PATCH /api/v1/projects/default`
4. Implemented admin memory management APIs (`/api/v1/admin/memory/*`):
   - session list/delete/restore
   - engram list/get/update/move/delete/restore
   - collection list/create/update/delete/add-items/remove-item
5. Hardened engram APIs:
   - `/api/v1/engrams*` now requires authenticated actor context.
   - `POST /api/v1/engrams` now supports default-project fallback when `project_id` is omitted and returns `resolved_project_id`/`used_default_project`.
6. Implemented MCP organization tools and policy controls:
   - project tools (`project_list`, `project_create`, `project_get_default`, `project_set_default`)
   - engram tools (`engram_list`, `engram_get`, `engram_update`, `engram_move_project`, `engram_delete`, `engram_restore`)
   - collection tools (`engram_collection_*`)
   - session tools (`chat_delete_session`, `chat_restore_session`)
   - owner/admin and read/write/project policy enforcement integrated with existing MCP token controls.
7. Implemented web app routing + admin management UX:
   - routes: `/` (workspace), `/admin/memory` (admin memory page)
   - project default controls in `SessionSidebar`
   - admin page table/filter/dialog workflows for session/engram/collection operations.
8. Added/updated tests for Phase 31 behavior:
   - backend integration: `api/tests/test_projects_api_integration.py`, `api/tests/test_memory_admin_api_integration.py`, `api/tests/test_schema_backfill_projects.py`
   - MCP integration expansions in `api/tests/test_mcp_api_integration.py`
   - web unit/integration: `web/src/api/projects.test.ts`, `web/src/api/memoryAdmin.test.ts`, `web/src/components/AdminMemoryPage.test.tsx`
   - acceptance mock coverage: `acceptance-tests/features/phase31-memory-admin-mock.feature`, `acceptance-tests/src/steps/phase31-memory-admin-mock.steps.ts`
9. Validation status at checkpoint:
   - `make -C /Users/mukundhan/Projects/engram check` passed.
   - `make -C /Users/mukundhan/Projects/engram web-check` passed.
   - `make -C /Users/mukundhan/Projects/engram acceptance-bddgen` passed.
   - `make -C /Users/mukundhan/Projects/engram acceptance-typecheck` passed.

### 2026-02-19 (Makefile UX refresh - colored help + detailed command output)

1. Updated `/Users/mukundhan/Projects/engram/Makefile` to use a richer command UX modeled after production-grade Docker workflows:
   - added `.DEFAULT_GOAL := help`
   - added shared config vars (`DOCKER_COMPOSE_DIR`, `DOCKER_COMPOSE_FILE`, `DOCKER_COMPOSE`, `ACCEPTANCE_DIR`)
   - added terminal color constants (`SUCCESS`, `PROGRESS`, `ERROR`, `INFO`, `NC`)
2. Enhanced help surface:
   - grouped command catalog with colorized sections
   - quick-start command recommendations
   - new `print-config` target to show effective compose and path settings
3. Improved command readability:
   - added progress/success status lines across database, stack, acceptance, API, web, and diagram targets
   - standardized Docker compose execution through `$(DOCKER_COMPOSE)`
4. Validation:
   - `make -C /Users/mukundhan/Projects/engram help` passed
   - `make -C /Users/mukundhan/Projects/engram print-config` passed

### 2026-02-19 (MCP token panel fix - JSON/SSE compatibility + acceptance hardening)

1. Fixed admin MCP token options loading in React:
   - `web/src/api/mcpClient.ts` now explicitly requests `Accept: text/event-stream, application/json`.
   - Added JSON fallback parsing when `/api/v1/mcp/stream` returns `application/json` (streamable HTTP request/response mode).
   - This prevents `MCP stream returned no jsonrpc frames` when the server chooses JSON over SSE.
2. Added frontend test coverage:
   - `web/src/api/mcpClient.test.ts` now verifies JSON fallback parsing and request Accept header behavior.
3. Hardened acceptance MCP flow parsing:
   - updated `acceptance-tests/src/steps/engram-auto-metadata-mock.steps.ts`
   - updated `acceptance-tests/src/steps/phase31-memory-admin-mock.steps.ts`
   - both now parse MCP responses for either SSE or JSON-RPC JSON payloads.
4. Removed race in admin token options acceptance assertion:
   - updated `acceptance-tests/src/steps/admin-mcp-token-ui.steps.ts` to poll until tool/project options are populated before strict assertions.
5. Dev-run consistency:
   - `make web` now defaults to port `5173` to match local acceptance/test workflow and avoid localhost port confusion.
6. Validation:
   - `make -C /Users/mukundhan/Projects/engram check` passed (`176` tests + eval pass).
   - `make -C /Users/mukundhan/Projects/engram web-check` passed (`52` web tests + build).
   - `make -C /Users/mukundhan/Projects/engram acceptance-bddgen` passed.
   - `make -C /Users/mukundhan/Projects/engram acceptance-typecheck` passed.
   - `make -C /Users/mukundhan/Projects/engram acceptance-test-mock` passed (`10` scenarios).

### 2026-02-19 (Refactor checkpoint - MCP service phase 1)

1. Refactored MCP catalog boundaries:
   - extracted MCP tool catalog/constants/helpers from `api/app/mcp/service.py` to `api/app/mcp/catalog.py`.
   - `McpService` now reads catalog data via `build_tool_catalog()`.
2. Refactored MCP dispatch architecture:
   - split monolithic `_dispatch_tool` into domain dispatchers:
     - `_dispatch_chat_tool`
     - `_dispatch_engram_tool`
     - `_dispatch_project_tool`
     - `_dispatch_user_tool`
   - `_dispatch_tool` is now a thin router.
3. Added shared authorization/access helpers:
   - `_require_session_access`
   - `_require_engram_access`
   - `_require_collection_access`
   - extracted `_resolve_project_from_session` / `_resolve_project_from_engram` for `_project_id_for_tool`.
4. Reduced stream authorization duplication:
   - extracted `_authorize_tool_call` and used it in `stream_call` direct and `tools/call` streaming paths.
5. Added backend tests for refactor safety:
   - new `api/tests/test_mcp_tool_catalog.py`
   - new `api/tests/test_mcp_service_unit.py`
   - expanded `api/tests/test_mcp_api_integration.py` with `tools/call` round-trip checks for `project.list`, `engram.list`, and unknown tool handling.
6. Validation:
   - `uv run --project api pytest api/tests/test_mcp_tool_catalog.py api/tests/test_mcp_service_unit.py -q` passed (`9 passed`).
   - `uv run --project api pytest api/tests/test_mcp_api_integration.py::test_mcp_tools_call_project_and_engram_round_trip_plus_unknown_tool -q` passed.
   - `uv run --project api pytest api/tests/test_mcp_api_integration.py -q` passed (`27 passed`).
   - `make -C /Users/mukundhan/Projects/engram test` passed (`189 passed`).
   - `make -C /Users/mukundhan/Projects/engram check` passed (lint + format + tests + eval).
7. CodeScene MCP health checks:
   - `code_health_review` on `api/app/mcp/service.py`: score `4.54`.
   - `pre_commit_code_health_safeguard` on repo: `quality_gates=passed`, `api/app/mcp/service.py` verdict `improved`.

### Next Immediate Steps (One By One)

1. Phase 31 closeout: update `AGENT.md` + skills with final project-default/soft-delete/collection invariants and MCP organization contracts.
2. Phase 31 closeout: run/record full acceptance mock execution (`make acceptance-test-mock`) with new admin-memory scenarios.
3. Phase 18 follow-up: add explicit consolidation merge/grouping event semantics in timeline rendering.
4. Phase 19 design: implement project membership and scoped sharing/revocation flows with audit trails.
5. Phase 20 security gate: OIDC integration + distributed rate-limit strategy + production auth hardening tests.

### 2026-02-22 (Phase tracking kickoff - export/import stash + audit remediation)

1. Added new roadmap phase definition in `Plan.md`:
   - **Phase 33**: portable memory export/import stash workflow.
   - scope decision locked as Option C (full project default + selective collection export).
   - embedding payload policy locked (exclude by default, optional include flag).
   - authorization policy locked to project owner + admin.
2. Added new roadmap phase definition in `Plan.md`:
   - **Phase 34**: security audit remediation program for prioritized hardening.
3. Updated `checkpoint.md` current summary and phase timeline to include:
   - Phase 33 (In Progress)
   - Phase 34 (Planned)
4. Updated `todo.md` near-term queue with explicit Phase 33/34 execution items to ensure follow-up pickup in later passes.

### 2026-02-22 (Phase 33 progress - project import path + quality gate)

1. Extended Phase 33 export module with import support in `api/app/export/`:
   - added `POST /api/v1/projects/{project_id}/import` endpoint accepting JSON or ZIP export bundle uploads.
   - added import conflict policy handling (`skip`, `overwrite`, `rename`).
2. Kept shared API data contracts in `api/app/models.py`:
   - `ProjectImportConflictPolicy`
   - `ProjectImportResponse`.
3. Added/updated tests:
   - `api/tests/test_export_api.py`
   - `api/tests/test_export_api_integration.py`
   - validated export subset behavior, ZIP export, non-owner denial, JSON import, and ZIP import rename behavior.
4. Validation results:
   - `uv run --project api ruff check` (targeted files) passed.
   - `uv run --project api pytest -q api/tests/test_export_api.py api/tests/test_export_api_integration.py` passed (`7 passed`).
5. CodeScene health gate:
   - `mcp_analyze_change_set` against `origin/main` -> `quality_gates=passed`.
   - findings flagged complexity/duplication in `api/app/export/service.py`; accepted for this slice and queued for follow-up refactor pass.
