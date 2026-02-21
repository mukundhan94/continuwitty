import { useCallback } from 'react'

import type {
  ChatMessage,
  ChatSession,
  ChatSourceReference,
  ChatTimelineEvent,
  DocumentRecord,
  EngramSummary,
  PinnedDocumentRecord,
  UserProfile,
} from '../api/types'

interface WorkspaceResetConfig {
  setUser: (value: UserProfile | null) => void
  setSessions: (value: ChatSession[]) => void
  setSelectedSessionId: (value: string | null) => void
  setDefaultProjectId: (value: string | null) => void
  setMessages: (value: ChatMessage[]) => void
  setPinnedEngrams: (value: EngramSummary[]) => void
  setPinnedDocuments: (value: PinnedDocumentRecord[]) => void
  setAvailableEngrams: (value: EngramSummary[]) => void
  setSourceReferences: (value: ChatSourceReference[]) => void
  setDocuments: (value: DocumentRecord[]) => void
  setDocumentsError: (value: string | null) => void
  setTimelineEvents: (value: ChatTimelineEvent[]) => void
  setNotice: (value: string | null) => void
  setAuthError: (value: string | null) => void
  resetAdminTokenState: () => void
}

export function useWorkspaceReset(config: WorkspaceResetConfig) {
  return useCallback(() => {
    config.setUser(null)
    config.setSessions([])
    config.setSelectedSessionId(null)
    config.setDefaultProjectId(null)
    config.setMessages([])
    config.setPinnedEngrams([])
    config.setPinnedDocuments([])
    config.setAvailableEngrams([])
    config.setSourceReferences([])
    config.setDocuments([])
    config.setDocumentsError(null)
    config.setTimelineEvents([])
    config.setNotice(null)
    config.setAuthError(null)
    config.resetAdminTokenState()
  }, [config])
}
