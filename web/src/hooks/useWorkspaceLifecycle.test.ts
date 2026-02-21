import { act, renderHook } from '@testing-library/react'
import { beforeEach, describe, expect, it, vi } from 'vitest'

import { ApiError } from '../api/http'
import { useWorkspaceLifecycle } from './useWorkspaceLifecycle'

const authMocks = vi.hoisted(() => ({
  getSessionProfile: vi.fn(),
}))

vi.mock('../api/auth', async () => {
  const actual = await vi.importActual<typeof import('../api/auth')>('../api/auth')
  return {
    ...actual,
    getSessionProfile: authMocks.getSessionProfile,
  }
})

beforeEach(() => {
  authMocks.getSessionProfile.mockReset()
})

function buildConfig(overrides: Partial<Parameters<typeof useWorkspaceLifecycle>[0]> = {}) {
  return {
    projectId: 'engram-vault',
    selectedSessionId: null,
    user: null,
    describeError: () => 'mapped error',
    setAuthChecking: vi.fn(),
    setAuthError: vi.fn(),
    setUser: vi.fn(),
    setMessages: vi.fn(),
    setPinnedEngrams: vi.fn(),
    setPinnedDocuments: vi.fn(),
    setSourceReferences: vi.fn(),
    setChatDebugTrace: vi.fn(),
    setTimelineEvents: vi.fn(),
    loadDefaultProject: vi.fn(async () => undefined),
    loadSessions: vi.fn(async () => undefined),
    loadProjectDocuments: vi.fn(async () => undefined),
    loadSessionData: vi.fn(async () => undefined),
    ...overrides,
  }
}

async function flushEffects() {
  await act(async () => {
    await Promise.resolve()
  })
}

describe('useWorkspaceLifecycle', () => {
  it('bootstraps profile and initial workspace loaders on mount', async () => {
    const config = buildConfig()
    authMocks.getSessionProfile.mockResolvedValue({
      user_id: 'user-1',
      username: 'admin',
      role: 'admin',
    })

    renderHook(() => useWorkspaceLifecycle(config))
    await flushEffects()

    expect(authMocks.getSessionProfile).toHaveBeenCalledTimes(1)
    expect(config.setUser).toHaveBeenCalledWith(
      expect.objectContaining({ user_id: 'user-1', username: 'admin' }),
    )
    expect(config.setAuthError).toHaveBeenCalledWith(null)
    expect(config.loadDefaultProject).toHaveBeenCalledTimes(1)
    expect(config.loadSessions).toHaveBeenCalledWith('engram-vault', null)
    expect(config.loadProjectDocuments).toHaveBeenCalledWith('engram-vault')
    expect(config.setAuthChecking).toHaveBeenLastCalledWith(false)
  })

  it('maps non-401 bootstrap failures to auth error state', async () => {
    const config = buildConfig({
      describeError: () => 'mapped bootstrap error',
    })
    authMocks.getSessionProfile.mockRejectedValue(new Error('network'))

    renderHook(() => useWorkspaceLifecycle(config))
    await flushEffects()

    expect(config.setAuthError).toHaveBeenCalledWith('mapped bootstrap error')
    expect(config.setAuthChecking).toHaveBeenLastCalledWith(false)
    expect(config.loadDefaultProject).not.toHaveBeenCalled()
  })

  it('does not map bootstrap auth errors for unauthorized responses', async () => {
    const config = buildConfig()
    authMocks.getSessionProfile.mockRejectedValue(new ApiError(401, 'Unauthorized'))

    renderHook(() => useWorkspaceLifecycle(config))
    await flushEffects()

    expect(config.setAuthError).not.toHaveBeenCalledWith('mapped error')
    expect(config.setAuthChecking).toHaveBeenLastCalledWith(false)
  })

  it('refreshes project workspace data when user context exists', async () => {
    const config = buildConfig({
      user: {
        user_id: 'user-1',
        username: 'admin',
        role: 'admin',
      },
      selectedSessionId: 'session-a',
    })
    authMocks.getSessionProfile.mockReturnValue(new Promise(() => {}))

    renderHook(() => useWorkspaceLifecycle(config))
    await flushEffects()

    expect(config.loadSessions).toHaveBeenCalledWith('engram-vault', 'session-a')
    expect(config.loadProjectDocuments).toHaveBeenCalledWith('engram-vault')
  })

  it('resets session details when session selection is empty', async () => {
    const config = buildConfig({
      user: {
        user_id: 'user-1',
        username: 'admin',
        role: 'admin',
      },
      selectedSessionId: null,
    })
    authMocks.getSessionProfile.mockReturnValue(new Promise(() => {}))

    renderHook(() => useWorkspaceLifecycle(config))
    await flushEffects()

    expect(config.setMessages).toHaveBeenCalledWith([])
    expect(config.setPinnedEngrams).toHaveBeenCalledWith([])
    expect(config.setPinnedDocuments).toHaveBeenCalledWith([])
    expect(config.setSourceReferences).toHaveBeenCalledWith([])
    expect(config.setChatDebugTrace).toHaveBeenCalledWith(null)
    expect(config.setTimelineEvents).toHaveBeenCalledWith([])
    expect(config.loadSessionData).not.toHaveBeenCalled()
  })
})
