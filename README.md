# Engram Vault (Local-First MVP)

This project turns long LLM research runs into durable, queryable "memory engrams" so context does not disappear between sessions.

This README is written for a newcomer and follows an implementation sequence based on `deep-research-report.md` and the unified roadmap in `Plan.md`.

## Docs

- Architecture playbook: `docs/architecture-playbook.md`
- UI user flow (sessions + engrams): `docs/user-flow-engram-workflow.md`
- MCP client integration commands (LibreChat/Copilot/Codex): `docs/mcp-client-integrations.md`
- Ready-to-use VS Code MCP config: `.vscode/mcp.json`
- PlantUML architecture/workflow map: `docs/architecture-workflows.puml`
- PlantUML use-case map (model switch + save/pin/continue): `docs/model-switch-engram-usecases.puml`
- Render PlantUML via Docker (no local `dot` needed): `make diagram-render`
- Scope note: playbook content documents existing behavior only (no API/schema changes).

PlantUML rendering commands:

```bash
make diagram-render
make diagram-render-png
```

Rendered files are written to `docs/rendered/`.
`make diagram-render` generates:

- `docs/rendered/architecture-workflows.svg`
- `docs/rendered/model-switch-engram-usecases.svg`

If local PlantUML fails with `Cannot run program "/opt/local/bin/dot"`, use the Docker render commands above (or install Graphviz and set `GRAPHVIZ_DOT` to your local `dot` binary).

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
- [x] Add multi-user auth model + role-based access controls (admin/analyst/viewer).
- [x] Add CLI/UI for upload/search/rehydrate workflows.
- [x] Add automated evaluation harness (temporal + multi-engram reasoning).
- [x] Add reranking/citation-packing improvements for retrieval quality.
- [x] Add background consolidation jobs for long-run memory maintenance.
- [x] Add local security baseline (audit log + login rate limiting/lockout).
- [x] Add schema/repository layer for chat sessions, pinning, and visibility.
- [x] Add provider adapter layer (OpenAI, Anthropic, Bedrock) with registry.
- [x] Add chat APIs and continuity workflows (pin, save-as-engram, continue, stream).
- [x] Add MCP JSON-RPC over SSE endpoint with chat/engram/user tool routing.
- [x] Add React chat UI with session, streaming, pin/save/continue workflows.
- [x] Migrate web styling to shared `styled-components` + Tailwind style system.
- [x] Add backend dev-mode parsed-config logging with secret redaction.
- [x] Dockerize API + web runtime with env-driven compose orchestration.
- [x] Add Playwright-BDD + `bddgen` acceptance framework with dockerized execution.
- [x] Add tagged Bedrock live acceptance flow for non-deterministic provider validation.
- [x] Add non-deterministic triage-continuity acceptance flow (save, continue, pin, handoff).
- [x] Improve chat snapshot engram rehydration with transcript-derived summaries.
- [x] Add markdown rendering support for chat transcript messages.
- [x] Fix SSE stream parsing for CRLF frame boundaries to restore live token streaming.
- [x] Refresh UI theme (no green palette) with richer markdown presentation.
- [x] Add persistent light/dark theme mode with in-app toggle.
- [x] Fix dark-theme scrollbar contrast and session sidebar split-scroll ergonomics.
- [x] Add sticky creator footer, previous-session label, and stronger active-session highlight.
- [x] Refine engram panel UX with title-first cards, compact actions, floating markdown tooltips, pinned-state highlights, and a fixed mid divider.
- [x] Merge baseline and next roadmap into a single unified planning document.
- [x] Add architecture playbook with Mermaid diagrams and multi-model continuity runbooks.
- [x] Add detailed PlantUML architecture/workflow map and Docker-based renderer.
- [x] Add detailed PlantUML use-case diagram for model switching and engram continuity lifecycle.
- [x] Add clean reset scripts and stale-session reconciliation to keep chat creation stable after DB resets.
- [x] Add MCP interoperability layer (`initialize`, `tools/list`, `tools/call`) plus typed Python/TypeScript JSON-RPC/SSE clients and contract tests.
- [x] Add Phase 17 ingestion stack: text/file document ingest APIs, deterministic chunking, embedding abstraction with local fallback, chat-context chunk blending, and upload UI.
- [x] Add session-level document pinning so uploaded docs can be explicitly carried into chat context and continuation sessions.
- [x] Guarantee multi-document pin behavior so all pinned docs contribute to context and source metadata.
- [x] Add chat debug traces with embed/LLM timing, token usage, and input/output inspection (optional Langfuse sink).
- [x] Harden observability with latest Langfuse API-only tracing, explicit publish/error logs, and scrollable in-UI debug trace inspection.
- [x] Add Phase 18 memory lifecycle controls (autosave strategies, retention pruning, consolidation-aware timeline events, and UI/MCP policy management).
- [x] Add deterministic mocked acceptance coverage for autosave lifecycle policies.
- [x] Add Phase 29 optional auto-metadata enrichment (fill-empty-only) across all engram create paths plus MCP conversation-only persistence tooling.
- [x] Add Phase 30 MCP personal access tokens (PAT) with scoped read/write, optional tool allowlists, optional project allowlists, bearer auth on MCP stream, and admin token lifecycle UI.
- [x] Add admin-only React MCP token manager (create/list/revoke) gated by authenticated `admin` role.
- [x] Add chip-based admin token policy UX: load available MCP tools/projects, select them from dropdowns, and manage applied restrictions as removable chips.
- [x] Add OAuth authorization server compatibility for MCP clients (metadata discovery, dynamic client registration, PKCE authorization code exchange, protected-resource metadata, and OAuth-guided `WWW-Authenticate` challenges).
- [ ] Complete Phase 31 project-default resolution + enterprise memory management closeout (docs/skills finalization + remaining acceptance evidence).
- [ ] Add production security hardening (full OIDC, centralized audit sink, distributed rate limits).

## Unified Plan Status

- `Plan.md`: canonical roadmap (historical baseline + chat/MCP expansion + future phases).
- Completed scope:
  - phases 0-15 completed (foundation, schema/storage, retrieval/rehydration, durability, chat continuity, providers, MCP, UI, acceptance, theme/UX hardening).
  - phase 29 completed (optional deterministic auto-metadata enrichment and MCP conversation-only persistence path).
  - phase 30 completed (MCP PAT lifecycle APIs/UI plus scoped bearer authorization for external agents).
- Next implementation scope:
  - phase 16: MCP developer tooling and typed clients (in progress: compatibility + typed clients complete, CLI smoke command deferred).
  - phase 17: document ingestion and RAG-ready retrieval (implemented and verified, including session-level document pinning support).
  - phase 18: memory lifecycle policies (core autosave/retention/timeline controls implemented and validated).
  - phase 31: project defaults + enterprise memory management + MCP organization (in progress: backend/API/MCP/web/tests implemented; docs/skills closeout pending).
  - phase 19: collaboration and sharing model.
  - phase 20: production security hardening.

## Phase 31 Progress Tracker

- [x] `Plan.md` updated with Phase 31 status and near-term execution order.
- [x] Added first-class `projects` table and per-user default project persistence.
- [x] Added default-project fallback when `project_id` is missing in engram create paths.
- [x] Added admin memory router (`/api/v1/admin/memory`) for session/engram/collection management.
- [x] Added MCP organization tools for project, engram, collection, and session lifecycle operations.
- [x] Added dedicated admin memory UI page with routing (`/admin/memory`).
- [x] Hardened `/api/v1/engrams*` with authenticated actor-scoped visibility.
- [x] Added backend/web/acceptance tests for Phase 31 behavior.
- [ ] Update `AGENT.md` + skills docs and append final phase-closeout validation evidence.

### Planned Phase 31 User-Facing Areas

- Project defaults:
  - Set default project once and reuse it when `project_id` is omitted.
  - Preserve explicit `project_id` when provided by callers.
- Memory management page:
  - Separate admin route for listing, editing, moving, deleting, and restoring sessions/engrams.
  - Project-bounded collections to organize related engrams.
- MCP organization tools:
  - Allow agent-driven memory organization directly from chat/tool calls with scoped authorization.

### Phase 31 Implemented So Far (Detailed)

1. Schema and persistence foundation:
   - `db/init/001_schema.sql`
     - added `projects` table
     - added `users.default_project_id` foreign key
     - added soft-delete metadata (`deleted_at`, actor and reason fields) for sessions/engrams
     - added `engram_collections` and `engram_collection_items`
     - added idempotent project backfill for existing data
2. Backend domain modules:
   - `api/app/projects/` (`api.py`, `service.py`, `repository.py`, `models.py`)
   - `api/app/memory_admin/` (`api.py`, `service.py`, `repository.py`, `models.py`)
   - wired into `api/app/main.py`
