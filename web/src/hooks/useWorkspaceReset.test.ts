import { act, renderHook } from '@testing-library/react'
import { describe, expect, it, vi } from 'vitest'

import { useWorkspaceReset } from './useWorkspaceReset'

function buildConfig(overrides: Partial<Parameters<typeof useWorkspaceReset>[0]> = {}) {
  return {
    setUser: vi.fn(),
    setSessions: vi.fn(),
    setSelectedSessionId: vi.fn(),
    setDefaultProjectId: vi.fn(),
    setMessages: vi.fn(),
    setPinnedEngrams: vi.fn(),
    setPinnedDocuments: vi.fn(),
    setAvailableEngrams: vi.fn(),
    setSourceReferences: vi.fn(),
    setDocuments: vi.fn(),
    setDocumentsError: vi.fn(),
    setTimelineEvents: vi.fn(),
    setNotice: vi.fn(),
    setAuthError: vi.fn(),
    resetAdminTokenState: vi.fn(),
    ...overrides,
  }
}

describe('useWorkspaceReset', () => {
  it('clears workspace state and admin token state', () => {
    const config = buildConfig()
    const { result } = renderHook(() => useWorkspaceReset(config))

    act(() => {
      result.current()
    })

    expect(config.setUser).toHaveBeenCalledWith(null)
    expect(config.setSessions).toHaveBeenCalledWith([])
    expect(config.setSelectedSessionId).toHaveBeenCalledWith(null)
    expect(config.setDefaultProjectId).toHaveBeenCalledWith(null)
    expect(config.setMessages).toHaveBeenCalledWith([])
    expect(config.setPinnedEngrams).toHaveBeenCalledWith([])
    expect(config.setPinnedDocuments).toHaveBeenCalledWith([])
    expect(config.setAvailableEngrams).toHaveBeenCalledWith([])
    expect(config.setSourceReferences).toHaveBeenCalledWith([])
    expect(config.setDocuments).toHaveBeenCalledWith([])
    expect(config.setDocumentsError).toHaveBeenCalledWith(null)
    expect(config.setTimelineEvents).toHaveBeenCalledWith([])
    expect(config.setNotice).toHaveBeenCalledWith(null)
    expect(config.setAuthError).toHaveBeenCalledWith(null)
    expect(config.resetAdminTokenState).toHaveBeenCalledTimes(1)
  })
})
