# Go Migration Checkpoints

> Track incremental migration from Python/FastAPI to Go using `migrate.md`.
> This file is intentionally phase-specific and complements `checkpoint.md`.

## Scope

- Source plan: `migrate.md`
- Migration style: side-by-side runtime with parity-first tests.
- Commit policy: one focused checkpoint commit per migration slice.

## Checkpoint Timeline

| Checkpoint | Date | Status | Scope |
|---|---|---|---|
| CP1 | 2026-02-22 | Completed | Go module scaffold + `internal/config` port + migrated config tests |
| CP2 | 2026-02-22 | Completed | `internal/db` port + migrated db tests |
| CP3 | 2026-02-22 | Completed | API entrypoint scaffold + health/version route parity tests |
| CP4 | 2026-02-22 | Completed | Auth baseline (`password` + CSRF) parity with migrated tests |
| CP5 | 2026-02-22 | Completed | User models + user repository parity tests |
| CP6 | 2026-02-22 | Completed | Embeddings baseline (local + fallback service parity tests) |
| CP7 | 2026-02-22 | Completed | Repository helper baseline for engram query/rerank/rehydration formatting parity |
| CP8 | 2026-02-22 | Completed | Engram repository DB read parity (`list_engrams` + `query_engrams`) with migrated tests |
| CP9 | 2026-02-22 | Completed | Engram rehydration/source read parity (`get_rehydration_bundle` + `get_engram_sources`) |
| CP10 | 2026-02-22 | In Progress | Engram write-path parity (`create_engram` + source/artifact inserts) |

## Checkpoint Details

### CP1 - Module + Config Baseline

- Added Go module and dependency baseline.
- Added `internal/config/config.go` with:
  - env-backed settings loading (`envconfig`)
  - production hardening validation parity
  - debug snapshot redaction parity
  - environment helpers (`ShouldLogSettings`, `IsProductionEnv`)
- Ported `api/tests/test_config_settings.py` to `internal/config/config_test.go`.
- Executed migrated tests one-by-one:
  - `TestBuildDebugSettingsSnapshotRedactsSecrets`
  - `TestShouldLogSettingsOnlyForDevModes`
  - `TestProductionSettingsRejectInsecureDefaults`
  - `TestProductionSettingsAcceptHardenedValues`

### CP2 - DB Baseline

- Added `internal/db/db.go` with:
  - transaction wrapper semantics (`WithTransaction`) with commit/rollback parity
  - schema bootstrap entrypoint (`EnsureSchemaInitialized`) and default schema path resolver
  - production bootstrap-admin hash hardening parity
  - pgx pool helper wiring (`NewPool`, `Ping`, `PgxPoolBeginner`)
- Ported `api/tests/test_db.py` behavior to `internal/db/db_test.go`.
- Executed migrated tests one-by-one:
  - `TestWithTransactionCommitsOnSuccess`
  - `TestWithTransactionRollsBackOnError`
  - `TestEnsureSchemaInitializedExecutesSchemaSQL`
  - `TestHardenBootstrapAdminCredentialsUpdatesDefaultHash`
  - `TestHardenBootstrapAdminCredentialsSkipsNonDefaultHash`

### CP3 - API Entrypoint Scaffold

- Added API runtime entrypoint:
  - `cmd/api/main.go`
- Added initial Chi router wiring:
  - `internal/api/router.go`
  - `/healthz`
  - `/api/v1/version`
- Ported basic API unit tests from `api/tests/test_api_unit.py`:
  - `internal/api/router_test.go`
- Executed migrated tests one-by-one:
  - `TestHealthz`
  - `TestVersionEndpoint`
- Ran file-level CodeScene checks for all migrated Go files before commit:
  - `cmd/api/main.go` → 10.0
  - `internal/api/router.go` → 10.0
  - `internal/api/router_test.go` → 10.0
  - `internal/config/config.go` → 9.68
  - `internal/config/config_test.go` → 10.0
  - `internal/db/db.go` → 10.0
  - `internal/db/db_test.go` → 9.61

### CP4 - Auth Baseline

- Added `internal/auth/password.go` with Python-parity behavior:
  - `GenerateCSRFToken` (URL-safe random token)
  - `HashPassword` (`pbkdf2_sha256$390000$<salt>$<digest>`)
  - `VerifyPassword` parser + constant-time compare
