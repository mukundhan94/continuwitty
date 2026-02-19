import { useEffect, useMemo, useState } from 'react'
import styled from 'styled-components'

import { getSessionProfile, loginWithPassword, logoutCurrentUser } from './api/auth'
import {
  continueSession,
  createChatSession,
  listChatSessions,
  listEngrams,
  listPinnedDocuments,
  listPinnedEngrams,
  listSessionMessages,
  listSessionTimeline,
  pinDocumentToSession,
  pinEngramToSession,
  saveSessionAsEngram,
  streamChatMessage,
  unpinDocumentFromSession,
  unpinEngramFromSession,
} from './api/chat'
import { ApiError } from './api/http'
import { ingestFileDocument, ingestTextDocument, listProjectDocuments } from './api/ingestion'
import type {
  ChatMessage,
  ChatSession,
  ChatDebugTrace,
  ChatSourceReference,
  ChatTimelineEvent,
  DocumentRecord,
  EngramSummary,
  PinnedDocumentRecord,
  UserProfile,
} from './api/types'
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

export default function App() {
  const { mode, toggleMode } = useThemeMode()
  const [authChecking, setAuthChecking] = useState(true)
  const [authSubmitting, setAuthSubmitting] = useState(false)
  const [authError, setAuthError] = useState<string | null>(null)
  const [user, setUser] = useState<UserProfile | null>(null)

  const [projectId, setProjectId] = useState(initialProjectId)

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
        await Promise.all([loadSessions(projectId, null), loadProjectDocuments(projectId)])
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
      await Promise.all([loadSessions(projectId, null), loadProjectDocuments(projectId)])
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

  const sendPrompt = async (prompt: string) => {
    if (!selectedSessionId) {
      return
    }

    const content = prompt.trim()
    if (!content) {
      return
    }

    setLastPrompt(content)
    setPendingUserText(content)
    setComposerText('')
    setStreamingAssistantText('')
    setSourceReferences([])
    setChatDebugTrace(null)
    setChatError(null)
    setChatSending(true)

    try {
      for await (const event of streamChatMessage(selectedSessionId, content)) {
        if (event.event === 'meta') {
          setSourceReferences(event.data.source_references)
          if (event.data.debug_trace) {
            setChatDebugTrace(event.data.debug_trace)
          }
        }
        if (event.event === 'chunk') {
          setStreamingAssistantText((current) => current + event.data.text)
        }
        if (event.event === 'done') {
          setStreamingAssistantText(event.data.assistant_text)
          setSourceReferences(event.data.source_references)
          setChatDebugTrace(event.data.debug_trace ?? null)
        }
        if (event.event === 'error') {
          throw new Error(event.data.detail)
        }
      }
      await refreshFromSession(selectedSessionId)
      setPendingUserText(null)
      setStreamingAssistantText('')
    } catch (error) {
      setChatError(describeError(error))
      setComposerText(content)
      setPendingUserText(null)
      setStreamingAssistantText('')
      setChatDebugTrace(null)
    } finally {
      setChatSending(false)
    }
  }

  const handleSend = async () => sendPrompt(composerText)

  const handleRetry = async () => {
    if (!lastPrompt.trim()) {
      setChatError('No previous prompt available to retry.')
      return
    }
    await sendPrompt(lastPrompt)
  }

  const handlePin = async (engramId: string) => {
    if (!selectedSessionId) {
      return
    }
    try {
      await pinEngramToSession(selectedSessionId, engramId)
      await refreshFromSession(selectedSessionId)
      setNotice(`Pinned engram ${engramId}`)
    } catch (error) {
      setChatError(describeError(error))
    }
  }

  const handleUnpin = async (engramId: string) => {
    if (!selectedSessionId) {
      return
    }
    try {
      await unpinEngramFromSession(selectedSessionId, engramId)
      await refreshFromSession(selectedSessionId)
      setNotice(`Unpinned engram ${engramId}`)
    } catch (error) {
      setChatError(describeError(error))
    }
  }

  const handleCopyEngramId = async (engramId: string) => {
    try {
      await navigator.clipboard.writeText(engramId)
      setNotice(`Copied engram id: ${engramId}`)
    } catch {
      setChatError('Clipboard access failed. Copy manually from the card.')
    }
  }

  const handleRefreshEngrams = async () => {
    if (!selectedSessionId) {
      return
    }
    await refreshFromSession(selectedSessionId)
  }

  const handleRefreshDocuments = async () => {
    await loadProjectDocuments(projectId)
  }

  const handlePinDocument = async (documentId: string) => {
    if (!selectedSessionId) {
      setChatError('Select a session before pinning a document.')
      return
    }
    try {
      await pinDocumentToSession(selectedSessionId, documentId)
      await refreshFromSession(selectedSessionId)
      setNotice(`Pinned document ${documentId}`)
    } catch (error) {
      setChatError(describeError(error))
    }
  }

  const handleUnpinDocument = async (documentId: string) => {
    if (!selectedSessionId) {
      return
    }
    try {
      await unpinDocumentFromSession(selectedSessionId, documentId)
      await refreshFromSession(selectedSessionId)
      setNotice(`Unpinned document ${documentId}`)
    } catch (error) {
      setChatError(describeError(error))
    }
  }

  const handleIngestText = async (payload: {
    title: string
    text: string
    visibility_scope: 'private' | 'project'
    chunk_size_chars: number
    chunk_overlap_chars: number
  }) => {
    setDocumentsSubmitting(true)
    setDocumentsError(null)
    const normalizedProjectId = normalizeProjectId(projectId)
    try {
      const created = await ingestTextDocument({
        project_id: normalizedProjectId,
        ...payload,
      })
      setNotice(`Ingested text document ${created.title} (${created.chunk_count} chunks)`)
      await loadProjectDocuments(normalizedProjectId)
    } catch (error) {
      setDocumentsError(describeError(error))
    } finally {
      setDocumentsSubmitting(false)
    }
  }

  const handleIngestFile = async (payload: {
    title: string
    file: File
    visibility_scope: 'private' | 'project'
    chunk_size_chars: number
    chunk_overlap_chars: number
  }) => {
    setDocumentsSubmitting(true)
    setDocumentsError(null)
    const normalizedProjectId = normalizeProjectId(projectId)
    try {
      const created = await ingestFileDocument({
        project_id: normalizedProjectId,
        ...payload,
      })
      setNotice(`Ingested file ${created.source_name || created.title} (${created.chunk_count} chunks)`)
      await loadProjectDocuments(normalizedProjectId)
    } catch (error) {
      setDocumentsError(describeError(error))
    } finally {
      setDocumentsSubmitting(false)
    }
  }

  const handleContinueSession = async () => {
    if (!selectedSessionId) {
      return
    }
    try {
      const continued = await continueSession(selectedSessionId)
      setSessions((current) => [continued.session, ...current])
      setSelectedSessionId(continued.session.session_id)
      setNotice(`Created continuation with ${continued.carried_engram_ids.length} carried engrams.`)
    } catch (error) {
      setChatError(describeError(error))
    }
  }

  const handleSaveEngram = async (payload: {
    title: string
    abstract: string
    visibility_scope: 'private' | 'project'
    tags: string[]
    keywords: string[]
  }) => {
    if (!selectedSessionId) {
      return
    }
    setSaveSubmitting(true)
    try {
      const created = await saveSessionAsEngram(selectedSessionId, payload)
      setNotice(`Saved session as engram ${created.engram_id}`)
      setSaveModalOpen(false)
      await refreshFromSession(selectedSessionId)
    } catch (error) {
      setChatError(describeError(error))
    } finally {
      setSaveSubmitting(false)
    }
  }

  if (authChecking) {
    return <LoadingScreen>Loading local workspace...</LoadingScreen>
  }

  if (!user) {
    return <LoginView isSubmitting={authSubmitting} error={authError} onSubmit={handleLogin} />
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
          <button type="button" onClick={toggleMode}>
            {mode === 'dark' ? 'Light Theme' : 'Dark Theme'}
          </button>
          <button type="button" onClick={handleLogout}>
            Logout
          </button>
        </TopNavUserBlock>
      </TopNavShell>

      {notice ? <NoticeBanner>{notice}</NoticeBanner> : null}

      <WorkspaceGrid>
        <SessionSidebar
          sessions={sessions}
          selectedSessionId={selectedSessionId}
          projectId={projectId}
          defaultProvider={WEB_CONFIG.defaultProvider}
          defaultVisibilityScope={WEB_CONFIG.defaultVisibility}
          modelDefaults={WEB_CONFIG.defaultModelByProvider}
          loading={sessionsLoading}
          creating={creatingSession}
          onProjectChange={(value) => setProjectId(normalizeProjectId(value))}
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

      {saveModalOpen ? (
        <SaveEngramModal
          defaultTitle={selectedSession ? `${selectedSession.title} Snapshot` : 'Chat Snapshot'}
          defaultAbstract={defaultSaveAbstract}
          saving={saveSubmitting}
          onClose={() => setSaveModalOpen(false)}
          onSave={handleSaveEngram}
        />
      ) : null}
    </AppShell>
  )
}
