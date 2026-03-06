import type { BreadcrumbSpec, PricingPlanTeaser } from './types'

export const MARKETING_ROUTES = {
  landing: '/',
  forEnterprise: '/for-enterprise',
  forDevelopers: '/for-developers',
  product: '/product',
  howItWorks: '/how-it-works',
  pricing: '/pricing',
  login: '/login',
} as const

export const APP_ROUTES = {
  workspace: '/app/workspace',
  agents: '/app/agents',
  agentRunDetail: '/app/agents/runs/:threadId',
  sessions: '/app/sessions',
  sessionsNew: '/app/sessions/new',
  sessionChat: '/app/sessions/:sessionId/chat',
  sessionLifecycle: '/app/sessions/:sessionId/lifecycle',
  sessionTimeline: '/app/sessions/:sessionId/timeline',
  sessionSaveEngram: '/app/sessions/:sessionId/save-engram',
  sessionContinue: '/app/sessions/:sessionId/continue',
  engrams: '/app/engrams',
  engramsQuery: '/app/engrams/query',
  engramDetail: '/app/engrams/:engramId',
  engramSources: '/app/engrams/:engramId/sources',
  engramRehydrate: '/app/engrams/:engramId/rehydrate',
  engramLinks: '/app/engrams/:engramId/links',
  engramTrace: '/app/engrams/:engramId/trace',
  sessionPinsEngrams: '/app/sessions/:sessionId/pins/engrams',
  sessionPinsDocuments: '/app/sessions/:sessionId/pins/documents',
  documents: '/app/documents',
  documentsIngestText: '/app/documents/ingest/text',
  documentsIngestFile: '/app/documents/ingest/file',
  projects: '/app/projects',
  projectMembers: '/app/projects/:projectId/members',
  projectAudit: '/app/projects/:projectId/audit',
  transferExport: '/app/projects/transfer/export',
  transferImport: '/app/projects/transfer/import',
  adminSessions: '/app/admin/sessions',
  adminEngrams: '/app/admin/engrams',
  adminCollections: '/app/admin/collections',
  adminCuration: '/app/admin/curation',
  adminContradictions: '/app/admin/contradictions',
  adminTokens: '/app/admin/tokens',
  adminObservability: '/app/admin/observability',
} as const

export const LEGACY_REDIRECTS = {
  rootWorkspace: '/app/workspace',
  adminMemory: '/app/admin/engrams',
  projectsTransfer: '/app/projects/transfer/export',
  ui: '/app/workspace',
  uiAdmin: '/app/admin/sessions',
} as const

export const PRICING_PLAN_TEASERS: PricingPlanTeaser[] = [
  {
    id: 'starter',
    name: 'Starter',
    headline: 'For solo builders who want durable AI memory.',
    audience: 'Individual developers and operators',
    cta: 'Join waitlist',
    points: ['Single workspace', 'Session continuity tools', 'Core memory retrieval'],
  },
  {
    id: 'team',
    name: 'Team',
    headline: 'For small teams shipping with shared agent memory.',
    audience: 'Product and engineering teams',
    cta: 'Talk to us',
    points: ['Role-aware projects', 'Audit timelines', 'Document + engram blending'],
  },
  {
    id: 'enterprise',
    name: 'Enterprise',
    headline: 'For governed AI memory at organizational scale.',
    audience: 'Platform and security organizations',
    cta: 'Book architecture review',
    points: ['Scoped MCP tokens', 'Policy controls', 'Operational observability'],
  },
]

export const MARKETING_TOP_NAV: ReadonlyArray<{ label: string; to: string }> = [
  { label: 'Product', to: MARKETING_ROUTES.product },
  { label: 'How It Works', to: MARKETING_ROUTES.howItWorks },
  { label: 'For Enterprise', to: MARKETING_ROUTES.forEnterprise },
  { label: 'For Developers', to: MARKETING_ROUTES.forDevelopers },
  { label: 'Pricing', to: MARKETING_ROUTES.pricing },
]

export function buildAppBreadcrumb(pathname: string): BreadcrumbSpec[] {
  const base: BreadcrumbSpec[] = [{ label: 'App', to: APP_ROUTES.workspace }]

  if (pathname.startsWith('/app/admin')) {
    return [...base, { label: 'Admin', to: APP_ROUTES.adminSessions }]
  }
  if (pathname.startsWith('/app/agents')) {
    return [...base, { label: 'Agents', to: APP_ROUTES.agents }]
  }
  if (pathname.startsWith('/app/projects/transfer')) {
    return [...base, { label: 'Project Transfer', to: APP_ROUTES.transferExport }]
  }
  if (pathname.startsWith('/app/projects')) {
    return [...base, { label: 'Projects', to: APP_ROUTES.projects }]
  }
  if (pathname.startsWith('/app/documents')) {
    return [...base, { label: 'Documents', to: APP_ROUTES.documents }]
  }
  if (pathname.startsWith('/app/engrams')) {
    return [...base, { label: 'Engrams', to: APP_ROUTES.engrams }]
  }
  if (pathname.startsWith('/app/sessions')) {
    return [...base, { label: 'Sessions', to: APP_ROUTES.sessions }]
  }

  return [...base, { label: 'Workspace', to: APP_ROUTES.workspace }]
}