- Ported auth tests to `internal/auth/password_test.go`.
- Executed migrated tests one-by-one:
  - `TestHashPasswordWithProvidedSaltProducesExpectedFormat`
  - `TestVerifyPasswordRoundTrip`
  - `TestVerifyPasswordMatchesBootstrapAdminHash`
  - `TestVerifyPasswordRejectsInvalidEncodings`
  - `TestGenerateCSRFTokenIsURLSafeAndRandom`
- Ran file-level CodeScene checks for current Go migration files before commit:
  - `cmd/api/main.go` → 10.0
  - `internal/api/router.go` → 10.0
  - `internal/api/router_test.go` → 10.0
  - `internal/config/config.go` → 9.68
  - `internal/config/config_test.go` → 10.0
  - `internal/db/db.go` → 10.0
  - `internal/db/db_test.go` → 9.61
  - `internal/auth/password.go` → 10.0
  - `internal/auth/password_test.go` → 9.68

### CP5 - User Repository Baseline

- Added user model primitives:
  - `internal/models/user.go`
- Added user repository operations:
  - `internal/repository/user.go`
  - auth lookup by username/id
  - list users
  - create user (duplicate username mapping)
  - update user (nullable patch fields)
- Ported tests from `api/tests/test_user_repository.py`:
  - `internal/repository/user_test.go`
- Executed migrated tests one-by-one:
  - `TestGetUserAuthRecordLooksUpUsername`
  - `TestListUsersReturnsRecords`
  - `TestCreateUserReturnsCreatedUser`
  - `TestCreateUserReturnsUsernameExistsOnDuplicate`
  - `TestUpdateUserReturnsNilWhenNotFound`
  - `TestUpdateUserAppliesProvidedFields`
- Ran file-level CodeScene checks for all Go migration files before commit:
  - `cmd/api/main.go` → 10.0
  - `internal/api/router.go` → 10.0
  - `internal/api/router_test.go` → 10.0
  - `internal/auth/password.go` → 10.0
  - `internal/auth/password_test.go` → 9.68
  - `internal/config/config.go` → 9.68
  - `internal/config/config_test.go` → 10.0
  - `internal/db/db.go` → 10.0
  - `internal/db/db_test.go` → 9.61
  - `internal/models/user.go` → 10.0
  - `internal/repository/user.go` → 9.38
  - `internal/repository/user_test.go` → 10.0

### CP6 - Embeddings Baseline

- Added embeddings package:
  - `internal/embeddings/errors.go`
  - `internal/embeddings/local.go`
  - `internal/embeddings/service.go`
- Ported deterministic local embedding algorithm parity (`embed_text_local`).
- Added fallback embedding service parity for provider failures.
- Ported tests from:
  - `api/tests/test_embedding.py`
  - `api/tests/test_embeddings_service.py`
  - new files:
    - `internal/embeddings/local_test.go`
    - `internal/embeddings/service_test.go`
- Executed migrated tests one-by-one:
  - `TestEmbedTextLocalIsDeterministicAndFixedDim`
  - `TestEmbedTextLocalHandlesEmptyText`
  - `TestEmbedTextLocalRejectsInvalidDim`
  - `TestEmbeddingServiceFallsBackToLocalProvider`
  - `TestEmbeddingServiceEmbedManyUsesProviderIDFromActiveProvider`
- Ran file-level CodeScene checks for all Go migration files before commit:
  - `cmd/api/main.go` → 10.0
  - `internal/api/router.go` → 10.0
  - `internal/api/router_test.go` → 10.0
  - `internal/auth/password.go` → 10.0
  - `internal/auth/password_test.go` → 9.68
  - `internal/config/config.go` → 9.68
  - `internal/config/config_test.go` → 10.0
  - `internal/db/db.go` → 10.0
  - `internal/db/db_test.go` → 9.61
  - `internal/models/user.go` → 10.0
  - `internal/repository/user.go` → 9.38
  - `internal/repository/user_test.go` → 10.0
  - `internal/embeddings/errors.go` → 10.0
  - `internal/embeddings/local.go` → 10.0
  - `internal/embeddings/service.go` → 9.09
  - `internal/embeddings/local_test.go` → 9.68
  - `internal/embeddings/service_test.go` → 10.0

