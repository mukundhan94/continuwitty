import { NavLink } from 'react-router-dom'
import styled, { keyframes } from 'styled-components'

import { MemoryStrandMark } from '../../../components/MemoryStrandMark'
import { MARKETING_ROUTES } from '../../constants'

const fadeUp = keyframes`
  from {
    opacity: 0;
    transform: translateY(24px);
  }
  to {
    opacity: 1;
    transform: translateY(0);
  }
`

const bob = keyframes`
  0%, 100% {
    transform: translateY(0);
  }
  50% {
    transform: translateY(8px);
  }
`

const waveShift = keyframes`
  from {
    transform: translateX(0);
  }
  to {
    transform: translateX(-50%);
  }
`

const drawLine = keyframes`
  to {
    stroke-dashoffset: 0;
  }
`

const Page = styled.div`
  display: grid;
  gap: 2rem;

  @media (prefers-reduced-motion: reduce) {
    *,
    *::before,
    *::after {
      animation: none !important;
      transition: none !important;
    }
  }
`

const Hero = styled.section`
  min-height: 74vh;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  text-align: center;
  padding: 1rem 0.8rem;
  position: relative;
`

const Wordmark = styled.h1`
  margin: 0.55rem 0 0;
  font-family: var(--font-display);
  font-weight: 700;
  font-size: clamp(2.4rem, 9vw, 5.2rem);
  letter-spacing: 0.02em;
  color: #8ec8c8;
  animation: ${fadeUp} 1.4s cubic-bezier(0.16, 1, 0.3, 1) 0.35s both;

  span {
    color: #5eead4;
  }
`

const Underline = styled.div`
  width: clamp(260px, 50vw, 520px);
  height: 14px;
  margin-top: 0.2rem;
  opacity: 0;
  animation: ${fadeUp} 1.2s ease 0.55s forwards;

  path {
    stroke-dasharray: 700;
    stroke-dashoffset: 700;
    animation: ${drawLine} 2s ease-out 0.8s forwards;
  }
`

const Tag = styled.p`
  margin: 0.9rem 0 0;
  font-weight: 300;
  font-size: clamp(0.76rem, 1.8vw, 1rem);
  color: rgba(94, 234, 212, 0.45);
  letter-spacing: 0.35em;
  text-transform: uppercase;
  animation: ${fadeUp} 1.2s ease 0.75s both;
`

const Sub = styled.p`
  margin: 0.65rem 0 0;
  max-width: 34rem;
  font-size: 0.9rem;
  color: rgba(178, 245, 234, 0.38);
  line-height: 1.75;
  font-style: italic;
  animation: ${fadeUp} 1.2s ease 0.95s both;
`

const Ctas = styled.div`
  display: flex;
  flex-wrap: wrap;
  justify-content: center;
  gap: 0.75rem;
  margin-top: 1.4rem;
  animation: ${fadeUp} 1.2s ease 1.1s both;
`

const PrimaryCTA = styled(NavLink)`
  text-decoration: none;
  border-radius: 999px;
  padding: 0.65rem 1.25rem;
  font-weight: 800;
  font-size: 0.8rem;
  letter-spacing: 0.04em;
  color: #001821;
  background: linear-gradient(135deg, #06b6d4, #0d9488, #5eead4);
  box-shadow: 0 4px 24px rgba(94, 234, 212, 0.2);
`

const GhostCTA = styled(NavLink)`
  text-decoration: none;
  border-radius: 999px;
  padding: 0.65rem 1.25rem;
  font-weight: 700;
  font-size: 0.8rem;
  letter-spacing: 0.04em;
  color: #cffff8;
  border: 1px solid rgba(94, 234, 212, 0.22);
  background: rgba(94, 234, 212, 0.08);
`

const ScrollHint = styled.div`
  position: absolute;
  bottom: 0.6rem;
  left: 50%;
  transform: translateX(-50%);
  font-family: var(--font-mono);
  font-size: 0.55rem;
  letter-spacing: 0.25em;
  text-transform: uppercase;
  color: rgba(94, 234, 212, 0.32);
  animation: ${fadeUp} 1s ease 1.4s both, ${bob} 2s ease-in-out infinite;
`

const Section = styled.section`
  border-radius: 24px;
  border: 1px solid rgba(94, 234, 212, 0.11);
  background: rgba(8, 20, 32, 0.6);
  padding: 1rem;
`

const SectionTitle = styled.h2`
  margin: 0;
  font-family: var(--font-display);
  font-size: clamp(1.35rem, 3.3vw, 2rem);
  color: #f7fffd;
`

const SectionLead = styled.p`
  margin: 0.55rem 0 0;
  max-width: 52rem;
  color: rgba(178, 245, 234, 0.76);
  line-height: 1.7;
`

const JourneyGrid = styled.div`
  margin-top: 0.95rem;
  display: grid;
  grid-template-columns: repeat(3, minmax(0, 1fr));
  gap: 0.7rem;

  @media (max-width: 960px) {
    grid-template-columns: 1fr;
  }
`

