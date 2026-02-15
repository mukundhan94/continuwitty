---
name: react-chat-ui-operator
description: Use this skill when implementing or refactoring the React chat workbench, including session controls, streaming transcript UX, and engram pin/save/continue interactions.
---

# React Chat UI Operator

## Use This Skill When
- Building workflows in `web/src/`.
- Updating chat session or engram interaction UX.
- Extending frontend test coverage for chat behaviors.

## Workflow
1. Keep API calls in `web/src/api/` modules.
2. Keep UI composition in `web/src/components/` and `web/src/App.tsx`.
3. Keep SSE parsing in `web/src/utils/sse.ts` and reuse for streaming endpoints.
4. Validate with `make web-check`.
5. Update README web setup and module map when adding files.

## Testing Baseline
- Unit tests with Vitest + Testing Library for UI helpers/components.
- Keep end-to-end browser tests (Playwright) for final hardening phase.