### CP7 - Repository Helper Baseline

- Added engram model primitives required for repository layer migration:
  - `internal/models/engram.go`
- Added engram repository helper baseline:
  - `internal/repository/engram.go`
  - helper parity for:
    - retrieval text assembly
    - vector literal formatting
    - lexical overlap + combined rerank score
    - query `WHERE` clause builder
    - citation/decision/open-question formatting
    - rehydration context markdown composition
    - compact summary + assistant excerpt extraction
    - engram JSON payload serialization
- Ported helper/unit tests from:
  - `api/tests/test_repository_helpers.py`
  - `api/tests/test_repository_unit.py`
  - new files:
    - `internal/repository/engram_helpers_test.go`
    - `internal/repository/engram_unit_test.go`
- Executed migrated tests one-by-one:
  - `TestVectorLiteralFormat`
  - `TestBuildRetrievalTextUsesOverride`
  - `TestBuildRetrievalTextFallbackComposesFields`
  - `TestLexicalOverlapScorePrefersMatchingTerms`
  - `TestCombinedRankScoreUsesDenseAndLexicalSignals`
  - `TestPackCitationsDeduplicatesURLs`
  - `TestResolveCompactSummaryUsesDetailedForGenericChatSnapshot`
  - `TestExtractDetailedExcerptPrefersAssistantSection`
  - `TestBuildEngramQueryWhereIncludesAllFilters`
  - `TestRerankByCombinedScorePrefersLexicalOverlap`
  - `TestFormatCitationsTruncatesAndStripsNewlines`
  - `TestFormatCitationsReturnsDefaultForEmptyList`
  - `TestFormatDecisionsFormatsEntriesAndDefaults`
  - `TestFormatOpenQuestionsFormatsEntriesAndDefaults`
  - `TestBuildRehydrationContextMarkdownIncludesExpectedSections`
  - `TestBuildEngramJSONPayloadSerializesReportAndSourceSessionID`
- Ran file-level CodeScene checks for all Go migration files before commit:
  - `cmd/api/main.go` → 10.0
  - `internal/api/router.go` → 10.0
  - `internal/api/router_test.go` → 10.0
  - `internal/auth/password.go` → 10.0
  - `internal/auth/password_test.go` → 9.68
  - `internal/config/config.go` → 9.68
  - `internal/config/config_test.go` → 10.0
  - `internal/db/db.go` → 10.0
  - `internal/db/db_test.go` → 9.61
  - `internal/embeddings/errors.go` → 10.0
  - `internal/embeddings/local.go` → 10.0
  - `internal/embeddings/local_test.go` → 9.68
  - `internal/embeddings/service.go` → 9.09
  - `internal/embeddings/service_test.go` → 10.0
  - `internal/models/engram.go` → N/A (struct-only file; score unavailable)
  - `internal/models/user.go` → 10.0
  - `internal/repository/engram.go` → 9.11
  - `internal/repository/engram_helpers_test.go` → 10.0
  - `internal/repository/engram_unit_test.go` → 10.0
  - `internal/repository/user.go` → 9.38
  - `internal/repository/user_test.go` → 10.0

### CP8 - Engram Repository DB Read Parity

- Added engram repository DB operations baseline:
  - `internal/repository/engram_store.go`
  - `ListEngrams` with actor/project visibility filtering and pagination parity
  - `QueryEngrams` with vector candidate query + lexical reranking parity
  - introduced input wrappers to reduce call-site argument coupling:
    - `ListEngramsInput`
    - `QueryEngramsInput`
- Refactored repository cohesion:
  - moved DB-facing code out of `internal/repository/engram.go` into `internal/repository/engram_store.go`
  - extracted payload/summary helpers in `internal/repository/engram.go` to keep helper methods small and maintainable
- Added DB-path tests:
  - `internal/repository/engram_repository_test.go`
  - `TestListEngramsAppliesVisibilityAndProjectFilters`
  - `TestListEngramsWithoutActorOmitsVisibilityClause`
  - `TestQueryEngramsBuildsQueryAndReranks`
- Executed migrated tests one-by-one:
  - `TestListEngramsAppliesVisibilityAndProjectFilters`
  - `TestListEngramsWithoutActorOmitsVisibilityClause`
  - `TestQueryEngramsBuildsQueryAndReranks`
