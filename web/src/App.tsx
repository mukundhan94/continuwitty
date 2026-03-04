import { useEffect, useMemo, useState } from 'react'
import { Navigate, Route, Routes, useLocation, useNavigate } from 'react-router-dom'
import styled from 'styled-components'

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
import { AppShell, LoadingScreen, NoticeBanner, WorkspaceGrid } from './styles/primitives'
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

const RightRail = styled.div`
  min-height: 0;
  display: grid;
  grid-template-rows: minmax(10rem, 0.6fr) minmax(10rem, 0.7fr) minmax(0, 1.2fr);
  gap: 0.9rem;

  @media (max-width: 1180px) {
    grid-template-rows: none;
  }
`

const RouteActionStrip = styled.div`
  border: 1px solid rgba(94, 234, 212, 0.2);
  border-radius: 16px;
  background: rgba(8, 20, 32, 0.45);
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

type AppSurfaceMode = 'workspace' | 'transfer' | 'admin' | 'adminTokens' | 'observability' | 'saveEngram'

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
  const [selectedSessionId, setSelectedSessionId] = useState<string | null>(null)

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

  const [saveModalOpen, setSaveModalOpen] = useState(false)
  const [saveSubmitting, setSaveSubmitting] = useState(false)

  const [notice, setNotice] = useState<string | null>(null)

  const selectedSession = useMemo(
    () => sessions.find((item) => item.session_id === selectedSessionId) || null,
    [selectedSessionId, sessions],
  )

  useEffect(() => {
    setUsedEngramIds([])
    setUsedEngramLinkIds([])
    setEngramTracePaths([])
  }, [selectedSessionId, setUsedEngramIds, setUsedEngramLinkIds, setEngramTracePaths])

  useEffect(() => {
    if (!sessionRouteInfo?.sessionId) {
      return
    }
    if (sessionRouteInfo.sessionId === selectedSessionId) {
      return
    }
    setSelectedSessionId(sessionRouteInfo.sessionId)
  }, [sessionRouteInfo, selectedSessionId])

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

  const handleRouteContinue = async () => {
    await handleContinueSession()
    navigate(APP_ROUTES.workspace)
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

  const renderWorkspace = () => (
    <WorkspaceGrid>
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
        onSelectSession={setSelectedSessionId}
        onCreateSession={handleCreateSession}
      />

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

      <RightRail>
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
      </RightRail>
    </WorkspaceGrid>
  )

  const renderWorkspaceRouteHint = () => {
    if (isContinueRoute) {
      return (
        <RouteActionStrip>
          <p>
            Continue this session as a new chat while carrying pinned context. This route is action-specific
            and mirrors the in-chat continuation control.
          </p>
          <button type="button" onClick={() => void handleRouteContinue()} disabled={!selectedSessionId}>
            Continue Session Now
          </button>
        </RouteActionStrip>
      )
    }

    if (isSaveRoute) {
      return (
        <RouteActionStrip>
          <p>
            Save this session as an engram. The dedicated route opens the save flow without leaving your
            continuity context.
          </p>
          <button type="button" onClick={() => setSaveModalOpen(true)} disabled={!selectedSessionId}>
            Open Save Form
          </button>
        </RouteActionStrip>
      )
    }

    if (location.pathname === APP_ROUTES.documentsIngestText || location.pathname === APP_ROUTES.documentsIngestFile) {
      return (
        <RouteActionStrip>
          <p>
            Use the Document Ingestion panel to add text or files, then pin relevant evidence into the
            selected session.
          </p>
        </RouteActionStrip>
      )
    }

    return null
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

    return (
      <AppLayout title={appRouteMeta.title} description={appRouteMeta.description}>
        {renderWorkspaceRouteHint()}
        {renderWorkspace()}
      </AppLayout>
    )
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
