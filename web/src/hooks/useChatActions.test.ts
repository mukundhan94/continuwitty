import { act, renderHook } from '@testing-library/react'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'

import { useIngestionActions, usePinActions, usePromptActions } from './useChatActions'

const chatMocks = vi.hoisted(() => ({
  streamChatMessage: vi.fn(),
  pinEngramToSession: vi.fn(),
  unpinEngramFromSession: vi.fn(),
  pinDocumentToSession: vi.fn(),
  unpinDocumentFromSession: vi.fn(),
}))

const ingestionMocks = vi.hoisted(() => ({
  ingestTextDocument: vi.fn(),
  ingestFileDocument: vi.fn(),
}))

vi.mock('../api/chat', async () => {
  const actual = await vi.importActual<typeof import('../api/chat')>('../api/chat')
  return {
    ...actual,
    streamChatMessage: chatMocks.streamChatMessage,
    pinEngramToSession: chatMocks.pinEngramToSession,
    unpinEngramFromSession: chatMocks.unpinEngramFromSession,
    pinDocumentToSession: chatMocks.pinDocumentToSession,
    unpinDocumentFromSession: chatMocks.unpinDocumentFromSession,
  }
})

vi.mock('../api/ingestion', async () => {
  const actual = await vi.importActual<typeof import('../api/ingestion')>('../api/ingestion')
  return {
    ...actual,
    ingestTextDocument: ingestionMocks.ingestTextDocument,
    ingestFileDocument: ingestionMocks.ingestFileDocument,
  }
})

beforeEach(() => {
  chatMocks.streamChatMessage.mockReset()
  chatMocks.pinEngramToSession.mockReset()
  chatMocks.unpinEngramFromSession.mockReset()
  chatMocks.pinDocumentToSession.mockReset()
  chatMocks.unpinDocumentFromSession.mockReset()
  ingestionMocks.ingestTextDocument.mockReset()
  ingestionMocks.ingestFileDocument.mockReset()
})

afterEach(() => {
  vi.restoreAllMocks()
})

describe('usePromptActions', () => {
  it('reports retry error when no previous prompt exists', async () => {
    const setChatError = vi.fn()
    const { result } = renderHook(() =>
      usePromptActions({
        composerText: '',
        lastPrompt: '   ',
        selectedSessionId: null,
        refreshFromSession: vi.fn(),
        setLastPrompt: vi.fn(),
        setPendingUserText: vi.fn(),
        setComposerText: vi.fn(),
        setStreamingAssistantText: vi.fn(),
        setSourceReferences: vi.fn(),
        setChatDebugTrace: vi.fn(),
        setChatError,
        setChatSending: vi.fn(),
        describeError: () => 'error',
      }),
    )

    await act(async () => {
      await result.current.handleRetry()
    })

    expect(setChatError).toHaveBeenCalledWith('No previous prompt available to retry.')
  })
})

describe('usePinActions', () => {
  it('requires active session before pinning a document', async () => {
    const setChatError = vi.fn()
    const { result } = renderHook(() =>
      usePinActions({
        selectedSessionId: null,
        refreshFromSession: vi.fn(),
        setNotice: vi.fn(),
        setChatError,
        describeError: () => 'error',
      }),
    )

    await act(async () => {
      await result.current.handlePinDocument('doc-1')
    })

    expect(setChatError).toHaveBeenCalledWith('Select a session before pinning a document.')
    expect(chatMocks.pinDocumentToSession).not.toHaveBeenCalled()
  })
})

describe('useIngestionActions', () => {
  it('refreshes project documents in current project scope', async () => {
    const loadProjectDocuments = vi.fn(async () => undefined)
    const { result } = renderHook(() =>
      useIngestionActions({
        projectId: 'engram-vault',
        normalizeProjectId: (value) => value,
        loadProjectDocuments,
        setDocumentsSubmitting: vi.fn(),
        setDocumentsError: vi.fn(),
        setNotice: vi.fn(),
        describeError: () => 'error',
      }),
    )

    await act(async () => {
      await result.current.handleRefreshDocuments()
    })

    expect(loadProjectDocuments).toHaveBeenCalledWith('engram-vault')
  })

  it('ingests text and refreshes normalized project documents', async () => {
    const loadProjectDocuments = vi.fn(async () => undefined)
    const setNotice = vi.fn()
    const setDocumentsSubmitting = vi.fn()
    const setDocumentsError = vi.fn()
    ingestionMocks.ingestTextDocument.mockResolvedValue({
      title: 'Incident',
      chunk_count: 3,
    })

    const { result } = renderHook(() =>
      useIngestionActions({
        projectId: 'engram-vault',
        normalizeProjectId: (value) => value.trim(),
        loadProjectDocuments,
        setDocumentsSubmitting,
        setDocumentsError,
        setNotice,
        describeError: () => 'error',
      }),
    )

    await act(async () => {
      await result.current.handleIngestText({
        title: 'Incident',
        text: 'Incident details',
        visibility_scope: 'private',
        chunk_size_chars: 800,
        chunk_overlap_chars: 100,
      })
    })

    expect(ingestionMocks.ingestTextDocument).toHaveBeenCalledWith(
      expect.objectContaining({ project_id: 'engram-vault' }),
    )
    expect(loadProjectDocuments).toHaveBeenCalledWith('engram-vault')
    expect(setNotice).toHaveBeenCalledWith('Ingested text document Incident (3 chunks)')
    expect(setDocumentsSubmitting).toHaveBeenCalledWith(true)
    expect(setDocumentsSubmitting).toHaveBeenLastCalledWith(false)
    expect(setDocumentsError).toHaveBeenCalledWith(null)
  })
})
