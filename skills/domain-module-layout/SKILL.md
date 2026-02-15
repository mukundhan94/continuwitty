---
name: domain-module-layout
description: Use this skill when introducing new domains to keep modules, packages, and tests organized for long-term readability and maintainability.
---

# Domain Module Layout

## Use This Skill When
- Adding a new functional domain (chat, providers, MCP, retrieval, auth extensions).
- Refactoring large files into maintainable packages.

## Layout Convention
- `api/app/<domain>_repository.py`: DB access and persistence.
- `api/app/<domain>_service.py`: orchestration/business logic when non-trivial.
- `api/app/<domain>_api.py` or route group file: transport handlers when domain grows.
- `api/app/providers/<provider>_provider.py`: provider-specific integrations.
- `api/tests/test_<domain>*.py`: domain-aligned tests.

## Rules
- No raw SQL in route handlers.
- No provider SDK calls outside provider modules.
- Keep shared contract types explicit and documented.
- Update README file map when adding modules.

## Validation
- Add or update tests for every new domain module.
- Run `make check` before phase commit.
