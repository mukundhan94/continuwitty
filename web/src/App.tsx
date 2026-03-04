import { useEffect, useMemo, useState } from 'react'
import { Navigate, Route, Routes, useLocation, useNavigate } from 'react-router-dom'
import styled from 'styled-components'

import { listProjects } from './api/projects'
import type {
  ChatDebugTrace,
  ChatMessage,
  ChatSession,
  ChatSourceReference,
  ChatTimelineEvent,
  DocumentRecord,
  EngramSummary,
  EngramTracePath,
  PinnedDocumentRecord,
  ProjectRecord,
  UserProfile,
} from './api/types'
import { AdminMemoryPage } from './components/AdminMemoryPage'
import { AdminMcpTokenPanel } from './components/AdminMcpTokenPanel'
import { AdminObservabilityPage } from './components/AdminObservabilityPage'
import { ChatPanel } from './components/ChatPanel'
import { DocumentIngestionPanel } from './components/DocumentIngestionPanel'
import { LinkedEngramPanel } from './components/LinkedEngramPanel'
import { LoginView } from './components/LoginView'
import { PinnedEngramPanel } from './components/PinnedEngramPanel'
import { ProjectTransferPage } from './components/ProjectTransferPage'
import { SaveEngramModal } from './components/SaveEngramModal'
import { SessionSidebar } from './components/SessionSidebar'
import { WorkspaceTopNav } from './components/WorkspaceTopNav'
import { WEB_CONFIG } from './config'
import { useAdminTokenActions } from './hooks/useAdminTokenActions'
import { useAuthActions } from './hooks/useAuthActions'
import { useIngestionActions, usePinActions, usePromptActions } from './hooks/useChatActions'
import { useLinkedEngramInsights } from './hooks/useLinkedEngramInsights'
import { useProjectScopePersistence } from './hooks/useProjectScopePersistence'
import { useSessionActions } from './hooks/useSessionActions'
import { useWorkspaceDataLoaders } from './hooks/useWorkspaceDataLoaders'
import { useWorkspaceLifecycle } from './hooks/useWorkspaceLifecycle'
import { useWorkspaceActions } from './hooks/useWorkspaceActions'
import { useWorkspaceReset } from './hooks/useWorkspaceReset'
import {
  APP_ROUTES,
  LEGACY_REDIRECTS,
  MARKETING_ROUTES,
  buildAppBreadcrumb,
} from './routes/constants'
import { AuthGuard } from './routes/guards/AuthGuard'
import { AdminGuard } from './routes/guards/AdminGuard'
import { AdminLayout } from './routes/layouts/AdminLayout'
import { AppLayout } from './routes/layouts/AppLayout'
import { MarketingLayout } from './routes/layouts/MarketingLayout'
import { ForDevelopersPage } from './routes/pages/marketing/ForDevelopersPage'
import { ForEnterprisePage } from './routes/pages/marketing/ForEnterprisePage'
import { HowItWorksPage } from './routes/pages/marketing/HowItWorksPage'
import { LandingPage } from './routes/pages/marketing/LandingPage'
import { PricingPage } from './routes/pages/marketing/PricingPage'
import { ProductPage } from './routes/pages/marketing/ProductPage'
import {
  AppShell,
  GlassPane,
  LoadingScreen,
  MutedText,
  NoticeBanner,
  PaneHeader,
  ScrollColumn,
  SessionMeta,
  SplitGrid,
} from './styles/primitives'
import { useThemeMode } from './styles/useThemeMode'
import { buildDefaultSaveAbstract } from './utils/chat'
import { describeError } from './utils/errors'
import { PROJECT_ID_STORAGE_KEY, initialProjectId, normalizeProjectId } from './utils/projectScope'

const APP_ROUTE_PATTERNS = [
  APP_ROUTES.workspace,
  APP_ROUTES.sessions,
  APP_ROUTES.sessionsNew,
  APP_ROUTES.sessionChat,
  APP_ROUTES.sessionLifecycle,
  APP_ROUTES.sessionTimeline,
  APP_ROUTES.sessionSaveEngram,
  APP_ROUTES.sessionContinue,
  APP_ROUTES.engrams,
  APP_ROUTES.engramsQuery,
  APP_ROUTES.engramDetail,
  APP_ROUTES.engramSources,
  APP_ROUTES.engramRehydrate,
  APP_ROUTES.engramLinks,
  APP_ROUTES.engramTrace,
  APP_ROUTES.sessionPinsEngrams,
  APP_ROUTES.sessionPinsDocuments,
  APP_ROUTES.documents,
  APP_ROUTES.documentsIngestText,
  APP_ROUTES.documentsIngestFile,
  APP_ROUTES.projects,
  APP_ROUTES.projectMembers,
  APP_ROUTES.projectAudit,
  APP_ROUTES.transferExport,
  APP_ROUTES.transferImport,
  APP_ROUTES.adminSessions,
  APP_ROUTES.adminEngrams,
  APP_ROUTES.adminCollections,
  APP_ROUTES.adminCuration,
  APP_ROUTES.adminContradictions,
  APP_ROUTES.adminTokens,
  APP_ROUTES.adminObservability,
] as const

const RouteActionStrip = styled.div`
  min-height: 0;
  border: 1px solid var(--color-line);
  border-radius: 16px;
  background: var(--surface-raised);
  padding: 0.65rem 0.75rem;
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 0.5rem;
  flex-wrap: wrap;

  p {
    margin: 0;
    color: var(--color-ink-muted);
    font-size: 0.8rem;
    line-height: 1.55;
  }
`

const WorkspaceHomeGrid = styled.div`
  min-height: 0;
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(220px, 1fr));
  gap: 0.9rem;
`

const WorkspaceHomeCard = styled.article`
  border: 1px solid var(--color-line);
  border-radius: 16px;
  background: var(--surface-raised);
  box-shadow: var(--shadow-panel);
  padding: 0.85rem;
  display: grid;
  gap: 0.55rem;
  align-content: start;
`

const WorkspaceHomeTitle = styled.h3`
  margin: 0;
  font-family: var(--font-display);
  font-size: 1.04rem;
  color: var(--color-ink);
`

