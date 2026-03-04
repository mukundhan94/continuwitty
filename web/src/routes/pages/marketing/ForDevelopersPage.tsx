import { MarketingPageTemplate } from './MarketingPageTemplate'

export function ForDevelopersPage() {
  return (
    <MarketingPageTemplate
      eyebrow="For Developers"
      title="Build agents that remember decisions, not just prompts."
      lead="Use REST and MCP workflows to create sessions, persist milestones as engrams, query prior context, and continue work with deterministic continuity."
      sideTitle="Developer workflow"
      points={[
        'Session APIs for long-running conversations.',
        'Engram query + rehydrate for durable context.',
        'MCP stream integration for agent orchestration.',
        'Project export/import for portable memory bundles.',
      ]}
    />
  )
}
