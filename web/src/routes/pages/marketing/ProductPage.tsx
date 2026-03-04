import { NavLink } from 'react-router-dom'
import styled, { keyframes } from 'styled-components'

import { MARKETING_ROUTES } from '../../constants'
import { AnimatedSvgDiagram } from './components/AnimatedSvgDiagram'
import { ChatIcon, DocIcon, EyeIcon, GraphIcon } from './components/FeatureIcon'
import { GlassCard } from './components/GlassCard'
import { MarketingSection } from './components/MarketingSection'
import { WaveDivider } from './components/WaveDivider'

const slideInLeft = keyframes`
  from { opacity: 0; transform: translateX(-30px); }
  to { opacity: 1; transform: translateX(0); }
`

const Page = styled.div`
  display: grid;
  gap: 3rem;
  max-width: 1200px;
  margin: 0 auto;

  @media (prefers-reduced-motion: reduce) {
    *, *::before, *::after { animation: none !important; transition: none !important; }
  }
`

const Hero = styled.section`
  text-align: center;
  padding: 4rem 1rem 2rem;
  position: relative;

  &::before {
    content: '';
    position: absolute;
    width: 600px;
    height: 600px;
    top: -100px;
    left: 50%;
    transform: translateX(-50%);
    background: radial-gradient(circle, rgba(6, 182, 212, 0.08), transparent 55%);
    pointer-events: none;
  }
`

const Eyebrow = styled.p`
  margin: 0 0 0.75rem;
  font-family: var(--font-mono);
  font-size: 0.625rem;
  letter-spacing: 0.25em;
  text-transform: uppercase;
  color: rgba(94, 234, 212, 0.4);
`

const Title = styled.h1`
  margin: 0 auto;
  font-family: var(--font-display);
  font-weight: 700;
  font-size: clamp(1.8rem, 3.5vw, 3rem);
  color: #f2fffd;
  max-width: 700px;
  line-height: 1.15;
`

const Lead = styled.p`
  margin: 1rem auto 0;
  max-width: 56ch;
  color: rgba(178, 245, 234, 0.65);
  line-height: 1.7;
  font-size: 0.95rem;
`

const HeroCtas = styled.div`
  margin-top: 1.5rem;
  display: flex;
  gap: 0.75rem;
  justify-content: center;
  flex-wrap: wrap;
`

const Btn = styled(NavLink)<{ $primary?: boolean }>`
  text-decoration: none;
  border-radius: 999px;
  padding: 0.72rem 1.5rem;
  font-weight: ${({ $primary }) => ($primary ? 800 : 700)};
  font-size: 0.85rem;
  color: ${({ $primary }) => ($primary ? '#001821' : '#cffff8')};
  background: ${({ $primary }) => ($primary ? 'linear-gradient(135deg, #06b6d4, #0d9488, #5eead4)' : 'rgba(94, 234, 212, 0.08)')};
  border: ${({ $primary }) => ($primary ? 'none' : '1px solid rgba(94, 234, 212, 0.22)')};
  box-shadow: ${({ $primary }) => ($primary ? '0 4px 24px rgba(94, 234, 212, 0.2)' : 'none')};
  transition: transform 200ms ease, box-shadow 200ms ease;
  &:hover { transform: translateY(-2px); }
`

const PillarGrid = styled.div`
  margin-top: 1.5rem;
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 1rem;
  @media (max-width: 768px) { grid-template-columns: 1fr; }
`

const FeatureRow = styled.div<{ $reverse?: boolean }>`
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 2rem;
  align-items: center;
  ${({ $reverse }) => $reverse && 'direction: rtl;'}
  & > * { direction: ltr; }
  @media (max-width: 768px) { grid-template-columns: 1fr; direction: ltr; }
`

