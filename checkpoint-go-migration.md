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
| CP7 | 2026-02-22 | In Progress | Repository layer migration kickoff (engram/document/chat paths) |

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

### CP7 - Repository Layer Kickoff (Planned)

- Start with engram repository read/write parity for the first API-backed flows.
- Migrate repository-focused tests incrementally and keep one-by-one test execution.
