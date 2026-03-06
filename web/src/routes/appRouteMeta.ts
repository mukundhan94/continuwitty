import { APP_ROUTES } from './constants'

export type AppSurfaceMode = 'workspace' | 'transfer' | 'admin' | 'adminTokens' | 'observability' | 'saveEngram'
export type WorkspaceSection = 'home' | 'agents' | 'sessions' | 'engrams' | 'documents' | 'projects'

export type SessionRouteAction =
  | 'chat'
  | 'lifecycle'
  | 'timeline'
  | 'save-engram'
  | 'continue'
  | 'pins-engrams'
  | 'pins-documents'

export interface SessionRouteInfo {
  sessionId: string
  action: SessionRouteAction
}

export type ProjectRouteAction = 'members' | 'audit'

export interface ProjectRouteInfo {
  projectId: string
  action: ProjectRouteAction
}

export interface AppRouteMeta {
  title: string
  description: string
  mode: AppSurfaceMode
  adminOnly: boolean
}

interface AppRouteMetaRule {
  matches: (pathname: string) => boolean
  meta: AppRouteMeta
}

const SAVE_ENGRAM_ROUTE_PATTERN = /^\/app\/sessions\/[^/]+\/save-engram$/

const DEFAULT_APP_ROUTE_META: AppRouteMeta = {
  title: 'Memory Continuity Workspace',
  description: 'Coordinate sessions, context retrieval, and memory actions in one command surface.',
  mode: 'workspace',
  adminOnly: false,
}

const APP_ROUTE_META_RULES: AppRouteMetaRule[] = [
  {
    matches: (pathname) => pathname === APP_ROUTES.adminTokens,
    meta: {
      title: 'MCP Token Administration',
      description: 'Create, scope, and revoke MCP access tokens in a dedicated admin control plane.',
      mode: 'adminTokens',
      adminOnly: true,
    },
  },
  {
    matches: (pathname) => pathname === APP_ROUTES.adminObservability,
    meta: {
      title: 'Observability and Runtime Health',
      description: 'Inspect request metrics and release metadata for this running environment.',
      mode: 'observability',
      adminOnly: true,
    },
  },
  {
    matches: (pathname) => pathname.startsWith('/app/admin'),
    meta: {
      title: 'Memory Administration',
      description: 'Manage sessions, engrams, collections, curation, contradictions, members, and security controls.',
      mode: 'admin',
      adminOnly: true,
    },
  },
  {
    matches: (pathname) => pathname.startsWith('/app/projects/transfer'),
    meta: {
      title: 'Project Export and Import',
      description: 'Move continuity bundles across workspaces with deterministic conflict policies.',
      mode: 'transfer',
      adminOnly: false,
    },
  },
  {
    matches: (pathname) => pathname.startsWith(APP_ROUTES.agents),
    meta: {
      title: 'Agent Runs and Checkpoints',
      description: 'Start durable agent threads, inspect checkpoint state, and resume long-running work without context loss.',
      mode: 'workspace',
      adminOnly: false,
    },
  },
  {
    matches: (pathname) => pathname === APP_ROUTES.sessionsNew,
    meta: {
      title: 'Create Session',
      description: 'Configure provider, model, and lifecycle defaults for a new memory session.',
      mode: 'workspace',
      adminOnly: false,
    },
  },
  {
    matches: (pathname) => pathname === APP_ROUTES.sessions,
    meta: {
      title: 'Session Workspace',
      description: 'Browse and continue previous chat sessions with durable memory continuity.',
      mode: 'workspace',
      adminOnly: false,
    },
  },
  {
    matches: (pathname) => SAVE_ENGRAM_ROUTE_PATTERN.test(pathname),
    meta: {
      title: 'Save Session as Engram',
      description: 'Promote this conversation into durable memory with title, abstract, visibility, and tags.',
      mode: 'saveEngram',
      adminOnly: false,
    },
  },
  {
    matches: (pathname) => pathname.startsWith('/app/engrams'),
    meta: {
      title: 'Engram Retrieval and Graph',
      description: 'Search, inspect, trace, and curate memory artifacts with provenance-first workflows.',
      mode: 'workspace',
      adminOnly: false,
    },
  },
  {
    matches: (pathname) => pathname.startsWith('/app/documents'),
    meta: {
      title: 'Document Ingestion and Pinning',
      description: 'Ingest source documents and pin evidence for active sessions.',
      mode: 'workspace',
      adminOnly: false,
    },
  },
  {
    matches: (pathname) => /^\/app\/projects\/[^/]+\/members$/.test(pathname),
    meta: {
      title: 'Project Members',
      description: 'Manage project access with dedicated membership controls and role updates.',
      mode: 'workspace',
      adminOnly: false,
    },
  },
  {
    matches: (pathname) => /^\/app\/projects\/[^/]+\/audit$/.test(pathname),
    meta: {
      title: 'Project Audit Timeline',
      description: 'Inspect project-scoped governance events, memory actions, and collaboration history.',
      mode: 'workspace',
      adminOnly: false,
    },
  },
  {
    matches: (pathname) => pathname.startsWith('/app/projects'),
    meta: {
      title: 'Project Collaboration Controls',
      description: 'Manage project scope, members, defaults, and audit trails.',
      mode: 'workspace',
      adminOnly: false,
    },
  },
]

export function parseSessionRoute(pathname: string): SessionRouteInfo | null {
  const directMatch = pathname.match(/^\/app\/sessions\/([^/]+)\/(chat|lifecycle|timeline|save-engram|continue)$/)
  if (directMatch) {
    return {
      sessionId: directMatch[1],
      action: directMatch[2] as SessionRouteAction,
    }
  }

  const pinMatch = pathname.match(/^\/app\/sessions\/([^/]+)\/pins\/(engrams|documents)$/)
  if (pinMatch) {
    return {
      sessionId: pinMatch[1],
      action: pinMatch[2] === 'engrams' ? 'pins-engrams' : 'pins-documents',
    }
  }

  return null
}

export function parseAgentThreadRoute(pathname: string): string | null {
  const match = pathname.match(/^\/app\/agents\/runs\/([^/]+)$/)
  if (!match) {
    return null
  }
  return decodeURIComponent(match[1])
}

export function parseProjectRoute(pathname: string): ProjectRouteInfo | null {
  const match = pathname.match(/^\/app\/projects\/([^/]+)\/(members|audit)$/)
  if (!match) {
    return null
  }
  return {
    projectId: decodeURIComponent(match[1]),
    action: match[2] as ProjectRouteAction,
  }
}

export function resolveAppRouteMeta(pathname: string): AppRouteMeta {
  for (const routeMetaRule of APP_ROUTE_META_RULES) {
    if (routeMetaRule.matches(pathname)) {
      return routeMetaRule.meta
    }
  }
  return DEFAULT_APP_ROUTE_META
}

export function resolveWorkspaceSection(pathname: string): WorkspaceSection {
  if (pathname === APP_ROUTES.workspace) {
    return 'home'
  }
  if (pathname.startsWith(APP_ROUTES.agents)) {
    return 'agents'
  }
  if (pathname.startsWith('/app/sessions')) {
    return 'sessions'
  }
  if (pathname.startsWith('/app/engrams')) {
    return 'engrams'
  }
  if (pathname.startsWith('/app/documents')) {
    return 'documents'
  }
  return 'projects'
}