const FeatureDiagram = styled.div`
  border-radius: 18px;
  border: 1px solid rgba(94, 234, 212, 0.1);
  background: rgba(8, 20, 32, 0.5);
  padding: 1.5rem;
  display: flex;
  align-items: center;
  justify-content: center;
  min-height: 200px;
  opacity: 0;
  svg { width: 100%; max-width: 320px; height: auto; }
  .visible & { animation: ${slideInLeft} 600ms ease-out both; }
  @media (prefers-reduced-motion: reduce) { opacity: 1; .visible & { animation: none; } }
`

const FeatureContent = styled.div`
  h3 { margin: 0; font-family: var(--font-display); font-weight: 600; font-size: 1.2rem; color: #f2fffd; }
  ul { margin: 0.75rem 0 0; padding: 0; list-style: none; display: grid; gap: 0.5rem; }
  li {
    font-size: 0.85rem; color: rgba(201, 244, 236, 0.75); line-height: 1.55; padding-left: 1.2rem; position: relative;
    &::before { content: ''; position: absolute; left: 0; top: 0.45em; width: 6px; height: 6px; border-radius: 50%; background: rgba(94, 234, 212, 0.3); }
  }
`

const DeepDiveGrid = styled.div`
  margin-top: 1.5rem;
  display: grid;
  gap: 2.5rem;
`

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
  h2 { margin: 0; font-family: var(--font-display); font-weight: 700; font-size: clamp(1.3rem, 3vw, 1.8rem); color: #f7fffd; }
  p { margin: 0; color: rgba(178, 245, 234, 0.5); font-size: 0.9rem; }
`

function ProductHero() {
  return (
    <Hero>
      <Eyebrow>Product</Eyebrow>
      <Title>One continuity system across chat, memory graph, and document context.</Title>
      <Lead>ContinuWitty combines session continuity, retrieval quality, link intelligence, and admin controls in one operational surface for humans and AI agents.</Lead>
      <HeroCtas>
        <Btn to={MARKETING_ROUTES.login} $primary>Start Flowing</Btn>
        <Btn to={MARKETING_ROUTES.howItWorks}>See How It Works</Btn>
      </HeroCtas>
    </Hero>
  )
}

function ProductCapabilityPillars() {
  return (
    <MarketingSection eyebrow="Capabilities" title="Four pillars of agent memory.">
      <PillarGrid>
        <GlassCard icon={<ChatIcon />} title="Session Continuity" description="Chat continuity with save-and-continue loops. Every session picks up where the last one left off." />
        <GlassCard icon={<GraphIcon />} title="Memory Graph" description="Engram graph links for richer context traversal. Explore how memories connect across sessions." />
        <GlassCard icon={<DocIcon />} title="Document Intelligence" description="Document ingestion and pinning per session. Ground your conversations in source material." />
        <GlassCard icon={<EyeIcon />} title="Admin & Curation" description="Admin memory curation, contradiction handling, and observability for operational confidence." />
      </PillarGrid>
    </MarketingSection>
  )
}

function ProductDeepDive() {
  return (
    <MarketingSection eyebrow="Deep Dive" title="See how each layer works.">
      <DeepDiveGrid>
        <AnimatedSvgDiagram>
          <FeatureRow>
            <FeatureDiagram>
              <svg viewBox="0 0 300 180" xmlns="http://www.w3.org/2000/svg">
                <rect x="20" y="10" width="120" height="24" rx="8" fill="rgba(94,234,212,0.06)" stroke="rgba(94,234,212,0.15)" strokeWidth="1" />
                <text x="80" y="26" textAnchor="middle" fontSize="8" fill="rgba(94,234,212,0.4)" fontFamily="JetBrains Mono">user message</text>
                <rect x="40" y="42" width="120" height="24" rx="8" fill="rgba(13,148,136,0.1)" stroke="rgba(94,234,212,0.15)" strokeWidth="1" />
                <text x="100" y="58" textAnchor="middle" fontSize="8" fill="rgba(94,234,212,0.4)" fontFamily="JetBrains Mono">agent reply</text>
                <rect x="50" y="82" width="80" height="22" rx="11" fill="rgba(6,182,212,0.15)" stroke="rgba(94,234,212,0.3)" strokeWidth="1" />
                <text x="90" y="97" textAnchor="middle" fontSize="8" fill="#5eead4" fontFamily="Comfortaa">Save Engram</text>
                <path d="M135,93 L165,93" stroke="rgba(94,234,212,0.25)" strokeWidth="1.5" strokeDasharray="4 3" />
                <polygon points="165,90 172,93 165,96" fill="rgba(94,234,212,0.3)" />
                <rect x="175" y="55" width="105" height="60" rx="12" fill="rgba(12,53,71,0.4)" stroke="rgba(94,234,212,0.2)" strokeWidth="1" />
                <text x="227" y="73" textAnchor="middle" fontSize="7" fill="rgba(94,234,212,0.3)" fontFamily="JetBrains Mono">NEW SESSION</text>
                <text x="227" y="95" textAnchor="middle" fontSize="7" fill="rgba(167,243,208,0.4)" fontFamily="JetBrains Mono">context loaded</text>
                <text x="150" y="165" textAnchor="middle" fontSize="7" fill="rgba(94,234,212,0.2)" fontFamily="JetBrains Mono" letterSpacing="1">SAVE AND CONTINUE</text>
              </svg>
            </FeatureDiagram>
            <FeatureContent>
              <h3>Chat that remembers</h3>
              <ul>
                <li>Save any session milestone as a durable engram with metadata</li>
                <li>Continue into a new session with pinned context already loaded</li>
                <li>No copy-paste, no re-explanation — the agent knows what happened</li>
              </ul>
            </FeatureContent>
          </FeatureRow>
        </AnimatedSvgDiagram>

        <AnimatedSvgDiagram>
          <FeatureRow $reverse>
            <FeatureDiagram>
              <svg viewBox="0 0 300 180" xmlns="http://www.w3.org/2000/svg">
                <rect x="40" y="15" width="220" height="28" rx="14" fill="rgba(8,20,32,0.7)" stroke="rgba(94,234,212,0.2)" strokeWidth="1" />
                <circle cx="62" cy="29" r="8" fill="none" stroke="rgba(94,234,212,0.3)" strokeWidth="1.5" />
                <line x1="68" y1="35" x2="72" y2="39" stroke="rgba(94,234,212,0.3)" strokeWidth="1.5" />
                <text x="85" y="33" fontSize="9" fill="rgba(94,234,212,0.25)" fontFamily="Nunito">architecture decisions...</text>
                {[0, 1, 2].map((i) => (
                  <g key={i}>
                    <rect x={60 + i * 12} y={55 + i * 32} width="180" height="26" rx="8" fill="rgba(12,53,71,0.5)" stroke="rgba(94,234,212,0.15)" strokeWidth="1" />
                    <text x={75 + i * 12} y={72 + i * 32} fontSize="8" fill="#a7f3d0" fontFamily="Nunito">engram: decision record #{i + 1}</text>
                    <rect x={200 + i * 12} y={58 + i * 32} width="32" height="16" rx="8" fill="rgba(94,234,212,0.1)" stroke="rgba(94,234,212,0.2)" strokeWidth="0.75" />
                    <text x={216 + i * 12} y={69 + i * 32} textAnchor="middle" fontSize="7" fill="#5eead4" fontFamily="JetBrains Mono">{(0.95 - i * 0.1).toFixed(2)}</text>
                  </g>
                ))}
                <text x="150" y="170" textAnchor="middle" fontSize="7" fill="rgba(94,234,212,0.2)" fontFamily="JetBrains Mono" letterSpacing="1">SEMANTIC QUERY</text>
              </svg>
            </FeatureDiagram>
            <FeatureContent>
              <h3>Memory you can query</h3>
              <ul>
                <li>Search engrams by meaning with relevance scoring</li>
                <li>Filter by tags, projects, visibility, and authority signals</li>
                <li>Rehydrate full context from any stored memory</li>
              </ul>
            </FeatureContent>
          </FeatureRow>
        </AnimatedSvgDiagram>

        <AnimatedSvgDiagram>
          <FeatureRow>
            <FeatureDiagram>
              <svg viewBox="0 0 300 170" xmlns="http://www.w3.org/2000/svg">
                <rect x="110" y="10" width="80" height="28" rx="8" fill="rgba(6,182,212,0.1)" stroke="rgba(94,234,212,0.25)" strokeWidth="1" />
                <text x="150" y="28" textAnchor="middle" fontSize="8" fill="#a7f3d0" fontFamily="Comfortaa">Organization</text>
                <line x1="130" y1="38" x2="90" y2="60" stroke="rgba(94,234,212,0.15)" strokeWidth="1" />
                <line x1="170" y1="38" x2="210" y2="60" stroke="rgba(94,234,212,0.15)" strokeWidth="1" />
                <rect x="45" y="60" width="90" height="24" rx="8" fill="rgba(13,148,136,0.08)" stroke="rgba(94,234,212,0.2)" strokeWidth="1" />
                <text x="90" y="76" textAnchor="middle" fontSize="8" fill="#a7f3d0" fontFamily="Comfortaa">Project A</text>
                <rect x="165" y="60" width="90" height="24" rx="8" fill="rgba(13,148,136,0.08)" stroke="rgba(94,234,212,0.2)" strokeWidth="1" />
                <text x="210" y="76" textAnchor="middle" fontSize="8" fill="#a7f3d0" fontFamily="Comfortaa">Project B</text>
                <line x1="70" y1="84" x2="70" y2="100" stroke="rgba(94,234,212,0.1)" strokeWidth="1" />
                <line x1="110" y1="84" x2="110" y2="100" stroke="rgba(94,234,212,0.1)" strokeWidth="1" />
                <rect x="42" y="100" width="56" height="20" rx="6" fill="rgba(8,20,32,0.6)" stroke="rgba(94,234,212,0.1)" strokeWidth="0.75" />
                <text x="70" y="114" textAnchor="middle" fontSize="7" fill="rgba(178,245,234,0.4)" fontFamily="JetBrains Mono">Sessions</text>
                <rect x="82" y="100" width="56" height="20" rx="6" fill="rgba(8,20,32,0.6)" stroke="rgba(94,234,212,0.1)" strokeWidth="0.75" />
                <text x="110" y="114" textAnchor="middle" fontSize="7" fill="rgba(178,245,234,0.4)" fontFamily="JetBrains Mono">Tokens</text>
                <text x="150" y="150" textAnchor="middle" fontSize="7" fill="rgba(94,234,212,0.2)" fontFamily="JetBrains Mono" letterSpacing="1">SCOPED GOVERNANCE</text>
              </svg>
            </FeatureDiagram>
            <FeatureContent>
              <h3>Governed by design</h3>
              <ul>
                <li>Organization-level ownership with project-scoped isolation</li>
                <li>Token-based access control per tool and project</li>
                <li>Full audit timeline for every memory mutation</li>
              </ul>
            </FeatureContent>
          </FeatureRow>
        </AnimatedSvgDiagram>
      </DeepDiveGrid>
    </MarketingSection>
  )
}

function ProductCTA() {
  return (
    <CtaBand>
      <h2>Ready to give your agents memory?</h2>
      <p>Start flowing in under two minutes.</p>
      <HeroCtas>
        <Btn to={MARKETING_ROUTES.login} $primary>Start Flowing</Btn>
        <Btn to={MARKETING_ROUTES.pricing}>See Pricing</Btn>
      </HeroCtas>
    </CtaBand>
  )
}

export function ProductPage() {
  return (
    <Page>
      <ProductHero />
      <ProductCapabilityPillars />
      <WaveDivider />
      <ProductDeepDive />
      <ProductCTA />
    </Page>
  )
}
