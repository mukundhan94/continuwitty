import styled, { css, keyframes } from 'styled-components'

const slideUp = keyframes`
  from {
    opacity: 0;
    transform: translateY(12px);
  }
  to {
    opacity: 1;
    transform: translateY(0);
  }
`

const fadeIn = keyframes`
  from {
    opacity: 0;
    transform: translateY(4px);
  }
  to {
    opacity: 1;
    transform: translateY(0);
  }
`

export const LoadingScreen = styled.div`
  min-height: 100vh;
  display: grid;
  place-items: center;
  color: var(--color-ink-muted);
  font-size: 1.05rem;
`

export const LoginShell = styled.div`
  min-height: 100vh;
  display: grid;
  place-items: center;
  padding: 1.5rem;
  background:
    radial-gradient(circle at 18% 16%, ${({ theme }) => theme.colors.bgGlowPrimary} 0%, transparent 43%),
    radial-gradient(circle at 82% 76%, ${({ theme }) => theme.colors.bgGlowSecondary} 0%, transparent 46%);
  position: relative;
  overflow: hidden;

  &::before,
  &::after {
    content: '';
    position: absolute;
    left: -10%;
    bottom: -16px;
    width: 220%;
    height: 210px;
    background-repeat: repeat-x;
    background-size: 240px 210px;
    pointer-events: none;
  }

  &::before {
    background-image: radial-gradient(120px 44px at 50% 114%, rgba(94, 234, 212, 0.08), transparent 74%);
    animation: waveDrift 42s linear infinite reverse;
    opacity: 0.28;
  }

  &::after {
    left: 0;
    background-image: radial-gradient(120px 44px at 50% 114%, rgba(6, 182, 212, 0.06), transparent 74%);
    animation: waveDrift 30s linear infinite;
    opacity: 0.22;
  }
  
  @keyframes waveDrift {
    0% { transform: translateX(0); }
    100% { transform: translateX(-50%); }
  }
`

export const LoginCard = styled.div`
  width: min(520px, 100%);
  background: var(--surface-glass);
  backdrop-filter: blur(10px);
  border: 1px solid var(--surface-glass-border);
  border-radius: ${({ theme }) => theme.radius.xl};
  box-shadow: var(--shadow-card);
  padding: 1.7rem;
  animation: ${slideUp} 400ms ease;
  z-index: 10;
`

export const AppShell = styled.div`
  min-height: 100vh;
  height: 100vh;
  padding: 0.65rem;
  display: flex;
  flex-direction: column;
  gap: 0.65rem;
  overflow: hidden;
`

export const TopNavShell = styled.header`
  border-radius: 20px;
  padding: 0.68rem 0.92rem;
  background: var(--surface-glass);
  backdrop-filter: blur(10px);
  border: 1px solid var(--surface-glass-border);
  box-shadow: var(--shadow-nav);
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 1rem;

  @media (max-width: 1180px) {
    flex-direction: column;
    align-items: flex-start;
  }
`

export const WorkspaceGrid = styled.main`
  display: grid;
  grid-template-columns: minmax(260px, 320px) 1fr minmax(280px, 340px);
  grid-template-rows: minmax(0, 1fr);
  gap: 0.9rem;
  flex: 1;
  min-height: 0;

  @media (max-width: 1180px) {
    grid-template-columns: 1fr;
    grid-template-rows: none;
    min-height: auto;
    overflow: auto;
  }
`

export const GlassPane = styled.section`
  background: var(--surface-glass);
  backdrop-filter: blur(10px);
  border: 1px solid var(--surface-glass-border);
  border-radius: 20px;
  box-shadow: var(--shadow-panel);
  min-height: 0;
  display: flex;
  flex-direction: column;
  gap: 0.7rem;
  padding: 1rem;
  overflow: hidden;

  @media (max-width: 1180px) {
    min-height: 320px;
  }
`

