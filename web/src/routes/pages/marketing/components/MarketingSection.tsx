import type { ReactNode } from 'react'
import styled, { keyframes } from 'styled-components'

const waveShift = keyframes`
  from { transform: translateX(0); }
  to { transform: translateX(-50%); }
`

const Shell = styled.section`
  position: relative;
  border-radius: 24px;
  border: 1px solid rgba(94, 234, 212, 0.11);
  background: rgba(8, 20, 32, 0.6);
  backdrop-filter: blur(10px);
  padding: 3rem 2rem;
  overflow: hidden;

  @media (max-width: 768px) {
    padding: 2rem 1.2rem;
  }
`

const Eyebrow = styled.p`
  margin: 0 0 0.75rem;
  font-family: var(--font-mono);
  font-size: 0.625rem;
  letter-spacing: 0.25em;
  text-transform: uppercase;
  color: rgba(94, 234, 212, 0.35);
  display: flex;
  align-items: center;
  gap: 0.6rem;

  &::before {
    content: '';
    display: block;
    width: 20px;
    height: 1px;
    background: rgba(94, 234, 212, 0.25);
  }
`

const Heading = styled.h2`
  margin: 0;
  font-family: var(--font-display);
  font-weight: 700;
  font-size: clamp(1.5rem, 3.5vw, 2.2rem);
  color: #f7fffd;
  line-height: 1.15;
`

const Lead = styled.p`
  margin: 0.75rem 0 0;
  max-width: 56ch;
  color: rgba(178, 245, 234, 0.76);
  line-height: 1.7;
  font-size: 0.95rem;
`

const WaveDecoration = styled.div`
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

  @media (prefers-reduced-motion: reduce) {
    &::before, &::after {
      animation: none;
    }
  }
`

const Content = styled.div`
  position: relative;
  z-index: 1;
`

interface MarketingSectionProps {
  eyebrow?: string
  title: string
  lead?: string
  showWave?: boolean
  className?: string
  children?: ReactNode
}

export function MarketingSection({
  eyebrow,
  title,
  lead,
  showWave = false,
  className,
  children,
}: MarketingSectionProps) {
  return (
    <Shell className={className}>
      <Content>
        {eyebrow && <Eyebrow>{eyebrow}</Eyebrow>}
        <Heading>{title}</Heading>
        {lead && <Lead>{lead}</Lead>}
        {children}
      </Content>
      {showWave && <WaveDecoration />}
    </Shell>
  )
}
