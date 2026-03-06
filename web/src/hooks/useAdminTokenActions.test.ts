import { act, renderHook } from '@testing-library/react'
import { beforeEach, describe, expect, it, vi } from 'vitest'

import { useAdminTokenActions } from './useAdminTokenActions'

const mcpTokenMocks = vi.hoisted(() => ({
  listMcpTokens: vi.fn(),
  createMcpToken: vi.fn(),
  revokeMcpToken: vi.fn(),
  listAvailableMcpTools: vi.fn(),
  listAvailableMcpProjects: vi.fn(),
}))

vi.mock('../api/mcpTokens', async () => {
  const actual = await vi.importActual<typeof import('../api/mcpTokens')>('../api/mcpTokens')
  return {
    ...actual,
    listMcpTokens: mcpTokenMocks.listMcpTokens,
    createMcpToken: mcpTokenMocks.createMcpToken,
    revokeMcpToken: mcpTokenMocks.revokeMcpToken,
    listAvailableMcpTools: mcpTokenMocks.listAvailableMcpTools,
    listAvailableMcpProjects: mcpTokenMocks.listAvailableMcpProjects,
  }
})

beforeEach(() => {
  mcpTokenMocks.listMcpTokens.mockReset()
  mcpTokenMocks.createMcpToken.mockReset()
  mcpTokenMocks.revokeMcpToken.mockReset()
  mcpTokenMocks.listAvailableMcpTools.mockReset()
  mcpTokenMocks.listAvailableMcpProjects.mockReset()
})

describe('useAdminTokenActions', () => {
  it('does not open panel or call APIs when user is not admin', async () => {
    const { result } = renderHook(() =>
      useAdminTokenActions({
        isAdmin: false,
        setNotice: vi.fn(),
        describeError: () => 'error',
      }),
    )

    await act(async () => {
      await result.current.openAdminTokenPanel()
    })

    expect(result.current.adminTokenPanelOpen).toBe(false)
    expect(mcpTokenMocks.listMcpTokens).not.toHaveBeenCalled()
    expect(mcpTokenMocks.listAvailableMcpTools).not.toHaveBeenCalled()
    expect(mcpTokenMocks.listAvailableMcpProjects).not.toHaveBeenCalled()
  })

  it('opens panel and loads token inventory/options for admins', async () => {
    mcpTokenMocks.listMcpTokens.mockResolvedValue([
      {
        token_id: 'token-1',
        name: 'Read Token',
        scope: 'read',
        allowed_tools: ['user.get_profile'],
        allowed_project_ids: ['engram-vault'],
        token_secret_hint: 'abcd',
        expires_at: '2026-03-01T00:00:00Z',
        last_used_at: null,
        revoked_at: null,
        created_at: '2026-02-01T00:00:00Z',
        is_active: true,
      },
    ])
    mcpTokenMocks.listAvailableMcpTools.mockResolvedValue(['chat.send_message'])
    mcpTokenMocks.listAvailableMcpProjects.mockResolvedValue(['engram-vault'])

    const { result } = renderHook(() =>
      useAdminTokenActions({
        isAdmin: true,
        setNotice: vi.fn(),
        describeError: () => 'error',
      }),
    )

    await act(async () => {
      await result.current.openAdminTokenPanel()
    })

    expect(result.current.adminTokenPanelOpen).toBe(true)
    expect(result.current.adminLatestToken).toBeNull()
    expect(result.current.adminTokens).toHaveLength(1)
    expect(result.current.adminAvailableTools).toEqual(['chat.send_message'])
    expect(result.current.adminAvailableProjects).toEqual(['engram-vault'])
    expect(mcpTokenMocks.listMcpTokens).toHaveBeenCalledTimes(1)
    expect(mcpTokenMocks.listAvailableMcpTools).toHaveBeenCalledTimes(1)
    expect(mcpTokenMocks.listAvailableMcpProjects).toHaveBeenCalledTimes(1)
  })

  it('keeps the refresh handler stable across panel state updates', async () => {
    const describeError = vi.fn(() => 'error')
    mcpTokenMocks.listMcpTokens.mockResolvedValue([])
    mcpTokenMocks.listAvailableMcpTools.mockResolvedValue(['chat.send_message'])
    mcpTokenMocks.listAvailableMcpProjects.mockResolvedValue(['engram-vault'])

    const { result } = renderHook(() =>
      useAdminTokenActions({
        isAdmin: true,
        setNotice: vi.fn(),
        describeError,
      }),
    )

    const initialRefreshHandler = result.current.handleRefreshAdminTokenPanel

    await act(async () => {
      await result.current.openAdminTokenPanel()
    })

    expect(result.current.handleRefreshAdminTokenPanel).toBe(initialRefreshHandler)
  })

  it('creates token, emits notice, and refreshes token list', async () => {
    const setNotice = vi.fn()
    mcpTokenMocks.createMcpToken.mockResolvedValue({
      token_id: 'token-2',
      name: 'Write Token',
      scope: 'write',
      allowed_tools: ['chat.send_message'],
      allowed_project_ids: ['engram-vault'],
      token_secret_hint: 'wxyz',
      expires_at: '2026-05-01T00:00:00Z',
      last_used_at: null,
      revoked_at: null,
      created_at: '2026-02-20T00:00:00Z',
      is_active: true,
      token: 'secret-value',
    })
    mcpTokenMocks.listMcpTokens.mockResolvedValue([
      {
        token_id: 'token-2',
        name: 'Write Token',
        scope: 'write',
        allowed_tools: ['chat.send_message'],
        allowed_project_ids: ['engram-vault'],
        token_secret_hint: 'wxyz',
        expires_at: '2026-05-01T00:00:00Z',
        last_used_at: null,
        revoked_at: null,
        created_at: '2026-02-20T00:00:00Z',
        is_active: true,
      },
    ])

    const { result } = renderHook(() =>
      useAdminTokenActions({
        isAdmin: true,
        setNotice,
        describeError: () => 'error',
      }),
    )

    await act(async () => {
      await result.current.handleCreateAdminToken({
        name: 'Write Token',
        scope: 'write',
        allowed_tools: ['chat.send_message'],
        allowed_project_ids: ['engram-vault'],
        expires_in_days: 30,
      })
    })

    expect(mcpTokenMocks.createMcpToken).toHaveBeenCalledWith({
      name: 'Write Token',
      scope: 'write',
      allowed_tools: ['chat.send_message'],
      allowed_project_ids: ['engram-vault'],
      expires_in_days: 30,
    })
    expect(setNotice).toHaveBeenCalledWith('Created MCP token Write Token.')
    expect(mcpTokenMocks.listMcpTokens).toHaveBeenCalledTimes(1)
    expect(result.current.adminLatestToken?.token_id).toBe('token-2')
    expect(result.current.adminTokens).toHaveLength(1)
    expect(result.current.adminTokensCreating).toBe(false)
  })

  it('captures revoke failures in admin token error state', async () => {
    const setNotice = vi.fn()
    mcpTokenMocks.revokeMcpToken.mockRejectedValue(new Error('boom'))

    const { result } = renderHook(() =>
      useAdminTokenActions({
        isAdmin: true,
        setNotice,
        describeError: () => 'mapped revoke error',
      }),
    )

    await act(async () => {
      await result.current.handleRevokeAdminToken('token-3')
    })

    expect(mcpTokenMocks.revokeMcpToken).toHaveBeenCalledWith('token-3')
    expect(result.current.adminTokenError).toBe('mapped revoke error')
    expect(setNotice).not.toHaveBeenCalled()
    expect(mcpTokenMocks.listMcpTokens).not.toHaveBeenCalled()
  })
})
