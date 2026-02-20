import { useEffect, useMemo, useState } from 'react'
import { Navigate, Route, Routes, useLocation, useNavigate } from 'react-router-dom'
import styled from 'styled-components'

import { getSessionProfile, loginWithPassword, logoutCurrentUser } from './api/auth'
import {
  getDefaultProject,
  setDefaultProject,
} from './api/projects'
import {
  createChatSession,
  listChatSessions,
  listEngrams,
  listPinnedDocuments,
  listPinnedEngrams,
  listSessionMessages,
  listSessionTimeline,
} from './api/chat'
import { ApiError } from './api/http'
import { listProjectDocuments } from './api/ingestion'
import type {
  ChatMessage,
  ChatDebugTrace,
  ChatSession,
  ChatSourceReference,
  ChatTimelineEvent,
  DocumentRecord,
  EngramSummary,
  PinnedDocumentRecord,
  UserProfile,
} from './api/types'
import { AdminMemoryPage } from './components/AdminMemoryPage'
import { AdminMcpTokenPanel } from './components/AdminMcpTokenPanel'
import { ChatPanel } from './components/ChatPanel'
import { DocumentIngestionPanel } from './components/DocumentIngestionPanel'
import { LoginView } from './components/LoginView'
import { PinnedEngramPanel } from './components/PinnedEngramPanel'
import { SaveEngramModal } from './components/SaveEngramModal'
import { SessionSidebar } from './components/SessionSidebar'
import { WEB_CONFIG } from './config'
import {
  AppShell,
  LoadingScreen,
  NoticeBanner,
  TopNavShell,
  TopNavTitleBlock,
  TopNavUserBlock,
  WorkspaceGrid,
} from './styles/primitives'
import { useThemeMode } from './styles/useThemeMode'
import { useIngestionActions, usePinActions, usePromptActions } from './hooks/useChatActions'
import { useAdminTokenActions } from './hooks/useAdminTokenActions'
import { useSessionActions } from './hooks/useSessionActions'
import { buildDefaultSaveAbstract } from './utils/chat'

const PROJECT_ID_STORAGE_KEY = 'engram.lastProjectId'

function normalizeProjectId(value: string): string {
  const trimmed = value.trim()
  return trimmed || WEB_CONFIG.defaultProjectId
}

function initialProjectId(): string {
  if (typeof window === 'undefined') {
    return WEB_CONFIG.defaultProjectId
  }
  const stored = window.localStorage.getItem(PROJECT_ID_STORAGE_KEY)
  return normalizeProjectId(stored || WEB_CONFIG.defaultProjectId)
}

const RightRail = styled.div`
  min-height: 0;
  display: grid;
  grid-template-rows: minmax(12rem, 0.65fr) minmax(0, 1.35fr);
  gap: 0.9rem;

  @media (max-width: 1180px) {
    grid-template-rows: none;
  }
`

function describeError(error: unknown): string {
  if (error instanceof ApiError) {
    return error.detail
  }
  if (error instanceof Error) {
    return error.message
  }
  return 'Unexpected error'
}

function isUnauthorized(error: unknown): boolean {
  return error instanceof ApiError && error.status === 401
}

function pickSession(sessions: ChatSession[], previousId: string | null): string | null {
  if (previousId && sessions.some((item) => item.session_id === previousId)) {
    return previousId
  }
  return sessions.length > 0 ? sessions[0].session_id : null
}

