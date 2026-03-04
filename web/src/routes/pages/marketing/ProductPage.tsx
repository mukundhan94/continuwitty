import { MarketingPageTemplate } from './MarketingPageTemplate'

export function ProductPage() {
  return (
    <MarketingPageTemplate
      eyebrow="Product"
      title="One continuity system across chat, memory graph, and document context."
      lead="ContinuWitty combines session continuity, retrieval quality, link intelligence, and admin controls in one operational surface for humans and AI agents."
      sideTitle="Capability pillars"
      points={[
        'Chat continuity with save-and-continue loops.',
        'Engram graph links for richer context traversal.',
        'Document ingestion and pinning per session.',
        'Admin memory curation, contradiction handling, and observability.',
      ]}
    />
  )
}
