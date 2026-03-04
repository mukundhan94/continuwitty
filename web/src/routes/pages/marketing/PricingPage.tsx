import styled from 'styled-components'

import { PRICING_PLAN_TEASERS } from '../../constants'

const Shell = styled.section`
  display: grid;
  gap: 0.8rem;
`

const Header = styled.article`
  border-radius: 22px;
  border: 1px solid rgba(94, 234, 212, 0.2);
  background: rgba(8, 20, 32, 0.62);
  padding: 1rem;

  h1 {
    margin: 0;
    font-family: var(--font-display);
    font-size: clamp(1.7rem, 2.6vw, 2.8rem);
  }

  p {
    margin: 0.6rem 0 0;
    color: var(--color-ink-muted);
    max-width: 72ch;
    line-height: 1.65;
  }
`

const Grid = styled.div`
  display: grid;
  grid-template-columns: repeat(3, minmax(0, 1fr));
  gap: 0.7rem;

  @media (max-width: 980px) {
    grid-template-columns: 1fr;
  }
`

const Card = styled.article`
  border-radius: 18px;
  border: 1px solid rgba(94, 234, 212, 0.17);
  background: rgba(8, 20, 32, 0.56);
  padding: 0.85rem;

  h2 {
    margin: 0;
    font-family: var(--font-display);
    font-size: 1.1rem;
  }

  h3 {
    margin: 0.35rem 0 0;
    font-size: 0.9rem;
    color: #c9f4ec;
    line-height: 1.45;
  }

  p {
    margin: 0.45rem 0 0;
    color: var(--color-ink-muted);
    font-size: 0.82rem;
  }

  ul {
    margin: 0.6rem 0 0;
    padding-left: 1.1rem;
    display: grid;
    gap: 0.3rem;
    color: #d7f9f3;
    font-size: 0.82rem;
  }

  button {
    margin-top: 0.7rem;
    width: 100%;
    border-radius: 999px;
    font-weight: 700;
    letter-spacing: 0.04em;
  }
`

export function PricingPage() {
  return (
    <Shell>
      <Header>
        <h1>Pricing plans are coming soon.</h1>
        <p>
          We are finalizing usage-based plans for individuals, teams, and enterprise workloads. The
          pricing surface is scaffolded now so your procurement and rollout conversations can start early.
        </p>
      </Header>

      <Grid>
        {PRICING_PLAN_TEASERS.map((plan) => (
          <Card key={plan.id}>
            <h2>{plan.name}</h2>
            <h3>{plan.headline}</h3>
            <p>{plan.audience}</p>
            <ul>
              {plan.points.map((point) => (
                <li key={point}>{point}</li>
              ))}
            </ul>
            <button type="button">{plan.cta}</button>
          </Card>
        ))}
      </Grid>
    </Shell>
  )
}
