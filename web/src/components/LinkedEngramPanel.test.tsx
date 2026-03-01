import { fireEvent, render, screen } from '@testing-library/react'
import type { ComponentProps } from 'react'
import { ThemeProvider } from 'styled-components'
import { describe, expect, it, vi } from 'vitest'

import type {
  ChatSourceReference,
  EngramLinkRecord,
  EngramLinkSuggestion,
  EngramSummary,
  EngramTracePath,
} from '../api/types'
import { lightTheme } from '../styles/theme'
import { LinkedEngramPanel } from './LinkedEngramPanel'

function buildEngram(overrides: Partial<EngramSummary>): EngramSummary {
  return {
    engram_id: 'engram-source',
    project_id: 'engram-vault',
    thread_id: null,
    title: 'Source Engram',
    abstract: 'Source abstract',
    created_at: '2026-03-01T08:00:00Z',
    tags: [],
    keywords: [],
    owner_user_id: 'user-1',
    visibility_scope: 'private',
    ...overrides,
  }
}

function renderPanel(overrides: Partial<ComponentProps<typeof LinkedEngramPanel>> = {}) {
  const links: EngramLinkRecord[] = [
    {
      link_id: 'link-1',
      project_id: 'engram-vault',
      source_engram_id: 'engram-source',
      target_engram_id: 'engram-target',
      relation_type: 'supports',
      weight: 0.82,
      temporal_weight: 0.7,
      confidence: 0.76,
      origin: 'suggested',
      status: 'active',
      evidence_json: {},
      created_by_user_id: 'user-1',
      last_reinforced_at: '2026-03-01T09:00:00Z',
      created_at: '2026-03-01T08:00:00Z',
      updated_at: '2026-03-01T09:00:00Z',
    },
  ]
  const suggestions: EngramLinkSuggestion[] = [
    {
      source_engram_id: 'engram-source',
      target_engram_id: 'engram-suggested',
      project_id: 'engram-vault',
      target_title: 'Suggested Engram',
      target_abstract: 'Suggestion abstract',
      target_created_at: '2026-02-28T11:00:00Z',
      relation_type: 'related_to',
      weight: 0.67,
      temporal_weight: 0.52,
      confidence: 0.63,
      score: 0.71,
      origin: 'suggested',
      status: 'suggested',
      reasons: ['semantic overlap'],
      evidence_json: {},
    },
  ]
  const tracePaths: EngramTracePath[] = [
    {
      root_engram_id: 'engram-source',
      target_engram_id: 'engram-target',
      depth: 1,
      link_ids: ['link-1'],
      engram_ids: ['engram-source', 'engram-target'],
      score: 0.82,
    },
  ]
  const sourceReferences: ChatSourceReference[] = [
    {
      engram_id: 'engram-source',
      engram_title: 'Source Engram',
      url: 'https://example.com',
      title: 'Example',
      snippet: 'snippet',
      captured_at: '2026-03-01T08:30:00Z',
    },
  ]
  const props: ComponentProps<typeof LinkedEngramPanel> = {
    selectedSessionId: 'session-1',
    loading: false,
    error: null,
    sourceEngramIds: ['engram-source'],
    links,
    suggestions,
    pendingSuggestionKeys: [],
    availableEngrams: [
      buildEngram({ engram_id: 'engram-source', title: 'Source Engram' }),
      buildEngram({ engram_id: 'engram-target', title: 'Target Engram' }),
      buildEngram({ engram_id: 'engram-suggested', title: 'Suggested Engram' }),
    ],
    sourceReferences,
    tracePaths,
    onRefresh: vi.fn(async () => {}),
    onAcceptSuggestion: vi.fn(async () => {}),
    onRejectSuggestion: vi.fn(async () => {}),
    ...overrides,
  }

  render(
    <ThemeProvider theme={lightTheme}>
      <LinkedEngramPanel {...props} />
    </ThemeProvider>,
  )
  return props
}

describe('LinkedEngramPanel', () => {
  it('renders linked edge metadata and explainability counts', () => {
    renderPanel()

    expect(screen.getByText('Linked Memory')).toBeInTheDocument()
    expect(screen.getByText('Seed engrams: 1')).toBeInTheDocument()
    expect(screen.getByText('Trace paths: 1')).toBeInTheDocument()
    expect(screen.getByTestId('linked-edge-link-1')).toBeInTheDocument()
    expect(screen.getByText(/Relation: supports/)).toBeInTheDocument()
  })

  it('invokes suggestion accept and reject callbacks', () => {
    const onAcceptSuggestion = vi.fn(async () => {})
    const onRejectSuggestion = vi.fn(async () => {})
    const props = renderPanel({ onAcceptSuggestion, onRejectSuggestion })

    fireEvent.click(screen.getByRole('button', { name: 'Accept' }))
    expect(onAcceptSuggestion).toHaveBeenCalledWith(props.suggestions[0])

    fireEvent.click(screen.getByRole('button', { name: 'Reject' }))
    expect(onRejectSuggestion).toHaveBeenCalledWith(props.suggestions[0])
  })
})
