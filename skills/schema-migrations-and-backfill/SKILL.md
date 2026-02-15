---
name: schema-migrations-and-backfill
description: Use this skill when evolving SQL schema, adding indexes, and preserving compatibility for existing local data.
---

# Schema Migrations and Backfill

## Use This Skill When
- Adding/altering tables or indexes.
- Backfilling ownership or visibility fields.
- Introducing retrieval-side tables such as document/chunk stores.

## Workflow
1. Add forward-only idempotent SQL changes.
2. Add compatible defaults for existing rows.
3. Update repository queries for new constraints.
4. Add integration tests that cover old + new data assumptions.

## Safety Rules
- Avoid destructive DDL in active schema scripts.
- Use explicit defaults and check constraints.
- Add indexes for new list/query paths.

## Validation
- Local DB bootstraps without errors.
- Existing API tests remain green.
- New retrieval paths have explicit indexes (owner/project filters and vector search index).
