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
| CP3 | 2026-02-22 | In Progress | API entrypoint scaffold and phase docs sync |

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

### CP3 - API Entrypoint Scaffold (Planned)

- Add `cmd/api/main.go` placeholder startup path.
- Add router/bootstrap stubs needed for side-by-side rollout.
- Sync migration logs and checkpoint docs.
