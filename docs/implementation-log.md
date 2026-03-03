# Engram Vault - Implementation Log

> Chronological log of implementation changes by date and pass.
> See [README](../README.md) for project overview. See [migration/checkpoints/checkpoint.md](../migration/checkpoints/checkpoint.md) for milestone summary.

---

## Implementation Log

### 2026-03-03 (Phase 40 hardening: acceptance coverage for scoped link-curation refresh)

1. Extended Phase 40 acceptance feature in `acceptance-tests/features/phase40-curation-mock.feature`:
   - added scenario: "Link hygiene refresh rejects mismatched project scope".
2. Added step support in `acceptance-tests/src/steps/phase40-curation-mock.steps.ts`:
   - refresh helper now supports scoped payloads.
   - happy-path link refresh now sends scoped `project_id`.
   - mismatch scenario asserts HTTP `400` and detail:
     - `project_id does not match target engram project`.
3. Validation:
   - `make acceptance-bddgen`
   - `make acceptance-typecheck`
   - `make acceptance-test-mock-docker` -> `26 passed`

### 2026-03-03 (Phase 40 hardening: scoped project guard for link-curation refresh)

1. Added scoped project enforcement for link-curation refresh in `internal/admin/service_link_curation.go`:
   - `EngramLinkCurationSuggestionRefreshRequest` now accepts optional `project_id`.
   - refresh now returns `ErrProjectScopeMismatch` when scoped `project_id` does not match the source engram project.
2. Extended REST + MCP parity for scoped refresh:
   - REST route `POST /api/v1/admin/memory/engrams/{engram_id}/links/curation/refresh` now accepts optional `project_id` and forwards it to service.
   - MCP `engram.curation_refresh_links` now accepts optional `project_id` and forwards scope through compatibility dispatch + runtime adapter.
3. Extended bad-request error mapping:
   - REST admin error writer now maps `ErrProjectScopeMismatch` to HTTP `400`.
   - MCP dispatch now maps `ErrProjectScopeMismatch` to invalid-params response.
4. Added regression coverage:
   - admin service mismatch guard test.
   - REST refresh-route payload forwarding + mismatch-to-400 test.
   - MCP parity/validation tests for `project_id` forwarding and mismatch error mapping.
5. Validation:
   - `go test ./internal/admin ./internal/api ./internal/mcp ./cmd/api -count=1`
   - `make lint`
   - `make test-unit`
   - CodeScene `pre_commit_code_health_safeguard`: `quality_gates=passed`

### 2026-03-03 (Phase 40 continuation: MCP parity for link-curation refresh)

1. Added MCP tool support for on-demand link curation refresh:
   - new tool: `engram.curation_refresh_links`
   - dispatch parses admin-only inputs (`source_engram_id`, optional hygiene controls) and returns `curation_refresh` payload.
2. Extended MCP compatibility contracts:
   - added `EngramCurationRefreshService` plus request/response types in `internal/mcp/compatibility_service.go`.
   - wired dispatch handler registration, tool catalog visibility, and metadata schema entries.
3. Added runtime adapter wiring:
   - `cmd/api/mcp_engram_admin_curation_adapter.go` now bridges MCP refresh requests to `admin.Service.RefreshEngramLinkCurationSuggestions`.
   - `cmd/api/main.go` now injects `EngramCurationRefresh` dependency into compatibility service construction.
4. Added MCP regression coverage:
   - `internal/mcp/compatibility_service_engram_curation_refresh_test.go` validates direct/tools-call parity, request mapping, admin-only enforcement, input validation, and service error mapping.
5. Validation:
   - `go test ./internal/mcp ./cmd/api -count=1`

### 2026-03-03 (Phase 40 continuation: on-demand admin link-curation refresh route)

1. Added on-demand link curation refresh workflow in `internal/admin/service_link_curation.go`:
   - new service method `RefreshEngramLinkCurationSuggestions` generates link hygiene recommendations for one source engram and persists deduped `suggested` link curation rows.
   - persisted payload dedupe key uses (`link_id`, `target_engram_id`, `suggested_action`) to avoid duplicate pending records.
   - skips auto-archived hygiene action (`archive_stale_low_value`) while preserving actionable/manual-review link curation suggestions.
2. Extended admin service dependencies/contracts in `internal/admin/service.go`:
   - added `EngramLinkCurationSuggestionRefreshRequest/Response`.
   - added `recommendLinkHygiene` dependency with graph-service default binding.
3. Added admin REST route in `internal/api`:
   - `POST /api/v1/admin/memory/engrams/{engram_id}/links/curation/refresh`
   - payload supports optional hygiene controls (`include_archived`, `limit`, `stale_after_days`, `low_value_threshold`).
4. Expanded test coverage:
   - `internal/admin/service_link_curation_test.go` for create/dedupe/not-found behavior.
   - `internal/api/admin_memory_curation_routes_test.go` for refresh route actor/payload forwarding and validation.
   - `acceptance-tests/features/phase40-curation-mock.feature` + steps now validate on-demand link refresh generates actionable `link` curation suggestions and `status=applied` succeeds.
5. Code health refactors:
   - refactored duplicated acceptance/action assertions and API route tests to satisfy CodeScene duplication gates.
   - split admin refresh route payload tests into a dedicated file to keep test modules below CodeScene duplication thresholds.
6. Validation:
   - `go test ./internal/admin ./internal/api -count=1`
   - `make acceptance-bddgen`
   - `make acceptance-typecheck`
   - `make acceptance-test-mock-docker` -> `25 passed`
   - CodeScene `pre_commit_code_health_safeguard`: `quality_gates=passed`

### 2026-03-03 (Phase 40 continuation: conflict-review link curation apply support)

1. Extended link curation apply-action handling in `internal/admin/service_curation.go`:
   - `status=applied` for `suggestion_type=link` now treats `review_relation_conflict` as an actionable link-archive workflow.
   - this aligns generated link curation suggestions with apply behavior so conflict-review suggestions no longer fail as unsupported.
2. Expanded regression coverage in `internal/admin/service_curation_test.go`:
   - added apply-path test for `review_relation_conflict`.
   - retained unsupported-action rejection by asserting unknown link actions still return `ErrMemoryCurationSuggestionApplyUnsupported`.
3. Validation:
   - `go test ./internal/admin -count=1`

### 2026-03-03 (Phase 40 continuation: scheduled link-hygiene curation persistence)

1. Extended scheduled hygiene execution in `cmd/api/chat_link_reinforcement.go`:
   - successful reinforcement hygiene runs now persist `suggestion_type=link` curation records for non-auto-archived recommendations.
   - link curation payloads include source/target/link identifiers, suggested action, hygiene category/severity/score, and detail metadata.
   - pending-suggestion dedupe now reuses payload identity (`link_id`, `target_engram_id`, `suggested_action`) to avoid duplicate suggestions across repeated scheduler runs.
2. Added targeted regression coverage in `cmd/api/chat_link_reinforcement_test.go`:
   - validates mixed auto-archive + manual-review recommendation handling.
   - validates dedupe behavior when matching pending link curation suggestions already exist.
3. Code health refactor:
   - decomposed curation persistence workflow into focused helpers to keep complexity and argument count within CodeScene gates.
4. Validation:
   - `go test ./cmd/api -count=1`
   - `make lint`
   - `make test-unit`
   - CodeScene `pre_commit_code_health_safeguard`: `quality_gates=passed`

### 2026-03-03 (Phase 40 continuation: link-applied curation orchestration)

1. Extended curation apply behavior for link suggestions in `internal/admin/service_curation.go`:
   - `status=applied` now supports `suggestion_type=link` payloads with archive-oriented hygiene actions.
   - archive actions dispatch through link archive workflow using payload `link_id`.
   - unsupported/non-archive link actions return `ErrMemoryCurationSuggestionApplyUnsupported`.
2. Expanded regression coverage:
   - admin service tests for link-archive apply success and unsupported-link-action rejection.
   - REST route test for bad-request mapping of unsupported apply workflow.
   - MCP compatibility test for unsupported apply error mapping.
3. Validation:
   - `go test ./internal/admin ./internal/api ./internal/mcp -count=1`
   - `make lint`
   - `make test-unit`
   - CodeScene `pre_commit_code_health_safeguard`: `quality_gates=passed`

### 2026-03-03 (Phase 40 continuation: applied-side-effect benchmark expansion)

1. Expanded curation benchmark suite in `internal/admin/service_curation_benchmark_test.go`:
   - `BenchmarkActionMemoryCurationSuggestionAppliedConsolidate`
   - `BenchmarkActionMemoryCurationSuggestionAppliedContradiction`
2. Updated benchmark artifact in `docs/phase40-curation-benchmark.md`:
   - refreshed command and benchmark table to include applied-side-effect baselines.
3. Validation:
   - `go test ./internal/admin -run '^$' -bench 'Benchmark(ActionMemoryCurationSuggestion|ActionMemoryCurationSuggestionAppliedConsolidate|ActionMemoryCurationSuggestionAppliedContradiction|SyncConsolidationCurationSuggestions(50Candidates|200Candidates))$' -benchmem`

### 2026-03-03 (Phase 40 continuation: applied-cascade acceptance coverage)

1. Extended Phase 40 acceptance feature with downstream-apply scenario:
   - `acceptance-tests/features/phase40-curation-mock.feature`
   - new scenario validates `status=applied` curation actions for both consolidation and contradiction paths.
2. Added step coverage in `acceptance-tests/src/steps/phase40-curation-mock.steps.ts`:
   - verifies curation payload references contain downstream record ids.
   - actions both curation suggestions with `status=applied`.
   - asserts applied curation status records include action audit fields.
   - asserts downstream records transition as expected:
     - consolidation suggestion `status=merged`
     - contradiction alert `status=resolved` with resolver metadata.
