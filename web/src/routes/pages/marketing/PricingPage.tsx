import { NavLink } from 'react-router-dom'
import styled, { css, keyframes } from 'styled-components'

import { CheckIcon } from './components/FeatureIcon'
import { PRICING_PLAN_TEASERS } from '../../constants'
import { MARKETING_ROUTES } from '../../constants'

/* ------------------------------------------------------------------ */
/*  Keyframes                                                         */
/* ------------------------------------------------------------------ */

const cardFloat = keyframes`
  0%, 100% { transform: scale(1.04) translateY(0); }
  50%      { transform: scale(1.04) translateY(-4px); }
`

/* ------------------------------------------------------------------ */
/*  Hero                                                              */
/* ------------------------------------------------------------------ */

const HeroSection = styled.section`
  text-align: center;
  padding: 3.5rem 1.5rem 1.5rem;

  @media (max-width: 768px) {
    padding: 2.5rem 1rem 1rem;
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
`

const HeroLead = styled.p`
  margin: 1rem auto 0;
  max-width: 60ch;
  color: rgba(178, 245, 234, 0.76);
  line-height: 1.7;
  font-size: 0.95rem;
`

/* ------------------------------------------------------------------ */
/*  Plan Cards Grid                                                   */
/* ------------------------------------------------------------------ */

const PlansGrid = styled.div`
  display: grid;
  grid-template-columns: repeat(3, 1fr);
  gap: 0.8rem;
  align-items: start;

  @media (max-width: 900px) {
    grid-template-columns: 1fr;
    max-width: 440px;
    margin-inline: auto;
  }
`

const accentGradients: Record<string, string> = {
  starter: 'linear-gradient(90deg, rgba(94,234,212,0.4), rgba(6,182,212,0.3))',
  team: 'linear-gradient(90deg, #06b6d4, #14b8a6, #5eead4)',
  enterprise: 'linear-gradient(90deg, rgba(6,182,212,0.4), rgba(94,234,212,0.3))',
}

const PlanCard = styled.article<{ $planId: string }>`
  position: relative;
  border-radius: 20px;
  border: 1px solid ${(p) =>
    p.$planId === 'team' ? 'rgba(94,234,212,0.35)' : 'rgba(94,234,212,0.14)'};
  background: rgba(8, 20, 32, ${(p) => (p.$planId === 'team' ? '0.72' : '0.56')});
  padding: 0;
  overflow: hidden;
  transform: ${(p) => (p.$planId === 'team' ? 'scale(1.04)' : 'none')};
  box-shadow: ${(p) =>
    p.$planId === 'team'
      ? '0 0 40px rgba(94,234,212,0.08), 0 20px 60px rgba(0,0,0,0.25)'
      : '0 8px 30px rgba(0,0,0,0.15)'};

  ${(p) =>
    p.$planId === 'team' &&
    css`
      @media (prefers-reduced-motion: no-preference) {
        animation: ${cardFloat} 5s ease-in-out infinite;
      }
    `}
`

const AccentBar = styled.div<{ $planId: string }>`
  height: 3px;
  background: ${(p) => accentGradients[p.$planId] || accentGradients.starter};
`

const CardContent = styled.div`
  padding: 1.2rem 1rem;
`

const Badge = styled.span`
  display: inline-block;
  margin-bottom: 0.5rem;
  font-family: var(--font-mono);
  font-size: 0.6rem;
  letter-spacing: 0.12em;
  text-transform: uppercase;
  color: #001821;
  background: linear-gradient(135deg, #06b6d4, #5eead4);
  padding: 0.2rem 0.55rem;
  border-radius: 999px;
  font-weight: 700;
`

const PlanName = styled.h2`
  margin: 0;
  font-family: var(--font-display);
  font-size: 1.2rem;
  color: #f2fffd;
`

const PlanHeadline = styled.h3`
  margin: 0.3rem 0 0;
  font-size: 0.88rem;
  font-weight: 500;
  color: #c9f4ec;
  line-height: 1.45;
`

const PlanAudience = styled.p`
  margin: 0.5rem 0 0;
  font-family: var(--font-mono);
  font-size: 0.68rem;
  letter-spacing: 0.06em;
  text-transform: uppercase;
  color: rgba(94, 234, 212, 0.4);
`

const FeatureList = styled.ul`
  margin: 0.8rem 0 0;
  padding: 0;
  list-style: none;
  display: grid;
  gap: 0.4rem;
`

const FeatureItem = styled.li`
  display: flex;
  align-items: center;
  gap: 0.45rem;
  font-size: 0.82rem;
  color: #d7f9f3;
  line-height: 1.4;

  svg {
    flex-shrink: 0;
    width: 14px;
    height: 14px;
  }
`

