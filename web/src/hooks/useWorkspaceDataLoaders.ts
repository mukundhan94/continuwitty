import { useCallback, type Dispatch, type SetStateAction } from 'react'

import {
  listChatSessions,
  listEngrams,
  listPinnedDocuments,
  listPinnedEngrams,
  listSessionMessages,
  listSessionTimeline,
} from '../api/chat'
import { listProjectDocuments } from '../api/ingestion'
import { getDefaultProject } from '../api/projects'
import type {
  ChatMessage,
  ChatSession,
  ChatTimelineEvent,
  DocumentRecord,
  EngramSummary,
  PinnedDocumentRecord,
} from '../api/types'

interface WorkspaceDataLoadersConfig {
  projectId: string
  normalizeProjectId: (value: string) => string
  describeError: (error: unknown) => string
  setProjectId: (value: string) => void
  setDefaultProjectId: (value: string | null) => void
  setSessionsLoading: (value: boolean) => void
  setSessions: (value: ChatSession[]) => void
  setSelectedSessionId: Dispatch<SetStateAction<string | null>>
  setChatError: (value: string | null) => void
  setEngramLoading: (value: boolean) => void
  setMessages: (value: ChatMessage[]) => void
  setPinnedEngrams: (value: EngramSummary[]) => void
  setPinnedDocuments: (value: PinnedDocumentRecord[]) => void
  setAvailableEngrams: (value: EngramSummary[]) => void
  setTimelineEvents: (value: ChatTimelineEvent[]) => void
  setDocumentsLoading: (value: boolean) => void
  setDocumentsError: (value: string | null) => void
  setDocuments: (value: DocumentRecord[]) => void
}

function pickSession(sessions: ChatSession[], previousId: string | null): string | null {
  if (previousId && sessions.some((item) => item.session_id === previousId)) {
    return previousId
  }
  return sessions.length > 0 ? sessions[0].session_id : null
}

function useDefaultProjectLoader(config: WorkspaceDataLoadersConfig) {
  return useCallback(async () => {
    const response = await getDefaultProject()
    config.setDefaultProjectId(response.default_project_id)
    if (response.default_project_id && !config.projectId.trim()) {
      config.setProjectId(response.default_project_id)
    }
  }, [config])
}

function useSessionsLoader(config: WorkspaceDataLoadersConfig) {
  return useCallback(
    async (nextProjectId: string, preferredSessionId: string | null) => {
      config.setSessionsLoading(true)
      try {
        const loaded = await listChatSessions(config.normalizeProjectId(nextProjectId))
        config.setSessions(loaded)
        config.setSelectedSessionId((current) => pickSession(loaded, preferredSessionId ?? current))
      } catch (error) {
        config.setChatError(config.describeError(error))
      } finally {
        config.setSessionsLoading(false)
      }
    },
    [config],
  )
}

function useSessionDataLoader(config: WorkspaceDataLoadersConfig) {
  return useCallback(
    async (sessionId: string, currentProjectId: string) => {
      config.setEngramLoading(true)
      try {
        const [loadedMessages, loadedPinned, loadedPinnedDocuments, loadedEngrams, loadedTimeline] =
          await Promise.all([
            listSessionMessages(sessionId),
            listPinnedEngrams(sessionId),
            listPinnedDocuments(sessionId),
            listEngrams(config.normalizeProjectId(currentProjectId)),
            listSessionTimeline(sessionId),
          ])
        config.setMessages(loadedMessages)
        config.setPinnedEngrams(loadedPinned)
        config.setPinnedDocuments(loadedPinnedDocuments)
        config.setAvailableEngrams(loadedEngrams)
        config.setTimelineEvents(loadedTimeline)
      } catch (error) {
        config.setChatError(config.describeError(error))
      } finally {
        config.setEngramLoading(false)
      }
    },
    [config],
  )
}

function useProjectDocumentsLoader(config: WorkspaceDataLoadersConfig) {
  return useCallback(
    async (currentProjectId: string) => {
      config.setDocumentsLoading(true)
      config.setDocumentsError(null)
      try {
        const loaded = await listProjectDocuments(config.normalizeProjectId(currentProjectId))
        config.setDocuments(loaded)
      } catch (error) {
        config.setDocumentsError(config.describeError(error))
      } finally {
        config.setDocumentsLoading(false)
      }
    },
    [config],
  )
}

function useSessionRefreshLoader(
  projectId: string,
  loadSessionData: (sessionId: string, currentProjectId: string) => Promise<void>,
) {
  return useCallback(
    async (sessionId: string) => {
      await loadSessionData(sessionId, projectId)
    },
    [loadSessionData, projectId],
  )
}

export function useWorkspaceDataLoaders(config: WorkspaceDataLoadersConfig) {
  const loadDefaultProject = useDefaultProjectLoader(config)
  const loadSessions = useSessionsLoader(config)
  const loadSessionData = useSessionDataLoader(config)
  const loadProjectDocuments = useProjectDocumentsLoader(config)
  const refreshFromSession = useSessionRefreshLoader(config.projectId, loadSessionData)

  return {
    loadDefaultProject,
    loadSessions,
    loadSessionData,
    loadProjectDocuments,
    refreshFromSession,
  }
}
