import styled from 'styled-components'

import { MarketingPageTemplate } from './MarketingPageTemplate'

const Steps = styled.section`
  margin-top: 0.85rem;
  border-radius: 18px;
  border: 1px solid rgba(94, 234, 212, 0.16);
  background: rgba(8, 20, 32, 0.55);
  padding: 0.9rem;
  display: grid;
  gap: 0.55rem;
`

const Step = styled.article`
  border-left: 2px solid rgba(94, 234, 212, 0.4);
  padding-left: 0.6rem;

  h3 {
    margin: 0;
    font-size: 0.86rem;
  }

  p {
    margin: 0.26rem 0 0;
    color: var(--color-ink-muted);
    line-height: 1.55;
    font-size: 0.8rem;
  }
`

export function HowItWorksPage() {
  return (
    <>
      <MarketingPageTemplate
        eyebrow="How it works"
        title="From conversation to durable memory in a single operating loop."
        lead="Use this product loop to keep your agents and teams aligned: create session, gather context, save milestone memory, and continue with pinned continuity artifacts."
        sideTitle="Who this helps"
        points={[
          'AI operators handling long-running workflows.',
          'Product teams preserving decision traceability.',
          'Engineering teams reducing rediscovery work.',
          'Security teams enforcing scoped memory access.',
        ]}
      />

      <Steps>
        <Step>
          <h3>Step 1: Start a session</h3>
          <p>Create a scoped chat session in your project and set model/provider controls.</p>
        </Step>
        <Step>
          <h3>Step 2: Blend context</h3>
          <p>Pull engrams, ingest documents, and pin relevant artifacts for the active conversation.</p>
        </Step>
        <Step>
          <h3>Step 3: Save memory</h3>
          <p>Convert the session result into an engram with abstract, tags, and visibility metadata.</p>
        </Step>
        <Step>
          <h3>Step 4: Continue with confidence</h3>
          <p>Start a new session while preserving continuity through pinned engrams and traceable sources.</p>
        </Step>
      </Steps>
    </>
  )
}
