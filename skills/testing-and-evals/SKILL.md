---
name: testing-and-evals
description: Use this skill when adding tests, extending eval scenarios, and validating memory quality regressions.
---

# Testing and Evals

## Use This Skill When
- Adding new behavior in API, repositories, providers, MCP, or UI.
- Extending memory quality and continuity validation.

## Test Strategy
- Unit tests for deterministic logic.
- Integration tests for API + DB + auth/visibility behavior.
- Ingestion tests for text/file intake, chunk persistence, and blended retrieval responses.
- Lifecycle-policy tests for autosave cadence, retention pruning, and timeline event behavior.
- Project-default contract tests for write flows that omit `project_id` (REST + MCP), including `resolved_project_id`/`used_default_project` assertions.
- Soft-delete contract tests for session/engram/collection delete + restore paths and default list filtering (`include_deleted` behavior).
- Collection-boundary tests for add/remove/update/delete behavior and cross-project engram move auto-detach semantics.
- Eval harness tests for recall, temporal correctness, abstention.
- Add chat continuity eval scenarios when chat logic changes.
- For MCP contract changes, include:
  - transport + JSON-RPC envelope integration coverage (`test_mcp_api_integration.py`)
  - typed client parser/transport coverage (Python + TypeScript helper tests).

## Required Commands
- `make lint`
- `make test`
- `make eval`
- `make check`
- `make acceptance-test-mock` when admin memory, MCP organization tools, or export/import memory management flows change.

## Regression Rule
Every bug fix gets at least one test that fails before and passes after the fix.