export const TopNavTitleBlock = styled.div`
  display: grid;
  gap: 0.2rem;
`

export const TopNavUserBlock = styled.div`
  display: flex;
  align-items: center;
  gap: 0.52rem;
  flex-wrap: wrap;
`

export const PaneHeader = styled.div`
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 0.5rem;
  min-height: 2rem;
  border-bottom: 1px dashed var(--color-line);
  padding-bottom: 0.42rem;
`

export const SectionDivider = styled.div`
  border-top: 1px dashed var(--color-line);
  padding-top: 0.55rem;
`

export const FormGrid = styled.form`
  display: grid;
  gap: 0.55rem;
  border-top: 1px dashed var(--color-line);
  border-bottom: 1px dashed var(--color-line);
  padding: 0.6rem 0;
`

export const SplitGrid = styled.div`
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 0.5rem;

  @media (max-width: 560px) {
    grid-template-columns: 1fr;
  }
`

export const ScrollColumn = styled.div`
  display: flex;
  flex-direction: column;
  gap: 0.45rem;
  overflow: auto;
  padding-right: 0.2rem;
`

export const EyebrowText = styled.p`
  text-transform: uppercase;
  letter-spacing: 0.08em;
  color: var(--color-ink-muted);
  font-size: 0.72rem;
`

export const MutedText = styled.p`
  color: var(--color-ink-muted);
  font-size: 0.84rem;
`

export const SupportText = styled.p`
  color: var(--color-ink-muted);
  margin-top: 0.45rem;
`

export const ErrorText = styled.p`
  color: var(--color-error);
  font-size: 0.86rem;
`

export const NoticeBanner = styled.p`
  padding: 0.55rem 0.8rem;
  border-radius: ${({ theme }) => theme.radius.md};
  background: var(--color-notice-bg);
  color: var(--color-success);
  border: 1px solid var(--color-notice-border);
`

export const SessionItemButton = styled.button<{ $active: boolean }>`
  width: 100%;
  text-align: left;
  background: var(--surface-raised);
  color: var(--color-ink);
  border: 1px solid var(--color-line);
  display: grid;
  gap: 0.12rem;
  padding: 0.5rem 0.6rem;
  border-radius: 12px;
  transition:
    background 200ms ease-out,
    border-color 200ms ease-out,
    box-shadow 200ms ease-out,
    transform 200ms ease-out;

  &:hover {
    transform: translateX(4px);
    background: var(--color-input-bg);
  }

  ${({ $active }) =>
    $active
      ? css`
          background: var(--session-active-bg);
          border-color: var(--session-active-border);
          box-shadow:
            inset 0 0 0 1px var(--session-active-border),
            0 0 0 1px var(--session-active-shadow);
        `
      : null}
`

export const SessionMeta = styled.span`
  font-family: var(--font-mono);
  font-size: 0.72rem;
  color: var(--color-ink-muted);
`

export const ChatMessageBubble = styled.article<{ $role: string }>`
  border-radius: 16px;
  padding: 0.65rem 0.85rem;
  max-width: 95%;
  animation: ${fadeIn} 200ms ease-out;
  box-shadow: 0 4px 12px rgba(0, 0, 0, 0.04);

  ${({ $role }) =>
    $role === 'user'
      ? css`
          align-self: flex-end;
          background: linear-gradient(140deg, var(--bubble-user-start), var(--bubble-user-end));
          border-bottom-right-radius: 8px;
        `
      : css`
          align-self: flex-start;
          background: linear-gradient(130deg, var(--bubble-assistant-start), var(--bubble-assistant-end));
          border-top-left-radius: 8px;
        `}
`

export const MessageRole = styled.p`
  font-size: 0.68rem;
  text-transform: uppercase;
  letter-spacing: 0.06em;
  color: var(--color-ink-muted);
  margin-bottom: 0.2rem;
`

