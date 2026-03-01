# Engram Vault - Implementation Log

> Chronological log of implementation changes by date and pass.
> See [README](../README.md) for project overview. See [migration/checkpoints/checkpoint.md](../migration/checkpoints/checkpoint.md) for milestone summary.

---

## Implementation Log

### 2026-03-01 (Phase 24 kickoff: link graph schema + repository foundation)

1. Added graph persistence schema foundation in `db/init/001_schema.sql`:
   - `engram_links` directed edge table with:
     - `source_engram_id`, `target_engram_id`, `relation_type`
     - `weight`, `temporal_weight`, `confidence`
     - `origin`, `status`, `evidence_json`
     - `created_by_user_id`, `last_reinforced_at`, timestamps
   - optional lifecycle/audit table `engram_link_events`.
2. Added index coverage for graph query paths:
   - project/status/relation/recency lookup
   - source-node and target-node weighted traversal ordering
   - status + reinforcement recency lookups.
3. Added typed link models in `internal/models/engram_link.go`:
   - relation type enum + parser
   - origin enum + parser
   - status enum + parser
   - link and traversal record contracts.
4. Added repository baseline in `internal/repository/engram_links.go`:
   - `CreateEngramLink`
   - `ListEngramLinks`
   - `UpdateEngramLink`
   - `ArchiveEngramLink`
   - `TraverseEngramLinks` (depth-limited, cycle-safe expansion)
5. Added visibility-safe graph access semantics:
   - source/target engram membership/visibility checks on reads.
   - write access checks for owner/editor/admin in graph mutations.
6. Added tests:
   - `internal/models/engram_link_test.go`
   - `internal/repository/engram_links_test.go`
7. Validation:
   - `go test ./internal/models ./internal/repository -count=1`

### 2026-03-01 (Phase 23 closeout: EvalOps + prompt/policy governance)

1. Added governance version constants and runtime config:
   - `internal/governance/versions.go` now defines:
     - `chat-prompt-policy-v1`
     - `mcp-tool-policy-v1`
     - `eval-suite-v1`
   - `internal/config/config.go` now supports:
     - `CHAT_PROMPT_POLICY_VERSION`
     - `MCP_TOOL_POLICY_VERSION`
     - `EVAL_SUITE_VERSION`
2. Exposed policy/eval metadata in runtime contracts:
   - chat send response + stream meta/done payloads now include `prompt_policy_version`.
   - MCP `initialize` now includes `policy.tool_policy_version` and `policy.eval_suite_version`.
   - `/api/v1/version` now includes:
     - `chat_prompt_policy_version`
     - `mcp_tool_policy_version`
     - `eval_suite_version`
3. Added deterministic EvalOps suite in Go:
   - `internal/evalops` with case runner, delta-gate evaluator, history storage, and trend reporting.
   - suite dimensions:
     - continuity
     - citation trust
     - memory drift
4. Added EvalOps CLI runner:
   - `cmd/evalops/main.go`
   - writes:
     - `data/evals/latest.json`
     - `data/evals/history.jsonl`
     - `data/evals/trend-report.md`
5. Added baseline and gate controls:
   - baseline file: `evals/baselines/eval-suite-v1.json`
   - delta thresholds:
     - overall >= `-0.03`
     - dimension >= `-0.05`
   - compares current run against baseline and previous run.
6. Integrated quality/release automation:
   - Makefile:
     - `make eval`
     - `make eval-report`
     - `make check` now includes eval gate.
   - CI:
     - Go backend job now runs `make eval`.
7. Added/updated test coverage:
   - `internal/evalops/runner_test.go`
   - `internal/evalops/gate_test.go`
   - `internal/evalops/storage_report_test.go`
   - updated config/router/MCP tests for new governance metadata fields.
8. Docs consolidation and alignment:
   - added `docs/evalops-governance-v1.md`
   - updated `README.md`, `docs/testing-guide.md`, `docs/env-reference.md`, `docs/api-reference.md`, `docs/mcp-guide.md`, `Plan.md`, `todo.md`, and `migration/checkpoints/checkpoint.md`.
9. Validation:
   - `go test ./internal/evalops ./cmd/evalops ./internal/config ./internal/api ./internal/mcp ./cmd/api -count=1`
   - `make eval`

### 2026-03-01 (Phase 22 closeout: release automation + deployment profiles)

1. Split CI into explicit release stages:
   - `.github/workflows/ci.yml` now runs backend, web, deterministic acceptance, and release-smoke jobs.
2. Added optional gated live-provider release gate:
   - CI supports `workflow_dispatch` with `run_live_provider=true` to run Bedrock/triage live acceptance checks.
3. Standardized compose profiles:
   - `docker-compose.yml` now declares `dev`, `acceptance`, and `release-smoke`.
   - added `release-smoke` probe service for API/web readiness and observability endpoint checks.
4. Added Makefile release automation targets:
   - `release-smoke-docker`
   - `release-gate`
   - `release-live-provider-gate`
5. Added versioned release operational docs:
   - `docs/release-checklist-v1.md`
   - `docs/release-rollback-runbook-v1.md`
6. Consolidated docs/roadmap alignment:
   - `README.md`, `docs/testing-guide.md`, `docs/user-workflows.md`, `docs/go-rollout-playbook.md`, `Plan.md`, `todo.md`, and checkpoint trackers.
7. Validation:
   - `docker compose --profile acceptance config`
   - `docker compose --profile release-smoke config`
   - `go test ./... -count=1`

### 2026-03-01 (Phase 21 closeout: chat reliability + observability expansion)

1. Added provider reliability primitives:
   - `internal/chat/provider_circuit.go` implements transient-failure circuit breaker policy (threshold + cooldown).
   - `internal/chat/provider_fallback.go` implements ordered fallback provider/model candidate strategy.
2. Wired reliability policies into chat send/stream execution:
   - `internal/chat/service.go` now executes provider attempts with fallback on transient failures and circuit-open short-circuiting.
   - persisted assistant metadata now uses actual serving provider/model (`internal/chat/message_runtime.go`).
3. Added lifecycle trace hooks:
   - chat service now emits prepare/provider/persist/lifecycle trace samples for send and stream paths.
   - `internal/chat/observability.go` defines chat observability contracts.
4. Expanded `/api/v1/metrics` payload:
   - `internal/api/request_observability.go` now includes:
     - `provider_failures` counters by operation/provider/error code,
     - `stream_health` counters and latency/chunk aggregates by operation/provider/outcome,
     - `lifecycle_traces` counters by operation/stage/error class.
5. Wired runtime dependencies for both REST and MCP chat paths:
   - `cmd/api/chat_runtime.go`, `cmd/api/main.go`, `cmd/api/mcp_message_send_adapter.go`.
6. Added/updated regression tests:
   - `internal/chat/provider_circuit_test.go`
   - `internal/chat/provider_fallback_test.go`
   - `internal/chat/service_test.go`
   - `internal/api/request_observability_test.go`
   - `cmd/api/chat_runtime_test.go`
7. Validation:
   - `go test ./internal/chat -count=1`
   - `go test ./cmd/api ./internal/api ./internal/chat -count=1`

### 2026-02-28 (Phase 19: collaboration, sharing, and membership-safe access)

1. Added collaboration schema and persistence:
   - `project_members` table + active/revoked indexes and owner-membership backfill in `db/init/001_schema.sql`.
   - `project_audit_events` table + query indexes.
2. Added repository support:
   - `internal/repository/project_membership.go` (membership CRUD + actor role resolution).
   - `internal/repository/project_audit.go` (audit insert/list).
   - `internal/repository/engram_share.go` (share-target read + visibility update).
3. Enforced membership-gated visibility reads across core paths:
   - introduced shared SQL helpers in `internal/repository/access_policy_sql.go`.
   - updated engram/chat/chat_message/chat_pinning/document/rehydration read predicates for owner/admin/membership policy.
4. Added Phase 19 project service behavior:
   - member list/add/update/remove methods with owner/admin-only management checks.
   - audit list method with owner/admin policy.
   - engram `share`/`unshare` methods with owner/editor/viewer/admin role matrix.
   - hardened `ResolveProjectIDForWrite` for inaccessible existing project IDs and viewer write denial.
5. Added REST API routes:
   - project member CRUD and audit list in `internal/api/projects_api.go`.
   - `POST /api/v1/engrams/{engram_id}/share` and `/unshare` in session-auth routing.
6. Added MCP parity:
   - new tools `project.member_list`, `project.member_add`, `project.member_update`, `project.member_remove`, `engram.share`, `engram.unshare`.
   - updated MCP catalog metadata, dispatch wiring, and token project-scope policy maps.
7. Added audit emission for pin/unpin:
   - `chat.pin_engram` and `chat.unpin_engram` now write DB audit rows.
8. Extended admin UI:
   - `AdminMemoryPage` now includes project member management and project audit timeline panels.
   - extended web API/type clients for member/audit routes and share/unshare helpers.
9. Added/updated tests:
   - fixed placeholder/index regressions in repository query builders and updated SQL expectation tests.
   - added API tests for project member/audit handlers and engram share/unshare handlers.
   - extended projects service tests for new member/share authorization behavior.
   - extended web API and admin UI tests for member/audit workflows.
10. Validation:
   - `go test ./internal/api ./internal/projects ./internal/repository ./internal/mcp ./cmd/api`
   - `npm test -- --run src/api/projects.test.ts src/api/memoryAdmin.test.ts src/components/AdminMemoryPage.test.tsx` (from `web/`)
   - `npm run build` (from `web/`)

### 2026-02-22 (Go migration CP1: module scaffold + config parity)

1. Started phased Go migration per `migration/migrate.md` with a dedicated checkpoint tracker:
   - `checkpoint-go-migration.md`
2. Added Go module scaffold and first internal package:
   - `go.mod`
   - `internal/config/config.go`
3. Ported config behavior from Python:
   - production hardening validation parity (`APP_SESSION_SECRET`, `MCP_TOKEN_PEPPER`, `OAUTH_CLIENT_SECRET_PEPPER`, `UI_DEMO_PASSWORD`, protected OAuth registration)
   - debug snapshot redaction parity for sensitive fields
   - environment helpers for dev/prod gating
4. Ported config tests from `api/tests/test_config_settings.py`:
   - `internal/config/config_test.go`
5. Executed migrated Go tests one-by-one:
   - `go test ./internal/config -run '^TestBuildDebugSettingsSnapshotRedactsSecrets$' -v`
   - `go test ./internal/config -run '^TestShouldLogSettingsOnlyForDevModes$' -v`
   - `go test ./internal/config -run '^TestProductionSettingsRejectInsecureDefaults$' -v`
   - `go test ./internal/config -run '^TestProductionSettingsAcceptHardenedValues$' -v`
6. CodeScene pre-commit safeguard:
   - `pre_commit_code_health_safeguard(git_repository_path=/Users/mukundhan/Projects/engram)`
   - result: `quality_gates=passed`
   - findings: none

### 2026-02-22 (Go migration CP2: DB transaction/bootstrap parity)

1. Added Go DB baseline package:
   - `internal/db/db.go`
2. Ported DB runtime behavior from Python:
   - transaction commit/rollback semantics via `WithTransaction`
   - schema bootstrap execution via `EnsureSchemaInitialized`
   - production bootstrap-admin credential hardening parity
   - pgx pool wiring helpers (`NewPool`, `Ping`, `PgxPoolBeginner`)
3. Ported DB tests from `api/tests/test_db.py`:
   - `internal/db/db_test.go`
4. Executed migrated Go tests one-by-one:
   - `go test ./internal/db -run '^TestWithTransactionCommitsOnSuccess$' -v`
   - `go test ./internal/db -run '^TestWithTransactionRollsBackOnError$' -v`
   - `go test ./internal/db -run '^TestEnsureSchemaInitializedExecutesSchemaSQL$' -v`
   - `go test ./internal/db -run '^TestHardenBootstrapAdminCredentialsUpdatesDefaultHash$' -v`
   - `go test ./internal/db -run '^TestHardenBootstrapAdminCredentialsSkipsNonDefaultHash$' -v`
5. Package-level verification:
   - `go test ./internal/config ./internal/db`
6. CodeScene pre-commit safeguard:
   - `pre_commit_code_health_safeguard(git_repository_path=/Users/mukundhan/Projects/engram)`
   - result: `quality_gates=passed`
   - findings: none

### 2026-02-22 (Go migration CP3: API entrypoint + health/version parity)

1. Added Go API entrypoint:
   - `cmd/api/main.go`
2. Added initial Chi router skeleton:
   - `internal/api/router.go`
   - `GET /healthz` parity response (`{\"status\":\"ok\"}`)
   - `GET /api/v1/version` parity response (`semantic_version`, `release`, `commit_id`)
3. Ported basic API unit tests from `api/tests/test_api_unit.py`:
   - `internal/api/router_test.go`
4. Executed migrated tests one-by-one:
   - `go test ./internal/api -run '^TestHealthz$' -v`
   - `go test ./internal/api -run '^TestVersionEndpoint$' -v`
5. Full Go verification:
   - `go test ./...`
6. Code health quality pass (file-level checks before commit):
   - refactored `internal/config/config.go` to reduce nested/complex default handling and redaction logic.
   - refactored `internal/db/db.go` `EnsureSchemaInitialized` signature to use options struct.
   - refactored `internal/db/db_test.go` into table-driven tests to reduce duplication.
   - file-level CodeScene scores:
     - `/Users/mukundhan/Projects/engram/cmd/api/main.go` -> `10.0`
     - `/Users/mukundhan/Projects/engram/internal/api/router.go` -> `10.0`
     - `/Users/mukundhan/Projects/engram/internal/api/router_test.go` -> `10.0`
     - `/Users/mukundhan/Projects/engram/internal/config/config.go` -> `9.68`
     - `/Users/mukundhan/Projects/engram/internal/config/config_test.go` -> `10.0`
     - `/Users/mukundhan/Projects/engram/internal/db/db.go` -> `10.0`
     - `/Users/mukundhan/Projects/engram/internal/db/db_test.go` -> `9.61`
   - attempted non-code file checks for `/Users/mukundhan/Projects/engram/checkpoint-go-migration.md` and `/Users/mukundhan/Projects/engram/go.mod`; CodeScene reported unsupported file types (`.md`, `.mod`).
7. CodeScene pre-commit safeguard:
   - `pre_commit_code_health_safeguard(git_repository_path=/Users/mukundhan/Projects/engram)`
   - result: `quality_gates=passed`
   - improvements observed:
     - `internal/config/config.go`: fixed prior `Complex Method` and `Deep, Nested Complexity` findings.
     - `internal/db/db.go`: fixed prior `Excess Number of Function Arguments` finding.
     - `internal/db/db_test.go`: fixed prior duplication finding; remaining findings are low-severity test complexity smells.

### 2026-02-22 (Go migration CP4: auth parity)

1. Added Go auth primitives:
   - `internal/auth/password.go`
   - `GenerateCSRFToken` with URL-safe random encoding.
   - `HashPassword` + `VerifyPassword` with Python-compatible `pbkdf2_sha256` format and 390000 iterations.
2. Ported auth tests:
   - `internal/auth/password_test.go`
3. Executed migrated tests one-by-one:
   - `go test ./internal/auth -run '^TestHashPasswordWithProvidedSaltProducesExpectedFormat$' -v`
   - `go test ./internal/auth -run '^TestVerifyPasswordRoundTrip$' -v`
   - `go test ./internal/auth -run '^TestVerifyPasswordMatchesBootstrapAdminHash$' -v`
   - `go test ./internal/auth -run '^TestVerifyPasswordRejectsInvalidEncodings$' -v`
   - `go test ./internal/auth -run '^TestGenerateCSRFTokenIsURLSafeAndRandom$' -v`
4. Full Go verification:
   - `go test ./...`
5. File-level CodeScene checks (all current Go migration files before commit):
   - `/Users/mukundhan/Projects/engram/cmd/api/main.go` -> `10.0`
   - `/Users/mukundhan/Projects/engram/internal/api/router.go` -> `10.0`
   - `/Users/mukundhan/Projects/engram/internal/api/router_test.go` -> `10.0`
   - `/Users/mukundhan/Projects/engram/internal/config/config.go` -> `9.68`
   - `/Users/mukundhan/Projects/engram/internal/config/config_test.go` -> `10.0`
   - `/Users/mukundhan/Projects/engram/internal/db/db.go` -> `10.0`
   - `/Users/mukundhan/Projects/engram/internal/db/db_test.go` -> `9.61`
   - `/Users/mukundhan/Projects/engram/internal/auth/password.go` -> `10.0`
   - `/Users/mukundhan/Projects/engram/internal/auth/password_test.go` -> `9.68`
6. CodeScene pre-commit safeguard:
   - `pre_commit_code_health_safeguard(git_repository_path=/Users/mukundhan/Projects/engram)`
   - result: `quality_gates=passed`
   - findings: none

### 2026-02-22 (Go migration CP5: user model + repository parity)

1. Added Go user model primitives:
   - `internal/models/user.go`
   - `UserRole`, role parser, `UserRecord`, `UserAuthRecord`
2. Added Go user repository baseline:
   - `internal/repository/user.go`
   - `GetUserAuthRecord`, `GetUserAuthRecordByID`, `ListUsers`, `CreateUser`, `UpdateUser`
   - duplicate username mapping to `ErrUsernameExists`
3. Ported repository tests from `api/tests/test_user_repository.py`:
   - `internal/repository/user_test.go`
4. Executed migrated tests one-by-one:
   - `go test ./internal/repository -run '^TestGetUserAuthRecordLooksUpUsername$' -v`
   - `go test ./internal/repository -run '^TestListUsersReturnsRecords$' -v`
   - `go test ./internal/repository -run '^TestCreateUserReturnsCreatedUser$' -v`
   - `go test ./internal/repository -run '^TestCreateUserReturnsUsernameExistsOnDuplicate$' -v`
   - `go test ./internal/repository -run '^TestUpdateUserReturnsNilWhenNotFound$' -v`
   - `go test ./internal/repository -run '^TestUpdateUserAppliesProvidedFields$' -v`
5. Full Go verification:
   - `go test ./...`
6. File-level CodeScene checks (all current Go migration files before commit):
   - `/Users/mukundhan/Projects/engram/cmd/api/main.go` -> `10.0`
   - `/Users/mukundhan/Projects/engram/internal/api/router.go` -> `10.0`
   - `/Users/mukundhan/Projects/engram/internal/api/router_test.go` -> `10.0`
   - `/Users/mukundhan/Projects/engram/internal/auth/password.go` -> `10.0`
   - `/Users/mukundhan/Projects/engram/internal/auth/password_test.go` -> `9.68`
   - `/Users/mukundhan/Projects/engram/internal/config/config.go` -> `9.68`
   - `/Users/mukundhan/Projects/engram/internal/config/config_test.go` -> `10.0`
   - `/Users/mukundhan/Projects/engram/internal/db/db.go` -> `10.0`
   - `/Users/mukundhan/Projects/engram/internal/db/db_test.go` -> `9.61`
   - `/Users/mukundhan/Projects/engram/internal/models/user.go` -> `10.0`
   - `/Users/mukundhan/Projects/engram/internal/repository/user.go` -> `9.38`
   - `/Users/mukundhan/Projects/engram/internal/repository/user_test.go` -> `10.0`
7. CodeScene pre-commit safeguard:
   - `pre_commit_code_health_safeguard(git_repository_path=/Users/mukundhan/Projects/engram)`
   - result: `quality_gates=passed`
   - findings: none

### 2026-02-22 (Go migration CP6: embeddings local + fallback parity)

1. Added Go embeddings baseline package:
   - `internal/embeddings/errors.go`
   - `internal/embeddings/local.go`
   - `internal/embeddings/service.go`
2. Ported embeddings tests from Python (`api/tests/test_embedding.py`, `api/tests/test_embeddings_service.py`):
   - `internal/embeddings/local_test.go`
   - `internal/embeddings/service_test.go`
3. Executed migrated tests one-by-one:
   - `go test ./internal/embeddings -run '^TestEmbedTextLocalIsDeterministicAndFixedDim$' -v`
   - `go test ./internal/embeddings -run '^TestEmbedTextLocalHandlesEmptyText$' -v`
   - `go test ./internal/embeddings -run '^TestEmbedTextLocalRejectsInvalidDim$' -v`
   - `go test ./internal/embeddings -run '^TestEmbeddingServiceFallsBackToLocalProvider$' -v`
   - `go test ./internal/embeddings -run '^TestEmbeddingServiceEmbedManyUsesProviderIDFromActiveProvider$' -v`
4. Full Go verification:
   - `go test ./...`
5. File-level CodeScene checks (all current Go migration files before commit):
   - `/Users/mukundhan/Projects/engram/cmd/api/main.go` -> `10.0`
   - `/Users/mukundhan/Projects/engram/internal/api/router.go` -> `10.0`
   - `/Users/mukundhan/Projects/engram/internal/api/router_test.go` -> `10.0`
   - `/Users/mukundhan/Projects/engram/internal/auth/password.go` -> `10.0`
   - `/Users/mukundhan/Projects/engram/internal/auth/password_test.go` -> `9.68`
   - `/Users/mukundhan/Projects/engram/internal/config/config.go` -> `9.68`
   - `/Users/mukundhan/Projects/engram/internal/config/config_test.go` -> `10.0`
   - `/Users/mukundhan/Projects/engram/internal/db/db.go` -> `10.0`
   - `/Users/mukundhan/Projects/engram/internal/db/db_test.go` -> `9.61`
   - `/Users/mukundhan/Projects/engram/internal/embeddings/errors.go` -> `10.0`
   - `/Users/mukundhan/Projects/engram/internal/embeddings/local.go` -> `10.0`
   - `/Users/mukundhan/Projects/engram/internal/embeddings/local_test.go` -> `9.68`
   - `/Users/mukundhan/Projects/engram/internal/embeddings/service.go` -> `9.09`
   - `/Users/mukundhan/Projects/engram/internal/embeddings/service_test.go` -> `10.0`
   - `/Users/mukundhan/Projects/engram/internal/models/user.go` -> `10.0`
   - `/Users/mukundhan/Projects/engram/internal/repository/user.go` -> `9.38`
   - `/Users/mukundhan/Projects/engram/internal/repository/user_test.go` -> `10.0`
6. CodeScene pre-commit safeguard:
   - `pre_commit_code_health_safeguard(git_repository_path=/Users/mukundhan/Projects/engram)`
   - result: `quality_gates=passed`
   - findings: none

### 2026-02-22 (Go migration CP7: engram repository helper/query parity)

1. Added engram model primitives required for repository helper migration:
   - `internal/models/engram.go`
2. Added engram repository helper baseline:
   - `internal/repository/engram.go`
   - ported helper logic from Python repository:
     - retrieval text composition
     - vector literal formatting
     - lexical overlap + combined rerank scoring
     - engram query `WHERE` clause builder
     - citation/decision/open-question formatting
     - rehydration context markdown composition
     - compact summary resolution + assistant excerpt extraction
     - engram JSON payload serialization
3. Ported helper-focused tests from:
   - `api/tests/test_repository_helpers.py`
   - `api/tests/test_repository_unit.py`
   - new files:
     - `internal/repository/engram_helpers_test.go`
     - `internal/repository/engram_unit_test.go`
4. Executed migrated tests one-by-one:
   - `go test ./internal/repository -run '^TestVectorLiteralFormat$' -v`
   - `go test ./internal/repository -run '^TestBuildRetrievalTextUsesOverride$' -v`
   - `go test ./internal/repository -run '^TestBuildRetrievalTextFallbackComposesFields$' -v`
   - `go test ./internal/repository -run '^TestLexicalOverlapScorePrefersMatchingTerms$' -v`
   - `go test ./internal/repository -run '^TestCombinedRankScoreUsesDenseAndLexicalSignals$' -v`
   - `go test ./internal/repository -run '^TestPackCitationsDeduplicatesURLs$' -v`
   - `go test ./internal/repository -run '^TestResolveCompactSummaryUsesDetailedForGenericChatSnapshot$' -v`
   - `go test ./internal/repository -run '^TestExtractDetailedExcerptPrefersAssistantSection$' -v`
   - `go test ./internal/repository -run '^TestBuildEngramQueryWhereIncludesAllFilters$' -v`
   - `go test ./internal/repository -run '^TestRerankByCombinedScorePrefersLexicalOverlap$' -v`
   - `go test ./internal/repository -run '^TestFormatCitationsTruncatesAndStripsNewlines$' -v`
   - `go test ./internal/repository -run '^TestFormatCitationsReturnsDefaultForEmptyList$' -v`
   - `go test ./internal/repository -run '^TestFormatDecisionsFormatsEntriesAndDefaults$' -v`
   - `go test ./internal/repository -run '^TestFormatOpenQuestionsFormatsEntriesAndDefaults$' -v`
   - `go test ./internal/repository -run '^TestBuildRehydrationContextMarkdownIncludesExpectedSections$' -v`
   - `go test ./internal/repository -run '^TestBuildEngramJSONPayloadSerializesReportAndSourceSessionID$' -v`
5. Full Go verification:
   - `go test ./...`
6. File-level CodeScene checks (all current Go migration files before commit):
   - `/Users/mukundhan/Projects/engram/cmd/api/main.go` -> `10.0`
   - `/Users/mukundhan/Projects/engram/internal/api/router.go` -> `10.0`
   - `/Users/mukundhan/Projects/engram/internal/api/router_test.go` -> `10.0`
   - `/Users/mukundhan/Projects/engram/internal/auth/password.go` -> `10.0`
   - `/Users/mukundhan/Projects/engram/internal/auth/password_test.go` -> `9.68`
   - `/Users/mukundhan/Projects/engram/internal/config/config.go` -> `9.68`
   - `/Users/mukundhan/Projects/engram/internal/config/config_test.go` -> `10.0`
   - `/Users/mukundhan/Projects/engram/internal/db/db.go` -> `10.0`
   - `/Users/mukundhan/Projects/engram/internal/db/db_test.go` -> `9.61`
   - `/Users/mukundhan/Projects/engram/internal/embeddings/errors.go` -> `10.0`
   - `/Users/mukundhan/Projects/engram/internal/embeddings/local.go` -> `10.0`
   - `/Users/mukundhan/Projects/engram/internal/embeddings/local_test.go` -> `9.68`
   - `/Users/mukundhan/Projects/engram/internal/embeddings/service.go` -> `9.09`
   - `/Users/mukundhan/Projects/engram/internal/embeddings/service_test.go` -> `10.0`
   - `/Users/mukundhan/Projects/engram/internal/models/engram.go` -> `N/A` (struct-only file; score unavailable)
   - `/Users/mukundhan/Projects/engram/internal/models/user.go` -> `10.0`
   - `/Users/mukundhan/Projects/engram/internal/repository/engram.go` -> `9.11`
   - `/Users/mukundhan/Projects/engram/internal/repository/engram_helpers_test.go` -> `10.0`
   - `/Users/mukundhan/Projects/engram/internal/repository/engram_unit_test.go` -> `10.0`
   - `/Users/mukundhan/Projects/engram/internal/repository/user.go` -> `9.38`
   - `/Users/mukundhan/Projects/engram/internal/repository/user_test.go` -> `10.0`
7. CodeScene pre-commit safeguard:
   - `pre_commit_code_health_safeguard(git_repository_path=/Users/mukundhan/Projects/engram)`
   - result: `quality_gates=passed`
   - findings: none

### 2026-02-22 (Go migration CP8: engram repository DB read parity)

1. Added DB-facing engram repository layer:
   - `internal/repository/engram_store.go`
   - `ListEngrams` with actor/project visibility filtering + pagination parity
   - `QueryEngrams` with candidate vector distance query + lexical rerank parity
2. Added typed operation inputs to reduce argument sprawl:
   - `ListEngramsInput`
   - `QueryEngramsInput`
3. Refactored helper/store cohesion:
   - moved store logic out of `internal/repository/engram.go`
   - extracted helper payload/section functions to reduce method complexity and restore code health
4. Added DB-path tests:
   - `internal/repository/engram_repository_test.go`
   - `TestListEngramsAppliesVisibilityAndProjectFilters`
   - `TestListEngramsWithoutActorOmitsVisibilityClause`
   - `TestQueryEngramsBuildsQueryAndReranks`
5. Executed migrated tests one-by-one:
   - `go test ./internal/repository -run '^TestListEngramsAppliesVisibilityAndProjectFilters$' -v`
   - `go test ./internal/repository -run '^TestListEngramsWithoutActorOmitsVisibilityClause$' -v`
   - `go test ./internal/repository -run '^TestQueryEngramsBuildsQueryAndReranks$' -v`
6. Full Go verification:
   - `go test ./...`
