import { act, renderHook } from '@testing-library/react'
import { beforeEach, describe, expect, it, vi } from 'vitest'

import type { ChatSession } from '../api/types'
import { useSessionActions } from './useSessionActions'

type SessionActionsConfig = Parameters<typeof useSessionActions>[0]

const chatMocks = vi.hoisted(() => ({
  continueSession: vi.fn(),
  saveSessionAsEngram: vi.fn(),
}))

vi.mock('../api/chat', async () => {
  const actual = await vi.importActual<typeof import('../api/chat')>('../api/chat')
  return {
    ...actual,
    continueSession: chatMocks.continueSession,
    saveSessionAsEngram: chatMocks.saveSessionAsEngram,
  }
})

function buildSession(overrides: Partial<ChatSession>): ChatSession {
  return {
    session_id: 'session-1',
    owner_user_id: 'user-1',
    project_id: 'engram-vault',
    title: 'Session',
    provider: 'openai',
    model_id: 'gpt-4o-mini',
    system_prompt: '',
    visibility_scope: 'private',
    autosave_enabled: false,
    autosave_strategy: 'off',
    autosave_interval_minutes: 30,
    autosave_min_messages: 6,
    retention_days: 30,
    retention_max_snapshots: 60,
    created_at: '2026-02-01T00:00:00Z',
    updated_at: '2026-02-01T00:00:00Z',
    ...overrides,
  }
}

beforeEach(() => {
  chatMocks.continueSession.mockReset()
  chatMocks.saveSessionAsEngram.mockReset()
})

function buildConfig(overrides: Partial<SessionActionsConfig> = {}): SessionActionsConfig {
  return {
    selectedSessionId: 'session-1',
    refreshFromSession: vi.fn(async () => undefined),
    setSessions: vi.fn(),
    setSelectedSessionId: vi.fn(),
    setSaveModalOpen: vi.fn(),
    setSaveSubmitting: vi.fn(),
    setNotice: vi.fn(),
    setChatError: vi.fn(),
    describeError: () => 'error',
    copyText: vi.fn(async () => undefined),
    ...overrides,
  }
}

function renderSessionActions(overrides: Partial<SessionActionsConfig> = {}) {
  const config = buildConfig(overrides)
  const rendered = renderHook(() => useSessionActions(config))
  return { ...rendered, config }
}

describe('useSessionActions', () => {
  it('refreshes engrams only when a session is selected', async () => {
    const refreshFromSession = vi.fn(async () => undefined)
    const { result } = renderSessionActions({
      selectedSessionId: null,
      refreshFromSession,
    })

    await act(async () => {
      await result.current.handleRefreshEngrams()
    })

    expect(refreshFromSession).not.toHaveBeenCalled()
  })

  it('continues selected session and prepends continuation in list', async () => {
    const setSessions = vi.fn()
    const setSelectedSessionId = vi.fn()
    const setNotice = vi.fn()
    chatMocks.continueSession.mockResolvedValue({
      session: buildSession({ session_id: 'session-2', title: 'Continuation' }),
      carried_engram_ids: ['engram-1', 'engram-2'],
    })

    const { result } = renderSessionActions({
      setSessions,
      setSelectedSessionId,
      setNotice,
    })

    await act(async () => {
      await result.current.handleContinueSession()
    })

    expect(chatMocks.continueSession).toHaveBeenCalledWith('session-1')
    expect(setSessions).toHaveBeenCalledTimes(1)
    const updater = setSessions.mock.calls[0][0] as (current: ChatSession[]) => ChatSession[]
    const updated = updater([buildSession({ session_id: 'existing' })])
    expect(updated[0].session_id).toBe('session-2')
    expect(setSelectedSessionId).toHaveBeenCalledWith('session-2')
    expect(setNotice).toHaveBeenCalledWith('Created continuation with 2 carried engrams.')
  })

  it('saves selected session as engram and refreshes session data', async () => {
    const setSaveSubmitting = vi.fn()
    const setSaveModalOpen = vi.fn()
    const setNotice = vi.fn()
    const refreshFromSession = vi.fn(async () => undefined)
    chatMocks.saveSessionAsEngram.mockResolvedValue({
      engram_id: 'engram-1',
      session_id: 'session-1',
      created_at: '2026-02-20T00:00:00Z',
    })

    const { result } = renderSessionActions({
      refreshFromSession,
      setSaveModalOpen,
      setSaveSubmitting,
      setNotice,
    })

    await act(async () => {
      await result.current.handleSaveEngram({
        title: 'Snapshot',
        abstract: 'Summary',
        visibility_scope: 'private',
        tags: ['tag'],
        keywords: ['keyword'],
      })
    })

    expect(chatMocks.saveSessionAsEngram).toHaveBeenCalledWith(
      'session-1',
      expect.objectContaining({ title: 'Snapshot' }),
    )
    expect(setSaveSubmitting).toHaveBeenCalledWith(true)
    expect(setSaveSubmitting).toHaveBeenLastCalledWith(false)
    expect(setSaveModalOpen).toHaveBeenCalledWith(false)
    expect(setNotice).toHaveBeenCalledWith('Saved session as engram engram-1')
    expect(refreshFromSession).toHaveBeenCalledWith('session-1')
  })

  it('copies engram id and posts notice', async () => {
    const copyText = vi.fn(async () => undefined)
    const setNotice = vi.fn()
    const { result } = renderSessionActions({
      copyText,
      setNotice,
    })

    await act(async () => {
      await result.current.handleCopyEngramId('engram-1')
    })

    expect(copyText).toHaveBeenCalledWith('engram-1')
    expect(setNotice).toHaveBeenCalledWith('Copied engram id: engram-1')
  })

  it('reports clipboard error when copy fails', async () => {
    const copyText = vi.fn(async () => {
      throw new Error('blocked')
    })
    const setChatError = vi.fn()
    const { result } = renderSessionActions({
      copyText,
      setChatError,
    })

    await act(async () => {
      await result.current.handleCopyEngramId('engram-2')
    })

    expect(setChatError).toHaveBeenCalledWith('Clipboard access failed. Copy manually from the card.')
  })
})
