import { useMemo, useState } from 'react'
import { Navigate, Route, Routes, useLocation, useNavigate } from 'react-router-dom'
import styled from 'styled-components'

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
import { useAuthActions } from './hooks/useAuthActions'
import { useIngestionActions, usePinActions, usePromptActions } from './hooks/useChatActions'
import { useAdminTokenActions } from './hooks/useAdminTokenActions'
import { useProjectScopePersistence } from './hooks/useProjectScopePersistence'
import { useWorkspaceReset } from './hooks/useWorkspaceReset'
import { useSessionActions } from './hooks/useSessionActions'
import { useWorkspaceDataLoaders } from './hooks/useWorkspaceDataLoaders'
import { useWorkspaceLifecycle } from './hooks/useWorkspaceLifecycle'
import { useWorkspaceActions } from './hooks/useWorkspaceActions'
import { buildDefaultSaveAbstract } from './utils/chat'
import { describeError } from './utils/errors'
import {
  initialProjectId,
  normalizeProjectId,
  PROJECT_ID_STORAGE_KEY,
} from './utils/projectScope'

const RightRail = styled.div`
  min-height: 0;
  display: grid;
  grid-template-rows: minmax(12rem, 0.65fr) minmax(0, 1.35fr);
  gap: 0.9rem;

  @media (max-width: 1180px) {
    grid-template-rows: none;
  }
`

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

  const {
    loadDefaultProject,
    loadSessions,
    loadSessionData,
    loadProjectDocuments,
    refreshFromSession,
  } = useWorkspaceDataLoaders({
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
    loadDefaultProject,
    loadSessions,
    loadProjectDocuments,
    loadSessionData,
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
    resetAdminTokenState,
  })

  const { handleLogin, handleLogout } = useAuthActions({
    projectId,
    loadDefaultProject,
    loadSessions,
    loadProjectDocuments,
    setUser,
    setAuthSubmitting,
    setAuthError,
    describeError,
    resetWorkspaceState,
  })

  const {
    handleCreateSession,
    handleSetDefaultProject,
  } = useWorkspaceActions({
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

  const renderTopNav = (activeUser: UserProfile) => (
    <TopNavShell>
      <TopNavTitleBlock>
        <h1 className="font-display text-lg font-semibold tracking-tight text-ink">Memory Continuity Workbench</h1>
      </TopNavTitleBlock>

      <TopNavUserBlock>
        <p className="text-sm text-inkMuted">
          {activeUser.username} · {activeUser.role}
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
  )

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
  )

  const renderBody = () => {
    if (isAdminMemoryRoute) {
      return (
        <AdminMemoryPage
          projectId={projectId}
          onProjectChange={(value) => setProjectId(normalizeProjectId(value))}
          onNotice={(message) => setNotice(message)}
        />
      )
    }
    return renderWorkspace()
  }

  const renderSaveModal = () => {
    if (isAdminMemoryRoute || !saveModalOpen) {
      return null
    }
    return (
      <SaveEngramModal
        defaultTitle={selectedSession ? `${selectedSession.title} Snapshot` : 'Chat Snapshot'}
        defaultAbstract={defaultSaveAbstract}
        saving={saveSubmitting}
        onClose={() => setSaveModalOpen(false)}
        onSave={handleSaveEngram}
      />
    )
  }

  const renderAdminTokenPanel = () => {
    if (!isAdmin) {
      return null
    }
    return (
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
    )
  }

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
      {renderTopNav(user)}
      {notice ? <NoticeBanner>{notice}</NoticeBanner> : null}
      {renderBody()}
      {renderSaveModal()}
      {renderAdminTokenPanel()}
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
