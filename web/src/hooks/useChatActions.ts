import { useCallback, type Dispatch, type SetStateAction } from 'react'

import {
  type ChatRecallOptions,
  pinDocumentToSession,
  pinEngramToSession,
  streamChatMessage,
  type ChatStreamEvent,
  type StreamDonePayload,
  type StreamMetaPayload,
  unpinDocumentFromSession,
  unpinEngramFromSession,
} from '../api/chat'
import { ingestFileDocument, ingestTextDocument } from '../api/ingestion'
import type { ChatDebugTrace, ChatSourceReference, EngramTracePath } from '../api/types'

interface PromptActionsConfig {
  composerText: string
  lastPrompt: string
  selectedSessionId: string | null
  refreshFromSession: (sessionId: string) => Promise<void>
  setLastPrompt: (value: string) => void
  setPendingUserText: (value: string | null) => void
  setComposerText: (value: string) => void
  setStreamingAssistantText: Dispatch<SetStateAction<string>>
  setSourceReferences: Dispatch<SetStateAction<ChatSourceReference[]>>
  setUsedEngramIds?: Dispatch<SetStateAction<string[]>>
  setUsedEngramLinkIds?: Dispatch<SetStateAction<string[]>>
  setEngramTracePaths?: Dispatch<SetStateAction<EngramTracePath[]>>
  setChatDebugTrace: Dispatch<SetStateAction<ChatDebugTrace | null>>
  setChatError: (value: string | null) => void
  setChatSending: (value: boolean) => void
  recallOptions?: ChatRecallOptions
  describeError: (error: unknown) => string
}

interface PinActionsConfig {
  selectedSessionId: string | null
  refreshFromSession: (sessionId: string) => Promise<void>
  setNotice: (value: string | null) => void
  setChatError: (value: string | null) => void
  describeError: (error: unknown) => string
}

interface IngestTextPayload {
  title: string
  text: string
  visibility_scope: 'private' | 'project'
  chunk_size_chars: number
  chunk_overlap_chars: number
}

interface IngestFilePayload {
  title: string
  file: File
  visibility_scope: 'private' | 'project'
  chunk_size_chars: number
  chunk_overlap_chars: number
}

interface IngestionActionsConfig {
  projectId: string
  normalizeProjectId: (value: string) => string
  loadProjectDocuments: (projectId: string) => Promise<void>
  setDocumentsSubmitting: (value: boolean) => void
  setDocumentsError: (value: string | null) => void
  setNotice: (value: string | null) => void
  describeError: (error: unknown) => string
}

function beginPromptStreaming(config: PromptActionsConfig, content: string): void {
  config.setLastPrompt(content)
  config.setPendingUserText(content)
  config.setComposerText('')
  config.setStreamingAssistantText('')
  config.setSourceReferences([])
  config.setUsedEngramIds?.([])
  config.setUsedEngramLinkIds?.([])
  config.setEngramTracePaths?.([])
  config.setChatDebugTrace(null)
  config.setChatError(null)
  config.setChatSending(true)
}

function applyPromptMetaEvent(config: PromptActionsConfig, payload: StreamMetaPayload): void {
  config.setUsedEngramIds?.(payload.used_engram_ids)
  config.setSourceReferences(payload.source_references)
  config.setUsedEngramLinkIds?.(payload.used_engram_link_ids ?? [])
  config.setEngramTracePaths?.(payload.engram_trace_paths ?? [])
  if (payload.debug_trace) {
    config.setChatDebugTrace(payload.debug_trace)
  }
}

function applyPromptDoneEvent(config: PromptActionsConfig, payload: StreamDonePayload): void {
  config.setStreamingAssistantText(payload.assistant_text)
  config.setUsedEngramIds?.(payload.used_engram_ids)
  config.setSourceReferences(payload.source_references)
  config.setUsedEngramLinkIds?.(payload.used_engram_link_ids ?? [])
  config.setEngramTracePaths?.(payload.engram_trace_paths ?? [])
  config.setChatDebugTrace(payload.debug_trace ?? null)
}

function applyPromptStreamEvent(config: PromptActionsConfig, event: ChatStreamEvent): void {
  if (event.event === 'meta') {
    applyPromptMetaEvent(config, event.data)
    return
  }
  if (event.event === 'chunk') {
    config.setStreamingAssistantText((current) => current + event.data.text)
    return
  }
  if (event.event === 'done') {
    applyPromptDoneEvent(config, event.data)
    return
  }
  if (event.event === 'error') {
    throw new Error(event.data.detail)
  }
}

function finalizePromptStreamSuccess(config: PromptActionsConfig): void {
  config.setPendingUserText(null)
  config.setStreamingAssistantText('')
}

function finalizePromptStreamError(
  config: PromptActionsConfig,
  content: string,
  error: unknown,
): void {
  config.setChatError(config.describeError(error))
  config.setComposerText(content)
  config.setPendingUserText(null)
  config.setStreamingAssistantText('')
  config.setUsedEngramIds?.([])
  config.setUsedEngramLinkIds?.([])
  config.setEngramTracePaths?.([])
  config.setChatDebugTrace(null)
}

interface SessionMutationRequest {
  selectedSessionId: string | null
  missingSessionMessage: string | null
  mutate: (sessionId: string) => Promise<void>
  notice: string
  refreshFromSession: (sessionId: string) => Promise<void>
  setNotice: (value: string | null) => void
  setChatError: (value: string | null) => void
  describeError: (error: unknown) => string
}

