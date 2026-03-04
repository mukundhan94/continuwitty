import { NavLink } from 'react-router-dom'
import styled, { keyframes } from 'styled-components'

import { AnimatedSvgDiagram } from './components/AnimatedSvgDiagram'
import { GlassCard } from './components/GlassCard'
import { MarketingSection } from './components/MarketingSection'
import {
  ClockIcon,
  EyeIcon,
  KeyIcon,
  LockIcon,
  ShieldIcon,
  UsersIcon,
} from './components/FeatureIcon'
import { MARKETING_ROUTES } from '../../constants'

/* ------------------------------------------------------------------ */
/*  Keyframes                                                         */
/* ------------------------------------------------------------------ */

const fadeUp = keyframes`
  from { opacity: 0; transform: translateY(16px); }
  to   { opacity: 1; transform: translateY(0); }
`

const drawOn = keyframes`
  from { stroke-dashoffset: 1; }
  to   { stroke-dashoffset: 0; }
`

const popIn = keyframes`
  from { opacity: 0; transform: scale(0.6); }
  to   { opacity: 1; transform: scale(1); }
`

/* ------------------------------------------------------------------ */
/*  Hero                                                              */
/* ------------------------------------------------------------------ */

const HeroSection = styled.section`
  text-align: center;
  padding: 3.5rem 1.5rem 2rem;
  position: relative;

  @media (max-width: 768px) {
    padding: 2.5rem 1rem 1.5rem;
  }
`

const Eyebrow = styled.p`
  margin: 0 0 0.6rem;
  font-family: var(--font-mono);
  font-size: 0.625rem;
  letter-spacing: 0.25em;
  text-transform: uppercase;
  color: rgba(94, 234, 212, 0.4);
`

const HeroTitle = styled.h1`
  margin: 0;
  font-family: var(--font-display);
  font-weight: 700;
  font-size: clamp(1.8rem, 4vw, 2.8rem);
  color: #f2fffd;
  line-height: 1.12;
  max-width: 48ch;
  margin-inline: auto;
`

const HeroLead = styled.p`
  margin: 1rem auto 0;
  max-width: 58ch;
  color: rgba(178, 245, 234, 0.76);
  line-height: 1.7;
  font-size: 0.95rem;
`

const HeroActions = styled.div`
  margin-top: 1.3rem;
  display: flex;
  justify-content: center;
  gap: 0.7rem;
  flex-wrap: wrap;
`

const PrimaryBtn = styled(NavLink)`
  text-decoration: none;
  border-radius: 999px;
  padding: 0.6rem 1.3rem;
  font-weight: 800;
  font-size: 0.85rem;
  letter-spacing: 0.03em;
  color: #001821;
  background: linear-gradient(135deg, #06b6d4, #14b8a6, #5eead4);
`

const GhostBtn = styled(NavLink)`
  text-decoration: none;
  border-radius: 999px;
  padding: 0.6rem 1.3rem;
  font-weight: 700;
  font-size: 0.85rem;
  letter-spacing: 0.03em;
  color: #dffcf8;
  border: 1px solid rgba(94, 234, 212, 0.3);
  background: rgba(94, 234, 212, 0.06);
`

/* ------------------------------------------------------------------ */
/*  Trust Indicators                                                  */
/* ------------------------------------------------------------------ */

const TrustRow = styled.div`
  display: grid;
  grid-template-columns: repeat(4, 1fr);
  gap: 0.7rem;
  margin-top: 1.2rem;

  @media (max-width: 900px) {
    grid-template-columns: repeat(2, 1fr);
  }

  @media (max-width: 500px) {
    grid-template-columns: 1fr;
  }
`

const TrustCard = styled.div`
  border-radius: 16px;
  border: 1px solid rgba(94, 234, 212, 0.14);
  background: rgba(8, 20, 32, 0.55);
  padding: 1rem 0.9rem;
  position: relative;
  overflow: hidden;

  &::before {
    content: '';
    position: absolute;
    top: 0;
    left: 0;
    right: 0;
    height: 2px;
    background: linear-gradient(90deg, #06b6d4, #5eead4);
    opacity: 0.6;
  }

  h3 {
    margin: 0;
    font-family: var(--font-display);
    font-size: 0.9rem;
    color: #f2fffd;
    display: flex;
    align-items: center;
    gap: 0.5rem;
  }

  p {
    margin: 0.4rem 0 0;
    color: rgba(178, 245, 234, 0.65);
    font-size: 0.8rem;
    line-height: 1.55;
  }
`

/* ------------------------------------------------------------------ */
/*  Governance Architecture Diagram                                   */
/* ------------------------------------------------------------------ */

