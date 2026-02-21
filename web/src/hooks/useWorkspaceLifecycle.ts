import { useEffect } from 'react'

import { getSessionProfile } from '../api/auth'
import { ApiError } from '../api/http'
import type {
  ChatDebugTrace,
  ChatMessage,
  ChatSourceReference,
  ChatTimelineEvent,
  EngramSummary,
  PinnedDocumentRecord,
  UserProfile,
} from '../api/types'

interface WorkspaceLifecycleConfig {
  projectId: string
  selectedSessionId: string | null
  user: UserProfile | null
  describeError: (error: unknown) => string
  setAuthChecking: (value: boolean) => void
  setAuthError: (value: string | null) => void
  setUser: (value: UserProfile | null) => void
  setMessages: (value: ChatMessage[]) => void
  setPinnedEngrams: (value: EngramSummary[]) => void
  setPinnedDocuments: (value: PinnedDocumentRecord[]) => void
  setSourceReferences: (value: ChatSourceReference[]) => void
  setChatDebugTrace: (value: ChatDebugTrace | null) => void
  setTimelineEvents: (value: ChatTimelineEvent[]) => void
  loadDefaultProject: () => Promise<void>
  loadSessions: (projectId: string, preferredSessionId: string | null) => Promise<void>
  loadProjectDocuments: (projectId: string) => Promise<void>
  loadSessionData: (sessionId: string, projectId: string) => Promise<void>
}

function isUnauthorized(error: unknown): boolean {
  return error instanceof ApiError && error.status === 401
}

function resetSessionDetails(config: WorkspaceLifecycleConfig) {
  config.setMessages([])
  config.setPinnedEngrams([])
  config.setPinnedDocuments([])
  config.setSourceReferences([])
  config.setChatDebugTrace(null)
  config.setTimelineEvents([])
}

function useAuthBootstrap(config: WorkspaceLifecycleConfig) {
  useEffect(() => {
    const run = async () => {
      try {
        const profile = await getSessionProfile()
        config.setUser(profile)
        config.setAuthError(null)
        await Promise.all([
          config.loadDefaultProject(),
          config.loadSessions(config.projectId, null),
          config.loadProjectDocuments(config.projectId),
        ])
      } catch (error) {
        if (!isUnauthorized(error)) {
          config.setAuthError(config.describeError(error))
        }
      } finally {
        config.setAuthChecking(false)
      }
    }
    void run()
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [])
}

function useProjectWorkspaceRefresh(config: WorkspaceLifecycleConfig) {
  useEffect(() => {
    if (!config.user) {
      return
    }
    void Promise.all([
      config.loadSessions(config.projectId, config.selectedSessionId),
      config.loadProjectDocuments(config.projectId),
    ])
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [config.projectId])
}

function useSessionWorkspaceRefresh(config: WorkspaceLifecycleConfig) {
  useEffect(() => {
    if (!config.user || !config.selectedSessionId) {
      resetSessionDetails(config)
      return
    }
    void config.loadSessionData(config.selectedSessionId, config.projectId)
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [config.selectedSessionId])
}

export function useWorkspaceLifecycle(config: WorkspaceLifecycleConfig) {
  useAuthBootstrap(config)
  useProjectWorkspaceRefresh(config)
  useSessionWorkspaceRefresh(config)
}
