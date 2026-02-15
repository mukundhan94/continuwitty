---
name: provider-bedrock
description: Use this skill when adding or maintaining AWS Bedrock provider adapter logic and request signing/config behavior.
---

# Provider Bedrock

## Use This Skill When
- Implementing Bedrock model invocation adapters.
- Handling AWS credentials/region config flow.

## Adapter Contract
- `generate(...)`
- `stream_generate(...)`
- `healthcheck()`

## Rules
- Keep AWS SDK wiring isolated in provider module.
- Validate region and credentials at startup/healthcheck.
- Normalize Bedrock response and errors to shared format.

## Tests
- Unit tests with mocked Bedrock client responses and failures.
