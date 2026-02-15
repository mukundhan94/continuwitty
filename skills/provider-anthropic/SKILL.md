---
name: provider-anthropic
description: Use this skill when adding or maintaining Anthropic provider adapter logic under the shared provider contract.
---

# Provider Anthropic

## Use This Skill When
- Implementing Anthropic request/response normalization.
- Supporting streaming responses for Anthropic models.

## Adapter Contract
- `generate(...)`
- `stream_generate(...)`
- `healthcheck()`

## Rules
- Keep Anthropic-specific payload details inside adapter module.
- Return normalized token usage and text output shape.
- Map provider errors to app-specific error classes.

## Tests
- Adapter unit tests for normal, throttled, and invalid-auth cases.
