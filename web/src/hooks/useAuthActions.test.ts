import { act, renderHook } from '@testing-library/react'
import { beforeEach, describe, expect, it, vi } from 'vitest'

import { useAuthActions } from './useAuthActions'

const authMocks = vi.hoisted(() => ({
  loginWithPassword: vi.fn(),
  logoutCurrentUser: vi.fn(),
  getSessionProfile: vi.fn(),
}))

vi.mock('../api/auth', async () => {
  const actual = await vi.importActual<typeof import('../api/auth')>('../api/auth')
  return {
    ...actual,
    loginWithPassword: authMocks.loginWithPassword,
    logoutCurrentUser: authMocks.logoutCurrentUser,
    getSessionProfile: authMocks.getSessionProfile,
  }
})

beforeEach(() => {
  authMocks.loginWithPassword.mockReset()
  authMocks.logoutCurrentUser.mockReset()
  authMocks.getSessionProfile.mockReset()
})

function buildConfig(overrides: Partial<Parameters<typeof useAuthActions>[0]> = {}) {
  return {
    projectId: 'engram-vault',
    loadDefaultProject: vi.fn(async () => undefined),
    loadSessions: vi.fn(async () => undefined),
    loadProjectDocuments: vi.fn(async () => undefined),
    setUser: vi.fn(),
    setAuthSubmitting: vi.fn(),
    setAuthError: vi.fn(),
    describeError: () => 'mapped error',
    resetWorkspaceState: vi.fn(),
    ...overrides,
  }
}

describe('useAuthActions', () => {
  it('logs in and loads workspace data', async () => {
    const config = buildConfig()
    authMocks.loginWithPassword.mockResolvedValue(undefined)
    authMocks.getSessionProfile.mockResolvedValue({
      user_id: 'user-1',
      username: 'admin',
      role: 'admin',
    })
    const { result } = renderHook(() => useAuthActions(config))

    await act(async () => {
      await result.current.handleLogin('admin', 'secret')
    })

    expect(config.setAuthSubmitting).toHaveBeenCalledWith(true)
    expect(config.setAuthSubmitting).toHaveBeenLastCalledWith(false)
    expect(config.setAuthError).toHaveBeenCalledWith(null)
    expect(authMocks.loginWithPassword).toHaveBeenCalledWith('admin', 'secret')
    expect(authMocks.getSessionProfile).toHaveBeenCalledTimes(1)
    expect(config.setUser).toHaveBeenCalledWith(
      expect.objectContaining({ user_id: 'user-1', username: 'admin' }),
    )
    expect(config.loadDefaultProject).toHaveBeenCalledTimes(1)
    expect(config.loadSessions).toHaveBeenCalledWith('engram-vault', null)
    expect(config.loadProjectDocuments).toHaveBeenCalledWith('engram-vault')
  })

  it('maps login failures to auth error state', async () => {
    const config = buildConfig({
      describeError: () => 'mapped login error',
    })
    authMocks.loginWithPassword.mockRejectedValue(new Error('denied'))
    const { result } = renderHook(() => useAuthActions(config))

    await act(async () => {
      await result.current.handleLogin('admin', 'wrong')
    })

    expect(config.setAuthError).toHaveBeenCalledWith('mapped login error')
    expect(authMocks.getSessionProfile).not.toHaveBeenCalled()
    expect(config.loadDefaultProject).not.toHaveBeenCalled()
  })

  async function expectLogoutResetState(logoutError: Error | null): Promise<void> {
    const config = buildConfig()
    if (logoutError) {
      authMocks.logoutCurrentUser.mockRejectedValue(logoutError)
    } else {
      authMocks.logoutCurrentUser.mockResolvedValue(undefined)
    }
    const { result } = renderHook(() => useAuthActions(config))

    await act(async () => {
      await result.current.handleLogout()
    })

    expect(authMocks.logoutCurrentUser).toHaveBeenCalledTimes(1)
    expect(config.resetWorkspaceState).toHaveBeenCalledTimes(1)
  }

  it.each([
    { label: 'logout success', logoutError: null },
    { label: 'logout failure', logoutError: new Error('network') },
  ])('resets workspace state on $label', async ({ logoutError }) => {
    await expectLogoutResetState(logoutError)
  })
})