export const MessageText = styled.div`
  line-height: 1.45;

  h1,
  h2,
  h3,
  h4 {
    font-family: var(--font-display);
    line-height: 1.22;
    margin: 0.55rem 0 0.35rem;
    letter-spacing: 0.01em;
    color: var(--color-ink);
  }

  h1 {
    font-size: 1.02rem;
  }

  h2 {
    font-size: 0.96rem;
  }

  h3,
  h4 {
    font-size: 0.9rem;
  }

  strong {
    color: var(--color-ink);
  }

  p,
  ul,
  ol,
  pre,
  blockquote,
  table,
  hr {
    margin: 0.35rem 0;
  }

  ul,
  ol {
    padding-left: 1.15rem;
  }

  li + li {
    margin-top: 0.2rem;
  }

  code {
    font-family: var(--font-mono);
    font-size: 0.83rem;
    background: var(--markdown-code-bg);
    border-radius: 8px;
    padding: 0.1rem 0.3rem;
  }

  pre {
    background: var(--markdown-code-bg);
    border-radius: 8px;
    padding: 0.55rem 0.65rem;
    overflow-x: auto;
  }

  pre code {
    background: transparent;
    padding: 0;
  }

  blockquote {
    border-left: 3px solid var(--markdown-blockquote-border);
    padding-left: 0.6rem;
    color: var(--color-ink-muted);
    background: var(--markdown-blockquote-bg);
    border-radius: 0 8px 8px 0;
  }

  a {
    color: var(--color-accent-alt);
    text-decoration: underline;
  }

  table {
    width: 100%;
    border-collapse: collapse;
    font-size: 0.84rem;
    border: 1px solid var(--markdown-table-border);
    background: var(--surface-raised);
  }

  th,
  td {
    border: 1px solid var(--markdown-table-border);
    padding: 0.32rem 0.4rem;
    text-align: left;
    vertical-align: top;
  }

  th {
    background: var(--markdown-table-header-bg);
    font-weight: 700;
  }

  hr {
    border: 0;
    border-top: 1px dashed var(--markdown-table-border);
  }
`

export const TranscriptRail = styled.div`
  flex: 1;
  overflow: auto;
  display: flex;
  flex-direction: column;
  gap: 0.55rem;
  padding-right: 0.25rem;
`

export const SourceStrip = styled.div`
  border-top: 1px dashed var(--color-line);
  padding-top: 0.5rem;
  font-size: 0.82rem;

  ul {
    padding-left: 1rem;
    margin: 0.4rem 0 0;
  }

  a {
    color: var(--color-accent-alt);
  }
`

export const SourceTitle = styled.p`
  font-weight: 700;
  color: var(--color-ink-muted);
`

export const EngramCard = styled.article`
  border: 1px solid var(--color-line);
  border-radius: 12px;
  padding: 0.65rem;
  background: var(--surface-raised);
  display: grid;
  gap: 0.35rem;
  transition: transform 200ms ease-out, box-shadow 200ms ease-out, border-color 200ms ease-out;

  &:hover {
    transform: translateY(-2px);
    box-shadow: 0 8px 24px rgba(13, 148, 136, 0.15);
    border-color: var(--color-accent);
  }
`

export const EngramTitle = styled.p`
  font-size: 0.88rem;
  font-weight: 700;
`

export const EngramAbstract = styled.p`
  font-size: 0.8rem;
  color: var(--color-ink-muted);
  line-height: 1.3;
`

export const ModalBackdrop = styled.div`
  position: fixed;
  inset: 0;
  background: var(--modal-backdrop);
  display: grid;
  place-items: center;
  padding: 1rem;
`

export const ModalCard = styled.div`
  width: min(560px, 100%);
  border-radius: ${({ theme }) => theme.radius.lg};
  background: var(--color-card);
  border: 1px solid var(--modal-border);
  box-shadow: var(--shadow-modal);
  padding: 1rem;
`