function AppScreen() {
  const navigate = useNavigate()
  const location = useLocation()
  const isAdminMemoryRoute = location.pathname === '/admin/memory'
  const { mode, toggleMode } = useThemeMode()
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

  const [sourceReferences, setSourceReferences] = useState<ChatSourceReference[]>([])
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
  const defaultSaveAbstract = useMemo(() => buildDefaultSaveAbstract(messages), [messages])
  const isAdmin = user?.role === 'admin'
  const {
    adminTokenPanelOpen,
    adminTokensLoading,
    adminTokensCreating,
    adminTokens,
    adminLatestToken,
    adminTokenError,
    adminTokenOptionsLoading,
    adminAvailableTools,
    adminAvailableProjects,
    openAdminTokenPanel,
    closeAdminTokenPanel,
    handleRefreshAdminTokenPanel,
    handleCreateAdminToken,
    handleRevokeAdminToken,
    resetAdminTokenState,
  } = useAdminTokenActions({
    isAdmin,
    setNotice,
    describeError,
  })

  const loadDefaultProject = async () => {
    const response = await getDefaultProject()
    setDefaultProjectId(response.default_project_id)
    if (response.default_project_id && !projectId.trim()) {
      setProjectId(response.default_project_id)
    }
  }

  const loadSessions = async (nextProjectId: string, preferredSessionId: string | null) => {
    setSessionsLoading(true)
    try {
      const loaded = await listChatSessions(normalizeProjectId(nextProjectId))
      setSessions(loaded)
      setSelectedSessionId((current) => pickSession(loaded, preferredSessionId ?? current))
    } catch (error) {
      setChatError(describeError(error))
    } finally {
      setSessionsLoading(false)
    }
  }

  const loadSessionData = async (sessionId: string, currentProjectId: string) => {
    setEngramLoading(true)
    try {
      const [loadedMessages, loadedPinned, loadedPinnedDocuments, loadedEngrams, loadedTimeline] = await Promise.all([
        listSessionMessages(sessionId),
        listPinnedEngrams(sessionId),
        listPinnedDocuments(sessionId),
        listEngrams(normalizeProjectId(currentProjectId)),
        listSessionTimeline(sessionId),
      ])
      setMessages(loadedMessages)
      setPinnedEngrams(loadedPinned)
      setPinnedDocuments(loadedPinnedDocuments)
      setAvailableEngrams(loadedEngrams)
      setTimelineEvents(loadedTimeline)
    } catch (error) {
      setChatError(describeError(error))
    } finally {
      setEngramLoading(false)
    }
  }

  const loadProjectDocuments = async (currentProjectId: string) => {
    setDocumentsLoading(true)
    setDocumentsError(null)
    try {
      const loaded = await listProjectDocuments(normalizeProjectId(currentProjectId))
      setDocuments(loaded)
    } catch (error) {
      setDocumentsError(describeError(error))
    } finally {
      setDocumentsLoading(false)
    }
  }

  const refreshFromSession = async (sessionId: string) => {
    await loadSessionData(sessionId, projectId)
  }

  useEffect(() => {
    // Keep the current project scope sticky across hard refreshes.
    window.localStorage.setItem(PROJECT_ID_STORAGE_KEY, normalizeProjectId(projectId))
  }, [projectId])

  useEffect(() => {
    const run = async () => {
      try {
        const profile = await getSessionProfile()
        setUser(profile)
        setAuthError(null)
        await Promise.all([
          loadDefaultProject(),
          loadSessions(projectId, null),
          loadProjectDocuments(projectId),
        ])
      } catch (error) {
        if (!isUnauthorized(error)) {
          setAuthError(describeError(error))
        }
      } finally {
        setAuthChecking(false)
      }
    }
    void run()
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [])

  useEffect(() => {
    if (!user) {
      return
    }
    void Promise.all([loadSessions(projectId, selectedSessionId), loadProjectDocuments(projectId)])
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [projectId])

  useEffect(() => {
    if (!user || !selectedSessionId) {
      setMessages([])
      setPinnedEngrams([])
      setPinnedDocuments([])
      setSourceReferences([])
      setChatDebugTrace(null)
      setTimelineEvents([])
      return
    }
    void loadSessionData(selectedSessionId, projectId)
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [selectedSessionId])

  const handleLogin = async (username: string, password: string) => {
    setAuthSubmitting(true)
    setAuthError(null)
    try {
      await loginWithPassword(username, password)
      const profile = await getSessionProfile()
      setUser(profile)
      await Promise.all([
        loadDefaultProject(),
        loadSessions(projectId, null),
        loadProjectDocuments(projectId),
      ])
    } catch (error) {
      setAuthError(describeError(error))
    } finally {
      setAuthSubmitting(false)
    }
  }

  const handleLogout = async () => {
    try {
      await logoutCurrentUser()
    } catch {
      // Ignore local logout errors and reset state.
    }
    setUser(null)
    setSessions([])
    setSelectedSessionId(null)
    setDefaultProjectId(null)
    setMessages([])
    setPinnedEngrams([])
    setPinnedDocuments([])
    setAvailableEngrams([])
    setSourceReferences([])
    setDocuments([])
    setDocumentsError(null)
    setTimelineEvents([])
    setNotice(null)
    setAuthError(null)
    resetAdminTokenState()
  }

  const handleCreateSession = async (payload: {
    project_id: string
    title: string
    provider: 'openai' | 'anthropic' | 'bedrock'
    model_id: string
    system_prompt: string
    visibility_scope: 'private' | 'project'
    autosave_enabled: boolean
    autosave_strategy: 'off' | 'interval' | 'message_count'
    autosave_interval_minutes: number
    autosave_min_messages: number
    retention_days: number
    retention_max_snapshots: number
  }) => {
    setCreatingSession(true)
    setChatError(null)
    try {
      const created = await createChatSession(payload)
      setSessions((current) => [created, ...current])
      setSelectedSessionId(created.session_id)
      setComposerText('')
    } catch (error) {
      setChatError(describeError(error))
    } finally {
      setCreatingSession(false)
    }
  }

  const handleSetDefaultProject = async () => {
    const normalized = normalizeProjectId(projectId)
    setSettingDefaultProject(true)
    setChatError(null)
    try {
      const updated = await setDefaultProject(normalized)
      setDefaultProjectId(updated.default_project_id)
      setNotice(`Default project set to ${updated.default_project_id}`)
    } catch (error) {
      setChatError(describeError(error))
    } finally {
      setSettingDefaultProject(false)
    }
  }

  const { handleSend, handleRetry } = usePromptActions({
    composerText,
    lastPrompt,
    selectedSessionId,
    refreshFromSession,
    setLastPrompt,
    setPendingUserText,
    setComposerText,
    setStreamingAssistantText,
    setSourceReferences,
    setChatDebugTrace,
    setChatError,
    setChatSending,
    describeError,
  })

  const {
    handlePin,
    handleUnpin,
    handlePinDocument,
    handleUnpinDocument,
  } = usePinActions({
    selectedSessionId,
    refreshFromSession,
    setNotice,
    setChatError,
    describeError,
  })

  const {
    handleRefreshDocuments,
    handleIngestText,
    handleIngestFile,
  } = useIngestionActions({
    projectId,
    normalizeProjectId,
    loadProjectDocuments,
    setDocumentsSubmitting,
    setDocumentsError,
    setNotice,
    describeError,
  })

  const {
    handleCopyEngramId,
    handleRefreshEngrams,
    handleContinueSession,
    handleSaveEngram,
  } = useSessionActions({
    selectedSessionId,
    refreshFromSession,
    setSessions,
    setSelectedSessionId,
    setSaveModalOpen,
    setSaveSubmitting,
    setNotice,
    setChatError,
    describeError,
  })

  if (authChecking) {
    return <LoadingScreen>Loading local workspace...</LoadingScreen>
  }

  if (!user) {
    return <LoginView isSubmitting={authSubmitting} error={authError} onSubmit={handleLogin} />
  }

  if (isAdminMemoryRoute && !isAdmin) {
    return <Navigate to="/" replace />
  }

  return (
    <AppShell>
      <TopNavShell>
        <TopNavTitleBlock>
          <h1 className="font-display text-lg font-semibold tracking-tight text-ink">Memory Continuity Workbench</h1>
        </TopNavTitleBlock>

        <TopNavUserBlock>
          <p className="text-sm text-inkMuted">
            {user.username} · {user.role}
          </p>
          {isAdmin ? (
            <button type="button" data-testid="open-admin-token-panel" onClick={() => void openAdminTokenPanel()}>
              MCP Tokens
            </button>
          ) : null}
          {isAdmin ? (
            <button
              type="button"
              onClick={() => navigate(isAdminMemoryRoute ? '/' : '/admin/memory')}
            >
              {isAdminMemoryRoute ? 'Chat Workspace' : 'Memory Admin'}
            </button>
          ) : null}
          <button type="button" onClick={toggleMode}>
            {mode === 'dark' ? 'Light Theme' : 'Dark Theme'}
          </button>
          <button type="button" onClick={handleLogout}>
            Logout
          </button>
        </TopNavUserBlock>
      </TopNavShell>

      {notice ? <NoticeBanner>{notice}</NoticeBanner> : null}

      {isAdminMemoryRoute ? (
        <AdminMemoryPage
          projectId={projectId}
          onProjectChange={(value) => setProjectId(normalizeProjectId(value))}
          onNotice={(message) => setNotice(message)}
        />
      ) : (
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
            debugTrace={chatDebugTrace}
            timelineEvents={timelineEvents}
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
      )}

      {!isAdminMemoryRoute && saveModalOpen ? (
        <SaveEngramModal
          defaultTitle={selectedSession ? `${selectedSession.title} Snapshot` : 'Chat Snapshot'}
          defaultAbstract={defaultSaveAbstract}
          saving={saveSubmitting}
          onClose={() => setSaveModalOpen(false)}
          onSave={handleSaveEngram}
        />
      ) : null}

      {isAdmin ? (
        <AdminMcpTokenPanel
          isOpen={adminTokenPanelOpen}
          loading={adminTokensLoading}
          optionsLoading={adminTokenOptionsLoading}
          creating={adminTokensCreating}
          tokens={adminTokens}
          latestToken={adminLatestToken}
          error={adminTokenError}
          availableTools={adminAvailableTools}
          availableProjects={adminAvailableProjects}
          onClose={closeAdminTokenPanel}
          onRefresh={handleRefreshAdminTokenPanel}
          onCreate={handleCreateAdminToken}
          onRevoke={handleRevokeAdminToken}
        />
      ) : null}
    </AppShell>
  )
}

export default function App() {
  return (
    <Routes>
      <Route path="/" element={<AppScreen />} />
      <Route path="/admin/memory" element={<AppScreen />} />
      <Route path="*" element={<Navigate to="/" replace />} />
    </Routes>
  )
}
