# Engram Web Workbench

React + Vite frontend for chat sessions, streaming responses, and engram continuity workflows.
Styling stack: `styled-components` + Tailwind utility classes with shared theme tokens.

## Commands

```bash
npm install
npm run dev
npm run lint
npm run test
npm run build
```

## Local Runtime

- Expects backend API at `http://localhost:8000`.
- Vite dev server proxies:
  - `/api/*`
  - `/login`
  - `/logout`
  - `/ui`
- Optional frontend defaults via `.env` (copy from `.env.example`):
  - `VITE_DEFAULT_PROJECT_ID`
  - `VITE_DEFAULT_PROVIDER`
  - `VITE_DEFAULT_VISIBILITY`
  - `VITE_DEFAULT_OPENAI_MODEL`
  - `VITE_DEFAULT_ANTHROPIC_MODEL`
  - `VITE_DEFAULT_BEDROCK_MODEL`
- Shared theme and style modules:
  - `src/styles/theme.ts` (single source of tokens)
  - `src/styles/globalStyles.ts` (global CSS variables/base styles)
  - `src/styles/primitives.ts` (reusable styled UI shells/cards)
- Tailwind config:
  - `tailwind.config.ts`
  - `postcss.config.cjs`

On startup, the parsed frontend config is printed once in the browser console for debugging.

Run backend first (`make api`) and then start frontend (`make web-dev`).