const GovDiagramWrap = styled.div`
  margin-top: 1.2rem;

  svg text {
    font-family: var(--font-display);
  }

  svg .gov-node {
    opacity: 0;
  }

  svg .gov-path {
    stroke-dasharray: 1;
    stroke-dashoffset: 1;
  }

  &.visible svg .gov-node {
    animation: ${fadeUp} 450ms ease forwards;
  }

  &.visible svg .gov-node:nth-child(1) { animation-delay: 100ms; }
  &.visible svg .gov-node:nth-child(2) { animation-delay: 300ms; }
  &.visible svg .gov-node:nth-child(3) { animation-delay: 400ms; }
  &.visible svg .gov-node:nth-child(4) { animation-delay: 500ms; }
  &.visible svg .gov-node:nth-child(5) { animation-delay: 600ms; }
  &.visible svg .gov-node:nth-child(6) { animation-delay: 700ms; }
  &.visible svg .gov-node:nth-child(7) { animation-delay: 750ms; }

  &.visible svg .gov-path {
    animation: ${drawOn} 500ms ease forwards;
    animation-delay: 900ms;
  }

  @media (prefers-reduced-motion: reduce) {
    svg .gov-node { opacity: 1; }
    svg .gov-path { stroke-dashoffset: 0; }
    &.visible svg .gov-node,
    &.visible svg .gov-path {
      animation: none;
    }
  }
`

function GovernanceDiagram() {
  return (
    <AnimatedSvgDiagram>
      <GovDiagramWrap>
        <svg viewBox="0 0 700 340" fill="none" xmlns="http://www.w3.org/2000/svg" role="img" aria-label="Governance architecture showing organization, projects, sessions, and engrams hierarchy">
          {/* Organization node */}
          <g className="gov-node">
            <rect x={270} y={15} width={160} height={50} rx={12} fill="rgba(8,20,32,0.8)" stroke="rgba(94,234,212,0.35)" strokeWidth={1.2} />
            <text x={350} y={45} textAnchor="middle" fill="#5eead4" fontSize={12} fontWeight={600}>Organization</text>
          </g>

          {/* Project nodes */}
          <g className="gov-node">
            <rect x={100} y={110} width={140} height={45} rx={10} fill="rgba(8,20,32,0.7)" stroke="rgba(94,234,212,0.25)" strokeWidth={1} />
            <text x={170} y={137} textAnchor="middle" fill="#f2fffd" fontSize={11} fontWeight={500}>Project Alpha</text>
          </g>
          <g className="gov-node">
            <rect x={460} y={110} width={140} height={45} rx={10} fill="rgba(8,20,32,0.7)" stroke="rgba(94,234,212,0.25)" strokeWidth={1} />
            <text x={530} y={137} textAnchor="middle" fill="#f2fffd" fontSize={11} fontWeight={500}>Project Beta</text>
          </g>

          {/* Session / Token nodes under Project Alpha */}
          <g className="gov-node">
            <rect x={50} y={200} width={110} height={40} rx={8} fill="rgba(8,20,32,0.6)" stroke="rgba(94,234,212,0.18)" strokeWidth={1} />
            <text x={105} y={224} textAnchor="middle" fill="rgba(201,244,236,0.85)" fontSize={10}>Sessions</text>
          </g>
          <g className="gov-node">
            <rect x={180} y={200} width={110} height={40} rx={8} fill="rgba(8,20,32,0.6)" stroke="rgba(94,234,212,0.18)" strokeWidth={1} />
            <text x={235} y={224} textAnchor="middle" fill="rgba(201,244,236,0.85)" fontSize={10}>MCP Tokens</text>
          </g>

          {/* Engram nodes */}
          <g className="gov-node">
            <rect x={100} y={280} width={120} height={38} rx={8} fill="rgba(6,182,212,0.08)" stroke="rgba(6,182,212,0.22)" strokeWidth={1} />
            <text x={160} y={303} textAnchor="middle" fill="rgba(178,245,234,0.8)" fontSize={10}>Engrams</text>
          </g>
          <g className="gov-node">
            <rect x={460} y={200} width={140} height={40} rx={8} fill="rgba(8,20,32,0.6)" stroke="rgba(94,234,212,0.18)" strokeWidth={1} />
            <text x={530} y={224} textAnchor="middle" fill="rgba(201,244,236,0.85)" fontSize={10}>Sessions + Tokens</text>
          </g>

          {/* Connection lines */}
          <path className="gov-path" d="M350,65 L170,110" stroke="rgba(94,234,212,0.2)" strokeWidth={1.2} />
          <path className="gov-path" d="M350,65 L530,110" stroke="rgba(94,234,212,0.2)" strokeWidth={1.2} />
          <path className="gov-path" d="M170,155 L105,200" stroke="rgba(94,234,212,0.15)" strokeWidth={1} />
          <path className="gov-path" d="M170,155 L235,200" stroke="rgba(94,234,212,0.15)" strokeWidth={1} />
          <path className="gov-path" d="M140,240 L160,280" stroke="rgba(94,234,212,0.12)" strokeWidth={1} />
          <path className="gov-path" d="M530,155 L530,200" stroke="rgba(94,234,212,0.15)" strokeWidth={1} />

          {/* Lock icons on connections */}
          <g className="gov-node">
            <circle cx={260} cy={88} r={8} fill="rgba(94,234,212,0.1)" stroke="rgba(94,234,212,0.3)" strokeWidth={0.8} />
            <text x={260} y={91} textAnchor="middle" fill="#5eead4" fontSize={8}>🔒</text>
          </g>
        </svg>
      </GovDiagramWrap>
    </AnimatedSvgDiagram>
  )
}

