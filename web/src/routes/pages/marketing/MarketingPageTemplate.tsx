import { NavLink } from 'react-router-dom'
import styled, { keyframes } from 'styled-components'

import { MemoryStrandMark } from '../../../components/MemoryStrandMark'
import { MARKETING_ROUTES } from '../../constants'

const reveal = keyframes`
  from {
    opacity: 0;
    transform: translateY(14px);
  }
  to {
    opacity: 1;
    transform: translateY(0);
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

const Grid = styled.section`
  display: grid;
  grid-template-columns: minmax(0, 1.3fr) minmax(280px, 1fr);
  gap: 1rem;

  @media (max-width: 1060px) {
    grid-template-columns: 1fr;
  }
`

const HeroCard = styled.article`
  position: relative;
  overflow: hidden;
  border-radius: 24px;
  border: 1px solid rgba(94, 234, 212, 0.2);
  background: rgba(8, 20, 32, 0.6);
  padding: 1.2rem;
  box-shadow: 0 28px 80px rgba(0, 0, 0, 0.3);
  animation: ${reveal} 480ms ease;
`

const Eyebrow = styled.p`
  margin: 0;
  text-transform: uppercase;
  letter-spacing: 0.22em;
  color: rgba(178, 245, 234, 0.7);
  font-size: 0.72rem;
`

const Title = styled.h1`
  margin: 0.45rem 0 0;
  font-size: clamp(1.9rem, 3.2vw, 3.2rem);
  line-height: 1.06;
  color: #f2fffd;
  font-family: var(--font-display);
`

const Lead = styled.p`
  margin: 0.7rem 0 0;
  color: rgba(201, 244, 236, 0.88);
  max-width: 70ch;
  line-height: 1.65;
`

const ActionRow = styled.div`
  margin-top: 1.1rem;
  display: flex;
  flex-wrap: wrap;
  gap: 0.6rem;
`

const Primary = styled(NavLink)`
  text-decoration: none;
  border-radius: 999px;
  padding: 0.58rem 1rem;
  font-weight: 800;
  font-size: 0.82rem;
  letter-spacing: 0.03em;
  color: #001821;
  background: linear-gradient(135deg, #06b6d4, #14b8a6, #5eead4);
`

const Ghost = styled(NavLink)`
  text-decoration: none;
  border-radius: 999px;
  padding: 0.58rem 1rem;
  font-weight: 700;
  font-size: 0.82rem;
  letter-spacing: 0.03em;
  color: #dffcf8;
  border: 1px solid rgba(94, 234, 212, 0.35);
  background: rgba(94, 234, 212, 0.08);
`

const SideCard = styled.aside`
  border-radius: 20px;
  border: 1px solid rgba(94, 234, 212, 0.16);
  background: rgba(8, 20, 32, 0.58);
  padding: 0.95rem;
  display: grid;
  align-content: start;
  gap: 0.6rem;
  animation: ${reveal} 540ms ease;
`

const SideTitle = styled.h2`
  margin: 0;
  font-family: var(--font-display);
  font-size: 1.05rem;
`

const SideList = styled.ul`
  margin: 0;
  padding-left: 1.1rem;
  display: grid;
  gap: 0.35rem;
  color: rgba(201, 244, 236, 0.88);
  line-height: 1.5;
`

interface MarketingPageTemplateProps {
  eyebrow: string
  title: string
  lead: string
  points: string[]
  sideTitle: string
}

const WaveBand = styled.div`
  position: absolute;
  inset: auto 0 0 0;
  height: 36px;
  pointer-events: none;
  opacity: 0.6;

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
    background-image: radial-gradient(48px 18px at 50% 120%, rgba(94, 234, 212, 0.22), transparent 74%);
    animation: ${waveShift} 10s linear infinite;
  }

  &::after {
    background-image: radial-gradient(48px 18px at 50% 120%, rgba(6, 182, 212, 0.2), transparent 74%);
    animation: ${waveShift} 15s linear infinite reverse;
  }
`

const HeroTop = styled.div`
  display: flex;
  align-items: center;
  gap: 0.55rem;
  margin-bottom: 0.35rem;
`

export function MarketingPageTemplate({
  eyebrow,
  title,
  lead,
  points,
  sideTitle,
}: MarketingPageTemplateProps) {
  return (
    <Grid>
      <HeroCard>
        <HeroTop>
          <MemoryStrandMark size="sm" />
          <Eyebrow>{eyebrow}</Eyebrow>
        </HeroTop>
        <Title>{title}</Title>
        <Lead>{lead}</Lead>
        <ActionRow>
          <Primary to={MARKETING_ROUTES.login}>Start Flowing</Primary>
          <Ghost to={MARKETING_ROUTES.howItWorks}>See Workflow</Ghost>
        </ActionRow>
        <WaveBand />
      </HeroCard>
      <SideCard>
        <SideTitle>{sideTitle}</SideTitle>
        <SideList>
          {points.map((point) => (
            <li key={point}>{point}</li>
          ))}
        </SideList>
      </SideCard>
    </Grid>
  )
}
