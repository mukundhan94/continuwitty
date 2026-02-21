import { act, renderHook } from '@testing-library/react'
import { beforeEach, describe, expect, it, vi } from 'vitest'

import type { ChatSession } from '../api/types'
import { useWorkspaceDataLoaders } from './useWorkspaceDataLoaders'

const projectMocks = vi.hoisted(() => ({
  getDefaultProject: vi.fn(),
}))

const chatMocks = vi.hoisted(() => ({
  listChatSessions: vi.fn(),
  listSessionMessages: vi.fn(),
  listPinnedEngrams: vi.fn(),
  listPinnedDocuments: vi.fn(),
  listEngrams: vi.fn(),
  listSessionTimeline: vi.fn(),
}))

const ingestionMocks = vi.hoisted(() => ({
  listProjectDocuments: vi.fn(),
}))

vi.mock('../api/projects', async () => {
  const actual = await vi.importActual<typeof import('../api/projects')>('../api/projects')
  return {
    ...actual,
    getDefaultProject: projectMocks.getDefaultProject,
  }
})

vi.mock('../api/chat', async () => {
  const actual = await vi.importActual<typeof import('../api/chat')>('../api/chat')
  return {
    ...actual,
    listChatSessions: chatMocks.listChatSessions,
    listSessionMessages: chatMocks.listSessionMessages,
    listPinnedEngrams: chatMocks.listPinnedEngrams,
    listPinnedDocuments: chatMocks.listPinnedDocuments,
    listEngrams: chatMocks.listEngrams,
    listSessionTimeline: chatMocks.listSessionTimeline,
  }
})

vi.mock('../api/ingestion', async () => {
  const actual = await vi.importActual<typeof import('../api/ingestion')>('../api/ingestion')
  return {
    ...actual,
    listProjectDocuments: ingestionMocks.listProjectDocuments,
  }
})

beforeEach(() => {
  projectMocks.getDefaultProject.mockReset()
  chatMocks.listChatSessions.mockReset()
  chatMocks.listSessionMessages.mockReset()
  chatMocks.listPinnedEngrams.mockReset()
  chatMocks.listPinnedDocuments.mockReset()
  chatMocks.listEngrams.mockReset()
  chatMocks.listSessionTimeline.mockReset()
  ingestionMocks.listProjectDocuments.mockReset()
})

function buildSession(sessionId: string): ChatSession {
  return {
    session_id: sessionId,
    owner_user_id: 'user-1',
    project_id: 'engram-vault',
    title: sessionId,
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
    created_at: '2026-02-21T00:00:00Z',
    updated_at: '2026-02-21T00:00:00Z',
  }
}

function buildConfig(overrides: Partial<Parameters<typeof useWorkspaceDataLoaders>[0]> = {}) {
  return {
    projectId: 'engram-vault',
    normalizeProjectId: (value: string) => value.trim(),
    describeError: () => 'mapped error',
    setProjectId: vi.fn(),
    setDefaultProjectId: vi.fn(),
    setSessionsLoading: vi.fn(),
    setSessions: vi.fn(),
    setSelectedSessionId: vi.fn(),
    setChatError: vi.fn(),
    setEngramLoading: vi.fn(),
    setMessages: vi.fn(),
    setPinnedEngrams: vi.fn(),
    setPinnedDocuments: vi.fn(),
    setAvailableEngrams: vi.fn(),
    setTimelineEvents: vi.fn(),
    setDocumentsLoading: vi.fn(),
    setDocumentsError: vi.fn(),
    setDocuments: vi.fn(),
    ...overrides,
  }
}

