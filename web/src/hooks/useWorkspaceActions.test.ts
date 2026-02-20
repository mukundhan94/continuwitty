import { act, renderHook } from '@testing-library/react'
import { beforeEach, describe, expect, it, vi } from 'vitest'

import type { ChatSession } from '../api/types'
import { useWorkspaceActions } from './useWorkspaceActions'

const chatMocks = vi.hoisted(() => ({
  createChatSession: vi.fn(),
}))

const projectMocks = vi.hoisted(() => ({
  setDefaultProject: vi.fn(),
}))

vi.mock('../api/chat', async () => {
  const actual = await vi.importActual<typeof import('../api/chat')>('../api/chat')
  return {
    ...actual,
    createChatSession: chatMocks.createChatSession,
  }
})

vi.mock('../api/projects', async () => {
  const actual = await vi.importActual<typeof import('../api/projects')>('../api/projects')
  return {
    ...actual,
    setDefaultProject: projectMocks.setDefaultProject,
  }
})

function buildSession(overrides: Partial<ChatSession> = {}): ChatSession {
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
    created_at: '2026-02-21T00:00:00Z',
    updated_at: '2026-02-21T00:00:00Z',
    ...overrides,
  }
}

beforeEach(() => {
  chatMocks.createChatSession.mockReset()
  projectMocks.setDefaultProject.mockReset()
})

describe('useWorkspaceActions', () => {
  it('creates a session, prepends list, and clears composer', async () => {
    const setCreatingSession = vi.fn()
    const setChatError = vi.fn()
    const setSessions = vi.fn()
    const setSelectedSessionId = vi.fn()
    const setComposerText = vi.fn()
    chatMocks.createChatSession.mockResolvedValue(buildSession({ session_id: 'session-2' }))

    const { result } = renderHook(() =>
      useWorkspaceActions({
        projectId: 'engram-vault',
        normalizeProjectId: (value) => value.trim(),
        describeError: () => 'error',
        setCreatingSession,
        setChatError,
        setSessions,
        setSelectedSessionId,
        setComposerText,
        setSettingDefaultProject: vi.fn(),
        setDefaultProjectId: vi.fn(),
        setNotice: vi.fn(),
      }),
    )

    await act(async () => {
      await result.current.handleCreateSession({
        project_id: 'engram-vault',
        title: 'Incident',
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
      })
    })

    expect(chatMocks.createChatSession).toHaveBeenCalledWith(expect.objectContaining({ title: 'Incident' }))
    expect(setCreatingSession).toHaveBeenCalledWith(true)
    expect(setCreatingSession).toHaveBeenLastCalledWith(false)
    expect(setChatError).toHaveBeenCalledWith(null)
    const updater = setSessions.mock.calls[0][0] as (current: ChatSession[]) => ChatSession[]
    const updated = updater([buildSession({ session_id: 'session-existing' })])
    expect(updated[0].session_id).toBe('session-2')
    expect(setSelectedSessionId).toHaveBeenCalledWith('session-2')
    expect(setComposerText).toHaveBeenCalledWith('')
  })

  it('maps create session errors to chat error state', async () => {
    const setChatError = vi.fn()
    chatMocks.createChatSession.mockRejectedValue(new Error('boom'))

    const { result } = renderHook(() =>
      useWorkspaceActions({
        projectId: 'engram-vault',
        normalizeProjectId: (value) => value.trim(),
        describeError: () => 'mapped create error',
        setCreatingSession: vi.fn(),
        setChatError,
        setSessions: vi.fn(),
        setSelectedSessionId: vi.fn(),
        setComposerText: vi.fn(),
        setSettingDefaultProject: vi.fn(),
        setDefaultProjectId: vi.fn(),
        setNotice: vi.fn(),
      }),
    )

    await act(async () => {
      await result.current.handleCreateSession({
        project_id: 'engram-vault',
        title: 'Incident',
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
      })
    })

    expect(setChatError).toHaveBeenCalledWith('mapped create error')
  })

  it('sets normalized default project and notice', async () => {
    const setSettingDefaultProject = vi.fn()
    const setChatError = vi.fn()
    const setDefaultProjectId = vi.fn()
    const setNotice = vi.fn()
    projectMocks.setDefaultProject.mockResolvedValue({ default_project_id: 'engram-vault' })

    const { result } = renderHook(() =>
      useWorkspaceActions({
        projectId: '  engram-vault ',
        normalizeProjectId: (value) => value.trim(),
        describeError: () => 'error',
        setCreatingSession: vi.fn(),
        setChatError,
        setSessions: vi.fn(),
        setSelectedSessionId: vi.fn(),
        setComposerText: vi.fn(),
        setSettingDefaultProject,
        setDefaultProjectId,
        setNotice,
      }),
    )

    await act(async () => {
      await result.current.handleSetDefaultProject()
    })

    expect(projectMocks.setDefaultProject).toHaveBeenCalledWith('engram-vault')
    expect(setSettingDefaultProject).toHaveBeenCalledWith(true)
    expect(setSettingDefaultProject).toHaveBeenLastCalledWith(false)
    expect(setChatError).toHaveBeenCalledWith(null)
    expect(setDefaultProjectId).toHaveBeenCalledWith('engram-vault')
    expect(setNotice).toHaveBeenCalledWith('Default project set to engram-vault')
  })

  it('maps default-project errors to chat error state', async () => {
    const setChatError = vi.fn()
    projectMocks.setDefaultProject.mockRejectedValue(new Error('boom'))

    const { result } = renderHook(() =>
      useWorkspaceActions({
        projectId: 'engram-vault',
        normalizeProjectId: (value) => value.trim(),
        describeError: () => 'mapped default error',
        setCreatingSession: vi.fn(),
        setChatError,
        setSessions: vi.fn(),
        setSelectedSessionId: vi.fn(),
        setComposerText: vi.fn(),
        setSettingDefaultProject: vi.fn(),
        setDefaultProjectId: vi.fn(),
        setNotice: vi.fn(),
      }),
    )

    await act(async () => {
      await result.current.handleSetDefaultProject()
    })

    expect(setChatError).toHaveBeenCalledWith('mapped default error')
  })
})