const PlanCta = styled.button<{ $planId: string }>`
  margin-top: 1rem;
  width: 100%;
  border-radius: 999px;
  padding: 0.55rem;
  font-weight: 700;
  font-size: 0.82rem;
  letter-spacing: 0.04em;
  border: none;
  cursor: pointer;
  transition: opacity 150ms ease;

  color: ${(p) => (p.$planId === 'team' ? '#001821' : '#dffcf8')};
  background: ${(p) =>
    p.$planId === 'team'
      ? 'linear-gradient(135deg, #06b6d4, #14b8a6, #5eead4)'
      : 'rgba(94,234,212,0.1)'};
  border: ${(p) =>
    p.$planId === 'team' ? 'none' : '1px solid rgba(94,234,212,0.25)'};

  &:hover {
    opacity: 0.85;
  }
`

/* ------------------------------------------------------------------ */
/*  Comparison Teaser                                                 */
/* ------------------------------------------------------------------ */

const ComparisonTeaser = styled.section`
  text-align: center;
  padding: 2rem 1.5rem;
  border-radius: 22px;
  border: 1px solid rgba(94, 234, 212, 0.1);
  background: rgba(8, 20, 32, 0.4);

  h3 {
    margin: 0;
    font-family: var(--font-display);
    font-size: 1rem;
    color: rgba(178, 245, 234, 0.6);
  }

  p {
    margin: 0.5rem auto 0;
    color: rgba(178, 245, 234, 0.4);
    font-size: 0.82rem;
    max-width: 40ch;
  }
`

const TableOutline = styled.div`
  margin: 1.2rem auto 0;
  max-width: 500px;
  display: grid;
  grid-template-columns: 1fr 1fr 1fr;
  gap: 1px;
  border-radius: 12px;
  overflow: hidden;
  border: 1px solid rgba(94, 234, 212, 0.06);
`

const TableCell = styled.div<{ $header?: boolean }>`
  height: ${(p) => (p.$header ? '28px' : '22px')};
  background: ${(p) =>
    p.$header ? 'rgba(94,234,212,0.04)' : 'rgba(8,20,32,0.3)'};
  border-bottom: 1px solid rgba(94, 234, 212, 0.04);
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
/*  Page Shell                                                        */
/* ------------------------------------------------------------------ */

const PageShell = styled.div`
  display: grid;
  gap: 1.2rem;
`

/* ------------------------------------------------------------------ */
/*  Component                                                         */
/* ------------------------------------------------------------------ */

export function PricingPage() {
  return (
    <PageShell>
      {/* 1. Pricing Hero */}
      <HeroSection>
        <Eyebrow>Pricing</Eyebrow>
        <HeroTitle>Plans for every scale of agent memory.</HeroTitle>
        <HeroLead>
          We are finalizing usage-based plans for individuals, teams, and enterprise workloads.
          Start exploring now — your procurement and rollout conversations can begin early.
        </HeroLead>
      </HeroSection>

      {/* 2. Plan Cards */}
      <PlansGrid>
        {PRICING_PLAN_TEASERS.map((plan) => (
          <PlanCard key={plan.id} $planId={plan.id}>
            <AccentBar $planId={plan.id} />
            <CardContent>
              {plan.id === 'team' && <Badge>Recommended</Badge>}
              <PlanName>{plan.name}</PlanName>
              <PlanHeadline>{plan.headline}</PlanHeadline>
              <PlanAudience>{plan.audience}</PlanAudience>
              <FeatureList>
                {plan.points.map((point) => (
                  <FeatureItem key={point}>
                    <CheckIcon />
                    {point}
                  </FeatureItem>
                ))}
              </FeatureList>
              <PlanCta $planId={plan.id} type="button">{plan.cta}</PlanCta>
            </CardContent>
          </PlanCard>
        ))}
      </PlansGrid>

      {/* 3. Comparison Teaser */}
      <ComparisonTeaser>
        <h3>Detailed feature comparison coming soon.</h3>
        <p>We are building a full comparison matrix so you can evaluate plans side by side.</p>
        <TableOutline>
          {/* Header row */}
          <TableCell $header />
          <TableCell $header />
          <TableCell $header />
          {/* 4 empty rows to suggest a table */}
          {Array.from({ length: 12 }, (_, i) => (
            <TableCell key={i} />
          ))}
        </TableOutline>
      </ComparisonTeaser>

      {/* 4. CTA */}
      <CtaBand>
        <h2>Not sure which plan fits?</h2>
        <p>Talk to our team and we will help you find the right level of agent memory for your workload.</p>
        <CtaRow>
          <PrimaryBtn to={MARKETING_ROUTES.login}>Get Started</PrimaryBtn>
          <GhostBtn to={MARKETING_ROUTES.forEnterprise}>Enterprise Details</GhostBtn>
        </CtaRow>
      </CtaBand>
    </PageShell>
  )
}