3. API behavior + security:
   - `/api/v1/engrams*` now requires authenticated actors
   - `POST /api/v1/engrams` now resolves missing project using caller default project
   - response models expose `resolved_project_id` and `used_default_project`
4. MCP organization capability:
   - new project tools: `project_list`, `project_create`, `project_get_default`, `project_set_default`
   - new engram tools: `engram_list`, `engram_get`, `engram_update`, `engram_move_project`, `engram_delete`, `engram_restore`
   - new collection tools: `engram_collection_list`, `engram_collection_create`, `engram_collection_update`, `engram_collection_delete`, `engram_collection_add_items`, `engram_collection_remove_items`
   - new session tools: `chat_delete_session`, `chat_restore_session`
   - retained dotted aliases for backward compatibility
5. Web admin UX:
   - routing introduced (`/` workspace, `/admin/memory` admin page)
   - default-project set/get controls in session sidebar
   - admin page supports list/filter/edit/move/delete/restore for sessions/engrams and collection assignment workflows
6. Phase 31 tests added/updated:
   - backend: `api/tests/test_projects_api_integration.py`, `api/tests/test_memory_admin_api_integration.py`, `api/tests/test_schema_backfill_projects.py`
   - MCP integration extensions in `api/tests/test_mcp_api_integration.py`
   - web: `web/src/api/projects.test.ts`, `web/src/api/memoryAdmin.test.ts`, `web/src/components/AdminMemoryPage.test.tsx`
   - acceptance: `acceptance-tests/features/phase31-memory-admin-mock.feature`, `acceptance-tests/src/steps/phase31-memory-admin-mock.steps.ts`

## Agent Guide

- `AGENT.md` is the canonical contribution and maintenance contract for human contributors and agents.
- Before changing architecture, API contracts, or schema behavior, read `AGENT.md` first.
- Every phase updates this README and keeps `Plan.md` and `skills/` aligned.

## Skills Catalog

Agent workflow skills are under `skills/`:

- `engram-lifecycle`: creation/query/rehydration and ownership/visibility behavior.
- `chat-rag-operator`: chat session continuity and context assembly with engrams.
- `mcp-http-stream-tools`: JSON-RPC over SSE tool contracts and handlers.
- `provider-openai`: OpenAI adapter implementation conventions.
- `provider-anthropic`: Anthropic adapter implementation conventions.
- `provider-bedrock`: Bedrock adapter implementation conventions.
- `schema-migrations-and-backfill`: forward-only schema evolution and compatibility.
- `testing-and-evals`: test matrix and eval harness extension guidance.
- `release-and-maintenance`: release checklist and long-run maintenance cadence.
- `domain-module-layout`: module/package conventions for long-run maintainability.
- `react-chat-ui-operator`: frontend workflow conventions for chat/session/engram UX.
- `frontend-style-system`: token-driven styled-components + Tailwind workflow rules.
- `dockerized-acceptance-testing`: Playwright-BDD (`bddgen`) dockerized quality-gate workflow.
- `document-ingestion-rag`: deterministic document chunking, ingestion APIs, and blended retrieval workflow.
- `memory-lifecycle-policies`: autosave cadence, retention bounds, consolidation guards, and timeline event workflow.
- `engram-auto-metadata-enrichment`: deterministic fill-empty metadata derivation rules and integration points.
- `mcp-token-authz`: MCP PAT lifecycle, scope/allowlist/project authorization rules, and external agent integration checks.

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
| (chat + ingest workflows) |         | (automation/agent expansion)  |
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
  .vscode/
    mcp.json
  README.md
  AGENT.md
  Plan.md
  docs/
    architecture-playbook.md
    user-flow-engram-workflow.md
    mcp-client-integrations.md
    architecture-workflows.puml
    model-switch-engram-usecases.puml
    render-plantuml.sh
    screenshots/
      user-flow/
        01-login.png
        ...
        12-continued-session.png
  deep-research-report.md
  .dockerignore
  .env.example
  docker-compose.yml
  Makefile
  skills/
    engram-lifecycle/
      SKILL.md
    chat-rag-operator/
      SKILL.md
    mcp-http-stream-tools/
      SKILL.md
    provider-openai/
      SKILL.md
    provider-anthropic/
      SKILL.md
    provider-bedrock/
      SKILL.md
    schema-migrations-and-backfill/
      SKILL.md
    testing-and-evals/
      SKILL.md
    release-and-maintenance/
      SKILL.md
    domain-module-layout/
      SKILL.md
    react-chat-ui-operator/
      SKILL.md
    frontend-style-system/
      SKILL.md
    dockerized-acceptance-testing/
      SKILL.md
    document-ingestion-rag/
      SKILL.md
    memory-lifecycle-policies/
      SKILL.md
    engram-auto-metadata-enrichment/
      SKILL.md
    mcp-token-authz/
      SKILL.md
  db/
    init/
      001_schema.sql
  api/
    Dockerfile
    pyproject.toml
    uv.lock
    evals/
      __init__.py
      harness.py
      run_eval.py
    app/
      __init__.py
      agent_models.py
      agent_workflow.py
      audit.py
      auth.py
      chat/
        __init__.py
        api.py
        context.py
        errors.py
        lifecycle_policy.py
        service.py
      chat_repository.py
      cli.py
      consolidation.py
      config.py
      db.py
      embedding.py
      embeddings/
        __init__.py
        base.py
        errors.py
        local_provider.py
        openai_provider.py
        service.py
      engram_enrichment/
        __init__.py
        models.py
        service.py
      ingestion/
        __init__.py
        api.py
        chunking.py
        errors.py
        models.py
        repository.py
        service.py
      login_guard.py
      main.py
      mcp/
        __init__.py
        api.py
        errors.py
        service.py
      models.py
      providers/
        __init__.py
        base.py
        errors.py
        openai_provider.py
        anthropic_provider.py
        bedrock_provider.py
        registry.py
      repository.py
      user_repository.py
      templates/
        admin.html
        login.html
        dashboard.html
  web/
    Dockerfile
    .env.example
    package.json
    package-lock.json
    postcss.config.cjs
    tailwind.config.ts
    vite.config.ts
    tsconfig.app.json
    src/
      App.tsx
      ThemedApp.tsx
      config.ts
      index.css
      styles/
        globalStyles.ts
        primitives.ts
        styled.d.ts
        theme.ts
        themeMode.tsx
        themeModeContext.ts
        themeModeUtils.ts
        useThemeMode.ts
      api/
        auth.ts
        auth.test.ts
        chat.ts
        http.ts
        ingestion.ts
        types.ts
      components/
        ChatPanel.tsx
        ChatPanel.test.tsx
        DocumentIngestionPanel.tsx
        DocumentIngestionPanel.test.tsx
        LoginView.tsx
        LoginView.test.tsx
        PinnedEngramPanel.tsx
        SaveEngramModal.tsx
        SaveEngramModal.test.tsx
        SessionSidebar.tsx
        SessionSidebar.test.tsx
      test/
        setup.ts
      utils/
        chat.ts
        chat.test.ts
        sse.ts
        sse.test.ts
  acceptance-tests/
    Dockerfile
    README.md
    package.json
    package-lock.json
    playwright.config.ts
    .env.example
    features/
      authentication.feature
      bedrock-live.feature
      engram-auto-metadata-mock.feature
      lifecycle-autosave-mock.feature
      mcp-token-auth-mock.feature
      session-layout.feature
      triage-live.feature
    src/
      support/
        chat.ts
        env.ts
        fixtures.ts
      steps/
        auth.steps.ts
        bedrock.steps.ts
        engram-auto-metadata-mock.steps.ts
        lifecycle-mock.steps.ts
        mcp-token-auth-mock.steps.ts
        session.steps.ts
        triage.steps.ts
