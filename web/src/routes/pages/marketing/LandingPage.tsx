import { NavLink } from 'react-router-dom'
import styled, { keyframes } from 'styled-components'

import { MemoryStrandMark } from '../../../components/MemoryStrandMark'
import { MARKETING_ROUTES } from '../../constants'
import { AnimatedSvgDiagram } from './components/AnimatedSvgDiagram'
import { ChatIcon, FlowIcon, GraphIcon, LockIcon, SearchIcon, ShieldIcon } from './components/FeatureIcon'
import { FlowLoopDiagram } from './components/FlowLoopDiagram'
import { GlassCard } from './components/GlassCard'
import { MarketingSection } from './components/MarketingSection'
import { WaveDivider } from './components/WaveDivider'

/* ── Keyframes ────────────────────────────────────── */

const fadeUp = keyframes`
  from { opacity: 0; transform: translateY(24px); }
  to { opacity: 1; transform: translateY(0); }
`

const bob = keyframes`
  0%, 100% { transform: translateY(0); }
  50% { transform: translateY(8px); }
`

const drawLine = keyframes`
  to { stroke-dashoffset: 0; }
`

const glowPulse = keyframes`
  0%, 100% { opacity: 0.5; }
  50% { opacity: 1; }
`

const float1 = keyframes`
  0%, 100% { transform: translate(0, 0); }
  50% { transform: translate(8px, -18px); }
`
const float2 = keyframes`
  0%, 100% { transform: translate(0, 0); }
  50% { transform: translate(-12px, -14px); }
`
const float3 = keyframes`
  0%, 100% { transform: translate(0, 0); }
  50% { transform: translate(6px, -22px); }
`

const layerFade = keyframes`
  from { opacity: 0; transform: translateY(16px); }
  to { opacity: 1; transform: translateY(0); }
`

/* ── Page Shell ───────────────────────────────────── */

const Page = styled.div`
  display: grid;
  gap: 3rem;
  max-width: 1200px;
  margin: 0 auto;

  @media (prefers-reduced-motion: reduce) {
    *, *::before, *::after {
      animation: none !important;
      transition: none !important;
    }
  }
`

/* ── 1. Hero ──────────────────────────────────────── */

const Hero = styled.section`
  min-height: 90vh;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  text-align: center;
  padding: 1rem 0.8rem;
  position: relative;

  &::before, &::after {
    content: '';
    position: absolute;
    border-radius: 50%;
    pointer-events: none;
  }

  &::before {
    width: 500px;
    height: 500px;
    top: 10%;
    left: 10%;
    background: radial-gradient(circle, rgba(6, 182, 212, 0.12), transparent 60%);
    animation: ${glowPulse} 5s ease-in-out infinite;
  }

  &::after {
    width: 400px;
    height: 400px;
    bottom: 15%;
    right: 8%;
    background: radial-gradient(circle, rgba(13, 148, 136, 0.1), transparent 55%);
    animation: ${glowPulse} 6s ease-in-out infinite 1.5s;
  }
`

const FLOAT_ANIMS = [float1, float2, float3]

const Particle = styled.div<{ $top: string; $left: string; $dur: string; $anim: number }>`
  position: absolute;
  width: 4px;
  height: 4px;
  border-radius: 50%;
  background: rgba(94, 234, 212, 0.12);
  pointer-events: none;
  top: ${({ $top }) => $top};
  left: ${({ $left }) => $left};
  animation: ${({ $anim }) => FLOAT_ANIMS[$anim % 3]} ${({ $dur }) => $dur} ease-in-out infinite;
`

const Wordmark = styled.h1`
  margin: 0.55rem 0 0;
  font-family: var(--font-display);
  font-weight: 700;
  font-size: clamp(2.4rem, 9vw, 5.2rem);
  letter-spacing: 0.02em;
  color: #8ec8c8;
  animation: ${fadeUp} 1.4s cubic-bezier(0.16, 1, 0.3, 1) 0.35s both;
  position: relative;
  z-index: 1;

  span { color: #5eead4; }
`

const Underline = styled.div`
  width: clamp(260px, 50vw, 520px);
  height: 14px;
  margin-top: 0.2rem;
  opacity: 0;
  animation: ${fadeUp} 1.2s ease 0.55s forwards;
  position: relative;
  z-index: 1;

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
  position: relative;
  z-index: 1;
`

