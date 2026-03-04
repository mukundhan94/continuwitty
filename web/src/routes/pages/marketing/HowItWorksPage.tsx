import { NavLink } from 'react-router-dom'
import styled, { keyframes } from 'styled-components'

import { AnimatedSvgDiagram } from './components/AnimatedSvgDiagram'
import { GlassCard } from './components/GlassCard'
import { MarketingSection } from './components/MarketingSection'
import {
  BotIcon,
  FlowIcon,
  ShieldIcon,
  UsersIcon,
} from './components/FeatureIcon'
import { MARKETING_ROUTES } from '../../constants'

/* ------------------------------------------------------------------ */
/*  Keyframes                                                         */
/* ------------------------------------------------------------------ */

const fadeUp = keyframes`
  from { opacity: 0; transform: translateY(18px); }
  to   { opacity: 1; transform: translateY(0); }
`

const drawOn = keyframes`
  from { stroke-dashoffset: 1; }
  to   { stroke-dashoffset: 0; }
`

const marchAnts = keyframes`
  to { stroke-dashoffset: -20; }
`

const dotTravel = keyframes`
  0%   { offset-distance: 0%; opacity: 0; }
  8%   { opacity: 1; }
  92%  { opacity: 1; }
  100% { offset-distance: 100%; opacity: 0; }
`

/* ------------------------------------------------------------------ */
/*  Hero                                                              */
/* ------------------------------------------------------------------ */

const HeroSection = styled.section`
  text-align: center;
  padding: 3.5rem 1.5rem 2rem;

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
  max-width: 52ch;
  margin-inline: auto;
`

const HeroLead = styled.p`
  margin: 1rem auto 0;
  max-width: 60ch;
  color: rgba(178, 245, 234, 0.76);
  line-height: 1.7;
  font-size: 0.95rem;
`

/* ------------------------------------------------------------------ */
/*  Workflow Diagram                                                  */
/* ------------------------------------------------------------------ */

const DiagramWrap = styled.div`
  margin-top: 1rem;

  svg text {
    font-family: var(--font-display);
  }

  svg .step-node {
    opacity: 0;
  }

  svg .step-path {
    stroke-dasharray: 1;
    stroke-dashoffset: 1;
  }

  svg .ant-path {
    stroke-dasharray: 6 4;
    stroke-dashoffset: 0;
  }

  svg .flow-dot {
    opacity: 0;
  }

  &.visible svg .step-node {
    animation: ${fadeUp} 500ms ease forwards;
  }

  &.visible svg .step-node:nth-child(2) { animation-delay: 100ms; }
  &.visible svg .step-node:nth-child(3) { animation-delay: 300ms; }
  &.visible svg .step-node:nth-child(4) { animation-delay: 500ms; }
  &.visible svg .step-node:nth-child(5) { animation-delay: 700ms; }

  &.visible svg .step-path {
    animation: ${drawOn} 600ms ease forwards;
  }

  &.visible svg .step-path:nth-of-type(1) { animation-delay: 900ms; }
  &.visible svg .step-path:nth-of-type(2) { animation-delay: 1100ms; }
  &.visible svg .step-path:nth-of-type(3) { animation-delay: 1300ms; }
  &.visible svg .step-path:nth-of-type(4) { animation-delay: 1500ms; }

  &.visible svg .ant-path {
    animation: ${marchAnts} 1.2s linear infinite;
    animation-delay: 1700ms;
  }

  &.visible svg .flow-dot {
    animation: ${dotTravel} 3s linear infinite;
    animation-delay: 2s;
  }

  @media (prefers-reduced-motion: reduce) {
    svg .step-node { opacity: 1; }
    svg .step-path { stroke-dashoffset: 0; }
    &.visible svg .step-node,
    &.visible svg .step-path,
    &.visible svg .ant-path,
    &.visible svg .flow-dot {
      animation: none;
    }
  }
`