3. Validation:
   - `make acceptance-bddgen`
   - `make acceptance-typecheck`
   - `make acceptance-test-mock-docker` -> `24 passed`
   - CodeScene `pre_commit_code_health_safeguard`: `quality_gates=passed`

### 2026-03-03 (Phase 40 continuation: curation applied-status orchestration)

1. Extended curation action behavior in `internal/admin/service_curation.go`:
   - `status=applied` now loads the targeted curation suggestion and executes deterministic downstream actions before status transition.
   - `consolidate` suggestions dispatch consolidation action as `merged`.
   - `contradiction` suggestions dispatch contradiction-alert resolve as `resolved`.
2. Added repository read primitive:
   - `GetMemoryCurationSuggestion` in `internal/repository/memory_curation_suggestions.go`.
   - regression coverage for get-by-id + project scope in `internal/repository/memory_curation_suggestions_test.go`.
3. Expanded regression coverage:
   - admin service curation tests for apply-side-effect dispatch and invalid payload rejection.
   - REST route test for bad-request mapping of invalid curation payload actions.
   - MCP compatibility test for payload-invalid error mapping.
4. Validation:
   - `go test ./internal/admin ./internal/repository ./internal/api ./internal/mcp -count=1`
   - `make lint`
   - `make test-unit`
   - `make acceptance-test-mock-docker` -> `23 passed`
   - CodeScene `pre_commit_code_health_safeguard`: `quality_gates=passed`

### 2026-03-03 (Code health refactor: acceptance step argument-shape cleanup)

1. Refactored acceptance step helper signatures in:
   - `acceptance-tests/src/steps/phase38-contradiction-mock.steps.ts`
   - `acceptance-tests/src/steps/phase40-curation-mock.steps.ts`
2. Replaced multi-string helper arguments with structured input objects to reduce string-heavy argument density while preserving scenario behavior.
3. Validation:
   - `make acceptance-typecheck`
   - `make acceptance-test-mock-docker` -> `23 passed`
   - CodeScene `analyze_change_set` against `origin/migrate`: `quality_gates=passed`

### 2026-03-03 (Phase 39 closeout: complex temporal/engagement/trace benchmark coverage)

1. Added Phase 39 repository benchmark coverage in `internal/repository/engram_benchmark_test.go`:
   - `BenchmarkBuildEngramQueryWhereComplexTemporalEngagementTrace`
   - `BenchmarkBuildEngramQueryWhereComplexFilterMatrix`
2. Added benchmark artifact in `docs/phase39-query-benchmark.md`:
   - command, environment, `ns/op`, memory, and allocation profiles.
3. Validation:
   - `go test ./internal/repository -run '^$' -bench 'Benchmark(BuildEngramQueryWhereComplexTemporalEngagementTrace|BuildEngramQueryWhereComplexFilterMatrix)$' -benchmem`

### 2026-03-03 (Phase 40 closeout: curation benchmark baseline)

1. Added Phase 40 benchmark suite in `internal/admin/service_curation_benchmark_test.go`:
   - `BenchmarkActionMemoryCurationSuggestion`
   - `BenchmarkSyncConsolidationCurationSuggestions50Candidates`
   - `BenchmarkSyncConsolidationCurationSuggestions200Candidates`
2. Captured benchmark artifact in `docs/phase40-curation-benchmark.md`:
   - command, environment, `ns/op`, memory, and allocation profiles.
3. Validation:
   - `go test ./internal/admin -run '^$' -bench 'Benchmark(ActionMemoryCurationSuggestion|SyncConsolidationCurationSuggestions(50Candidates|200Candidates))$' -benchmem`

### 2026-03-03 (Phase 40 continuation: curation acceptance coverage)

1. Added deterministic acceptance coverage for Phase 40 curation flows:
   - feature: `acceptance-tests/features/phase40-curation-mock.feature`
   - steps: `acceptance-tests/src/steps/phase40-curation-mock.steps.ts`
2. Scenario validates end-to-end curation behavior:
   - seeds deterministic consolidation + contradiction prerequisites.
   - triggers consolidation/contradiction refresh workflows.
   - asserts generated curation type coverage (`consolidate`, `contradiction`).
   - actions one curation suggestion to `accepted`.
   - verifies accepted-list payload includes actioned record + audit fields.
3. Validation:
   - `make acceptance-bddgen`
   - `make acceptance-typecheck`
   - `make acceptance-test-mock-docker` -> `23 passed`

### 2026-03-03 (Phase 40 continuation: deterministic curation-generation hooks)

1. Added deterministic curation-generation sync in admin service:
   - new generation helpers in `internal/admin/service_curation_generation.go`.
   - `RefreshEngramConsolidationSuggestions` now rebuilds type `consolidate` curation suggestions from current `suggested` consolidation candidates.
   - `RefreshEngramContradictionAlerts` now rebuilds type `contradiction` curation suggestions from current `open` contradiction alerts.
2. Added repository reset primitive for deterministic rebuild:
   - `ResetSuggestedMemoryCurationSuggestions` with optional project scope and per-type filtering.
   - regression coverage added in `internal/repository/memory_curation_suggestions_test.go`.
3. Added generation-focused admin tests:
   - `internal/admin/service_curation_generation_test.go`.
   - updated refresh tests to assert/allow curation hook behavior.
4. Validation:
   - `make lint`
   - `make test-unit`
   - `make acceptance-test-mock-docker`
   - CodeScene `pre_commit_code_health_safeguard`: `passed`

### 2026-03-03 (Phase 40 continuation: memory curation suggestion API + MCP list/action parity)

1. Added admin memory service workflows for curation suggestion review/action:
   - `internal/admin/service_curation.go`:
     - `ListMemoryCurationSuggestions`
     - `ActionMemoryCurationSuggestion`
   - new admin-domain errors and request contracts in `internal/admin/service.go`.
2. Added admin REST routes for curation suggestion management:
   - `GET /api/v1/admin/memory/engrams/curation/suggestions`
   - `POST /api/v1/admin/memory/engrams/curation/suggestions/{suggestion_id}/action`
   - route parsing supports `project_id`, `session_id`, `suggestion_type`, `status`, paging.
3. Added MCP curation tooling parity:
   - `engram.curation_list`
   - `engram.curation_action`
   - compatibility dispatch parsers/handlers, catalog metadata entries, handler registration, and token project-policy coverage.
4. Added MCP runtime adapters in `cmd/api` for memory-admin service bridging:
   - `newMCPEngramCurationListAdapter`
   - `newMCPEngramCurationActionAdapter`
5. Added regression coverage:
   - admin service tests: `internal/admin/service_curation_test.go`
   - admin route tests: `internal/api/admin_memory_curation_routes_test.go`
   - MCP compatibility tests:
     - `internal/mcp/compatibility_service_engram_curation_list_test.go`
     - `internal/mcp/compatibility_service_engram_curation_action_test.go`
6. Validation:
   - `make lint`
   - `make test-unit`
   - `make acceptance-test-mock-docker`
   - CodeScene `pre_commit_code_health_safeguard`: `passed`

### 2026-03-03 (Phase 40 kickoff: memory curation suggestion schema/model/repository baseline)

1. Added schema baseline in `db/init/001_schema.sql`:
   - new table `memory_curation_suggestions`.
   - type/status checks:
     - `suggestion_type`: `auto_save`, `consolidate`, `contradiction`, `link`
     - `status`: `suggested`, `accepted`, `rejected`, `applied`
   - indexes:
     - `memory_curation_suggestions_project_status_idx`
     - `memory_curation_suggestions_session_type_idx`
2. Added model contracts in `internal/models/memory_curation_suggestion.go`:
   - `MemoryCurationSuggestionType` + parse helper.
   - `MemoryCurationSuggestionStatus` + parse helper.
   - `MemoryCurationSuggestion` persisted record shape.
3. Added repository baseline in `internal/repository/memory_curation_suggestions.go`:
   - `CreateMemoryCurationSuggestion`
   - `ListMemoryCurationSuggestions`
   - `ApplyMemoryCurationSuggestionAction`
   - normalized validation for project/reason/confidence/status inputs.
4. Added repository regression coverage in `internal/repository/memory_curation_suggestions_test.go`:
   - create defaults.
   - list filters + argument wiring.
   - action transition success, invalid status, and missing suggestion handling.
5. Validation:
   - `go test ./internal/models ./internal/repository -count=1`
   - CodeScene `pre_commit_code_health_safeguard`: `passed`

### 2026-03-03 (Phase 39 extension: trace-aware relation/depth query constraints)

1. Extended engram query contracts with trace-aware filters:
   - `relation_type`
   - `trace_depth`
2. Added repository trace filter support:
   - `internal/repository/engram.go` now applies depth-1 `EXISTS` filtering over active `engram_links`.
   - optional `relation_type` filtering is applied within trace constraints.
3. Added REST + MCP parity:
   - `internal/api/session_engrams.go` validates relation/trace filter contracts and defaults `trace_depth` to `1` when `relation_type` is provided.
   - `internal/mcp/compatibility_dispatch_engram_query_support.go` parses/validates `relation_type` + `trace_depth` and applies the same default.
   - `internal/mcp/catalog_metadata_data.go` exposes the new `engram.query` schema fields.
4. Added regression coverage:
   - repository SQL/parameter coverage in `internal/repository/engram_unit_test.go`.
   - MCP parity/validation coverage in `internal/mcp/compatibility_service_engram_query_test.go`.
   - session route request parsing/validation coverage in `internal/api/session_engrams_test.go`.
5. Validation:
   - `go test ./internal/models ./internal/repository ./internal/mcp ./internal/api -count=1`
   - CodeScene `pre_commit_code_health_safeguard`: `passed`

### 2026-03-03 (Phase 39 extension: recall/timeline temporal windows)

