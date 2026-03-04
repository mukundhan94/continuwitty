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
    --surface-glass: ${({ theme }) => theme.colors.surfaceGlass};
    --surface-glass-border: ${({ theme }) => theme.colors.surfaceGlassBorder};
    --surface-raised: ${({ theme }) => theme.colors.surfaceRaised};
    --surface-raised-border: ${({ theme }) => theme.colors.surfaceRaisedBorder};
    --surface-mute: ${({ theme }) => theme.colors.surfaceMute};
    --color-input-bg: ${({ theme }) => theme.colors.inputBg};
    --color-notice-bg: ${({ theme }) => theme.colors.noticeBg};
    --color-notice-border: ${({ theme }) => theme.colors.noticeBorder};
    --bubble-user-start: ${({ theme }) => theme.colors.bubbleUserStart};
    --bubble-user-end: ${({ theme }) => theme.colors.bubbleUserEnd};
    --bubble-assistant-start: ${({ theme }) => theme.colors.bubbleAssistantStart};
    --bubble-assistant-end: ${({ theme }) => theme.colors.bubbleAssistantEnd};
    --markdown-code-bg: ${({ theme }) => theme.colors.markdownCodeBg};
    --markdown-blockquote-border: ${({ theme }) => theme.colors.markdownBlockquoteBorder};
    --markdown-blockquote-bg: ${({ theme }) => theme.colors.markdownBlockquoteBg};
    --markdown-table-border: ${({ theme }) => theme.colors.markdownTableBorder};
    --markdown-table-header-bg: ${({ theme }) => theme.colors.markdownTableHeaderBg};
    --modal-backdrop: ${({ theme }) => theme.colors.modalBackdrop};
    --modal-border: ${({ theme }) => theme.colors.modalBorder};
    --scrollbar-track: ${({ theme }) => theme.colors.scrollbarTrack};
    --scrollbar-thumb: ${({ theme }) => theme.colors.scrollbarThumb};
    --scrollbar-thumb-hover: ${({ theme }) => theme.colors.scrollbarThumbHover};
    --session-active-bg: ${({ theme }) => theme.colors.sessionActiveBg};
    --session-active-border: ${({ theme }) => theme.colors.sessionActiveBorder};
    --session-active-shadow: ${({ theme }) => theme.colors.sessionActiveShadow};
    --font-display: ${({ theme }) => theme.fonts.display};
    --font-body: ${({ theme }) => theme.fonts.body};
    --font-mono: ${({ theme }) => theme.fonts.mono};
    --shadow-panel: ${({ theme }) => theme.shadows.panel};
    --shadow-nav: ${({ theme }) => theme.shadows.nav};
    --shadow-card: ${({ theme }) => theme.shadows.card};
    --shadow-modal: ${({ theme }) => theme.shadows.modal};
    
    --cta-gradient: linear-gradient(135deg, var(--color-accent-alt), var(--color-accent), var(--session-active-border));
    --primary-gradient: linear-gradient(135deg, var(--color-accent-alt), var(--color-accent), var(--session-active-border));
  }

  * {
    box-sizing: border-box;
    scrollbar-width: thin;
    scrollbar-color: var(--scrollbar-thumb) var(--scrollbar-track);
  }

  *::-webkit-scrollbar {
    width: 11px;
    height: 11px;
  }

  *::-webkit-scrollbar-track {
    background: var(--scrollbar-track);
    border-radius: 9999px;
  }

  *::-webkit-scrollbar-thumb {
    background: var(--scrollbar-thumb);
    border-radius: 9999px;
    border: 2px solid var(--scrollbar-track);
  }

  *::-webkit-scrollbar-thumb:hover {
    background: var(--scrollbar-thumb-hover);
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
      radial-gradient(circle at 16% 14%, ${({ theme }) => theme.colors.bgGlowPrimary} 0%, transparent 32%),
      radial-gradient(circle at 84% 82%, ${({ theme }) => theme.colors.bgGlowSecondary} 0%, transparent 35%),
      linear-gradient(165deg, var(--color-bg-soft), var(--color-bg-strong));
    overflow-x: hidden;
  }

  button,
  input,
  textarea,
  select {
    font: inherit;
  }

  button {
    border: none;
    border-radius: 9999px;
    background: var(--cta-gradient);
    color: #ffffff;
    cursor: pointer;
    padding: 0.5rem 1rem;
    font-size: 0.85rem;
    font-weight: 700;
    display: inline-flex;
    align-items: center;
    justify-content: center;
    line-height: 1.2;
    box-shadow: 0 5px 14px var(--session-active-shadow);
    transition:
      transform 200ms ease-out,
      filter 200ms ease-out,
      box-shadow 200ms ease-out,
      opacity 200ms ease-out;
  }

  button:hover {
    transform: translateY(-1px);
    filter: brightness(1.04);
    box-shadow: 0 8px 20px var(--session-active-shadow);
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
    border: 1px solid transparent;
    border-radius: ${({ theme }) => theme.radius.md};
    padding: 0.55rem 0.7rem;
    background: var(--color-input-bg);
    color: var(--color-ink);
    line-height: 1.35;
    transition: border-color 200ms ease-out, box-shadow 200ms ease-out;
  }
  
  input:focus,
  textarea:focus,
  select:focus {
    outline: none;
    border-color: var(--color-accent);
    box-shadow: 0 0 0 2px var(--session-active-shadow);
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
  h4 {
    margin: 0;
    font-family: var(--font-display);
    font-weight: 700;
  }
  
  p {
    margin: 0;
  }

  a {
    color: inherit;
  }
`