```

## File-by-File Guide

- `.dockerignore`: build context exclusions for API/web/acceptance Docker builds.
- `docker-compose.yml`: local DB + API + web + acceptance test orchestration.
- `AGENT.md`: project operating guide for contributors and agents.
- `Plan.md`: canonical merged roadmap (completed phases + upcoming phases).
- `docs/architecture-playbook.md`: newcomer-first and technical architecture narrative with call-flow diagrams and multi-model continuity runbooks.
- `docs/user-flow-engram-workflow.md`: ordered UI walkthrough for session creation, save-as-engram, pinning, and continuation with screenshots.
- `docs/architecture-workflows.puml`: detailed PlantUML architecture and workflow map covering REST, MCP, provider streaming, and persistence flows.
- `docs/model-switch-engram-usecases.puml`: detailed PlantUML use-case diagram for model switching, save/pin/continue workflows, and continuity metadata inspection.
- `docs/render-plantuml.sh`: Docker-based PlantUML renderer that avoids local Graphviz path requirements.
- `docs/screenshots/user-flow/`: screenshot assets used in the user-flow walkthrough.
- `db/init/001_schema.sql`: Database extension, tables, and indexes.
- `api/app/main.py`: FastAPI routes and API surface.
- `api/app/agent_models.py`: request/response models for agent runs.
- `api/app/agent_workflow.py`: LangGraph workflow, checkpointing, and resume logic.
- `api/app/audit.py`: append-only local audit event writer (`jsonl`).
- `api/app/auth.py`: password hashing/verification and CSRF token helpers.
- `api/app/chat/api.py`: chat/session REST route layer (`/api/v1/chat/*`) including lifecycle policy/timeline and pinned engram/document management routes.
- `api/app/chat/context.py`: context assembler for pinned + retrieved engram packs plus pinned/retrieved document chunk context.
- `api/app/chat/errors.py`: chat-domain error types mapped to HTTP responses.
- `api/app/chat/lifecycle_policy.py`: pure lifecycle policy helpers for autosave normalization, snapshot triggers, retention pruning, and timeline event typing.
- `api/app/chat/service.py`: chat continuity orchestration, lifecycle maintenance (autosave/retention), and provider call workflow.
- `api/app/chat_repository.py`: chat session/message, lifecycle policy field persistence, and pinned-engram/pinned-document visibility checks.
- `api/app/cli.py`: local terminal workflows for upload/search/rehydrate.
- `api/app/consolidation.py`: local background maintenance logic for consolidation snapshots.
- `api/app/ingestion/api.py`: ingestion REST routes (`/api/v1/ingestion/*`) for text/file intake and retrieval.
- `api/app/ingestion/chunking.py`: deterministic content hashing + chunk boundary and chunk ID generation.
- `api/app/ingestion/repository.py`: document/chunk persistence and vector query with lexical rerank blend.
- `api/app/ingestion/service.py`: ingestion validation, UTF-8 file guards, and blended retrieval orchestration.
- `api/app/ingestion/models.py`: ingestion request/response contracts and blended query models.
- `api/app/login_guard.py`: login attempt rate-limit and lockout state machine.
- `api/app/mcp/api.py`: MCP JSON-RPC over SSE route layer.
- `api/app/mcp/auth.py`: bearer-token/session actor resolution for MCP stream requests.
- `api/app/mcp/client.py`: typed Python JSON-RPC/SSE MCP client helper for external integrations.
- `api/app/mcp/errors.py`: MCP RPC error types and codes.
- `api/app/mcp/service.py`: MCP tool dispatch, compatibility methods (`initialize`/`tools/*`), JSON-RPC frame generation, and token scope/allowlist/project authz checks.
- `api/app/mcp_tokens/models.py`: MCP token entity and auth-context contracts.
- `api/app/mcp_tokens/repository.py`: MCP token persistence/read/revoke/last-used operations.
- `api/app/mcp_tokens/service.py`: token issue/parse/hash/verify helpers and token policy normalization.
- `api/app/oauth/api.py`: OAuth metadata, dynamic client registration, authorization-code + PKCE endpoints, and protected-resource metadata routes.
- `api/app/oauth/models.py`: OAuth client and authorization-code persistence record contracts.
- `api/app/oauth/repository.py`: OAuth client/auth-code CRUD and code-consumption operations.
- `api/app/oauth/service.py`: OAuth scope normalization, PKCE validation, secret hashing, and issuer URL helpers.
- `api/app/observability/chat_debug.py`: chat debug collector hooks and optional Langfuse trace publishing.
- `api/app/models.py`: Request/response and engram schema models.
- `api/app/providers/base.py`: provider adapter contract and normalized request/response types.
- `api/app/providers/errors.py`: provider-layer error taxonomy.
- `api/app/providers/openai_provider.py`: OpenAI endpoint adapter implementation.
- `api/app/providers/anthropic_provider.py`: Anthropic endpoint adapter implementation.
- `api/app/providers/bedrock_provider.py`: Bedrock adapter implementation.
- `api/app/providers/registry.py`: provider adapter factory and resolver.
- `api/app/embeddings/base.py`: embedding provider protocol and result metadata contract.
- `api/app/embeddings/local_provider.py`: deterministic local embedding provider implementation.
- `api/app/embeddings/openai_provider.py`: optional OpenAI embeddings adapter (with size coercion guards).
- `api/app/embeddings/service.py`: embedding provider resolver + local fallback strategy.
- `api/app/engram_enrichment/models.py`: deterministic enrichment payload/report contracts and normalization helpers.
- `api/app/engram_enrichment/service.py`: fill-empty-only abstract/keyword/tag derivation logic for engram persistence.
- `api/app/repository.py`: SQL persistence, reranked semantic query, and citation-packed rehydration builder.
- `api/app/user_repository.py`: user persistence, lookup, and role-aware updates.
- `api/app/embedding.py`: Deterministic local embedding helper.
- `api/app/db.py`: DB connection lifecycle.
- `api/app/config.py`: Environment-backed settings plus dev-mode safe config snapshot helpers.
- `api/app/templates/admin.html`: admin console for user/role inspection.
- `api/app/templates/login.html`: local sign-in page for UI testing.
- `api/app/templates/dashboard.html`: authenticated local dashboard for API testing.
- `api/pyproject.toml`: project dependencies, pytest config, and ruff config.
- `api/uv.lock`: locked dependency graph for reproducible local runs.
- `api/Dockerfile`: containerized API runtime (`uvicorn` + `uv` lockfile sync).
- `api/evals/harness.py`: local scenario-driven evaluation harness and scoring logic.
- `api/evals/run_eval.py`: CLI runner for local evaluation output (`make eval`).
- `api/tests/conftest.py`: API client, schema bootstrap, DB cleanup fixtures.
- `api/tests/test_api_unit.py`: unit tests for API routing behavior.
- `api/tests/test_api_integration.py`: full API + DB integration tests.
- `api/tests/test_user_rbac.py`: multi-user auth and role-based access checks.
- `api/tests/test_eval_harness.py`: integration check that evaluation scenarios pass.
- `api/tests/test_cli.py`: unit tests for CLI argument handling and command behavior.
- `api/tests/test_consolidation.py`: unit tests for consolidation job behavior and guardrails.
- `api/tests/test_ui_auth.py`: login/logout/session workflow + CSRF + rate-limit + audit checks.
- `api/tests/test_chat_repository.py`: integration coverage for chat sessions/messages/pinning visibility.
- `api/tests/test_chat_api_integration.py`: end-to-end chat API lifecycle and continuity flow tests.
- `api/tests/test_chat_context.py`: context assembly merge/dedupe behavior tests.
- `api/tests/test_chat_lifecycle_policy.py`: lifecycle policy normalization/trigger/pruning helper tests.
- `api/tests/test_chat_service.py`: chat service orchestration and save/continue behavior tests.
- `api/tests/test_mcp_api_integration.py`: MCP SSE transport and tool success/error/auth coverage.
- `api/tests/test_mcp_client.py`: typed Python MCP client parsing/auth/transport contract tests.
- `api/tests/test_mcp_token_service.py`: token generation/hash/parse/expiry/revocation behavior tests.
- `api/tests/test_mcp_token_api_integration.py`: MCP token lifecycle REST authz tests (admin vs non-admin).
- `api/tests/test_admin_mcp_tokens_ui.py`: admin UI MCP token create/revoke workflow coverage.
- `api/tests/test_mcp_oauth_integration.py`: OAuth metadata, dynamic registration, authorize/token exchange, and OAuth-issued MCP bearer usage coverage.
- `api/tests/test_oauth_service.py`: OAuth scope + PKCE helper unit tests.
- `api/tests/test_engram_enrichment.py`: deterministic enrichment behavior and non-overwrite contract tests.
- `api/tests/test_engram_visibility.py`: integration checks for owner/project scope filtering behavior.
- `api/tests/test_provider_registry.py`: provider registry construction and adapter selection checks.
- `api/tests/test_provider_adapters.py`: adapter normalization and error-path tests.
- `api/tests/test_embedding.py`: embedding utility tests.
- `api/tests/test_repository_helpers.py`: repository helper tests.
- `api/tests/test_ingestion_chunking.py`: deterministic chunking/hash/document-id behavior tests.
- `api/tests/test_ingestion_service.py`: ingestion validation and persistence orchestration tests.
- `api/tests/test_ingestion_api_integration.py`: authenticated ingestion + retrieval API lifecycle tests.
- `api/tests/test_embeddings_service.py`: embedding provider fallback behavior tests.
- `api/tests/test_config_settings.py`: settings debug logging and secret redaction checks.
- `web/src/App.tsx`: React chat workbench composition, workflow state, and theme toggle control.
- `web/src/ThemedApp.tsx`: mode-aware `ThemeProvider` wrapper for runtime light/dark switching.
- `web/src/config.ts`: frontend runtime config parsing + one-time debug console print.
- `web/src/api/*.ts`: browser API clients for auth/chat/engram interactions.
- `web/src/api/chat.ts`: browser API client for chat/session/pin flows, including pinned-document route calls.
- `web/src/api/ingestion.ts`: browser API client for document ingestion and project document listing.
- `web/src/api/mcpTokens.ts`: browser API client for admin MCP token lifecycle routes.
- `web/src/api/mcpClient.ts`: typed TypeScript JSON-RPC/SSE MCP client helper.
- `web/src/api/mcpClient.test.ts`: TypeScript MCP client protocol parsing and stream contract tests.
- `web/src/components/*.tsx`: UI modules for login, sessions, chat transcript, pinning, and save modal.
- `web/src/components/AdminMcpTokenPanel.tsx`: admin-only token manager modal for create/list/revoke token flows.
- `web/src/components/AdminMcpTokenPanel.test.tsx`: token manager form parsing + revoke workflow unit coverage.
- `web/src/components/DocumentIngestionPanel.tsx`: project-scoped file/text ingestion surface with recent docs and session pin/unpin controls.
- `web/src/components/DocumentIngestionPanel.test.tsx`: ingestion panel interaction tests (text/file plus pin/unpin flows).
- `web/src/components/SessionSidebar.test.tsx`: sidebar interaction tests (toggle, labels, active-state marker).
- `web/src/styles/theme.ts`: shared frontend light/dark design tokens and theme registry.
- `web/src/styles/globalStyles.ts`: global CSS variables and base element styles.
- `web/src/styles/primitives.ts`: reusable styled panels/cards/typography primitives.
- `web/src/styles/themeMode.tsx`: persisted theme-mode provider state.
- `web/src/styles/themeModeContext.ts`: theme-mode context contract.
- `web/src/styles/themeModeUtils.ts`: theme-mode parsing and initialization helpers.
- `web/src/styles/useThemeMode.ts`: theme-mode hook.
- `web/src/utils/chat.ts`: save-as-engram default abstract derivation.
- `web/src/utils/chat.test.ts`: tests for abstract derivation behavior.
- `web/src/utils/sse.ts`: SSE parser used for streaming chat responses.
- `web/src/**/*.test.ts(x)`: Vitest + Testing Library frontend tests.
- `web/postcss.config.cjs`: PostCSS pipeline for Tailwind.
- `web/tailwind.config.ts`: Tailwind token mapping to shared CSS variables.
- `web/vite.config.ts`: Vite proxy/allowed-host config + Vitest configuration.
- `web/Dockerfile`: containerized web runtime (Vite dev server for API proxy parity).
- `acceptance-tests/README.md`: acceptance framework guide and commands.
- `acceptance-tests/features/*.feature`: Gherkin acceptance scenarios.
- `acceptance-tests/features/engram-auto-metadata-mock.feature`: deterministic MCP conversation-only enrichment behavior.
- `acceptance-tests/features/mcp-token-auth-mock.feature`: MCP bearer read/write scope authorization scenarios.
- `acceptance-tests/features/admin-mcp-token-ui.feature`: admin UI chip-selection workflow for tool/project-scoped MCP token creation.
- `acceptance-tests/src/steps/*.ts`: Playwright-backed step definitions.
- `acceptance-tests/src/steps/engram-auto-metadata-mock.steps.ts`: step bindings for MCP conversation-only metadata enrichment checks.
- `acceptance-tests/src/steps/mcp-token-auth-mock.steps.ts`: step bindings for MCP token creation, bearer invocation, and scope enforcement checks.
- `acceptance-tests/src/steps/admin-mcp-token-ui.steps.ts`: step bindings for admin token panel option loading, chip selection, and role-gated visibility checks.
- `acceptance-tests/src/support/*.ts`: shared fixtures, env parsing, chat helpers, login helpers, and failure artifacts.
- `acceptance-tests/playwright.config.ts`: `playwright-bdd` + `defineBddConfig` + runtime fixture wiring.
- `acceptance-tests/Dockerfile`: Playwright runtime image for dockerized acceptance runs.
- `Makefile`: Local run shortcuts.
- `.env.example`: Starter configuration for local setup.
- `skills/`: reusable agent workflows for implementation and maintenance.

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

4. Install web dependencies:

```bash
cd web
npm install
cd ..
```

5. Optional: set frontend defaults:

```bash
cp web/.env.example web/.env
```

6. Run API:

```bash
make api
```

7. In a second terminal, run web app:

```bash
make web-dev
```

8. Open API docs:

- [http://localhost:8000/docs](http://localhost:8000/docs)
- [http://localhost:5173](http://localhost:5173) (React chat workbench)

UI testing entrypoints:

- [http://localhost:8000/login](http://localhost:8000/login)
- [http://localhost:8000/ui](http://localhost:8000/ui)
- [http://localhost:8000/ui/admin](http://localhost:8000/ui/admin) (admin role)

Default local UI credentials (seeded in `db/init/001_schema.sql`):

- username: `admin`
- password: `admin123`

### Admin Console Test Runbook (Multi-Document Pin + MCP)

Use this sequence to validate the latest multi-document continuity path end-to-end with admin credentials.

1. Sign in as admin:
- open [http://localhost:5173](http://localhost:5173) (or [http://localhost:5174](http://localhost:5174) in docker mode)
- login with `admin` / `admin123`

2. Optional admin-role sanity check:
- open [http://localhost:8000/ui/admin](http://localhost:8000/ui/admin)
- verify user list renders (confirms admin role/session health)

3. Create a session:
- in `Create Session`, set `Project ID` to `engram-vault` (or your test project)
- click `Create Session`

4. Ingest two documents:
- in `Document Ingestion` -> `Show Upload Form`, ingest two text/file docs into the same project
- verify both appear in `Recent Documents`

5. Pin both docs:
- click `Pin to Chat` on both documents
- verify both cards show pinned state

6. Send a prompt:
- ask for a summary that should use both docs
- verify `Source references used` includes document entries

7. Continue chat:
- click `Continue in New Chat`
- verify pinned docs are still present in the continued session

8. MCP parity check:
- use `tools/list` and confirm document-pin tools are exposed
- call `chat.list_pinned_documents` and verify both document IDs are returned

Local security baseline knobs (optional in `.env`):

- `AUDIT_LOG_PATH` (default `./data/audit_events.jsonl`)
- `LOGIN_RATE_LIMIT_MAX_ATTEMPTS` (default `5`)
- `LOGIN_RATE_LIMIT_WINDOW_SECONDS` (default `300`)
- `LOGIN_LOCKOUT_SECONDS` (default `900`)

Backend debug logging knobs (optional in `.env`):

- `APP_ENV` (default `development`)
- `LOG_CONFIG_IN_DEV` (default `true`, prints a redacted parsed-config snapshot on startup in dev/local envs)
- `CHAT_DEBUG_ENABLED` (default `true`, attach debug trace payloads to chat responses and stream completion events)
- `CHAT_DEBUG_LOG_CONSOLE` (default `true`, print chat debug payloads in API logs)
- `CHAT_DEBUG_INCLUDE_RAW_TEXT` (default `true`, include request/response text in debug payloads)
- `LANGFUSE_ENABLED` (default `false`, publish traces to Langfuse when configured)
- `LANGFUSE_HOST` (default `https://cloud.langfuse.com`)
- `LANGFUSE_PUBLIC_KEY`, `LANGFUSE_SECRET_KEY`

Optional Langfuse dependency (required only for remote trace export):

```bash
cd api
uv add langfuse
```

Langfuse SDK compatibility:

- publisher targets the latest observation API only (`start_as_current_observation`).
- if an older/incompatible client is installed, tracing is skipped with an explicit warning log.

Expected tracing log signals:

- startup:
  - `[engram-chat-debug] Langfuse tracing disabled by config` when `LANGFUSE_ENABLED=false`
  - `[engram-chat-debug] Langfuse tracing enabled` when client initialization succeeds
  - warning logs for missing dependency or missing keys
- per message:
  - `[engram-chat-debug]` JSON payload printed when `CHAT_DEBUG_LOG_CONSOLE=true`
  - `[engram-chat-debug] Langfuse trace published` on successful remote publish
  - `[engram-chat-debug] Langfuse trace publish failed` with stack trace on exceptions

Provider configuration knobs (optional in `.env` unless provider enabled):

- `DEFAULT_CHAT_PROVIDER` (default `openai`)
- `DEFAULT_CHAT_MODEL` (default `gpt-4o-mini`)
- `OPENAI_API_KEY`, `OPENAI_BASE_URL`
- `ANTHROPIC_API_KEY`, `ANTHROPIC_BASE_URL`, `ANTHROPIC_VERSION`
- `AWS_REGION`, `AWS_ACCESS_KEY_ID`, `AWS_SECRET_ACCESS_KEY`, `AWS_SESSION_TOKEN`

LangGraph checkpoint file (local):

- `LANGGRAPH_CHECKPOINT_PATH=./data/langgraph_checkpoints.sqlite`

9. Run backend tests:

```bash
make test
```

10. Run web checks:

```bash
make web-check
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

9. Run local memory quality evaluations:

```bash
make eval
```

Evaluation output is written to `api/evals/last_eval.json`.

10. Run CLI workflows (local operator path):

```bash
make cli ARGS="upload --file /tmp/engram.json"
make cli ARGS="search --query 'durable runs' --project-id engram-vault --top-k 5"
make cli ARGS="rehydrate --engram-id <engram_uuid>"
make consolidate ARGS="--project-id engram-vault --dry-run"
```

## Dockerized Stack (API + Web + DB)

Use this path when you want one-command local infrastructure with no host-level Python/Node runtimes.

1. Copy env:

```bash
cp .env.example .env
```

2. Start stack:

```bash
make stack-up
docker compose ps
```

3. Open:

- [http://localhost:8000/docs](http://localhost:8000/docs)
- [http://localhost:5174](http://localhost:5174) (React chat workbench)

4. Stop stack:

```bash
make stack-down
```

Fresh clean reset commands (drops DB volume and local runtime files):

```bash
make db-reset
make stack-reset
```

### Docker Env Matrix (Core)

- DB: `POSTGRES_USER`, `POSTGRES_PASSWORD`, `POSTGRES_DB`, `POSTGRES_PORT`
- API runtime: `APP_ENV`, `LOG_CONFIG_IN_DEV`, `APP_SEMANTIC_VERSION`, `APP_COMMIT_SHA`, `EMBEDDING_DIM`, `EMBEDDING_PROVIDER`, `EMBEDDING_MODEL`, `EMBEDDING_FALLBACK_TO_LOCAL`, `INGESTION_MAX_FILE_BYTES`, `INGESTION_MAX_TEXT_CHARS`, `APP_SESSION_SECRET`
- UI auth defaults come from DB seed (`db/init/001_schema.sql`); acceptance runner reads `UI_USERNAME`/`UI_PASSWORD`.
- Providers: `DEFAULT_CHAT_PROVIDER`, `DEFAULT_CHAT_MODEL`, `OPENAI_*`, `ANTHROPIC_*`, `AWS_*`
- Web runtime: `WEB_PORT`, `VITE_API_PROXY_TARGET`, `VITE_ALLOWED_HOSTS`, `VITE_DEFAULT_*`
- Acceptance profile: `ACCEPTANCE_WEB_BASE_URL`, `ACCEPTANCE_API_BASE_URL`, `ACCEPTANCE_BDD_TAGS`, `BEDROCK_LIVE_*`, `PW_HEADLESS`, `PW_TIMEOUT_MS`

## Acceptance Tests (Playwright-BDD + bddgen)

Local runner:

```bash
make acceptance-sync
make acceptance-bddgen
make acceptance-typecheck
make acceptance-test
make acceptance-test-mock
```

Dockerized runner (uses compose services):

```bash
make acceptance-test-docker
```

Dockerized deterministic mocked lifecycle runner:

```bash
make acceptance-test-mock-docker
```

Live Bedrock runner (real provider call, excluded from default deterministic suite):

```bash
make acceptance-test-bedrock-live
```

Live triage continuity runner (real provider call, excluded from default deterministic suite):

```bash
make acceptance-test-triage-live
```

Deterministic mocked lifecycle runner:

```bash
make acceptance-test-mock
```

Dockerized live Bedrock runner:

```bash
make acceptance-test-bedrock-live-docker
```

Dockerized live triage continuity runner:

```bash
make acceptance-test-triage-live-docker
```

Failure screenshots are persisted to `acceptance-tests/artifacts/`.
Generated Playwright spec files are written to `acceptance-tests/.features-gen/` by `bddgen`.
Default acceptance execution filters to `not @bedrock-live` so CI-style local runs stay deterministic.

Current feature coverage:

- authentication handoff (sign-in without manual refresh)
- chat pane layout stability while creating sessions
- continue-in-new-chat continuity behavior
- tagged live Bedrock scenario for non-deterministic response validation with default model selection
- tagged live triage scenario for save-as-engram, pinning, and continuity handoff generation
- deterministic mocked lifecycle policy scenario for autosave `off` and `message_count` thresholds

## Workflow Notes (Current Validation Sequence)

Use this exact flow while you validate current local behavior end-to-end.

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

5. Run memory evaluation harness:

```bash
make eval
```

Comment: validates fact recall, cross-engram retrieval proxy, temporal ordering, and abstention behavior.

6. Manual API smoke test:

```bash
make api
```

Comment: open `http://localhost:8000/login`, sign in, then use `/ui` to run create/list/query/rehydrate from the dashboard.

For admin role validation, open `http://localhost:8000/ui/admin` and verify user list visibility.

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

7. CLI smoke test:

```bash
make cli ARGS="search --query 'local-first memory' --project-id engram-vault --top-k 3"
```

Comment: validates non-UI operator workflow from terminal.

8. Consolidation dry-run:

```bash
make consolidate ARGS="--project-id engram-vault --dry-run"
```

Comment: previews background memory maintenance without writing a new consolidation engram.

9. Inspect recent audit events:

```bash
tail -n 10 data/audit_events.jsonl
```

Comment: verifies login/logout and admin actions are being written to local audit log.

If you want to use a hashed local UI password instead of plaintext in `.env`, generate one with:

```bash
cd api
uv run python -c "from app.auth import hash_password; print(hash_password('admin123'))"
```

## High-Impact Test Workflow (Recommended)

Use this when you want to exercise continuity, persistence, and replay behavior in one pass.

1. Start full stack and acceptance baseline:

```bash
make stack-up
make acceptance-test-docker
```

2. Open [http://localhost:5174](http://localhost:5174) and sign in (`admin` / `admin123`).

3. Create three sessions in the same project:
- Session A (`openai`) for general prompts.
- Session B (`anthropic`) for comparison prompts.
- Session C (`bedrock`) to validate adapter config/error surface.

4. In Session A:
- send a prompt,
- save transcript as engram,
- copy the returned engram ID from pinned panel.

5. In Session B:
- search project engrams,
- pin the engram from Session A,
- send a prompt that requires prior context,
- verify `used_engram_ids` and source references are shown in response metadata/API.

6. Run “Continue in New Chat” from Session B and verify pinned IDs carry forward automatically.

7. Validate persistence outside UI:

```bash
make cli ARGS="search --query 'continued' --project-id engram-vault --top-k 5"
```

8. Confirm diagnostics:
- API startup logs include redacted parsed config in dev mode.
- Bedrock request errors are specific (`ValidationException`, `AccessDeniedException`, etc.), not generic failures.

## Implementation Log

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
   - `make cli ARGS=\"...\"`
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
   - `make consolidate ARGS=\"--project-id <id> [--dry-run]\"`
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

### Next Immediate Steps (One By One)

1. Phase 31 closeout: update `AGENT.md` + skills with final project-default/soft-delete/collection invariants and MCP organization contracts.
2. Phase 31 closeout: run/record full acceptance mock execution (`make acceptance-test-mock`) with new admin-memory scenarios.
3. Phase 18 follow-up: add explicit consolidation merge/grouping event semantics in timeline rendering.
4. Phase 19 design: implement project membership and scoped sharing/revocation flows with audit trails.
5. Phase 20 security gate: OIDC integration + distributed rate-limit strategy + production auth hardening tests.

## MVP API Surface

- `GET /api/v1/me`
- `GET /api/v1/users`
- `POST /api/v1/users`
- `PATCH /api/v1/users/{user_id}`
- `POST /api/v1/engrams`
- `GET /api/v1/engrams`
- `POST /api/v1/engrams/query`
- `GET /api/v1/engrams/{engram_id}/sources`
- `GET /api/v1/engrams/{engram_id}/rehydrate`
- `POST /api/v1/ingestion/text`
- `POST /api/v1/ingestion/file`
- `GET /api/v1/ingestion/documents`
- `POST /api/v1/ingestion/query`
- `POST /api/v1/ingestion/query/blended`
- `POST /api/v1/chat/sessions`
- `GET /api/v1/chat/sessions`
- `GET /api/v1/chat/sessions/{session_id}`
- `PATCH /api/v1/chat/sessions/{session_id}`
- `GET /api/v1/chat/sessions/{session_id}/messages`
- `GET /api/v1/chat/sessions/{session_id}/lifecycle-policy`
- `PATCH /api/v1/chat/sessions/{session_id}/lifecycle-policy`
- `GET /api/v1/chat/sessions/{session_id}/timeline`
- `GET /api/v1/chat/sessions/{session_id}/engrams`
- `GET /api/v1/chat/sessions/{session_id}/documents`
- `POST /api/v1/chat/sessions/{session_id}/messages`
- `POST /api/v1/chat/sessions/{session_id}/messages/stream`
- `POST /api/v1/chat/sessions/{session_id}/engrams/pin`
- `POST /api/v1/chat/sessions/{session_id}/documents/pin`
- `DELETE /api/v1/chat/sessions/{session_id}/engrams/{engram_id}`
- `DELETE /api/v1/chat/sessions/{session_id}/documents/{document_id}`
- `POST /api/v1/chat/sessions/{session_id}/save-engram`
- `POST /api/v1/chat/sessions/{session_id}/continue`
- `GET /api/v1/projects`
- `POST /api/v1/projects`
- `GET /api/v1/projects/default`
- `PATCH /api/v1/projects/default`
- `GET /api/v1/admin/memory/sessions`
- `DELETE /api/v1/admin/memory/sessions/{session_id}`
- `POST /api/v1/admin/memory/sessions/{session_id}/restore`
- `GET /api/v1/admin/memory/engrams`
- `GET /api/v1/admin/memory/engrams/{engram_id}`
- `PATCH /api/v1/admin/memory/engrams/{engram_id}`
- `POST /api/v1/admin/memory/engrams/{engram_id}/move`
- `DELETE /api/v1/admin/memory/engrams/{engram_id}`
- `POST /api/v1/admin/memory/engrams/{engram_id}/restore`
- `GET /api/v1/admin/memory/collections`
- `POST /api/v1/admin/memory/collections`
- `PATCH /api/v1/admin/memory/collections/{collection_id}`
- `DELETE /api/v1/admin/memory/collections/{collection_id}`
- `POST /api/v1/admin/memory/collections/{collection_id}/items`
- `DELETE /api/v1/admin/memory/collections/{collection_id}/items/{engram_id}`
- `POST /api/v1/mcp/tokens`
- `GET /api/v1/mcp/tokens`
- `POST /api/v1/mcp/tokens/{token_id}/revoke`
- `POST /api/v1/mcp/stream`
- `GET /.well-known/oauth-authorization-server`
- `GET /.well-known/openid-configuration`
- `GET /.well-known/oauth-protected-resource`
- `POST /oauth/register`
- `GET /oauth/authorize`
- `POST /oauth/token`
- `POST /api/v1/agent-runs`
- `GET /api/v1/agent-runs/{thread_id}`
- `POST /api/v1/agent-runs/{thread_id}/resume`
- `GET /healthz`
- `GET /api/v1/version`

## MCP Stream Usage (JSON-RPC over SSE)

Endpoint:

- `POST /api/v1/mcp/stream`
- `GET /api/v1/mcp/stream` (probe metadata)
- `HEAD /api/v1/mcp/stream` (probe health)

Auth/session requirement:

- Preferred for external tools: MCP bearer token via `Authorization: Bearer <token>`.
- Backward-compatible fallback: authenticated UI session cookies.
- Bearer and session auth resolve the same actor model and visibility checks.
- Some MCP clients preflight with `GET`/`HEAD`; these now return `200` to avoid noisy `405` logs.

## MCP Token Workflow (Admin + External Agent)

`ENGRAM_MCP_TOKEN` is now first-class and backed by persisted token records. Create token credentials as admin, store only the plaintext token client-side, and send it in the `Authorization` header for MCP calls.
For client-specific setup commands, see `docs/mcp-client-integrations.md`.

Token policy model:

- Scope: `read` or `write`.
- Optional `allowed_tools`: when empty, all tools in scope are allowed.
- Optional `allowed_project_ids`: when empty, no additional project restriction; when provided, calls are limited to those projects.
- Default token expiry: `90` days.

Create a read token (admin session):

```bash
COOKIE_JAR=/tmp/engram-admin.cookies
BASE_URL=http://localhost:8000
USERNAME=admin
PASSWORD=admin123

CSRF_TOKEN=$(
  curl -s -c "$COOKIE_JAR" "$BASE_URL/login" \
  | sed -n 's/.*name="csrf_token" value="\([^"]*\)".*/\1/p' \
  | head -n 1
)