7. File-level CodeScene checks (all current Go migration files before commit):
   - `/Users/mukundhan/Projects/engram/cmd/api/main.go` -> `10.0`
   - `/Users/mukundhan/Projects/engram/internal/api/router.go` -> `10.0`
   - `/Users/mukundhan/Projects/engram/internal/api/router_test.go` -> `10.0`
   - `/Users/mukundhan/Projects/engram/internal/auth/password.go` -> `10.0`
   - `/Users/mukundhan/Projects/engram/internal/auth/password_test.go` -> `9.68`
   - `/Users/mukundhan/Projects/engram/internal/config/config.go` -> `9.68`
   - `/Users/mukundhan/Projects/engram/internal/config/config_test.go` -> `10.0`
   - `/Users/mukundhan/Projects/engram/internal/db/db.go` -> `10.0`
   - `/Users/mukundhan/Projects/engram/internal/db/db_test.go` -> `9.61`
   - `/Users/mukundhan/Projects/engram/internal/embeddings/errors.go` -> `10.0`
   - `/Users/mukundhan/Projects/engram/internal/embeddings/local.go` -> `10.0`
   - `/Users/mukundhan/Projects/engram/internal/embeddings/local_test.go` -> `9.68`
   - `/Users/mukundhan/Projects/engram/internal/embeddings/service.go` -> `9.09`
   - `/Users/mukundhan/Projects/engram/internal/embeddings/service_test.go` -> `10.0`
   - `/Users/mukundhan/Projects/engram/internal/models/engram.go` -> `N/A` (struct-only file; score unavailable)
   - `/Users/mukundhan/Projects/engram/internal/models/user.go` -> `10.0`
   - `/Users/mukundhan/Projects/engram/internal/repository/engram.go` -> `9.68`
   - `/Users/mukundhan/Projects/engram/internal/repository/engram_helpers_test.go` -> `10.0`
   - `/Users/mukundhan/Projects/engram/internal/repository/engram_repository_test.go` -> `10.0`
   - `/Users/mukundhan/Projects/engram/internal/repository/engram_store.go` -> `10.0`
   - `/Users/mukundhan/Projects/engram/internal/repository/engram_unit_test.go` -> `10.0`
   - `/Users/mukundhan/Projects/engram/internal/repository/user.go` -> `9.38`
   - `/Users/mukundhan/Projects/engram/internal/repository/user_test.go` -> `10.0`
8. CodeScene pre-commit safeguard:
   - `pre_commit_code_health_safeguard(git_repository_path=/Users/mukundhan/Projects/engram)`
   - result: `quality_gates=passed`
   - findings: none

### 2026-02-22 (Go migration CP51: non-checkpoint code-health uplift for chat pinning repository tests)

1. Improved legacy non-checkpoint chat pinning repository tests:
   - `internal/repository/chat_pinning_test.go`
   - consolidated duplicated missing-row pin/unpin tests into:
     - `TestPinnedDocumentOperationsReturnZeroValueWhenRowMissing`
   - consolidated duplicated pinned-list tests into:
     - `TestListPinnedResourcesBuildsExpectedQueryAndArgs`
   - preserved engram/document query SQL assertions and argument contract checks.
2. Executed chat pinning tests one-by-one:
   - `TestPinEngramToSessionReturnsPinnedRecord`
   - `TestUnpinEngramFromSessionReturnsTrueWhenRemoved`
   - `TestPinnedDocumentOperationsReturnZeroValueWhenRowMissing`
   - `TestListPinnedResourcesBuildsExpectedQueryAndArgs`
   - `TestListPinnedEngramSummariesAppliesVisibilityFilters`
3. Full Go verification:
   - `/usr/local/go/bin/go test ./...` passed.
4. CodeScene checks:
   - `/Users/mukundhan/Projects/engram/internal/repository/chat_pinning_test.go` improved from `9.38` to `9.51`
   - remaining note is a non-blocking large-test threshold edge.
5. CodeScene pre-commit safeguard:
   - `pre_commit_code_health_safeguard(git_repository_path=/Users/mukundhan/Projects/engram)`
   - result: `quality_gates=passed`
   - findings:
     - duplication fixed across the prior four pinning tests
     - one non-blocking large-method note introduced for the consolidated list test.

### 2026-02-22 (Go migration CP50: non-checkpoint code-health uplift for admin service implementation)

1. Improved legacy non-checkpoint admin service implementation:
   - `internal/admin/service.go`
   - reduced duplicated list input construction across session/engram/collection list operations via shared mapping helpers:
     - `sharedListRequestInput`
     - `toSharedListRequestInput`
     - `sessionInput`
     - `engramInput`
     - `collectionInput`
   - centralized repeated bool-result + not-found handling:
     - `boolResultNotFoundError`
   - centralized project resolution for write operations:
     - `resolveProjectIDForWrite`
   - extracted actor-structured move implementation in `moveEngramWithActor` while keeping public `MoveEngram` compatibility.
2. Executed affected admin service tests one-by-one:
   - `TestListMemoryAdminRequestsForwardSharedObject`
   - `TestListEngramsUsesRequestObject`
   - `TestUpdateCollectionRejectsStaleExpectedUpdatedAt`
   - `TestUpdateEngramRejectsStaleExpectedUpdatedAt`
   - `TestUpdateEngramUsesRepositoryRequestObject`
3. Full Go verification:
   - `/usr/local/go/bin/go test ./...` passed.
4. CodeScene checks:
   - `/Users/mukundhan/Projects/engram/internal/admin/service.go` improved from `8.54` to `9.68`
   - remaining note is a non-blocking public signature argument-count edge.
5. CodeScene pre-commit safeguard:
   - `pre_commit_code_health_safeguard(git_repository_path=/Users/mukundhan/Projects/engram)`
   - result: `quality_gates=passed`
   - findings: none.

### 2026-02-22 (Go migration CP49: non-checkpoint code-health uplift for admin service tests)

1. Improved legacy non-checkpoint admin service tests:
   - `internal/admin/service_test.go`
   - consolidated duplicated list-forwarding tests into one table-driven test:
     - `TestListMemoryAdminRequestsForwardSharedObject`
   - introduced `sharedListRequestCapture` to centralize forwarding assertions.
2. Executed admin service tests one-by-one:
   - `TestListMemoryAdminRequestsForwardSharedObject`
   - `TestListEngramsUsesRequestObject`
   - `TestUpdateCollectionRejectsStaleExpectedUpdatedAt`
   - `TestUpdateEngramRejectsStaleExpectedUpdatedAt`
   - `TestUpdateEngramUsesRepositoryRequestObject`
3. Full Go verification:
   - `/usr/local/go/bin/go test ./...` passed.
4. CodeScene checks:
   - `/Users/mukundhan/Projects/engram/internal/admin/service_test.go` improved from `9.38` to `10.0`
   - no remaining findings.
5. CodeScene pre-commit safeguard:
   - `pre_commit_code_health_safeguard(git_repository_path=/Users/mukundhan/Projects/engram)`
   - result: `quality_gates=passed`
   - findings: none.

### 2026-02-22 (Go migration CP48: non-checkpoint code-health uplift for admin engram repository tests)

1. Improved legacy non-checkpoint admin engram repository tests:
   - `internal/repository/admin_engram_test.go`
   - replaced high-argument row helper signatures with fixture structs:
     - `adminEngramRowFixture`
     - `adminEngramSourceRowFixture`
   - updated admin engram and admin engram update test call sites to use typed fixtures instead of long positional helper arguments.
2. Executed affected tests one-by-one:
   - `TestListAdminEngramsBuildsFiltersAndSearch`
   - `TestListAdminEngramSourcesReturnsRows`
   - `TestGetAdminEngramReturnsNilWhenMissing`
   - `TestGetAdminEngramHydratesSources`
   - `TestMoveAdminEngramProjectReturnsUpdatedRecordAndRunsDetachQuery`
   - `TestMoveAdminEngramProjectReturnsNilWhenNotFound`
   - `TestSoftDeleteEngramReturnsBool`
   - `TestRestoreEngramReturnsBool`
   - `TestBuildAdminEngramUpdateFieldsUsesPayloadValuesAndDefaults`
   - `TestBuildAdminEngramRetrievalTextUsesUpdateFields`
   - `TestBuildAdminEngramJSONPayloadOverridesMutableFields`
   - `TestReplaceAdminEngramSourcesReplacesRows`
   - `TestUpdateAdminEngramReturnsNilWhenMissing`
   - `TestUpdateAdminEngramPersistsFieldsAndOptionallySources`
3. Full Go verification:
   - `/usr/local/go/bin/go test ./...` passed.
4. CodeScene checks:
   - `/Users/mukundhan/Projects/engram/internal/repository/admin_engram_test.go` improved from `9.38` to `10.0`
   - `/Users/mukundhan/Projects/engram/internal/repository/admin_engram_update_test.go` remained `10.0` after helper migration.
   - no remaining findings.
5. CodeScene pre-commit safeguard:
   - `pre_commit_code_health_safeguard(git_repository_path=/Users/mukundhan/Projects/engram)`
   - result: `quality_gates=passed`
   - findings: none.

### 2026-02-22 (Go migration CP47: non-checkpoint code-health uplift for admin engram update repository tests)

1. Improved legacy non-checkpoint admin engram update repository tests:
   - `internal/repository/admin_engram_update_test.go`
   - reduced large/complex update-path test by extracting fixture-driven setup/assert helpers:
     - `buildUpdateAdminEngramFixture`
     - `buildUpdateAdminEngramFakeQueryer`
     - `setupUpdateAdminEngramStubs`
     - `assertUpdateAdminEngramRecord`
     - `assertUpdateAdminEngramWrites`
   - preserved JSON payload assertions, SQL argument checks, source replacement verification, and embedding contract checks.
2. Executed admin engram update tests one-by-one:
   - `TestBuildAdminEngramUpdateFieldsUsesPayloadValuesAndDefaults`
   - `TestBuildAdminEngramRetrievalTextUsesUpdateFields`
   - `TestBuildAdminEngramJSONPayloadOverridesMutableFields`
   - `TestReplaceAdminEngramSourcesReplacesRows`
   - `TestUpdateAdminEngramReturnsNilWhenMissing`
   - `TestUpdateAdminEngramPersistsFieldsAndOptionallySources`
3. Full Go verification:
   - `/usr/local/go/bin/go test ./...` passed.
4. CodeScene checks:
   - `/Users/mukundhan/Projects/engram/internal/repository/admin_engram_update_test.go` improved from `9.25` to `10.0`
   - no remaining findings.
5. CodeScene pre-commit safeguard:
   - `pre_commit_code_health_safeguard(git_repository_path=/Users/mukundhan/Projects/engram)`
   - result: `quality_gates=passed`
   - findings: none.

### 2026-02-22 (Go migration CP46: non-checkpoint code-health uplift for engram write repository tests)

1. Improved legacy non-checkpoint engram write repository tests:
   - `internal/repository/engram_write_test.go`
   - reduced large/complex primary write-path test by extracting fixture-driven helpers:
     - `buildCreateEngramWithReportFixture`
     - `buildCreateEngramWithReportFakeQueryer`
     - `setupCreateEngramWithReportStubs`
     - `assertCreateEngramWithReportResult`
     - `assertCreateEngramWithReportWrites`
   - preserved SQL/argument assertions and enrichment report behavior while reducing method size/complexity.
2. Executed engram write tests one-by-one:
   - `TestCreateEngramWithReportPersistsEngramSourcesAndArtifacts`
   - `TestCreateEngramWithReportReturnsEmbedErrorAndSkipsWrites`
   - `TestCreateEngramReturnsCreatedResponse`
3. Full Go verification:
   - `/usr/local/go/bin/go test ./...` passed.
4. CodeScene checks:
   - `/Users/mukundhan/Projects/engram/internal/repository/engram_write_test.go` improved from `9.26` to `10.0`
   - no remaining findings.
5. CodeScene pre-commit safeguard:
   - `pre_commit_code_health_safeguard(git_repository_path=/Users/mukundhan/Projects/engram)`
   - result: `quality_gates=passed`
   - findings: none.

### 2026-02-22 (Go migration CP45: non-checkpoint code-health uplift for chat pinning repository implementation)

1. Improved legacy non-checkpoint chat pinning repository implementation:
   - `internal/repository/chat_pinning.go`
   - reduced duplicated wrapper logic by extracting generic helpers:
     - `pinResourceRecord`
     - `listPinnedResourceRecords`
     - `mapPinnedResourceRows`
   - introduced shared mutation builders:
     - `engramMutationInput`
     - `documentMutationInput`
   - replaced high-argument internal mutation helper signatures with:
     - `pinnedResourceMutationInput`
2. Executed chat pinning tests one-by-one:
   - `TestPinEngramToSessionReturnsPinnedRecord`
   - `TestPinDocumentToSessionReturnsNilWhenResourceNotVisible`
   - `TestUnpinEngramFromSessionReturnsTrueWhenRemoved`
   - `TestUnpinDocumentFromSessionReturnsFalseWhenMissing`
   - `TestListPinnedEngramsReturnsRows`
   - `TestListPinnedDocumentsAppliesVisibilityFilter`
   - `TestListPinnedEngramSummariesAppliesVisibilityFilters`
3. Full Go verification:
   - `/usr/local/go/bin/go test ./...` passed.
4. CodeScene checks:
   - `/Users/mukundhan/Projects/engram/internal/repository/chat_pinning.go` improved from `8.81` to `9.68`
   - remaining note is a non-blocking helper argument-count threshold edge.
5. CodeScene pre-commit safeguard:
   - `pre_commit_code_health_safeguard(git_repository_path=/Users/mukundhan/Projects/engram)`
   - result: `quality_gates=passed`
   - findings:
     - fixed code duplication across pin/unpin/list wrappers
     - fixed argument-count findings on `pinResourceToSession` and `unpinResourceFromSession`
     - introduced one non-blocking helper arg-count note.

### 2026-02-22 (Go migration CP44: non-checkpoint code-health uplift for document repository implementation)

1. Improved legacy non-checkpoint document repository implementation:
   - `internal/repository/document.go`
   - reduced query-path complexity by extracting focused helpers:
     - `normalizeDocumentChunkTopK`
     - `buildDocumentChunkQuerySQLAndParams`
     - `queryDocumentChunkCandidates`
     - `buildDocumentChunkQueryResults`
   - replaced argument-heavy chunk replacement input with:
     - `documentChunkReplaceInput`
   - split chunk replacement flow into dedicated helpers:
     - `deleteDocumentChunks`
     - `insertDocumentChunk`
2. Executed document repository tests one-by-one:
   - `TestUpsertDocumentWithChunksPersistsDocumentAndChunks`
   - `TestUpsertDocumentWithChunksFailsOnEmbeddingCountMismatch`
   - `TestListDocumentsAppliesProjectFilter`
   - `TestQueryDocumentChunksBuildsQueryAndReranks`
   - `TestBuildDocumentChunkWhereDefaultsToActorScopeOnly`
   - `TestUpsertDocumentWithChunksRejectsInvalidVisibility`
3. Full Go verification:
   - `/usr/local/go/bin/go test ./...` passed.
4. CodeScene checks:
   - `/Users/mukundhan/Projects/engram/internal/repository/document.go` improved from `8.81` to `9.68`
   - remaining note is a non-blocking helper arg-count threshold edge.
5. CodeScene pre-commit safeguard:
   - `pre_commit_code_health_safeguard(git_repository_path=/Users/mukundhan/Projects/engram)`
   - result: `quality_gates=passed`
   - findings:
     - fixed `Complex Method` and `Overall Code Complexity`
     - replaced previous argument-count issue on `replaceDocumentChunks` with a helper arg-count note.

### 2026-02-22 (Go migration CP43: non-checkpoint code-health uplift for document repository tests)

1. Improved legacy non-checkpoint document repository tests:
   - `internal/repository/document_test.go`
   - replaced high-argument row helper with fixture struct:
     - `documentRecordRowFixture`
     - `documentRecordRowValues(fixture documentRecordRowFixture)`
   - reduced large tests by extracting focused setup/assertion helpers for:
     - deterministic clock and embed stubs
     - payload construction
     - SQL/args/result assertions
2. Executed document repository tests one-by-one:
   - `TestUpsertDocumentWithChunksPersistsDocumentAndChunks`
   - `TestUpsertDocumentWithChunksFailsOnEmbeddingCountMismatch`
   - `TestListDocumentsAppliesProjectFilter`
   - `TestQueryDocumentChunksBuildsQueryAndReranks`
   - `TestBuildDocumentChunkWhereDefaultsToActorScopeOnly`
   - `TestUpsertDocumentWithChunksRejectsInvalidVisibility`
3. Full Go verification:
   - `/usr/local/go/bin/go test ./...` passed.
4. CodeScene checks:
   - `/Users/mukundhan/Projects/engram/internal/repository/document_test.go` improved from `8.72` to `9.68`
   - remaining notes are non-blocking helper arg-count threshold edges.
5. CodeScene pre-commit safeguard:
   - `pre_commit_code_health_safeguard(git_repository_path=/Users/mukundhan/Projects/engram)`
   - result: `quality_gates=passed`
   - findings:
     - fixed both large-method findings
     - replaced old argument-heavy helper finding with new helper arg-count notes.

### 2026-02-22 (Go migration CP42: non-checkpoint code-health uplift for collection repository tests)

1. Improved legacy non-checkpoint collection repository tests:
   - `internal/repository/collection_test.go`
   - replaced high-argument row-value helper with fixture struct:
     - `collectionRowFixture`
     - `collectionRowValues(fixture collectionRowFixture)`
   - consolidated duplicated missing-row tests:
     - merged `TestGetCollectionReturnsNilWhenMissing` and `TestUpdateCollectionReturnsNilWhenMissing`
       into `TestGetAndUpdateCollectionReturnNilWhenMissing` with subtests.
2. Executed collection repository tests one-by-one:
   - `TestListCollectionsBuildsFilters`
   - `TestGetAndUpdateCollectionReturnNilWhenMissing`
   - `TestCreateCollectionUsesGeneratedID`
   - `TestCreateCollectionMapsDuplicateNameError`
   - `TestSoftDeleteCollectionReturnsBool`
   - `TestAddCollectionItemsReturnsCountAndSkipsEmpty`
   - `TestRemoveCollectionItemReturnsBool`
3. Full Go verification:
   - `/usr/local/go/bin/go test ./...` passed.
4. CodeScene checks:
   - `/Users/mukundhan/Projects/engram/internal/repository/collection_test.go` improved from `9.09` to `10.0`
   - `code_health_review` now reports no findings.
5. CodeScene pre-commit safeguard:
   - `pre_commit_code_health_safeguard(git_repository_path=/Users/mukundhan/Projects/engram)`
   - result: `quality_gates=passed`
   - findings: none

### 2026-02-22 (Go migration CP41: non-checkpoint code-health uplift for OAuth repository tests)

1. Improved legacy non-checkpoint OAuth repository tests:
   - `internal/repository/oauth_test.go`
   - replaced high-argument row-value helpers with fixture structs:
     - `oauthClientRowFixture`
     - `oauthAuthorizationCodeRowFixture`
   - updated helper signatures:
     - `oauthClientRowValues(fixture oauthClientRowFixture)`
     - `oauthAuthorizationCodeRowValues(fixture oauthAuthorizationCodeRowFixture)`
2. Executed OAuth repository tests one-by-one:
   - `TestCreateOAuthClientReturnsCreatedRecord`
   - `TestGetOAuthClientReturnsNilWhenMissing`
   - `TestCreateOAuthAuthorizationCodeReturnsRecord`
   - `TestGetOAuthAuthorizationCodeByHashReturnsRecord`
   - `TestConsumeOAuthAuthorizationCodeReturnsNilWhenAlreadyConsumed`
3. Full Go verification:
   - `/usr/local/go/bin/go test ./...` passed.
4. CodeScene checks:
   - `/Users/mukundhan/Projects/engram/internal/repository/oauth_test.go` improved from `9.38` to `10.0`
   - `code_health_review` now reports no findings.
5. CodeScene pre-commit safeguard:
   - `pre_commit_code_health_safeguard(git_repository_path=/Users/mukundhan/Projects/engram)`
   - result: `quality_gates=passed`
   - findings: none

### 2026-02-22 (Go migration CP40: non-checkpoint code-health uplift for chat repository payload normalization)

1. Improved legacy non-checkpoint chat repository implementation:
   - `internal/repository/chat.go`
   - refactored payload normalization flow:
     - create path split into defaults and enum validation helpers.
     - update path switched to shared optional enum normalizer.
   - added helpers:
     - `applyCreatePayloadDefaults`
     - `validateCreatePayloadEnums`
     - `normalizeOptionalEnum[T ~string]`
2. Executed chat repository tests one-by-one:
   - `TestCreateChatSessionReturnsInsertedRecord`
   - `TestListChatSessionsAppliesVisibilityAndProjectFilter`
   - `TestGetChatSessionAdminRecordReturnsRecord`
   - `TestChatSessionGetAndUpdateReturnNilWhenRowMissing`
   - `TestUpdateChatSessionValidatesProviderValue`
   - `TestNormalizeCreatePayloadRejectsInvalidVisibility`
3. Full Go verification:
   - `/usr/local/go/bin/go test ./...` passed.
4. CodeScene checks:
   - `/Users/mukundhan/Projects/engram/internal/repository/chat.go` improved from `9.02` to `9.68`
   - `code_health_review` retains a non-blocking threshold-edge note:
     - `applyCreatePayloadDefaults` with `cc = 9`.
5. CodeScene pre-commit safeguard:
   - `pre_commit_code_health_safeguard(git_repository_path=/Users/mukundhan/Projects/engram)`
   - result: `quality_gates=passed`
   - findings:
     - fixed `Bumpy Road Ahead` and `Overall Code Complexity`
     - one `Complex Method` finding moved to new helper at threshold edge (`cc = 9`).

### 2026-02-22 (Go migration CP39: non-checkpoint code-health uplift for chat repository tests)

1. Improved legacy non-checkpoint chat repository tests:
   - `internal/repository/chat_test.go`
   - consolidated duplicated nil-row tests into one subtest-based test:
     - `TestChatSessionGetAndUpdateReturnNilWhenRowMissing`
   - replaced high-argument row helper with fixture struct:
     - added `chatSessionRowFixture`
     - changed `chatSessionRowValues` to accept fixture struct instead of 16 parameters.
2. Executed chat repository tests one-by-one:
   - `TestCreateChatSessionReturnsInsertedRecord`
   - `TestListChatSessionsAppliesVisibilityAndProjectFilter`
   - `TestGetChatSessionAdminRecordReturnsRecord`
   - `TestChatSessionGetAndUpdateReturnNilWhenRowMissing`
   - `TestUpdateChatSessionValidatesProviderValue`
   - `TestNormalizeCreatePayloadRejectsInvalidVisibility`
3. Full Go verification:
   - `/usr/local/go/bin/go test ./...` passed.
4. CodeScene checks:
   - `/Users/mukundhan/Projects/engram/internal/repository/chat_test.go` improved from `9.09` to `10.0`
   - `code_health_review` now reports no findings.
5. CodeScene pre-commit safeguard:
   - `pre_commit_code_health_safeguard(git_repository_path=/Users/mukundhan/Projects/engram)`
   - result: `quality_gates=passed`
   - findings: none

### 2026-02-22 (Go migration CP38: non-checkpoint code-health uplift for OAuth repository fetch paths)

1. Improved legacy non-checkpoint OAuth repository querying:
   - `internal/repository/oauth.go`
   - replaced duplicated `QueryRow + scan + ErrNoRows` patterns with shared helper:
     - `queryOptionalRecord[T any]`
   - applied helper in:
     - `GetOAuthClient`
     - `GetOAuthAuthorizationCodeByHash`
     - `ConsumeOAuthAuthorizationCode`
2. Executed OAuth repository tests one-by-one:
   - `TestCreateOAuthClientReturnsCreatedRecord`
   - `TestGetOAuthClientReturnsNilWhenMissing`
   - `TestCreateOAuthAuthorizationCodeReturnsRecord`
   - `TestGetOAuthAuthorizationCodeByHashReturnsRecord`
   - `TestConsumeOAuthAuthorizationCodeReturnsNilWhenAlreadyConsumed`
3. Full Go verification:
   - `/usr/local/go/bin/go test ./...` passed.
4. CodeScene checks:
   - `/Users/mukundhan/Projects/engram/internal/repository/oauth.go` improved from `9.38` to `10.0`
   - `code_health_review` now reports no findings.
5. CodeScene pre-commit safeguard:
   - `pre_commit_code_health_safeguard(git_repository_path=/Users/mukundhan/Projects/engram)`
   - result: `quality_gates=passed`
   - findings: none

### 2026-02-22 (Go migration CP37: non-checkpoint code-health uplift for embeddings fallback flow)

1. Improved legacy non-checkpoint embedding service fallback handling:
   - `internal/embeddings/service.go`
   - removed duplicated fallback logic by introducing shared generic helper:
     - `callWithFallback[T any]`
   - `embedWithFallback` and `embedManyWithFallback` now route through one fallback execution path.
2. Executed embeddings tests one-by-one:
   - `TestEmbeddingServiceFallsBackToLocalProvider`
   - `TestEmbeddingServiceEmbedManyUsesProviderIDFromActiveProvider`
3. Full Go verification:
   - `/usr/local/go/bin/go test ./...` passed.
4. CodeScene checks:
   - `/Users/mukundhan/Projects/engram/internal/embeddings/service.go` improved from `9.09` to `9.68`
   - remaining note is non-blocking module-level `String Heavy Function Arguments`.
5. CodeScene pre-commit safeguard:
   - `pre_commit_code_health_safeguard(git_repository_path=/Users/mukundhan/Projects/engram)`
   - result: `quality_gates=passed`
   - findings: none

### 2026-02-22 (Go migration CP36: non-checkpoint code-health uplift for user repository scans)

1. Improved legacy non-checkpoint repository mapping:
   - `internal/repository/user.go`
   - replaced duplicated scan-to-model field assignment by introducing shared helper:
     - `buildScannedUserRecord`
   - `userRecordFromScan` and `userAuthRecordFromScan` now reuse shared scanned user fields mapping.
2. Executed repository user tests one-by-one:
   - `TestGetUserAuthRecordLooksUpUsername`
   - `TestListUsersReturnsRecords`
   - `TestCreateUserReturnsCreatedUser`
   - `TestCreateUserReturnsUsernameExistsOnDuplicate`
   - `TestUpdateUserReturnsNilWhenNotFound`
   - `TestUpdateUserAppliesProvidedFields`
3. Full Go verification:
   - `/usr/local/go/bin/go test ./...` passed.
4. CodeScene checks:
   - `/Users/mukundhan/Projects/engram/internal/repository/user.go` improved from `9.38` to `10.0`
   - `code_health_review` now reports no findings for this file.
5. CodeScene pre-commit safeguard:
   - `pre_commit_code_health_safeguard(git_repository_path=/Users/mukundhan/Projects/engram)`
   - result: `quality_gates=passed`
   - findings: none

### 2026-02-22 (Go migration CP35: non-checkpoint code-health uplift for session token decode)

1. Improved legacy non-checkpoint auth utility:
   - `internal/auth/session.go`
   - refactored `Decode` into focused helpers:
     - `parseSessionToken`
     - `validateSessionTokenSignature`
     - `decodeSessionStatePayload`
     - `validateSessionTTL`
   - preserved behavior for invalid tokens, signature verification, and issued-at/TTL expiry handling.
2. Executed auth session tests one-by-one:
   - `TestSessionManagerEncodeDecodeRoundTrip`
   - `TestSessionManagerDecodeRejectsTamperedToken`
   - `TestSessionManagerDecodeRequestReadsCookie`
   - `TestNewSessionManagerRejectsEmptySecret`
   - `TestSessionManagerDecodeRejectsExpiredToken`
3. Full Go verification:
   - `/usr/local/go/bin/go test ./...` passed.
4. CodeScene checks:
   - `/Users/mukundhan/Projects/engram/internal/auth/session.go` improved from `9.38` to `9.68`
   - `code_health_review` now has no `Complex Method` or `Complex Conditional` findings on `Decode`.
5. CodeScene pre-commit safeguard:
   - `pre_commit_code_health_safeguard(git_repository_path=/Users/mukundhan/Projects/engram)`
   - result: `quality_gates=passed`
   - findings:
     - fixed `Complex Method` and `Complex Conditional` in `Decode`
     - introduced non-blocking module-level `String Heavy Function Arguments` note.

### 2026-02-22 (Go migration CP34: session-auth engram route parity + >9.5 code-health gate)

1. Added session-auth engram route set:
   - `internal/api/session_engrams.go` (new)
   - mounted in `internal/api/session_auth.go`:
     - `POST /api/v1/engrams`
     - `GET /api/v1/engrams`
     - `POST /api/v1/engrams/query`
     - `GET /api/v1/engrams/{engram_id}/sources`
     - `GET /api/v1/engrams/{engram_id}/rehydrate`