1. Extended engram query contracts with recall/timeline window filters:
   - `last_accessed_after`
   - `last_accessed_before`
   - `freshness_computed_after`
   - `freshness_computed_before`
2. Added repository query support:
   - `internal/repository/engram.go` now applies optional predicates on:
     - `COALESCE(last_accessed_at, created_at)`
     - `COALESCE(freshness_last_computed_at, created_at)`
3. Added REST + MCP parity:
   - `internal/api/session_engrams.go` validates temporal window ordering for:
     - created window
     - last-accessed window
     - freshness-computed window
   - `internal/mcp/compatibility_dispatch_engram_query_support.go` parses and validates the same windows.
   - `internal/mcp/catalog_metadata_data.go` exposes the new `engram.query` arguments.
4. Added regression coverage:
   - repository SQL/placeholder/arg coverage in `internal/repository/engram_unit_test.go`.
   - MCP parity/validation coverage in `internal/mcp/compatibility_service_engram_query_test.go`.
   - session route parsing/validation coverage in `internal/api/session_engrams_test.go`.
5. Validation:
   - `go test ./internal/models ./internal/repository ./internal/mcp ./internal/api -count=1`

### 2026-03-03 (Phase 39 extension: temporal query filters for engagement + freshness)

1. Extended engram query contracts with temporal/engagement filters:
   - `internal/models/engram.go` now includes:
     - `access_count_min`
     - `freshness_score_min`
2. Added repository query support:
   - `internal/repository/engram.go` now applies optional predicates:
     - `COALESCE(access_count, 0) >= access_count_min`
     - `COALESCE(freshness_score, 1.0) >= freshness_score_min`
3. Added MCP `engram.query` parser + catalog support:
   - `internal/mcp/compatibility_dispatch_engram_query_support.go` parses and validates the new filters.
   - `internal/mcp/catalog_metadata_data.go` exposes schema metadata for both arguments.
4. Added regression coverage:
   - repository SQL/placeholder/arg coverage in `internal/repository/engram_unit_test.go`.
   - MCP parity/validation coverage in `internal/mcp/compatibility_service_engram_query_test.go`.
5. Validation:
   - `go test ./internal/models ./internal/repository ./internal/mcp -count=1`
   - `make lint`
   - `make test-unit`
   - `make acceptance-test-mock-docker`
   - CodeScene `pre_commit_code_health_safeguard`: `passed`
   - CodeScene scores on touched Go files: `10.0`

### 2026-03-03 (Phase 39 kickoff: cost-aware context token budget baseline)

1. Added bounded context-budget controls to chat send workflows:
   - new optional `context_token_budget` in `chat.ChatMessageCreateRequest`.
   - forwarded from MCP `chat.send_message` params via `SessionMessageSendRequest` and message-send adapters.
2. Added cost-aware context assembly enforcement:
   - `internal/chat/context_token_budget.go` introduces deterministic token estimation and section-budget truncation helpers.
   - `AssembleChatContext` now applies bounded markdown assembly using normalized token budgets.
3. Added retrieval-audit budget diagnostics in chat context metadata:
   - `context_token_budget`
   - `context_token_estimate`
   - `context_token_truncated`
4. Updated tool schema metadata:
   - `internal/mcp/catalog_metadata_data.go` now advertises `context_token_budget` for `chat.send_message`.
5. Added regression tests:
   - context-budget unit coverage in `internal/chat/context_token_budget_test.go`.
   - context-budget assembly audit coverage in `internal/chat/context_budget_assembly_test.go`.
   - message runtime forwarding coverage in `internal/chat/message_runtime_test.go`.
   - REST forwarding coverage in `internal/api/chat_api_messages_test.go`.
   - MCP forwarding/validation coverage in:
     - `internal/mcp/compatibility_service_chat_send_message_test.go`
     - `internal/mcp/compatibility_service_chat_send_message_stream_test.go`
6. Validation:
   - `go test ./internal/chat ./internal/mcp ./internal/api ./cmd/api -count=1`
   - `make lint`
   - `make test-unit`
   - `make acceptance-test-mock-docker`
   - CodeScene `pre_commit_code_health_safeguard`: `passed`
   - CodeScene scores on touched Go files: `10.0`

### 2026-03-03 (Phase 38 closeout: contradiction quality acceptance enabled in default mock gate)

1. Fixed contradiction-link create SQL defect in `internal/repository/engram_links.go`:
   - `source_engram` and `target_engram` CTE membership filters referenced aliased columns without aliasing the `FROM engrams` relation.
   - updated CTEs to `FROM engrams source_engram` / `FROM engrams target_engram`, removing PostgreSQL missing-from-clause failures that surfaced as `500` in contradiction seed workflows.
2. Added regression guard in `internal/repository/engram_links_test.go`:
   - create-link SQL assertions now require explicit source/target CTE aliases.
3. Enabled contradiction warning quality scenario in default deterministic acceptance suite:
   - `acceptance-tests/features/phase38-contradiction-mock.feature` now includes `@mock` tag.
4. Validation:
   - targeted repo tests: `go test ./internal/repository -run 'TestCreateEngramLinkReturnsRecord|TestCreateEngramLinkReturnsDuplicateError' -count=1`
   - contradiction quality scenario: `ACCEPTANCE_BDD_TAGS='@phase38' docker compose --profile acceptance up --build --force-recreate --abort-on-container-exit acceptance-tests`
   - deterministic acceptance gate: `make acceptance-test-mock-docker`
   - backend quality gates: `make lint`, `make test-unit`

### 2026-03-03 (Phase 38 benchmark pass: contradiction warning synthesis latency baseline)

1. Added contradiction warning microbenchmarks in `internal/chat/context_contradictions_benchmark_test.go`:
   - `BenchmarkBuildContradictionWarnings50Paths`
   - `BenchmarkBuildContradictionWarnings200Paths`
2. Captured benchmark artifact in `docs/phase38-contradiction-benchmark.md`:
   - command, environment, measured `ns/op`, memory, and allocation profiles.
3. Added phase-scoped acceptance benchmark scenario scaffolding:
   - `acceptance-tests/features/phase38-contradiction-mock.feature`
   - `acceptance-tests/src/steps/phase38-contradiction-mock.steps.ts`
   - tagged for explicit Phase 38 runs (`@phase38`) and excluded from default `@mock` CI gate while contradiction-link seed path investigation continues.
4. Validation:
   - `go test ./internal/chat -run '^$' -bench 'BenchmarkBuildContradictionWarnings(50Paths|200Paths)$' -benchmem`
   - `make acceptance-bddgen`
   - `make acceptance-typecheck`
   - `make acceptance-test-mock`

### 2026-03-03 (Phase 38 continuation: contradiction alert admin API + MCP maintenance parity)

1. Added contradiction alert maintenance workflows in admin service:
   - new contracts and methods:
     - `RefreshEngramContradictionAlerts`
     - `ListEngramContradictionAlerts`
     - `ResolveEngramContradictionAlert`
   - repository dependency wiring for contradiction refresh/list/resolve paths.
   - admin-domain validation/not-found errors for contradiction resolve workflows.
2. Added admin REST routes for contradiction alerts:
   - `POST /api/v1/admin/memory/engrams/contradictions/refresh`
   - `GET /api/v1/admin/memory/engrams/contradictions/alerts`
   - `POST /api/v1/admin/memory/engrams/contradictions/alerts/{alert_id}/resolve`
   - route/query/payload validation for contradiction status (`open|resolved|dismissed`) with resolve-path guardrails (`resolved|dismissed` only).
3. Added MCP contradiction maintenance tools:
   - `engram.refresh_contradictions`
   - `engram.contradiction_list`
   - `engram.contradiction_resolve`
   - end-to-end MCP compatibility wiring: tool parsing, dispatch handlers, dependency interfaces, runtime dependencies, token project-policy scoping, and catalog metadata.
4. Added adapters and tests:
   - admin service tests in `internal/admin/service_contradiction_test.go`.
   - admin REST contradiction route tests in `internal/api/admin_memory_contradiction_routes_test.go`.
   - MCP compatibility tests:
     - `internal/mcp/compatibility_service_engram_refresh_contradictions_test.go`
     - `internal/mcp/compatibility_service_engram_contradiction_list_test.go`
     - `internal/mcp/compatibility_service_engram_contradiction_resolve_test.go`
   - API + MCP docs updated (`docs/api-reference.md`, `docs/mcp-guide.md`).
5. Validation:
   - `go test ./internal/admin ./internal/api ./internal/mcp ./cmd/api -count=1`
   - `make lint`
   - `make test-unit`
   - CodeScene `pre_commit_code_health_safeguard`: `passed`
   - CodeScene scores on touched Go files: `10.0` (catalog metadata data file reports `None` in score API).

### 2026-03-03 (Phase 38 continuation: contradiction alert persistence baseline)

1. Added contradiction alert persistence schema in `db/init/001_schema.sql`:
   - new `engram_contradiction_alerts` table keyed by `alert_id` + deterministic `alert_hash`.
   - status lifecycle (`open`/`resolved`/`dismissed`), contradiction link references, confidence, and resolve metadata.
   - supporting indexes for hash uniqueness, project/status recency reads, source-target lifecycle lookups, and link-id GIN lookup.
2. Added contradiction alert domain model in `internal/models/engram_contradiction.go`:
   - `ContradictionAlertStatus` enum with parser validation.
   - `EngramContradictionAlert` record used by repository + future API/MCP surfaces.
3. Added repository workflows in `internal/repository/engram_contradiction_alerts.go`:
   - `RefreshContradictionAlerts` groups active `contradicts` links and upserts deterministic alerts.
   - `ListContradictionAlerts` supports optional project/status filters and pagination.
   - `ResolveContradictionAlert` supports `resolved`/`dismissed` transitions for open alerts.
