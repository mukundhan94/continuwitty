import { createGlobalStyle } from 'styled-components'

export const GlobalStyle = createGlobalStyle`
  :root {
    --color-bg-soft: ${({ theme }) => theme.colors.bgSoft};
    --color-bg-strong: ${({ theme }) => theme.colors.bgStrong};
    --color-ink: ${({ theme }) => theme.colors.ink};
    --color-ink-muted: ${({ theme }) => theme.colors.inkMuted};
    --color-accent: ${({ theme }) => theme.colors.accent};
    --color-accent-alt: ${({ theme }) => theme.colors.accentAlt};
    --color-card: ${({ theme }) => theme.colors.card};
    --color-line: ${({ theme }) => theme.colors.line};
    --color-success: ${({ theme }) => theme.colors.success};
    --color-error: ${({ theme }) => theme.colors.error};
    --font-display: ${({ theme }) => theme.fonts.display};
    --font-body: ${({ theme }) => theme.fonts.body};
    --font-mono: ${({ theme }) => theme.fonts.mono};
    --shadow-panel: ${({ theme }) => theme.shadows.panel};
    --shadow-nav: ${({ theme }) => theme.shadows.nav};
    --shadow-card: ${({ theme }) => theme.shadows.card};
    --shadow-modal: ${({ theme }) => theme.shadows.modal};
  }

  * {
    box-sizing: border-box;
  }

  html,
  body,
  #root {
    min-height: 100%;
  }

  body {
    margin: 0;
    min-width: 320px;
    color: var(--color-ink);
    font-family: var(--font-body);
    text-rendering: optimizeLegibility;
    -webkit-font-smoothing: antialiased;
    -moz-osx-font-smoothing: grayscale;
    background:
      radial-gradient(circle at 16% 14%, #ffd8bf 0%, transparent 32%),
      radial-gradient(circle at 84% 82%, #b4ece3 0%, transparent 35%),
      linear-gradient(165deg, var(--color-bg-soft), var(--color-bg-strong));
  }

  button,
  input,
  textarea,
  select {
    font: inherit;
  }

  button {
    border: 1px solid transparent;
    border-radius: ${({ theme }) => theme.radius.md};
    background: var(--color-accent);
    color: #ffffff;
    cursor: pointer;
    padding: 0.5rem 0.8rem;
    display: inline-flex;
    align-items: center;
    justify-content: center;
    line-height: 1.2;
    transition:
      transform 160ms ease,
      filter 160ms ease,
      opacity 160ms ease;
  }

  button:hover {
    transform: translateY(-1px);
    filter: brightness(1.05);
  }

  button:disabled {
    cursor: not-allowed;
    opacity: 0.55;
    transform: none;
    filter: none;
  }

  input,
  textarea,
  select {
    width: 100%;
    border: 1px solid var(--color-line);
    border-radius: ${({ theme }) => theme.radius.md};
    padding: 0.55rem 0.7rem;
    background: #ffffff;
    color: var(--color-ink);
    line-height: 1.35;
  }

  textarea {
    min-height: 6rem;
  }

  label {
    font-size: 0.82rem;
    font-weight: 700;
    letter-spacing: 0.03em;
    text-transform: uppercase;
    color: var(--color-ink-muted);
  }

  h1,
  h2,
  h3,
  p {
    margin: 0;
  }

  a {
    color: inherit;
  }
`
