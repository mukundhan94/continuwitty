---
name: engram-lifecycle
description: Use this skill when implementing or operating engram creation, query, rehydration, ownership, visibility, and cross-session reuse flows.
---

# Engram Lifecycle

## Use This Skill When
- Adding or changing engram create/query/rehydrate behavior.
- Adding ownership/visibility rules for engrams.
- Building save-from-chat or pin-to-session workflows.

## Workflow
1. Confirm schema fields needed (`owner_user_id`, `visibility_scope`, `source_session_id`, soft-delete metadata).
2. Update models in `api/app/models.py`.
3. Update repository logic in `api/app/repository.py` or dedicated repository modules.
4. Route all write-time project fallback through `ProjectService.resolve_project_id_for_write`.
5. Add/update route handlers in `api/app/main.py` or domain routers.
6. Add tests in `api/tests/` for auth + visibility + behavior.
7. Update README with API and workflow examples.

## Rules
- `private` visibility is default.
- Use explicit checks for user ownership before returning engrams.
- Keep engram soft-delete/recovery additive (`deleted_at` + restore path), never hard-delete from admin flows.
- For create flows that can omit `project_id`, preserve response-level project resolution metadata (`resolved_project_id`, `used_default_project`) where defined.
- Keep collection/project integrity: if an engram is moved across projects, detach collection links that no longer match the engram project.
- Always return stable IDs in responses.
- Ensure rehydration includes compact context + citations.

## Validation
- Run `make lint`.
- Run `make test`.
- Run `make eval`.