4. Added tests:
   - `internal/repository/engram_contradiction_alerts_test.go` for refresh/list/resolve behavior, filter forwarding, status validation, and missing-row handling.
   - `internal/models/engram_contradiction_test.go` for status parser coverage.
5. Validation:
   - `go test ./internal/models ./internal/repository -count=1`
   - `make lint`
   - `make test-unit`
   - CodeScene `pre_commit_code_health_safeguard`: `passed`
   - CodeScene scores: touched Go files `10.0`.

### 2026-03-03 (Phase 38 kickoff: contradiction warning baseline for chat + MCP)

1. Added contradiction trace metadata in linked-recall path assembly:
   - `internal/chat/context_links_trace.go` now marks contradiction-bearing paths when any traversed link relation is `contradicts`.
   - `EngramTracePath` now carries:
     - `has_contradiction`
     - `contradicting_link_ids`
2. Added contradiction warning synthesis in assembled chat context:
   - new warning model in `internal/chat/context.go`:
     - `ChatContradictionWarning`
     - `AssembledChatContext.contradiction_warnings`
   - new warning builder module `internal/chat/context_contradictions.go` with severity mapping and deterministic ordering.
3. Added response-payload parity across runtime surfaces:
   - `internal/chat/service.go`: send response now includes `contradiction_warnings`.
   - `internal/chat/message_runtime.go`: stream `meta`/`done` payloads now include `contradiction_warnings`.
   - `internal/mcp/compatibility_service.go` + `cmd/api/mcp_message_send_adapter.go`: MCP send-message response now forwards `contradiction_warnings`.
4. Expanded tests:
   - `internal/chat/context_test.go` now covers contradiction-warning generation for `contradicts` trace paths.
   - `internal/chat/message_runtime_test.go` and `internal/chat/service_test.go` now assert contradiction-warning propagation in send/stream payloads.
   - split linked-recall helper content into `internal/chat/context_linked_helpers_test.go` to keep test-module size maintainable and preserve code-health thresholds.
5. Roadmap and docs alignment:
   - `Plan.md`: added Phase 38 section and marked as in-progress with delivered baseline + remaining scope.
   - `migration/checkpoints/checkpoint.md`: added Phase 38 progress tracker and current-state summary.
   - `docs/api-reference.md` and `docs/mcp-guide.md`: chat response metadata now documents `contradiction_warnings`.
6. Validation:
   - `go test ./internal/chat ./internal/mcp ./cmd/api -count=1`
   - `make lint`
   - `make test-unit`
   - CodeScene scores on touched Go files: `10.0`.

### 2026-03-03 (Phase 36 closeout: engagement/freshness weighting calibration + latency benchmarks)

1. Calibrated retrieval reranking with engagement/freshness signals:
   - `internal/repository/engram_store.go` now includes:
     - `COALESCE(access_count, 0) AS access_count`
     - `COALESCE(freshness_score, 1.0) AS freshness_score`
   - composite ranking weights now include five deterministic factors:
     - dense `0.55`
     - lexical `0.20`
     - feedback `0.10`
     - engagement `0.10`
     - freshness `0.05`
2. Refactored ranking logic into focused module for maintainability and code health:
   - moved ranking/tokenization/score-normalization routines into `internal/repository/engram_rerank.go`.
   - kept `internal/repository/engram.go` focused on retrieval text/context helpers.
3. Expanded deterministic coverage:
   - `internal/repository/engram_unit_test.go` now validates engagement+freshness preference behavior.
   - `internal/repository/engram_repository_test.go` asserts query projection includes calibrated ranking inputs.
   - added microbenchmarks in `internal/repository/engram_benchmark_test.go`:
     - `BenchmarkRerankByCombinedScore50Candidates`
     - `BenchmarkRerankByCombinedScore200Candidates`
4. Captured benchmark notes and calibration rationale:
   - `docs/phase36-relevance-calibration.md` records weight rationale, command, environment, and measured latency.
5. Roadmap/checkpoint alignment:
   - `Plan.md`: Phase 36 marked completed with calibration + benchmark documentation in delivered scope.
   - `migration/checkpoints/checkpoint.md`: Phase 36 summary/tracker marked completed.
6. Validation:
   - `go test ./internal/repository -count=1`
   - `go test ./internal/repository -run '^$' -bench 'BenchmarkRerankByCombinedScore(50Candidates|200Candidates)$' -benchmem`
   - `make lint`
   - `make test-unit`
   - CodeScene scores on touched Go files: `10.0`.

### 2026-03-03 (Phase 37 kickoff slice: freshness maintenance baseline)

1. Added freshness schema baseline in `db/init/001_schema.sql`:
   - `engrams.freshness_score` (`DOUBLE PRECISION`, default `1.0`).
   - `engrams.freshness_last_computed_at` (`TIMESTAMPTZ`).
   - `engrams_freshness_idx` for freshness/recency maintenance queries.
2. Added repository maintenance routine in `internal/repository/engram_freshness.go`:
   - `RefreshEngramFreshnessScores` recomputes freshness using exponential half-life decay (`exp(-ln(2) * age_days / half_life_days)`).
   - supports optional project scoping and deterministic defaults (`half_life_days=70`, `reference_time=utc now`).
3. Added admin service and API wiring for explicit maintenance execution:
   - `internal/admin/service.go`: `RefreshEngramFreshness`.
   - `internal/api/admin_memory_engrams.go`: `POST /api/v1/admin/memory/engrams/freshness/refresh`.
4. Added/updated tests:
   - `internal/repository/engram_freshness_test.go`.
   - `internal/admin/service_test.go`.
   - `internal/api/admin_memory_test.go`.
5. Documentation/state alignment:
   - `plan.md`: Phase 35 marked completed; Phase 36/37 set to in-progress with delivered/remaining details.
   - `migration/checkpoints/checkpoint.md`: added Phase 36 and Phase 37 progress trackers.
6. Validation:
   - `go test ./internal/repository ./internal/admin ./internal/api -count=1`
   - `make test-unit`
   - CodeScene scores on touched Go files: `10.0`.

### 2026-03-03 (Phase 37 extension: MCP freshness maintenance tool)

1. Added MCP freshness maintenance dispatch path:
   - new tool `engram.refresh_freshness` (tools-call alias `engram_refresh_freshness`) in:
     - `internal/mcp/compatibility_dispatch_engram_freshness_support.go`
     - `internal/mcp/compatibility_dispatch_engram_mutation_handlers.go`
     - `internal/mcp/catalog.go`
     - `internal/mcp/catalog_metadata_data.go`
   - admin-only actor guard enforced in dispatch.
2. Added compatibility service + runtime wiring:
   - new compatibility interface/request/response contracts in `internal/mcp/compatibility_service.go`.
   - dependency wiring in `cmd/api/main.go`.
   - new adapter `cmd/api/mcp_engram_admin_freshness_adapter.go` forwarding to admin freshness service.
3. Added token project-policy support:
   - `internal/mcp/token_authorization_policy.go` now treats `engram.refresh_freshness` as an optional-project tool for project allowlist normalization/autofill behavior.
4. Added regression coverage:
   - `internal/mcp/compatibility_service_engram_refresh_freshness_test.go` verifies direct/tools parity, admin guard, parameter validation, and internal error mapping.
5. Documentation updates:
   - `docs/mcp-guide.md` tool catalog now includes `engram.feedback` and `engram.refresh_freshness`.
   - roadmap/checkpoint entries updated for Phase 37 MCP slice progress.
6. Validation:
   - `go test ./internal/mcp ./cmd/api -count=1`
   - `make test-unit`
   - CodeScene scores on touched Go files: `10.0`.

### 2026-03-03 (Phase 37 extension: consolidation suggestion schema + repository baseline)

1. Added consolidation suggestion persistence baseline in `db/init/001_schema.sql`:
   - new table `engram_consolidation_suggestions`.
   - new indexes:
     - `engram_consolidation_suggestions_hash_uidx`
     - `engram_consolidation_suggestions_project_status_idx`
     - `engram_consolidation_suggestions_source_gin_idx`
2. Added consolidation suggestion domain model in `internal/models/engram_consolidation.go`:
   - suggestion type and status enums.
   - status parser (`ParseConsolidationSuggestionStatus`).
   - persisted entity contract (`EngramConsolidationSuggestion`).
3. Added repository baseline in `internal/repository/engram_consolidation_suggestions.go`:
   - `RefreshExactDuplicateConsolidationSuggestions` for deterministic exact-duplicate grouping (`LOWER(TRIM(title))`) with configurable minimum group size.
   - conflict-safe upsert keyed by deterministic `consolidation_hash`.
   - confidence scoring tied to duplicate cluster size.
   - `ListEngramConsolidationSuggestions` with project/status filtering and pagination.
4. Added repository coverage in `internal/repository/engram_consolidation_suggestions_test.go`:
   - default normalization and query argument assertions.
   - invalid min-group guardrail coverage.
   - list filtering and status parsing assertions.
5. Documentation/state alignment:
   - `plan.md`: Phase 37 delivered-scope updated to include consolidation schema/repository baseline.
   - `migration/checkpoints/checkpoint.md`: Phase 37 tracker and summary updated for consolidation baseline progress.
6. Validation:
   - `go test ./internal/models ./internal/repository -count=1`
   - `make lint`
   - `make test-unit`
   - CodeScene scores on touched Go files: `10.0`.

### 2026-03-03 (Phase 37 extension: admin consolidation refresh/list service + REST routes)

1. Added admin-service contracts and methods for consolidation suggestions:
   - `RefreshEngramConsolidationSuggestions`
   - `ListEngramConsolidationSuggestions`
   - request/response contracts in `internal/admin/service.go` + implementation in `internal/admin/service_consolidation.go`.
2. Added memory-admin API routes:
   - `POST /api/v1/admin/memory/engrams/consolidation/refresh`
   - `GET /api/v1/admin/memory/engrams/consolidation/suggestions`
