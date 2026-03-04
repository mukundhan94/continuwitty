import { NavLink } from 'react-router-dom'
import styled, { keyframes } from 'styled-components'

import { MemoryStrandMark } from '../../../../components/MemoryStrandMark'
import { MARKETING_ROUTES } from '../../../constants'

const drift = keyframes`
  from { transform: translateX(0); }
  to { transform: translateX(-50%); }
`

const Shell = styled.footer`
  position: relative;
  background: rgba(4, 10, 18, 0.95);
  border-top: 1px solid rgba(94, 234, 212, 0.08);
  padding: 4rem 2rem 2rem;

  @media (prefers-reduced-motion: reduce) {
    * { animation: none !important; }
  }
`

const WaveSep = styled.div`
  position: absolute;
  top: -40px;
  left: 0;
  right: 0;
  height: 40px;
  pointer-events: none;
  overflow: hidden;

  &::before,
  &::after {
    content: '';
    position: absolute;
    inset: 0;
    width: 200%;
    background-repeat: repeat-x;
    background-size: 120px 40px;
  }

  &::before {
    background-image: radial-gradient(80px 28px at 50% 120%, rgba(94, 234, 212, 0.18), transparent 70%);
    animation: ${drift} 12s linear infinite;
  }

  &::after {
    background-image: radial-gradient(80px 28px at 50% 120%, rgba(6, 182, 212, 0.14), transparent 70%);
    animation: ${drift} 18s linear infinite reverse;
    opacity: 0.75;
  }
`

const Grid = styled.div`
  max-width: 1100px;
  margin: 0 auto;
  display: grid;
  grid-template-columns: 1.5fr repeat(3, 1fr);
  gap: 2rem;

  @media (max-width: 768px) {
    grid-template-columns: 1fr 1fr;
    gap: 1.5rem;
  }

  @media (max-width: 480px) {
    grid-template-columns: 1fr;
  }
`

const BrandCol = styled.div`
  display: flex;
  flex-direction: column;
  gap: 0.5rem;
`

const BrandName = styled.div`
  font-family: var(--font-display);
  font-weight: 700;
  font-size: 1.1rem;
  color: #e8fffb;
  margin-top: 0.5rem;
`

const BrandTag = styled.div`
  font-family: var(--font-mono);
  font-size: 0.6rem;
  letter-spacing: 0.2em;
  text-transform: uppercase;
  color: rgba(167, 243, 208, 0.4);
`

const ColTitle = styled.h4`
  margin: 0 0 0.75rem;
  font-family: var(--font-mono);
  font-size: 0.6rem;
  letter-spacing: 0.2em;
  text-transform: uppercase;
  color: rgba(94, 234, 212, 0.35);
`

const ColLinks = styled.div`
  display: flex;
  flex-direction: column;
  gap: 0.5rem;

  a {
    text-decoration: none;
    font-size: 0.85rem;
    color: rgba(178, 245, 234, 0.5);
    transition: color 200ms ease;

    &:hover {
      color: #5eead4;
    }
  }
`

const Bottom = styled.div`
  max-width: 1100px;
  margin: 2rem auto 0;
  padding-top: 1.5rem;
  border-top: 1px solid rgba(94, 234, 212, 0.06);
  display: flex;
  justify-content: space-between;
  align-items: center;
  font-family: var(--font-mono);
  font-size: 0.6rem;
  letter-spacing: 0.08em;
  color: rgba(94, 234, 212, 0.15);

  @media (max-width: 480px) {
    flex-direction: column;
    gap: 0.5rem;
    text-align: center;
  }
`

export function MarketingFooter() {
  return (
    <Shell>
      <WaveSep />
      <Grid>
        <BrandCol>
          <MemoryStrandMark size="sm" animated={false} />
          <BrandName>ContinuWitty</BrandName>
          <BrandTag>Intelligence that flows</BrandTag>
        </BrandCol>

        <div>
          <ColTitle>Product</ColTitle>
          <ColLinks>
            <NavLink to={MARKETING_ROUTES.product}>Product</NavLink>
            <NavLink to={MARKETING_ROUTES.howItWorks}>How It Works</NavLink>
            <NavLink to={MARKETING_ROUTES.pricing}>Pricing</NavLink>
          </ColLinks>
        </div>

        <div>
          <ColTitle>Use Cases</ColTitle>
          <ColLinks>
            <NavLink to={MARKETING_ROUTES.forEnterprise}>For Enterprise</NavLink>
            <NavLink to={MARKETING_ROUTES.forDevelopers}>For Developers</NavLink>
          </ColLinks>
        </div>

        <div>
          <ColTitle>Get Started</ColTitle>
          <ColLinks>
            <NavLink to={MARKETING_ROUTES.login}>Login</NavLink>
          </ColLinks>
        </div>
      </Grid>

      <Bottom>
        <span>ContinuWitty — Memory that flows</span>
        <span>Two strands, one thought</span>
      </Bottom>
    </Shell>
  )
}