async function runSessionMutation(request: SessionMutationRequest): Promise<void> {
  if (!request.selectedSessionId) {
    if (request.missingSessionMessage) {
      request.setChatError(request.missingSessionMessage)
    }
    return
  }

  try {
    await request.mutate(request.selectedSessionId)
    await request.refreshFromSession(request.selectedSessionId)
    request.setNotice(request.notice)
  } catch (error) {
    request.setChatError(request.describeError(error))
  }
}

export function usePromptActions(config: PromptActionsConfig) {
  const sendPrompt = useCallback(
    async (prompt: string) => {
      if (!config.selectedSessionId) {
        return
      }

      const content = prompt.trim()
      if (!content) {
        return
      }

      beginPromptStreaming(config, content)

      try {
        for await (const event of streamChatMessage(config.selectedSessionId, content, config.recallOptions)) {
          applyPromptStreamEvent(config, event)
        }

        await config.refreshFromSession(config.selectedSessionId)
        finalizePromptStreamSuccess(config)
      } catch (error) {
        finalizePromptStreamError(config, content, error)
      } finally {
        config.setChatSending(false)
      }
    },
    [config],
  )

  const handleSend = useCallback(async () => sendPrompt(config.composerText), [sendPrompt, config.composerText])

  const handleRetry = useCallback(async () => {
    if (!config.lastPrompt.trim()) {
      config.setChatError('No previous prompt available to retry.')
      return
    }
    await sendPrompt(config.lastPrompt)
  }, [config, sendPrompt])

  return {
    sendPrompt,
    handleSend,
    handleRetry,
  }
}

export function usePinActions(config: PinActionsConfig) {
  const handlePin = useCallback(
    async (engramId: string) => {
      await runSessionMutation({
        selectedSessionId: config.selectedSessionId,
        missingSessionMessage: null,
        mutate: async (sessionId) => pinEngramToSession(sessionId, engramId),
        notice: `Pinned engram ${engramId}`,
        refreshFromSession: config.refreshFromSession,
        setNotice: config.setNotice,
        setChatError: config.setChatError,
        describeError: config.describeError,
      })
    },
    [config],
  )

  const handleUnpin = useCallback(
    async (engramId: string) => {
      await runSessionMutation({
        selectedSessionId: config.selectedSessionId,
        missingSessionMessage: null,
        mutate: async (sessionId) => unpinEngramFromSession(sessionId, engramId),
        notice: `Unpinned engram ${engramId}`,
        refreshFromSession: config.refreshFromSession,
        setNotice: config.setNotice,
        setChatError: config.setChatError,
        describeError: config.describeError,
      })
    },
    [config],
  )

  const handlePinDocument = useCallback(
    async (documentId: string) => {
      await runSessionMutation({
        selectedSessionId: config.selectedSessionId,
        missingSessionMessage: 'Select a session before pinning a document.',
        mutate: async (sessionId) => pinDocumentToSession(sessionId, documentId),
        notice: `Pinned document ${documentId}`,
        refreshFromSession: config.refreshFromSession,
        setNotice: config.setNotice,
        setChatError: config.setChatError,
        describeError: config.describeError,
      })
    },
    [config],
  )

  const handleUnpinDocument = useCallback(
    async (documentId: string) => {
      await runSessionMutation({
        selectedSessionId: config.selectedSessionId,
        missingSessionMessage: null,
        mutate: async (sessionId) => unpinDocumentFromSession(sessionId, documentId),
        notice: `Unpinned document ${documentId}`,
        refreshFromSession: config.refreshFromSession,
        setNotice: config.setNotice,
        setChatError: config.setChatError,
        describeError: config.describeError,
      })
    },
    [config],
  )

  return {
    handlePin,
    handleUnpin,
    handlePinDocument,
    handleUnpinDocument,
  }
}

export function useIngestionActions(config: IngestionActionsConfig) {
  const handleRefreshDocuments = useCallback(async () => {
    await config.loadProjectDocuments(config.projectId)
  }, [config])

  const handleIngestText = useCallback(
    async (payload: IngestTextPayload) => {
      config.setDocumentsSubmitting(true)
      config.setDocumentsError(null)
      const normalizedProjectId = config.normalizeProjectId(config.projectId)
      try {
        const created = await ingestTextDocument({
          project_id: normalizedProjectId,
          ...payload,
        })
        config.setNotice(`Ingested text document ${created.title} (${created.chunk_count} chunks)`)
        await config.loadProjectDocuments(normalizedProjectId)
      } catch (error) {
        config.setDocumentsError(config.describeError(error))
      } finally {
        config.setDocumentsSubmitting(false)
      }
    },
    [config],
  )

  const handleIngestFile = useCallback(
    async (payload: IngestFilePayload) => {
      config.setDocumentsSubmitting(true)
      config.setDocumentsError(null)
      const normalizedProjectId = config.normalizeProjectId(config.projectId)
      try {
        const created = await ingestFileDocument({
          project_id: normalizedProjectId,
          ...payload,
        })
        config.setNotice(`Ingested file ${created.source_name || created.title} (${created.chunk_count} chunks)`)
        await config.loadProjectDocuments(normalizedProjectId)
      } catch (error) {
        config.setDocumentsError(config.describeError(error))
      } finally {
        config.setDocumentsSubmitting(false)
      }
    },
    [config],
  )

  return {
    handleRefreshDocuments,
    handleIngestText,
    handleIngestFile,
  }
}
