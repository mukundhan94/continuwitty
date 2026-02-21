import { renderHook } from '@testing-library/react'
import { describe, expect, it, vi } from 'vitest'

import { useProjectScopePersistence } from './useProjectScopePersistence'

describe('useProjectScopePersistence', () => {
  it('persists normalized project id in local storage', () => {
    const setItemSpy = vi.spyOn(Storage.prototype, 'setItem')

    renderHook(() =>
      useProjectScopePersistence({
        projectId: '  engram-vault  ',
        normalizeProjectId: (value) => value.trim(),
        storageKey: 'engram.lastProjectId',
      }),
    )

    expect(setItemSpy).toHaveBeenCalledWith('engram.lastProjectId', 'engram-vault')
  })

  it('updates persisted project id when project scope changes', () => {
    const setItemSpy = vi.spyOn(Storage.prototype, 'setItem')
    const normalizeProjectId = (value: string) => value.trim()

    const { rerender } = renderHook(
      (projectId: string) =>
        useProjectScopePersistence({
          projectId,
          normalizeProjectId,
          storageKey: 'engram.lastProjectId',
        }),
      { initialProps: 'first-project' },
    )

    rerender(' second-project ')

    expect(setItemSpy).toHaveBeenLastCalledWith('engram.lastProjectId', 'second-project')
  })
})