2. Wired runtime dependencies and project-resolution bridge:
   - `cmd/api/main.go`
   - added repository-backed `SessionAuthDependencies` callbacks for engram create/list/query/rehydrate/sources.
   - added explicit project/default-project resolution helpers for session engram create behavior parity.
   - refactored dependency builders into focused helper functions to keep CodeScene score at `10.0`.
3. Added repository helper:
   - `internal/repository/engram.go`
   - `BuildLocalQueryLiteral(query string, embeddingDim int)` for deterministic local query embedding literal generation.
4. Added/updated migrated tests:
   - `internal/api/session_engrams_test.go` (new):
     - `TestMountSessionAuthRoutesEngramCollectionRoutesUseRepository`
     - `TestMountSessionAuthRoutesCreateEngramUsesProjectResolution`
     - `TestMountSessionAuthRoutesRehydrateReturns404WhenMissing`
     - `TestMountSessionAuthRoutesListEngramSourcesUsesRepository`
     - `TestMountSessionAuthRoutesCreateEngramRequiresProjectOrDefault`
   - `internal/api/router_test.go`:
     - added route mount assertions for `/api/v1/engrams` with/without required session engram dependencies.
5. Executed migrated tests one-by-one:
   - `TestMountSessionAuthRoutesEngramCollectionRoutesUseRepository`
   - `TestMountSessionAuthRoutesCreateEngramUsesProjectResolution`
   - `TestMountSessionAuthRoutesRehydrateReturns404WhenMissing`
   - `TestMountSessionAuthRoutesListEngramSourcesUsesRepository`
   - `TestMountSessionAuthRoutesCreateEngramRequiresProjectOrDefault`
   - `TestSessionAuthRoutesNotMountedWithoutDependencies`
   - `TestSessionAuthRoutesMountedWithDependencies`
6. Full Go verification:
   - `/usr/local/go/bin/go test ./...` passed.
7. File-level CodeScene checks (checkpoint-touched files):
   - `/Users/mukundhan/Projects/engram/cmd/api/main.go` -> `10.0`
   - `/Users/mukundhan/Projects/engram/internal/api/session_auth.go` -> `10.0`
   - `/Users/mukundhan/Projects/engram/internal/api/session_engrams.go` -> `10.0`
   - `/Users/mukundhan/Projects/engram/internal/api/session_engrams_test.go` -> `10.0`
   - `/Users/mukundhan/Projects/engram/internal/api/router_test.go` -> `10.0`
   - `/Users/mukundhan/Projects/engram/internal/repository/engram.go` -> `9.68`
8. CodeScene pre-commit safeguard:
   - `pre_commit_code_health_safeguard(git_repository_path=/Users/mukundhan/Projects/engram)`
   - result: `quality_gates=passed`
   - finding: unchanged non-blocking `Primitive Obsession` note in `internal/repository/engram.go`

### 2026-02-22 (Go migration CP33: role-aware `/api/v1/users` parity + auth/session code-health uplift)

1. Added session-auth user-management API routes:
   - `internal/api/session_users.go` (new)
   - mounted under `internal/api/session_auth.go`:
     - `GET /api/v1/users` (admin-only list with limit/offset validation)
     - `POST /api/v1/users` (admin-only create with role/password/username validation + duplicate conflict mapping)
     - `PATCH /api/v1/users/{user_id}` (admin-only partial update with empty-payload guard)
   - preserved `/api/v1/me` behavior via shared authenticated actor resolution helper.
2. Wired runtime dependencies for new routes:
   - `cmd/api/main.go`
   - added repository-backed list/create/update callbacks and password hashing dependency to `SessionAuthDependencies`.
3. Added/updated migrated tests:
   - `internal/api/session_users_test.go` (new):
     - `TestMountSessionAuthRoutesAdminCanManageUsers`
     - `TestMountSessionAuthRoutesNonAdminCannotAccessAdminRoutes`
     - `TestMountSessionAuthRoutesCreateUserReturnsConflictForDuplicateUsername`
     - `TestMountSessionAuthRoutesUpdateUserRequiresFields`
   - `internal/api/router_test.go`:
     - route-mount assertions for `/api/v1/users` with and without session-auth dependencies.
4. Executed migrated tests one-by-one:
   - `TestSessionAuthRoutesNotMountedWithoutDependencies`
   - `TestSessionAuthRoutesMountedWithDependencies`
   - `TestMountSessionUIRoutesHomeRedirectsToLoginWhenUnauthenticated`
   - `TestMountSessionUIRoutesDashboardRedirectsToLoginWhenUnauthenticated`
   - `TestMountSessionUIRoutesAdminRedirectsToLoginWhenUnauthenticated`
   - `TestMountSessionUIRoutesLoginRejectsInvalidCredentials`
   - `TestMountSessionUIRoutesLoginRejectsInvalidCSRF`
   - `TestMountSessionUIRoutesLoginRateLimitAfterRepeatedFailures`
   - `TestMountSessionUIRoutesLoginAndLogoutWorkflow`
   - `TestMountSessionUIRoutesAdminRejectsNonAdminRole`
   - `TestMountSessionUIRoutesAdminAllowsAdminRole`
   - `TestMountSessionUIRoutesLoginRedirectPathSanitization`
   - `TestMountSessionUIRoutesLogoutRejectsInvalidCSRF`
   - `TestMountSessionUIRoutesLoginFailureWritesAuditLog`
   - `TestMountSessionAuthRoutesCSRFIssuesTokenAndCookie`
   - `TestMountSessionAuthRoutesCSRFCookieHonorsSecureFlag`
   - `TestMountSessionAuthRoutesLoginSetsSessionCookie`
   - `TestMountSessionAuthRoutesLoginRejectsInvalidCSRF`
   - `TestMountSessionAuthRoutesMeUsesSessionActorMiddleware`
   - `TestSessionActorMiddlewareInjectsActorFromSessionCookie`
   - `TestSessionActorMiddlewareSkipsInactiveUsers`
   - `TestSessionActorMiddlewareIgnoresInvalidSessionCookie`
   - `TestMountSessionAuthRoutesAdminCanManageUsers`
   - `TestMountSessionAuthRoutesNonAdminCannotAccessAdminRoutes`
   - `TestMountSessionAuthRoutesCreateUserReturnsConflictForDuplicateUsername`
   - `TestMountSessionAuthRoutesUpdateUserRequiresFields`
5. Additional code-health uplift while in this slice:
   - `internal/api/session_actor_middleware.go`: extracted actor-resolution helpers to reduce complexity.
   - `cmd/api/main.go`: split runtime auth wiring into smaller helper functions to satisfy CodeScene pre-commit gate.
6. Full Go verification:
   - `go test ./...` passed.
7. CodeScene health checks:
   - scored all `.go` files in repository (79 files), including struct-only model review follow-up.
   - touched/uplifted file scores:
     - `cmd/api/main.go` -> `10.0`
     - `internal/api/session_auth.go` -> `10.0`
     - `internal/api/session_users.go` -> `10.0`
     - `internal/api/session_users_test.go` -> `9.68`
     - `internal/api/session_actor_middleware.go` -> `9.68`
     - `internal/api/router_test.go` -> `10.0`
8. CodeScene pre-commit safeguard:
   - `pre_commit_code_health_safeguard(git_repository_path=/Users/mukundhan/Projects/engram)`
   - result: `quality_gates=passed`
   - findings: none

### 2026-02-22 (Go migration CP31: distributed limiter parity baseline)

1. Added distributed `rate_limit_state` persistence for Go auth limiter state:
   - `/Users/mukundhan/Projects/engram/internal/auth/ratelimit_store_pgx.go`
   - introduced `PGXRateLimitStore` with row-lock + mutate semantics for namespace/key limiter entries.
2. Refactored auth limiter code into focused modules while preserving behavior:
   - `/Users/mukundhan/Projects/engram/internal/auth/ratelimit.go` (shared limiter contracts/helpers)
   - `/Users/mukundhan/Projects/engram/internal/auth/login_guard.go` (`LoginAttemptGuard`)
   - `/Users/mukundhan/Projects/engram/internal/auth/request_limiter.go` (`RequestRateLimiter`)
   - added options-based constructors and `SetDistributedStore(...)` hooks for runtime wiring.
3. Updated runtime wiring:
   - `/Users/mukundhan/Projects/engram/cmd/api/main.go`
   - login guard now uses `auth.NewPGXRateLimitStore(pool)` and enables distributed state under `login_attempts`.
4. Added/updated migrated tests:
   - `/Users/mukundhan/Projects/engram/internal/auth/ratelimit_test.go`
   - added distributed/fallback coverage:
     - `TestLoginAttemptGuardDistributedStateSharedAcrossInstances`
     - `TestLoginAttemptGuardFallsBackToLocalStateWhenDistributedStoreFails`
     - `TestRequestRateLimiterDistributedStateSharedAcrossInstances`
     - `TestRequestRateLimiterFallsBackToLocalStateWhenDistributedStoreFails`
   - refactored request limiter threshold assertions into helpers to reduce test complexity.
5. Executed migrated tests one-by-one:
   - `/usr/local/go/bin/go test ./internal/auth -run '^TestRegisterFailureLocksAfterMaxAttempts$' -v`
   - `/usr/local/go/bin/go test ./internal/auth -run '^TestCheckUnlocksAfterLockoutExpiry$' -v`
   - `/usr/local/go/bin/go test ./internal/auth -run '^TestFailureWindowDropsStaleAttempts$' -v`
   - `/usr/local/go/bin/go test ./internal/auth -run '^TestRegisterSuccessClearsPriorFailures$' -v`
   - `/usr/local/go/bin/go test ./internal/auth -run '^TestLoginAttemptGuardDistributedStateSharedAcrossInstances$' -v`
   - `/usr/local/go/bin/go test ./internal/auth -run '^TestLoginAttemptGuardFallsBackToLocalStateWhenDistributedStoreFails$' -v`
   - `/usr/local/go/bin/go test ./internal/auth -run '^TestRequestRateLimiterThresholdBehavior$' -v`
   - `/usr/local/go/bin/go test ./internal/auth -run '^TestRequestRateLimiterDistributedStateSharedAcrossInstances$' -v`
   - `/usr/local/go/bin/go test ./internal/auth -run '^TestRequestRateLimiterFallsBackToLocalStateWhenDistributedStoreFails$' -v`
   - `/usr/local/go/bin/go test ./internal/api -run '^TestMountSessionUIRoutesLoginRateLimitAfterRepeatedFailures$' -v`
   - `/usr/local/go/bin/go test ./internal/api -run '^TestMountSessionUIRoutesLoginFailureWritesAuditLog$' -v`
   - `/usr/local/go/bin/go test ./internal/api -run '^TestMountSessionAuthRoutesLoginSetsSessionCookie$' -v`
6. Full Go verification:
   - `/usr/local/go/bin/go test ./...` passed.
7. File-level CodeScene checks (all Go files before commit):
   - scored all `.go` files in repository (78 files at this checkpoint).
   - struct-only model files reviewed with `code_health_review`:
     - `/Users/mukundhan/Projects/engram/internal/models/collection.go` -> `score=null`, findings none
     - `/Users/mukundhan/Projects/engram/internal/models/engram.go` -> `score=null`, findings none
     - `/Users/mukundhan/Projects/engram/internal/models/oauth.go` -> `score=null`, findings none
     - `/Users/mukundhan/Projects/engram/internal/models/project.go` -> `score=null`, findings none
   - checkpoint-touched file scores:
     - `/Users/mukundhan/Projects/engram/cmd/api/main.go` -> `10.0`
     - `/Users/mukundhan/Projects/engram/internal/auth/ratelimit.go` -> `10.0`
     - `/Users/mukundhan/Projects/engram/internal/auth/login_guard.go` -> `9.68`
     - `/Users/mukundhan/Projects/engram/internal/auth/request_limiter.go` -> `10.0`
     - `/Users/mukundhan/Projects/engram/internal/auth/ratelimit_store_pgx.go` -> `10.0`
     - `/Users/mukundhan/Projects/engram/internal/auth/ratelimit_test.go` -> `9.68`
8. CodeScene pre-commit safeguard:
   - `pre_commit_code_health_safeguard(git_repository_path=/Users/mukundhan/Projects/engram)`
   - result: `quality_gates=passed`
   - findings: non-blocking arg-count warning in test helper (`assertConsumeBlocked`)

### 2026-02-22 (Go migration CP32: UI/admin auth integration hardening)

1. Added admin UI route parity with role enforcement:
   - `/Users/mukundhan/Projects/engram/internal/api/session_ui.go`
   - added `GET /ui/admin` behavior:
     - redirects unauthenticated requests to `/login`
     - returns `403` JSON (`Admin role required`) for authenticated non-admin actors
     - renders admin migration page for admin users
2. Hardened auth/session UI helpers and reduced complexity:
   - `/Users/mukundhan/Projects/engram/internal/api/session_ui.go`
   - added shared user+csrf state resolver for UI route handlers.
   - replaced argument-heavy helpers with request structs.
   - encapsulated repeated auth/session conditionals.
3. Hardened session auth CSRF + auth record validation:
   - `/Users/mukundhan/Projects/engram/internal/api/session_auth.go`
   - unified required/optional CSRF validation paths.
   - reduced duplicated and complex conditional branches in login/logout validation helpers.
4. Expanded migrated tests:
   - `/Users/mukundhan/Projects/engram/internal/api/session_ui_test.go`
     - added:
       - `TestMountSessionUIRoutesAdminRedirectsToLoginWhenUnauthenticated`
       - `TestMountSessionUIRoutesAdminRejectsNonAdminRole`
       - `TestMountSessionUIRoutesAdminAllowsAdminRole`
     - refactored test helpers to reduce duplication/complexity.
   - `/Users/mukundhan/Projects/engram/internal/api/router_test.go`
     - route mount assertions now include `/ui/admin` behavior when session routes are mounted/unmounted.
5. Executed migrated tests one-by-one:
   - `/usr/local/go/bin/go test ./internal/api -run '^TestSessionAuthRoutesNotMountedWithoutDependencies$' -v`
   - `/usr/local/go/bin/go test ./internal/api -run '^TestSessionAuthRoutesMountedWithDependencies$' -v`
   - `/usr/local/go/bin/go test ./internal/api -run '^TestMountSessionUIRoutesHomeRedirectsToLoginWhenUnauthenticated$' -v`
   - `/usr/local/go/bin/go test ./internal/api -run '^TestMountSessionUIRoutesDashboardRedirectsToLoginWhenUnauthenticated$' -v`
   - `/usr/local/go/bin/go test ./internal/api -run '^TestMountSessionUIRoutesAdminRedirectsToLoginWhenUnauthenticated$' -v`
   - `/usr/local/go/bin/go test ./internal/api -run '^TestMountSessionUIRoutesLoginRejectsInvalidCredentials$' -v`
   - `/usr/local/go/bin/go test ./internal/api -run '^TestMountSessionUIRoutesLoginRejectsInvalidCSRF$' -v`
   - `/usr/local/go/bin/go test ./internal/api -run '^TestMountSessionUIRoutesLoginRateLimitAfterRepeatedFailures$' -v`
   - `/usr/local/go/bin/go test ./internal/api -run '^TestMountSessionUIRoutesLoginAndLogoutWorkflow$' -v`
   - `/usr/local/go/bin/go test ./internal/api -run '^TestMountSessionUIRoutesAdminRejectsNonAdminRole$' -v`
   - `/usr/local/go/bin/go test ./internal/api -run '^TestMountSessionUIRoutesAdminAllowsAdminRole$' -v`
   - `/usr/local/go/bin/go test ./internal/api -run '^TestMountSessionUIRoutesLoginRedirectPathSanitization$' -v`
   - `/usr/local/go/bin/go test ./internal/api -run '^TestMountSessionUIRoutesLogoutRejectsInvalidCSRF$' -v`
   - `/usr/local/go/bin/go test ./internal/api -run '^TestMountSessionUIRoutesLoginFailureWritesAuditLog$' -v`
   - `/usr/local/go/bin/go test ./internal/api -run '^TestMountSessionAuthRoutesCSRFIssuesTokenAndCookie$' -v`
   - `/usr/local/go/bin/go test ./internal/api -run '^TestMountSessionAuthRoutesCSRFCookieHonorsSecureFlag$' -v`
   - `/usr/local/go/bin/go test ./internal/api -run '^TestMountSessionAuthRoutesLoginSetsSessionCookie$' -v`
   - `/usr/local/go/bin/go test ./internal/api -run '^TestMountSessionAuthRoutesLoginRejectsInvalidCSRF$' -v`
   - `/usr/local/go/bin/go test ./internal/api -run '^TestMountSessionAuthRoutesMeUsesSessionActorMiddleware$' -v`
6. Full Go verification:
   - `/usr/local/go/bin/go test ./...` passed.
7. File-level CodeScene checks (all Go files before commit):
   - scored all `.go` files in repository (78 files at this checkpoint).
   - struct-only model files reviewed:
     - `/Users/mukundhan/Projects/engram/internal/models/collection.go` -> `score=null`, findings none
     - `/Users/mukundhan/Projects/engram/internal/models/engram.go` -> `score=null`, findings none
     - `/Users/mukundhan/Projects/engram/internal/models/oauth.go` -> `score=null`, findings none
     - `/Users/mukundhan/Projects/engram/internal/models/project.go` -> `score=null`, findings none
   - CP32 touched files:
     - `/Users/mukundhan/Projects/engram/internal/api/session_ui.go` -> `10.0`
     - `/Users/mukundhan/Projects/engram/internal/api/session_auth.go` -> `10.0`
     - `/Users/mukundhan/Projects/engram/internal/api/session_ui_test.go` -> `10.0`
     - `/Users/mukundhan/Projects/engram/internal/api/router_test.go` -> `10.0`
8. CodeScene pre-commit safeguard:
   - `pre_commit_code_health_safeguard(git_repository_path=/Users/mukundhan/Projects/engram)`
   - result: `quality_gates=passed`
   - findings: none

### 2026-02-22 (Go migration CP30: login guard + audit parity baseline)

1. Added Go login/rate-limit primitives:
   - `internal/auth/ratelimit.go`
   - `LoginAttemptGuard` and `RequestRateLimiter` parity semantics with configurable thresholds/windows and test clock injection.
2. Added Go audit event logger:
   - `internal/audit/audit.go`
   - JSONL request-event writer with sanitization/truncation and optional stdout mirroring.
3. Wired runtime dependencies:
   - `cmd/api/main.go`
   - session auth dependencies now include:
     - login-attempt guard (`LOGIN_RATE_LIMIT_*` settings)
     - audit logger (`AUDIT_LOG_*` settings)
4. Updated session auth/UI flow:
   - `internal/api/session_auth.go`
     - added dependency hooks (`LoginAttemptGuard`, `LogAuditEvent`).
     - refactored audit helper signature to satisfy CodeScene argument-count gate.
   - `internal/api/session_ui.go`
     - login preconditions now include login-attempt guard enforcement.
     - login/logout paths now emit audit events (`login_success`, `login_failed`, `login_csrf_rejected`, `login_rate_limited`, `logout_success`, `logout_csrf_rejected`).
     - lockout responses return `429` JSON with retry detail.
5. Added/updated tests:
   - `internal/auth/ratelimit_test.go`
   - `internal/audit/audit_test.go`
   - `internal/api/session_ui_test.go`:
     - `TestMountSessionUIRoutesLoginRateLimitAfterRepeatedFailures`
     - `TestMountSessionUIRoutesLoginFailureWritesAuditLog`
6. Executed migrated tests one-by-one:
   - `go test ./internal/auth -run '^TestRegisterFailureLocksAfterMaxAttempts$'`
   - `go test ./internal/auth -run '^TestCheckUnlocksAfterLockoutExpiry$'`
   - `go test ./internal/auth -run '^TestFailureWindowDropsStaleAttempts$'`
   - `go test ./internal/auth -run '^TestRegisterSuccessClearsPriorFailures$'`
   - `go test ./internal/auth -run '^TestRequestRateLimiterThresholdBehavior$'`
   - `go test ./internal/audit -run '^TestLogRequestEventWritesJSONLine$'`
   - `go test ./internal/audit -run '^TestLogRequestEventTruncatesOversizedPayload$'`
   - `go test ./internal/api -run '^TestMountSessionUIRoutesHomeRedirectsToLoginWhenUnauthenticated$'`
   - `go test ./internal/api -run '^TestMountSessionUIRoutesDashboardRedirectsToLoginWhenUnauthenticated$'`
   - `go test ./internal/api -run '^TestMountSessionUIRoutesLoginRejectsInvalidCredentials$'`
   - `go test ./internal/api -run '^TestMountSessionUIRoutesLoginRejectsInvalidCSRF$'`
   - `go test ./internal/api -run '^TestMountSessionUIRoutesLoginRateLimitAfterRepeatedFailures$'`
   - `go test ./internal/api -run '^TestMountSessionUIRoutesLoginAndLogoutWorkflow$'`
   - `go test ./internal/api -run '^TestMountSessionUIRoutesLoginRedirectPathSanitization$'`
   - `go test ./internal/api -run '^TestMountSessionUIRoutesLogoutRejectsInvalidCSRF$'`
   - `go test ./internal/api -run '^TestMountSessionUIRoutesLoginFailureWritesAuditLog$'`
   - `go test ./internal/api -run '^TestSessionAuthRoutesNotMountedWithoutDependencies$'`
   - `go test ./internal/api -run '^TestSessionAuthRoutesMountedWithDependencies$'`
   - `go test ./internal/api -run '^TestMountSessionAuthRoutesCSRFIssuesTokenAndCookie$'`
   - `go test ./internal/api -run '^TestMountSessionAuthRoutesCSRFCookieHonorsSecureFlag$'`
   - `go test ./internal/api -run '^TestMountSessionAuthRoutesLoginSetsSessionCookie$'`
   - `go test ./internal/api -run '^TestMountSessionAuthRoutesLoginRejectsInvalidCSRF$'`
   - `go test ./internal/api -run '^TestMountSessionAuthRoutesMeUsesSessionActorMiddleware$'`
   - `go test ./internal/auth -run '^TestSessionManagerEncodeDecodeRoundTrip$'`
   - `go test ./internal/auth -run '^TestSessionManagerDecodeRejectsExpiredToken$'`
7. Full Go verification:
   - `go test ./...` passed.
8. File-level CodeScene checks before commit:
   - scored all Go files in repository (73 files at this checkpoint).
   - struct-only model files explicitly reviewed:
     - `internal/models/oauth.go`
     - `internal/models/project.go`
     - `internal/models/collection.go`
     - `internal/models/engram.go`
     - `code_health_review` returned `score=null`, `review=[]`.
   - changed-file score highlights:
     - `/Users/mukundhan/Projects/engram/internal/auth/ratelimit.go` -> `10.0`
     - `/Users/mukundhan/Projects/engram/internal/auth/ratelimit_test.go` -> `9.61`
     - `/Users/mukundhan/Projects/engram/internal/audit/audit.go` -> `9.68`
     - `/Users/mukundhan/Projects/engram/internal/audit/audit_test.go` -> `10.0`
     - `/Users/mukundhan/Projects/engram/internal/api/session_ui.go` -> `8.81`
     - `/Users/mukundhan/Projects/engram/internal/api/session_auth.go` -> `8.28`
     - `/Users/mukundhan/Projects/engram/cmd/api/main.go` -> `10.0`
9. CodeScene pre-commit safeguard:
   - initial run failed on:
     - `handleLoginSubmit` complexity threshold
     - `logAuditEventRequest` argument-count threshold
   - refactored and reran:
     - final `pre_commit_code_health_safeguard(git_repository_path=/Users/mukundhan/Projects/engram)` -> `quality_gates=passed`

### 2026-02-22 (Go migration CP29: UI/login parity baseline)

1. Added session-backed UI auth routes:
   - `internal/api/session_ui.go`
   - `GET /` redirects to `/login` when unauthenticated and `/ui` when authenticated.
   - `GET /login` renders login form with hidden `csrf_token`.
   - `POST /login` authenticates against user repository and refreshes session/CSRF state.
   - `POST /logout` validates form CSRF token and clears session cookie.
   - `GET /ui` renders authenticated dashboard and logout form with CSRF token.
2. Updated router composition to mount UI routes alongside session API routes:
   - `internal/api/router.go`
3. Refactored shared session helpers for route parity:
   - `internal/api/session_auth.go`
   - extracted reusable CSRF session-state helper and authenticated-session state builder.
4. Added migrated tests:
   - `internal/api/session_ui_test.go`
   - updated `internal/api/router_test.go` for `/login` mounting expectations.
5. Executed migrated tests one-by-one:
   - `go test ./internal/api -run '^TestMountSessionUIRoutesHomeRedirectsToLoginWhenUnauthenticated$'`
   - `go test ./internal/api -run '^TestMountSessionUIRoutesDashboardRedirectsToLoginWhenUnauthenticated$'`
   - `go test ./internal/api -run '^TestMountSessionUIRoutesLoginRejectsInvalidCredentials$'`
   - `go test ./internal/api -run '^TestMountSessionUIRoutesLoginRejectsInvalidCSRF$'`
   - `go test ./internal/api -run '^TestMountSessionUIRoutesLoginAndLogoutWorkflow$'`
   - `go test ./internal/api -run '^TestMountSessionUIRoutesLoginRedirectPathSanitization$'`
   - `go test ./internal/api -run '^TestMountSessionUIRoutesLogoutRejectsInvalidCSRF$'`
   - `go test ./internal/api -run '^TestSessionAuthRoutesNotMountedWithoutDependencies$'`
   - `go test ./internal/api -run '^TestSessionAuthRoutesMountedWithDependencies$'`
   - `go test ./internal/api -run '^TestMountSessionAuthRoutesCSRFIssuesTokenAndCookie$'`
   - `go test ./internal/api -run '^TestMountSessionAuthRoutesCSRFCookieHonorsSecureFlag$'`
   - `go test ./internal/api -run '^TestMountSessionAuthRoutesLoginSetsSessionCookie$'`
   - `go test ./internal/api -run '^TestMountSessionAuthRoutesLoginRejectsInvalidCSRF$'`
   - `go test ./internal/api -run '^TestMountSessionAuthRoutesMeUsesSessionActorMiddleware$'`
   - `go test ./internal/auth -run '^TestSessionManagerEncodeDecodeRoundTrip$'`
   - `go test ./internal/auth -run '^TestSessionManagerDecodeRejectsExpiredToken$'`
6. Full Go verification:
   - `go test ./...` passed.
7. File-level CodeScene checks before commit:
   - scored all 71 Go files via `code_health_score`.
   - struct-only model files explicitly reviewed:
     - `internal/models/oauth.go`
     - `internal/models/project.go`
     - `internal/models/collection.go`
     - `internal/models/engram.go`
     - `code_health_review` results: `score=null`, `review=[]`.
   - changed-file score highlights:
     - `/Users/mukundhan/Projects/engram/internal/api/router.go` -> `10.0`
     - `/Users/mukundhan/Projects/engram/internal/api/router_test.go` -> `10.0`
     - `/Users/mukundhan/Projects/engram/internal/api/session_auth.go` -> `8.28`
     - `/Users/mukundhan/Projects/engram/internal/api/session_ui.go` -> `8.81`
     - `/Users/mukundhan/Projects/engram/internal/api/session_ui_test.go` -> `8.81`
8. CodeScene pre-commit safeguard:
   - `pre_commit_code_health_safeguard(git_repository_path=/Users/mukundhan/Projects/engram)`
   - result: `quality_gates=passed`
   - findings: none

### 2026-02-22 (Go migration CP28: session hardening baseline)

1. Hardened session token lifecycle semantics:
   - `internal/auth/session.go`
   - added `SessionState.IssuedAt`.
   - added `SessionManagerOptions` (`CookieName`, `TTL`, `Now`) and `NewSessionManagerWithOptions`.
   - `NewSessionManager` now applies a default 24-hour TTL.
   - `Encode` now backfills issued-at when empty; `Decode` rejects missing/invalid/expired issued-at.
2. Hardened session-auth cookie behavior and improved route maintainability:
   - `internal/api/session_auth.go`
   - session cookie write path now sets `Secure`, `MaxAge`, and `Expires` using runtime config and session TTL.
   - logout clear-cookie path now preserves secure-flag parity.
   - refactored `MountSessionAuthRoutes` by extracting csrf/login/logout/me handlers and helper functions to satisfy code-health gates.
3. Updated runtime wiring:
   - `cmd/api/main.go`
   - `SessionAuthDependencies` now receives `CookieSecure` from `config.IsProductionEnv`.