/* ------------------------------------------------------------------ */
/*  Audit Timeline                                                    */
/* ------------------------------------------------------------------ */

const TimelineWrap = styled.div`
  margin-top: 1.2rem;

  svg .tl-marker {
    opacity: 0;
  }

  svg .tl-line {
    stroke-dasharray: 1;
    stroke-dashoffset: 1;
  }

  &.visible svg .tl-line {
    animation: ${drawOn} 800ms ease forwards;
    animation-delay: 200ms;
  }

  &.visible svg .tl-marker {
    animation: ${popIn} 350ms ease forwards;
  }

  &.visible svg .tl-marker:nth-child(2) { animation-delay: 600ms; }
  &.visible svg .tl-marker:nth-child(3) { animation-delay: 800ms; }
  &.visible svg .tl-marker:nth-child(4) { animation-delay: 1000ms; }
  &.visible svg .tl-marker:nth-child(5) { animation-delay: 1200ms; }
  &.visible svg .tl-marker:nth-child(6) { animation-delay: 1400ms; }

  @media (prefers-reduced-motion: reduce) {
    svg .tl-marker { opacity: 1; }
    svg .tl-line { stroke-dashoffset: 0; }
    &.visible svg .tl-marker,
    &.visible svg .tl-line {
      animation: none;
    }
  }
`

const timelineEvents = [
  { x: 80, label: 'Session\nCreated', time: '09:12' },
  { x: 230, label: 'Engram\nSaved', time: '09:28' },
  { x: 380, label: 'Token\nIssued', time: '10:05' },
  { x: 530, label: 'Member\nAdded', time: '11:30' },
  { x: 680, label: 'Audit\nExported', time: '14:15' },
]

function AuditTimeline() {
  return (
    <AnimatedSvgDiagram>
      <TimelineWrap>
        <svg viewBox="0 0 800 130" fill="none" xmlns="http://www.w3.org/2000/svg" role="img" aria-label="Audit timeline showing event markers along a horizontal axis">
          {/* Main timeline line */}
          <line className="tl-line" x1={40} y1={50} x2={760} y2={50} stroke="rgba(94,234,212,0.2)" strokeWidth={1.5} />

          {/* Event markers */}
          {timelineEvents.map((evt) => (
            <g key={evt.time} className="tl-marker">
              <circle cx={evt.x} cy={50} r={6} fill="rgba(94,234,212,0.15)" stroke="#5eead4" strokeWidth={1.2} />
              <circle cx={evt.x} cy={50} r={2.5} fill="#5eead4" />
              {evt.label.split('\n').map((line, i) => (
                <text key={i} x={evt.x} y={75 + i * 13} textAnchor="middle" fill="rgba(201,244,236,0.75)" fontSize={9} fontFamily="var(--font-display)">
                  {line}
                </text>
              ))}
              <text x={evt.x} y={38} textAnchor="middle" fill="rgba(94,234,212,0.45)" fontSize={8} fontFamily="var(--font-mono)">
                {evt.time}
              </text>
            </g>
          ))}
        </svg>
      </TimelineWrap>
    </AnimatedSvgDiagram>
  )
}

/* ------------------------------------------------------------------ */
/*  Outcomes Grid                                                     */
/* ------------------------------------------------------------------ */

const OutcomesGrid = styled.div`
  display: grid;
  grid-template-columns: repeat(3, 1fr);
  gap: 0.7rem;
  margin-top: 1.2rem;

  @media (max-width: 800px) {
    grid-template-columns: 1fr;
  }
`

/* ------------------------------------------------------------------ */
/*  CTA Band                                                          */
/* ------------------------------------------------------------------ */