const WorkspaceSplit = styled.div`
  min-height: 0;
  display: grid;
  grid-template-columns: minmax(280px, 320px) minmax(0, 1fr);
  gap: 0.9rem;

  @media (max-width: 1180px) {
    grid-template-columns: 1fr;
  }
`

const WorkspaceMain = styled.div`
  min-height: 0;
  display: grid;
  grid-template-rows: minmax(0, 1fr) auto;
  gap: 0.9rem;

  @media (max-width: 1180px) {
    grid-template-rows: auto;
  }
`

const WorkspaceSupportGrid = styled.div`
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 0.9rem;

  @media (max-width: 1180px) {
    grid-template-columns: 1fr;
  }
`

const WorkspaceSingleColumn = styled.div`
  min-height: 0;
  display: grid;
  gap: 0.9rem;
`

const ProjectsGrid = styled.div`
  min-height: 0;
  display: grid;
  grid-template-columns: minmax(260px, 340px) minmax(0, 1fr);
  gap: 0.9rem;

  @media (max-width: 1180px) {
    grid-template-columns: 1fr;
  }
`

const ProjectsList = styled(ScrollColumn)`
  min-height: 0;
  max-height: min(58vh, 34rem);
`

const ProjectCard = styled.article`
  border: 1px solid var(--color-line);
  border-radius: 12px;
  background: var(--surface-raised);
  padding: 0.62rem;
  display: grid;
  gap: 0.28rem;
`

const ProjectActionRow = styled.div`
  display: flex;
  justify-content: flex-end;
  gap: 0.35rem;
  flex-wrap: wrap;
`

type AppSurfaceMode = 'workspace' | 'transfer' | 'admin' | 'adminTokens' | 'observability' | 'saveEngram'
type WorkspaceSection = 'home' | 'sessions' | 'engrams' | 'documents' | 'projects'
const MAX_VISIBLE_PROJECTS = 180

type SessionRouteAction =
  | 'chat'
  | 'lifecycle'
  | 'timeline'
  | 'save-engram'
  | 'continue'
  | 'pins-engrams'
  | 'pins-documents'

interface SessionRouteInfo {
  sessionId: string
  action: SessionRouteAction
}

interface AppRouteMeta {
  title: string
  description: string
  mode: AppSurfaceMode
  adminOnly: boolean
}

interface AppRouteMetaRule {
  matches: (pathname: string) => boolean
  meta: AppRouteMeta
}

interface SaveModalVisibilityState {
  saveModalOpen: boolean
  appMode: AppSurfaceMode
  isSaveRoute: boolean
}

interface ProjectDirectoryState {
  totalProjects: number
  filteredProjects: ProjectRecord[]
  visibleProjects: ProjectRecord[]
  hasMoreProjects: boolean
}

interface ProjectDirectoryPaneProps {
  projectsLoading: boolean
  projectsError: string | null
  projectSearch: string
  directoryState: ProjectDirectoryState
  onProjectSearchChange: (value: string) => void
  onRefreshProjects: () => Promise<void>
  onUseProject: (projectId: string) => void
  onOpenMembers: (projectId: string) => void
  onOpenAudit: (projectId: string) => void
}

function buildProjectDirectoryState(projects: ProjectRecord[], projectSearch: string): ProjectDirectoryState {
  const normalizedSearch = projectSearch.trim().toLowerCase()
  const sortedProjects = [...projects].sort((left, right) => left.project_id.localeCompare(right.project_id))
  const filteredProjects = normalizedSearch
    ? sortedProjects.filter((project) => {
        const haystack = `${project.project_id} ${project.name ?? ''} ${project.description ?? ''}`.toLowerCase()
        return haystack.includes(normalizedSearch)
      })
    : sortedProjects
  const visibleProjects = filteredProjects.slice(0, MAX_VISIBLE_PROJECTS)
  return {
    totalProjects: sortedProjects.length,
    filteredProjects,
    visibleProjects,
    hasMoreProjects: filteredProjects.length > visibleProjects.length,
  }
}

function ProjectDirectoryPane({
  projectsLoading,
  projectsError,
  projectSearch,
  directoryState,
  onProjectSearchChange,
  onRefreshProjects,
  onUseProject,
  onOpenMembers,
  onOpenAudit,
}: ProjectDirectoryPaneProps) {
  return (
    <GlassPane>
      <PaneHeader>
        <h2>Available Projects</h2>
        <button type="button" onClick={() => void onRefreshProjects()} disabled={projectsLoading}>
          {projectsLoading ? 'Refreshing…' : 'Refresh'}
        </button>
      </PaneHeader>
      <label htmlFor="project-search">Find project</label>
      <input
        id="project-search"
        value={projectSearch}
        onChange={(event) => onProjectSearchChange(event.target.value)}
        placeholder="Search by id, name, or description"
      />
      <SessionMeta>
        Showing {directoryState.visibleProjects.length} of {directoryState.filteredProjects.length} matches (
        {directoryState.totalProjects} total)
      </SessionMeta>
      {projectsError ? <MutedText>{projectsError}</MutedText> : null}
      <ProjectsList>
        {projectsLoading ? <MutedText>Loading projects...</MutedText> : null}
        {!projectsLoading && directoryState.filteredProjects.length === 0 ? (
          <MutedText>No projects found for this filter.</MutedText>
        ) : null}
        {directoryState.visibleProjects.map((project) => (
          <ProjectCard key={project.project_id}>
            <strong>{project.name || project.project_id}</strong>
            <SessionMeta>{project.project_id}</SessionMeta>
            <MutedText>{project.description || 'No description provided.'}</MutedText>
            <SessionMeta>
              Owner {project.owner_user_id} · Updated {new Date(project.updated_at).toLocaleString()}
            </SessionMeta>
            <ProjectActionRow>
              <button type="button" onClick={() => onUseProject(project.project_id)}>
                Use Project
              </button>
              <button type="button" onClick={() => onOpenMembers(project.project_id)}>
                Members
              </button>
              <button type="button" onClick={() => onOpenAudit(project.project_id)}>
                Audit
              </button>
            </ProjectActionRow>
          </ProjectCard>
        ))}
        {directoryState.hasMoreProjects ? (
          <MutedText>
            More than {MAX_VISIBLE_PROJECTS} matches found. Add a tighter search term to narrow the list.
          </MutedText>
        ) : null}
      </ProjectsList>
    </GlassPane>
  )
}