3. Added API parsing and error handling support:
   - status query parsing (`suggested|merged|rejected`) for consolidation listing.
   - explicit invalid min-group guardrail mapping to `400`.
4. Added regression coverage:
   - `internal/admin/service_test.go` consolidation refresh/list forwarding and validation tests.
   - `internal/api/admin_memory_test.go` consolidation refresh/list route tests + invalid-status coverage.
5. Refactored admin memory route wiring into focused route files and shared helper file to preserve maintainability and restore full CodeScene quality gates.
6. Documentation/state alignment:
   - `docs/api-reference.md` now documents the two consolidation admin endpoints.
   - `plan.md` and `migration/checkpoints/checkpoint.md` updated for Phase 37 admin-API progress.
7. Validation:
   - `go test ./internal/admin ./internal/api -count=1`
   - `make lint`
   - `make test-unit`
   - CodeScene scores on touched Go files: `10.0`.

### 2026-03-03 (Phase 37 extension: MCP consolidation refresh/list parity tools)

1. Added MCP consolidation dispatch support:
   - `engram.refresh_consolidation` (alias `engram_refresh_consolidation`) for deterministic exact-duplicate suggestion refresh (admin-only).
   - `engram.consolidation_list` (alias `engram_consolidation_list`) for consolidation suggestion listing with optional `project_id`/`status` filters (admin-only).
2. Added compatibility service contracts and runtime wiring:
   - new request/response interfaces/types in `internal/mcp/compatibility_service.go`.
   - dispatch handlers in `internal/mcp/compatibility_dispatch_engram_consolidation_support.go`.
   - route registration in read/mutation handler registries.
3. Added admin-to-MCP adapters:
   - `cmd/api/mcp_engram_admin_consolidation_adapter.go`.
   - dependency wiring in `cmd/api/main.go` via extracted compatibility dependency builders.
4. Added token-policy + tool-catalog updates:
   - read/write tool classification and ordering in `internal/mcp/catalog.go`.
   - input schemas/metadata in `internal/mcp/catalog_metadata_data.go`.
   - optional-project token normalization support in `internal/mcp/token_authorization_policy.go`.
5. Added regression coverage:
   - `internal/mcp/compatibility_service_engram_refresh_consolidation_test.go`.
   - `internal/mcp/compatibility_service_engram_consolidation_list_test.go`.
6. Documentation/state alignment:
   - `docs/mcp-guide.md` tool catalog updated with consolidation MCP tools.
   - `plan.md` and `migration/checkpoints/checkpoint.md` updated for Phase 37 MCP parity progress.
7. Validation:
   - `go test ./internal/mcp ./cmd/api -count=1`
   - `make lint`
   - `make test-unit`
   - CodeScene scores on touched Go files: `10.0`.

### 2026-03-03 (Phase 37 extension: consolidation action workflow baseline in repository/admin REST)

1. Added repository action path in `internal/repository/engram_consolidation_suggestions.go`:
   - `ApplyEngramConsolidationSuggestionAction` updates suggestion status to `merged`/`rejected`.
   - records `actioned_at`, `action_taken_by`, and `updated_at` atomically.
   - validates actionable status (`merged`/`rejected`) and returns `nil` on missing suggestion IDs.
2. Added admin service action workflow:
   - `ActionEngramConsolidationSuggestion` in `internal/admin/service_consolidation.go`.
   - added service-level not-found and invalid-action errors with explicit status validation.
3. Added admin memory REST route:
   - `POST /api/v1/admin/memory/engrams/consolidation/suggestions/{suggestion_id}/action`.
   - actor attribution forwarded from authenticated admin actor to service/repository.
4. Added regression coverage:
   - repository tests for action success, invalid status, and missing suggestion behavior.
   - admin service tests for repository forwarding, invalid status rejection, and not-found mapping.
   - API tests for actor/payload forwarding and invalid-status rejection.
5. Documentation/state alignment:
   - `docs/api-reference.md` now documents consolidation action endpoint.
   - `plan.md` and `migration/checkpoints/checkpoint.md` updated for Phase 37 action-baseline progress.
6. Validation:
   - `go test ./internal/repository ./internal/admin ./internal/api -count=1`
   - `make lint`
   - `make test-unit`
   - CodeScene scores on touched Go files: `10.0`.

### 2026-03-03 (Phase 37 extension: MCP consolidation action parity + project-scoped action guard)

1. Added MCP consolidation action parity tooling:
   - new admin-only tool `engram.consolidation_action` (alias `engram_consolidation_action`) to mark one suggestion as `merged` or `rejected`.
   - dispatch routing/wiring added across:
     - `internal/mcp/compatibility_dispatch_engram_mutation_handlers.go`
     - `internal/mcp/compatibility_dispatch_engram_consolidation_action_support.go`
     - `internal/mcp/compatibility_service.go`
     - `cmd/api/mcp_engram_admin_consolidation_adapter.go`
     - `cmd/api/main.go`
2. Added MCP tool catalog and token-policy alignment:
   - tool classification/order updates in `internal/mcp/catalog.go`.
   - input schema metadata in `internal/mcp/catalog_metadata_data.go`.
   - optional project-token normalization support in `internal/mcp/token_authorization_policy.go`.
3. Hardened consolidation action scoping across repository/admin/API:
   - `ApplyEngramConsolidationSuggestionAction` now supports optional project guard (`project_id`) for status updates.
   - admin action request object and REST payload now accept `project_id` and forward it to repository writes.
4. Added/updated regression coverage:
   - `internal/mcp/compatibility_service_engram_consolidation_action_test.go` for direct/tools parity, admin guard, validation, and internal/not-found mappings.
   - updated repository/admin/API tests for optional `project_id` forwarding and action-path behavior.
5. Documentation/state alignment:
   - `docs/mcp-guide.md` tool catalog updated with consolidation action tooling.
   - `Plan.md` and `migration/checkpoints/checkpoint.md` updated for narrowed remaining Phase 37 scope.
6. Validation:
   - `go test ./internal/repository ./internal/admin ./internal/api ./internal/mcp ./cmd/api -count=1`
   - `make lint`
   - `make test-unit`
   - CodeScene scores on touched Go files: `10.0` (with `internal/mcp/catalog_metadata_data.go` reported as non-scorable/null by CodeScene).

### 2026-03-03 (Phase 37 closeout: deterministic acceptance + precision/recall benchmark coverage)

1. Added deterministic Phase 37 acceptance scenario:
   - `acceptance-tests/features/phase37-consolidation-mock.feature`
   - `acceptance-tests/src/steps/phase37-consolidation-mock.steps.ts`
   - validates end-to-end admin consolidation flow in one isolated project:
     - seed duplicate/non-duplicate engrams.
     - refresh/list consolidation suggestions.
     - compute grouping precision/recall and gate at `>=0.95`.
     - action one suggestion (`merged`) and verify action fields via merged listing.
2. Added benchmark documentation:
   - `docs/phase37-consolidation-benchmark.md` captures fixture design, formulas, thresholds, and run commands.
3. Acceptance docs alignment:
   - `acceptance-tests/README.md` now documents the `@phase37` deterministic mock scenario and quality checks.
4. Roadmap/checkpoint alignment:
   - `Plan.md`: Phase 37 marked completed and delivered scope expanded with acceptance/benchmark coverage.
   - `migration/checkpoints/checkpoint.md`: Phase 37 summary/progress tracker marked completed.
5. Validation:
   - `cd acceptance-tests && npm run bdd:gen`
   - `cd acceptance-tests && npm run typecheck`
   - `make lint`
   - `make test-unit`

### 2026-03-01 (Security follow-up closeout: OIDC rollout validation + centralized audit sink regression)

1. Added explicit OIDC-to-sink integration coverage in `internal/api/session_ui_oidc_test.go`:
   - new test `TestMountSessionUIRoutesOIDCCallbackFailureEmitsAuditEventToSink`.
   - validates callback verification failures emit `oidc_login_failed` events to a centralized sink with bearer auth forwarding and expected failure detail (`token_exchange_or_verification_failed`).
2. Extended session UI test wiring in `internal/api/session_ui_test.go`:
   - `sessionUITestHandlerOptions` now supports an `auditLogger` override so OIDC/security tests can inject sink-backed audit loggers directly.
3. Realigned technical-debt tracking docs:
   - `todo.md`: marked production security follow-up (OIDC rollout validation + centralized audit sink integration) as completed.
   - `migration/checkpoints/checkpoint.md`: updated technical-debt closeout note to include centralized audit sink validation evidence.
4. Validation:
   - `go test ./internal/api -run 'TestMountSessionUIRoutesOIDCCallbackFailureEmitsAuditEventToSink|TestMountSessionUIRoutesOIDCCallbackFailurePaths|TestMountSessionUIRoutesOIDCCallbackRejectsReplayAfterPendingStateConsumed' -count=1`
   - `go test ./... -count=1`
   - `make eval`
   - CodeScene pre-commit safeguard (`quality_gates=passed`; MCP server update notice only).

### 2026-03-01 (Technical debt closeout: cross-provider engram reuse validation + docs realignment)

1. Expanded cross-provider fallback regression coverage in `internal/chat/service_test.go`:
   - fallback adapter stubs now capture provider requests so primary and fallback request payloads can be compared directly.
   - `TestSendMessageFallsBackToSecondaryProviderOnTransientFailure` now asserts:
     - primary and fallback requests reuse the same message history and assembled engram-context system prompt.
     - only provider/model changes (`primary-model` -> `fallback-model`), while response/persistence still carry the same `UsedEngramIDs`.
   - `TestStreamMessageEventsFallsBackAfterTransientProviderFailure` now asserts fallback stream provenance parity (`used_engram_ids`, `used_engram_link_ids`, `engram_trace_paths`, `retrieval_audit`) and assistant persistence metadata reuse.