const Sub = styled.p`
  margin: 0.65rem 0 0;
  max-width: 34rem;
  font-size: 0.9rem;
  color: rgba(178, 245, 234, 0.38);
  line-height: 1.75;
  font-style: italic;
  animation: ${fadeUp} 1.2s ease 0.95s both;
  position: relative;
  z-index: 1;
`

const Ctas = styled.div`
  display: flex;
  flex-wrap: wrap;
  justify-content: center;
  gap: 0.75rem;
  margin-top: 1.4rem;
  animation: ${fadeUp} 1.2s ease 1.1s both;
  position: relative;
  z-index: 1;
`

const PrimaryCTA = styled(NavLink)`
  text-decoration: none;
  border-radius: 999px;
  padding: 0.72rem 1.5rem;
  font-weight: 800;
  font-size: 0.85rem;
  letter-spacing: 0.04em;
  color: #001821;
  background: linear-gradient(135deg, #06b6d4, #0d9488, #5eead4);
  box-shadow: 0 4px 24px rgba(94, 234, 212, 0.2);
  transition: transform 200ms ease, box-shadow 200ms ease;

  &:hover {
    transform: translateY(-2px);
    box-shadow: 0 8px 32px rgba(94, 234, 212, 0.35);
  }
`

const GhostCTA = styled(NavLink)`
  text-decoration: none;
  border-radius: 999px;
  padding: 0.72rem 1.5rem;
  font-weight: 700;
  font-size: 0.85rem;
  letter-spacing: 0.04em;
  color: #cffff8;
  border: 1px solid rgba(94, 234, 212, 0.22);
  background: rgba(94, 234, 212, 0.08);
  transition: all 200ms ease;

  &:hover {
    background: rgba(94, 234, 212, 0.16);
    transform: translateY(-2px);
  }
`

const ScrollHint = styled.div`
  position: absolute;
  bottom: 1.2rem;
  left: 50%;
  transform: translateX(-50%);
  font-family: var(--font-mono);
  font-size: 0.55rem;
  letter-spacing: 0.25em;
  text-transform: uppercase;
  color: rgba(94, 234, 212, 0.32);
  animation: ${fadeUp} 1s ease 1.4s both, ${bob} 2s ease-in-out infinite;
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 4px;
  z-index: 1;
`

/* ── 3. Journey Grid ──────────────────────────────── */

const JourneyGrid = styled.div`
  margin-top: 1.5rem;
  display: grid;
  grid-template-columns: repeat(3, minmax(0, 1fr));
  gap: 1rem;

  @media (max-width: 768px) {
    grid-template-columns: 1fr;
  }
`

/* ── 4. Architecture ──────────────────────────────── */

const ArchSvg = styled.div`
  width: 100%;
  max-width: 800px;
  margin: 2rem auto 0;

  svg { width: 100%; height: auto; }

  .arch-layer { opacity: 0; }

  &.visible {
    .arch-layer { animation: ${layerFade} 600ms ease-out both; }
    .arch-l1 { animation-delay: 100ms; }
    .arch-l2 { animation-delay: 400ms; }
    .arch-l3 { animation-delay: 700ms; }
  }

  @media (prefers-reduced-motion: reduce) {
    .arch-layer { opacity: 1; }
  }
`

/* ── 5. Benefits ──────────────────────────────────── */

const BenefitsGrid = styled.div`
  margin-top: 1.5rem;
  display: grid;
  grid-template-columns: 1.2fr 0.8fr;
  gap: 2rem;
  align-items: start;

  @media (max-width: 768px) {
    grid-template-columns: 1fr;
  }
`

const BenefitRow = styled.div`
  display: flex;
  gap: 1rem;
  align-items: flex-start;
  padding: 0.75rem 0;
  border-bottom: 1px solid rgba(94, 234, 212, 0.06);

  &:last-child { border-bottom: none; }
`

const BenefitIcon = styled.div`
  flex-shrink: 0;
  width: 36px;
  height: 36px;
  border-radius: 10px;
  background: rgba(94, 234, 212, 0.06);
  display: flex;
  align-items: center;
  justify-content: center;
  color: #5eead4;
`

const BenefitText = styled.div`
  h4 {
    margin: 0;
    font-family: var(--font-display);
    font-size: 0.9rem;
    color: #f2fffd;
  }

  p {
    margin: 0.25rem 0 0;
    font-size: 0.82rem;
    color: rgba(201, 244, 236, 0.7);
    line-height: 1.55;
  }
`

