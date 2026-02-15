---
name: provider-openai
description: Use this skill when adding or maintaining OpenAI provider adapter logic under the shared provider contract.
---

# Provider OpenAI

## Use This Skill When
- Implementing OpenAI chat generation/streaming in provider adapters.
- Mapping OpenAI response payloads into normalized app format.

## Adapter Contract
- Implement `generate(...)`.
- Implement `stream_generate(...)`.
- Implement `healthcheck()`.

## Rules
- Do not call OpenAI directly from routes/UI.
- Normalize errors to app-level provider exceptions.
- Keep model/provider configuration session-driven.

## Tests
- Unit tests for success path, stream parsing, and error mapping.
