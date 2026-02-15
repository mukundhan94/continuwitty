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

## Regression Rule
Every bug fix gets at least one test that fails before and passes after the fix.