const IllustrationBox = styled.div`
  display: flex;
  align-items: center;
  justify-content: center;
  border-radius: 20px;
  border: 1px solid rgba(94, 234, 212, 0.1);
  background: rgba(8, 20, 32, 0.5);
  padding: 2rem;
  min-height: 260px;

  svg { width: 100%; max-width: 280px; height: auto; }

  @media (max-width: 768px) { min-height: 200px; }
`

/* ── 6. Social Proof ──────────────────────────────── */

const ProofStrip = styled.section`
  border-radius: 16px;
  border: 1px solid rgba(94, 234, 212, 0.08);
  background: rgba(8, 20, 32, 0.45);
  backdrop-filter: blur(8px);
  padding: 1.25rem 2rem;
  display: flex;
  align-items: center;
  gap: 1rem;
  flex-wrap: wrap;
  justify-content: center;
`

const ProofLabel = styled.span`
  font-family: var(--font-mono);
  font-size: 0.65rem;
  letter-spacing: 0.2em;
  text-transform: uppercase;
  color: rgba(94, 234, 212, 0.4);
`

const ProofDot = styled.span`
  width: 3px;
  height: 3px;
  border-radius: 50%;
  background: rgba(94, 234, 212, 0.2);
`

const ProofTag = styled.span`
  font-family: var(--font-mono);
  font-size: 0.7rem;
  letter-spacing: 0.12em;
  color: rgba(94, 234, 212, 0.25);
  padding: 0.25rem 0.6rem;
  border: 1px solid rgba(94, 234, 212, 0.06);
  border-radius: 999px;
`

/* ── 7. Final CTA ─────────────────────────────────── */

const CtaBand = styled.section`
  border-radius: 24px;
  background: linear-gradient(135deg, rgba(12, 53, 71, 0.8), rgba(8, 20, 32, 0.9));
  border: 1px solid rgba(94, 234, 212, 0.15);
  padding: 3rem 2rem;
  text-align: center;
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 0.5rem;

  h2 {
    margin: 0;
    font-family: var(--font-display);
    font-weight: 700;
    font-size: clamp(1.3rem, 3vw, 1.8rem);
    color: #f7fffd;
  }

  p { margin: 0; color: rgba(178, 245, 234, 0.5); font-size: 0.9rem; }
`

const CtaBandActions = styled.div`
  display: flex;
  gap: 0.75rem;
  margin-top: 1rem;
  flex-wrap: wrap;
  justify-content: center;
`

/* ── Data ─────────────────────────────────────────── */

const PARTICLES = [
  { top: '15%', left: '12%', dur: '9s', anim: 0 },
  { top: '25%', left: '82%', dur: '11s', anim: 1 },
  { top: '60%', left: '8%', dur: '10s', anim: 2 },
  { top: '70%', left: '88%', dur: '8s', anim: 0 },
  { top: '40%', left: '5%', dur: '12s', anim: 1 },
  { top: '80%', left: '75%', dur: '9.5s', anim: 2 },
]

const PROOF_TAGS = ['AI Agents', 'MCP Workflows', 'Long-running Tasks', 'Enterprise Memory']

const BENEFITS = [
  { icon: <FlowIcon />, title: 'Eliminate repeated prompts', desc: 'Preserve session intelligence as reusable memory so agents never start from zero.' },
  { icon: <SearchIcon />, title: 'Trace answer provenance', desc: 'Link every output to its source engrams and timelines instead of opaque model recall.' },
  { icon: <LockIcon />, title: 'Scoped memory boundaries', desc: 'Give each workflow a clear project scope with ownership and permission control.' },
  { icon: <GraphIcon />, title: 'Grounded retrieval', desc: 'Keep outputs aligned with current project reality through context-aware memory queries.' },
]

const ARCH_COMPARTMENTS = ['Sessions', 'Engrams', 'Documents', 'Projects']
const ARCH_PROVIDERS = ['OpenAI', 'Anthropic', 'Custom']

/* ── Component ────────────────────────────────────── */