4. Added/updated tests:
   - `internal/auth/session_test.go`
     - `TestSessionManagerEncodeDecodeRoundTrip` now validates issued-at population.
     - `TestSessionManagerDecodeRejectsExpiredToken` verifies TTL expiry handling with deterministic clock injection.
   - `internal/api/session_auth_test.go`
     - added `TestMountSessionAuthRoutesCSRFCookieHonorsSecureFlag`.
     - strengthened `TestMountSessionAuthRoutesCSRFIssuesTokenAndCookie` to assert positive cookie `MaxAge`.
5. Executed migrated tests one-by-one:
   - `go test ./internal/auth -run '^TestSessionManagerEncodeDecodeRoundTrip$'`
   - `go test ./internal/auth -run '^TestSessionManagerDecodeRejectsExpiredToken$'`
   - `go test ./internal/api -run '^TestMountSessionAuthRoutesCSRFIssuesTokenAndCookie$'`
   - `go test ./internal/api -run '^TestMountSessionAuthRoutesCSRFCookieHonorsSecureFlag$'`
   - `go test ./internal/api -run '^TestMountSessionAuthRoutesLoginSetsSessionCookie$'`
   - `go test ./internal/api -run '^TestMountSessionAuthRoutesLoginRejectsInvalidCSRF$'`
   - `go test ./internal/api -run '^TestMountSessionAuthRoutesMeUsesSessionActorMiddleware$'`
6. Full Go verification:
   - `go test ./...` passed.
7. File-level CodeScene checks before commit:
   - scored all Go files in the repository via `code_health_score`.
   - struct-only model files with unavailable score were explicitly reviewed:
     - `internal/models/oauth.go`
     - `internal/models/project.go`
     - `internal/models/collection.go`
     - `internal/models/engram.go`
     - `code_health_review` results: `score=null`, `review=[]`.
   - changed-file score highlights:
     - `/Users/mukundhan/Projects/engram/cmd/api/main.go` -> `10.0`
     - `/Users/mukundhan/Projects/engram/internal/api/session_auth.go` -> `8.28`
     - `/Users/mukundhan/Projects/engram/internal/api/session_auth_test.go` -> `10.0`
     - `/Users/mukundhan/Projects/engram/internal/auth/session.go` -> `9.38`
     - `/Users/mukundhan/Projects/engram/internal/auth/session_test.go` -> `10.0`
8. CodeScene pre-commit safeguard:
   - first run failed due complexity findings in `MountSessionAuthRoutes` and `TestSessionManagerEncodeDecodeRoundTrip`.
   - applied refactors:
     - extracted session-auth handlers/helpers in `internal/api/session_auth.go`.
     - extracted assertion helpers in `internal/auth/session_test.go`.
   - reran tests and safeguard:
     - `pre_commit_code_health_safeguard(git_repository_path=/Users/mukundhan/Projects/engram)`
     - result: `quality_gates=passed`
     - notes: `MountSessionAuthRoutes` complex-method issue fixed; remaining findings were non-blocking.

### 2026-02-22 (Go migration CP9: rehydration/source read-path parity)

1. Added rehydration/source repository operations:
   - `internal/repository/engram_rehydration.go`
   - `GetRehydrationBundle`
   - `GetEngramSources`
   - actor-scoped visibility checks for parity with Python read-path access control
2. Added response model shapes:
   - `internal/models/engram.go`
   - `RehydrationBundle`
   - `EngramSourceRecord`
3. Added rehydration/source tests:
   - `internal/repository/engram_rehydration_test.go`
   - `TestGetRehydrationBundleReturnsNilWhenNotFound`
   - `TestGetRehydrationBundleBuildsContextWithVisibilityFilter`
   - `TestGetEngramSourcesReturnsRowsWhenVisible`
   - `TestGetEngramSourcesReturnsEmptyWhenNotVisible`
4. Executed migrated tests one-by-one:
   - `go test ./internal/repository -run '^TestGetRehydrationBundleReturnsNilWhenNotFound$' -v`
   - `go test ./internal/repository -run '^TestGetRehydrationBundleBuildsContextWithVisibilityFilter$' -v`
   - `go test ./internal/repository -run '^TestGetEngramSourcesReturnsRowsWhenVisible$' -v`
   - `go test ./internal/repository -run '^TestGetEngramSourcesReturnsEmptyWhenNotVisible$' -v`
5. Full Go verification:
   - `go test ./...`
6. File-level CodeScene checks (all current Go migration files before commit):
   - `/Users/mukundhan/Projects/engram/cmd/api/main.go` -> `10.0`
   - `/Users/mukundhan/Projects/engram/internal/api/router.go` -> `10.0`
   - `/Users/mukundhan/Projects/engram/internal/api/router_test.go` -> `10.0`
   - `/Users/mukundhan/Projects/engram/internal/auth/password.go` -> `10.0`
   - `/Users/mukundhan/Projects/engram/internal/auth/password_test.go` -> `9.68`
   - `/Users/mukundhan/Projects/engram/internal/config/config.go` -> `9.68`
   - `/Users/mukundhan/Projects/engram/internal/config/config_test.go` -> `10.0`
   - `/Users/mukundhan/Projects/engram/internal/db/db.go` -> `10.0`
   - `/Users/mukundhan/Projects/engram/internal/db/db_test.go` -> `9.61`
   - `/Users/mukundhan/Projects/engram/internal/embeddings/errors.go` -> `10.0`
   - `/Users/mukundhan/Projects/engram/internal/embeddings/local.go` -> `10.0`
   - `/Users/mukundhan/Projects/engram/internal/embeddings/local_test.go` -> `9.68`
   - `/Users/mukundhan/Projects/engram/internal/embeddings/service.go` -> `9.09`
   - `/Users/mukundhan/Projects/engram/internal/embeddings/service_test.go` -> `10.0`
   - `/Users/mukundhan/Projects/engram/internal/models/engram.go` -> `N/A` (struct-only file; score unavailable)
   - `/Users/mukundhan/Projects/engram/internal/models/user.go` -> `10.0`
   - `/Users/mukundhan/Projects/engram/internal/repository/engram.go` -> `9.68`
   - `/Users/mukundhan/Projects/engram/internal/repository/engram_helpers_test.go` -> `10.0`
   - `/Users/mukundhan/Projects/engram/internal/repository/engram_rehydration.go` -> `9.68`
   - `/Users/mukundhan/Projects/engram/internal/repository/engram_rehydration_test.go` -> `10.0`
   - `/Users/mukundhan/Projects/engram/internal/repository/engram_repository_test.go` -> `10.0`
   - `/Users/mukundhan/Projects/engram/internal/repository/engram_store.go` -> `10.0`
   - `/Users/mukundhan/Projects/engram/internal/repository/engram_unit_test.go` -> `10.0`
   - `/Users/mukundhan/Projects/engram/internal/repository/user.go` -> `9.38`
   - `/Users/mukundhan/Projects/engram/internal/repository/user_test.go` -> `10.0`
7. CodeScene pre-commit safeguard:
   - `pre_commit_code_health_safeguard(git_repository_path=/Users/mukundhan/Projects/engram)`
   - result: `quality_gates=passed`
   - findings: none

### 2026-02-22 (Go migration CP10: engram write-path parity)

1. Added engram write-path repository operations:
   - `internal/repository/engram_write.go`
   - `CreateEngramWithReport`
   - `CreateEngram`
   - insert helpers for engram/source/artifact persistence.
2. Added create response model:
   - `internal/models/engram.go`
   - `EngramCreateResponse`
3. Added deterministic write seams for testability:
   - `newEngramUUID`
   - `newWriteUUID`
   - `nowUTC`
   - `embedEngramText`
   - `resolveEnrichedPayload`
4. Added write-path tests:
   - `internal/repository/engram_write_test.go`
   - `TestCreateEngramWithReportPersistsEngramSourcesAndArtifacts`
   - `TestCreateEngramWithReportReturnsEmbedErrorAndSkipsWrites`
   - `TestCreateEngramReturnsCreatedResponse`
5. Executed migrated tests one-by-one:
   - `go test ./internal/repository -run '^TestCreateEngramWithReportPersistsEngramSourcesAndArtifacts$' -v`
   - `go test ./internal/repository -run '^TestCreateEngramWithReportReturnsEmbedErrorAndSkipsWrites$' -v`
   - `go test ./internal/repository -run '^TestCreateEngramReturnsCreatedResponse$' -v`
6. Full Go verification:
   - `go test ./...`
7. File-level CodeScene checks (all current Go migration files before commit):
   - `/Users/mukundhan/Projects/engram/cmd/api/main.go` -> `10.0`
   - `/Users/mukundhan/Projects/engram/internal/api/router.go` -> `10.0`
   - `/Users/mukundhan/Projects/engram/internal/api/router_test.go` -> `10.0`
   - `/Users/mukundhan/Projects/engram/internal/auth/password.go` -> `10.0`
   - `/Users/mukundhan/Projects/engram/internal/auth/password_test.go` -> `9.68`
   - `/Users/mukundhan/Projects/engram/internal/config/config.go` -> `9.68`
   - `/Users/mukundhan/Projects/engram/internal/config/config_test.go` -> `10.0`
   - `/Users/mukundhan/Projects/engram/internal/db/db.go` -> `10.0`
   - `/Users/mukundhan/Projects/engram/internal/db/db_test.go` -> `9.61`
   - `/Users/mukundhan/Projects/engram/internal/embeddings/errors.go` -> `10.0`
   - `/Users/mukundhan/Projects/engram/internal/embeddings/local.go` -> `10.0`
   - `/Users/mukundhan/Projects/engram/internal/embeddings/local_test.go` -> `9.68`
   - `/Users/mukundhan/Projects/engram/internal/embeddings/service.go` -> `9.09`
   - `/Users/mukundhan/Projects/engram/internal/embeddings/service_test.go` -> `10.0`
   - `/Users/mukundhan/Projects/engram/internal/models/engram.go` -> `N/A` (struct-only file; score unavailable)
   - `/Users/mukundhan/Projects/engram/internal/models/user.go` -> `10.0`
   - `/Users/mukundhan/Projects/engram/internal/repository/engram.go` -> `9.68`
   - `/Users/mukundhan/Projects/engram/internal/repository/engram_helpers_test.go` -> `10.0`
   - `/Users/mukundhan/Projects/engram/internal/repository/engram_rehydration.go` -> `9.68`
   - `/Users/mukundhan/Projects/engram/internal/repository/engram_rehydration_test.go` -> `10.0`
   - `/Users/mukundhan/Projects/engram/internal/repository/engram_repository_test.go` -> `10.0`
   - `/Users/mukundhan/Projects/engram/internal/repository/engram_store.go` -> `10.0`
   - `/Users/mukundhan/Projects/engram/internal/repository/engram_unit_test.go` -> `10.0`
   - `/Users/mukundhan/Projects/engram/internal/repository/engram_write.go` -> `10.0`
   - `/Users/mukundhan/Projects/engram/internal/repository/engram_write_test.go` -> `9.26`
   - `/Users/mukundhan/Projects/engram/internal/repository/user.go` -> `9.38`
   - `/Users/mukundhan/Projects/engram/internal/repository/user_test.go` -> `10.0`
8. CodeScene pre-commit safeguard:
   - `pre_commit_code_health_safeguard(git_repository_path=/Users/mukundhan/Projects/engram)`
   - result: `quality_gates=passed`
   - findings: none

### 2026-02-22 (Go migration CP11: chat session repository baseline)

1. Added chat models and enum parsers:
   - `internal/models/chat.go`
   - `ChatProvider`, `VisibilityScope`, `ChatAutosaveStrategy`
   - `ChatSessionCreateRequest`, `ChatSessionUpdateRequest`, `ChatSessionRecord`
2. Added chat session repository operations:
   - `internal/repository/chat.go`
   - `CreateChatSession`
   - `ListChatSessions`
   - `GetChatSession`
   - `UpdateChatSession`
3. Added chat repository tests:
   - `internal/repository/chat_test.go`
   - `TestCreateChatSessionReturnsInsertedRecord`
   - `TestListChatSessionsAppliesVisibilityAndProjectFilter`
   - `TestGetChatSessionReturnsNilWhenMissing`
   - `TestUpdateChatSessionReturnsNilWhenNoRows`
   - `TestUpdateChatSessionValidatesProviderValue`
   - `TestNormalizeCreatePayloadRejectsInvalidVisibility`
4. Executed migrated tests one-by-one:
   - `go test ./internal/repository -run '^TestCreateChatSessionReturnsInsertedRecord$' -v`
   - `go test ./internal/repository -run '^TestListChatSessionsAppliesVisibilityAndProjectFilter$' -v`
   - `go test ./internal/repository -run '^TestGetChatSessionReturnsNilWhenMissing$' -v`
   - `go test ./internal/repository -run '^TestUpdateChatSessionReturnsNilWhenNoRows$' -v`
   - `go test ./internal/repository -run '^TestUpdateChatSessionValidatesProviderValue$' -v`
   - `go test ./internal/repository -run '^TestNormalizeCreatePayloadRejectsInvalidVisibility$' -v`
5. Full Go verification:
   - `go test ./...`
6. File-level CodeScene checks (all current Go migration files before commit):
   - `/Users/mukundhan/Projects/engram/cmd/api/main.go` -> `10.0`
   - `/Users/mukundhan/Projects/engram/internal/api/router.go` -> `10.0`
   - `/Users/mukundhan/Projects/engram/internal/api/router_test.go` -> `10.0`
   - `/Users/mukundhan/Projects/engram/internal/auth/password.go` -> `10.0`
   - `/Users/mukundhan/Projects/engram/internal/auth/password_test.go` -> `9.68`
   - `/Users/mukundhan/Projects/engram/internal/config/config.go` -> `9.68`
   - `/Users/mukundhan/Projects/engram/internal/config/config_test.go` -> `10.0`
   - `/Users/mukundhan/Projects/engram/internal/db/db.go` -> `10.0`
   - `/Users/mukundhan/Projects/engram/internal/db/db_test.go` -> `9.61`
   - `/Users/mukundhan/Projects/engram/internal/embeddings/errors.go` -> `10.0`
   - `/Users/mukundhan/Projects/engram/internal/embeddings/local.go` -> `10.0`
   - `/Users/mukundhan/Projects/engram/internal/embeddings/local_test.go` -> `9.68`
   - `/Users/mukundhan/Projects/engram/internal/embeddings/service.go` -> `9.09`
   - `/Users/mukundhan/Projects/engram/internal/embeddings/service_test.go` -> `10.0`
   - `/Users/mukundhan/Projects/engram/internal/models/chat.go` -> `10.0`
   - `/Users/mukundhan/Projects/engram/internal/models/engram.go` -> `N/A` (struct-only file; score unavailable)
   - `/Users/mukundhan/Projects/engram/internal/models/user.go` -> `10.0`
   - `/Users/mukundhan/Projects/engram/internal/repository/chat.go` -> `9.02`
   - `/Users/mukundhan/Projects/engram/internal/repository/chat_test.go` -> `9.09`
   - `/Users/mukundhan/Projects/engram/internal/repository/engram.go` -> `9.68`
   - `/Users/mukundhan/Projects/engram/internal/repository/engram_helpers_test.go` -> `10.0`
   - `/Users/mukundhan/Projects/engram/internal/repository/engram_rehydration.go` -> `9.68`
   - `/Users/mukundhan/Projects/engram/internal/repository/engram_rehydration_test.go` -> `10.0`
   - `/Users/mukundhan/Projects/engram/internal/repository/engram_repository_test.go` -> `10.0`
   - `/Users/mukundhan/Projects/engram/internal/repository/engram_store.go` -> `10.0`
   - `/Users/mukundhan/Projects/engram/internal/repository/engram_unit_test.go` -> `10.0`
   - `/Users/mukundhan/Projects/engram/internal/repository/engram_write.go` -> `10.0`
   - `/Users/mukundhan/Projects/engram/internal/repository/engram_write_test.go` -> `9.26`
   - `/Users/mukundhan/Projects/engram/internal/repository/user.go` -> `9.38`
   - `/Users/mukundhan/Projects/engram/internal/repository/user_test.go` -> `10.0`
7. CodeScene pre-commit safeguard:
   - `pre_commit_code_health_safeguard(git_repository_path=/Users/mukundhan/Projects/engram)`
   - result: `quality_gates=passed`
   - findings: none

### 2026-02-22 (Go migration CP12: project repository baseline)

1. Added project model:
   - `internal/models/project.go`
   - `ProjectRecord`
2. Added project repository operations:
   - `internal/repository/project.go`
   - `ListProjectsForActor`
   - `GetProjectForActor`
   - `CreateProject`
   - `EnsureProjectExists`
   - `GetUserDefaultProjectID`
   - `SetUserDefaultProjectID`
3. Added project repository tests:
   - `internal/repository/project_test.go`
   - `TestListProjectsForActorScopesNonAdminAndArchived`
   - `TestGetProjectForActorReturnsNilWhenMissing`
   - `TestCreateProjectReturnsRecord`
   - `TestEnsureProjectExistsReturnsExistingBeforeCreate`
   - `TestGetUserDefaultProjectIDReturnsValueAndNil`
   - `TestSetUserDefaultProjectIDReturnsBool`
4. Executed migrated tests one-by-one:
   - `go test ./internal/repository -run '^TestListProjectsForActorScopesNonAdminAndArchived$' -v`
   - `go test ./internal/repository -run '^TestGetProjectForActorReturnsNilWhenMissing$' -v`
   - `go test ./internal/repository -run '^TestCreateProjectReturnsRecord$' -v`
   - `go test ./internal/repository -run '^TestEnsureProjectExistsReturnsExistingBeforeCreate$' -v`
   - `go test ./internal/repository -run '^TestGetUserDefaultProjectIDReturnsValueAndNil$' -v`
   - `go test ./internal/repository -run '^TestSetUserDefaultProjectIDReturnsBool$' -v`
5. Full Go verification:
   - `go test ./...`
6. File-level CodeScene checks (all current Go migration files before commit):
   - `/Users/mukundhan/Projects/engram/cmd/api/main.go` -> `10.0`
   - `/Users/mukundhan/Projects/engram/internal/api/router.go` -> `10.0`
   - `/Users/mukundhan/Projects/engram/internal/api/router_test.go` -> `10.0`
   - `/Users/mukundhan/Projects/engram/internal/auth/password.go` -> `10.0`
   - `/Users/mukundhan/Projects/engram/internal/auth/password_test.go` -> `9.68`
   - `/Users/mukundhan/Projects/engram/internal/config/config.go` -> `9.68`
   - `/Users/mukundhan/Projects/engram/internal/config/config_test.go` -> `10.0`
   - `/Users/mukundhan/Projects/engram/internal/db/db.go` -> `10.0`
   - `/Users/mukundhan/Projects/engram/internal/db/db_test.go` -> `9.61`
   - `/Users/mukundhan/Projects/engram/internal/embeddings/errors.go` -> `10.0`
   - `/Users/mukundhan/Projects/engram/internal/embeddings/local.go` -> `10.0`
   - `/Users/mukundhan/Projects/engram/internal/embeddings/local_test.go` -> `9.68`
   - `/Users/mukundhan/Projects/engram/internal/embeddings/service.go` -> `9.09`
   - `/Users/mukundhan/Projects/engram/internal/embeddings/service_test.go` -> `10.0`
   - `/Users/mukundhan/Projects/engram/internal/models/chat.go` -> `10.0`
   - `/Users/mukundhan/Projects/engram/internal/models/engram.go` -> `N/A` (struct-only file; score unavailable)
   - `/Users/mukundhan/Projects/engram/internal/models/project.go` -> `N/A` (struct-only file; score unavailable)
   - `/Users/mukundhan/Projects/engram/internal/models/user.go` -> `10.0`
   - `/Users/mukundhan/Projects/engram/internal/repository/chat.go` -> `9.02`
   - `/Users/mukundhan/Projects/engram/internal/repository/chat_test.go` -> `9.09`
   - `/Users/mukundhan/Projects/engram/internal/repository/engram.go` -> `9.68`
   - `/Users/mukundhan/Projects/engram/internal/repository/engram_helpers_test.go` -> `10.0`
   - `/Users/mukundhan/Projects/engram/internal/repository/engram_rehydration.go` -> `9.68`
   - `/Users/mukundhan/Projects/engram/internal/repository/engram_rehydration_test.go` -> `10.0`
   - `/Users/mukundhan/Projects/engram/internal/repository/engram_repository_test.go` -> `10.0`
   - `/Users/mukundhan/Projects/engram/internal/repository/engram_store.go` -> `10.0`
   - `/Users/mukundhan/Projects/engram/internal/repository/engram_unit_test.go` -> `10.0`
   - `/Users/mukundhan/Projects/engram/internal/repository/engram_write.go` -> `10.0`
   - `/Users/mukundhan/Projects/engram/internal/repository/engram_write_test.go` -> `9.26`
   - `/Users/mukundhan/Projects/engram/internal/repository/project.go` -> `10.0`
   - `/Users/mukundhan/Projects/engram/internal/repository/project_test.go` -> `10.0`
   - `/Users/mukundhan/Projects/engram/internal/repository/user.go` -> `9.38`
   - `/Users/mukundhan/Projects/engram/internal/repository/user_test.go` -> `10.0`
7. CodeScene pre-commit safeguard:
   - `pre_commit_code_health_safeguard(git_repository_path=/Users/mukundhan/Projects/engram)`
   - result: `quality_gates=passed`
   - findings: none

### 2026-02-22 (CodeScene pre-commit safeguard for immediate-phase closeout bundle)

1. Ran CodeScene MCP pre-commit health gate on the working tree:
   - `pre_commit_code_health_safeguard(git_repository_path=/Users/mukundhan/Projects/engram)`
  - result: `quality_gates=passed`
  - findings: none

### 2026-02-22 (Phase 34 security audit remediation program)

1. Implemented production config hardening with fail-fast validation:
   - `api/app/config.py`
   - production startup now rejects insecure defaults/short values for:
     - `APP_SESSION_SECRET`
     - `MCP_TOKEN_PEPPER`
     - `OAUTH_CLIENT_SECRET_PEPPER`
     - `UI_DEMO_PASSWORD`
   - added `OAUTH_REQUIRE_PROTECTED_REGISTRATION` enforcement for production.
2. Hardened bootstrap auth behavior for production deployments:
   - `api/app/db.py`
   - if seeded bootstrap admin hash is detected in production, rotate to configured secure password/hash during schema initialization.
3. Added distributed rate limiting backed by `rate_limit_state`:
   - `db/init/001_schema.sql` (`rate_limit_state` table + indexes)
   - `api/app/login_guard.py` now supports DB-backed login lockout state.
   - added `RequestRateLimiter` and wired MCP transport throttling into:
     - `api/app/main.py`
     - `api/app/mcp/api.py`
4. Applied OAuth hardening:
   - `PKCE S256` only (`plain` removed from supported methods).
   - dynamic client registration (`POST /oauth/register`) now requires an authenticated admin session when protected registration is enabled.
5. Applied ingestion and audit hardening:
   - ingestion file MIME allowlist + metadata JSON size guard.
   - audit logging now sanitizes payloads, truncates oversized events, and supports stdout emission for centralized log collection.
6. Added security-focused regression coverage:
   - API/unit/integration:
     - `api/tests/test_config_settings.py`
     - `api/tests/test_db.py`
     - `api/tests/test_login_guard.py`
     - `api/tests/test_mcp_api_integration.py`
     - `api/tests/test_oauth_api_unit.py`
     - `api/tests/test_oauth_service.py`
     - `api/tests/test_mcp_oauth_integration.py`
     - `api/tests/test_ingestion_service.py`
     - `api/tests/test_ingestion_api_integration.py`
     - `api/tests/test_audit.py`
   - acceptance:
     - `acceptance-tests/features/authentication.feature`
     - `acceptance-tests/src/steps/auth.steps.ts`
7. Updated docs and env references:
   - `.env.example`
   - `docs/env-reference.md`
   - `docs/api-reference.md`
   - `docs/mcp-guide.md`
   - `AGENT.md` security contracts.
8. Verification:
   - `cd api && uv run ruff check ...` (all touched backend/test files)
   - `cd api && uv run pytest -q tests/test_config_settings.py tests/test_oauth_service.py tests/test_oauth_api_unit.py tests/test_mcp_oauth_integration.py tests/test_login_guard.py tests/test_mcp_api_integration.py tests/test_ingestion_service.py tests/test_ingestion_api_integration.py tests/test_audit.py tests/test_db.py`
   - `cd acceptance-tests && npm run bdd:gen`
   - `cd acceptance-tests && npm run typecheck`

### 2026-02-22 (Phase 16 CLI smoke utility: `engram-cli mcp-call`)

1. Added a new CLI subcommand in `api/app/cli.py`:
   - `engram-cli mcp-call`
   - supports:
     - `--method` (required)
     - `--params-json` or `--params-file`
     - `--request-id`
     - `--base-url`
     - bearer auth (`--bearer-token`) or form login (`--username` + `--password`)
2. Integrated CLI smoke call with the typed MCP client:
   - uses `McpSseClient` transport against `/api/v1/mcp/stream`
   - prints request + parsed JSON-RPC frames as JSON output for debugging
   - returns non-zero when an RPC error frame is returned
3. Added CLI regression tests in `api/tests/test_cli.py` for:
   - bearer-token MCP call path
   - username/password session-login MCP call path
   - explicit auth validation failure path
4. Updated docs:
   - `docs/user-workflows.md` (MCP smoke call examples)
   - `docs/testing-guide.md` (`test_cli.py` coverage row)
5. Validation:
   - `cd api && uv run pytest -q tests/test_cli.py`
   - `cd api && uv run ruff check app/cli.py tests/test_cli.py`

### 2026-02-22 (Phase 18 timeline consolidation semantics follow-up)

1. Added explicit consolidation timeline semantics in lifecycle policy:
   - `api/app/chat/lifecycle_policy.py` now parses consolidation tag metadata:
     - `consolidation_group_key:<value>`
     - `consolidation_merged_count:<int>`
   - added `TimelineEventSemantics` and expanded event-type classification:
     - `consolidation`
     - `consolidation_group`
     - `consolidation_merge`
2. Extended timeline response contract:
   - `api/app/models.py::ChatTimelineEvent` now includes optional
     - `consolidation_group_key`
     - `consolidation_merged_count`
   - `api/app/chat/session_operations.py` now emits those semantics per event.
3. Implemented timeline grouping semantics in UI rendering:
   - `web/src/components/ChatPanel.tsx` now groups consolidation events sharing a `consolidation_group_key` into a single rendered timeline row with grouped/merged counts.
   - added explicit user-facing labels for lifecycle event types.
4. Added regression coverage:
   - `api/tests/test_chat_lifecycle_policy.py`
   - `api/tests/test_chat_session_operations.py`
   - `web/src/components/ChatPanel.test.tsx`
5. Validation:
   - `cd api && uv run pytest -q tests/test_chat_lifecycle_policy.py tests/test_chat_session_operations.py`
   - `cd api && uv run ruff check app/chat/lifecycle_policy.py app/chat/session_operations.py tests/test_chat_lifecycle_policy.py tests/test_chat_session_operations.py`
   - `cd web && npm run test -- src/components/ChatPanel.test.tsx`
   - `cd web && npm run lint`

### 2026-02-22 (Phase 31 closeout docs + acceptance validation evidence)

1. Finalized Phase 31 contributor contracts in `AGENT.md`:
   - explicit default-project write resolution invariants.
   - explicit soft-delete/restore invariants for sessions/engrams/collections.
   - explicit collection project-boundary invariants (including auto-detach on cross-project engram moves).
   - explicit MCP organization tool contract set and dotted/underscored naming compatibility.
2. Updated skill docs with the same Phase 31 invariants:
   - `skills/mcp-http-stream-tools/SKILL.md`
   - `skills/engram-lifecycle/SKILL.md`
   - `skills/testing-and-evals/SKILL.md`
3. Fixed deterministic acceptance coverage for transfer-page suggestion checks:
   - `acceptance-tests/src/steps/project-export-import.steps.ts`
   - the dataset prep step now sets workspace project to the prepared export project before suggestion assertions, matching project-scoped collection suggestion behavior.
4. Validation evidence:
   - `make acceptance-test-mock` -> `18 passed` (includes `@phase31 @memory-admin` scenarios).

### 2026-02-22 (Phase 33 web export/import page + test coverage)

1. Added a dedicated Phase 33 project transfer page in `web/src/components/ProjectTransferPage.tsx`:
   - new export form (format, optional collection subset, optional embeddings flag).
   - new import form (bundle upload + deterministic conflict policy selector).
   - import result summary panel with per-counter visibility.
2. Added frontend export/import API client in `web/src/api/export.ts`:
   - `exportProjectBundle(...)` handles query construction, authenticated download, and filename parsing.
   - `importProjectBundle(...)` handles multipart upload with conflict-policy query.