2. Consolidated roadmap tracking docs to stay in sync with runtime state:
   - `todo.md`: marked cross-provider engram reuse validation as completed.
   - `migration/checkpoints/checkpoint.md`: updated Phase 32 timeline status to `Completed` and added a current-state note for cross-provider reuse validation coverage.
3. Validation:
   - `go test ./internal/chat -run 'TestSendMessageFallsBackToSecondaryProviderOnTransientFailure|TestStreamMessageEventsFallsBackAfterTransientProviderFailure|TestSendMessageReturnsUsedEngramIDsAndSources' -count=1`
   - `go test ./... -count=1`
   - `make eval`
   - CodeScene pre-commit safeguard (`quality_gates=passed`; MCP server update notice only).

### 2026-03-01 (Embeddings runtime upgrade: OpenAI provider with local fallback)

1. Added a real embedding provider path in `internal/embeddings/`:
   - introduced `OpenAIEmbeddingProvider` (`openai.go`) with `/v1/embeddings` calls, model/dimensions support, response ordering by index, and provider-error signaling for fallback handling.
   - introduced runtime embedding configuration (`runtime.go`) with process-wide default service, provider selection (`local` / `openai`), and optional local fallback behavior.
2. Wired startup provider configuration in `cmd/api/main.go`:
   - API boot now applies `EMBEDDING_PROVIDER`, `EMBEDDING_MODEL`, `EMBEDDING_FALLBACK_TO_LOCAL`, `OPENAI_API_KEY`, `OPENAI_BASE_URL`, and `EMBEDDING_TIMEOUT_SECONDS` to embedding runtime setup.
3. Updated repository write-path embedding defaults to runtime-configured providers:
   - `internal/repository/engram_write.go`
   - `internal/repository/document.go`
   - `internal/repository/admin_engram_update.go`
4. Added validation + tests:
   - `internal/config/config.go` now validates embedding provider and timeout semantics via `ValidateEmbeddingSettings`.
   - `internal/config/config_test.go` includes embedding-setting validation coverage.
   - `internal/embeddings/openai_test.go` and `internal/embeddings/runtime_test.go` validate provider behavior and runtime config rules.
5. Validation:
   - `go test ./internal/embeddings ./internal/config ./internal/repository ./cmd/api -count=1`
   - `go test ./... -count=1`
   - `make eval`
   - CodeScene pre-commit safeguard (`quality_gates=passed`; MCP server update notice only).

### 2026-03-01 (Security follow-up: OIDC callback replay abuse-path regression)

1. Added a replay-focused OIDC callback abuse-path test in `internal/api/session_ui_oidc_test.go`:
   - new test `TestMountSessionUIRoutesOIDCCallbackRejectsReplayAfterPendingStateConsumed`.
   - validates that once callback state is consumed into an authenticated session cookie, subsequent callback attempts with the consumed cookie are rejected (`403 invalid oidc state`).
   - asserts failed replay attempts emit `oidc_login_failed` audit logs with `state_mismatch`.
2. Validation:
   - `go test ./internal/api -count=1`
   - `go test ./... -count=1`
   - `make eval`
   - CodeScene pre-commit safeguard (`quality_gates=passed`).

### 2026-03-01 (Phase 32 closeout: retrieval-audit telemetry + cross-project usage signals)

1. Added retrieval-audit telemetry to chat context assembly in `internal/chat/context.go`:
   - introduced `ChatRetrievalAudit` with blocked candidate counts, trace suppression/filtering/truncation counts, and cross-project usage counters.
   - added `retrieval_audit` metadata to `AssembledChatContext` and populated it during engram context assembly.
   - included cross-project usage project IDs and per-trace/bundle cross-project counters based on packed bundles and filtered trace paths.
2. Propagated retrieval audit metadata through send/stream outputs:
   - `internal/chat/service.go`: `ChatSendResponse` now includes `retrieval_audit`.
   - `internal/chat/message_runtime.go`: stream `meta` and `done` payloads now include `retrieval_audit`.
   - `internal/mcp/compatibility_service.go` + `cmd/api/mcp_message_send_adapter.go`: MCP send-message responses now include `retrieval_audit`.
3. Added integration and propagation coverage:
   - `internal/chat/context_links_test.go`: verifies blocked linked-candidate backfill telemetry and cross-project usage telemetry.
   - `internal/chat/message_runtime_test.go` and `internal/chat/service_test.go`: verify `retrieval_audit` propagation in stream payloads and non-stream send responses.
4. Code health + quality-gate uplift:
   - refactored context and test fixture methods into smaller helpers to satisfy CodeScene guardrails (large-method/complexity thresholds).
5. Validation:
   - `go test ./internal/chat -count=1`
   - `go test ./... -count=1`
   - `make eval`
   - CodeScene pre-commit safeguard (`quality_gates=passed`; MCP server update notice only).

### 2026-03-01 (Phase 32 extension: federated ranking/context-packing fusion + rehydration backfill)

1. Tuned linked-context candidate selection in `internal/chat/context_links.go`:
   - expanded federated linked candidate pool sizing beyond strict final budget to keep fallback options available.
   - added fused semantic + trace scoring for candidate ranking (`rankEngramContextCandidates`) with depth-aware tie-breaking.
   - enforced linked-candidate coverage in the active top selection window when federated traces are present.
2. Updated engram context assembly in `internal/chat/context.go`:
   - switched to ranked candidate ordering driven by fused semantic + trace signals.
   - added resilient packing behavior so failed/filtered rehydration lookups backfill from lower-ranked candidates until budget is filled.
3. Added regression coverage in `internal/chat/context_links_test.go`:
   - verifies linked candidate coverage in top ranked window under high-seed-score pressure.
   - verifies fallback packing behavior when a high-ranked linked candidate cannot be rehydrated.
4. Validation:
   - `go test ./internal/chat -count=1`

### 2026-03-01 (Phase 32 extension: federated cross-project link visibility/traversal baseline)

1. Updated link repository SQL in `internal/repository/engram_links.go` to remove same-project-only traversal/read constraints:
   - removed source/target same-project join requirement from `CreateEngramLink` SQL path.
   - removed same-project gate from visible link select/traversal SQL builder.
   - removed same-project gate from link update SQL path while preserving source-project write authorization checks.
2. Access guardrails remain enforced per node:
   - source and target engram visibility checks (`buildMembershipReadClause`) are still required for every returned link.
   - source-project write checks (`buildMembershipWriteClause`) remain in mutation paths.
3. Added regression assertions in `internal/repository/engram_links_test.go`:
   - create/traversal SQL no longer embeds same-project-only predicates.
   - traversal remains depth-limited and cycle-safe.
4. Validation:
   - `go test ./internal/repository ./internal/chat ./internal/api ./internal/mcp ./cmd/api -count=1`
   - `go test ./... -count=1`
   - `make eval`
   - CodeScene pre-commit safeguard (`quality_gates=passed`; MCP server update notice only).

### 2026-03-01 (Phase 32 kickoff: `cw>` query protocol parser + runtime metadata baseline)

1. Added `cw>` parser module in `internal/chat/query_protocol.go`:
   - detects first-line `cw>` directives.
   - supports bare prefix default (`mode=auto`), intent form (`cw> retrieve`), and key-value directives (`mode=...`, `project=...`, `citations=required`).
   - returns normalized query content with directive lines stripped.
2. Integrated query protocol parsing into runtime preparation in `internal/chat/message_runtime.go`:
   - `PrepareGeneration` now parses `content_text`, stores normalized plan in `PreparedGeneration.CWPlanApplied`, and uses normalized content for both persisted user message content and context assembly query.
3. Added additive protocol metadata to outputs:
   - `internal/chat/service.go`: `ChatSendResponse` now includes optional `cw_plan_applied`.
   - `internal/chat/message_runtime.go`: stream `meta` and `done` payloads now include `cw_plan_applied`.
   - `internal/mcp/compatibility_service.go` + `cmd/api/mcp_message_send_adapter.go`: MCP send-message responses now include `cw_plan_applied`.
4. Added tests:
   - `internal/chat/query_protocol_test.go` for parser normalization/precedence behavior.
   - `internal/chat/message_runtime_test.go` for parser integration, sanitized user-query behavior, and stream payload metadata propagation.
5. Validation:
   - `go test ./internal/chat ./internal/mcp ./cmd/api -count=1`
   - `go test ./... -count=1`
   - `make eval`
   - CodeScene pre-commit safeguard (`quality_gates=passed`; MCP server update notice only).

### 2026-03-01 (Phase 31 closeout: AGENT/skills realignment + acceptance mock evidence refresh)

1. Completed Phase 31 documentation closeout and skills realignment:
   - updated `AGENT.md` with an explicit rule to keep active skills aligned to the Go runtime module layout (`internal/`, `cmd/api/`, `web/src/`) and remove deprecated `api/app`/`api/tests` references.
   - refreshed skill playbooks to current Go architecture and test surfaces:
     - `skills/domain-module-layout/SKILL.md`
     - `skills/chat-rag-operator/SKILL.md`
     - `skills/document-ingestion-rag/SKILL.md`
     - `skills/engram-lifecycle/SKILL.md`
     - `skills/engram-auto-metadata-enrichment/SKILL.md`
     - `skills/memory-lifecycle-policies/SKILL.md`
     - `skills/mcp-http-stream-tools/SKILL.md`
     - `skills/mcp-token-authz/SKILL.md`
2. Updated phase tracking and status alignment:
   - `Plan.md` now marks Phase 31 as completed and marks the docs/skills closeout deliverable as done.
   - `migration/checkpoints/checkpoint.md` now reflects current Phase 31 acceptance evidence and Go module references for implemented backend scope.