curl -s -b "$COOKIE_JAR" -c "$COOKIE_JAR" \
  -X POST "$BASE_URL/login" \
  -H "Content-Type: application/x-www-form-urlencoded" \
  --data-urlencode "username=$USERNAME" \
  --data-urlencode "password=$PASSWORD" \
  --data-urlencode "csrf_token=$CSRF_TOKEN" >/dev/null

TOKEN_JSON=$(
  curl -s -b "$COOKIE_JAR" \
    -H "Content-Type: application/json" \
    -X POST "$BASE_URL/api/v1/mcp/tokens" \
    -d '{
      "name": "librechat-read-token",
      "scope": "read",
      "allowed_tools": [],
      "allowed_project_ids": ["engram-vault"],
      "expires_in_days": 90
    }'
)

echo "$TOKEN_JSON" | jq
MCP_TOKEN=$(echo "$TOKEN_JSON" | jq -r '.token')
```

Use bearer token for MCP stream calls (no login cookie required):

```bash
curl -sN \
  -H "Accept: text/event-stream" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $MCP_TOKEN" \
  -X POST "$BASE_URL/api/v1/mcp/stream" \
  -d '{
    "jsonrpc": "2.0",
    "id": "tools-list-bearer",
    "method": "tools/list",
    "params": {}
  }' \
  | sed -n 's/^data: //p' \
  | jq