3. Wired route and navigation integration:
   - `web/src/App.tsx` new route: `/projects/transfer`.
   - `web/src/components/WorkspaceTopNav.tsx` new top-nav action: `Export / Import`.
4. Added frontend regression tests:
   - `web/src/components/ProjectTransferPage.test.tsx`
   - `web/src/api/export.test.ts`
   - updated `web/src/components/WorkspaceTopNav.test.tsx`
5. Added acceptance scenarios for export/import options using real API endpoints:
   - creates real projects/engrams/collections during setup.
   - covers separate scenarios for `include_embeddings`, `collection_ids`, `json/zip` export formats, and import conflict policies (`skip`, `rename`, `overwrite`).
   - files:
     - `acceptance-tests/features/project-export-import.feature`
     - `acceptance-tests/src/steps/project-export-import.steps.ts`
6. Verification:
   - `cd web && npm run lint`
   - `cd web && npm run test`
   - `cd web && npm run build`
   - `cd acceptance-tests && npm run bdd:gen`
   - `cd acceptance-tests && npm run typecheck`
7. Tightened contribution contract in `AGENT.md`:
   - CodeScene pre-commit safeguard is now required before **every** commit (via `skills/codescene/SKILL.md` workflow), not only non-trivial changes.
8. CodeScene pre-commit safeguard re-run after acceptance-step refactor:
   - `pre_commit_code_health_safeguard` -> `quality_gates=passed`.
   - remaining finding: minor duplication across three export step wrappers in `acceptance-tests/src/steps/project-export-import.steps.ts`; retained for readability of explicit scenario-step mapping.

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
3. Updated `migration/checkpoints/checkpoint.md` current summary and phase timeline to include:
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

### 2026-02-22 (UI smart ID search rollout - project + collection fields)

1. Added reusable searchable dropdown component in `web/src/components/SmartIdDropdown.tsx`:
   - ranked suggestion matching (prefix before contains),
   - de-duplicated option lists,
   - optional match-count visibility for compact toolbars.
2. Applied smart ID search in all relevant project/collection input surfaces:
   - `web/src/components/SessionSidebar.tsx` project ID selector,
   - `web/src/components/ProjectTransferPage.tsx` export/import project IDs and export collection ID selector,
   - `web/src/components/AdminMemoryPage.tsx` session project filter, engram move-target project, and collection project input.
3. Improved export collection ID UX in `web/src/components/ProjectTransferPage.tsx`:
   - searchable collection ID input with add/remove chips,
   - support for comma/whitespace paste and dedupe,
   - persisted behavior where typed-but-not-added IDs are still honored on export submit.
4. Added/updated tests:
   - `web/src/components/SmartIdDropdown.test.tsx`
   - `web/src/components/SessionSidebar.test.tsx`
   - `web/src/components/ProjectTransferPage.test.tsx`
   - `web/src/components/AdminMemoryPage.test.tsx`
   - `acceptance-tests/features/project-export-import.feature`
   - `acceptance-tests/src/steps/project-export-import.steps.ts`
5. Validation:
   - `npm run test` in `web/` passed (`35 files`, `134 tests`).
   - `npm run lint` in `web/` passed.
   - `npm run bdd:gen` in `acceptance-tests/` passed.
   - `npm run typecheck` in `acceptance-tests/` passed.
6. CodeScene pre-commit safeguard:
   - initial run flagged `Large Method` and `Code Duplication`.
   - follow-up refactor reduced `CollectionSection` size and removed duplicated test structure.
   - final `pre_commit_code_health_safeguard` result: `quality_gates=passed`.

### 2026-02-22 (Go migration CP13: chat messages + session pinning repository parity)

1. Extended chat model parity in Go:
   - `internal/models/chat.go`
   - added `ChatSessionAdminRecord`, `ChatMessageRecord`, `PinnedEngramRecord`, `PinnedDocumentRecord`.
2. Added chat repository continuation operations:
   - `internal/repository/chat.go`
   - added `GetChatSessionAdminRecord` parity read path.
3. Added message repository operations:
   - `internal/repository/chat_message.go`
   - `CreateChatMessage`
   - `ListChatMessages`
   - `ListSessionLinkedEngrams`
   - `CountSessionMessagesByRole`
   - `DeleteSessionAutosaveEngrams`
4. Added session pinning repository operations:
   - `internal/repository/chat_pinning.go`
   - `PinEngramToSession`
   - `UnpinEngramFromSession`
   - `ListPinnedEngrams`
   - `ListPinnedEngramSummaries`
   - `PinDocumentToSession`
   - `UnpinDocumentFromSession`
   - `ListPinnedDocuments`
5. Added migrated repository tests:
   - `internal/repository/chat_message_test.go`
   - `internal/repository/chat_pinning_test.go`
   - `internal/repository/chat_test.go` (`TestGetChatSessionAdminRecordReturnsRecord`)
6. Executed migrated tests one-by-one:
   - `go test ./internal/repository -run '^TestCreateChatMessageReturnsInsertedRecord$' -v`
   - `go test ./internal/repository -run '^TestCreateChatMessageReturnsNilWhenSessionNotVisible$' -v`
   - `go test ./internal/repository -run '^TestListChatMessagesParsesJSONAndDefaults$' -v`
   - `go test ./internal/repository -run '^TestListSessionLinkedEngramsAppliesVisibilityFilters$' -v`
   - `go test ./internal/repository -run '^TestCountSessionMessagesByRoleReturnsValue$' -v`
   - `go test ./internal/repository -run '^TestDeleteSessionAutosaveEngramsReturnsDeletedIDs$' -v`
   - `go test ./internal/repository -run '^TestDeleteSessionAutosaveEngramsSkipsQueryWhenNoIDs$' -v`
   - `go test ./internal/repository -run '^TestPinEngramToSessionReturnsPinnedRecord$' -v`
   - `go test ./internal/repository -run '^TestPinDocumentToSessionReturnsNilWhenResourceNotVisible$' -v`
   - `go test ./internal/repository -run '^TestUnpinEngramFromSessionReturnsTrueWhenRemoved$' -v`
   - `go test ./internal/repository -run '^TestUnpinDocumentFromSessionReturnsFalseWhenMissing$' -v`
   - `go test ./internal/repository -run '^TestListPinnedEngramsReturnsRows$' -v`
   - `go test ./internal/repository -run '^TestListPinnedDocumentsAppliesVisibilityFilter$' -v`
   - `go test ./internal/repository -run '^TestListPinnedEngramSummariesAppliesVisibilityFilters$' -v`
   - `go test ./internal/repository -run '^TestGetChatSessionAdminRecordReturnsRecord$' -v`
7. Full Go verification:
   - `go test ./...` passed.
8. File-level CodeScene checks (all Go migration files before commit):
   - `/Users/mukundhan/Projects/engram/cmd/api/main.go` -> `10.0`
   - `/Users/mukundhan/Projects/engram/internal/api/router.go` -> `10.0`
   - `/Users/mukundhan/Projects/engram/internal/api/router_test.go` -> `10.0`
   - `/Users/mukundhan/Projects/engram/internal/auth/password.go` -> `10.0`
   - `/Users/mukundhan/Projects/engram/internal/auth/password_test.go` -> `9.68`
   - `/Users/mukundhan/Projects/engram/internal/config/config.go` -> `9.68`
   - `/Users/mukundhan/Projects/engram/internal/config/config_test.go` -> `10.0`
   - `/Users/mukundhan/Projects/engram/internal/db/db.go` -> `10.0`
   - `/Users/mukundhan/Projects/engram/internal/db/db_test.go` -> `9.61`
   - `/Users/mukundhan/Projects/engram/internal/embeddings/errors.go` -> `10.0`
   - `/Users/mukundhan/Projects/engram/internal/embeddings/local.go` -> `10.0`
   - `/Users/mukundhan/Projects/engram/internal/embeddings/local_test.go` -> `9.68`
   - `/Users/mukundhan/Projects/engram/internal/embeddings/service.go` -> `9.09`
   - `/Users/mukundhan/Projects/engram/internal/embeddings/service_test.go` -> `10.0`
   - `/Users/mukundhan/Projects/engram/internal/models/chat.go` -> `10.0`
   - `/Users/mukundhan/Projects/engram/internal/models/engram.go` -> `None` (struct-only file; score unavailable)
   - `/Users/mukundhan/Projects/engram/internal/models/project.go` -> `None` (struct-only file; score unavailable)
   - `/Users/mukundhan/Projects/engram/internal/models/user.go` -> `10.0`
   - `/Users/mukundhan/Projects/engram/internal/repository/chat.go` -> `9.02`
   - `/Users/mukundhan/Projects/engram/internal/repository/chat_message.go` -> `10.0`
   - `/Users/mukundhan/Projects/engram/internal/repository/chat_message_test.go` -> `10.0`
   - `/Users/mukundhan/Projects/engram/internal/repository/chat_pinning.go` -> `8.81`
   - `/Users/mukundhan/Projects/engram/internal/repository/chat_pinning_test.go` -> `9.38`
   - `/Users/mukundhan/Projects/engram/internal/repository/chat_test.go` -> `9.09`
   - `/Users/mukundhan/Projects/engram/internal/repository/engram.go` -> `9.68`
   - `/Users/mukundhan/Projects/engram/internal/repository/engram_helpers_test.go` -> `10.0`
   - `/Users/mukundhan/Projects/engram/internal/repository/engram_rehydration.go` -> `9.68`
   - `/Users/mukundhan/Projects/engram/internal/repository/engram_rehydration_test.go` -> `10.0`
   - `/Users/mukundhan/Projects/engram/internal/repository/engram_repository_test.go` -> `10.0`
   - `/Users/mukundhan/Projects/engram/internal/repository/engram_store.go` -> `10.0`
   - `/Users/mukundhan/Projects/engram/internal/repository/engram_unit_test.go` -> `10.0`
   - `/Users/mukundhan/Projects/engram/internal/repository/engram_write.go` -> `10.0`
   - `/Users/mukundhan/Projects/engram/internal/repository/engram_write_test.go` -> `9.26`
   - `/Users/mukundhan/Projects/engram/internal/repository/project.go` -> `10.0`
   - `/Users/mukundhan/Projects/engram/internal/repository/project_test.go` -> `10.0`
   - `/Users/mukundhan/Projects/engram/internal/repository/user.go` -> `9.38`
   - `/Users/mukundhan/Projects/engram/internal/repository/user_test.go` -> `10.0`
9. CodeScene pre-commit safeguard:
   - `pre_commit_code_health_safeguard(git_repository_path=/Users/mukundhan/Projects/engram)`
   - result: `quality_gates=passed`
   - findings: none

### 2026-02-22 (Go migration CP14: document repository baseline parity)

1. Added document model parity in Go:
   - `internal/models/document.go`
   - `DocumentSourceType` + parser
   - `DocumentRecord`
   - `DocumentChunkQueryRequest`
   - `DocumentChunkQueryResult`
2. Added document repository baseline:
   - `internal/repository/document.go`
   - `UpsertDocumentWithChunks`
   - `ListDocuments`
   - `QueryDocumentChunks`
   - helper parity for chunk replacement, query filtering, and lexical reranking.
3. Added migrated repository tests:
   - `internal/repository/document_test.go`
4. Executed migrated tests one-by-one:
   - `go test ./internal/repository -run '^TestUpsertDocumentWithChunksPersistsDocumentAndChunks$' -v`
   - `go test ./internal/repository -run '^TestUpsertDocumentWithChunksFailsOnEmbeddingCountMismatch$' -v`
   - `go test ./internal/repository -run '^TestListDocumentsAppliesProjectFilter$' -v`
   - `go test ./internal/repository -run '^TestQueryDocumentChunksBuildsQueryAndReranks$' -v`
   - `go test ./internal/repository -run '^TestBuildDocumentChunkWhereDefaultsToActorScopeOnly$' -v`
   - `go test ./internal/repository -run '^TestUpsertDocumentWithChunksRejectsInvalidVisibility$' -v`
5. Full Go verification:
   - `go test ./...` passed.
6. File-level CodeScene checks (all Go migration files before commit):
   - `/Users/mukundhan/Projects/engram/cmd/api/main.go` -> `10.0`
   - `/Users/mukundhan/Projects/engram/internal/api/router.go` -> `10.0`
   - `/Users/mukundhan/Projects/engram/internal/api/router_test.go` -> `10.0`
   - `/Users/mukundhan/Projects/engram/internal/auth/password.go` -> `10.0`
   - `/Users/mukundhan/Projects/engram/internal/auth/password_test.go` -> `9.68`
   - `/Users/mukundhan/Projects/engram/internal/config/config.go` -> `9.68`
   - `/Users/mukundhan/Projects/engram/internal/config/config_test.go` -> `10.0`
   - `/Users/mukundhan/Projects/engram/internal/db/db.go` -> `10.0`
   - `/Users/mukundhan/Projects/engram/internal/db/db_test.go` -> `9.61`
   - `/Users/mukundhan/Projects/engram/internal/embeddings/errors.go` -> `10.0`
   - `/Users/mukundhan/Projects/engram/internal/embeddings/local.go` -> `10.0`
   - `/Users/mukundhan/Projects/engram/internal/embeddings/local_test.go` -> `9.68`
   - `/Users/mukundhan/Projects/engram/internal/embeddings/service.go` -> `9.09`
   - `/Users/mukundhan/Projects/engram/internal/embeddings/service_test.go` -> `10.0`
   - `/Users/mukundhan/Projects/engram/internal/models/chat.go` -> `10.0`
   - `/Users/mukundhan/Projects/engram/internal/models/document.go` -> `10.0`
   - `/Users/mukundhan/Projects/engram/internal/models/engram.go` -> `None` (struct-only file; score unavailable)
   - `/Users/mukundhan/Projects/engram/internal/models/project.go` -> `None` (struct-only file; score unavailable)
   - `/Users/mukundhan/Projects/engram/internal/models/user.go` -> `10.0`
   - `/Users/mukundhan/Projects/engram/internal/repository/chat.go` -> `9.02`
   - `/Users/mukundhan/Projects/engram/internal/repository/chat_message.go` -> `10.0`
   - `/Users/mukundhan/Projects/engram/internal/repository/chat_message_test.go` -> `10.0`
   - `/Users/mukundhan/Projects/engram/internal/repository/chat_pinning.go` -> `8.81`
   - `/Users/mukundhan/Projects/engram/internal/repository/chat_pinning_test.go` -> `9.38`
   - `/Users/mukundhan/Projects/engram/internal/repository/chat_test.go` -> `9.09`
   - `/Users/mukundhan/Projects/engram/internal/repository/document.go` -> `8.81`
   - `/Users/mukundhan/Projects/engram/internal/repository/document_test.go` -> `8.72`
   - `/Users/mukundhan/Projects/engram/internal/repository/engram.go` -> `9.68`
   - `/Users/mukundhan/Projects/engram/internal/repository/engram_helpers_test.go` -> `10.0`
   - `/Users/mukundhan/Projects/engram/internal/repository/engram_rehydration.go` -> `9.68`
   - `/Users/mukundhan/Projects/engram/internal/repository/engram_rehydration_test.go` -> `10.0`
   - `/Users/mukundhan/Projects/engram/internal/repository/engram_repository_test.go` -> `10.0`
   - `/Users/mukundhan/Projects/engram/internal/repository/engram_store.go` -> `10.0`
   - `/Users/mukundhan/Projects/engram/internal/repository/engram_unit_test.go` -> `10.0`
   - `/Users/mukundhan/Projects/engram/internal/repository/engram_write.go` -> `10.0`
   - `/Users/mukundhan/Projects/engram/internal/repository/engram_write_test.go` -> `9.26`
   - `/Users/mukundhan/Projects/engram/internal/repository/project.go` -> `10.0`
   - `/Users/mukundhan/Projects/engram/internal/repository/project_test.go` -> `10.0`
   - `/Users/mukundhan/Projects/engram/internal/repository/user.go` -> `9.38`
   - `/Users/mukundhan/Projects/engram/internal/repository/user_test.go` -> `10.0`
7. CodeScene pre-commit safeguard:
   - `pre_commit_code_health_safeguard(git_repository_path=/Users/mukundhan/Projects/engram)`
   - result: `quality_gates=passed`
   - findings: none

### 2026-02-22 (Go migration CP15: MCP token repository baseline parity)

1. Added MCP token model parity in Go:
   - `internal/models/mcp_token.go`
   - `MCPTokenScope` + parser
   - `MCPTokenRecord`
   - `MCPTokenAuthContext`
2. Added MCP token repository baseline:
   - `internal/repository/mcp_token.go`
   - `CreateMCPToken`
   - `ListMCPTokens`
   - `GetMCPTokenByID`
   - `RevokeMCPToken`
   - `TouchMCPTokenLastUsed`
3. Added migrated repository tests:
   - `internal/repository/mcp_token_test.go`
4. Executed migrated tests one-by-one:
   - `go test ./internal/repository -run '^TestCreateMCPTokenUsesGeneratedIDAndReturnsRecord$' -v`
   - `go test ./internal/repository -run '^TestCreateMCPTokenRejectsInvalidScope$' -v`
   - `go test ./internal/repository -run '^TestListMCPTokensReturnsDefaultsForNilArrays$' -v`
   - `go test ./internal/repository -run '^TestGetMCPTokenByIDReturnsNilWhenMissing$' -v`
   - `go test ./internal/repository -run '^TestRevokeMCPTokenReturnsNilWhenMissing$' -v`
   - `go test ./internal/repository -run '^TestTouchMCPTokenLastUsedUsesCurrentTimestamp$' -v`
5. Full Go verification:
   - `go test ./...` passed.
6. File-level CodeScene checks (all Go migration files before commit):
   - `/Users/mukundhan/Projects/engram/cmd/api/main.go` -> `10.0`
   - `/Users/mukundhan/Projects/engram/internal/api/router.go` -> `10.0`
   - `/Users/mukundhan/Projects/engram/internal/api/router_test.go` -> `10.0`
   - `/Users/mukundhan/Projects/engram/internal/auth/password.go` -> `10.0`
   - `/Users/mukundhan/Projects/engram/internal/auth/password_test.go` -> `9.68`
   - `/Users/mukundhan/Projects/engram/internal/config/config.go` -> `9.68`
   - `/Users/mukundhan/Projects/engram/internal/config/config_test.go` -> `10.0`
   - `/Users/mukundhan/Projects/engram/internal/db/db.go` -> `10.0`
   - `/Users/mukundhan/Projects/engram/internal/db/db_test.go` -> `9.61`
   - `/Users/mukundhan/Projects/engram/internal/embeddings/errors.go` -> `10.0`
   - `/Users/mukundhan/Projects/engram/internal/embeddings/local.go` -> `10.0`
   - `/Users/mukundhan/Projects/engram/internal/embeddings/local_test.go` -> `9.68`
   - `/Users/mukundhan/Projects/engram/internal/embeddings/service.go` -> `9.09`
   - `/Users/mukundhan/Projects/engram/internal/embeddings/service_test.go` -> `10.0`
   - `/Users/mukundhan/Projects/engram/internal/models/chat.go` -> `10.0`
   - `/Users/mukundhan/Projects/engram/internal/models/document.go` -> `10.0`
   - `/Users/mukundhan/Projects/engram/internal/models/engram.go` -> `None` (struct-only file; score unavailable)
   - `/Users/mukundhan/Projects/engram/internal/models/mcp_token.go` -> `10.0`
   - `/Users/mukundhan/Projects/engram/internal/models/project.go` -> `None` (struct-only file; score unavailable)
   - `/Users/mukundhan/Projects/engram/internal/models/user.go` -> `10.0`
   - `/Users/mukundhan/Projects/engram/internal/repository/chat.go` -> `9.02`
   - `/Users/mukundhan/Projects/engram/internal/repository/chat_message.go` -> `10.0`
   - `/Users/mukundhan/Projects/engram/internal/repository/chat_message_test.go` -> `10.0`
   - `/Users/mukundhan/Projects/engram/internal/repository/chat_pinning.go` -> `8.81`
   - `/Users/mukundhan/Projects/engram/internal/repository/chat_pinning_test.go` -> `9.38`
   - `/Users/mukundhan/Projects/engram/internal/repository/chat_test.go` -> `9.09`
   - `/Users/mukundhan/Projects/engram/internal/repository/document.go` -> `8.81`
   - `/Users/mukundhan/Projects/engram/internal/repository/document_test.go` -> `8.72`
   - `/Users/mukundhan/Projects/engram/internal/repository/engram.go` -> `9.68`
   - `/Users/mukundhan/Projects/engram/internal/repository/engram_helpers_test.go` -> `10.0`
   - `/Users/mukundhan/Projects/engram/internal/repository/engram_rehydration.go` -> `9.68`
   - `/Users/mukundhan/Projects/engram/internal/repository/engram_rehydration_test.go` -> `10.0`
   - `/Users/mukundhan/Projects/engram/internal/repository/engram_repository_test.go` -> `10.0`
   - `/Users/mukundhan/Projects/engram/internal/repository/engram_store.go` -> `10.0`
   - `/Users/mukundhan/Projects/engram/internal/repository/engram_unit_test.go` -> `10.0`
   - `/Users/mukundhan/Projects/engram/internal/repository/engram_write.go` -> `10.0`
   - `/Users/mukundhan/Projects/engram/internal/repository/engram_write_test.go` -> `9.26`
   - `/Users/mukundhan/Projects/engram/internal/repository/mcp_token.go` -> `10.0`
   - `/Users/mukundhan/Projects/engram/internal/repository/mcp_token_test.go` -> `9.68`
   - `/Users/mukundhan/Projects/engram/internal/repository/project.go` -> `10.0`
   - `/Users/mukundhan/Projects/engram/internal/repository/project_test.go` -> `10.0`
   - `/Users/mukundhan/Projects/engram/internal/repository/user.go` -> `9.38`
   - `/Users/mukundhan/Projects/engram/internal/repository/user_test.go` -> `10.0`
7. CodeScene pre-commit safeguard:
   - `pre_commit_code_health_safeguard(git_repository_path=/Users/mukundhan/Projects/engram)`
   - result: `quality_gates=passed`
   - findings: none

### 2026-02-22 (Go migration CP16: OAuth repository baseline parity)

1. Added OAuth model parity in Go:
   - `internal/models/oauth.go`
   - `OAuthClientRecord`
   - `OAuthAuthorizationCodeRecord`
2. Added OAuth repository baseline:
   - `internal/repository/oauth.go`
   - `CreateOAuthClient`
   - `GetOAuthClient`
   - `CreateOAuthAuthorizationCode`
   - `GetOAuthAuthorizationCodeByHash`
   - `ConsumeOAuthAuthorizationCode`
3. Added migrated repository tests:
   - `internal/repository/oauth_test.go`
4. Executed migrated tests one-by-one:
   - `go test ./internal/repository -run '^TestCreateOAuthClientReturnsCreatedRecord$' -v`
   - `go test ./internal/repository -run '^TestGetOAuthClientReturnsNilWhenMissing$' -v`
   - `go test ./internal/repository -run '^TestCreateOAuthAuthorizationCodeReturnsRecord$' -v`
   - `go test ./internal/repository -run '^TestGetOAuthAuthorizationCodeByHashReturnsRecord$' -v`
   - `go test ./internal/repository -run '^TestConsumeOAuthAuthorizationCodeReturnsNilWhenAlreadyConsumed$' -v`
5. Full Go verification:
   - `go test ./...` passed.
6. File-level CodeScene checks (all Go migration files before commit):
   - `/Users/mukundhan/Projects/engram/cmd/api/main.go` -> `10.0`
   - `/Users/mukundhan/Projects/engram/internal/api/router.go` -> `10.0`
   - `/Users/mukundhan/Projects/engram/internal/api/router_test.go` -> `10.0`
   - `/Users/mukundhan/Projects/engram/internal/auth/password.go` -> `10.0`
   - `/Users/mukundhan/Projects/engram/internal/auth/password_test.go` -> `9.68`
   - `/Users/mukundhan/Projects/engram/internal/config/config.go` -> `9.68`
   - `/Users/mukundhan/Projects/engram/internal/config/config_test.go` -> `10.0`
   - `/Users/mukundhan/Projects/engram/internal/db/db.go` -> `10.0`
   - `/Users/mukundhan/Projects/engram/internal/db/db_test.go` -> `9.61`
   - `/Users/mukundhan/Projects/engram/internal/embeddings/errors.go` -> `10.0`
   - `/Users/mukundhan/Projects/engram/internal/embeddings/local.go` -> `10.0`
   - `/Users/mukundhan/Projects/engram/internal/embeddings/local_test.go` -> `9.68`
   - `/Users/mukundhan/Projects/engram/internal/embeddings/service.go` -> `9.09`
   - `/Users/mukundhan/Projects/engram/internal/embeddings/service_test.go` -> `10.0`
   - `/Users/mukundhan/Projects/engram/internal/models/chat.go` -> `10.0`
   - `/Users/mukundhan/Projects/engram/internal/models/document.go` -> `10.0`
   - `/Users/mukundhan/Projects/engram/internal/models/engram.go` -> `None` (struct-only file; score unavailable)
   - `/Users/mukundhan/Projects/engram/internal/models/mcp_token.go` -> `10.0`
   - `/Users/mukundhan/Projects/engram/internal/models/oauth.go` -> `None` (struct-only file; score unavailable)
   - `/Users/mukundhan/Projects/engram/internal/models/project.go` -> `None` (struct-only file; score unavailable)
   - `/Users/mukundhan/Projects/engram/internal/models/user.go` -> `10.0`
   - `/Users/mukundhan/Projects/engram/internal/repository/chat.go` -> `9.02`
   - `/Users/mukundhan/Projects/engram/internal/repository/chat_message.go` -> `10.0`
   - `/Users/mukundhan/Projects/engram/internal/repository/chat_message_test.go` -> `10.0`
   - `/Users/mukundhan/Projects/engram/internal/repository/chat_pinning.go` -> `8.81`
   - `/Users/mukundhan/Projects/engram/internal/repository/chat_pinning_test.go` -> `9.38`
   - `/Users/mukundhan/Projects/engram/internal/repository/chat_test.go` -> `9.09`
   - `/Users/mukundhan/Projects/engram/internal/repository/document.go` -> `8.81`
   - `/Users/mukundhan/Projects/engram/internal/repository/document_test.go` -> `8.72`
   - `/Users/mukundhan/Projects/engram/internal/repository/engram.go` -> `9.68`
   - `/Users/mukundhan/Projects/engram/internal/repository/engram_helpers_test.go` -> `10.0`
   - `/Users/mukundhan/Projects/engram/internal/repository/engram_rehydration.go` -> `9.68`
   - `/Users/mukundhan/Projects/engram/internal/repository/engram_rehydration_test.go` -> `10.0`
   - `/Users/mukundhan/Projects/engram/internal/repository/engram_repository_test.go` -> `10.0`
   - `/Users/mukundhan/Projects/engram/internal/repository/engram_store.go` -> `10.0`
   - `/Users/mukundhan/Projects/engram/internal/repository/engram_unit_test.go` -> `10.0`
   - `/Users/mukundhan/Projects/engram/internal/repository/engram_write.go` -> `10.0`
   - `/Users/mukundhan/Projects/engram/internal/repository/engram_write_test.go` -> `9.26`
   - `/Users/mukundhan/Projects/engram/internal/repository/mcp_token.go` -> `10.0`
   - `/Users/mukundhan/Projects/engram/internal/repository/mcp_token_test.go` -> `9.68`
   - `/Users/mukundhan/Projects/engram/internal/repository/oauth.go` -> `9.38`
   - `/Users/mukundhan/Projects/engram/internal/repository/oauth_test.go` -> `9.38`
   - `/Users/mukundhan/Projects/engram/internal/repository/project.go` -> `10.0`
   - `/Users/mukundhan/Projects/engram/internal/repository/project_test.go` -> `10.0`
   - `/Users/mukundhan/Projects/engram/internal/repository/user.go` -> `9.38`
   - `/Users/mukundhan/Projects/engram/internal/repository/user_test.go` -> `10.0`
7. CodeScene pre-commit safeguard:
   - `pre_commit_code_health_safeguard(git_repository_path=/Users/mukundhan/Projects/engram)`
   - result: `quality_gates=passed`
   - findings: none

### 2026-02-22 (Go migration CP17: collection repository baseline parity)

1. Added collection model parity in Go:
   - `internal/models/collection.go`
   - `EngramCollectionRecord`
2. Added collection repository baseline:
   - `internal/repository/collection.go`
   - `ListCollections`
   - `GetCollection`
   - `CreateCollection`
   - `UpdateCollection`
   - `SoftDeleteCollection`
   - `AddCollectionItems`
   - `RemoveCollectionItem`