3. Refreshed acceptance mock validation evidence:
   - `make acceptance-bddgen` passed.
   - `make acceptance-typecheck` passed.
   - `make acceptance-test-mock` initially failed due missing Playwright browser binaries (`chromium_headless_shell`).
   - installed browsers with `npx playwright install` in `acceptance-tests/`, then reran `make acceptance-test-mock` successfully (`20 passed`).

### 2026-03-01 (Phase 20 closeout: OIDC abuse-path coverage + callback hardening)

1. Hardened OIDC callback state handling in `internal/api/session_auth.go`:
   - callbacks now consume pending OIDC state/nonce/next values once callback request validation succeeds.
   - this prevents reuse of pending callback state after provider exchange or identity-to-user mapping failures.
2. Extended OIDC failure auditing in `internal/api/session_auth.go`:
   - identity-to-user mapping failures now emit `oidc_login_failed` audit entries with explicit details:
     - `identity_missing_username`
     - `user_lookup_not_configured`
     - `user_lookup_error`
     - `user_not_authorized`
3. Expanded OIDC negative-path test coverage in `internal/api/session_ui_oidc_test.go`:
   - unsafe next-path sanitization test coverage for OIDC start.
   - callback request rejection coverage (`state_mismatch`, `missing_code`) with audit assertions.
   - provider failure and identity mapping failure coverage (missing username, unmapped user, inactive user) with audit assertions.
   - pending OIDC state-consumption assertions for validated callback attempts.
4. Test harness extension in `internal/api/session_ui_test.go`:
   - added optional lookup overrides in `sessionUITestHandlerOptions` for targeted OIDC authorization edge-case simulation.
5. Documentation realignment:
   - updated `Plan.md` and `migration/checkpoints/checkpoint.md` to mark Phase 20 completed and align remaining execution order.
6. Validation:
   - `go test ./internal/api -count=1`
   - `go test ./... -count=1`
   - `make eval`
   - CodeScene pre-commit safeguard (`quality_gates=passed`; MCP server update notice only).

### 2026-03-01 (Phase 20 extension: centralized audit sink rollout baseline)

1. Extended audit runtime for centralized sink delivery in `internal/audit/audit.go`:
   - added optional sink transport (`AUDIT_SINK_URL`) with HTTP/HTTPS POST of sanitized JSON events.
   - added optional bearer auth header support (`AUDIT_SINK_AUTH_TOKEN`).
   - added required/fail-open behavior control (`AUDIT_SINK_REQUIRED`).
   - added sink timeout control (`AUDIT_SINK_TIMEOUT_SECONDS`).
2. Refactored audit logging call contract:
   - `LogRequestEvent` now accepts typed `audit.RequestEvent` input to reduce primitive-heavy argument usage and improve code health.
   - updated session auth/runtime audit wiring in `cmd/api/main.go` and test helpers in `internal/api/session_ui_test.go`.
3. Added config validation and redaction coverage in `internal/config/config.go`:
   - `ValidateAuditSettings` now validates required sink URL semantics, absolute HTTP/HTTPS URL shape, and positive timeout.
   - `BuildDebugSettingsSnapshot` redacts `audit_sink_auth_token`.
4. Added/updated tests:
   - `internal/audit/audit_test.go` (sink success, fail-open behavior, required sink failure behavior).
   - `internal/config/config_test.go` (audit sink validation and redaction assertions).
   - `internal/api/session_ui_test.go` and `cmd/api` wiring tests updated for typed audit events.
5. Validation:
   - `go test ./internal/audit ./internal/config ./internal/api ./cmd/api -count=1`
   - `go test ./... -count=1`
   - `make eval`
   - CodeScene pre-commit safeguard (`quality_gates=passed`; MCP server update notice only).

### 2026-03-01 (Phase 28 closeout: configurable noisy-link suppression controls)

1. Added bounded noisy-link suppression controls in chat context assembly:
   - `internal/chat/context.go` now normalizes:
     - `link_noise_suppression_enabled` (default `true`)
     - `link_noise_score_threshold` (default `0.30`, bounded `0..1`)
   - `internal/chat/context_links.go` now filters low-score trace paths when suppression is enabled.
2. Added runtime policy defaults in API chat runtime wiring:
   - `cmd/api/chat_runtime.go` injects configured graph suppression defaults into context requests.
3. Added configuration controls and validation:
   - `internal/config/config.go` settings:
     - `GRAPH_LINK_NOISE_SUPPRESSION_ENABLED`
     - `GRAPH_LINK_NOISE_SCORE_THRESHOLD`
   - added bounded validation in `ValidateGraphSettings`.
4. Extended request and MCP contracts:
   - chat payload supports optional overrides:
     - `link_noise_suppression_enabled`
     - `link_noise_score_threshold`
   - MCP `chat.send_message` parsing, adapter forwarding, and catalog metadata updated with the same controls.
5. Added/updated tests:
   - `internal/chat/context_test.go` (suppression filter behavior and override policy).
   - `internal/chat/message_runtime_test.go` (forwarding of noisy-link controls).
   - `cmd/api/chat_runtime_test.go` (runtime default injection behavior).
   - `internal/api/chat_api_messages_test.go` (REST payload forwarding).
   - `internal/mcp/compatibility_service_chat_send_message_test.go`
   - `internal/mcp/compatibility_service_chat_send_message_stream_test.go`
   - `internal/config/config_test.go` (graph setting bounds validation).
6. Validation:
   - `go test ./internal/chat ./internal/config ./internal/mcp ./internal/api ./cmd/api -count=1`
   - `go test ./... -count=1`
   - `make eval`
   - CodeScene pre-commit safeguard (`quality_gates=passed`; MCP server update notice only).

### 2026-03-01 (Phase 28 extension: scheduled hygiene execution + auto-archival baseline)

1. Extended successful link reinforcement runtime in `cmd/api/chat_link_reinforcement.go`:
   - added per-source hygiene cadence tracker (`linkHygieneRunTracker`) with interval-based due checks.
   - added scheduled hygiene executor that reuses `internal/graph` recommendations.
   - integrated automatic archival for recommendation-driven stale/low-value links (`archive_stale_low_value` action).
2. Kept hygiene execution non-blocking for chat completion semantics:
   - reinforcement and scheduled hygiene continue to run after successful assistant persistence.
   - failures still surface through existing reinforcement lifecycle tracing/error path without breaking persistence logic.
3. Added unit coverage for scheduled hygiene behavior in `cmd/api/chat_link_reinforcement_test.go`:
   - cadence interval gating
   - recommendation-action filtering + dedupe for auto-archive candidates
   - due/not-due execution cycle assertions.
4. Validation:
   - `go test ./cmd/api ./internal/chat -count=1`
   - `go test ./... -count=1`
   - `make eval`
   - CodeScene pre-commit safeguard (`quality_gates=passed`; MCP server update notice only).

### 2026-03-01 (Phase 28 extension: graph-trace EvalOps coverage + docs realignment)

1. Expanded EvalOps suite contracts in `internal/evalops/types.go`:
   - added dimension `graph_trace`.
   - added `EvalCase` graph-trace fields:
     - `used_engram_link_ids`
     - `trace_target_ids`
     - `min_used_engram_link_count`
     - `required_trace_targets`
2. Expanded deterministic fixtures in `internal/evalops/fixtures.go`:
   - refactored fixture groups into helper constructors.
   - increased baseline fixture set from 6 to 8 cases by adding 2 `graph_trace` cases.
3. Added graph-trace checks to evaluator in `internal/evalops/runner.go`:
   - `used_engram_link_count` minimum-count check.
   - `required_trace_targets` presence check.
   - shared helper refactor for required-value matching to keep code-health clean.
4. Updated eval regression expectations:
   - `internal/evalops/runner_test.go` now expects 8 cases and 4 dimensions.
   - `evals/baselines/eval-suite-v1.json` updated with `graph_trace` dimension summary and 8-case totals.
5. Documentation cleanup and alignment:
   - updated `Plan.md` and `migration/checkpoints/checkpoint.md` for Phase 28 progress and remaining scope.
   - updated `docs/evalops-governance-v1.md` and `docs/testing-guide.md` with `graph_trace` dimension.
6. Validation:
   - `go test ./internal/evalops ./cmd/evalops -count=1`
   - `go test ./... -count=1`
   - `make eval` (8/8, delta gate pass)
   - CodeScene pre-commit safeguard (`quality_gates=passed`; MCP server update notice only).

### 2026-03-01 (Phase 28 extension: graph hygiene recommendation route)

1. Added graph hygiene models and service:
   - `internal/models/engram_link_hygiene.go` defines recommendation categories/shape.
   - `internal/graph/link_hygiene.go` detects:
     - duplicate target links
     - conflicting relation types
     - stale low-value link candidates
2. Added authenticated REST route:
   - `POST /api/v1/engrams/{engram_id}/links/hygiene`
   - wired through `internal/api/session_auth.go` and `internal/api/session_engrams_links.go`.
3. Added runtime dependency wiring:
   - `cmd/api/main.go` now provides `HygieneEngramLinks` via `hygieneEngramLinksDependency(...)`.
4. Added/updated tests:
   - `internal/graph/link_hygiene_test.go`
   - `internal/api/session_engrams_links_test.go` (hygiene route contract)
   - `internal/api/session_engrams_test.go` (test harness wiring update)
5. Validation:
   - `go test ./internal/graph ./internal/api ./cmd/api -count=1`
   - `go test ./... -count=1`
   - CodeScene pre-commit safeguard (`quality_gates=passed`).

### 2026-03-01 (Phase 28 kickoff: temporal decay + reinforcement in graph recall)

1. Added temporal weighting helpers in `internal/chat/link_temporal.go`:
   - decayed temporal-weight computation with half-life decay.
   - reinforcement boost computation for links reused in successful sessions.
   - shared recency scoring helper for link freshness.