const CtaBand = styled.section`
  text-align: center;
  padding: 2.5rem 1.5rem;
  border-radius: 24px;
  background: linear-gradient(135deg, rgba(6, 182, 212, 0.12), rgba(94, 234, 212, 0.06));
  border: 1px solid rgba(94, 234, 212, 0.14);
  margin-top: 1rem;

  h2 {
    margin: 0;
    font-family: var(--font-display);
    font-size: clamp(1.4rem, 3vw, 2rem);
    color: #f2fffd;
  }

  p {
    margin: 0.6rem auto 0;
    max-width: 50ch;
    color: rgba(178, 245, 234, 0.7);
    font-size: 0.9rem;
  }
`

const CtaRow = styled.div`
  margin-top: 1.2rem;
  display: flex;
  justify-content: center;
  gap: 0.7rem;
  flex-wrap: wrap;
`

/* ------------------------------------------------------------------ */
/*  Page Shell                                                        */
/* ------------------------------------------------------------------ */

const PageShell = styled.div`
  display: grid;
  gap: 1.2rem;
`

/* ------------------------------------------------------------------ */
/*  Component                                                         */
/* ------------------------------------------------------------------ */

export function ForEnterprisePage() {
  return (
    <PageShell>
      {/* 1. Enterprise Hero */}
      <HeroSection>
        <Eyebrow>For Enterprise</Eyebrow>
        <HeroTitle>Operational memory with governance, policy boundaries, and auditability.</HeroTitle>
        <HeroLead>
          Deploy memory continuity with role-aware project controls, scoped MCP tokens, and audit
          trails that security and platform teams can trust.
        </HeroLead>
        <HeroActions>
          <PrimaryBtn to={MARKETING_ROUTES.login}>Book Architecture Review</PrimaryBtn>
          <GhostBtn to={MARKETING_ROUTES.product}>See Platform</GhostBtn>
        </HeroActions>
      </HeroSection>

      {/* 2. Trust Indicators */}
      <MarketingSection
        eyebrow="Trust Layer"
        title="Enterprise-grade controls built into every layer."
      >
        <TrustRow>
          <TrustCard>
            <h3><ShieldIcon /> Role-Based Access</h3>
            <p>Project-level ownership with role management for members and contributors.</p>
          </TrustCard>
          <TrustCard>
            <h3><ClockIcon /> Audit Trails</h3>
            <p>Auditable memory mutations and access timelines for every project action.</p>
          </TrustCard>
          <TrustCard>
            <h3><KeyIcon /> Token Scopes</h3>
            <p>MCP tokens scoped to specific tools and project boundaries.</p>
          </TrustCard>
          <TrustCard>
            <h3><EyeIcon /> Policy Controls</h3>
            <p>Admin controls for session lifecycle, memory retention, and curation rules.</p>
          </TrustCard>
        </TrustRow>
      </MarketingSection>

      {/* 3. Governance Architecture Diagram */}
      <MarketingSection
        eyebrow="Architecture"
        title="Hierarchical governance from organization to engram."
        lead="Every entity inherits security boundaries from its parent. Projects scope sessions, tokens scope tools, and audit trails capture everything."
        showWave
      >
        <GovernanceDiagram />
      </MarketingSection>

      {/* 4. Audit Timeline */}
      <MarketingSection
        eyebrow="Observability"
        title="Every action leaves a trace."
        lead="View session creation, memory mutations, token issuance, and member changes on a unified audit timeline."
      >
        <AuditTimeline />
      </MarketingSection>

      {/* 5. Enterprise Outcomes */}
      <MarketingSection
        eyebrow="Outcomes"
        title="What enterprise teams get from governed agent memory."
      >
        <OutcomesGrid>
          <GlassCard
            icon={<LockIcon />}
            title="Compliance Confidence"
            description="Every memory access is logged, scoped, and traceable. Built for teams that answer to auditors."
          />
          <GlassCard
            icon={<UsersIcon />}
            title="Team Autonomy"
            description="Project boundaries let teams operate independently without leaking context across organizational lines."
          />
          <GlassCard
            icon={<EyeIcon />}
            title="Operational Visibility"
            description="Real-time observability into agent sessions, memory growth, and token usage across all projects."
          />
        </OutcomesGrid>
      </MarketingSection>

      {/* 6. CTA */}
      <CtaBand>
        <h2>Ready to govern your agent memory?</h2>
        <p>Talk to our team about enterprise deployment, compliance, and architecture fit.</p>
        <CtaRow>
          <PrimaryBtn to={MARKETING_ROUTES.login}>Book Architecture Review</PrimaryBtn>
          <GhostBtn to={MARKETING_ROUTES.pricing}>See Plans</GhostBtn>
        </CtaRow>
      </CtaBand>
    </PageShell>
  )
}