function WorkflowDiagram() {
  const steps = [
    { x: 60, label: 'Start Session', desc: 'Create scoped chat', icon: '1' },
    { x: 270, label: 'Blend Context', desc: 'Pull engrams & docs', icon: '2' },
    { x: 480, label: 'Save Memory', desc: 'Persist as engram', icon: '3' },
    { x: 690, label: 'Continue', desc: 'New session + pins', icon: '4' },
  ]

  return (
    <AnimatedSvgDiagram>
      <DiagramWrap>
        <svg viewBox="0 0 850 320" fill="none" xmlns="http://www.w3.org/2000/svg" role="img" aria-label="Workflow diagram showing the four-step ContinuWitty loop">
          {/* Step nodes */}
          {steps.map((s, i) => (
            <g key={s.label} className="step-node">
              {/* Card background */}
              <rect x={s.x} y={60} width={140} height={140} rx={16} fill="rgba(8,20,32,0.7)" stroke="rgba(94,234,212,0.25)" strokeWidth={1.2} />

              {/* Step badge */}
              <circle cx={s.x + 70} cy={95} r={20} fill="rgba(94,234,212,0.12)" stroke="rgba(94,234,212,0.4)" strokeWidth={1} />
              <text x={s.x + 70} y={100} textAnchor="middle" fill="#5eead4" fontSize={14} fontWeight={700}>{s.icon}</text>

              {/* Label */}
              <text x={s.x + 70} y={140} textAnchor="middle" fill="#f2fffd" fontSize={12.5} fontWeight={600}>{s.label}</text>

              {/* Description */}
              <text x={s.x + 70} y={162} textAnchor="middle" fill="rgba(178,245,234,0.6)" fontSize={10}>{s.desc}</text>

              {/* Bottom accent line */}
              <rect x={s.x + 30} y={185} width={80} height={2} rx={1} fill={`rgba(94,234,212,${0.15 + i * 0.08})`} />
            </g>
          ))}

          {/* Connection paths (dashed) */}
          <path className="step-path" d="M200,130 C230,130 240,130 270,130" stroke="rgba(94,234,212,0.3)" strokeWidth={1.5} strokeDasharray="6 4" />
          <path className="step-path" d="M410,130 C440,130 450,130 480,130" stroke="rgba(94,234,212,0.3)" strokeWidth={1.5} strokeDasharray="6 4" />
          <path className="step-path" d="M620,130 C650,130 660,130 690,130" stroke="rgba(94,234,212,0.3)" strokeWidth={1.5} strokeDasharray="6 4" />

          {/* Return loop (curved path below) */}
          <path
            className="step-path"
            d="M760,200 C760,270 425,290 130,270 C90,268 60,240 60,200"
            stroke="rgba(6,182,212,0.2)"
            strokeWidth={1.5}
            strokeDasharray="8 5"
            fill="none"
          />

          {/* Return loop label */}
          <text x={425} y={285} textAnchor="middle" fill="rgba(94,234,212,0.35)" fontSize={9} fontFamily="var(--font-mono)">continuity loop</text>

          {/* Arrow on return loop */}
          <polygon points="58,205 65,195 72,205" fill="rgba(6,182,212,0.35)" />

          {/* Marching ants on forward paths */}
          <path className="ant-path" d="M200,130 L270,130" stroke="rgba(94,234,212,0.15)" strokeWidth={1} fill="none" />
          <path className="ant-path" d="M410,130 L480,130" stroke="rgba(94,234,212,0.15)" strokeWidth={1} fill="none" />
          <path className="ant-path" d="M620,130 L690,130" stroke="rgba(94,234,212,0.15)" strokeWidth={1} fill="none" />

          {/* Flow dots */}
          <circle className="flow-dot" r={3} fill="#5eead4" style={{ offsetPath: `path('M200,130 L270,130')` }} />
          <circle className="flow-dot" r={3} fill="#5eead4" style={{ offsetPath: `path('M410,130 L480,130')`, animationDelay: '2.3s' }} />
          <circle className="flow-dot" r={3} fill="#5eead4" style={{ offsetPath: `path('M620,130 L690,130')`, animationDelay: '2.6s' }} />
        </svg>
      </DiagramWrap>
    </AnimatedSvgDiagram>
  )
}

