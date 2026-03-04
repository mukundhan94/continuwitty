import { NavLink } from 'react-router-dom'
import styled, { keyframes } from 'styled-components'

import { AnimatedSvgDiagram } from './components/AnimatedSvgDiagram'
import { GlassCard } from './components/GlassCard'
import { MarketingSection } from './components/MarketingSection'
import {
  CodeIcon,
  KeyIcon,
  LayersIcon,
  PlugIcon,
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

const pulseOpacity = keyframes`
  0%, 100% { opacity: 0.3; }
  50%      { opacity: 0.7; }
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

const DotGrid = styled.div`
  position: absolute;
  inset: 0;
  pointer-events: none;
  background-image: radial-gradient(rgba(94, 234, 212, 0.06) 1px, transparent 1px);
  background-size: 24px 24px;
  mask-image: radial-gradient(ellipse 60% 70% at 50% 30%, black, transparent);
`

const EyebrowText = styled.p`
  margin: 0 0 0.6rem;
  font-family: var(--font-mono);
  font-size: 0.625rem;
  letter-spacing: 0.25em;
  text-transform: uppercase;
  color: rgba(94, 234, 212, 0.4);
  position: relative;
`

const HeroTitle = styled.h1`
  margin: 0;
  font-family: var(--font-display);
  font-weight: 700;
  font-size: clamp(1.8rem, 4vw, 2.8rem);
  color: #f2fffd;
  line-height: 1.12;
  max-width: 46ch;
  margin-inline: auto;
  position: relative;
`

const HeroLead = styled.p`
  margin: 1rem auto 0;
  max-width: 58ch;
  color: rgba(178, 245, 234, 0.76);
  line-height: 1.7;
  font-size: 0.95rem;
  position: relative;
`

const HeroActions = styled.div`
  margin-top: 1.3rem;
  display: flex;
  justify-content: center;
  gap: 0.7rem;
  flex-wrap: wrap;
  position: relative;
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
/*  Code Snippet Showcase                                             */
/* ------------------------------------------------------------------ */

const SnippetGrid = styled.div`
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 0.8rem;
  margin-top: 1.2rem;

  @media (max-width: 800px) {
    grid-template-columns: 1fr;
  }
`

const CodePanel = styled.div`
  border-radius: 16px;
  border: 1px solid rgba(94, 234, 212, 0.14);
  background: rgba(4, 10, 18, 0.85);
  overflow: hidden;
`

const CodeHeader = styled.div`
  display: flex;
  align-items: center;
  gap: 0.5rem;
  padding: 0.6rem 0.8rem;
  border-bottom: 1px solid rgba(94, 234, 212, 0.08);

  span:first-child {
    display: flex;
    gap: 5px;
  }
`

const Dot = styled.span<{ $color: string }>`
  width: 8px;
  height: 8px;
  border-radius: 50%;
  background: ${(p) => p.$color};
  opacity: 0.6;
`

const CodeLabel = styled.span`
  font-family: var(--font-mono);
  font-size: 0.65rem;
  color: rgba(178, 245, 234, 0.5);
  letter-spacing: 0.05em;
  text-transform: uppercase;
`

const CodeBody = styled.pre`
  margin: 0;
  padding: 0.8rem;
  font-family: var(--font-mono);
  font-size: 0.75rem;
  line-height: 1.65;
  color: rgba(201, 244, 236, 0.88);
  overflow-x: auto;
  white-space: pre;

  .kw { color: #06b6d4; }
  .str { color: #5eead4; }
  .cmt { color: rgba(94, 234, 212, 0.35); }
  .fn { color: #a5f3fc; }
  .num { color: #fb923c; }
`

/* ------------------------------------------------------------------ */
/*  API Flow Diagram                                                  */
/* ------------------------------------------------------------------ */

const ApiDiagramWrap = styled.div`
  margin-top: 1.2rem;

  svg text {
    font-family: var(--font-display);
  }

  svg .api-node {
    opacity: 0;
  }

  svg .api-path {
    stroke-dasharray: 1;
    stroke-dashoffset: 1;
  }

  &.visible svg .api-node {
    animation: ${fadeUp} 450ms ease forwards;
  }

  &.visible svg .api-node:nth-child(1) { animation-delay: 100ms; }
  &.visible svg .api-node:nth-child(2) { animation-delay: 300ms; }
  &.visible svg .api-node:nth-child(3) { animation-delay: 500ms; }
  &.visible svg .api-node:nth-child(4) { animation-delay: 700ms; }

  &.visible svg .api-path {
    animation: ${drawOn} 500ms ease forwards;
    animation-delay: 900ms;
  }

  @media (prefers-reduced-motion: reduce) {
    svg .api-node { opacity: 1; }
    svg .api-path { stroke-dashoffset: 0; }
    &.visible svg .api-node,
    &.visible svg .api-path {
      animation: none;
    }
  }
`

function ApiFlowDiagram() {
  return (
    <AnimatedSvgDiagram>
      <ApiDiagramWrap>
        <svg viewBox="0 0 800 240" fill="none" xmlns="http://www.w3.org/2000/svg" role="img" aria-label="API flow diagram showing agent to ContinuWitty API to Engram Store with MCP stream">
          {/* Your Agent */}
          <g className="api-node">
            <rect x={40} y={80} width={140} height={60} rx={14} fill="rgba(8,20,32,0.8)" stroke="rgba(94,234,212,0.3)" strokeWidth={1.2} />
            <text x={110} y={105} textAnchor="middle" fill="#5eead4" fontSize={11} fontWeight={600}>Your Agent</text>
            <text x={110} y={122} textAnchor="middle" fill="rgba(178,245,234,0.5)" fontSize={9}>REST / MCP client</text>
          </g>

          {/* ContinuWitty API */}
          <g className="api-node">
            <rect x={280} y={60} width={200} height={100} rx={16} fill="rgba(6,182,212,0.08)" stroke="rgba(94,234,212,0.3)" strokeWidth={1.2} />
            <text x={380} y={95} textAnchor="middle" fill="#f2fffd" fontSize={12} fontWeight={600}>ContinuWitty API</text>
            <text x={380} y={115} textAnchor="middle" fill="rgba(178,245,234,0.55)" fontSize={9}>Sessions | Engrams | Docs</text>
            <text x={380} y={132} textAnchor="middle" fill="rgba(178,245,234,0.55)" fontSize={9}>Query | Admin</text>
          </g>

          {/* Engram Store */}
          <g className="api-node">
            <rect x={580} y={80} width={160} height={60} rx={14} fill="rgba(8,20,32,0.8)" stroke="rgba(94,234,212,0.25)" strokeWidth={1.2} />
            <text x={660} y={105} textAnchor="middle" fill="#5eead4" fontSize={11} fontWeight={600}>Engram Store</text>
            <text x={660} y={122} textAnchor="middle" fill="rgba(178,245,234,0.5)" fontSize={9}>Persistence layer</text>
          </g>

          {/* MCP Stream node */}
          <g className="api-node">
            <rect x={300} y={195} width={160} height={40} rx={10} fill="rgba(8,20,32,0.7)" stroke="rgba(6,182,212,0.3)" strokeWidth={1} strokeDasharray="4 3" />
            <text x={380} y={220} textAnchor="middle" fill="rgba(6,182,212,0.7)" fontSize={10}>MCP SSE Stream</text>
          </g>

          {/* Arrows */}
          {/* Agent -> API (REST) */}
          <path className="api-path" d="M180,110 L280,110" stroke="rgba(94,234,212,0.3)" strokeWidth={1.5} markerEnd="url(#arrowTeal)" />
          <text x={230} y={102} textAnchor="middle" fill="rgba(94,234,212,0.4)" fontSize={8} fontFamily="var(--font-mono)">REST</text>

          {/* API -> Store */}
          <path className="api-path" d="M480,110 L580,110" stroke="rgba(94,234,212,0.25)" strokeWidth={1.5} markerEnd="url(#arrowTeal)" />

          {/* Agent -> MCP (dashed) */}
          <path className="api-path" d="M140,140 C140,190 300,215 300,215" stroke="rgba(6,182,212,0.25)" strokeWidth={1.2} strokeDasharray="5 3" markerEnd="url(#arrowCyan)" />
          <text x={195} y={185} textAnchor="middle" fill="rgba(6,182,212,0.4)" fontSize={8} fontFamily="var(--font-mono)">MCP</text>

          {/* MCP -> API */}
          <path className="api-path" d="M380,195 L380,160" stroke="rgba(6,182,212,0.2)" strokeWidth={1} strokeDasharray="4 3" markerEnd="url(#arrowCyan)" />

          {/* Arrow markers */}
          <defs>
            <marker id="arrowTeal" markerWidth={8} markerHeight={6} refX={7} refY={3} orient="auto">
              <path d="M0,0 L8,3 L0,6" fill="rgba(94,234,212,0.4)" />
            </marker>
            <marker id="arrowCyan" markerWidth={8} markerHeight={6} refX={7} refY={3} orient="auto">
              <path d="M0,0 L8,3 L0,6" fill="rgba(6,182,212,0.35)" />
            </marker>
          </defs>
        </svg>
      </ApiDiagramWrap>
    </AnimatedSvgDiagram>
  )
}

/* ------------------------------------------------------------------ */
/*  MCP Protocol Visual                                               */
/* ------------------------------------------------------------------ */

const McpWrap = styled.div`
  display: grid;
  grid-template-columns: 1fr 1.2fr;
  gap: 1.5rem;
  margin-top: 1.2rem;
  align-items: center;

  @media (max-width: 800px) {
    grid-template-columns: 1fr;
  }
`

const McpText = styled.div`
  h3 {
    margin: 0;
    font-family: var(--font-display);
    font-size: 1.1rem;
    color: #f2fffd;
  }

  p {
    margin: 0.6rem 0 0;
    color: rgba(178, 245, 234, 0.7);
    font-size: 0.88rem;
    line-height: 1.65;
  }

  ul {
    margin: 0.8rem 0 0;
    padding-left: 1rem;
    color: rgba(201, 244, 236, 0.8);
    font-size: 0.82rem;
    line-height: 1.6;
    display: grid;
    gap: 0.3rem;
  }
`

const McpDiagramWrap = styled.div`
  svg text {
    font-family: var(--font-mono);
  }

  svg .mcp-node {
    opacity: 0;
  }

  svg .mcp-line {
    opacity: 0;
  }

  svg .mcp-pulse {
    opacity: 0;
  }

  &.visible svg .mcp-node {
    animation: ${fadeUp} 400ms ease forwards;
  }

  &.visible svg .mcp-node:nth-child(1) { animation-delay: 100ms; }
  &.visible svg .mcp-node:nth-child(2) { animation-delay: 250ms; }
  &.visible svg .mcp-node:nth-child(3) { animation-delay: 400ms; }
  &.visible svg .mcp-node:nth-child(4) { animation-delay: 550ms; }
  &.visible svg .mcp-node:nth-child(5) { animation-delay: 700ms; }
  &.visible svg .mcp-node:nth-child(6) { animation-delay: 850ms; }
  &.visible svg .mcp-node:nth-child(7) { animation-delay: 950ms; }

  &.visible svg .mcp-line {
    animation: ${fadeUp} 350ms ease forwards;
    animation-delay: 1100ms;
  }

  &.visible svg .mcp-pulse {
    animation: ${pulseOpacity} 2.5s ease-in-out infinite;
    animation-delay: 1400ms;
  }

  @media (prefers-reduced-motion: reduce) {
    svg .mcp-node,
    svg .mcp-line { opacity: 1; }
    &.visible svg .mcp-node,
    &.visible svg .mcp-line,
    &.visible svg .mcp-pulse {
      animation: none;
    }
    svg .mcp-pulse { opacity: 0.5; }
  }
`

function McpDiagram() {
  const tools = [
    { angle: -60, label: 'session.create' },
    { angle: -20, label: 'engram.save' },
    { angle: 20, label: 'engram.query' },
    { angle: 60, label: 'session.continue' },
  ]
  const resources = [
    { angle: 120, label: 'sessions' },
    { angle: 160, label: 'engrams' },
  ]
  const cx = 180
  const cy = 150
  const r = 110

  return (
    <AnimatedSvgDiagram>
      <McpDiagramWrap>
        <svg viewBox="0 0 360 300" fill="none" xmlns="http://www.w3.org/2000/svg" role="img" aria-label="MCP protocol diagram showing server hub with tool and resource nodes">
          {/* Center hub */}
          <g className="mcp-node">
            <polygon
              points={Array.from({ length: 6 }, (_, i) => {
                const a = (Math.PI / 3) * i - Math.PI / 2
                return `${cx + 32 * Math.cos(a)},${cy + 32 * Math.sin(a)}`
              }).join(' ')}
              fill="rgba(6,182,212,0.1)"
              stroke="rgba(94,234,212,0.35)"
              strokeWidth={1.2}
            />
            <text x={cx} y={cy - 4} textAnchor="middle" fill="#5eead4" fontSize={9} fontWeight={600}>MCP</text>
            <text x={cx} y={cy + 9} textAnchor="middle" fill="#5eead4" fontSize={8}>Server</text>
          </g>

          {/* Tool nodes */}
          {tools.map((t) => {
            const a = (t.angle * Math.PI) / 180
            const tx = cx + r * Math.cos(a)
            const ty = cy + r * Math.sin(a)
            return (
              <g key={t.label} className="mcp-node">
                <line className="mcp-line" x1={cx} y1={cy} x2={tx} y2={ty} stroke="rgba(94,234,212,0.15)" strokeWidth={1} />
                <circle className="mcp-pulse" cx={(cx + tx) / 2} cy={(cy + ty) / 2} r={2} fill="#5eead4" />
                <rect x={tx - 48} y={ty - 12} width={96} height={24} rx={6} fill="rgba(8,20,32,0.75)" stroke="rgba(94,234,212,0.22)" strokeWidth={0.8} />
                <text x={tx} y={ty + 3} textAnchor="middle" fill="rgba(201,244,236,0.8)" fontSize={8}>{t.label}</text>
              </g>
            )
          })}

          {/* Resource nodes */}
          {resources.map((res) => {
            const a = (res.angle * Math.PI) / 180
            const rx2 = cx + r * Math.cos(a)
            const ry = cy + r * Math.sin(a)
            return (
              <g key={res.label} className="mcp-node">
                <line className="mcp-line" x1={cx} y1={cy} x2={rx2} y2={ry} stroke="rgba(6,182,212,0.15)" strokeWidth={1} strokeDasharray="4 3" />
                <rect x={rx2 - 38} y={ry - 12} width={76} height={24} rx={6} fill="rgba(6,182,212,0.06)" stroke="rgba(6,182,212,0.22)" strokeWidth={0.8} />
                <text x={rx2} y={ry + 3} textAnchor="middle" fill="rgba(6,182,212,0.7)" fontSize={8}>{res.label}</text>
              </g>
            )
          })}
        </svg>
      </McpDiagramWrap>
    </AnimatedSvgDiagram>
  )
}

/* ------------------------------------------------------------------ */
/*  Integration Grid                                                  */
/* ------------------------------------------------------------------ */

const IntGrid = styled.div`
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

function DeveloperHero() {
  return (
    <HeroSection>
      <DotGrid />
      <EyebrowText>For Developers</EyebrowText>
      <HeroTitle>Build agents that remember decisions, not just prompts.</HeroTitle>
      <HeroLead>
        Use REST and MCP workflows to create sessions, persist milestones as engrams, query
        prior context, and continue work with deterministic continuity.
      </HeroLead>
      <HeroActions>
        <PrimaryBtn to={MARKETING_ROUTES.login}>Start Building</PrimaryBtn>
        <GhostBtn to={MARKETING_ROUTES.product}>See Platform</GhostBtn>
      </HeroActions>
    </HeroSection>
  )
}

function DeveloperQuickStart() {
  return (
    <MarketingSection
      eyebrow="Quick Start"
      title="From zero to persistent memory in minutes."
      lead="Create a session, chat, and save the result as a durable engram — all through simple API calls."
    >
      <SnippetGrid>
        <CodePanel>
          <CodeHeader>
            <span>
              <Dot $color="#ff5f57" />
              <Dot $color="#febc2e" />
              <Dot $color="#28c840" />
            </span>
            <CodeLabel>Create Session — REST</CodeLabel>
          </CodeHeader>
          <CodeBody>{`\
`}<span className="cmt"># Create a new session</span>{`
`}<span className="kw">POST</span>{` /api/v1/sessions
`}<span className="kw">Content-Type:</span>{` application/json
`}<span className="kw">Authorization:</span>{` Bearer <token>

{
  `}<span className="str">"project_id"</span>{`: `}<span className="str">"proj_abc123"</span>{`,
  `}<span className="str">"model"</span>{`:    `}<span className="str">"claude-sonnet-4-20250514"</span>{`,
  `}<span className="str">"provider"</span>{`: `}<span className="str">"anthropic"</span>{`
}`}</CodeBody>
        </CodePanel>

        <CodePanel>
          <CodeHeader>
            <span>
              <Dot $color="#ff5f57" />
              <Dot $color="#febc2e" />
              <Dot $color="#28c840" />
            </span>
            <CodeLabel>Save Engram — REST</CodeLabel>
          </CodeHeader>
          <CodeBody>{`\
`}<span className="cmt"># Persist session result as engram</span>{`
`}<span className="kw">POST</span>{` /api/v1/engrams
`}<span className="kw">Content-Type:</span>{` application/json

{
  `}<span className="str">"session_id"</span>{`: `}<span className="str">"sess_xyz"</span>{`,
  `}<span className="str">"content"</span>{`:    `}<span className="str">"Decision: use event sourcing..."</span>{`,
  `}<span className="str">"tags"</span>{`:       [`}<span className="str">"architecture"</span>{`, `}<span className="str">"v2"</span>{`],
  `}<span className="str">"visibility"</span>{`: `}<span className="str">"project"</span>{`
}`}</CodeBody>
        </CodePanel>
      </SnippetGrid>
    </MarketingSection>
  )
}

function DeveloperArchitecture() {
  return (
    <MarketingSection
      eyebrow="Architecture"
      title="Two protocols, one memory layer."
      lead="Use REST for direct integration or MCP for agent-native streaming. Both paths converge on the same engram store."
      showWave
    >
      <ApiFlowDiagram />
    </MarketingSection>
  )
}

function DeveloperMCP() {
  return (
    <MarketingSection
      eyebrow="MCP Protocol"
      title="Native agent integration via Model Context Protocol."
    >
      <McpWrap>
        <McpText>
          <h3>Tools + Resources over SSE</h3>
          <p>
            ContinuWitty exposes a full MCP server with tools for session lifecycle,
            engram persistence, and contextual query. Agents call tools via JSON-RPC
            and receive streamed responses over Server-Sent Events.
          </p>
          <ul>
            <li>session.create — Start scoped conversations</li>
            <li>engram.save — Persist milestone memory</li>
            <li>engram.query — Retrieve prior context by relevance</li>
            <li>session.continue — Resume with pinned artifacts</li>
          </ul>
        </McpText>
        <McpDiagram />
      </McpWrap>
    </MarketingSection>
  )
}

function DeveloperIntegrations() {
  return (
    <MarketingSection
      eyebrow="Integration"
      title="Multiple paths into the memory layer."
    >
      <IntGrid>
        <GlassCard icon={<CodeIcon />} title="REST API" description="Standard HTTP endpoints for sessions, engrams, documents, and admin operations." />
        <GlassCard icon={<PlugIcon />} title="MCP Protocol" description="Agent-native tools and resources over SSE for streaming agent orchestration." />
        <GlassCard icon={<LayersIcon />} title="Export / Import" description="Portable memory bundles for project transfer and environment migration." />
        <GlassCard icon={<KeyIcon />} title="Token Auth" description="Scoped personal access tokens with tool-level and project-level restrictions." />
      </IntGrid>
    </MarketingSection>
  )
}

function DeveloperCTA() {
  return (
    <CtaBand>
      <h2>Ready to build with persistent memory?</h2>
      <p>Get started with the API and give your agents context that lasts.</p>
      <CtaRow>
        <PrimaryBtn to={MARKETING_ROUTES.login}>Start Building</PrimaryBtn>
        <GhostBtn to={MARKETING_ROUTES.howItWorks}>See How It Works</GhostBtn>
      </CtaRow>
    </CtaBand>
  )
}

export function ForDevelopersPage() {
  return (
    <PageShell>
      <DeveloperHero />
      <DeveloperQuickStart />
      <DeveloperArchitecture />
      <DeveloperMCP />
      <DeveloperIntegrations />
      <DeveloperCTA />
    </PageShell>
  )
}
