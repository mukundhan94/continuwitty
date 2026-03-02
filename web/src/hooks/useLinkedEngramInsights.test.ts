import { act, renderHook, waitFor } from '@testing-library/react'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'

import type { EngramLinkSuggestion, EngramTracePath } from '../api/types'
import { useLinkedEngramInsights } from './useLinkedEngramInsights'

const apiMocks = vi.hoisted(() => ({
  listEngramLinks: vi.fn(),
  suggestEngramLinks: vi.fn(),
  createEngramLink: vi.fn(),
}))

vi.mock('../api/chat', async () => {
  const actual = await vi.importActual<typeof import('../api/chat')>('../api/chat')
  return {
    ...actual,
    listEngramLinks: apiMocks.listEngramLinks,
    suggestEngramLinks: apiMocks.suggestEngramLinks,
    createEngramLink: apiMocks.createEngramLink,
  }
})

beforeEach(() => {
  apiMocks.listEngramLinks.mockReset()
  apiMocks.suggestEngramLinks.mockReset()
  apiMocks.createEngramLink.mockReset()
})

afterEach(() => {
  vi.restoreAllMocks()
})

function suggestionFixture(): EngramLinkSuggestion {
  return {
    source_engram_id: 'engram-source',
    target_engram_id: 'engram-target',
    project_id: 'engram-vault',
    target_title: 'Target',
    target_abstract: 'Target abstract',
    target_created_at: '2026-03-01T08:00:00Z',
    relation_type: 'supports',
    weight: 0.77,
    temporal_weight: 0.62,
    confidence: 0.7,
    score: 0.83,
    origin: 'suggested',
    status: 'suggested',
    reasons: ['semantic overlap'],
    evidence_json: { mode: 'test' },
  }
}

describe('useLinkedEngramInsights', () => {
  it('loads links for source engrams and prioritizes used link ids', async () => {
    apiMocks.listEngramLinks.mockResolvedValue([
      {
        link_id: 'link-used',
        project_id: 'engram-vault',
        source_engram_id: 'engram-source',
        target_engram_id: 'engram-a',
        relation_type: 'supports',
        weight: 0.9,
        temporal_weight: 0.7,
        confidence: 0.8,
        origin: 'manual',
        status: 'active',
        evidence_json: {},
        created_by_user_id: 'user-1',
        created_at: '2026-03-01T08:00:00Z',
        updated_at: '2026-03-01T08:00:00Z',
        last_reinforced_at: null,
      },
      {
        link_id: 'link-unused',
        project_id: 'engram-vault',
        source_engram_id: 'engram-source',
        target_engram_id: 'engram-b',
        relation_type: 'related_to',
        weight: 0.4,
        temporal_weight: 0.4,
        confidence: 0.4,
        origin: 'manual',
        status: 'active',
        evidence_json: {},
        created_by_user_id: 'user-1',
        created_at: '2026-03-01T08:00:00Z',
        updated_at: '2026-03-01T08:00:00Z',
        last_reinforced_at: null,
      },
    ])
    apiMocks.suggestEngramLinks.mockResolvedValue([])

    const setNotice = vi.fn()
    const setChatError = vi.fn()
    const describeError = vi.fn(() => 'failed')
    const usedEngramIds = ['engram-source']
    const usedEngramLinkIds = ['link-used']
    const engramTracePaths: EngramTracePath[] = []
    const { result } = renderHook(() =>
      useLinkedEngramInsights({
        selectedSessionId: 'session-1',
        usedEngramIds,
        usedEngramLinkIds,
        engramTracePaths,
        setNotice,
        setChatError,
        describeError,
      }),
    )

    await waitFor(() => expect(result.current.loading).toBe(false))

    expect(apiMocks.listEngramLinks).toHaveBeenCalledWith(
      'engram-source',
      expect.objectContaining({ include_archived: false }),
    )
    expect(result.current.links.map((item) => item.link_id)).toEqual(['link-used'])
  })

  it('accepts and rejects suggestions through create-link mutations', async () => {
    const suggestion = suggestionFixture()
    const setNotice = vi.fn()
    const setChatError = vi.fn()
    const describeError = vi.fn(() => 'failed')
    const usedEngramIds = ['engram-source']
    const usedEngramLinkIds: string[] = []
    const engramTracePaths: EngramTracePath[] = []

    apiMocks.listEngramLinks.mockResolvedValue([])
    apiMocks.suggestEngramLinks.mockResolvedValue([suggestion])
    apiMocks.createEngramLink.mockResolvedValue({
      link_id: 'link-created',
    })

    const { result } = renderHook(() =>
      useLinkedEngramInsights({
        selectedSessionId: 'session-1',
        usedEngramIds,
        usedEngramLinkIds,
        engramTracePaths,
        setNotice,
        setChatError,
        describeError,
      }),
    )

    await waitFor(() => expect(result.current.suggestions.length).toBeGreaterThan(0))

    await act(async () => {
      await result.current.handleAcceptSuggestion(suggestion)
    })

    expect(apiMocks.createEngramLink).toHaveBeenCalledWith(
      'engram-source',
      expect.objectContaining({ target_engram_id: 'engram-target', status: 'active' }),
    )
    expect(setNotice).toHaveBeenCalledWith(expect.stringContaining('Accepted link suggestion'))

    await act(async () => {
      await result.current.handleRejectSuggestion(suggestion)
    })

    expect(apiMocks.createEngramLink).toHaveBeenCalledWith(
      'engram-source',
      expect.objectContaining({ target_engram_id: 'engram-target', status: 'rejected' }),
    )
    expect(setChatError).not.toHaveBeenCalled()
  })
})