3. Added duplicate-name parity mapping:
   - `ErrCollectionNameExists` for unique collection-name conflicts.
4. Added migrated repository tests:
   - `internal/repository/collection_test.go`
5. Executed migrated tests one-by-one:
   - `go test ./internal/repository -run '^TestListCollectionsBuildsFilters$' -v`
   - `go test ./internal/repository -run '^TestGetCollectionReturnsNilWhenMissing$' -v`
   - `go test ./internal/repository -run '^TestCreateCollectionUsesGeneratedID$' -v`
   - `go test ./internal/repository -run '^TestCreateCollectionMapsDuplicateNameError$' -v`
   - `go test ./internal/repository -run '^TestUpdateCollectionReturnsNilWhenMissing$' -v`
   - `go test ./internal/repository -run '^TestSoftDeleteCollectionReturnsBool$' -v`
   - `go test ./internal/repository -run '^TestAddCollectionItemsReturnsCountAndSkipsEmpty$' -v`
   - `go test ./internal/repository -run '^TestRemoveCollectionItemReturnsBool$' -v`
6. Full Go verification:
   - `go test ./...` passed.
7. File-level CodeScene checks (all Go migration files before commit):
   - `/Users/mukundhan/Projects/engram/cmd/api/main.go` -> `10.0`
   - `/Users/mukundhan/Projects/engram/internal/api/router.go` -> `10.0`
   - `/Users/mukundhan/Projects/engram/internal/api/router_test.go` -> `10.0`
   - `/Users/mukundhan/Projects/engram/internal/auth/password.go` -> `10.0`
   - `/Users/mukundhan/Projects/engram/internal/auth/password_test.go` -> `9.68`
   - `/Users/mukundhan/Projects/engram/internal/config/config.go` -> `9.68`
   - `/Users/mukundhan/Projects/engram/internal/config/config_test.go` -> `10.0`
   - `/Users/mukundhan/Projects/engram/internal/db/db.go` -> `10.0`
   - `/Users/mukundhan/Projects/engram/internal/db/db_test.go` -> `9.61`
   - `/Users/mukundhan/Projects/engram/internal/embeddings/errors.go` -> `10.0`
   - `/Users/mukundhan/Projects/engram/internal/embeddings/local.go` -> `10.0`
   - `/Users/mukundhan/Projects/engram/internal/embeddings/local_test.go` -> `9.68`
   - `/Users/mukundhan/Projects/engram/internal/embeddings/service.go` -> `9.09`
   - `/Users/mukundhan/Projects/engram/internal/embeddings/service_test.go` -> `10.0`
   - `/Users/mukundhan/Projects/engram/internal/models/chat.go` -> `10.0`
   - `/Users/mukundhan/Projects/engram/internal/models/collection.go` -> `None` (struct-only file; score unavailable)
   - `/Users/mukundhan/Projects/engram/internal/models/document.go` -> `10.0`
   - `/Users/mukundhan/Projects/engram/internal/models/engram.go` -> `None` (struct-only file; score unavailable)
   - `/Users/mukundhan/Projects/engram/internal/models/mcp_token.go` -> `10.0`
   - `/Users/mukundhan/Projects/engram/internal/models/oauth.go` -> `None` (struct-only file; score unavailable)
   - `/Users/mukundhan/Projects/engram/internal/models/project.go` -> `None` (struct-only file; score unavailable)
   - `/Users/mukundhan/Projects/engram/internal/models/user.go` -> `10.0`
   - `/Users/mukundhan/Projects/engram/internal/repository/chat.go` -> `9.02`
   - `/Users/mukundhan/Projects/engram/internal/repository/chat_message.go` -> `10.0`
   - `/Users/mukundhan/Projects/engram/internal/repository/chat_message_test.go` -> `10.0`
   - `/Users/mukundhan/Projects/engram/internal/repository/chat_pinning.go` -> `8.81`
   - `/Users/mukundhan/Projects/engram/internal/repository/chat_pinning_test.go` -> `9.38`
   - `/Users/mukundhan/Projects/engram/internal/repository/chat_test.go` -> `9.09`
   - `/Users/mukundhan/Projects/engram/internal/repository/collection.go` -> `9.68`
   - `/Users/mukundhan/Projects/engram/internal/repository/collection_test.go` -> `9.09`
   - `/Users/mukundhan/Projects/engram/internal/repository/document.go` -> `8.81`
   - `/Users/mukundhan/Projects/engram/internal/repository/document_test.go` -> `8.72`
   - `/Users/mukundhan/Projects/engram/internal/repository/engram.go` -> `9.68`
   - `/Users/mukundhan/Projects/engram/internal/repository/engram_helpers_test.go` -> `10.0`
   - `/Users/mukundhan/Projects/engram/internal/repository/engram_rehydration.go` -> `9.68`
   - `/Users/mukundhan/Projects/engram/internal/repository/engram_rehydration_test.go` -> `10.0`
   - `/Users/mukundhan/Projects/engram/internal/repository/engram_repository_test.go` -> `10.0`
   - `/Users/mukundhan/Projects/engram/internal/repository/engram_store.go` -> `10.0`
   - `/Users/mukundhan/Projects/engram/internal/repository/engram_unit_test.go` -> `10.0`
   - `/Users/mukundhan/Projects/engram/internal/repository/engram_write.go` -> `10.0`
   - `/Users/mukundhan/Projects/engram/internal/repository/engram_write_test.go` -> `9.26`
   - `/Users/mukundhan/Projects/engram/internal/repository/mcp_token.go` -> `10.0`
   - `/Users/mukundhan/Projects/engram/internal/repository/mcp_token_test.go` -> `9.68`
   - `/Users/mukundhan/Projects/engram/internal/repository/oauth.go` -> `9.38`
   - `/Users/mukundhan/Projects/engram/internal/repository/oauth_test.go` -> `9.38`
   - `/Users/mukundhan/Projects/engram/internal/repository/project.go` -> `10.0`
   - `/Users/mukundhan/Projects/engram/internal/repository/project_test.go` -> `10.0`
   - `/Users/mukundhan/Projects/engram/internal/repository/user.go` -> `9.38`
   - `/Users/mukundhan/Projects/engram/internal/repository/user_test.go` -> `10.0`
8. CodeScene pre-commit safeguard:
   - `pre_commit_code_health_safeguard(git_repository_path=/Users/mukundhan/Projects/engram)`
   - result: `quality_gates=passed`
   - findings: none

### 2026-02-22 (Go migration CP18: memory-admin session repository baseline parity)

1. Added admin session model parity in Go:
   - `internal/models/chat.go`
   - `AdminChatSessionRecord`
2. Added memory-admin session repository baseline:
   - `internal/repository/admin_session.go`
   - `ListAdminSessions`
   - `GetAdminSession`
   - `SoftDeleteSession`
   - `RestoreSession`
   - `SoftDeleteLinkedEngrams`
3. Added migrated repository tests:
   - `internal/repository/admin_session_test.go`
4. Executed migrated tests one-by-one:
   - `/usr/local/go/bin/go test ./internal/repository -run '^TestListAdminSessionsBuildsFiltersAndParsesEnums$' -v`
   - `/usr/local/go/bin/go test ./internal/repository -run '^TestGetAdminSessionReturnsNilWhenMissing$' -v`
   - `/usr/local/go/bin/go test ./internal/repository -run '^TestSoftDeleteSessionReturnsRecord$' -v`
   - `/usr/local/go/bin/go test ./internal/repository -run '^TestRestoreSessionReturnsBool$' -v`
   - `/usr/local/go/bin/go test ./internal/repository -run '^TestSoftDeleteLinkedEngramsReturnsCount$' -v`
5. Full Go verification:
   - `/usr/local/go/bin/go test ./...` passed.
6. File-level CodeScene checks (all Go migration files before commit):
   - `/Users/mukundhan/Projects/engram/cmd/api/main.go` -> `10.0`
   - `/Users/mukundhan/Projects/engram/internal/api/router.go` -> `10.0`
   - `/Users/mukundhan/Projects/engram/internal/api/router_test.go` -> `10.0`
   - `/Users/mukundhan/Projects/engram/internal/auth/password.go` -> `10.0`
   - `/Users/mukundhan/Projects/engram/internal/auth/password_test.go` -> `9.68`
   - `/Users/mukundhan/Projects/engram/internal/config/config.go` -> `9.68`
   - `/Users/mukundhan/Projects/engram/internal/config/config_test.go` -> `10.0`
   - `/Users/mukundhan/Projects/engram/internal/db/db.go` -> `10.0`
   - `/Users/mukundhan/Projects/engram/internal/db/db_test.go` -> `9.61`
   - `/Users/mukundhan/Projects/engram/internal/embeddings/errors.go` -> `10.0`
   - `/Users/mukundhan/Projects/engram/internal/embeddings/local.go` -> `10.0`
   - `/Users/mukundhan/Projects/engram/internal/embeddings/local_test.go` -> `9.68`
   - `/Users/mukundhan/Projects/engram/internal/embeddings/service.go` -> `9.09`
   - `/Users/mukundhan/Projects/engram/internal/embeddings/service_test.go` -> `10.0`
   - `/Users/mukundhan/Projects/engram/internal/models/chat.go` -> `10.0`
   - `/Users/mukundhan/Projects/engram/internal/models/collection.go` -> `None` (struct-only file; score unavailable)
   - `/Users/mukundhan/Projects/engram/internal/models/document.go` -> `10.0`
   - `/Users/mukundhan/Projects/engram/internal/models/engram.go` -> `None` (struct-only file; score unavailable)
   - `/Users/mukundhan/Projects/engram/internal/models/mcp_token.go` -> `10.0`
   - `/Users/mukundhan/Projects/engram/internal/models/oauth.go` -> `None` (struct-only file; score unavailable)
   - `/Users/mukundhan/Projects/engram/internal/models/project.go` -> `None` (struct-only file; score unavailable)
   - `/Users/mukundhan/Projects/engram/internal/models/user.go` -> `10.0`
   - `/Users/mukundhan/Projects/engram/internal/repository/admin_session.go` -> `9.68`
   - `/Users/mukundhan/Projects/engram/internal/repository/admin_session_test.go` -> `9.68`
   - `/Users/mukundhan/Projects/engram/internal/repository/chat.go` -> `9.02`
   - `/Users/mukundhan/Projects/engram/internal/repository/chat_message.go` -> `10.0`
   - `/Users/mukundhan/Projects/engram/internal/repository/chat_message_test.go` -> `10.0`
   - `/Users/mukundhan/Projects/engram/internal/repository/chat_pinning.go` -> `8.81`
   - `/Users/mukundhan/Projects/engram/internal/repository/chat_pinning_test.go` -> `9.38`
   - `/Users/mukundhan/Projects/engram/internal/repository/chat_test.go` -> `9.09`
   - `/Users/mukundhan/Projects/engram/internal/repository/collection.go` -> `9.68`
   - `/Users/mukundhan/Projects/engram/internal/repository/collection_test.go` -> `9.09`
   - `/Users/mukundhan/Projects/engram/internal/repository/document.go` -> `8.81`
   - `/Users/mukundhan/Projects/engram/internal/repository/document_test.go` -> `8.72`
   - `/Users/mukundhan/Projects/engram/internal/repository/engram.go` -> `9.68`
   - `/Users/mukundhan/Projects/engram/internal/repository/engram_helpers_test.go` -> `10.0`
   - `/Users/mukundhan/Projects/engram/internal/repository/engram_rehydration.go` -> `9.68`
   - `/Users/mukundhan/Projects/engram/internal/repository/engram_rehydration_test.go` -> `10.0`
   - `/Users/mukundhan/Projects/engram/internal/repository/engram_repository_test.go` -> `10.0`
   - `/Users/mukundhan/Projects/engram/internal/repository/engram_store.go` -> `10.0`
   - `/Users/mukundhan/Projects/engram/internal/repository/engram_unit_test.go` -> `10.0`
   - `/Users/mukundhan/Projects/engram/internal/repository/engram_write.go` -> `10.0`
   - `/Users/mukundhan/Projects/engram/internal/repository/engram_write_test.go` -> `9.26`
   - `/Users/mukundhan/Projects/engram/internal/repository/mcp_token.go` -> `10.0`
   - `/Users/mukundhan/Projects/engram/internal/repository/mcp_token_test.go` -> `9.68`
   - `/Users/mukundhan/Projects/engram/internal/repository/oauth.go` -> `9.38`
   - `/Users/mukundhan/Projects/engram/internal/repository/oauth_test.go` -> `9.38`
   - `/Users/mukundhan/Projects/engram/internal/repository/project.go` -> `10.0`
   - `/Users/mukundhan/Projects/engram/internal/repository/project_test.go` -> `10.0`
   - `/Users/mukundhan/Projects/engram/internal/repository/user.go` -> `9.38`
   - `/Users/mukundhan/Projects/engram/internal/repository/user_test.go` -> `10.0`
7. CodeScene pre-commit safeguard:
   - `pre_commit_code_health_safeguard(git_repository_path=/Users/mukundhan/Projects/engram)`
   - result: `quality_gates=passed`
   - findings: none

### 2026-02-22 (Go migration CP19: memory-admin engram repository baseline parity)

1. Added admin engram model parity in Go:
   - `internal/models/engram.go`
   - `AdminEngramSourceInput`
   - `AdminEngramSourceRecord`
   - `AdminEngramRecord`
2. Added memory-admin engram repository baseline:
   - `internal/repository/admin_engram.go`
   - `ListAdminEngrams`
   - `ListAdminEngramSources`
   - `GetAdminEngram`
   - `MoveAdminEngramProject`
   - `SoftDeleteEngram`
   - `RestoreEngram`
3. Added migrated repository tests:
   - `internal/repository/admin_engram_test.go`
4. Executed migrated tests one-by-one:
   - `/usr/local/go/bin/go test ./internal/repository -run '^TestListAdminEngramsBuildsFiltersAndSearch$' -v`
   - `/usr/local/go/bin/go test ./internal/repository -run '^TestListAdminEngramSourcesReturnsRows$' -v`
   - `/usr/local/go/bin/go test ./internal/repository -run '^TestGetAdminEngramReturnsNilWhenMissing$' -v`
   - `/usr/local/go/bin/go test ./internal/repository -run '^TestGetAdminEngramHydratesSources$' -v`
   - `/usr/local/go/bin/go test ./internal/repository -run '^TestMoveAdminEngramProjectReturnsUpdatedRecordAndRunsDetachQuery$' -v`
   - `/usr/local/go/bin/go test ./internal/repository -run '^TestMoveAdminEngramProjectReturnsNilWhenNotFound$' -v`
   - `/usr/local/go/bin/go test ./internal/repository -run '^TestSoftDeleteEngramReturnsBool$' -v`
   - `/usr/local/go/bin/go test ./internal/repository -run '^TestRestoreEngramReturnsBool$' -v`
5. Full Go verification:
   - `/usr/local/go/bin/go test ./...` passed.
6. File-level CodeScene checks (all Go migration files before commit):
   - `/Users/mukundhan/Projects/engram/cmd/api/main.go` -> `10.0`
   - `/Users/mukundhan/Projects/engram/internal/api/router.go` -> `10.0`
   - `/Users/mukundhan/Projects/engram/internal/api/router_test.go` -> `10.0`
   - `/Users/mukundhan/Projects/engram/internal/auth/password.go` -> `10.0`
   - `/Users/mukundhan/Projects/engram/internal/auth/password_test.go` -> `9.68`
   - `/Users/mukundhan/Projects/engram/internal/config/config.go` -> `9.68`
   - `/Users/mukundhan/Projects/engram/internal/config/config_test.go` -> `10.0`
   - `/Users/mukundhan/Projects/engram/internal/db/db.go` -> `10.0`
   - `/Users/mukundhan/Projects/engram/internal/db/db_test.go` -> `9.61`
   - `/Users/mukundhan/Projects/engram/internal/embeddings/errors.go` -> `10.0`
   - `/Users/mukundhan/Projects/engram/internal/embeddings/local.go` -> `10.0`
   - `/Users/mukundhan/Projects/engram/internal/embeddings/local_test.go` -> `9.68`
   - `/Users/mukundhan/Projects/engram/internal/embeddings/service.go` -> `9.09`
   - `/Users/mukundhan/Projects/engram/internal/embeddings/service_test.go` -> `10.0`
   - `/Users/mukundhan/Projects/engram/internal/models/chat.go` -> `10.0`
   - `/Users/mukundhan/Projects/engram/internal/models/collection.go` -> `None` (struct-only file; score unavailable)
   - `/Users/mukundhan/Projects/engram/internal/models/document.go` -> `10.0`
   - `/Users/mukundhan/Projects/engram/internal/models/engram.go` -> `None` (struct-only file; score unavailable)
   - `/Users/mukundhan/Projects/engram/internal/models/mcp_token.go` -> `10.0`
   - `/Users/mukundhan/Projects/engram/internal/models/oauth.go` -> `None` (struct-only file; score unavailable)
   - `/Users/mukundhan/Projects/engram/internal/models/project.go` -> `None` (struct-only file; score unavailable)
   - `/Users/mukundhan/Projects/engram/internal/models/user.go` -> `10.0`
   - `/Users/mukundhan/Projects/engram/internal/repository/admin_engram.go` -> `9.68`
   - `/Users/mukundhan/Projects/engram/internal/repository/admin_engram_test.go` -> `9.38`
   - `/Users/mukundhan/Projects/engram/internal/repository/admin_session.go` -> `9.68`
   - `/Users/mukundhan/Projects/engram/internal/repository/admin_session_test.go` -> `9.68`
   - `/Users/mukundhan/Projects/engram/internal/repository/chat.go` -> `9.02`
   - `/Users/mukundhan/Projects/engram/internal/repository/chat_message.go` -> `10.0`
   - `/Users/mukundhan/Projects/engram/internal/repository/chat_message_test.go` -> `10.0`
   - `/Users/mukundhan/Projects/engram/internal/repository/chat_pinning.go` -> `8.81`
   - `/Users/mukundhan/Projects/engram/internal/repository/chat_pinning_test.go` -> `9.38`
   - `/Users/mukundhan/Projects/engram/internal/repository/chat_test.go` -> `9.09`
   - `/Users/mukundhan/Projects/engram/internal/repository/collection.go` -> `9.68`
   - `/Users/mukundhan/Projects/engram/internal/repository/collection_test.go` -> `9.09`
   - `/Users/mukundhan/Projects/engram/internal/repository/document.go` -> `8.81`
   - `/Users/mukundhan/Projects/engram/internal/repository/document_test.go` -> `8.72`
   - `/Users/mukundhan/Projects/engram/internal/repository/engram.go` -> `9.68`
   - `/Users/mukundhan/Projects/engram/internal/repository/engram_helpers_test.go` -> `10.0`
   - `/Users/mukundhan/Projects/engram/internal/repository/engram_rehydration.go` -> `9.68`
   - `/Users/mukundhan/Projects/engram/internal/repository/engram_rehydration_test.go` -> `10.0`
   - `/Users/mukundhan/Projects/engram/internal/repository/engram_repository_test.go` -> `10.0`
   - `/Users/mukundhan/Projects/engram/internal/repository/engram_store.go` -> `10.0`
   - `/Users/mukundhan/Projects/engram/internal/repository/engram_unit_test.go` -> `10.0`
   - `/Users/mukundhan/Projects/engram/internal/repository/engram_write.go` -> `10.0`
   - `/Users/mukundhan/Projects/engram/internal/repository/engram_write_test.go` -> `9.26`
   - `/Users/mukundhan/Projects/engram/internal/repository/mcp_token.go` -> `10.0`
   - `/Users/mukundhan/Projects/engram/internal/repository/mcp_token_test.go` -> `9.68`
   - `/Users/mukundhan/Projects/engram/internal/repository/oauth.go` -> `9.38`
   - `/Users/mukundhan/Projects/engram/internal/repository/oauth_test.go` -> `9.38`
   - `/Users/mukundhan/Projects/engram/internal/repository/project.go` -> `10.0`
   - `/Users/mukundhan/Projects/engram/internal/repository/project_test.go` -> `10.0`
   - `/Users/mukundhan/Projects/engram/internal/repository/user.go` -> `9.38`
   - `/Users/mukundhan/Projects/engram/internal/repository/user_test.go` -> `10.0`
7. CodeScene pre-commit safeguard:
   - `pre_commit_code_health_safeguard(git_repository_path=/Users/mukundhan/Projects/engram)`
   - result: `quality_gates=passed`
   - findings: none

### 2026-02-22 (Go migration CP20: memory-admin engram update/source-replacement parity)

1. Added memory-admin engram update repository parity in Go:
   - `internal/repository/admin_engram_update.go`
   - `UpdateAdminEngram`
   - `replaceAdminEngramSources`
   - helper parity for update-field merge, retrieval text composition, engram JSON refresh, and embedding/vector persistence.
2. Added migrated update-focused repository tests:
   - `internal/repository/admin_engram_update_test.go`
3. Executed migrated tests one-by-one:
   - `/usr/local/go/bin/go test ./internal/repository -run '^TestBuildAdminEngramUpdateFieldsUsesPayloadValuesAndDefaults$' -v`
   - `/usr/local/go/bin/go test ./internal/repository -run '^TestBuildAdminEngramRetrievalTextUsesUpdateFields$' -v`
   - `/usr/local/go/bin/go test ./internal/repository -run '^TestBuildAdminEngramJSONPayloadOverridesMutableFields$' -v`
   - `/usr/local/go/bin/go test ./internal/repository -run '^TestReplaceAdminEngramSourcesReplacesRows$' -v`
   - `/usr/local/go/bin/go test ./internal/repository -run '^TestUpdateAdminEngramReturnsNilWhenMissing$' -v`
   - `/usr/local/go/bin/go test ./internal/repository -run '^TestUpdateAdminEngramPersistsFieldsAndOptionallySources$' -v`
4. Full Go verification:
   - `/usr/local/go/bin/go test ./...` passed.
5. File-level CodeScene checks (all Go migration files before commit):
   - `/Users/mukundhan/Projects/engram/cmd/api/main.go` -> `10.0`
   - `/Users/mukundhan/Projects/engram/internal/api/router.go` -> `10.0`
   - `/Users/mukundhan/Projects/engram/internal/api/router_test.go` -> `10.0`
   - `/Users/mukundhan/Projects/engram/internal/auth/password.go` -> `10.0`
   - `/Users/mukundhan/Projects/engram/internal/auth/password_test.go` -> `9.68`
   - `/Users/mukundhan/Projects/engram/internal/config/config.go` -> `9.68`
   - `/Users/mukundhan/Projects/engram/internal/config/config_test.go` -> `10.0`
   - `/Users/mukundhan/Projects/engram/internal/db/db.go` -> `10.0`
   - `/Users/mukundhan/Projects/engram/internal/db/db_test.go` -> `9.61`
   - `/Users/mukundhan/Projects/engram/internal/embeddings/errors.go` -> `10.0`
   - `/Users/mukundhan/Projects/engram/internal/embeddings/local.go` -> `10.0`
   - `/Users/mukundhan/Projects/engram/internal/embeddings/local_test.go` -> `9.68`
   - `/Users/mukundhan/Projects/engram/internal/embeddings/service.go` -> `9.09`
   - `/Users/mukundhan/Projects/engram/internal/embeddings/service_test.go` -> `10.0`
   - `/Users/mukundhan/Projects/engram/internal/models/chat.go` -> `10.0`
   - `/Users/mukundhan/Projects/engram/internal/models/collection.go` -> `None` (struct-only file; score unavailable)
   - `/Users/mukundhan/Projects/engram/internal/models/document.go` -> `10.0`
   - `/Users/mukundhan/Projects/engram/internal/models/engram.go` -> `None` (struct-only file; score unavailable)
   - `/Users/mukundhan/Projects/engram/internal/models/mcp_token.go` -> `10.0`
   - `/Users/mukundhan/Projects/engram/internal/models/oauth.go` -> `None` (struct-only file; score unavailable)
   - `/Users/mukundhan/Projects/engram/internal/models/project.go` -> `None` (struct-only file; score unavailable)
   - `/Users/mukundhan/Projects/engram/internal/models/user.go` -> `10.0`
   - `/Users/mukundhan/Projects/engram/internal/repository/admin_engram.go` -> `9.68`
   - `/Users/mukundhan/Projects/engram/internal/repository/admin_engram_test.go` -> `9.38`
   - `/Users/mukundhan/Projects/engram/internal/repository/admin_engram_update.go` -> `9.61`
   - `/Users/mukundhan/Projects/engram/internal/repository/admin_engram_update_test.go` -> `9.25`
   - `/Users/mukundhan/Projects/engram/internal/repository/admin_session.go` -> `9.68`
   - `/Users/mukundhan/Projects/engram/internal/repository/admin_session_test.go` -> `9.68`
   - `/Users/mukundhan/Projects/engram/internal/repository/chat.go` -> `9.02`
   - `/Users/mukundhan/Projects/engram/internal/repository/chat_message.go` -> `10.0`
   - `/Users/mukundhan/Projects/engram/internal/repository/chat_message_test.go` -> `10.0`
   - `/Users/mukundhan/Projects/engram/internal/repository/chat_pinning.go` -> `8.81`
   - `/Users/mukundhan/Projects/engram/internal/repository/chat_pinning_test.go` -> `9.38`
   - `/Users/mukundhan/Projects/engram/internal/repository/chat_test.go` -> `9.09`
   - `/Users/mukundhan/Projects/engram/internal/repository/collection.go` -> `9.68`
   - `/Users/mukundhan/Projects/engram/internal/repository/collection_test.go` -> `9.09`
   - `/Users/mukundhan/Projects/engram/internal/repository/document.go` -> `8.81`
   - `/Users/mukundhan/Projects/engram/internal/repository/document_test.go` -> `8.72`
   - `/Users/mukundhan/Projects/engram/internal/repository/engram.go` -> `9.68`
   - `/Users/mukundhan/Projects/engram/internal/repository/engram_helpers_test.go` -> `10.0`
   - `/Users/mukundhan/Projects/engram/internal/repository/engram_rehydration.go` -> `9.68`
   - `/Users/mukundhan/Projects/engram/internal/repository/engram_rehydration_test.go` -> `10.0`
   - `/Users/mukundhan/Projects/engram/internal/repository/engram_repository_test.go` -> `10.0`
   - `/Users/mukundhan/Projects/engram/internal/repository/engram_store.go` -> `10.0`
   - `/Users/mukundhan/Projects/engram/internal/repository/engram_unit_test.go` -> `10.0`
   - `/Users/mukundhan/Projects/engram/internal/repository/engram_write.go` -> `10.0`
   - `/Users/mukundhan/Projects/engram/internal/repository/engram_write_test.go` -> `9.26`
   - `/Users/mukundhan/Projects/engram/internal/repository/mcp_token.go` -> `10.0`
   - `/Users/mukundhan/Projects/engram/internal/repository/mcp_token_test.go` -> `9.68`
   - `/Users/mukundhan/Projects/engram/internal/repository/oauth.go` -> `9.38`
   - `/Users/mukundhan/Projects/engram/internal/repository/oauth_test.go` -> `9.38`
   - `/Users/mukundhan/Projects/engram/internal/repository/project.go` -> `10.0`
   - `/Users/mukundhan/Projects/engram/internal/repository/project_test.go` -> `10.0`
   - `/Users/mukundhan/Projects/engram/internal/repository/user.go` -> `9.38`
   - `/Users/mukundhan/Projects/engram/internal/repository/user_test.go` -> `10.0`
6. CodeScene pre-commit safeguard:
   - `pre_commit_code_health_safeguard(git_repository_path=/Users/mukundhan/Projects/engram)`
   - result: `quality_gates=passed`
   - findings: none

### 2026-02-22 (Go migration CP21: memory-admin service baseline parity)

1. Added memory-admin service package in Go:
   - `internal/admin/service.go`
   - service orchestration for session, engram, and collection operations
   - stale-update conflict checks for engram and collection update paths
   - project-resolution contract integration for write operations
   - typed service-level request/response and error contracts
2. Ported memory-admin service unit tests from `api/tests/test_memory_admin_service.py`:
   - `internal/admin/service_test.go`
3. Executed migrated tests one-by-one:
   - `/usr/local/go/bin/go test ./internal/admin -run '^TestListSessionsForwardsSharedRequestObject$' -v`
   - `/usr/local/go/bin/go test ./internal/admin -run '^TestListCollectionsForwardsSharedRequestObject$' -v`
   - `/usr/local/go/bin/go test ./internal/admin -run '^TestListEngramsUsesRequestObject$' -v`
   - `/usr/local/go/bin/go test ./internal/admin -run '^TestUpdateCollectionRejectsStaleExpectedUpdatedAt$' -v`
   - `/usr/local/go/bin/go test ./internal/admin -run '^TestUpdateEngramRejectsStaleExpectedUpdatedAt$' -v`
   - `/usr/local/go/bin/go test ./internal/admin -run '^TestUpdateEngramUsesRepositoryRequestObject$' -v`
