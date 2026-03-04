import type { ReactNode } from 'react'
import styled from 'styled-components'

const Container = styled.main`
  flex: 1;
  min-height: 0;
  display: grid;
  grid-template-rows: auto minmax(0, 1fr);
  align-content: start;
  gap: 0.7rem;
`

const Header = styled.header`
  border: 1px solid var(--surface-glass-border);
  border-radius: 18px;
  padding: 0.75rem 0.95rem;
  background: var(--surface-glass);
  backdrop-filter: blur(10px);
  box-shadow: var(--shadow-panel);
  display: grid;
  gap: 0.2rem;
`

const Title = styled.h2`
  margin: 0;
  font-family: var(--font-display);
  font-size: 1rem;
  letter-spacing: 0.02em;
`

const Description = styled.p`
  margin: 0;
  color: var(--color-ink-muted);
  font-size: 0.82rem;
`

interface AppLayoutProps {
  title: string
  description: string
  children: ReactNode
}

export function AppLayout({ title, description, children }: AppLayoutProps) {
  return (
    <Container>
      <Header>
        <Title>{title}</Title>
        <Description>{description}</Description>
      </Header>
      {children}
    </Container>
  )
}