const JourneyCard = styled.article`
  border-radius: 18px;
  border: 1px solid rgba(94, 234, 212, 0.12);
  background: linear-gradient(160deg, rgba(8, 20, 32, 0.8), rgba(12, 53, 71, 0.5));
  padding: 0.85rem;
  position: relative;
  overflow: hidden;

  h3 {
    margin: 0;
    font-size: 0.95rem;
    color: #f2fffd;
  }

  p {
    margin: 0.38rem 0 0;
    color: rgba(201, 244, 236, 0.8);
    line-height: 1.58;
    font-size: 0.82rem;
  }
`

const CardWave = styled.div`
  position: absolute;
  inset: auto 0 0 0;
  height: 36px;
  pointer-events: none;
  opacity: 0.7;

  &::before,
  &::after {
    content: '';
    position: absolute;
    inset: 0;
    width: 200%;
    background-repeat: repeat-x;
    background-size: 96px 36px;
  }

  &::before {
    background-image: radial-gradient(48px 20px at 50% 120%, rgba(94, 234, 212, 0.22), transparent 72%);
    animation: ${waveShift} 9s linear infinite;
  }

  &::after {
    opacity: 0.9;
    background-image: radial-gradient(48px 20px at 50% 120%, rgba(6, 182, 212, 0.24), transparent 72%);
    animation: ${waveShift} 14s linear infinite reverse;
  }
`

const AgentStrip = styled.section`
  border-radius: 22px;
  border: 1px solid rgba(94, 234, 212, 0.15);
  background: rgba(8, 20, 32, 0.66);
  padding: 1rem;

  h2 {
    margin: 0;
    font-family: var(--font-display);
    font-size: 1.25rem;
  }

  p {
    margin: 0.5rem 0 0;
    color: rgba(201, 244, 236, 0.82);
    line-height: 1.65;
    max-width: 72ch;
  }
`

const AgentPoints = styled.div`
  margin-top: 0.72rem;
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 0.55rem;

  @media (max-width: 900px) {
    grid-template-columns: 1fr;
  }
`

const AgentPoint = styled.div`
  border-radius: 14px;
  border: 1px solid rgba(94, 234, 212, 0.12);
  background: rgba(12, 53, 71, 0.35);
  padding: 0.62rem;
  font-size: 0.82rem;
  color: rgba(215, 249, 243, 0.9);
  line-height: 1.58;
`

export function LandingPage() {
  return (
    <Page>
      <Hero>
        <MemoryStrandMark size="lg" />
        <Wordmark>
          Continu<span>Witty</span>
        </Wordmark>
        <Underline>
          <svg width="100%" height="14" viewBox="0 0 520 14" preserveAspectRatio="none">
            <path
              d="M1 8 C 110 2, 220 12, 340 6 C 410 3, 460 9, 519 5"
              stroke="rgba(94,234,212,0.45)"
              strokeWidth="2"
              fill="none"
              strokeLinecap="round"
            />
          </svg>
        </Underline>
        <Tag>Intelligence that flows</Tag>
        <Sub>
          Persistent memory for agents and teams that run long, high-context work. Capture decisions, keep
          provenance, and resume without context loss.
        </Sub>
        <Ctas>
          <PrimaryCTA to={MARKETING_ROUTES.login}>Start Flowing</PrimaryCTA>
          <GhostCTA to={MARKETING_ROUTES.howItWorks}>See How It Works</GhostCTA>
        </Ctas>
        <ScrollHint>Scroll to explore</ScrollHint>
      </Hero>

      <Section>
        <SectionTitle>Memory continuity, not prompt repetition.</SectionTitle>
        <SectionLead>
          ContinuWitty turns each session into a governed, searchable memory stream. Your next interaction starts
          with what matters instead of replaying old context.
        </SectionLead>
        <JourneyGrid>
          <JourneyCard>
            <h3>1. Capture</h3>
            <p>Persist key milestones from active chats as engrams with tags, summaries, and visibility controls.</p>
            <CardWave />
          </JourneyCard>
          <JourneyCard>
            <h3>2. Continue</h3>
            <p>Start the next session with pinned engrams and documents so the agent can continue immediately.</p>
            <CardWave />
          </JourneyCard>
          <JourneyCard>
            <h3>3. Govern</h3>
            <p>Use project scopes, token controls, and audit timelines to scale memory operations safely.</p>
            <CardWave />
          </JourneyCard>
        </JourneyGrid>
      </Section>

      <AgentStrip>
        <h2>Built for human teams and autonomous agents.</h2>
        <p>
          Agents cannot ship reliable output without stable memory. ContinuWitty gives them durable context,
          explainable provenance, and project boundaries so they can operate like long-term collaborators, not
          stateless tools.
        </p>
        <AgentPoints>
          <AgentPoint>
            Reduce repeated prompts and onboarding loops by preserving session intelligence as reusable memory.
          </AgentPoint>
          <AgentPoint>
            Trace answer quality through linked sources and timelines instead of opaque model recall.
          </AgentPoint>
          <AgentPoint>
            Give each workflow a scoped memory boundary with clear ownership and permission control.
          </AgentPoint>
          <AgentPoint>
            Keep retrieval grounded in project context so outputs stay aligned with current product reality.
          </AgentPoint>
        </AgentPoints>
      </AgentStrip>
    </Page>
  )
}
