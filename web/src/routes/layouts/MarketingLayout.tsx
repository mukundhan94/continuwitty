import { useState } from 'react'
import { NavLink, Outlet } from 'react-router-dom'
import styled from 'styled-components'

import { MemoryStrandMark } from '../../components/MemoryStrandMark'
import { MARKETING_ROUTES, MARKETING_TOP_NAV } from '../constants'
import { MarketingFooter } from '../pages/marketing/components/MarketingFooter'

const Shell = styled.div`
  min-height: 100vh;
  display: flex;
  flex-direction: column;
  background:
    radial-gradient(circle at 20% 18%, rgba(94, 234, 212, 0.1), transparent 35%),
    radial-gradient(circle at 84% 78%, rgba(14, 116, 144, 0.24), transparent 42%),
    linear-gradient(170deg, var(--color-bg-soft) 0%, var(--color-bg-strong) 100%);
  color: var(--color-ink);
  overflow-x: hidden;

  --mktg-deep: var(--color-bg-soft);
  --mktg-sea: var(--color-accent);
  --mktg-cyan: #06b6d4;
  --mktg-teal: #0d9488;
  --mktg-mint: #a7f3d0;
  --mktg-glow: #ccfbf1;
  --mktg-surface: var(--surface-glass);
  --mktg-border: var(--surface-glass-border);
`

const TopBar = styled.header`
  position: sticky;
  top: 0;
  z-index: 25;
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 1rem;
  padding: 0.8rem 1.25rem;
  border-bottom: 1px solid var(--surface-glass-border);
  backdrop-filter: blur(14px);
  background: var(--surface-glass);
  box-shadow: 0 4px 30px rgba(0, 0, 0, 0.3);
`

const Brand = styled(NavLink)`
  display: inline-flex;
  align-items: center;
  gap: 0.5rem;
  text-decoration: none;
  color: #e8fffb;
  flex-shrink: 0;

  strong {
    font-family: var(--font-display);
    font-size: 1.15rem;
    font-weight: 700;
    letter-spacing: 0.01em;
    display: block;
  }

  span {
    display: block;
    font-family: var(--font-mono);
    font-size: 0.58rem;
    letter-spacing: 0.2em;
    color: rgba(167, 243, 208, 0.72);
    text-transform: uppercase;
  }
`

const BrandCopy = styled.div`
  line-height: 1.1;
`

const TopNav = styled.nav<{ $open?: boolean }>`
  display: flex;
  gap: 0.3rem;
  flex-wrap: wrap;

  a {
    text-decoration: none;
    color: var(--color-ink-muted);
    border-radius: 9999px;
    padding: 0.35rem 0.7rem;
    border: 1px solid transparent;
    font-size: 0.82rem;
    letter-spacing: 0.02em;
    transition: all 200ms ease;
  }

  a:hover,
  a.active {
    color: #dffff9;
    border-color: rgba(94, 234, 212, 0.3);
    background: rgba(94, 234, 212, 0.12);
  }

  @media (max-width: 768px) {
    display: ${({ $open }) => ($open ? 'flex' : 'none')};
    position: absolute;
    top: 100%;
    left: 0;
    right: 0;
    flex-direction: column;
    padding: 0.75rem 1rem;
    background: rgba(8, 20, 32, 0.95);
    backdrop-filter: blur(14px);
    border-bottom: 1px solid rgba(94, 234, 212, 0.12);
    gap: 0.2rem;
  }
`

const HamburgerBtn = styled.button`
  display: none;
  background: none;
  border: 1px solid rgba(94, 234, 212, 0.2);
  border-radius: 8px;
  padding: 0.4rem;
  cursor: pointer;
  color: #5eead4;

  @media (max-width: 768px) {
    display: flex;
    align-items: center;
    justify-content: center;
  }
`

const LoginLink = styled(NavLink)`
  text-decoration: none;
  color: #001822;
  border-radius: 9999px;
  padding: 0.4rem 0.95rem;
  font-weight: 700;
  font-size: 0.8rem;
  letter-spacing: 0.03em;
  background: var(--cta-gradient);
  box-shadow: 0 8px 30px rgba(94, 234, 212, 0.24);
  flex-shrink: 0;
  transition: transform 200ms ease-out, box-shadow 200ms ease-out;

  &:hover {
    transform: translateY(-2px);
    box-shadow: 0 12px 36px rgba(94, 234, 212, 0.3);
  }
`

const Content = styled.main`
  position: relative;
  padding: 1.25rem;
  flex: 1;
`

export function MarketingLayout() {
  const [navOpen, setNavOpen] = useState(false)

  return (
    <Shell>
      <TopBar>
        <Brand to={MARKETING_ROUTES.landing}>
          <MemoryStrandMark size="sm" />
          <BrandCopy>
            <strong>ContinuWitty</strong>
            <span>Intelligence that flows</span>
          </BrandCopy>
        </Brand>

        <HamburgerBtn onClick={() => setNavOpen((v) => !v)} aria-label="Toggle navigation">
          <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round">
            {navOpen ? (
              <>
                <line x1="18" y1="6" x2="6" y2="18" />
                <line x1="6" y1="6" x2="18" y2="18" />
              </>
            ) : (
              <>
                <line x1="3" y1="6" x2="21" y2="6" />
                <line x1="3" y1="12" x2="21" y2="12" />
                <line x1="3" y1="18" x2="21" y2="18" />
              </>
            )}
          </svg>
        </HamburgerBtn>

        <TopNav $open={navOpen}>
          {MARKETING_TOP_NAV.map((item) => (
            <NavLink key={item.to} to={item.to} onClick={() => setNavOpen(false)}>
              {item.label}
            </NavLink>
          ))}
        </TopNav>

        <LoginLink to={MARKETING_ROUTES.login}>Start Flowing</LoginLink>
      </TopBar>

      <Content>
        <Outlet />
      </Content>

      <MarketingFooter />
    </Shell>
  )
}