4. Full Go verification:
   - `/usr/local/go/bin/go test ./...` passed.
5. File-level CodeScene checks (all Go migration files before commit):
   - `/Users/mukundhan/Projects/engram/cmd/api/main.go` -> `10.0`
   - `/Users/mukundhan/Projects/engram/internal/admin/service.go` -> `8.54`
   - `/Users/mukundhan/Projects/engram/internal/admin/service_test.go` -> `9.38`
   - `/Users/mukundhan/Projects/engram/internal/api/router.go` -> `10.0`
   - `/Users/mukundhan/Projects/engram/internal/api/router_test.go` -> `10.0`
   - `/Users/mukundhan/Projects/engram/internal/auth/password.go` -> `10.0`
   - `/Users/mukundhan/Projects/engram/internal/auth/password_test.go` -> `9.68`
   - `/Users/mukundhan/Projects/engram/internal/config/config.go` -> `9.68`
   - `/Users/mukundhan/Projects/engram/internal/config/config_test.go` -> `10.0`
   - `/Users/mukundhan/Projects/engram/internal/db/db.go` -> `10.0`
   - `/Users/mukundhan/Projects/engram/internal/db/db_test.go` -> `9.61`
   - `/Users/mukundhan/Projects/engram/internal/embeddings/errors.go` -> `10.0`
   - `/Users/mukundhan/Projects/engram/internal/embeddings/local.go` -> `10.0`
   - `/Users/mukundhan/Projects/engram/internal/embeddings/local_test.go` -> `9.68`
   - `/Users/mukundhan/Projects/engram/internal/embeddings/service.go` -> `9.09`
   - `/Users/mukundhan/Projects/engram/internal/embeddings/service_test.go` -> `10.0`
   - `/Users/mukundhan/Projects/engram/internal/models/chat.go` -> `10.0`
   - `/Users/mukundhan/Projects/engram/internal/models/collection.go` -> `None` (struct-only file; score unavailable)
   - `/Users/mukundhan/Projects/engram/internal/models/document.go` -> `10.0`
   - `/Users/mukundhan/Projects/engram/internal/models/engram.go` -> `None` (struct-only file; score unavailable)
   - `/Users/mukundhan/Projects/engram/internal/models/mcp_token.go` -> `10.0`
   - `/Users/mukundhan/Projects/engram/internal/models/oauth.go` -> `None` (struct-only file; score unavailable)
   - `/Users/mukundhan/Projects/engram/internal/models/project.go` -> `None` (struct-only file; score unavailable)
   - `/Users/mukundhan/Projects/engram/internal/models/user.go` -> `10.0`
   - `/Users/mukundhan/Projects/engram/internal/repository/admin_engram.go` -> `9.68`
   - `/Users/mukundhan/Projects/engram/internal/repository/admin_engram_test.go` -> `9.38`
   - `/Users/mukundhan/Projects/engram/internal/repository/admin_engram_update.go` -> `9.61`
   - `/Users/mukundhan/Projects/engram/internal/repository/admin_engram_update_test.go` -> `9.25`
   - `/Users/mukundhan/Projects/engram/internal/repository/admin_session.go` -> `9.68`
   - `/Users/mukundhan/Projects/engram/internal/repository/admin_session_test.go` -> `9.68`
   - `/Users/mukundhan/Projects/engram/internal/repository/chat.go` -> `9.02`
   - `/Users/mukundhan/Projects/engram/internal/repository/chat_message.go` -> `10.0`
   - `/Users/mukundhan/Projects/engram/internal/repository/chat_message_test.go` -> `10.0`
   - `/Users/mukundhan/Projects/engram/internal/repository/chat_pinning.go` -> `8.81`
   - `/Users/mukundhan/Projects/engram/internal/repository/chat_pinning_test.go` -> `9.38`
   - `/Users/mukundhan/Projects/engram/internal/repository/chat_test.go` -> `9.09`
   - `/Users/mukundhan/Projects/engram/internal/repository/collection.go` -> `9.68`
   - `/Users/mukundhan/Projects/engram/internal/repository/collection_test.go` -> `9.09`
   - `/Users/mukundhan/Projects/engram/internal/repository/document.go` -> `8.81`
   - `/Users/mukundhan/Projects/engram/internal/repository/document_test.go` -> `8.72`
   - `/Users/mukundhan/Projects/engram/internal/repository/engram.go` -> `9.68`
   - `/Users/mukundhan/Projects/engram/internal/repository/engram_helpers_test.go` -> `10.0`
   - `/Users/mukundhan/Projects/engram/internal/repository/engram_rehydration.go` -> `9.68`
   - `/Users/mukundhan/Projects/engram/internal/repository/engram_rehydration_test.go` -> `10.0`
   - `/Users/mukundhan/Projects/engram/internal/repository/engram_repository_test.go` -> `10.0`
   - `/Users/mukundhan/Projects/engram/internal/repository/engram_store.go` -> `10.0`
   - `/Users/mukundhan/Projects/engram/internal/repository/engram_unit_test.go` -> `10.0`
   - `/Users/mukundhan/Projects/engram/internal/repository/engram_write.go` -> `10.0`
   - `/Users/mukundhan/Projects/engram/internal/repository/engram_write_test.go` -> `9.26`
   - `/Users/mukundhan/Projects/engram/internal/repository/mcp_token.go` -> `10.0`
   - `/Users/mukundhan/Projects/engram/internal/repository/mcp_token_test.go` -> `9.68`
   - `/Users/mukundhan/Projects/engram/internal/repository/oauth.go` -> `9.38`
   - `/Users/mukundhan/Projects/engram/internal/repository/oauth_test.go` -> `9.38`
   - `/Users/mukundhan/Projects/engram/internal/repository/project.go` -> `10.0`
   - `/Users/mukundhan/Projects/engram/internal/repository/project_test.go` -> `10.0`
   - `/Users/mukundhan/Projects/engram/internal/repository/user.go` -> `9.38`
   - `/Users/mukundhan/Projects/engram/internal/repository/user_test.go` -> `10.0`
6. CodeScene pre-commit safeguard:
   - `pre_commit_code_health_safeguard(git_repository_path=/Users/mukundhan/Projects/engram)`
   - result: `quality_gates=passed`
   - findings: none

### 2026-02-22 (Go migration CP22: memory-admin API route baseline parity)

1. Added memory-admin API route layer in Go:
   - `internal/api/admin_memory.go`
   - Chi handlers for `/api/v1/admin/memory/sessions*`, `/engrams*`, and `/collections*`.
   - request parsing for query/body payloads plus actor-resolution hook and service-error status mapping.
2. Added migrated route-focused tests:
   - `internal/api/admin_memory_test.go`
3. Executed migrated tests one-by-one:
   - `/usr/local/go/bin/go test ./internal/api -run '^TestMountMemoryAdminRoutesListSessionsForwardsQueryAndActorCheck$' -v`
   - `/usr/local/go/bin/go test ./internal/api -run '^TestMountMemoryAdminRoutesDeleteSessionUsesActorAndPayload$' -v`
   - `/usr/local/go/bin/go test ./internal/api -run '^TestMountMemoryAdminRoutesListEngramsUsesRequestObject$' -v`
   - `/usr/local/go/bin/go test ./internal/api -run '^TestMountMemoryAdminRoutesUpdateEngramMapsStaleTo409$' -v`
   - `/usr/local/go/bin/go test ./internal/api -run '^TestMountMemoryAdminRoutesCreateCollectionReturns201$' -v`
   - `/usr/local/go/bin/go test ./internal/api -run '^TestMountMemoryAdminRoutesReturnsForbiddenWhenActorCheckFails$' -v`
4. Full Go verification:
   - `/usr/local/go/bin/go test ./...` passed.
5. File-level CodeScene checks (all Go migration files before commit):
   - `/Users/mukundhan/Projects/engram/cmd/api/main.go` -> `10.0`
   - `/Users/mukundhan/Projects/engram/internal/admin/service.go` -> `8.54`
   - `/Users/mukundhan/Projects/engram/internal/admin/service_test.go` -> `9.38`
   - `/Users/mukundhan/Projects/engram/internal/api/admin_memory.go` -> `6.88`
   - `/Users/mukundhan/Projects/engram/internal/api/admin_memory_test.go` -> `9.68`
   - `/Users/mukundhan/Projects/engram/internal/api/router.go` -> `10.0`
   - `/Users/mukundhan/Projects/engram/internal/api/router_test.go` -> `10.0`
   - `/Users/mukundhan/Projects/engram/internal/auth/password.go` -> `10.0`
   - `/Users/mukundhan/Projects/engram/internal/auth/password_test.go` -> `9.68`
   - `/Users/mukundhan/Projects/engram/internal/config/config.go` -> `9.68`
   - `/Users/mukundhan/Projects/engram/internal/config/config_test.go` -> `10.0`
   - `/Users/mukundhan/Projects/engram/internal/db/db.go` -> `10.0`
   - `/Users/mukundhan/Projects/engram/internal/db/db_test.go` -> `9.61`
   - `/Users/mukundhan/Projects/engram/internal/embeddings/errors.go` -> `10.0`
   - `/Users/mukundhan/Projects/engram/internal/embeddings/local.go` -> `10.0`
   - `/Users/mukundhan/Projects/engram/internal/embeddings/local_test.go` -> `9.68`
   - `/Users/mukundhan/Projects/engram/internal/embeddings/service.go` -> `9.09`
   - `/Users/mukundhan/Projects/engram/internal/embeddings/service_test.go` -> `10.0`
   - `/Users/mukundhan/Projects/engram/internal/models/chat.go` -> `10.0`
   - `/Users/mukundhan/Projects/engram/internal/models/collection.go` -> `None` (struct-only file; score unavailable)
   - `/Users/mukundhan/Projects/engram/internal/models/document.go` -> `10.0`
   - `/Users/mukundhan/Projects/engram/internal/models/engram.go` -> `None` (struct-only file; score unavailable)
   - `/Users/mukundhan/Projects/engram/internal/models/mcp_token.go` -> `10.0`
   - `/Users/mukundhan/Projects/engram/internal/models/oauth.go` -> `None` (struct-only file; score unavailable)
   - `/Users/mukundhan/Projects/engram/internal/models/project.go` -> `None` (struct-only file; score unavailable)
   - `/Users/mukundhan/Projects/engram/internal/models/user.go` -> `10.0`
   - `/Users/mukundhan/Projects/engram/internal/repository/admin_engram.go` -> `9.68`
   - `/Users/mukundhan/Projects/engram/internal/repository/admin_engram_test.go` -> `9.38`
   - `/Users/mukundhan/Projects/engram/internal/repository/admin_engram_update.go` -> `9.61`
   - `/Users/mukundhan/Projects/engram/internal/repository/admin_engram_update_test.go` -> `9.25`
   - `/Users/mukundhan/Projects/engram/internal/repository/admin_session.go` -> `9.68`
   - `/Users/mukundhan/Projects/engram/internal/repository/admin_session_test.go` -> `9.68`
   - `/Users/mukundhan/Projects/engram/internal/repository/chat.go` -> `9.02`
   - `/Users/mukundhan/Projects/engram/internal/repository/chat_message.go` -> `10.0`
   - `/Users/mukundhan/Projects/engram/internal/repository/chat_message_test.go` -> `10.0`
   - `/Users/mukundhan/Projects/engram/internal/repository/chat_pinning.go` -> `8.81`
   - `/Users/mukundhan/Projects/engram/internal/repository/chat_pinning_test.go` -> `9.38`
   - `/Users/mukundhan/Projects/engram/internal/repository/chat_test.go` -> `9.09`
   - `/Users/mukundhan/Projects/engram/internal/repository/collection.go` -> `9.68`
   - `/Users/mukundhan/Projects/engram/internal/repository/collection_test.go` -> `9.09`
   - `/Users/mukundhan/Projects/engram/internal/repository/document.go` -> `8.81`
   - `/Users/mukundhan/Projects/engram/internal/repository/document_test.go` -> `8.72`
   - `/Users/mukundhan/Projects/engram/internal/repository/engram.go` -> `9.68`
   - `/Users/mukundhan/Projects/engram/internal/repository/engram_helpers_test.go` -> `10.0`
   - `/Users/mukundhan/Projects/engram/internal/repository/engram_rehydration.go` -> `9.68`
   - `/Users/mukundhan/Projects/engram/internal/repository/engram_rehydration_test.go` -> `10.0`
   - `/Users/mukundhan/Projects/engram/internal/repository/engram_repository_test.go` -> `10.0`
   - `/Users/mukundhan/Projects/engram/internal/repository/engram_store.go` -> `10.0`
   - `/Users/mukundhan/Projects/engram/internal/repository/engram_unit_test.go` -> `10.0`
   - `/Users/mukundhan/Projects/engram/internal/repository/engram_write.go` -> `10.0`
   - `/Users/mukundhan/Projects/engram/internal/repository/engram_write_test.go` -> `9.26`
   - `/Users/mukundhan/Projects/engram/internal/repository/mcp_token.go` -> `10.0`
   - `/Users/mukundhan/Projects/engram/internal/repository/mcp_token_test.go` -> `9.68`
   - `/Users/mukundhan/Projects/engram/internal/repository/oauth.go` -> `9.38`
   - `/Users/mukundhan/Projects/engram/internal/repository/oauth_test.go` -> `9.38`
   - `/Users/mukundhan/Projects/engram/internal/repository/project.go` -> `10.0`
   - `/Users/mukundhan/Projects/engram/internal/repository/project_test.go` -> `10.0`
   - `/Users/mukundhan/Projects/engram/internal/repository/user.go` -> `9.38`
   - `/Users/mukundhan/Projects/engram/internal/repository/user_test.go` -> `10.0`
6. CodeScene pre-commit safeguard:
   - `pre_commit_code_health_safeguard(git_repository_path=/Users/mukundhan/Projects/engram)`
   - result: `quality_gates=passed`
   - findings: none

### 2026-02-22 (Go migration CP23: API integration baseline for memory-admin route composition)

1. Added dependency-aware API router composition for memory-admin routes:
   - `internal/api/router.go`
   - introduced `RouterDependencies` and `NewRouterWithDependencies`.
   - kept `NewRouter(settings)` backward-compatible and delegated to dependency-aware constructor.
   - memory-admin routes mount only when both `MemoryAdminService` and `RequireAdminActor` are provided.
2. Added migrated router composition tests:
   - `internal/api/router_test.go`
   - `TestMemoryAdminRoutesNotMountedWithoutDependencies`
   - `TestMemoryAdminRoutesMountedWithDependencies`
3. Executed migrated tests one-by-one:
   - `/usr/local/go/bin/go test ./internal/api -run '^TestMemoryAdminRoutesNotMountedWithoutDependencies$' -v`
   - `/usr/local/go/bin/go test ./internal/api -run '^TestMemoryAdminRoutesMountedWithDependencies$' -v`
4. Full Go verification:
   - `/usr/local/go/bin/go test ./...` passed.
5. File-level CodeScene checks (all Go migration files before commit):
   - `/Users/mukundhan/Projects/engram/cmd/api/main.go` -> `10.0`
   - `/Users/mukundhan/Projects/engram/internal/admin/service.go` -> `8.54`
   - `/Users/mukundhan/Projects/engram/internal/admin/service_test.go` -> `9.38`
   - `/Users/mukundhan/Projects/engram/internal/api/admin_memory.go` -> `6.88`
   - `/Users/mukundhan/Projects/engram/internal/api/admin_memory_test.go` -> `9.68`
   - `/Users/mukundhan/Projects/engram/internal/api/router.go` -> `10.0`
   - `/Users/mukundhan/Projects/engram/internal/api/router_test.go` -> `10.0`
   - `/Users/mukundhan/Projects/engram/internal/auth/password.go` -> `10.0`
   - `/Users/mukundhan/Projects/engram/internal/auth/password_test.go` -> `9.68`
   - `/Users/mukundhan/Projects/engram/internal/config/config.go` -> `9.68`
   - `/Users/mukundhan/Projects/engram/internal/config/config_test.go` -> `10.0`
   - `/Users/mukundhan/Projects/engram/internal/db/db.go` -> `10.0`
   - `/Users/mukundhan/Projects/engram/internal/db/db_test.go` -> `9.61`
   - `/Users/mukundhan/Projects/engram/internal/embeddings/errors.go` -> `10.0`
   - `/Users/mukundhan/Projects/engram/internal/embeddings/local.go` -> `10.0`
   - `/Users/mukundhan/Projects/engram/internal/embeddings/local_test.go` -> `9.68`
   - `/Users/mukundhan/Projects/engram/internal/embeddings/service.go` -> `9.09`
   - `/Users/mukundhan/Projects/engram/internal/embeddings/service_test.go` -> `10.0`
   - `/Users/mukundhan/Projects/engram/internal/models/chat.go` -> `10.0`
   - `/Users/mukundhan/Projects/engram/internal/models/collection.go` -> `None` (struct-only file; score unavailable)
   - `/Users/mukundhan/Projects/engram/internal/models/document.go` -> `10.0`
   - `/Users/mukundhan/Projects/engram/internal/models/engram.go` -> `None` (struct-only file; score unavailable)
   - `/Users/mukundhan/Projects/engram/internal/models/mcp_token.go` -> `10.0`
   - `/Users/mukundhan/Projects/engram/internal/models/oauth.go` -> `None` (struct-only file; score unavailable)
   - `/Users/mukundhan/Projects/engram/internal/models/project.go` -> `None` (struct-only file; score unavailable)
   - `/Users/mukundhan/Projects/engram/internal/models/user.go` -> `10.0`
   - `/Users/mukundhan/Projects/engram/internal/repository/admin_engram.go` -> `9.68`
   - `/Users/mukundhan/Projects/engram/internal/repository/admin_engram_test.go` -> `9.38`
   - `/Users/mukundhan/Projects/engram/internal/repository/admin_engram_update.go` -> `9.61`
   - `/Users/mukundhan/Projects/engram/internal/repository/admin_engram_update_test.go` -> `9.25`
   - `/Users/mukundhan/Projects/engram/internal/repository/admin_session.go` -> `9.68`
   - `/Users/mukundhan/Projects/engram/internal/repository/admin_session_test.go` -> `9.68`
   - `/Users/mukundhan/Projects/engram/internal/repository/chat.go` -> `9.02`
   - `/Users/mukundhan/Projects/engram/internal/repository/chat_message.go` -> `10.0`
   - `/Users/mukundhan/Projects/engram/internal/repository/chat_message_test.go` -> `10.0`
   - `/Users/mukundhan/Projects/engram/internal/repository/chat_pinning.go` -> `8.81`
   - `/Users/mukundhan/Projects/engram/internal/repository/chat_pinning_test.go` -> `9.38`
   - `/Users/mukundhan/Projects/engram/internal/repository/chat_test.go` -> `9.09`
   - `/Users/mukundhan/Projects/engram/internal/repository/collection.go` -> `9.68`
   - `/Users/mukundhan/Projects/engram/internal/repository/collection_test.go` -> `9.09`
   - `/Users/mukundhan/Projects/engram/internal/repository/document.go` -> `8.81`
   - `/Users/mukundhan/Projects/engram/internal/repository/document_test.go` -> `8.72`
   - `/Users/mukundhan/Projects/engram/internal/repository/engram.go` -> `9.68`
   - `/Users/mukundhan/Projects/engram/internal/repository/engram_helpers_test.go` -> `10.0`
   - `/Users/mukundhan/Projects/engram/internal/repository/engram_rehydration.go` -> `9.68`
   - `/Users/mukundhan/Projects/engram/internal/repository/engram_rehydration_test.go` -> `10.0`
   - `/Users/mukundhan/Projects/engram/internal/repository/engram_repository_test.go` -> `10.0`
   - `/Users/mukundhan/Projects/engram/internal/repository/engram_store.go` -> `10.0`
   - `/Users/mukundhan/Projects/engram/internal/repository/engram_unit_test.go` -> `10.0`
   - `/Users/mukundhan/Projects/engram/internal/repository/engram_write.go` -> `10.0`
   - `/Users/mukundhan/Projects/engram/internal/repository/engram_write_test.go` -> `9.26`
   - `/Users/mukundhan/Projects/engram/internal/repository/mcp_token.go` -> `10.0`
   - `/Users/mukundhan/Projects/engram/internal/repository/mcp_token_test.go` -> `9.68`
   - `/Users/mukundhan/Projects/engram/internal/repository/oauth.go` -> `9.38`
   - `/Users/mukundhan/Projects/engram/internal/repository/oauth_test.go` -> `9.38`
   - `/Users/mukundhan/Projects/engram/internal/repository/project.go` -> `10.0`
   - `/Users/mukundhan/Projects/engram/internal/repository/project_test.go` -> `10.0`
   - `/Users/mukundhan/Projects/engram/internal/repository/user.go` -> `9.38`
   - `/Users/mukundhan/Projects/engram/internal/repository/user_test.go` -> `10.0`
6. CodeScene pre-commit safeguard:
   - `pre_commit_code_health_safeguard(git_repository_path=/Users/mukundhan/Projects/engram)`
   - result: `quality_gates=passed`
   - findings: none

### 2026-02-22 (Go migration CP24: runtime integration baseline for memory-admin dependencies)

1. Added runtime wiring for memory-admin route dependencies:
   - `cmd/api/main.go`
   - startup DB pool + ping using `internal/db`.
   - non-production wiring to `NewRouterWithDependencies` with DB-backed `admin.Service`.
   - production keeps memory-admin routes disabled until session-auth migration completes.
2. Added migration-time actor/project resolver bridges:
   - `internal/api/admin_actor.go` (`RequireAdminActorFromHeaders` with admin-role gating).
   - `internal/admin/project_resolver.go` (`PassthroughProjectResolver`).
3. Added/updated migrated tests:
   - `internal/api/admin_actor_test.go`
   - `internal/admin/project_resolver_test.go`
   - `internal/api/admin_memory_test.go` (`TestMountMemoryAdminRoutesCreateCollectionMapsMissingProjectTo400`)
4. Executed migrated tests one-by-one:
   - `/usr/local/go/bin/go test ./internal/admin -run '^TestPassthroughProjectResolverReturnsTrimmedProjectID$' -v`
   - `/usr/local/go/bin/go test ./internal/admin -run '^TestPassthroughProjectResolverRejectsEmptyProjectID$' -v`
   - `/usr/local/go/bin/go test ./internal/api -run '^TestRequireAdminActorFromHeadersReturnsActor$' -v`
   - `/usr/local/go/bin/go test ./internal/api -run '^TestRequireAdminActorFromHeadersRejectsMissingHeaders$' -v`
   - `/usr/local/go/bin/go test ./internal/api -run '^TestRequireAdminActorFromHeadersRejectsInvalidUserID$' -v`
   - `/usr/local/go/bin/go test ./internal/api -run '^TestRequireAdminActorFromHeadersRejectsNonAdminRole$' -v`
   - `/usr/local/go/bin/go test ./internal/api -run '^TestMountMemoryAdminRoutesCreateCollectionMapsMissingProjectTo400$' -v`
5. Full Go verification:
   - `/usr/local/go/bin/go test ./...` passed.
6. File-level CodeScene checks (all Go migration files before commit):
   - `/Users/mukundhan/Projects/engram/cmd/api/main.go` -> `10.0`
   - `/Users/mukundhan/Projects/engram/internal/admin/project_resolver.go` -> `10.0`
   - `/Users/mukundhan/Projects/engram/internal/admin/project_resolver_test.go` -> `10.0`
   - `/Users/mukundhan/Projects/engram/internal/admin/service.go` -> `8.54`
   - `/Users/mukundhan/Projects/engram/internal/admin/service_test.go` -> `9.38`
   - `/Users/mukundhan/Projects/engram/internal/api/admin_actor.go` -> `10.0`
   - `/Users/mukundhan/Projects/engram/internal/api/admin_actor_test.go` -> `10.0`
   - `/Users/mukundhan/Projects/engram/internal/api/admin_memory.go` -> `6.88`
   - `/Users/mukundhan/Projects/engram/internal/api/admin_memory_test.go` -> `9.68`
   - `/Users/mukundhan/Projects/engram/internal/api/router.go` -> `10.0`
   - `/Users/mukundhan/Projects/engram/internal/api/router_test.go` -> `10.0`
   - `/Users/mukundhan/Projects/engram/internal/auth/password.go` -> `10.0`
   - `/Users/mukundhan/Projects/engram/internal/auth/password_test.go` -> `9.68`
   - `/Users/mukundhan/Projects/engram/internal/config/config.go` -> `9.68`
   - `/Users/mukundhan/Projects/engram/internal/config/config_test.go` -> `10.0`
   - `/Users/mukundhan/Projects/engram/internal/db/db.go` -> `10.0`
   - `/Users/mukundhan/Projects/engram/internal/db/db_test.go` -> `9.61`
   - `/Users/mukundhan/Projects/engram/internal/embeddings/errors.go` -> `10.0`
   - `/Users/mukundhan/Projects/engram/internal/embeddings/local.go` -> `10.0`
   - `/Users/mukundhan/Projects/engram/internal/embeddings/local_test.go` -> `9.68`
   - `/Users/mukundhan/Projects/engram/internal/embeddings/service.go` -> `9.09`
   - `/Users/mukundhan/Projects/engram/internal/embeddings/service_test.go` -> `10.0`
   - `/Users/mukundhan/Projects/engram/internal/models/chat.go` -> `10.0`
   - `/Users/mukundhan/Projects/engram/internal/models/collection.go` -> `None` (struct-only file; score unavailable)
   - `/Users/mukundhan/Projects/engram/internal/models/document.go` -> `10.0`
   - `/Users/mukundhan/Projects/engram/internal/models/engram.go` -> `None` (struct-only file; score unavailable)
   - `/Users/mukundhan/Projects/engram/internal/models/mcp_token.go` -> `10.0`
   - `/Users/mukundhan/Projects/engram/internal/models/oauth.go` -> `None` (struct-only file; score unavailable)
   - `/Users/mukundhan/Projects/engram/internal/models/project.go` -> `None` (struct-only file; score unavailable)
   - `/Users/mukundhan/Projects/engram/internal/models/user.go` -> `10.0`
   - `/Users/mukundhan/Projects/engram/internal/repository/admin_engram.go` -> `9.68`
   - `/Users/mukundhan/Projects/engram/internal/repository/admin_engram_test.go` -> `9.38`
   - `/Users/mukundhan/Projects/engram/internal/repository/admin_engram_update.go` -> `9.61`
   - `/Users/mukundhan/Projects/engram/internal/repository/admin_engram_update_test.go` -> `9.25`
   - `/Users/mukundhan/Projects/engram/internal/repository/admin_session.go` -> `9.68`
   - `/Users/mukundhan/Projects/engram/internal/repository/admin_session_test.go` -> `9.68`
   - `/Users/mukundhan/Projects/engram/internal/repository/chat.go` -> `9.02`
   - `/Users/mukundhan/Projects/engram/internal/repository/chat_message.go` -> `10.0`
   - `/Users/mukundhan/Projects/engram/internal/repository/chat_message_test.go` -> `10.0`
   - `/Users/mukundhan/Projects/engram/internal/repository/chat_pinning.go` -> `8.81`
   - `/Users/mukundhan/Projects/engram/internal/repository/chat_pinning_test.go` -> `9.38`
   - `/Users/mukundhan/Projects/engram/internal/repository/chat_test.go` -> `9.09`
   - `/Users/mukundhan/Projects/engram/internal/repository/collection.go` -> `9.68`
   - `/Users/mukundhan/Projects/engram/internal/repository/collection_test.go` -> `9.09`
   - `/Users/mukundhan/Projects/engram/internal/repository/document.go` -> `8.81`
   - `/Users/mukundhan/Projects/engram/internal/repository/document_test.go` -> `8.72`
   - `/Users/mukundhan/Projects/engram/internal/repository/engram.go` -> `9.68`
   - `/Users/mukundhan/Projects/engram/internal/repository/engram_helpers_test.go` -> `10.0`
   - `/Users/mukundhan/Projects/engram/internal/repository/engram_rehydration.go` -> `9.68`
   - `/Users/mukundhan/Projects/engram/internal/repository/engram_rehydration_test.go` -> `10.0`
   - `/Users/mukundhan/Projects/engram/internal/repository/engram_repository_test.go` -> `10.0`
   - `/Users/mukundhan/Projects/engram/internal/repository/engram_store.go` -> `10.0`
   - `/Users/mukundhan/Projects/engram/internal/repository/engram_unit_test.go` -> `10.0`
   - `/Users/mukundhan/Projects/engram/internal/repository/engram_write.go` -> `10.0`
   - `/Users/mukundhan/Projects/engram/internal/repository/engram_write_test.go` -> `9.26`
   - `/Users/mukundhan/Projects/engram/internal/repository/mcp_token.go` -> `10.0`
   - `/Users/mukundhan/Projects/engram/internal/repository/mcp_token_test.go` -> `9.68`
   - `/Users/mukundhan/Projects/engram/internal/repository/oauth.go` -> `9.38`
   - `/Users/mukundhan/Projects/engram/internal/repository/oauth_test.go` -> `9.38`
   - `/Users/mukundhan/Projects/engram/internal/repository/project.go` -> `10.0`
   - `/Users/mukundhan/Projects/engram/internal/repository/project_test.go` -> `10.0`
   - `/Users/mukundhan/Projects/engram/internal/repository/user.go` -> `9.38`
   - `/Users/mukundhan/Projects/engram/internal/repository/user_test.go` -> `10.0`