export function LandingPage() {
  return (
    <Page>
      {/* 1. Hero */}
      <Hero>
        {PARTICLES.map((p, i) => (
          <Particle key={i} $top={p.top} $left={p.left} $dur={p.dur} $anim={p.anim} />
        ))}
        <MemoryStrandMark size="lg" />
        <Wordmark>Continu<span>Witty</span></Wordmark>
        <Underline>
          <svg width="100%" height="14" viewBox="0 0 520 14" preserveAspectRatio="none">
            <path d="M1 8 C 110 2, 220 12, 340 6 C 410 3, 460 9, 519 5" stroke="rgba(94,234,212,0.45)" strokeWidth="2" fill="none" strokeLinecap="round" />
          </svg>
        </Underline>
        <Tag>Intelligence that flows</Tag>
        <Sub>Persistent memory for agents and teams that run long, high-context work. Capture decisions, keep provenance, and resume without context loss.</Sub>
        <Ctas>
          <PrimaryCTA to={MARKETING_ROUTES.login}>Start Flowing</PrimaryCTA>
          <GhostCTA to={MARKETING_ROUTES.howItWorks}>See How It Works</GhostCTA>
        </Ctas>
        <ScrollHint>
          <span>Scroll to explore</span>
          <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2"><path d="M12 5v14M5 12l7 7 7-7" /></svg>
        </ScrollHint>
      </Hero>

      {/* 2. Product Loop Diagram */}
      <MarketingSection eyebrow="How it works" title="One continuous loop. No context lost." lead="Session to context to memory to continuation — an unbroken cycle that keeps agents and teams aligned across every interaction.">
        <FlowLoopDiagram />
      </MarketingSection>

      <WaveDivider />

      {/* 3. Journey Grid */}
      <MarketingSection eyebrow="The memory loop" title="Memory continuity, not prompt repetition." lead="ContinuWitty turns each session into a governed, searchable memory stream. Your next interaction starts with what matters." showWave>
        <JourneyGrid>
          <GlassCard icon={<ChatIcon />} title="1. Capture" description="Persist key milestones from active chats as engrams with tags, summaries, and visibility controls." />
          <GlassCard icon={<FlowIcon />} title="2. Continue" description="Start the next session with pinned engrams and documents so the agent can pick up immediately." />
          <GlassCard icon={<ShieldIcon />} title="3. Govern" description="Use project scopes, token controls, and audit timelines to scale memory operations safely." />
        </JourneyGrid>
      </MarketingSection>

      {/* 4. Architecture Strip */}
      <MarketingSection eyebrow="Architecture" title="One platform, every actor." lead="Humans and agents connect through the same continuity layer, powered by any AI provider.">
        <AnimatedSvgDiagram>
          <ArchSvg>
            <svg viewBox="0 0 800 220" xmlns="http://www.w3.org/2000/svg">
              <g className="arch-layer arch-l1">
                <rect x="220" y="10" width="140" height="44" rx="12" fill="rgba(94,234,212,0.06)" stroke="rgba(94,234,212,0.2)" strokeWidth="1" />
                <text x="290" y="37" textAnchor="middle" fontFamily="Comfortaa, sans-serif" fontSize="11" fill="#e8fffb">Humans</text>
                <rect x="440" y="10" width="140" height="44" rx="12" fill="rgba(94,234,212,0.06)" stroke="rgba(94,234,212,0.2)" strokeWidth="1" />
                <text x="510" y="37" textAnchor="middle" fontFamily="Comfortaa, sans-serif" fontSize="11" fill="#e8fffb">AI Agents</text>
                <line x1="360" y1="32" x2="440" y2="32" stroke="rgba(94,234,212,0.15)" strokeWidth="1" strokeDasharray="4 3" />
              </g>
              <g className="arch-layer arch-l2">
                <line x1="290" y1="54" x2="290" y2="80" stroke="rgba(94,234,212,0.12)" strokeWidth="1" />
                <line x1="510" y1="54" x2="510" y2="80" stroke="rgba(94,234,212,0.12)" strokeWidth="1" />
                <rect x="100" y="80" width="600" height="56" rx="14" fill="rgba(12,53,71,0.5)" stroke="rgba(94,234,212,0.25)" strokeWidth="1" />
                <text x="400" y="99" textAnchor="middle" fontFamily="JetBrains Mono, monospace" fontSize="8" fill="rgba(94,234,212,0.4)" letterSpacing="2">CONTINUWITTY PLATFORM</text>
                {ARCH_COMPARTMENTS.map((label, i) => (
                  <g key={label}>
                    <text x={175 + i * 150} y="124" textAnchor="middle" fontFamily="Comfortaa, sans-serif" fontSize="10" fill="#a7f3d0">{label}</text>
                    {i < 3 && <line x1={250 + i * 150} y1="106" x2={250 + i * 150} y2="130" stroke="rgba(94,234,212,0.08)" strokeWidth="1" strokeDasharray="3 3" />}
                  </g>
                ))}
              </g>
              <g className="arch-layer arch-l3">
                {ARCH_PROVIDERS.map((label, i) => (
                  <g key={label}>
                    <line x1={250 + i * 150} y1="136" x2={250 + i * 150} y2="165" stroke="rgba(94,234,212,0.1)" strokeWidth="1" />
                    <rect x={190 + i * 150} y="165" width="120" height="36" rx="10" fill="rgba(8,20,32,0.7)" stroke="rgba(94,234,212,0.12)" strokeWidth="1" />
                    <text x={250 + i * 150} y="188" textAnchor="middle" fontFamily="Comfortaa, sans-serif" fontSize="10" fill="rgba(178,245,234,0.5)">{label}</text>
                  </g>
                ))}
              </g>
            </svg>
          </ArchSvg>
        </AnimatedSvgDiagram>
      </MarketingSection>

      <WaveDivider />

      {/* 5. Agent Benefits */}
      <MarketingSection eyebrow="For agents & teams" title="Built for human teams and autonomous agents." lead="Agents cannot ship reliable output without stable memory. ContinuWitty gives them durable context, explainable provenance, and project boundaries.">
        <BenefitsGrid>
          <div>
            {BENEFITS.map((b) => (
              <BenefitRow key={b.title}>
                <BenefitIcon>{b.icon}</BenefitIcon>
                <BenefitText>
                  <h4>{b.title}</h4>
                  <p>{b.desc}</p>
                </BenefitText>
              </BenefitRow>
            ))}
          </div>
          <IllustrationBox>
            <svg viewBox="0 0 280 200" xmlns="http://www.w3.org/2000/svg">
              <circle cx="70" cy="55" r="18" fill="none" stroke="rgba(94,234,212,0.3)" strokeWidth="1.5" />
              <circle cx="70" cy="55" r="6" fill="rgba(94,234,212,0.15)" />
              <rect x="54" y="80" width="32" height="45" rx="10" fill="rgba(94,234,212,0.06)" stroke="rgba(94,234,212,0.2)" strokeWidth="1" />
              <text x="70" y="140" textAnchor="middle" fontFamily="JetBrains Mono, monospace" fontSize="8" fill="rgba(94,234,212,0.3)">HUMAN</text>
              <rect x="192" y="40" width="36" height="36" rx="8" fill="none" stroke="rgba(94,234,212,0.3)" strokeWidth="1.5" />
              <circle cx="203" cy="55" r="3" fill="rgba(94,234,212,0.25)" />
              <circle cx="217" cy="55" r="3" fill="rgba(94,234,212,0.25)" />
              <rect x="194" y="80" width="32" height="45" rx="10" fill="rgba(94,234,212,0.06)" stroke="rgba(94,234,212,0.2)" strokeWidth="1" />
              <text x="210" y="140" textAnchor="middle" fontFamily="JetBrains Mono, monospace" fontSize="8" fill="rgba(94,234,212,0.3)">AGENT</text>
              <path d="M95,75 C120,60 130,60 140,70 C150,80 160,80 170,65 C175,58 180,62 195,70" stroke="rgba(94,234,212,0.25)" strokeWidth="2" fill="none" strokeLinecap="round" />
              <path d="M95,85 C115,75 130,72 140,80 C150,88 165,85 175,75 C180,70 185,72 195,80" stroke="rgba(167,243,208,0.15)" strokeWidth="1.5" fill="none" strokeLinecap="round" />
              <text x="140" y="170" textAnchor="middle" fontFamily="JetBrains Mono, monospace" fontSize="7" fill="rgba(94,234,212,0.2)" letterSpacing="1">SHARED MEMORY</text>
            </svg>
          </IllustrationBox>
        </BenefitsGrid>
      </MarketingSection>

      {/* 6. Social Proof Strip */}
      <ProofStrip>
        <ProofLabel>Built for</ProofLabel>
        {PROOF_TAGS.map((tag, i) => (
          <span key={tag} style={{ display: 'contents' }}>
            {i > 0 && <ProofDot />}
            <ProofTag>{tag}</ProofTag>
          </span>
        ))}
      </ProofStrip>

      {/* 7. Final CTA Band */}
      <CtaBand>
        <h2>Ready to give your agents memory?</h2>
        <p>Start flowing in under two minutes.</p>
        <CtaBandActions>
          <PrimaryCTA to={MARKETING_ROUTES.login}>Start Flowing</PrimaryCTA>
          <GhostCTA to={MARKETING_ROUTES.pricing}>See Pricing</GhostCTA>
        </CtaBandActions>
      </CtaBand>
    </Page>
  )
}