/* ------------------------------------------------------------------ */
/*  Before / After Comparison                                         */
/* ------------------------------------------------------------------ */

const CompareGrid = styled.div`
  display: grid;
  grid-template-columns: 1fr auto 1fr;
  gap: 0;
  margin-top: 1.2rem;

  @media (max-width: 768px) {
    grid-template-columns: 1fr;
    gap: 0.6rem;
  }
`

const ComparePanel = styled.div<{ $variant: 'without' | 'with' }>`
  border-radius: 20px;
  padding: 1.5rem;
  border: 1px solid ${(p) =>
    p.$variant === 'without' ? 'rgba(251,146,60,0.18)' : 'rgba(94,234,212,0.18)'};
  background: ${(p) =>
    p.$variant === 'without' ? 'rgba(251,146,60,0.04)' : 'rgba(94,234,212,0.04)'};
`

const CompareTitle = styled.h3<{ $variant: 'without' | 'with' }>`
  margin: 0 0 1rem;
  font-family: var(--font-display);
  font-size: 1rem;
  color: ${(p) => (p.$variant === 'without' ? '#fb923c' : '#5eead4')};
`

const CompareItem = styled.div<{ $variant: 'without' | 'with' }>`
  display: flex;
  gap: 0.6rem;
  align-items: flex-start;
  margin-bottom: 0.8rem;

  svg {
    flex-shrink: 0;
    margin-top: 2px;
  }

  p {
    margin: 0;
    font-size: 0.85rem;
    color: rgba(201, 244, 236, 0.8);
    line-height: 1.5;
  }
`

const Divider = styled.div`
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 0 1rem;

  @media (max-width: 768px) {
    padding: 0;
    justify-content: center;
  }
`

const VsBadge = styled.span`
  display: flex;
  align-items: center;
  justify-content: center;
  width: 40px;
  height: 40px;
  border-radius: 50%;
  background: rgba(94, 234, 212, 0.1);
  border: 1px solid rgba(94, 234, 212, 0.25);
  font-family: var(--font-mono);
  font-size: 0.7rem;
  font-weight: 700;
  color: #5eead4;
  letter-spacing: 0.08em;
`

function XMark() {
  return (
    <svg width={16} height={16} viewBox="0 0 16 16" fill="none">
      <circle cx={8} cy={8} r={7} stroke="rgba(251,146,60,0.5)" strokeWidth={1} />
      <path d="M5.5 5.5L10.5 10.5M10.5 5.5L5.5 10.5" stroke="#fb923c" strokeWidth={1.2} strokeLinecap="round" />
    </svg>
  )
}

function CheckMark() {
  return (
    <svg width={16} height={16} viewBox="0 0 16 16" fill="none">
      <circle cx={8} cy={8} r={7} stroke="rgba(94,234,212,0.5)" strokeWidth={1} />
      <path d="M5 8L7 10.5L11 5.5" stroke="#5eead4" strokeWidth={1.2} strokeLinecap="round" strokeLinejoin="round" />
    </svg>
  )
}

/* ------------------------------------------------------------------ */
/*  Who This Helps Grid                                               */
/* ------------------------------------------------------------------ */

const HelpGrid = styled.div`
  display: grid;
  grid-template-columns: repeat(4, 1fr);
  gap: 0.7rem;
  margin-top: 1.2rem;

  @media (max-width: 980px) {
    grid-template-columns: repeat(2, 1fr);
  }

  @media (max-width: 600px) {
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

const PrimaryBtn = styled(NavLink)`
  text-decoration: none;
  border-radius: 9999px;
  padding: 0.6rem 1.3rem;
  font-weight: 800;
  font-size: 0.85rem;
  letter-spacing: 0.03em;
  color: #001821;
  background: var(--cta-gradient);