7. CodeScene pre-commit safeguard:
   - `pre_commit_code_health_safeguard(git_repository_path=/Users/mukundhan/Projects/engram)`
   - result: `quality_gates=passed`
   - findings: stable complexity advisory in `internal/api/admin_memory.go` (no gate failure)

### 2026-02-22 (Go migration CP25: context-first admin actor hardening baseline)

1. Hardened admin actor resolution to a context-first flow:
   - `internal/api/admin_actor.go`
   - added `WithAdminActor`, `AdminActorFromContext`, and `RequireAdminActorFromContext`.
   - retained header parsing as migration adapter and added `AdminActorHeaderBridge` middleware for non-production compatibility.
2. Updated runtime wiring to consume context-based actor resolution:
   - `cmd/api/main.go`
   - `RouterDependencies.RequireAdminActor` now uses `RequireAdminActorFromContext`.
   - non-production runtime wraps the handler with `AdminActorHeaderBridge`.
3. Added/updated migrated tests:
   - `internal/api/admin_actor_test.go`
   - `TestAdminActorResolversReturnActor`
   - `TestRequireAdminActorFromHeadersRejectsMissingHeaders`
   - `TestRequireAdminActorFromHeadersRejectsInvalidUserID`
   - `TestRequireAdminActorFromHeadersRejectsNonAdminRole`
   - `TestRequireAdminActorFromContextRejectsMissingActor`
   - `TestRequireAdminActorFromContextRejectsNonAdminRole`
   - `TestAdminActorHeaderBridgeInjectsContextActor`
   - `TestAdminActorHeaderBridgeLeavesRequestUnauthenticatedWhenHeadersMissing`
4. Executed migrated tests one-by-one:
   - `/usr/local/go/bin/go test ./internal/api -run '^TestAdminActorResolversReturnActor$' -v`
   - `/usr/local/go/bin/go test ./internal/api -run '^TestRequireAdminActorFromHeadersRejectsMissingHeaders$' -v`
   - `/usr/local/go/bin/go test ./internal/api -run '^TestRequireAdminActorFromHeadersRejectsInvalidUserID$' -v`
   - `/usr/local/go/bin/go test ./internal/api -run '^TestRequireAdminActorFromHeadersRejectsNonAdminRole$' -v`
   - `/usr/local/go/bin/go test ./internal/api -run '^TestRequireAdminActorFromContextRejectsMissingActor$' -v`
   - `/usr/local/go/bin/go test ./internal/api -run '^TestRequireAdminActorFromContextRejectsNonAdminRole$' -v`
   - `/usr/local/go/bin/go test ./internal/api -run '^TestAdminActorHeaderBridgeInjectsContextActor$' -v`
   - `/usr/local/go/bin/go test ./internal/api -run '^TestAdminActorHeaderBridgeLeavesRequestUnauthenticatedWhenHeadersMissing$' -v`
5. Full Go verification:
   - `/usr/local/go/bin/go test ./...` passed.
6. File-level CodeScene checks (all Go migration files before commit):
   - `/Users/mukundhan/Projects/engram/cmd/api/main.go` -> `10.0`
   - `/Users/mukundhan/Projects/engram/internal/admin/project_resolver.go` -> `10.0`
   - `/Users/mukundhan/Projects/engram/internal/admin/project_resolver_test.go` -> `10.0`
   - `/Users/mukundhan/Projects/engram/internal/admin/service.go` -> `8.54`
   - `/Users/mukundhan/Projects/engram/internal/admin/service_test.go` -> `9.38`
   - `/Users/mukundhan/Projects/engram/internal/api/admin_actor.go` -> `10.0`
   - `/Users/mukundhan/Projects/engram/internal/api/admin_actor_test.go` -> `10.0`
   - `/Users/mukundhan/Projects/engram/internal/api/admin_memory.go` -> `6.88`
   - `/Users/mukundhan/Projects/engram/internal/api/admin_memory_test.go` -> `9.68`
   - `/Users/mukundhan/Projects/engram/internal/api/router.go` -> `10.0`
   - `/Users/mukundhan/Projects/engram/internal/api/router_test.go` -> `10.0`
   - `/Users/mukundhan/Projects/engram/internal/auth/password.go` -> `10.0`
   - `/Users/mukundhan/Projects/engram/internal/auth/password_test.go` -> `9.68`
   - `/Users/mukundhan/Projects/engram/internal/config/config.go` -> `9.68`
   - `/Users/mukundhan/Projects/engram/internal/config/config_test.go` -> `10.0`
   - `/Users/mukundhan/Projects/engram/internal/db/db.go` -> `10.0`
   - `/Users/mukundhan/Projects/engram/internal/db/db_test.go` -> `9.61`
   - `/Users/mukundhan/Projects/engram/internal/embeddings/errors.go` -> `10.0`
   - `/Users/mukundhan/Projects/engram/internal/embeddings/local.go` -> `10.0`
   - `/Users/mukundhan/Projects/engram/internal/embeddings/local_test.go` -> `9.68`
   - `/Users/mukundhan/Projects/engram/internal/embeddings/service.go` -> `9.09`
   - `/Users/mukundhan/Projects/engram/internal/embeddings/service_test.go` -> `10.0`
   - `/Users/mukundhan/Projects/engram/internal/models/chat.go` -> `10.0`
   - `/Users/mukundhan/Projects/engram/internal/models/collection.go` -> `None` (struct-only file; score unavailable)
   - `/Users/mukundhan/Projects/engram/internal/models/document.go` -> `10.0`
   - `/Users/mukundhan/Projects/engram/internal/models/engram.go` -> `None` (struct-only file; score unavailable)
   - `/Users/mukundhan/Projects/engram/internal/models/mcp_token.go` -> `10.0`
   - `/Users/mukundhan/Projects/engram/internal/models/oauth.go` -> `None` (struct-only file; score unavailable)
   - `/Users/mukundhan/Projects/engram/internal/models/project.go` -> `None` (struct-only file; score unavailable)
   - `/Users/mukundhan/Projects/engram/internal/models/user.go` -> `10.0`
   - `/Users/mukundhan/Projects/engram/internal/repository/admin_engram.go` -> `9.68`
   - `/Users/mukundhan/Projects/engram/internal/repository/admin_engram_test.go` -> `9.38`
   - `/Users/mukundhan/Projects/engram/internal/repository/admin_engram_update.go` -> `9.61`
   - `/Users/mukundhan/Projects/engram/internal/repository/admin_engram_update_test.go` -> `9.25`
   - `/Users/mukundhan/Projects/engram/internal/repository/admin_session.go` -> `9.68`
   - `/Users/mukundhan/Projects/engram/internal/repository/admin_session_test.go` -> `9.68`
   - `/Users/mukundhan/Projects/engram/internal/repository/chat.go` -> `9.02`
   - `/Users/mukundhan/Projects/engram/internal/repository/chat_message.go` -> `10.0`
   - `/Users/mukundhan/Projects/engram/internal/repository/chat_message_test.go` -> `10.0`
   - `/Users/mukundhan/Projects/engram/internal/repository/chat_pinning.go` -> `8.81`
   - `/Users/mukundhan/Projects/engram/internal/repository/chat_pinning_test.go` -> `9.38`
   - `/Users/mukundhan/Projects/engram/internal/repository/chat_test.go` -> `9.09`
   - `/Users/mukundhan/Projects/engram/internal/repository/collection.go` -> `9.68`
   - `/Users/mukundhan/Projects/engram/internal/repository/collection_test.go` -> `9.09`
   - `/Users/mukundhan/Projects/engram/internal/repository/document.go` -> `8.81`
   - `/Users/mukundhan/Projects/engram/internal/repository/document_test.go` -> `8.72`
   - `/Users/mukundhan/Projects/engram/internal/repository/engram.go` -> `9.68`
   - `/Users/mukundhan/Projects/engram/internal/repository/engram_helpers_test.go` -> `10.0`
   - `/Users/mukundhan/Projects/engram/internal/repository/engram_rehydration.go` -> `9.68`
   - `/Users/mukundhan/Projects/engram/internal/repository/engram_rehydration_test.go` -> `10.0`
   - `/Users/mukundhan/Projects/engram/internal/repository/engram_repository_test.go` -> `10.0`
   - `/Users/mukundhan/Projects/engram/internal/repository/engram_store.go` -> `10.0`
   - `/Users/mukundhan/Projects/engram/internal/repository/engram_unit_test.go` -> `10.0`
   - `/Users/mukundhan/Projects/engram/internal/repository/engram_write.go` -> `10.0`
   - `/Users/mukundhan/Projects/engram/internal/repository/engram_write_test.go` -> `9.26`
   - `/Users/mukundhan/Projects/engram/internal/repository/mcp_token.go` -> `10.0`
   - `/Users/mukundhan/Projects/engram/internal/repository/mcp_token_test.go` -> `9.68`
   - `/Users/mukundhan/Projects/engram/internal/repository/oauth.go` -> `9.38`
   - `/Users/mukundhan/Projects/engram/internal/repository/oauth_test.go` -> `9.38`
   - `/Users/mukundhan/Projects/engram/internal/repository/project.go` -> `10.0`
   - `/Users/mukundhan/Projects/engram/internal/repository/project_test.go` -> `10.0`
   - `/Users/mukundhan/Projects/engram/internal/repository/user.go` -> `9.38`
   - `/Users/mukundhan/Projects/engram/internal/repository/user_test.go` -> `10.0`
7. CodeScene pre-commit safeguard:
   - `pre_commit_code_health_safeguard(git_repository_path=/Users/mukundhan/Projects/engram)`
   - result: `quality_gates=passed`
   - findings: none

### 2026-02-22 (Go migration CP26: session-cookie actor middleware baseline)

1. Added canonical Go session primitives:
   - `internal/auth/session.go`
   - signed session token encode/decode via `SessionManager` with default `session` cookie name.
2. Added DB-backed session actor middleware:
   - `internal/api/session_actor_middleware.go`
   - decodes session cookie, canonicalizes actor from DB via `GetUserAuthRecordByID`, injects actor context for admin route authorization.
3. Updated actor bridge behavior and runtime composition:
   - `internal/api/admin_actor.go` header bridge now preserves an existing context actor.
   - `cmd/api/main.go` now wires `SessionActorMiddleware` before optional non-production header bridge.
   - extracted runtime helper functions in `cmd/api/main.go` to keep startup flow maintainable.
4. Added/updated migrated tests:
   - `internal/auth/session_test.go`
   - `internal/api/session_actor_middleware_test.go`
   - `internal/api/admin_actor_test.go`
5. Executed migrated tests one-by-one:
   - `/usr/local/go/bin/go test ./internal/auth -run '^TestSessionManagerEncodeDecodeRoundTrip$' -v`
   - `/usr/local/go/bin/go test ./internal/auth -run '^TestSessionManagerDecodeRejectsTamperedToken$' -v`
   - `/usr/local/go/bin/go test ./internal/auth -run '^TestSessionManagerDecodeRequestReadsCookie$' -v`
   - `/usr/local/go/bin/go test ./internal/auth -run '^TestNewSessionManagerRejectsEmptySecret$' -v`
   - `/usr/local/go/bin/go test ./internal/api -run '^TestSessionActorMiddlewareInjectsActorFromSessionCookie$' -v`
   - `/usr/local/go/bin/go test ./internal/api -run '^TestSessionActorMiddlewareSkipsInactiveUsers$' -v`
   - `/usr/local/go/bin/go test ./internal/api -run '^TestSessionActorMiddlewareIgnoresInvalidSessionCookie$' -v`
   - `/usr/local/go/bin/go test ./internal/api -run '^TestAdminActorHeaderBridgeResolvesActor$' -v`
   - `/usr/local/go/bin/go test ./internal/api -run '^TestAdminActorHeaderBridgeLeavesRequestUnauthenticatedWhenHeadersMissing$' -v`
6. Full Go verification:
   - `/usr/local/go/bin/go test ./...` passed.
7. File-level CodeScene checks (all Go migration files before commit):
   - `/Users/mukundhan/Projects/engram/cmd/api/main.go` -> `10.0`
   - `/Users/mukundhan/Projects/engram/internal/admin/project_resolver.go` -> `10.0`
   - `/Users/mukundhan/Projects/engram/internal/admin/project_resolver_test.go` -> `10.0`
   - `/Users/mukundhan/Projects/engram/internal/admin/service.go` -> `8.54`
   - `/Users/mukundhan/Projects/engram/internal/admin/service_test.go` -> `9.38`
   - `/Users/mukundhan/Projects/engram/internal/api/admin_actor.go` -> `10.0`
   - `/Users/mukundhan/Projects/engram/internal/api/admin_actor_test.go` -> `10.0`
   - `/Users/mukundhan/Projects/engram/internal/api/admin_memory.go` -> `6.88`
   - `/Users/mukundhan/Projects/engram/internal/api/admin_memory_test.go` -> `9.68`
   - `/Users/mukundhan/Projects/engram/internal/api/router.go` -> `10.0`
   - `/Users/mukundhan/Projects/engram/internal/api/router_test.go` -> `10.0`
   - `/Users/mukundhan/Projects/engram/internal/api/session_actor_middleware.go` -> `9.38`
   - `/Users/mukundhan/Projects/engram/internal/api/session_actor_middleware_test.go` -> `9.68`
   - `/Users/mukundhan/Projects/engram/internal/auth/password.go` -> `10.0`
   - `/Users/mukundhan/Projects/engram/internal/auth/password_test.go` -> `9.68`
   - `/Users/mukundhan/Projects/engram/internal/auth/session.go` -> `9.38`
   - `/Users/mukundhan/Projects/engram/internal/auth/session_test.go` -> `10.0`
   - `/Users/mukundhan/Projects/engram/internal/config/config.go` -> `9.68`
   - `/Users/mukundhan/Projects/engram/internal/config/config_test.go` -> `10.0`
   - `/Users/mukundhan/Projects/engram/internal/db/db.go` -> `10.0`
   - `/Users/mukundhan/Projects/engram/internal/db/db_test.go` -> `9.61`
   - `/Users/mukundhan/Projects/engram/internal/embeddings/errors.go` -> `10.0`
   - `/Users/mukundhan/Projects/engram/internal/embeddings/local.go` -> `10.0`
   - `/Users/mukundhan/Projects/engram/internal/embeddings/local_test.go` -> `9.68`
   - `/Users/mukundhan/Projects/engram/internal/embeddings/service.go` -> `9.09`
   - `/Users/mukundhan/Projects/engram/internal/embeddings/service_test.go` -> `10.0`
   - `/Users/mukundhan/Projects/engram/internal/models/chat.go` -> `10.0`
   - `/Users/mukundhan/Projects/engram/internal/models/collection.go` -> `None` (struct-only file; score unavailable)
   - `/Users/mukundhan/Projects/engram/internal/models/document.go` -> `10.0`
   - `/Users/mukundhan/Projects/engram/internal/models/engram.go` -> `None` (struct-only file; score unavailable)
   - `/Users/mukundhan/Projects/engram/internal/models/mcp_token.go` -> `10.0`
   - `/Users/mukundhan/Projects/engram/internal/models/oauth.go` -> `None` (struct-only file; score unavailable)
   - `/Users/mukundhan/Projects/engram/internal/models/project.go` -> `None` (struct-only file; score unavailable)
   - `/Users/mukundhan/Projects/engram/internal/models/user.go` -> `10.0`
   - `/Users/mukundhan/Projects/engram/internal/repository/admin_engram.go` -> `9.68`
   - `/Users/mukundhan/Projects/engram/internal/repository/admin_engram_test.go` -> `9.38`
   - `/Users/mukundhan/Projects/engram/internal/repository/admin_engram_update.go` -> `9.61`
   - `/Users/mukundhan/Projects/engram/internal/repository/admin_engram_update_test.go` -> `9.25`
   - `/Users/mukundhan/Projects/engram/internal/repository/admin_session.go` -> `9.68`
   - `/Users/mukundhan/Projects/engram/internal/repository/admin_session_test.go` -> `9.68`
   - `/Users/mukundhan/Projects/engram/internal/repository/chat.go` -> `9.02`
   - `/Users/mukundhan/Projects/engram/internal/repository/chat_message.go` -> `10.0`
   - `/Users/mukundhan/Projects/engram/internal/repository/chat_message_test.go` -> `10.0`
   - `/Users/mukundhan/Projects/engram/internal/repository/chat_pinning.go` -> `8.81`
   - `/Users/mukundhan/Projects/engram/internal/repository/chat_pinning_test.go` -> `9.38`
   - `/Users/mukundhan/Projects/engram/internal/repository/chat_test.go` -> `9.09`
   - `/Users/mukundhan/Projects/engram/internal/repository/collection.go` -> `9.68`
   - `/Users/mukundhan/Projects/engram/internal/repository/collection_test.go` -> `9.09`
   - `/Users/mukundhan/Projects/engram/internal/repository/document.go` -> `8.81`
   - `/Users/mukundhan/Projects/engram/internal/repository/document_test.go` -> `8.72`
   - `/Users/mukundhan/Projects/engram/internal/repository/engram.go` -> `9.68`
   - `/Users/mukundhan/Projects/engram/internal/repository/engram_helpers_test.go` -> `10.0`
   - `/Users/mukundhan/Projects/engram/internal/repository/engram_rehydration.go` -> `9.68`
   - `/Users/mukundhan/Projects/engram/internal/repository/engram_rehydration_test.go` -> `10.0`
   - `/Users/mukundhan/Projects/engram/internal/repository/engram_repository_test.go` -> `10.0`
   - `/Users/mukundhan/Projects/engram/internal/repository/engram_store.go` -> `10.0`
   - `/Users/mukundhan/Projects/engram/internal/repository/engram_unit_test.go` -> `10.0`
   - `/Users/mukundhan/Projects/engram/internal/repository/engram_write.go` -> `10.0`
   - `/Users/mukundhan/Projects/engram/internal/repository/engram_write_test.go` -> `9.26`
   - `/Users/mukundhan/Projects/engram/internal/repository/mcp_token.go` -> `10.0`
   - `/Users/mukundhan/Projects/engram/internal/repository/mcp_token_test.go` -> `9.68`
   - `/Users/mukundhan/Projects/engram/internal/repository/oauth.go` -> `9.38`
   - `/Users/mukundhan/Projects/engram/internal/repository/oauth_test.go` -> `9.38`
   - `/Users/mukundhan/Projects/engram/internal/repository/project.go` -> `10.0`
   - `/Users/mukundhan/Projects/engram/internal/repository/project_test.go` -> `10.0`
   - `/Users/mukundhan/Projects/engram/internal/repository/user.go` -> `9.38`
   - `/Users/mukundhan/Projects/engram/internal/repository/user_test.go` -> `10.0`
8. CodeScene pre-commit safeguard:
   - `pre_commit_code_health_safeguard(git_repository_path=/Users/mukundhan/Projects/engram)`
   - result: `quality_gates=passed`
   - findings: none

### 2026-02-22 (Go migration CP27: session login/logout/csrf route baseline)

1. Added session-auth API routes:
   - `internal/api/session_auth.go`
   - `GET /api/v1/session/csrf`
   - `POST /api/v1/session/login`
   - `POST /api/v1/session/logout`
   - `GET /api/v1/me`
2. Updated API router composition:
   - `internal/api/router.go`
   - `RouterDependencies` now carries `SessionAuth` and mounts session-auth routes when configured.
3. Updated runtime dependency wiring:
   - `cmd/api/main.go`
   - injects session-auth dependencies (session manager, user lookup by username/id, password verification, CSRF token generator).
   - runtime no longer depends on migration-time header bridge for actor context.
4. Added migrated tests:
   - `internal/api/session_auth_test.go`
   - `internal/api/router_test.go` session-auth mounting tests
5. Executed migrated tests one-by-one:
   - `/usr/local/go/bin/go test ./internal/api -run '^TestSessionAuthRoutesNotMountedWithoutDependencies$' -v`
   - `/usr/local/go/bin/go test ./internal/api -run '^TestSessionAuthRoutesMountedWithDependencies$' -v`
   - `/usr/local/go/bin/go test ./internal/api -run '^TestMountSessionAuthRoutesCSRFIssuesTokenAndCookie$' -v`
   - `/usr/local/go/bin/go test ./internal/api -run '^TestMountSessionAuthRoutesLoginSetsSessionCookie$' -v`
   - `/usr/local/go/bin/go test ./internal/api -run '^TestMountSessionAuthRoutesLoginRejectsInvalidCSRF$' -v`
   - `/usr/local/go/bin/go test ./internal/api -run '^TestMountSessionAuthRoutesMeUsesSessionActorMiddleware$' -v`
6. Full Go verification:
   - `/usr/local/go/bin/go test ./...` passed.
7. File-level CodeScene checks (all Go migration files before commit):
   - `/Users/mukundhan/Projects/engram/cmd/api/main.go` -> `10.0`
   - `/Users/mukundhan/Projects/engram/internal/admin/project_resolver.go` -> `10.0`
   - `/Users/mukundhan/Projects/engram/internal/admin/project_resolver_test.go` -> `10.0`
   - `/Users/mukundhan/Projects/engram/internal/admin/service.go` -> `8.54`
   - `/Users/mukundhan/Projects/engram/internal/admin/service_test.go` -> `9.38`
   - `/Users/mukundhan/Projects/engram/internal/api/admin_actor.go` -> `10.0`
   - `/Users/mukundhan/Projects/engram/internal/api/admin_actor_test.go` -> `10.0`
   - `/Users/mukundhan/Projects/engram/internal/api/admin_memory.go` -> `6.88`
   - `/Users/mukundhan/Projects/engram/internal/api/admin_memory_test.go` -> `9.68`
   - `/Users/mukundhan/Projects/engram/internal/api/router.go` -> `10.0`
   - `/Users/mukundhan/Projects/engram/internal/api/router_test.go` -> `10.0`
   - `/Users/mukundhan/Projects/engram/internal/api/session_actor_middleware.go` -> `9.38`
   - `/Users/mukundhan/Projects/engram/internal/api/session_actor_middleware_test.go` -> `9.68`
   - `/Users/mukundhan/Projects/engram/internal/api/session_auth.go` -> `8.15`
   - `/Users/mukundhan/Projects/engram/internal/api/session_auth_test.go` -> `10.0`
   - `/Users/mukundhan/Projects/engram/internal/auth/password.go` -> `10.0`
   - `/Users/mukundhan/Projects/engram/internal/auth/password_test.go` -> `9.68`
   - `/Users/mukundhan/Projects/engram/internal/auth/session.go` -> `9.38`
   - `/Users/mukundhan/Projects/engram/internal/auth/session_test.go` -> `10.0`
   - `/Users/mukundhan/Projects/engram/internal/config/config.go` -> `9.68`
   - `/Users/mukundhan/Projects/engram/internal/config/config_test.go` -> `10.0`
   - `/Users/mukundhan/Projects/engram/internal/db/db.go` -> `10.0`
   - `/Users/mukundhan/Projects/engram/internal/db/db_test.go` -> `9.61`
   - `/Users/mukundhan/Projects/engram/internal/embeddings/errors.go` -> `10.0`
   - `/Users/mukundhan/Projects/engram/internal/embeddings/local.go` -> `10.0`
   - `/Users/mukundhan/Projects/engram/internal/embeddings/local_test.go` -> `9.68`
   - `/Users/mukundhan/Projects/engram/internal/embeddings/service.go` -> `9.09`
   - `/Users/mukundhan/Projects/engram/internal/embeddings/service_test.go` -> `10.0`
   - `/Users/mukundhan/Projects/engram/internal/models/chat.go` -> `10.0`
   - `/Users/mukundhan/Projects/engram/internal/models/collection.go` -> `None` (struct-only file; score unavailable)
   - `/Users/mukundhan/Projects/engram/internal/models/document.go` -> `10.0`
   - `/Users/mukundhan/Projects/engram/internal/models/engram.go` -> `None` (struct-only file; score unavailable)
   - `/Users/mukundhan/Projects/engram/internal/models/mcp_token.go` -> `10.0`
   - `/Users/mukundhan/Projects/engram/internal/models/oauth.go` -> `None` (struct-only file; score unavailable)
   - `/Users/mukundhan/Projects/engram/internal/models/project.go` -> `None` (struct-only file; score unavailable)
   - `/Users/mukundhan/Projects/engram/internal/models/user.go` -> `10.0`
   - `/Users/mukundhan/Projects/engram/internal/repository/admin_engram.go` -> `9.68`
   - `/Users/mukundhan/Projects/engram/internal/repository/admin_engram_test.go` -> `9.38`
   - `/Users/mukundhan/Projects/engram/internal/repository/admin_engram_update.go` -> `9.61`
   - `/Users/mukundhan/Projects/engram/internal/repository/admin_engram_update_test.go` -> `9.25`
   - `/Users/mukundhan/Projects/engram/internal/repository/admin_session.go` -> `9.68`
   - `/Users/mukundhan/Projects/engram/internal/repository/admin_session_test.go` -> `9.68`
   - `/Users/mukundhan/Projects/engram/internal/repository/chat.go` -> `9.02`
   - `/Users/mukundhan/Projects/engram/internal/repository/chat_message.go` -> `10.0`
   - `/Users/mukundhan/Projects/engram/internal/repository/chat_message_test.go` -> `10.0`
   - `/Users/mukundhan/Projects/engram/internal/repository/chat_pinning.go` -> `8.81`
   - `/Users/mukundhan/Projects/engram/internal/repository/chat_pinning_test.go` -> `9.38`
   - `/Users/mukundhan/Projects/engram/internal/repository/chat_test.go` -> `9.09`
   - `/Users/mukundhan/Projects/engram/internal/repository/collection.go` -> `9.68`
   - `/Users/mukundhan/Projects/engram/internal/repository/collection_test.go` -> `9.09`
   - `/Users/mukundhan/Projects/engram/internal/repository/document.go` -> `8.81`
   - `/Users/mukundhan/Projects/engram/internal/repository/document_test.go` -> `8.72`
   - `/Users/mukundhan/Projects/engram/internal/repository/engram.go` -> `9.68`
   - `/Users/mukundhan/Projects/engram/internal/repository/engram_helpers_test.go` -> `10.0`
   - `/Users/mukundhan/Projects/engram/internal/repository/engram_rehydration.go` -> `9.68`
   - `/Users/mukundhan/Projects/engram/internal/repository/engram_rehydration_test.go` -> `10.0`
   - `/Users/mukundhan/Projects/engram/internal/repository/engram_repository_test.go` -> `10.0`
   - `/Users/mukundhan/Projects/engram/internal/repository/engram_store.go` -> `10.0`
   - `/Users/mukundhan/Projects/engram/internal/repository/engram_unit_test.go` -> `10.0`
   - `/Users/mukundhan/Projects/engram/internal/repository/engram_write.go` -> `10.0`
   - `/Users/mukundhan/Projects/engram/internal/repository/engram_write_test.go` -> `9.26`
   - `/Users/mukundhan/Projects/engram/internal/repository/mcp_token.go` -> `10.0`
   - `/Users/mukundhan/Projects/engram/internal/repository/mcp_token_test.go` -> `9.68`
   - `/Users/mukundhan/Projects/engram/internal/repository/oauth.go` -> `9.38`
   - `/Users/mukundhan/Projects/engram/internal/repository/oauth_test.go` -> `9.38`
   - `/Users/mukundhan/Projects/engram/internal/repository/project.go` -> `10.0`
   - `/Users/mukundhan/Projects/engram/internal/repository/project_test.go` -> `10.0`
   - `/Users/mukundhan/Projects/engram/internal/repository/user.go` -> `9.38`
   - `/Users/mukundhan/Projects/engram/internal/repository/user_test.go` -> `10.0`
8. CodeScene pre-commit safeguard:
   - `pre_commit_code_health_safeguard(git_repository_path=/Users/mukundhan/Projects/engram)`
   - result: `quality_gates=passed`
   - findings: none