function useLinkRecallControls() {
  const [linkRecallEnabled, setLinkRecallEnabled] = useState(true)
  const [linkRecallDepth, setLinkRecallDepth] = useState(1)
  const [linkRecallMaxNeighbors, setLinkRecallMaxNeighbors] = useState(8)
  return {
    linkRecallEnabled,
    linkRecallDepth,
    linkRecallMaxNeighbors,
    setLinkRecallEnabled,
    setLinkRecallDepth,
    setLinkRecallMaxNeighbors,
  }
}

function useLinkedTraceState() {
  const [usedEngramIds, setUsedEngramIds] = useState<string[]>([])
  const [usedEngramLinkIds, setUsedEngramLinkIds] = useState<string[]>([])
  const [engramTracePaths, setEngramTracePaths] = useState<EngramTracePath[]>([])
  return {
    usedEngramIds,
    usedEngramLinkIds,
    engramTracePaths,
    setUsedEngramIds,
    setUsedEngramLinkIds,
    setEngramTracePaths,
  }
}

function parseSessionRoute(pathname: string): SessionRouteInfo | null {
  const directMatch = pathname.match(
    /^\/app\/sessions\/([^/]+)\/(chat|lifecycle|timeline|save-engram|continue)$/,
  )
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
      description:
        'Manage sessions, engrams, collections, curation, contradictions, members, and security controls.',
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
    matches: (pathname) => pathname.startsWith('/app/projects'),
    meta: {
      title: 'Project Collaboration Controls',
      description: 'Manage project scope, members, defaults, and audit trails.',
      mode: 'workspace',
      adminOnly: false,
    },
  },
]

function resolveAppRouteMeta(pathname: string): AppRouteMeta {
  for (const routeMetaRule of APP_ROUTE_META_RULES) {
    if (routeMetaRule.matches(pathname)) {
      return routeMetaRule.meta
    }
  }
  return DEFAULT_APP_ROUTE_META
}

