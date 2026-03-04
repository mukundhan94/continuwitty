import { NavLink, Outlet } from 'react-router-dom'
import styled, { keyframes } from 'styled-components'

import { MemoryStrandMark } from '../../components/MemoryStrandMark'
import { MARKETING_ROUTES, MARKETING_TOP_NAV } from '../constants'

const drift = keyframes`
  0% {
    transform: translateX(0);
  }
  100% {
    transform: translateX(-50%);
  }
`

const Shell = styled.div`
  min-height: 100vh;
  background:
    radial-gradient(circle at 20% 18%, rgba(94, 234, 212, 0.1), transparent 35%),
    radial-gradient(circle at 84% 78%, rgba(14, 116, 144, 0.24), transparent 42%),
    linear-gradient(170deg, #081420 0%, #0b1f31 55%, #0c2d48 100%);
  color: var(--color-ink);
  overflow: hidden;
`

const TopBar = styled.header`
  position: sticky;
  top: 0;
  z-index: 25;
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 1rem;
  padding: 0.8rem 1rem;
  border-bottom: 1px solid rgba(94, 234, 212, 0.14);
  backdrop-filter: blur(12px);
  background: rgba(8, 20, 32, 0.72);
`

const Brand = styled(NavLink)`
  display: inline-flex;
  align-items: center;
  gap: 0.5rem;
  text-decoration: none;
  color: #e8fffb;

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

const TopNav = styled.nav`
  display: flex;
  gap: 0.3rem;
  flex-wrap: wrap;

  a {
    text-decoration: none;
    color: var(--color-ink-muted);
    border-radius: 999px;
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
`

const LoginLink = styled(NavLink)`
  text-decoration: none;
  color: #001822;
  border-radius: 999px;
  padding: 0.4rem 0.95rem;
  font-weight: 700;
  font-size: 0.8rem;
  letter-spacing: 0.03em;
  background: linear-gradient(135deg, #06b6d4, #14b8a6, #5eead4);
  box-shadow: 0 8px 30px rgba(94, 234, 212, 0.24);
`

const Content = styled.main`
  position: relative;
  padding: 1.25rem;
`

const WaveBand = styled.div`
  position: absolute;
  left: 0;
  right: 0;
  bottom: 0;
  height: 44px;
  pointer-events: none;
  opacity: 0.58;

  &::before,
  &::after {
    content: '';
    position: absolute;
    inset: 0;
    width: 200%;
    background-repeat: repeat-x;
    background-size: 120px 44px;
    border-top: 1px solid rgba(94, 234, 212, 0.12);
    animation: ${drift} 12s linear infinite;
  }

  &::before {
    background-image: radial-gradient(80px 30px at 50% 120%, rgba(94, 234, 212, 0.2), transparent 70%);
  }

  &::after {
    animation-duration: 18s;
    animation-direction: reverse;
    background-image: radial-gradient(80px 30px at 50% 120%, rgba(6, 182, 212, 0.18), transparent 70%);
    opacity: 0.75;
  }
`

const BrandCopy = styled.div`
  line-height: 1.1;
`

export function MarketingLayout() {
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
        <TopNav>
          {MARKETING_TOP_NAV.map((item) => (
            <NavLink key={item.to} to={item.to}>
              {item.label}
            </NavLink>
          ))}
        </TopNav>
        <LoginLink to={MARKETING_ROUTES.login}>Start Flowing</LoginLink>
      </TopBar>
      <Content>
        <Outlet />
      </Content>
      <WaveBand />
    </Shell>
  )
}
