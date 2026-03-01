---
name: engram-auto-metadata-enrichment
description: Use this skill when implementing fill-empty-only metadata enrichment for engram creation and MCP conversation persistence.
---

# Engram Auto Metadata Enrichment

## Use This Skill When
- Adding or modifying deterministic abstract/tag/keyword generation.
- Wiring MCP conversation-only persistence (`engram.create_from_conversation`).
- Debugging why metadata was generated or skipped.

## Core Rules
1. Fill empty fields only (`abstract`, `tags`, `keywords`).
2. Never overwrite non-empty caller metadata.
3. Keep v1 deterministic and local-first (no provider calls).
4. Enrichment failures must never block persistence.

## Module Boundaries (Go Runtime)
- `internal/repository/engram_enrichment.go`: deterministic enrichment logic.
- `internal/repository/engram_write*.go`: centralized create-path invocation and persistence.
- `internal/mcp/compatibility_dispatch_engram_create_conversation_support.go`: MCP create-from-conversation contract surface.

## Test Requirements
- Unit/integration: `internal/repository/engram_enrichment_test.go`, `internal/repository/engram_write_test.go`.
- API/MCP integration: relevant `internal/api/*engram*_test.go` and `internal/mcp/compatibility_service_engram_create_conversation_test.go`.
- Acceptance mock: `acceptance-tests/features/engram-auto-metadata-mock.feature`.

## Future Hooks
- Optional model-assisted enrichment mode must remain explicitly opt-in.
- Any non-deterministic mode must preserve fill-empty-only semantics.
