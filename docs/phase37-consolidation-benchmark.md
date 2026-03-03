# Phase 37 Consolidation Grouping Benchmark

This benchmark captures deterministic grouping quality for exact-duplicate consolidation suggestions.

## Scope

- API path under test:
  - `POST /api/v1/admin/memory/engrams/consolidation/refresh`
  - `GET /api/v1/admin/memory/engrams/consolidation/suggestions`
  - `POST /api/v1/admin/memory/engrams/consolidation/suggestions/{suggestion_id}/action`
- Grouping criteria under test:
  - exact duplicate title grouping via normalized title (`LOWER(TRIM(title))`)
  - minimum group size = `2`

## Deterministic Fixture

The acceptance fixture creates an isolated project and seeds:

- Duplicate Group A (3 engrams): case/whitespace variants of `Incident Response Playbook`
- Duplicate Group B (2 engrams): case variants of `Rollback Checklist`
- Non-duplicates (2 engrams): unrelated unique titles

Expected duplicate clusters: `2`

## Metrics

The acceptance scenario computes:

- `precision = true_positives / predicted_clusters`
- `recall = true_positives / expected_clusters`

Gate thresholds:

- precision `>= 0.95`
- recall `>= 0.95`

Current deterministic expectation for this fixture is `precision=1.0`, `recall=1.0`.

## Action Workflow Validation

After metric checks pass, the scenario action-tests one suggested cluster:

1. marks suggestion as `merged` with project scoping.
2. verifies merged listing contains the same `suggestion_id`.
3. verifies action audit fields (`actioned_at`, `action_taken_by`) are populated.

## Run

```bash
cd acceptance-tests
ACCEPTANCE_BDD_TAGS='@phase37' npm run test
```

Or through Make:

```bash
ACCEPTANCE_BDD_TAGS='@phase37' make acceptance-test
```
