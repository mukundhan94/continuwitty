import type { ReactNode } from 'react'
import styled from 'styled-components'

const Shell = styled.article`
  border-radius: 18px;
  border: 1px solid rgba(94, 234, 212, 0.12);
  background: linear-gradient(160deg, rgba(8, 20, 32, 0.8), rgba(12, 53, 71, 0.5));
  backdrop-filter: blur(12px);
  padding: 1.5rem;
  transition: transform 300ms ease, border-color 300ms ease, box-shadow 300ms ease;

  &:hover {
    transform: translateY(-3px);
    border-color: rgba(94, 234, 212, 0.22);
    box-shadow: 0 12px 40px rgba(0, 0, 0, 0.25);
  }
`

const IconWrap = styled.div`
  width: 52px;
  height: 52px;
  border-radius: 14px;
  background: linear-gradient(135deg, rgba(6, 182, 212, 0.15), rgba(13, 148, 136, 0.1));
  border: 1px solid rgba(94, 234, 212, 0.12);
  display: flex;
  align-items: center;
  justify-content: center;
  margin-bottom: 1rem;
  color: #5eead4;
`

const Title = styled.h3`
  margin: 0;
  font-family: var(--font-display);
  font-weight: 600;
  font-size: 1rem;
  color: #f2fffd;
`

const Description = styled.p`
  margin: 0.4rem 0 0;
  color: rgba(201, 244, 236, 0.8);
  line-height: 1.6;
  font-size: 0.85rem;
`

interface GlassCardProps {
  icon?: ReactNode
  title: string
  description: string
  children?: ReactNode
  className?: string
}

export function GlassCard({ icon, title, description, children, className }: GlassCardProps) {
  return (
    <Shell className={className}>
      {icon && <IconWrap>{icon}</IconWrap>}
      <Title>{title}</Title>
      <Description>{description}</Description>
      {children}
    </Shell>
  )
}