```

Transport note:
- `POST /api/v1/mcp/stream` returns JSON when `Accept` includes `application/json`.
- It returns SSE when `Accept` explicitly prefers `text/event-stream`.
- If both are sent with equal priority (common in editor MCP clients), JSON is returned for better `initialize` compatibility.
- JSON-RPC notifications without `id` (for example `notifications/initialized`) are accepted with `202` and no body.
- `tools/list` exposes client-safe tool names with underscores (example: `chat_send_message`).
- For backward compatibility, dotted names (`chat.send_message`) are still accepted in direct calls and `tools/call`.
- `chat_save_as_engram` supports two modes:
  - session snapshot mode with `session_id`
  - conversation-only mode with `project_id` + `conversation_markdown` (no session required)

Revoke token:

```bash
TOKEN_ID=$(echo "$TOKEN_JSON" | jq -r '.token_id')
curl -s -b "$COOKIE_JAR" \
  -H "Content-Type: application/json" \
  -X POST "$BASE_URL/api/v1/mcp/tokens/$TOKEN_ID/revoke" \
  -d '{"reason":"rotation"}' | jq
```

LibreChat-style MCP bearer endpoint example:

```json
{
  "type": "sse",
  "url": "https://your-host.example.com/api/v1/mcp/stream",
  "headers": {
    "Authorization": "Bearer engram_mcp_<token_id_hex>_<secret>",
    "Accept": "text/event-stream"
  }
}
```

Python helper (bearer mode):

```python
from app.mcp.client import McpSseClient

