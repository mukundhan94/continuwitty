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
| CP2 | 2026-02-22 | In Progress | `internal/db` port + migrated db tests |
| CP3 | 2026-02-22 | Pending | API entrypoint scaffold and phase docs sync |

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

### CP2 - DB Baseline (Planned)

- Port DB connection helpers and schema bootstrap logic into `internal/db`.
- Port `api/tests/test_db.py` behavior into Go tests.
- Execute migrated DB tests one-by-one before commit.

### CP3 - API Entrypoint Scaffold (Planned)

- Add `cmd/api/main.go` placeholder startup path.
- Add router/bootstrap stubs needed for side-by-side rollout.
- Sync migration logs and checkpoint docs.