describe('useWorkspaceDataLoaders', () => {
  it('loads default project and backfills empty project input', async () => {
    projectMocks.getDefaultProject.mockResolvedValue({ default_project_id: 'engram-vault' })
    const setProjectId = vi.fn()
    const setDefaultProjectId = vi.fn()
    const config = buildConfig({
      projectId: '   ',
      setProjectId,
      setDefaultProjectId,
    })
    const { result } = renderHook(() => useWorkspaceDataLoaders(config))

    await act(async () => {
      await result.current.loadDefaultProject()
    })

    expect(projectMocks.getDefaultProject).toHaveBeenCalledTimes(1)
    expect(setDefaultProjectId).toHaveBeenCalledWith('engram-vault')
    expect(setProjectId).toHaveBeenCalledWith('engram-vault')
  })

  it('loads sessions and selects preferred session when available', async () => {
    const setSessionsLoading = vi.fn()
    const setSessions = vi.fn()
    const setSelectedSessionId = vi.fn()
    const sessions = [buildSession('session-a'), buildSession('session-b')]
    chatMocks.listChatSessions.mockResolvedValue(sessions)
    const config = buildConfig({
      setSessionsLoading,
      setSessions,
      setSelectedSessionId,
    })
    const { result } = renderHook(() => useWorkspaceDataLoaders(config))

    await act(async () => {
      await result.current.loadSessions('  engram-vault ', 'session-b')
    })

    expect(chatMocks.listChatSessions).toHaveBeenCalledWith('engram-vault')
    expect(setSessionsLoading).toHaveBeenCalledWith(true)
    expect(setSessionsLoading).toHaveBeenLastCalledWith(false)
    expect(setSessions).toHaveBeenCalledWith(sessions)
    const updater = setSelectedSessionId.mock.calls[0][0] as (current: string | null) => string | null
    expect(updater('session-z')).toBe('session-b')
  })

  it('falls back to first session when preferred session is missing', async () => {
    const setSelectedSessionId = vi.fn()
    const sessions = [buildSession('session-a'), buildSession('session-b')]
    chatMocks.listChatSessions.mockResolvedValue(sessions)
    const config = buildConfig({
      setSelectedSessionId,
    })
    const { result } = renderHook(() => useWorkspaceDataLoaders(config))

    await act(async () => {
      await result.current.loadSessions('engram-vault', 'missing-session')
    })

    const updater = setSelectedSessionId.mock.calls[0][0] as (current: string | null) => string | null
    expect(updater('session-b')).toBe('session-a')
  })

  it('loads session data and timeline details for selected session', async () => {
    const setEngramLoading = vi.fn()
    const setMessages = vi.fn()
    const setPinnedEngrams = vi.fn()
    const setPinnedDocuments = vi.fn()
    const setAvailableEngrams = vi.fn()
    const setTimelineEvents = vi.fn()
    chatMocks.listSessionMessages.mockResolvedValue([{ message_id: 'm1' }])
    chatMocks.listPinnedEngrams.mockResolvedValue([{ engram_id: 'e1' }])
    chatMocks.listPinnedDocuments.mockResolvedValue([{ document_id: 'd1' }])
    chatMocks.listEngrams.mockResolvedValue([{ engram_id: 'e2' }])
    chatMocks.listSessionTimeline.mockResolvedValue([{ event_id: 't1' }])
    const config = buildConfig({
      setEngramLoading,
      setMessages,
      setPinnedEngrams,
      setPinnedDocuments,
      setAvailableEngrams,
      setTimelineEvents,
    })
    const { result } = renderHook(() => useWorkspaceDataLoaders(config))

    await act(async () => {
      await result.current.loadSessionData('session-a', '  engram-vault ')
    })

    expect(chatMocks.listEngrams).toHaveBeenCalledWith('engram-vault')
    expect(setEngramLoading).toHaveBeenCalledWith(true)
    expect(setEngramLoading).toHaveBeenLastCalledWith(false)
    expect(setMessages).toHaveBeenCalledWith([{ message_id: 'm1' }])
    expect(setPinnedEngrams).toHaveBeenCalledWith([{ engram_id: 'e1' }])
    expect(setPinnedDocuments).toHaveBeenCalledWith([{ document_id: 'd1' }])
    expect(setAvailableEngrams).toHaveBeenCalledWith([{ engram_id: 'e2' }])
    expect(setTimelineEvents).toHaveBeenCalledWith([{ event_id: 't1' }])
  })

  it('loads project documents and refreshes current session snapshot', async () => {
    const setDocumentsLoading = vi.fn()
    const setDocumentsError = vi.fn()
    const setDocuments = vi.fn()
    chatMocks.listSessionMessages.mockResolvedValue([])
    chatMocks.listPinnedEngrams.mockResolvedValue([])
    chatMocks.listPinnedDocuments.mockResolvedValue([])
    chatMocks.listEngrams.mockResolvedValue([])
    chatMocks.listSessionTimeline.mockResolvedValue([])
    ingestionMocks.listProjectDocuments.mockResolvedValue([{ document_id: 'doc-1' }])
    const config = buildConfig({
      projectId: ' current-project ',
      setDocumentsLoading,
      setDocumentsError,
      setDocuments,
    })
    const { result } = renderHook(() => useWorkspaceDataLoaders(config))

    await act(async () => {
      await result.current.loadProjectDocuments('  engram-vault ')
      await result.current.refreshFromSession('session-a')
    })

    expect(ingestionMocks.listProjectDocuments).toHaveBeenCalledWith('engram-vault')
    expect(setDocumentsLoading).toHaveBeenCalledWith(true)
    expect(setDocumentsLoading).toHaveBeenLastCalledWith(false)
    expect(setDocumentsError).toHaveBeenCalledWith(null)
    expect(setDocuments).toHaveBeenCalledWith([{ document_id: 'doc-1' }])
    expect(chatMocks.listSessionMessages).toHaveBeenCalledWith('session-a')
    expect(chatMocks.listEngrams).toHaveBeenCalledWith('current-project')
  })
})