with McpSseClient(
    base_url="http://localhost:8000",
    bearer_token="engram_mcp_<token_id_hex>_<secret>",
) as client:
    tools = client.call_tool(method="tools/list", params={}).require_result()
    print("visible tools:", [tool["name"] for tool in tools["tools"]])
```

Quick start (curl, local):

```bash
# 1) Login and persist session cookie
COOKIE_JAR=/tmp/engram-mcp.cookies
BASE_URL=http://localhost:8000
USERNAME=admin
PASSWORD=admin123

CSRF_TOKEN=$(
  curl -s -c "$COOKIE_JAR" "$BASE_URL/login" \
  | sed -n 's/.*name="csrf_token" value="\([^"]*\)".*/\1/p' \
  | head -n 1
)

curl -s -i -b "$COOKIE_JAR" -c "$COOKIE_JAR" \
  -X POST "$BASE_URL/login" \
  -H "Content-Type: application/x-www-form-urlencoded" \
  --data-urlencode "username=$USERNAME" \
  --data-urlencode "password=$PASSWORD" \
  --data-urlencode "csrf_token=$CSRF_TOKEN" \
  | head -n 1

# Expect: HTTP/1.1 303 See Other
```

```bash
# 2) Verify session auth is active
curl -s -b "$COOKIE_JAR" "$BASE_URL/api/v1/me" | jq
```

```bash
# 3) MCP initialize
curl -sN -b "$COOKIE_JAR" \
  -H "Accept: text/event-stream" \
  -H "Content-Type: application/json" \
  -X POST "$BASE_URL/api/v1/mcp/stream" \
  -d '{
    "jsonrpc": "2.0",
    "id": "init-1",
    "method": "initialize",
    "params": {
      "protocolVersion": "2025-03-26",
      "clientInfo": {"name": "curl-client", "version": "0.1.0"}
    }
  }' \
  | sed -n 's/^data: //p' \
  | jq