`

const GhostBtn = styled(NavLink)`
  text-decoration: none;
  border-radius: 9999px;
  padding: 0.6rem 1.3rem;
  font-weight: 700;
  font-size: 0.85rem;
  letter-spacing: 0.03em;
  color: #dffcf8;
  border: 1px solid rgba(94, 234, 212, 0.3);
  background: rgba(94, 234, 212, 0.06);
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

export function HowItWorksPage() {
  return (
    <PageShell>
      {/* 1. Hero */}
      <HeroSection>
        <Eyebrow>How it works</Eyebrow>
        <HeroTitle>From conversation to durable memory in a single operating loop.</HeroTitle>
        <HeroLead>
          Create a session, gather context, save milestone memory, and continue with pinned
          continuity artifacts. The loop keeps your agents and teams aligned across every
          interaction.
        </HeroLead>
      </HeroSection>

      {/* 2. Animated Workflow Diagram */}
      <MarketingSection
        eyebrow="The Loop"
        title="Four steps to persistent agent memory."
        lead="Each step builds on the last, forming a continuous loop that ensures no context is ever lost."
        showWave
      >
        <WorkflowDiagram />
      </MarketingSection>

      {/* 3. Before / After Comparison */}
      <MarketingSection
        eyebrow="The Difference"
        title="What changes when agents remember."
      >
        <CompareGrid>
          <ComparePanel $variant="without">
            <CompareTitle $variant="without">Without ContinuWitty</CompareTitle>
            <CompareItem $variant="without">
              <XMark />
              <p>Context is lost at the end of every session.</p>
            </CompareItem>
            <CompareItem $variant="without">
              <XMark />
              <p>Teams rediscover the same decisions repeatedly.</p>
            </CompareItem>
            <CompareItem $variant="without">
              <XMark />
              <p>No provenance trail for AI-generated conclusions.</p>
            </CompareItem>
            <CompareItem $variant="without">
              <XMark />
              <p>Memory access is ungoverned and unscoped.</p>
            </CompareItem>
          </ComparePanel>

          <Divider>
            <VsBadge>VS</VsBadge>
          </Divider>

          <ComparePanel $variant="with">
            <CompareTitle $variant="with">With ContinuWitty</CompareTitle>
            <CompareItem $variant="with">
              <CheckMark />
              <p>Memory persists as engrams across sessions.</p>
            </CompareItem>
            <CompareItem $variant="with">
              <CheckMark />
              <p>Decisions are traceable with source provenance.</p>
            </CompareItem>
            <CompareItem $variant="with">
              <CheckMark />
              <p>Every conclusion links back to its evidence chain.</p>
            </CompareItem>
            <CompareItem $variant="with">
              <CheckMark />
              <p>Project-level controls scope who sees what.</p>
            </CompareItem>
          </ComparePanel>
        </CompareGrid>
      </MarketingSection>

      {/* 4. Who This Helps */}
      <MarketingSection
        eyebrow="Built For"
        title="Teams and operators who need agents that remember."
      >
        <HelpGrid>
          <GlassCard icon={<BotIcon />} title="AI Operators" description="Manage long-running agent workflows where context continuity is critical to output quality." />
          <GlassCard icon={<UsersIcon />} title="Product Teams" description="Preserve decision traceability across sprints so rationale survives team rotation." />
          <GlassCard icon={<FlowIcon />} title="Engineering Teams" description="Reduce rediscovery work by giving agents access to prior session knowledge." />
          <GlassCard icon={<ShieldIcon />} title="Security Teams" description="Enforce scoped memory access with project boundaries and audit trails." />
        </HelpGrid>
      </MarketingSection>

      {/* 5. CTA Band */}
      <CtaBand>
        <h2>Ready to close the loop?</h2>
        <p>Start building agents that remember context like long-term collaborators.</p>
        <CtaRow>
          <PrimaryBtn to={MARKETING_ROUTES.login}>Start Flowing</PrimaryBtn>
          <GhostBtn to={MARKETING_ROUTES.product}>Explore Features</GhostBtn>
        </CtaRow>
      </CtaBand>
    </PageShell>
  )
}
