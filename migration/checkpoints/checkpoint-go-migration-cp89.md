# Go Migration Checkpoint CP89

Date: 2026-02-26
Branch: `migrate`

## Scope

Phase 4 ingestion file-upload API baseline (`POST /api/v1/ingestion/file`) with route-level multipart parser options.

## Completed Work

- Extended ingestion REST route module:
  - `internal/api/ingestion_api.go`
  - added endpoint:
    - `POST /api/v1/ingestion/file`
  - supports multipart ingestion fields:
    - `project_id`
    - `title` (optional)
    - `visibility_scope` (optional)
    - `chunk_size_chars` (optional)
    - `chunk_overlap_chars` (optional)
    - `metadata_json` (optional)
    - `file` (required)
  - includes metadata JSON parsing with route-level max-byte guard and normalized ingestion defaults.
- Added route-level parser options wiring:
  - `internal/api/router.go`
  - `cmd/api/main.go`
  - `RouterDependencies.IngestionOptions` now configures ingestion route parser limits from runtime settings.
- Updated runtime ingestion adapter contract:
  - `cmd/api/ingestion_routes_adapter.go`
  - added `IngestFile` adapter support for `ingestion.Service.IngestFile`.
- Expanded route tests:
  - `internal/api/ingestion_api_test.go`
  - added file upload happy path coverage
  - added metadata validation rejection coverage (invalid JSON + too-large JSON).

## One-by-One Tests Executed

- `TestMountIngestionRoutesRegistersEndpoints`
- `TestIngestTextHandlerWritesCreatedResponse`
- `TestIngestFileHandlerWritesCreatedResponse`
- `TestIngestFileHandlerRejectsInvalidMetadataJSON`
- `TestListDocumentsHandlerUsesDefaultPaging`
- `TestQueryDocumentsHandlerAppliesDefaultTopK`
- `TestQueryBlendedHandlerAppliesDefaultTopK`
- `TestIngestionRoutesRequireAuthenticatedActor`
- `TestIngestTextHandlerMapsServiceError`
- `TestDataRoutesMountedWithDependencies`
- `TestBuildChatRouterRegistersRuntimeRoutes`
- `TestProjectResolutionAdaptersReturnServiceResolution`

## Full Verification

- `go test ./...` passed.

## Code Health (CodeScene)

- `cmd/api/main.go`: `10.0`
- `cmd/api/ingestion_routes_adapter.go`: `10.0`
- `internal/api/ingestion_api.go`: `10.0`
- `internal/api/ingestion_api_test.go`: `10.0`
- `internal/api/router.go`: `10.0`

Pre-commit safeguard:

- `pre_commit_code_health_safeguard(git_repository_path=/Users/mukundhan/Projects/engram)` -> `quality_gates=passed`
- Findings: none.