```

```bash
# 4) Discover tools via MCP-compatible method
curl -sN -b "$COOKIE_JAR" \
  -H "Accept: text/event-stream" \
  -H "Content-Type: application/json" \
  -X POST "$BASE_URL/api/v1/mcp/stream" \
  -d '{
    "jsonrpc": "2.0",
    "id": "tools-list-1",
    "method": "tools/list",
    "params": {}
  }' \
  | sed -n 's/^data: //p' \
  | jq
```

```bash
# 5) Persist conversation-only memory with backend auto-metadata enrichment
curl -sN -b "$COOKIE_JAR" \
  -H "Accept: text/event-stream" \
  -H "Content-Type: application/json" \
  -X POST "$BASE_URL/api/v1/mcp/stream" \
  -d '{
    "jsonrpc": "2.0",
    "id": "engram-from-conv-1",
    "method": "tools/call",
    "params": {
      "name": "engram.create_from_conversation",
      "arguments": {
        "project_id": "engram-vault",
        "conversation_markdown": "## User\nCheckout latency spiked after deploy.\n## Assistant\nLikely cache miss storm and connection pool saturation."
      }
    }
  }' \
  | sed -n 's/^data: //p' \
  | jq
```

```bash
# 6) Streaming chat tool call (observe `mcp.event` chunk/meta/done frames)
curl -sN -b "$COOKIE_JAR" \
  -H "Accept: text/event-stream" \
  -H "Content-Type: application/json" \
  -X POST "$BASE_URL/api/v1/mcp/stream" \
  -d '{
    "jsonrpc": "2.0",
    "id": "chat-stream-1",
    "method": "tools/call",
    "params": {
      "name": "chat.send_message",
      "arguments": {
        "session_id": "00000000-0000-0000-0000-000000000000",
        "content_text": "Summarize pinned engrams and list action items.",
        "stream": true
      }
    }
  }'
```

Quick start (typed Python MCP client):

```python
from app.mcp.client import McpSseClient

with McpSseClient(base_url="http://localhost:8000") as client:
    client.login_with_password(username="admin", password="admin123")

    init_result = client.call_tool(method="initialize", params={}).require_result()
    print("initialized:", init_result)

    tools_result = client.call_tool(method="tools/list", params={}).require_result()
    print("tool count:", len(tools_result["tools"]))

    save_result = client.call_tool(
        method="tools/call",
        params={
            "name": "engram.create_from_conversation",
            "arguments": {
                "project_id": "engram-vault",
                "conversation_markdown": "## User\\nNeed handoff summary\\n## Assistant\\nCaptured timeline and risks.",
            },
        },
    ).require_result()
    print("engram id:", save_result["structuredContent"]["engram"]["engram_id"])
