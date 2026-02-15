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
`

export const LoginCard = styled.div`
  width: min(480px, 100%);
  background: rgba(255, 255, 255, 0.92);
  border: 1px solid rgba(255, 255, 255, 0.8);
  border-radius: ${({ theme }) => theme.radius.xl};
  box-shadow: var(--shadow-card);
  padding: 1.5rem;
  animation: ${slideUp} 400ms ease;
`

export const AppShell = styled.div`
  min-height: 100vh;
  height: 100vh;
  padding: 1.1rem;
  display: flex;
  flex-direction: column;
  gap: 0.9rem;
  overflow: hidden;
`

export const TopNavShell = styled.header`
  border-radius: 18px;
  padding: 1rem 1.1rem;
  background: rgba(255, 255, 255, 0.88);
  border: 1px solid rgba(255, 255, 255, 0.7);
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
  grid-template-columns: minmax(250px, 300px) 1fr minmax(260px, 330px);
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
  background: rgba(255, 255, 255, 0.9);
  border: 1px solid rgba(255, 255, 255, 0.75);
  border-radius: ${({ theme }) => theme.radius.lg};
  box-shadow: var(--shadow-panel);
  min-height: 0;
  display: flex;
  flex-direction: column;
  gap: 0.7rem;
  padding: 0.95rem;
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
  gap: 0.75rem;
  flex-wrap: wrap;
`

export const PaneHeader = styled.div`
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 0.5rem;
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
  padding: 0.5rem 0.75rem;
  border-radius: ${({ theme }) => theme.radius.md};
  background: #e0f2ee;
  color: var(--color-success);
  border: 1px solid #c4e6dc;
`

export const SessionItemButton = styled.button<{ $active: boolean }>`
  width: 100%;
  text-align: left;
  background: #ffffff;
  color: var(--color-ink);
  border: 1px solid var(--color-line);
  display: grid;
  gap: 0.12rem;
  padding: 0.5rem 0.6rem;

  ${({ $active }) =>
    $active
      ? css`
          border-color: var(--color-accent-alt);
          box-shadow: inset 0 0 0 1px var(--color-accent-alt);
        `
      : null}
`

export const SessionMeta = styled.span`
  font-family: var(--font-mono);
  font-size: 0.72rem;
  color: var(--color-ink-muted);
`

export const ChatMessageBubble = styled.article<{ $role: string }>`
  border-radius: 14px;
  padding: 0.6rem 0.75rem;
  max-width: 95%;
  animation: ${fadeIn} 190ms ease;

  ${({ $role }) =>
    $role === 'user'
      ? css`
          align-self: flex-end;
          background: linear-gradient(140deg, #fbe2cb, #f8c8aa);
        `
      : css`
          align-self: flex-start;
          background: linear-gradient(130deg, #d9f1ec, #bfe5dd);
        `}
`

export const MessageRole = styled.p`
  font-size: 0.68rem;
  text-transform: uppercase;
  letter-spacing: 0.06em;
  color: var(--color-ink-muted);
  margin-bottom: 0.2rem;
`

export const MessageText = styled.p`
  white-space: pre-wrap;
  line-height: 1.4;
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
  padding: 0.55rem;
  background: #ffffff;
  display: grid;
  gap: 0.35rem;
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
  background: rgba(18, 28, 32, 0.45);
  display: grid;
  place-items: center;
  padding: 1rem;
`

export const ModalCard = styled.div`
  width: min(560px, 100%);
  border-radius: ${({ theme }) => theme.radius.lg};
  background: var(--color-card);
  border: 1px solid #f0f0f0;
  box-shadow: var(--shadow-modal);
  padding: 1rem;
`