- Ran file-level CodeScene checks for all Go migration files before commit:
  - `cmd/api/main.go` → 10.0
  - `internal/api/router.go` → 10.0
  - `internal/api/router_test.go` → 10.0
  - `internal/auth/password.go` → 10.0
  - `internal/auth/password_test.go` → 9.68
  - `internal/config/config.go` → 9.68
  - `internal/config/config_test.go` → 10.0
  - `internal/db/db.go` → 10.0
  - `internal/db/db_test.go` → 9.61
  - `internal/embeddings/errors.go` → 10.0
  - `internal/embeddings/local.go` → 10.0
  - `internal/embeddings/local_test.go` → 9.68
  - `internal/embeddings/service.go` → 9.09
  - `internal/embeddings/service_test.go` → 10.0
  - `internal/models/engram.go` → N/A (struct-only file; score unavailable)
  - `internal/models/user.go` → 10.0
  - `internal/repository/engram.go` → 9.68
  - `internal/repository/engram_helpers_test.go` → 10.0
  - `internal/repository/engram_repository_test.go` → 10.0
  - `internal/repository/engram_store.go` → 10.0
  - `internal/repository/engram_unit_test.go` → 10.0
  - `internal/repository/user.go` → 9.38
  - `internal/repository/user_test.go` → 10.0

### CP9 - Engram Rehydration/Source Read Parity

- Added rehydration/source repository operations:
  - `internal/repository/engram_rehydration.go`
  - `GetRehydrationBundle`
  - `GetEngramSources`
  - actor-scoped visibility checks for read-path access control parity
- Added model shapes for rehydration/source responses:
  - `internal/models/engram.go`
  - `RehydrationBundle`
  - `EngramSourceRecord`
- Added DB-path tests:
  - `internal/repository/engram_rehydration_test.go`
  - `TestGetRehydrationBundleReturnsNilWhenNotFound`
  - `TestGetRehydrationBundleBuildsContextWithVisibilityFilter`
  - `TestGetEngramSourcesReturnsRowsWhenVisible`
  - `TestGetEngramSourcesReturnsEmptyWhenNotVisible`
- Executed migrated tests one-by-one:
  - `TestGetRehydrationBundleReturnsNilWhenNotFound`
  - `TestGetRehydrationBundleBuildsContextWithVisibilityFilter`
  - `TestGetEngramSourcesReturnsRowsWhenVisible`
  - `TestGetEngramSourcesReturnsEmptyWhenNotVisible`
- Ran file-level CodeScene checks for all Go migration files before commit:
  - `cmd/api/main.go` → 10.0
  - `internal/api/router.go` → 10.0
  - `internal/api/router_test.go` → 10.0
  - `internal/auth/password.go` → 10.0
  - `internal/auth/password_test.go` → 9.68
  - `internal/config/config.go` → 9.68
  - `internal/config/config_test.go` → 10.0
  - `internal/db/db.go` → 10.0
  - `internal/db/db_test.go` → 9.61
  - `internal/embeddings/errors.go` → 10.0
  - `internal/embeddings/local.go` → 10.0
  - `internal/embeddings/local_test.go` → 9.68
  - `internal/embeddings/service.go` → 9.09
  - `internal/embeddings/service_test.go` → 10.0
  - `internal/models/engram.go` → N/A (struct-only file; score unavailable)
  - `internal/models/user.go` → 10.0
  - `internal/repository/engram.go` → 9.68
  - `internal/repository/engram_helpers_test.go` → 10.0
  - `internal/repository/engram_rehydration.go` → 9.68
  - `internal/repository/engram_rehydration_test.go` → 10.0
  - `internal/repository/engram_repository_test.go` → 10.0
  - `internal/repository/engram_store.go` → 10.0
  - `internal/repository/engram_unit_test.go` → 10.0
  - `internal/repository/user.go` → 9.38
  - `internal/repository/user_test.go` → 10.0

### CP10 - Engram Write Path Parity (Planned)

- Port write-path operations next:
  - `create_engram_with_report`
  - `create_engram`
  - source/artifact insert paths
- Migrate parity tests incrementally with one-by-one execution.