```

Two interoperable invocation styles are supported:

1. Direct tool method (backward-compatible):

```json
{
  "jsonrpc": "2.0",
  "id": "tool-call-1",
  "method": "chat.send_message",
  "params": {
    "session_id": "00000000-0000-0000-0000-000000000000",
    "content_text": "Summarize the pinned engrams",
    "stream": true
  }
}
```

2. MCP-compatible tool call (`tools/call`) for external clients:

```json
{
  "jsonrpc": "2.0",
  "id": "tool-call-2",
  "method": "tools/call",
  "params": {
    "name": "chat.send_message",
    "arguments": {
      "session_id": "00000000-0000-0000-0000-000000000000",
      "content_text": "Summarize the pinned engrams",
      "stream": true
    }
  }
}
```

SSE framing:

- `event: jsonrpc`
- `data: <JSON-RPC frame>`

Frame types:

- success frame: `{"jsonrpc":"2.0","id":"...","result":{...}}`
- error frame: `{"jsonrpc":"2.0","id":"...","error":{"code":...,"message":"...","data":{...}}}`
- progress frame: `{"jsonrpc":"2.0","method":"mcp.event","params":{"id":"...","tool":"chat.send_message","event":"chunk|meta|done","data":{...}}}`

Compatibility methods:

- `initialize`
- `tools/list`
- `tools/call`

Core tools:

- `chat.create_session`
- `chat.list_sessions`
- `chat.get_session`
- `chat.list_messages`
- `chat.get_lifecycle_policy`
- `chat.update_lifecycle_policy`
- `chat.list_timeline`
- `chat.send_message`
- `chat.list_pinned_engrams`
- `chat.pin_engram`
- `chat.unpin_engram`
- `chat.list_pinned_documents`
- `chat.pin_document`
- `chat.unpin_document`
- `chat.list_project_documents`
- `chat.save_as_engram`
- `chat.continue_session`
- `engram.create`
- `engram.create_from_conversation`
- `engram.query`
- `engram.rehydrate`
- `engram.pin_to_session`
- `user.get_profile`
- `user.list_projects`

Typed client helpers:

- Python: `api/app/mcp/client.py` (`McpSseClient`)
- TypeScript: `web/src/api/mcpClient.ts` (`streamMcpCall`, frame parsers)

Example calls by tool group:

1. `chat.*`

```json
{
  "jsonrpc": "2.0",
  "id": "chat-create-1",
  "method": "chat.create_session",
  "params": {
    "project_id": "engram-vault",
    "title": "MCP chat session",
    "provider": "openai",
    "model_id": "gpt-4o-mini",
    "visibility_scope": "private",
    "autosave_enabled": true,
    "autosave_strategy": "interval",
    "autosave_interval_minutes": 30,
    "autosave_min_messages": 4,
    "retention_days": 30,
    "retention_max_snapshots": 25
  }
}
```

Document continuity helpers:

```json
{
  "jsonrpc": "2.0",
  "id": "chat-pin-doc-1",
  "method": "tools/call",
  "params": {
    "name": "chat.pin_document",
    "arguments": {
      "session_id": "00000000-0000-0000-0000-000000000000",
      "document_id": "11111111-1111-1111-1111-111111111111"
    }
  }
}
```

```json
{
  "jsonrpc": "2.0",
  "id": "chat-list-pins-1",
  "method": "tools/call",
  "params": {
    "name": "chat.list_pinned_documents",
    "arguments": {
      "session_id": "00000000-0000-0000-0000-000000000000"
    }
  }
}
```

2. `engram.*`

```json
{
  "jsonrpc": "2.0",
  "id": "engram-query-1",
  "method": "engram.query",
  "params": {
    "query": "incident mitigation",
    "project_id": "engram-vault",
    "top_k": 5
  }
}
```

```json
{
  "jsonrpc": "2.0",
  "id": "engram-create-conversation-1",
  "method": "tools/call",
  "params": {
    "name": "engram.create_from_conversation",
    "arguments": {
      "project_id": "engram-vault",
      "conversation_markdown": "## User\nDatabase latency spiked.\n## Assistant\nLikely cache miss storm after deploy."
    }
  }
}
```

Expected response highlights:

- `result.engram`: persisted engram payload.
- `result.enrichment_report.enrichment_applied`: whether empty metadata fields were auto-filled.
- `result.enrichment_report.auto_tags` / `auto_keywords`: deterministic derived values.

3. `user.*`

```json
{
  "jsonrpc": "2.0",
  "id": "user-profile-1",
  "method": "user.get_profile",
  "params": {}
}
```

## Auto Metadata Enrichment (Phase 29)

- Scope: applies to all engram create paths through centralized repository logic (`api/app/repository.py`).
- Behavior: fill-empty-only for `abstract`, `tags`, and `keywords`; non-empty caller values are never overwritten.
- Strategy: deterministic local parsing in v1 (no provider call required).
- Traceability: enrichment details are stored in `engram_json.auto_metadata` with schema version and origin.
- MCP-first path: `engram.create_from_conversation` accepts transcript markdown and returns enrichment report fields.
- API compatibility: runtime endpoints remain unchanged (`POST /api/v1/engrams` keeps backward-compatible payload support).

## CLI Workflow (Local)

Use CLI mode when you want a terminal-only path (no browser).

1. Create a minimal upload payload:

```bash
cat > /tmp/engram.json <<'JSON'
{
  "project_id": "engram-vault",
  "thread_id": "cli-run-001",
  "title": "CLI upload sample",
  "abstract": "Uploaded from local CLI.",
  "detailed_summary_markdown": "Sample summary for CLI upload testing.",
  "tags": ["cli"],
  "keywords": ["upload", "local"]
}
JSON
```

2. Upload engram:

```bash
make cli ARGS="upload --file /tmp/engram.json"
```

3. Search engrams:

```bash
make cli ARGS="search --query 'uploaded from local cli' --project-id engram-vault --top-k 5"
```

4. Rehydrate engram by id:

```bash
make cli ARGS="rehydrate --engram-id <engram_uuid>"
```

5. Run consolidation maintenance (dry-run or create):

```bash
make consolidate ARGS="--project-id engram-vault --dry-run"
make consolidate ARGS="--project-id engram-vault"
```

## Test Suite

Backend tests live under `api/tests`:

- `test_embedding.py`: deterministic embedding behavior.
- `test_repository_helpers.py`: retrieval text and vector literal helpers.
- `test_api_unit.py`: endpoint behavior with repository function mocking.
- `test_api_integration.py`: end-to-end API roundtrip + sources endpoint checks.
- `test_ui_auth.py`: login/logout/session-protected UI + CSRF + rate-limit + audit checks.
- `test_user_rbac.py`: user management and role-based access control checks.
- `test_agent_workflow.py`: LangGraph checkpoint/resume + auto-persist + snapshot checks.
- `test_eval_harness.py`: scenario-based evaluation harness pass/fail checks.
- `test_cli.py`: upload/search/rehydrate CLI command behavior.
- `test_consolidation.py`: consolidation snapshot generation and safety checks.
- `test_chat_lifecycle_policy.py`: autosave normalization/trigger rules and retention pruning heuristics.
- `test_chat_repository.py`: chat session/message, pinning, and lifecycle policy persistence checks.
- `test_chat_api_integration.py`: end-to-end chat API lifecycle/continuity including policy and timeline routes.
- `test_chat_service.py`: chat orchestration with autosave execution and retention pruning behavior.
- `test_mcp_api_integration.py`: MCP tool contract coverage including lifecycle policy/timeline tools and `engram.create_from_conversation`.
- `conftest.py`: DB fixture, schema bootstrap, and cleanup.
- `test_engram_enrichment.py`: deterministic metadata derivation and fill-empty-only/non-overwrite contract checks.

Frontend unit/component tests live under `web/src/**/*.test.ts(x)`:

- `src/api/auth.test.ts`
- `src/styles/themeMode.test.ts`
- `src/utils/chat.test.ts`
- `src/utils/sse.test.ts`
- `src/components/LoginView.test.tsx`
- `src/components/ChatPanel.test.tsx`
- `src/components/SaveEngramModal.test.tsx`
- `src/components/SessionSidebar.test.tsx`

Acceptance tests live under `acceptance-tests`:

- `features/authentication.feature`: login handoff regression.
- `features/session-layout.feature`: pane height stability + continuation behavior.
- `features/bedrock-live.feature`: tagged non-deterministic Bedrock live-provider flow.
- `features/triage-live.feature`: tagged non-deterministic triage continuity flow (save/pin/continue/handoff).
- `features/lifecycle-autosave-mock.feature`: deterministic mocked lifecycle autosave policy behavior coverage.
- `features/engram-auto-metadata-mock.feature`: deterministic mocked MCP conversation-only persistence + auto-metadata behavior.
- `src/steps/*.ts`: Playwright step bindings.
- `src/support/*.ts`: shared world/env/hooks.

Notes:

- Integration tests are marked with `@pytest.mark.integration` and configured in `api/pyproject.toml`.
- If the DB is unavailable, integration tests are skipped with a clear reason.
- `make eval` runs local memory-quality scenarios and exits non-zero if any case fails.
- `make web-check` runs frontend lint/test/build.
- `make acceptance-bddgen` regenerates Playwright specs from `.feature` files.
- `make acceptance-typecheck` validates acceptance TypeScript.
- `make acceptance-test-docker` runs Gherkin acceptance tests against dockerized API+web.
- `make acceptance-test-mock` runs only `@mock` deterministic acceptance scenarios.
- `make acceptance-test-mock-docker` runs `@mock` deterministic acceptance scenarios against dockerized API+web.
- `make acceptance-test-bedrock-live` runs only `@bedrock-live` scenarios against live Bedrock.
- `make acceptance-test-triage-live` runs only `@triage-live` incident triage continuity scenarios.

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

### Milestone 4 (Completed)

- Added CSRF protection for login/logout UI forms
- Added optional hashed-password authentication path (`UI_DEMO_PASSWORD_HASH`)
- Added source-inspection endpoint and dashboard workflow
- Added multi-user auth model and role-based access controls

### Milestone 5 (Completed)

- Added local eval harness:
  - fact recall
  - cross-engram reasoning proxy
  - temporal updates
  - abstention checks

### Milestone 6 (Completed)

- Added local CLI workflow:
  - upload engrams from JSON
  - query/search from terminal
  - rehydrate bundles from terminal

### Milestone 7 (Completed)

- Retrieval quality improvements:
  - reranking (dense + lexical overlap)
  - citation-packing in rehydration context

### Milestone 8 (Completed)

- Memory maintenance improvements:
  - background consolidation jobs

### Milestone 9 (In Progress)

- Completed local baseline:
  - local audit event logging
  - login rate limiting + lockout guard
- Remaining production auth/security hardening:
  - oauth/oidc integration
  - centralized audit-event pipeline
  - distributed auth rate limits and lockout policy

### Milestone 10 (Completed)

- Unified roadmap phase 10 schema and repository layer:
  - chat/session/pinning tables
  - engram ownership and visibility fields
  - visibility-aware repository filtering

### Milestone 11 (Completed)

- Provider adapter layer completed:
  - OpenAI adapter with normalized request/response mapping
  - Anthropic adapter with normalized request/response mapping
  - Bedrock adapter with normalized request/response mapping
  - provider registry and configuration contract

### Milestone 12 (Completed)

- Chat API and continuity layer completed:
  - session/message endpoints
  - stream endpoint for incremental assistant output
  - save-as-engram and continue-session flows
  - context assembly with `used_engram_ids` and `source_references`

### Milestone 13 (Completed)

- MCP HTTP stream layer completed:
  - JSON-RPC over SSE transport
  - chat/engram/user tool routing
  - auth/visibility parity with REST APIs
  - structured error frames and correlation IDs

### Milestone 14 (Completed)

- React chat UI layer completed:
  - session list/create workflow with provider and model selectors
  - streaming transcript with retry/error handling
  - pin/save/continue controls and engram ID copy workflows
  - frontend test baseline (`vitest`) and `make web-check`

### Milestone 15 (Completed)

- UI hardening and browser integration testing:
  - Playwright-BDD acceptance baseline for login/chat/continuation flows
  - tagged live Bedrock and triage continuity scenarios
  - markdown rendering + stream parsing reliability fixes
  - dark/light theming and sidebar UX hardening

### Milestone 16 (In Progress)

- MCP developer-experience and external interoperability:
  - MCP compatibility methods: `initialize`, `tools/list`, `tools/call`
  - typed Python and TypeScript MCP JSON-RPC/SSE client helpers
  - contract tests for compatibility envelopes and stream-call behavior
- Remaining in milestone:
  - CLI smoke utility (`engram-cli mcp-call`) deferred to the next pass

### Milestone 17 (Completed)

- RAG-ready ingestion and retrieval expansion:
  - file/document upload ingestion
  - deterministic chunking + metadata pipeline
  - retrieval blending between chat snapshots and document chunks
  - session-level document pin/unpin and continuation carry-forward

### Milestone 18 (In Progress)

- Memory lifecycle controls (core shipped):
  - session-level autosave strategy options (`off`/`interval`/`message_count`)
  - retention windows and bounded snapshot pruning
  - lifecycle policy and timeline APIs + MCP tool coverage
  - lifecycle timeline surfaced in chat UI
- Remaining in milestone:
  - richer consolidation merge/group timeline semantics

### Milestone 29 (Completed)

- Optional auto-metadata enrichment for engram persistence:
  - deterministic fill-empty-only derivation for `abstract`, `tags`, and `keywords`
  - repository-level centralization so all create paths share the same behavior
  - MCP `engram.create_from_conversation` for conversation-only persistence requests
  - enrichment trace persisted under `engram_json.auto_metadata`

### Milestone 19 (Planned)

- Collaboration and sharing:
  - project membership model
  - scoped sharing/revocation workflows
  - audit-visible memory collaboration events

### Milestone 20 (Planned)

- Production security hardening:
  - OIDC integration
  - distributed auth rate limiting
  - centralized audit and security regression gates

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
- [x] I can manage users and enforce admin-only routes with role checks.
- [x] I can run local eval scenarios for fact/cross/temporal/abstention behavior.
- [x] I can upload/search/rehydrate engrams from terminal using local CLI commands.
- [x] Query and rehydration quality include reranking and citation-packing improvements.
- [x] I can run local consolidation maintenance jobs (dry-run or persist) per project.
- [x] I can enforce local login rate limits and inspect local audit events.
- [x] A newcomer can run the system locally using this README alone.
- [ ] The same stored engram can be reused with different LLM providers later.
