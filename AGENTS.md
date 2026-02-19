# Engram Vault - Agent & Skill Reference

## Operational Contract

Read `AGENT.md` for the full contribution and maintenance contract.
All non-negotiable rules, architecture boundaries, and workflow standards are defined there.

## Agents

### sentinel
- **File:** `agents/sentinel.md`
- **Invocable:** yes (`/sentinel [path-or-scope]`)
- **Description:** Full-codebase code health agent. Discovers hotspots, refactors
  every file below a score of 9.5 using CodeScene MCP tools, runs tests after
  every change, and commits each improvement atomically.
- **Depends on skill:** `skills/codescene/SKILL.md`

## Skills

All skills live under `skills/<name>/SKILL.md`. Read the relevant SKILL.md
before working in that domain.

| Skill | Description |
|-------|-------------|
| `skills/engram-lifecycle/SKILL.md` | Engram creation, query, rehydration, ownership/visibility. |
| `skills/chat-rag-operator/SKILL.md` | Chat session continuity and context assembly. |
| `skills/mcp-http-stream-tools/SKILL.md` | JSON-RPC over SSE tool contracts and handlers. |
| `skills/mcp-token-authz/SKILL.md` | MCP PAT lifecycle and scope/project authorization. |
| `skills/provider-openai/SKILL.md` | OpenAI adapter conventions. |
| `skills/provider-anthropic/SKILL.md` | Anthropic adapter conventions. |
| `skills/provider-bedrock/SKILL.md` | Bedrock adapter conventions. |
| `skills/schema-migrations-and-backfill/SKILL.md` | Forward-only schema evolution. |
| `skills/testing-and-evals/SKILL.md` | Test matrix and eval harness. |
| `skills/release-and-maintenance/SKILL.md` | Release checklist and maintenance cadence. |
| `skills/domain-module-layout/SKILL.md` | Module/package conventions. |
| `skills/react-chat-ui-operator/SKILL.md` | Frontend chat/session/engram UX workflow. |
| `skills/frontend-style-system/SKILL.md` | Token-driven styled-components + Tailwind rules. |
| `skills/dockerized-acceptance-testing/SKILL.md` | Playwright-BDD dockerized quality gate. |
| `skills/document-ingestion-rag/SKILL.md` | Document chunking, ingestion, blended retrieval. |
| `skills/memory-lifecycle-policies/SKILL.md` | Autosave, retention, consolidation, timeline events. |
| `skills/engram-auto-metadata-enrichment/SKILL.md` | Fill-empty metadata derivation rules. |
| `skills/codescene/SKILL.md` | Code health analysis using CodeScene MCP tools. |
