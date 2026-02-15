---
name: frontend-style-system
description: Use this skill when changing web UI styling so all components stay on shared theme tokens and the styled-components + Tailwind architecture remains consistent.
---

# Frontend Style System

## Use This Skill When
- Refactoring visual styles in `web/src/components/`.
- Adding new pages, panels, cards, or modal surfaces in `web/`.
- Changing colors, spacing, typography, or elevation.

## Workflow
1. Define or update tokens in `web/src/styles/theme.ts` first.
2. Expose tokens globally through `web/src/styles/globalStyles.ts` CSS variables.
3. Reuse structural styled components from `web/src/styles/primitives.ts`.
4. Use Tailwind utility classes for local layout/spacing details.
5. Keep color and shadow literals out of feature components unless there is a clear one-off need.

## Quality Gates
- Run `make web-check` after style changes.
- If API or backend config changed in same phase, also run `make check`.
- Update `README.md` file map and implementation log for any new style modules.
