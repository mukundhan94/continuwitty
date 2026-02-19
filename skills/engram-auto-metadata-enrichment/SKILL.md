---
name: engram-auto-metadata-enrichment
description: Use this skill when implementing fill-empty-only metadata enrichment for engram creation and MCP conversation persistence.
---

# Engram Auto Metadata Enrichment

## Use This Skill When
- Adding or modifying automatic abstract/tag/keyword generation.
- Wiring MCP conversation-only persistence (`engram.create_from_conversation`).
- Debugging why metadata was generated or skipped.

## Core Rules
1. Fill empty fields only (`abstract`, `tags`, `keywords`).
2. Never overwrite non-empty caller metadata.
3. Keep v1 deterministic and local-first (no provider call).
4. Enrichment failures must never block persistence.

## Module Boundaries
- `api/app/engram_enrichment/service.py`: deterministic parsing and enrichment decisions.
- `api/app/repository.py`: centralized invocation + `engram_json.auto_metadata` persistence.
- `api/app/mcp/service.py`: MCP tool contracts and enrichment report responses.

## Test Requirements
- Unit: `api/tests/test_engram_enrichment.py`
- API integration: `api/tests/test_api_integration.py`
- Chat integration: `api/tests/test_chat_api_integration.py`
- MCP integration: `api/tests/test_mcp_api_integration.py`
- Acceptance: `acceptance-tests/features/engram-auto-metadata-mock.feature`

## Future Hooks
- v2: optional LLM enrichment mode.
- v3: hybrid deterministic+LLM fallback with confidence thresholds.
