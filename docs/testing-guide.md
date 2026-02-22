# Engram Vault - Testing Guide

> Complete testing reference for backend, frontend, and acceptance tests.
> See [AGENT.md](../AGENT.md) section 9 for testing policy. See [User Workflows](user-workflows.md) for manual validation.

---

## Quick Commands

```bash
make test              # Run all backend tests
make test-unit         # Unit tests only
make test-integration  # Integration tests only
make lint              # Lint checks (ruff)
make format-check      # Format verification
make check             # All quality gates (lint + format + tests + eval)
make eval              # Memory quality evaluations
make web-check         # Frontend lint + test + build
```

---

## Backend Test Suite

Tests live under `api/tests/`:

| File | Coverage |
|------|----------|
| `test_embedding.py` | Deterministic embedding behavior |
| `test_repository_helpers.py` | Retrieval text and vector literal helpers |
| `test_api_unit.py` | Endpoint behavior with repository function mocking |
| `test_api_integration.py` | End-to-end API roundtrip + sources endpoint checks |
| `test_ui_auth.py` | Login/logout/session-protected UI + CSRF + rate-limit + audit |
| `test_user_rbac.py` | User management and role-based access control |
| `test_agent_workflow.py` | LangGraph checkpoint/resume + auto-persist + snapshot |
| `test_eval_harness.py` | Evaluation harness scenario pass/fail |
| `test_cli.py` | Upload/search/rehydrate CLI commands |
| `test_consolidation.py` | Consolidation snapshot generation and safety |
| `test_chat_lifecycle_policy.py` | Autosave normalization/trigger and retention pruning |
| `test_chat_repository.py` | Chat session/message, pinning, lifecycle persistence |
| `test_chat_api_integration.py` | End-to-end chat API lifecycle/continuity + policy/timeline |
| `test_chat_service.py` | Chat orchestration with autosave and retention |
| `test_chat_context.py` | Context assembly merge/dedupe behavior |
| `test_mcp_api_integration.py` | MCP tool contract coverage including lifecycle tools |
| `test_mcp_client.py` | Typed Python MCP client parsing/auth/transport |
| `test_mcp_token_service.py` | Token generation/hash/parse/expiry/revocation |
| `test_mcp_token_api_integration.py` | MCP token lifecycle REST authz (admin vs non-admin) |
| `test_admin_mcp_tokens_ui.py` | Admin UI MCP token create/revoke workflows |
| `test_mcp_oauth_integration.py` | OAuth metadata, registration, authorize/token exchange |
| `test_oauth_service.py` | OAuth scope + PKCE helper unit tests |
| `test_engram_enrichment.py` | Deterministic metadata derivation + non-overwrite contract |
| `test_engram_visibility.py` | Owner/project scope filtering |
| `test_provider_registry.py` | Provider registry construction and adapter selection |
| `test_provider_adapters.py` | Adapter normalization and error-path tests |
| `test_embeddings_service.py` | Embedding provider fallback behavior |
| `test_ingestion_chunking.py` | Deterministic chunking/hash/document-id behavior |
| `test_ingestion_service.py` | Ingestion validation and persistence orchestration |
| `test_ingestion_api_integration.py` | Authenticated ingestion + retrieval lifecycle |
| `test_config_settings.py` | Settings debug logging and secret redaction |
| `conftest.py` | DB fixture, schema bootstrap, and cleanup |

Notes:
- Integration tests are marked with `@pytest.mark.integration` and configured in `api/pyproject.toml`.
- If the DB is unavailable, integration tests are skipped with a clear reason.
- `make eval` runs local memory-quality scenarios and exits non-zero if any case fails.

---

## Frontend Test Suite

Tests live under `web/src/**/*.test.ts(x)`:

| File | Coverage |
|------|----------|
| `src/api/auth.test.ts` | Auth API client |
| `src/styles/themeMode.test.ts` | Theme mode normalization |
| `src/utils/chat.test.ts` | Save-as-engram abstract derivation |
| `src/utils/sse.test.ts` | SSE parser (CRLF + chunk boundaries) |
| `src/components/LoginView.test.tsx` | Login form behavior |
| `src/components/ChatPanel.test.tsx` | Markdown rendering in transcript |
| `src/components/SaveEngramModal.test.tsx` | Save modal behavior |
| `src/components/SessionSidebar.test.tsx` | Sidebar interaction (toggle, labels, active state) |
| `src/components/DocumentIngestionPanel.test.tsx` | Ingestion panel (text/file + pin/unpin) |
| `src/components/AdminMcpTokenPanel.test.tsx` | Token manager form + revoke workflows |
| `src/api/mcpClient.test.ts` | MCP client protocol parsing + SSE handling |

---

## Acceptance Tests (Playwright-BDD)

### Local Runner

```bash
make acceptance-sync
make acceptance-bddgen
make acceptance-typecheck
make acceptance-test
make acceptance-test-mock
```

### Dockerized Runner

```bash
make acceptance-test-docker
make acceptance-test-mock-docker
```

### Live Provider Runners

```bash
make acceptance-test-bedrock-live
make acceptance-test-bedrock-live-docker
make acceptance-test-triage-live
make acceptance-test-triage-live-docker
```

### Feature Coverage

| Feature | Tags | Description |
|---------|------|-------------|
| `authentication.feature` | - | Login handoff (sign-in without manual refresh) |
| `session-layout.feature` | - | Chat pane layout stability during session creation |
| `bedrock-live.feature` | `@bedrock-live` | Non-deterministic Bedrock response validation |
| `triage-live.feature` | `@triage-live @bedrock-live` | Save-as-engram, pinning, continuity handoff |
| `lifecycle-autosave-mock.feature` | `@mock` | Autosave `off` and `message_count` thresholds |
| `engram-auto-metadata-mock.feature` | `@mock` | MCP conversation-only persistence + metadata |
| `mcp-token-auth-mock.feature` | `@mock` | Bearer read/write scope authorization |
| `admin-mcp-token-ui.feature` | `@mock` | Admin chip-selection for tool/project-scoped tokens |
| `phase31-memory-admin-mock.feature` | `@mock` | Admin memory management workflows |

Notes:
- Default execution filters to `not @bedrock-live` for deterministic local runs.
- `make acceptance-bddgen` regenerates Playwright specs from `.feature` files.
- Failure screenshots are persisted to `acceptance-tests/artifacts/`.
- Generated specs are written to `acceptance-tests/.features-gen/`.

---

## Evaluation Harness

```bash
make eval
```

Evaluation output: `api/evals/last_eval.json`

Scenarios: fact recall, cross-engram reasoning proxy, temporal ordering, abstention behavior.

---

## High-Impact Test Workflow

Use this to exercise continuity, persistence, and replay behavior in one pass.

1. Start full stack and acceptance baseline:
```bash
make stack-up
make acceptance-test-docker
```

2. Open [http://localhost:5173](http://localhost:5173) and sign in (`admin` / `admin123`).

3. Create three sessions in the same project:
   - Session A (`openai`) for general prompts
   - Session B (`anthropic`) for comparison prompts
   - Session C (`bedrock`) to validate adapter config/error surface

4. In Session A: send a prompt, save transcript as engram, copy the engram ID.

5. In Session B: search engrams, pin the engram from Session A, send a prompt requiring prior context, verify `used_engram_ids` and source references.

6. Run "Continue in New Chat" from Session B and verify pinned IDs carry forward.

7. Validate persistence outside UI:
```bash
make cli ARGS="search --query 'continued' --project-id engram-vault --top-k 5"
```

8. Confirm diagnostics: redacted config in dev mode, specific Bedrock error messages.