2. Updated graph-aware context scoring in `internal/chat/context_links.go`:
   - link quality now uses decayed temporal weight instead of raw persisted value.
3. Added successful-session reinforcement wiring in `internal/chat/service.go`:
   - send/stream paths now invoke reinforcement hook after successful assistant persistence.
   - reinforcement failures are non-blocking and traced via lifecycle stage `link_reinforce_failed`.
4. Added runtime DB dependency wiring in `cmd/api/chat_link_reinforcement.go` + `cmd/api/chat_runtime.go`:
   - dedupe used link IDs, load links, skip archived/rejected links.
   - update `temporal_weight`, `last_reinforced_at`, and promote `suggested` links to `active` on reinforcement.
5. Added/updated tests:
   - `internal/chat/link_temporal_test.go`
   - `internal/chat/service_test.go`
   - `cmd/api/chat_link_reinforcement_test.go`
6. Validation:
   - `go test ./internal/chat ./cmd/api -count=1`
   - `go test ./... -count=1`
   - CodeScene pre-commit safeguard (`quality_gates=passed`).

### 2026-03-01 (Phase 27 closeout: linked-memory panel + suggestion workflow)

1. Added dedicated linked-memory panel in `web/src/components/LinkedEngramPanel.tsx`:
   - relation-aware edge cards with weight/confidence and freshness-age display.
   - explainability summary chips for seed engrams, trace paths, linked edges, and citations.
2. Added suggestion queue UX:
   - accept/reject actions on scored candidates.
   - pending-action disabling and deterministic source-target keying.
3. Added typed engram-link API client module in `web/src/api/engramLinks.ts`:
   - `createEngramLink`
   - `listEngramLinks`
   - `suggestEngramLinks`
4. Added linked-insight orchestration hook in `web/src/hooks/useLinkedEngramInsights.ts`:
   - source engram derivation from trace roots + used engram ids.
   - bounded per-source fetch, dedupe, recency/weight ordering, and used-link prioritization.
   - accept/reject mutation bridge with panel refresh and notice/error reporting.
5. Wired app integration in `web/src/App.tsx`:
   - added linked-insight panel in right rail.
   - propagated `used_engram_ids` state through stream metadata handling to insight loaders.
6. Added/updated tests:
   - `web/src/components/LinkedEngramPanel.test.tsx`
   - `web/src/hooks/useLinkedEngramInsights.test.ts`
   - `web/src/api/engramLinks.test.ts`
7. Validation:
   - `make web-check`
   - CodeScene pre-commit safeguard (`quality_gates=passed`).

### 2026-03-01 (Phase 27 UX baseline: linked traceability + recall controls in web chat)

1. Extended web chat API/client contracts for linked trace metadata:
   - added `EngramTracePath` type.
   - added `used_engram_link_ids` + `engram_trace_paths` support in send and stream payloads.
   - added optional per-message recall options (`link_recall_enabled`, `link_recall_depth`, `link_recall_max_neighbors`) in web chat send/stream helpers.
2. Updated prompt-stream hook wiring in `web/src/hooks/useChatActions.ts`:
   - propagate recall options on send.
   - capture/reset linked trace metadata from stream `meta` and `done` events.
3. Added chat UI explainability baseline in `web/src/components/ChatPanel.tsx`:
   - chat-composer controls for linked recall enable/depth/max-neighbors.
   - "Linked trace paths used" transcript strip with compact path rendering and score/depth hints.
4. Updated `web/src/App.tsx` orchestration:
   - session-scoped linked trace state reset on session change.
   - pass recall controls and trace metadata through `ChatPanel` and `usePromptActions`.
5. Added/updated tests:
   - `web/src/components/ChatPanel.test.tsx` (trace strip + recall control behavior).
   - `web/src/hooks/useChatActions.test.ts`
   - `web/src/api/chat.test.ts`
6. Validation:
   - `make web-check`
   - CodeScene pre-commit safeguard (`quality_gates=passed`).

### 2026-03-01 (Phase 26 backend baseline: graph-aware context assembly + trace metadata)

1. Upgraded chat context assembly with linked-memory recall:
   - added depth-bounded traversal integration into `internal/chat/context.go`.
   - default recall depth is `1` with bounded controls for depth and neighbor expansion.
2. Added configurable recall controls to chat send payloads:
   - `link_recall_enabled`
   - `link_recall_depth`
   - `link_recall_max_neighbors`
   - wired through REST + MCP send/stream paths into chat context assembly.
3. Added link-trace metadata in chat outputs:
   - `used_engram_link_ids`
   - `engram_trace_paths`
   - emitted in send responses and stream `meta`/`done` events.
4. Added graph scoring/pruning policy:
   - fused seed relevance + link quality (`weight`, `confidence`, `temporal_weight`) + recency + depth penalty.
   - bounded context merge logic keeps prior `used_engram_ids` behavior backward compatible.
5. Refactored context dependency wiring for code-health:
   - extracted dependency builders into `internal/chat/context_dependencies.go`.
6. Added/updated tests:
   - `internal/chat/context_test.go`
   - `internal/chat/message_runtime_test.go`
   - `internal/chat/service_test.go`
   - `internal/api/chat_api_messages_test.go`
   - `internal/mcp/compatibility_service_chat_send_message_test.go`
   - `internal/mcp/compatibility_service_chat_send_message_stream_test.go`
7. Validation:
   - `go test ./... -count=1`
   - CodeScene pre-commit safeguard (`quality_gates=passed`).

### 2026-03-01 (Phase 25 closeout: link REST/MCP + suggestion pipeline)

1. Added session-auth link REST routes in `internal/api/session_engrams_links.go`:
   - create/list/update/archive links
   - suggestion and trace endpoints.
2. Added MCP link tools and dispatch wiring:
   - `engram.link_create`, `engram.link_list`, `engram.link_update`
   - `engram.link_archive`, `engram.link_suggest`, `engram.trace_path`
   - token project-scope policy support for link resources.
3. Added hybrid suggestion service in `internal/graph/link_suggestions.go`:
   - semantic overlap
   - source overlap
   - lexical continuity hints
   - recency-aware scoring.
4. Added runtime wiring adapters in `cmd/api` for REST + MCP link services.
5. Added/updated tests:
   - `internal/api/session_engrams_links_test.go`
   - `internal/graph/link_suggestions_test.go`
   - `internal/mcp/compatibility_service_engram_link_test.go`
   - `internal/mcp/compatibility_service_engram_link_token_scope_test.go`
   - `internal/repository/engram_links_test.go`
6. Validation:
   - `go test ./internal/repository ./internal/graph ./internal/api ./internal/mcp ./cmd/api -count=1`
   - CodeScene branch safeguard clean after refactors.

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

### 2026-03-01 (Code health uplift + ContinuWitty target-state guide)

1. Prioritized code-health uplift to `10.0` for active chat service surface:
   - refactored `internal/chat/service.go` into focused helper modules:
     - `internal/chat/service_stream_helpers.go`
     - `internal/chat/service_generation_helpers.go`
     - `internal/chat/service_fallback_helpers.go`
     - `internal/chat/service_observability_helpers.go`
   - kept behavior stable while reducing complexity/duplication and argument-smell hotspots.
2. Added test-side code-health uplift:
   - `internal/chat/service_test.go` now uses trace expectation helpers for reduced conditional complexity while preserving assertions.
3. Added high-priority strategic documentation:
   - new `docs/continuwitty-target-state.md` describing full-phase end-state value, workflows, role-specific benefits, KPI model, and rollout guidance once the roadmap is complete.
4. Validation executed:
   - `go test ./internal/chat`
   - `go test ./...`
   - `code_health_review` confirmed `10.0` for:
     - `internal/chat/service.go`
     - `internal/chat/service_stream_helpers.go`
     - `internal/chat/service_generation_helpers.go`
     - `internal/chat/service_fallback_helpers.go`
     - `internal/chat/service_observability_helpers.go`
     - `internal/chat/service_test.go`
   - `pre_commit_code_health_safeguard(git_repository_path=/Users/mukundhan/Projects/engram)` => `quality_gates=passed`.

### 2026-03-01 (Phase 35 kickoff: memory engagement tracking baseline)

1. Roadmap alignment and docs consolidation:
   - updated `Plan.md` with a canonical "Memory Improvements Alignment" section mapping `docs/memory-improvements.md` to Phase 35-40 execution.
   - added explicit Phase 35/36/37 sections and near-term execution order updates to avoid roadmap drift.
2. Schema baseline for engagement telemetry:
   - `db/init/001_schema.sql`
   - added engram counters: `access_count`, `last_accessed_at`, `useful_count`, `contradiction_count`.
   - added `engram_access_events` table and indexes for engram/session/source + recency access patterns.
3. Repository implementation:
   - added `internal/repository/engram_access.go` with `RecordEngramAccessEvents`.
   - write path appends access events and updates engram aggregate counters in one SQL flow per event.
4. Chat runtime integration:
   - `internal/chat/service.go` now records engram access usage for successful send and stream flows.
   - access recording failures are explicitly non-blocking and captured as lifecycle trace signals.
   - refactored stream-success handling into helper methods to keep code-health gates green.
5. Tests added/updated:
   - `internal/repository/engram_access_test.go`
   - `internal/chat/service_test.go`
   - coverage includes normalization defaults, dedupe behavior, and side-effect failure isolation.
6. Verification:
   - `go test ./...` passed.
   - CodeScene pre-commit safeguard passed (`quality_gates=passed`).
   - CodeScene reported a tooling notice: MCP server update available (`MCP-0.2.1`).

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

1. Phase 20: extend OIDC/provider abuse-path and negative acceptance coverage.
2. Phase 32 prep: Phase 24-28 graph baseline is complete; begin cross-project federated query protocol rollout.

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
