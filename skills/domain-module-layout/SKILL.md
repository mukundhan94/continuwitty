---
name: domain-module-layout
description: Use this skill when introducing new domains to keep Go modules, packages, and tests organized for long-term readability and maintainability.
---

# Domain Module Layout

## Use This Skill When
- Adding a new functional domain (chat, providers, MCP, retrieval, auth extensions).
- Refactoring large files into maintainable packages.

## Layout Convention (Go Runtime)
- `internal/<domain>/`: pure domain logic/helpers.
- `internal/repository/<domain>*.go`: DB access and persistence.
- `internal/api/<domain>*.go`: REST transport handlers and request/response contracts.
- `internal/mcp/compatibility_dispatch_<domain>*.go`: MCP tool routing/dispatch wiring.
- `cmd/api/<domain>*.go`: runtime dependency wiring and adapters.
- `internal/models/*.go`: shared contracts used across packages.

## Rules
- No raw SQL in route handlers or MCP dispatch modules.
- No provider SDK calls outside `internal/providers` and `internal/embeddings`.
- Keep shared contract types explicit and documented.
- Update `AGENT.md` and `Plan.md` when adding major new module boundaries.

## Validation
- Add or update tests for every new domain module.
- Run `go test ./... -count=1` and `make check` before commit.
- Run CodeScene pre-commit safeguard for the branch change set.
