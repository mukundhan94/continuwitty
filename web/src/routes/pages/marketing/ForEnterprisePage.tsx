import { MarketingPageTemplate } from './MarketingPageTemplate'

export function ForEnterprisePage() {
  return (
    <MarketingPageTemplate
      eyebrow="For Enterprise"
      title="Operational memory with governance, policy boundaries, and auditability."
      lead="Deploy memory continuity with role-aware project controls, scoped MCP tokens, and audit trails that security and platform teams can trust."
      sideTitle="Enterprise outcomes"
      points={[
        'Project-level ownership and role management.',
        'Auditable memory mutations and access timelines.',
        'Token scopes for tool and project restrictions.',
        'Admin controls for session and memory lifecycle operations.',
      ]}
    />
  )
}