function resolveWorkspaceSection(pathname: string): WorkspaceSection {
  if (pathname === APP_ROUTES.workspace) {
    return 'home'
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

function shouldShowSaveModal(state: SaveModalVisibilityState): boolean {
  return state.saveModalOpen && state.appMode === 'workspace' && !state.isSaveRoute
}

function resolvePostLoginPath(rawNextPath: string | null): string {
  const nextPath = (rawNextPath || '').trim()
  if (!nextPath) {
    return APP_ROUTES.workspace
  }

  if (nextPath === '/app' || nextPath.startsWith('/app/')) {
    return nextPath
  }

  return APP_ROUTES.workspace
}

function AppScreen() {
  const navigate = useNavigate()
  const location = useLocation()
  const { mode, toggleMode } = useThemeMode()
  const appRouteMeta = useMemo(() => resolveAppRouteMeta(location.pathname), [location.pathname])
  const workspaceSection = useMemo(
    () => resolveWorkspaceSection(location.pathname),
    [location.pathname],
  )
  const sessionRouteInfo = useMemo(() => parseSessionRoute(location.pathname), [location.pathname])
  const isLoginRoute = location.pathname === MARKETING_ROUTES.login
  const isAppRoute = location.pathname.startsWith('/app')
  const isAdminTokenRoute = location.pathname === APP_ROUTES.adminTokens
  const isSaveRoute = sessionRouteInfo?.action === 'save-engram'
  const isContinueRoute = sessionRouteInfo?.action === 'continue'

  const [authChecking, setAuthChecking] = useState(true)
  const [authSubmitting, setAuthSubmitting] = useState(false)
  const [authError, setAuthError] = useState<string | null>(null)
  const [user, setUser] = useState<UserProfile | null>(null)

  const [projectId, setProjectId] = useState(initialProjectId)
  const [defaultProjectId, setDefaultProjectId] = useState<string | null>(null)
  const [settingDefaultProject, setSettingDefaultProject] = useState(false)

  const [sessionsLoading, setSessionsLoading] = useState(false)
  const [creatingSession, setCreatingSession] = useState(false)
  const [sessions, setSessions] = useState<ChatSession[]>([])
  const [selectedSessionIdState, setSelectedSessionId] = useState<string | null>(null)

  const [messages, setMessages] = useState<ChatMessage[]>([])
  const [pendingUserText, setPendingUserText] = useState<string | null>(null)
  const [streamingAssistantText, setStreamingAssistantText] = useState('')
  const [chatSending, setChatSending] = useState(false)
  const [chatError, setChatError] = useState<string | null>(null)
  const [composerText, setComposerText] = useState('')
  const [lastPrompt, setLastPrompt] = useState('')
  const {
    linkRecallEnabled,
    linkRecallDepth,
    linkRecallMaxNeighbors,
    setLinkRecallEnabled,
    setLinkRecallDepth,
    setLinkRecallMaxNeighbors,
  } = useLinkRecallControls()
  const [sourceReferences, setSourceReferences] = useState<ChatSourceReference[]>([])
  const {
    usedEngramIds,
    usedEngramLinkIds,
    engramTracePaths,
    setUsedEngramIds,
    setUsedEngramLinkIds,
    setEngramTracePaths,
  } = useLinkedTraceState()
  const [chatDebugTrace, setChatDebugTrace] = useState<ChatDebugTrace | null>(null)
  const [timelineEvents, setTimelineEvents] = useState<ChatTimelineEvent[]>([])

  const [engramLoading, setEngramLoading] = useState(false)
  const [pinnedEngrams, setPinnedEngrams] = useState<EngramSummary[]>([])
  const [availableEngrams, setAvailableEngrams] = useState<EngramSummary[]>([])
  const [engramSearch, setEngramSearch] = useState('')
  const [documentsLoading, setDocumentsLoading] = useState(false)
  const [documentsSubmitting, setDocumentsSubmitting] = useState(false)
  const [documents, setDocuments] = useState<DocumentRecord[]>([])
  const [documentsError, setDocumentsError] = useState<string | null>(null)
  const [pinnedDocuments, setPinnedDocuments] = useState<PinnedDocumentRecord[]>([])
  const [projectsLoading, setProjectsLoading] = useState(false)
  const [projectsError, setProjectsError] = useState<string | null>(null)
  const [projects, setProjects] = useState<ProjectRecord[]>([])
  const [projectSearch, setProjectSearch] = useState('')

  const [saveModalOpen, setSaveModalOpen] = useState(false)
  const [saveSubmitting, setSaveSubmitting] = useState(false)

  const [notice, setNotice] = useState<string | null>(null)

  const selectedSessionId = sessionRouteInfo?.sessionId ?? selectedSessionIdState

  const selectedSession = useMemo(
    () => sessions.find((item) => item.session_id === selectedSessionId) || null,
    [selectedSessionId, sessions],
  )

  useEffect(() => {
    setUsedEngramIds([])
    setUsedEngramLinkIds([])
    setEngramTracePaths([])
  }, [selectedSessionId, setUsedEngramIds, setUsedEngramLinkIds, setEngramTracePaths])

  const defaultSaveAbstract = useMemo(() => buildDefaultSaveAbstract(messages), [messages])
  const isAdmin = user?.role === 'admin'
  const adminTokenActions = useAdminTokenActions({ isAdmin, setNotice, describeError })

  const workspaceDataLoaders = useWorkspaceDataLoaders({
    projectId,
    normalizeProjectId,
    describeError,
    setProjectId,
    setDefaultProjectId,
    setSessionsLoading,
    setSessions,
    setSelectedSessionId,
    setChatError,
    setEngramLoading,
    setMessages,
    setPinnedEngrams,
    setPinnedDocuments,
    setAvailableEngrams,
    setTimelineEvents,
    setDocumentsLoading,
    setDocumentsError,
    setDocuments,
  })

  useProjectScopePersistence({
    projectId,
    normalizeProjectId,
    storageKey: PROJECT_ID_STORAGE_KEY,
  })

  useWorkspaceLifecycle({
    projectId,
    selectedSessionId,
    user,
    describeError,
    setAuthChecking,
    setAuthError,
    setUser,
    setMessages,
    setPinnedEngrams,
    setPinnedDocuments,
    setSourceReferences,
    setChatDebugTrace,
    setTimelineEvents,
    loadDefaultProject: workspaceDataLoaders.loadDefaultProject,
    loadSessions: workspaceDataLoaders.loadSessions,
    loadProjectDocuments: workspaceDataLoaders.loadProjectDocuments,
    loadSessionData: workspaceDataLoaders.loadSessionData,
  })

  const resetWorkspaceState = useWorkspaceReset({
    setUser,
    setSessions,
    setSelectedSessionId,
    setDefaultProjectId,
    setMessages,
    setPinnedEngrams,
    setPinnedDocuments,
    setAvailableEngrams,
    setSourceReferences,
    setDocuments,
    setDocumentsError,
    setTimelineEvents,
    setNotice,
    setAuthError,
    resetAdminTokenState: adminTokenActions.resetAdminTokenState,
  })

  const { handleLogin, handleLogout } = useAuthActions({
    projectId,
    loadDefaultProject: workspaceDataLoaders.loadDefaultProject,
    loadSessions: workspaceDataLoaders.loadSessions,
    loadProjectDocuments: workspaceDataLoaders.loadProjectDocuments,
    setUser,
    setAuthSubmitting,
    setAuthError,
    describeError,
    resetWorkspaceState,
  })

  const { handleCreateSession, handleSetDefaultProject } = useWorkspaceActions({
    projectId,
    normalizeProjectId,
    describeError,
    setCreatingSession,
    setChatError,
    setSessions,
    setSelectedSessionId,
    setComposerText,
    setSettingDefaultProject,
    setDefaultProjectId,
    setNotice,
  })

  const { handleSend, handleRetry } = usePromptActions({
    composerText,
    lastPrompt,
    selectedSessionId,
    refreshFromSession: workspaceDataLoaders.refreshFromSession,
    setLastPrompt,
    setPendingUserText,
    setComposerText,
    setStreamingAssistantText,
    setSourceReferences,
    setUsedEngramIds,
    setUsedEngramLinkIds,
    setEngramTracePaths,
    setChatDebugTrace,
    setChatError,
    setChatSending,
    recallOptions: {
      link_recall_enabled: linkRecallEnabled,
      link_recall_depth: linkRecallDepth,
      link_recall_max_neighbors: linkRecallMaxNeighbors,
    },
    describeError,
  })

  const {
    sourceEngramIds: linkedSourceEngramIds,
    links: linkedEngramLinks,
    suggestions: linkedEngramSuggestions,
    loading: linkedEngramLoading,
    error: linkedEngramError,
    pendingSuggestionKeys,
    refreshLinkInsights,
    handleAcceptSuggestion,
    handleRejectSuggestion,
  } = useLinkedEngramInsights({
    selectedSessionId,
    usedEngramIds,
    usedEngramLinkIds,
    engramTracePaths,
    setNotice,
    setChatError,
    describeError,
  })

  const { handlePin, handleUnpin, handlePinDocument, handleUnpinDocument } = usePinActions({
    selectedSessionId,
    refreshFromSession: workspaceDataLoaders.refreshFromSession,
    setNotice,
    setChatError,
    describeError,
  })

  const { handleRefreshDocuments, handleIngestText, handleIngestFile } = useIngestionActions({
    projectId,
    normalizeProjectId,
    loadProjectDocuments: workspaceDataLoaders.loadProjectDocuments,
    setDocumentsSubmitting,
    setDocumentsError,
    setNotice,
    describeError,
  })

  const { handleCopyEngramId, handleRefreshEngrams, handleContinueSession, handleSaveEngram } =
    useSessionActions({
      selectedSessionId,
      refreshFromSession: workspaceDataLoaders.refreshFromSession,
      setSessions,
      setSelectedSessionId,
      setSaveModalOpen,
      setSaveSubmitting,
      setNotice,
      setChatError,
      describeError,
    })

  useEffect(() => {
    if (authChecking || !user || !isLoginRoute) {
      return
    }
    const params = new URLSearchParams(location.search)
    const nextPath = resolvePostLoginPath(params.get('next'))
    navigate(nextPath, { replace: true })
  }, [authChecking, isLoginRoute, location.search, navigate, user])

  useEffect(() => {
    if (!isAdmin || !isAdminTokenRoute) {
      return
    }
    void adminTokenActions.handleRefreshAdminTokenPanel()
  }, [adminTokenActions, isAdmin, isAdminTokenRoute])

  useEffect(() => {
    if (!user || workspaceSection !== 'projects') {
      return
    }
    let cancelled = false
    const loadProjects = async () => {
      setProjectsLoading(true)
      setProjectsError(null)
      try {
        const loaded = await listProjects(false)
        if (!cancelled) {
          setProjects(loaded)
        }
      } catch (error) {
        if (!cancelled) {
          setProjectsError(describeError(error))
        }
      } finally {
        if (!cancelled) {
          setProjectsLoading(false)
        }
      }
    }
    void loadProjects()
    return () => {
      cancelled = true
    }
  }, [user, workspaceSection])

  useEffect(() => {
    if (!sessionRouteInfo || sessionsLoading) {
      return
    }
    if (sessions.length === 0) {
      return
    }
    const routeSessionExists = sessions.some((item) => item.session_id === sessionRouteInfo.sessionId)
    if (routeSessionExists) {
      return
    }
    navigate(APP_ROUTES.sessions, { replace: true })
  }, [navigate, sessionRouteInfo, sessions, sessionsLoading])

  useEffect(() => {
    if (!selectedSessionId) {
      return
    }
    if (location.pathname !== APP_ROUTES.sessions && location.pathname !== APP_ROUTES.sessionsNew) {
      return
    }
    navigate(`/app/sessions/${selectedSessionId}/chat`, { replace: true })
  }, [location.pathname, navigate, selectedSessionId])

  const handleSelectSession = (sessionId: string) => {
    setSelectedSessionId(sessionId)

    if (!location.pathname.startsWith('/app/sessions/')) {
      navigate(`/app/sessions/${sessionId}/chat`)
      return
    }

    if (!sessionRouteInfo) {
      navigate(`/app/sessions/${sessionId}/chat`)
      return
    }

    if (sessionRouteInfo.sessionId !== sessionId) {
      navigate(`/app/sessions/${sessionId}/chat`)
      return
    }

    if (sessionRouteInfo.action !== 'chat') {
      navigate(`/app/sessions/${sessionId}/chat`)
    }
  }

  const handleRouteContinue = async () => {
    await handleContinueSession()
    navigate(APP_ROUTES.sessions)
  }

  const handleRouteSave = async (payload: {
    title: string
    abstract: string
    visibility_scope: 'private' | 'project'
    tags: string[]
    keywords: string[]
  }) => {
    await handleSaveEngram(payload)
    if (selectedSessionId && isSaveRoute) {
      navigate(`/app/sessions/${selectedSessionId}/chat`, { replace: true })
    }
  }

  const closeSaveModal = () => {
    setSaveModalOpen(false)
    if (selectedSessionId && isSaveRoute) {
      navigate(`/app/sessions/${selectedSessionId}/chat`, { replace: true })
    }
  }

  const renderWorkspaceHome = () => (
    <WorkspaceHomeGrid>
      <WorkspaceHomeCard>
        <WorkspaceHomeTitle>Sessions and Chat</WorkspaceHomeTitle>
        <MutedText>
          Focused chat workspace with a cleaner transcript flow and dedicated memory actions.
        </MutedText>
        <SessionMeta>{sessions.length} sessions loaded</SessionMeta>
        <button type="button" onClick={() => navigate(APP_ROUTES.sessions)}>
          Open Sessions
        </button>
      </WorkspaceHomeCard>
      <WorkspaceHomeCard>
        <WorkspaceHomeTitle>Engram Memory</WorkspaceHomeTitle>
        <MutedText>Review pinned engrams, graph edges, and suggestion queue in a separate memory page.</MutedText>
        <SessionMeta>{availableEngrams.length} engrams available</SessionMeta>
        <button type="button" onClick={() => navigate(APP_ROUTES.engrams)}>
          Open Engrams
        </button>
      </WorkspaceHomeCard>
      <WorkspaceHomeCard>
        <WorkspaceHomeTitle>Document Ingestion</WorkspaceHomeTitle>
        <MutedText>Upload and chunk docs without competing for screen space with chat controls.</MutedText>
        <SessionMeta>{documents.length} project documents</SessionMeta>
        <button type="button" onClick={() => navigate(APP_ROUTES.documents)}>
          Open Documents
        </button>
      </WorkspaceHomeCard>
      <WorkspaceHomeCard>
        <WorkspaceHomeTitle>Projects</WorkspaceHomeTitle>
        <MutedText>Manage active project scope, defaults, members, and transfer entry points.</MutedText>
        <SessionMeta>Active: {projectId || 'not set'}</SessionMeta>
        <button type="button" onClick={() => navigate(APP_ROUTES.projects)}>
          Open Projects
        </button>
      </WorkspaceHomeCard>
    </WorkspaceHomeGrid>
  )

  const renderSessionPrimarySurface = () => {
    if (sessionRouteInfo?.action === 'pins-engrams') {
      return (
        <PinnedEngramPanel
          selectedSessionId={selectedSessionId}
          pinnedEngrams={pinnedEngrams}
          availableEngrams={availableEngrams}
          search={engramSearch}
          loading={engramLoading}
          onSearchChange={setEngramSearch}
          onRefresh={handleRefreshEngrams}
          onPin={handlePin}
          onUnpin={handleUnpin}
          onCopyId={handleCopyEngramId}
        />
      )
    }

    if (sessionRouteInfo?.action === 'pins-documents') {
      return (
        <DocumentIngestionPanel
          projectId={projectId}
          selectedSessionId={selectedSessionId}
          documents={documents}
          pinnedDocumentIds={pinnedDocuments.map((item) => item.document_id)}
          loading={documentsLoading}
          submitting={documentsSubmitting}
          error={documentsError}
          onRefresh={handleRefreshDocuments}
          onIngestText={handleIngestText}
          onIngestFile={handleIngestFile}
          onPinDocument={handlePinDocument}
          onUnpinDocument={handleUnpinDocument}
        />
      )
    }

    return (
      <ChatPanel
        session={selectedSession}
        messages={messages}
        pendingUserText={pendingUserText}
        streamingAssistantText={streamingAssistantText}
        composerText={composerText}
        sending={chatSending}
        error={chatError}
        sourceReferences={sourceReferences}
        usedEngramLinkIds={usedEngramLinkIds}
        engramTracePaths={engramTracePaths}
        debugTrace={chatDebugTrace}
        timelineEvents={timelineEvents}
        showTimeline={sessionRouteInfo?.action === 'timeline' || sessionRouteInfo?.action === 'lifecycle'}
        showDebugTrace={sessionRouteInfo?.action === 'lifecycle'}
        linkRecallEnabled={linkRecallEnabled}
        linkRecallDepth={linkRecallDepth}
        linkRecallMaxNeighbors={linkRecallMaxNeighbors}
        onLinkRecallEnabledChange={setLinkRecallEnabled}
        onLinkRecallDepthChange={setLinkRecallDepth}
        onLinkRecallMaxNeighborsChange={setLinkRecallMaxNeighbors}
        onComposerChange={setComposerText}
        onSend={handleSend}
        onRetry={handleRetry}
        onOpenSaveModal={() => setSaveModalOpen(true)}
        onContinueSession={handleContinueSession}
      />
    )
  }

  const renderSessionsWorkspace = () => (
    <WorkspaceSplit>
      <SessionSidebar
        sessions={sessions}
        selectedSessionId={selectedSessionId}
        projectId={projectId}
        defaultProjectId={defaultProjectId}
        settingDefaultProject={settingDefaultProject}
        defaultProvider={WEB_CONFIG.defaultProvider}
        defaultVisibilityScope={WEB_CONFIG.defaultVisibility}
        modelDefaults={WEB_CONFIG.defaultModelByProvider}
        loading={sessionsLoading}
        creating={creatingSession}
        onProjectChange={(value) => setProjectId(normalizeProjectId(value))}
        onSetDefaultProject={handleSetDefaultProject}
        onSelectSession={handleSelectSession}
        onCreateSession={handleCreateSession}
      />

      <WorkspaceMain>
        <WorkspaceSingleColumn>
          {isContinueRoute ? (
            <RouteActionStrip>
              <p>Continue this thread as a fresh session while carrying your pinned continuity context.</p>
              <button type="button" onClick={() => void handleRouteContinue()} disabled={!selectedSessionId}>
                Continue Session Now
              </button>
            </RouteActionStrip>
          ) : null}
          {renderSessionPrimarySurface()}
        </WorkspaceSingleColumn>

        <WorkspaceSupportGrid>
          <RouteActionStrip>
            <p>
              Chat is intentionally focused. Memory graph, timeline, and pins are available via dedicated
              routes and this quick-action strip.
            </p>
            <SplitGrid>
              <button
                type="button"
                onClick={() =>
                  selectedSessionId
                    ? navigate(`/app/sessions/${selectedSessionId}/pins/engrams`)
                    : navigate(APP_ROUTES.engrams)
                }
              >
                Engram Pins
              </button>
              <button
                type="button"
                onClick={() =>
                  selectedSessionId
                    ? navigate(`/app/sessions/${selectedSessionId}/pins/documents`)
                    : navigate(APP_ROUTES.documents)
                }
              >
                Document Pins
              </button>
            </SplitGrid>
          </RouteActionStrip>

          <LinkedEngramPanel
            selectedSessionId={selectedSessionId}
            loading={linkedEngramLoading}
            error={linkedEngramError}
            sourceEngramIds={linkedSourceEngramIds}
            links={linkedEngramLinks}
            suggestions={linkedEngramSuggestions}
            pendingSuggestionKeys={pendingSuggestionKeys}
            availableEngrams={availableEngrams}
            sourceReferences={sourceReferences}
            tracePaths={engramTracePaths}
            onRefresh={refreshLinkInsights}
            onAcceptSuggestion={handleAcceptSuggestion}
            onRejectSuggestion={handleRejectSuggestion}
          />
        </WorkspaceSupportGrid>
      </WorkspaceMain>
    </WorkspaceSplit>
  )

  const renderEngramsWorkspace = () => (
    <WorkspaceSingleColumn>
      <RouteActionStrip>
        <p>Engram management is separated from chat so memory curation has full, dedicated space.</p>
        <button type="button" onClick={() => navigate(APP_ROUTES.sessions)}>
          Open Session Chat
        </button>
      </RouteActionStrip>
      <WorkspaceSupportGrid>
        <PinnedEngramPanel
          selectedSessionId={selectedSessionId}
          pinnedEngrams={pinnedEngrams}
          availableEngrams={availableEngrams}
          search={engramSearch}
          loading={engramLoading}
          onSearchChange={setEngramSearch}
          onRefresh={handleRefreshEngrams}
          onPin={handlePin}
          onUnpin={handleUnpin}
          onCopyId={handleCopyEngramId}
        />
        <LinkedEngramPanel
          selectedSessionId={selectedSessionId}
          loading={linkedEngramLoading}
          error={linkedEngramError}
          sourceEngramIds={linkedSourceEngramIds}
          links={linkedEngramLinks}
          suggestions={linkedEngramSuggestions}
          pendingSuggestionKeys={pendingSuggestionKeys}
          availableEngrams={availableEngrams}
          sourceReferences={sourceReferences}
          tracePaths={engramTracePaths}
          onRefresh={refreshLinkInsights}
          onAcceptSuggestion={handleAcceptSuggestion}
          onRejectSuggestion={handleRejectSuggestion}
        />
      </WorkspaceSupportGrid>
    </WorkspaceSingleColumn>
  )

  const renderDocumentsWorkspace = () => (
    <WorkspaceSingleColumn>
      <RouteActionStrip>
        <p>Documents and evidence pinning are isolated here to reduce noise in active chat workflows.</p>
        <button
          type="button"
          onClick={() =>
            selectedSessionId
              ? navigate(`/app/sessions/${selectedSessionId}/pins/documents`)
              : navigate(APP_ROUTES.sessions)
          }
        >
          Open Session Document Pins
        </button>
      </RouteActionStrip>

      <WorkspaceSupportGrid>
        <DocumentIngestionPanel
          projectId={projectId}
          selectedSessionId={selectedSessionId}
          documents={documents}
          pinnedDocumentIds={pinnedDocuments.map((item) => item.document_id)}
          loading={documentsLoading}
          submitting={documentsSubmitting}
          error={documentsError}
          onRefresh={handleRefreshDocuments}
          onIngestText={handleIngestText}
          onIngestFile={handleIngestFile}
          onPinDocument={handlePinDocument}
          onUnpinDocument={handleUnpinDocument}
        />

        <GlassPane>
          <PaneHeader>
            <h2>Pinned Document Context</h2>
          </PaneHeader>
          <MutedText>
            {selectedSessionId
              ? `Active session: ${selectedSessionId}`
              : 'Pick a session to pin or unpin project documents.'}
          </MutedText>
          <SessionMeta>{pinnedDocuments.length} documents pinned</SessionMeta>
          <ProjectsList>
            {pinnedDocuments.length === 0 ? (
              <MutedText>No pinned documents for the selected session.</MutedText>
            ) : (
              pinnedDocuments.map((item) => (
                <ProjectCard key={item.document_id}>
                  <strong>{item.document_id}</strong>
                  <SessionMeta>Pinned at {new Date(item.created_at).toLocaleString()}</SessionMeta>
                  <ProjectActionRow>
                    <button type="button" onClick={() => void handleUnpinDocument(item.document_id)}>
                      Unpin
                    </button>
                  </ProjectActionRow>
                </ProjectCard>
              ))
            )}
          </ProjectsList>
        </GlassPane>
      </WorkspaceSupportGrid>
    </WorkspaceSingleColumn>
  )

  const renderProjectsWorkspace = () => {
    const directoryState = buildProjectDirectoryState(projects, projectSearch)

    const refreshProjects = async () => {
      setProjectsLoading(true)
      setProjectsError(null)
      try {
        setProjects(await listProjects(false))
      } catch (error) {
        setProjectsError(describeError(error))
      } finally {
        setProjectsLoading(false)
      }
    }

    return (
      <ProjectsGrid>
        <GlassPane>
          <PaneHeader>
            <h2>Active Project Scope</h2>
            <button type="button" onClick={() => void handleSetDefaultProject()} disabled={settingDefaultProject}>
              {settingDefaultProject ? 'Saving…' : 'Set as Default'}
            </button>
          </PaneHeader>
          <label htmlFor="active-project-id">Project ID</label>
          <input
            id="active-project-id"
            value={projectId}
            onChange={(event) => setProjectId(normalizeProjectId(event.target.value))}
            placeholder="project-id"
          />
          <MutedText>
            Default project: <strong>{defaultProjectId || 'not configured'}</strong>
          </MutedText>
          <RouteActionStrip>
            <p>Project routes are now dedicated pages so governance actions do not interrupt chat.</p>
            <SplitGrid>
              <button type="button" onClick={() => navigate(APP_ROUTES.transferExport)}>
                Export Project
              </button>
              <button type="button" onClick={() => navigate(APP_ROUTES.transferImport)}>
                Import Bundle
              </button>
            </SplitGrid>
          </RouteActionStrip>
        </GlassPane>

        <ProjectDirectoryPane
          projectsLoading={projectsLoading}
          projectsError={projectsError}
          projectSearch={projectSearch}
          directoryState={directoryState}
          onProjectSearchChange={setProjectSearch}
          onRefreshProjects={refreshProjects}
          onUseProject={setProjectId}
          onOpenMembers={(projectIdValue) => navigate(`/app/projects/${projectIdValue}/members`)}
          onOpenAudit={(projectIdValue) => navigate(`/app/projects/${projectIdValue}/audit`)}
        />
      </ProjectsGrid>
    )
  }

  const renderWorkspaceSurface = () => {
    if (workspaceSection === 'home') {
      return renderWorkspaceHome()
    }
    if (workspaceSection === 'sessions') {
      return renderSessionsWorkspace()
    }
    if (workspaceSection === 'engrams') {
      return renderEngramsWorkspace()
    }
    if (workspaceSection === 'documents') {
      return renderDocumentsWorkspace()
    }
    return renderProjectsWorkspace()
  }

  const renderBody = () => {
    if (appRouteMeta.mode === 'admin') {
      return (
        <AdminLayout title={appRouteMeta.title} description={appRouteMeta.description}>
          <AdminMemoryPage
            projectId={projectId}
            onProjectChange={(value) => setProjectId(normalizeProjectId(value))}
            onNotice={(message) => setNotice(message)}
          />
        </AdminLayout>
      )
    }

    if (appRouteMeta.mode === 'adminTokens') {
      return (
        <AdminLayout title={appRouteMeta.title} description={appRouteMeta.description}>
          <AdminMcpTokenPanel
            variant="inline"
            isOpen={isAdmin}
            loading={adminTokenActions.adminTokensLoading}
            optionsLoading={adminTokenActions.adminTokenOptionsLoading}
            creating={adminTokenActions.adminTokensCreating}
            tokens={adminTokenActions.adminTokens}
            latestToken={adminTokenActions.adminLatestToken}
            error={adminTokenActions.adminTokenError}
            availableTools={adminTokenActions.adminAvailableTools}
            availableProjects={adminTokenActions.adminAvailableProjects}
            onClose={() => navigate(APP_ROUTES.adminSessions, { replace: true })}
            onRefresh={adminTokenActions.handleRefreshAdminTokenPanel}
            onCreate={adminTokenActions.handleCreateAdminToken}
            onRevoke={adminTokenActions.handleRevokeAdminToken}
          />
        </AdminLayout>
      )
    }

    if (appRouteMeta.mode === 'transfer') {
      return (
        <AppLayout title={appRouteMeta.title} description={appRouteMeta.description}>
          <ProjectTransferPage
            projectId={projectId}
            onProjectChange={(value) => setProjectId(normalizeProjectId(value))}
            onNotice={(message) => setNotice(message)}
          />
        </AppLayout>
      )
    }

    if (appRouteMeta.mode === 'saveEngram') {
      return (
        <AppLayout title={appRouteMeta.title} description={appRouteMeta.description}>
          {selectedSessionId ? (
            <SaveEngramModal
              variant="inline"
              defaultTitle={selectedSession ? `${selectedSession.title} Snapshot` : 'Chat Snapshot'}
              defaultAbstract={defaultSaveAbstract}
              saving={saveSubmitting}
              onClose={closeSaveModal}
              onSave={handleRouteSave}
            />
          ) : (
            <RouteActionStrip>
              <p>Select a valid session from the sidebar first, then save it as an engram.</p>
            </RouteActionStrip>
          )}
        </AppLayout>
      )
    }

    if (appRouteMeta.mode === 'observability') {
      return (
        <AdminLayout title={appRouteMeta.title} description={appRouteMeta.description}>
          <AdminObservabilityPage />
        </AdminLayout>
      )
    }

    return <AppLayout title={appRouteMeta.title} description={appRouteMeta.description}>{renderWorkspaceSurface()}</AppLayout>
  }

  const renderSaveModal = () => {
    if (!shouldShowSaveModal({ saveModalOpen, appMode: appRouteMeta.mode, isSaveRoute })) {
      return null
    }

    return (
      <SaveEngramModal
        defaultTitle={selectedSession ? `${selectedSession.title} Snapshot` : 'Chat Snapshot'}
        defaultAbstract={defaultSaveAbstract}
        saving={saveSubmitting}
        onClose={closeSaveModal}
        onSave={handleRouteSave}
      />
    )
  }

  const renderAdminTokenPanel = () => {
    if (!isAdmin || isAdminTokenRoute) {
      return null
    }

    return (
      <AdminMcpTokenPanel
        isOpen={adminTokenActions.adminTokenPanelOpen}
        loading={adminTokenActions.adminTokensLoading}
        optionsLoading={adminTokenActions.adminTokenOptionsLoading}
        creating={adminTokenActions.adminTokensCreating}
        tokens={adminTokenActions.adminTokens}
        latestToken={adminTokenActions.adminLatestToken}
        error={adminTokenActions.adminTokenError}
        availableTools={adminTokenActions.adminAvailableTools}
        availableProjects={adminTokenActions.adminAvailableProjects}
        onClose={() => {
          adminTokenActions.closeAdminTokenPanel()
          if (isAdminTokenRoute) {
            navigate(APP_ROUTES.adminSessions, { replace: true })
          }
        }}
        onRefresh={adminTokenActions.handleRefreshAdminTokenPanel}
        onCreate={adminTokenActions.handleCreateAdminToken}
        onRevoke={adminTokenActions.handleRevokeAdminToken}
      />
    )
  }

  if (authChecking) {
    return <LoadingScreen>Loading local workspace...</LoadingScreen>
  }

  if (!user) {
    if (isLoginRoute) {
      return <LoginView isSubmitting={authSubmitting} error={authError} onSubmit={handleLogin} />
    }

    const nextPath = encodeURIComponent(location.pathname + location.search)
    return <Navigate to={`${MARKETING_ROUTES.login}?next=${nextPath}`} replace />
  }

  if (isLoginRoute) {
    return <Navigate to={APP_ROUTES.workspace} replace />
  }

  if (!isAppRoute) {
    return <Navigate to={APP_ROUTES.workspace} replace />
  }

  const appContent = (
    <AppShell>
      <WorkspaceTopNav
        user={user}
        isAdmin={isAdmin}
        mode={mode}
        breadcrumbs={buildAppBreadcrumb(location.pathname)}
        onToggleTheme={toggleMode}
        onLogout={handleLogout}
      />
      {notice ? <NoticeBanner>{notice}</NoticeBanner> : null}
      {renderBody()}
      {renderSaveModal()}
      {renderAdminTokenPanel()}
    </AppShell>
  )

  return (
    <AuthGuard isAuthenticated={Boolean(user)}>
      {appRouteMeta.adminOnly ? <AdminGuard isAdmin={isAdmin}>{appContent}</AdminGuard> : appContent}
    </AuthGuard>
  )
}

export default function App() {
  return (
    <Routes>
      <Route element={<MarketingLayout />}>
        <Route path={MARKETING_ROUTES.landing} element={<LandingPage />} />
        <Route path={MARKETING_ROUTES.forEnterprise} element={<ForEnterprisePage />} />
        <Route path={MARKETING_ROUTES.forDevelopers} element={<ForDevelopersPage />} />
        <Route path={MARKETING_ROUTES.product} element={<ProductPage />} />
        <Route path={MARKETING_ROUTES.howItWorks} element={<HowItWorksPage />} />
        <Route path={MARKETING_ROUTES.pricing} element={<PricingPage />} />
      </Route>

      <Route path={MARKETING_ROUTES.login} element={<AppScreen />} />
      <Route path="/app" element={<Navigate to={APP_ROUTES.workspace} replace />} />
      {APP_ROUTE_PATTERNS.map((path) => (
        <Route key={path} path={path} element={<AppScreen />} />
      ))}

      <Route path="/admin/memory" element={<Navigate to={LEGACY_REDIRECTS.adminMemory} replace />} />
      <Route path="/projects/transfer" element={<Navigate to={LEGACY_REDIRECTS.projectsTransfer} replace />} />
      <Route path="/ui" element={<Navigate to={LEGACY_REDIRECTS.ui} replace />} />
      <Route path="/ui/admin" element={<Navigate to={LEGACY_REDIRECTS.uiAdmin} replace />} />
      <Route path="*" element={<Navigate to={MARKETING_ROUTES.landing} replace />} />
    </Routes>
  )
}
