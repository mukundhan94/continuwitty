import { useEffect, useMemo, useState } from 'react'
import { Navigate, Route, Routes, useLocation, useNavigate } from 'react-router-dom'
import styled from 'styled-components'

import type {
  ChatMessage,
  ChatDebugTrace,
  ChatSession,
  ChatSourceReference,
  EngramTracePath,
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
import { LinkedEngramPanel } from './components/LinkedEngramPanel'
import { LoginView } from './components/LoginView'
import { PinnedEngramPanel } from './components/PinnedEngramPanel'
import { ProjectTransferPage } from './components/ProjectTransferPage'
import { SaveEngramModal } from './components/SaveEngramModal'
import { SessionSidebar } from './components/SessionSidebar'
import { WorkspaceTopNav } from './components/WorkspaceTopNav'
import { WEB_CONFIG } from './config'
import {
  AppShell,
  LoadingScreen,
  NoticeBanner,
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
import { useLinkedEngramInsights } from './hooks/useLinkedEngramInsights'
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
  grid-template-rows: minmax(10rem, 0.6fr) minmax(10rem, 0.7fr) minmax(0, 1.2fr);
  gap: 0.9rem;

  @media (max-width: 1180px) {
    grid-template-rows: none;
  }
`

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

function AppScreen() {
  const navigate = useNavigate()
  const location = useLocation()
  const isAdminMemoryRoute = location.pathname === '/admin/memory'
  const isProjectTransferRoute = location.pathname === '/projects/transfer'
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

  const {
    handlePin,
    handleUnpin,
    handlePinDocument,
    handleUnpinDocument,
  } = usePinActions({
    selectedSessionId,
    refreshFromSession: workspaceDataLoaders.refreshFromSession,
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
    loadProjectDocuments: workspaceDataLoaders.loadProjectDocuments,
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
    refreshFromSession: workspaceDataLoaders.refreshFromSession,
    setSessions,
    setSelectedSessionId,
    setSaveModalOpen,
    setSaveSubmitting,
    setNotice,
    setChatError,
    describeError,
  })

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
    if (isProjectTransferRoute) {
      return (
        <ProjectTransferPage
          projectId={projectId}
          onProjectChange={(value) => setProjectId(normalizeProjectId(value))}
          onNotice={(message) => setNotice(message)}
        />
      )
    }
    return renderWorkspace()
  }

  const renderSaveModal = () => {
    if (!saveModalOpen) {
      return null
    }
    if (isAdminMemoryRoute) {
      return null
    }
    if (isProjectTransferRoute) {
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
        isOpen={adminTokenActions.adminTokenPanelOpen}
        loading={adminTokenActions.adminTokensLoading}
        optionsLoading={adminTokenActions.adminTokenOptionsLoading}
        creating={adminTokenActions.adminTokensCreating}
        tokens={adminTokenActions.adminTokens}
        latestToken={adminTokenActions.adminLatestToken}
        error={adminTokenActions.adminTokenError}
        availableTools={adminTokenActions.adminAvailableTools}
        availableProjects={adminTokenActions.adminAvailableProjects}
        onClose={adminTokenActions.closeAdminTokenPanel}
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
    return <LoginView isSubmitting={authSubmitting} error={authError} onSubmit={handleLogin} />
  }

  if (isAdminMemoryRoute && !isAdmin) {
    return <Navigate to="/" replace />
  }

  return (
    <AppShell>
      <WorkspaceTopNav
        user={user}
        isAdmin={isAdmin}
        isAdminMemoryRoute={isAdminMemoryRoute}
        isProjectTransferRoute={isProjectTransferRoute}
        mode={mode}
        onOpenAdminTokenPanel={adminTokenActions.openAdminTokenPanel}
        onToggleAdminMemoryRoute={() => navigate(isAdminMemoryRoute ? '/' : '/admin/memory')}
        onToggleProjectTransferRoute={() =>
          navigate(isProjectTransferRoute ? '/' : '/projects/transfer')
        }
        onToggleTheme={toggleMode}
        onLogout={handleLogout}
      />
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
      <Route path="/projects/transfer" element={<AppScreen />} />
      <Route path="*" element={<Navigate to="/" replace />} />
    </Routes>
  )
}
