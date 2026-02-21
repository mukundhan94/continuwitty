import { render, screen, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { ThemeProvider } from 'styled-components'
import { describe, expect, it, vi } from 'vitest'

import type { EngramSummary } from '../api/types'
import { lightTheme } from '../styles/theme'
import { PinnedEngramPanel } from './PinnedEngramPanel'

function buildEngram(overrides: Partial<EngramSummary>): EngramSummary {
  return {
    engram_id: 'engram-1',
    project_id: 'engram-vault',
    thread_id: null,
    title: 'Pinned synthesis',
    abstract: '## Summary\nSee [source](https://example.com) for details.',
    created_at: '2026-02-15T00:00:00Z',
    tags: ['incident'],
    keywords: ['synthesis'],
    owner_user_id: 'user-1',
    visibility_scope: 'project',
    ...overrides,
  }
}

function renderPanel() {
  const pinned = buildEngram({ engram_id: 'engram-pinned', title: 'Pinned Engram' })
  const unpinned = buildEngram({
    engram_id: 'engram-available',
    title: 'Available Engram',
    abstract: '### Context\nUse **memory** carry-forward.',
  })

  render(
    <ThemeProvider theme={lightTheme}>
      <PinnedEngramPanel
        selectedSessionId="session-1"
        pinnedEngrams={[pinned]}
        availableEngrams={[pinned, unpinned]}
        search=""
        loading={false}
        onSearchChange={vi.fn()}
        onRefresh={vi.fn(async () => {})}
        onPin={vi.fn(async () => {})}
        onUnpin={vi.fn(async () => {})}
        onCopyId={vi.fn(async () => {})}
      />
    </ThemeProvider>,
  )
}

describe('PinnedEngramPanel', () => {
  it('keeps a fixed split layout with a middle dotted divider', () => {
    renderPanel()
    expect(screen.getByTestId('pinned-section')).toBeInTheDocument()
    expect(screen.getByTestId('engram-mid-divider')).toBeInTheDocument()
    expect(screen.getByTestId('unpinned-section')).toBeInTheDocument()
  })

  it('marks pinned cards with selected state', () => {
    renderPanel()
    const pinnedCards = screen.getAllByTestId('engram-card-engram-pinned')
    expect(pinnedCards.length).toBe(2)
    for (const card of pinnedCards) {
      expect(card).toHaveAttribute('aria-selected', 'true')
    }
    expect(screen.getByTestId('engram-card-engram-available')).toHaveAttribute('aria-selected', 'false')
  })

  it('renders abstract markdown inside hover tooltip', async () => {
    const user = userEvent.setup()
    renderPanel()
    const pinnedCard = screen.getAllByTestId('engram-card-engram-pinned')[0]

    expect(screen.queryByTestId('engram-tooltip')).not.toBeInTheDocument()
    await user.hover(pinnedCard)

    const tooltip = await screen.findByTestId('engram-tooltip')
    const markdownLink = within(tooltip).getByRole('link', { name: 'source' })
    expect(markdownLink).toHaveAttribute('href', 'https://example.com')
    expect(markdownLink).toHaveAttribute('target', '_blank')
    expect(markdownLink).toHaveAttribute('rel', 'noreferrer')
  })

  it('invokes refresh and pin actions from toolbar and search list', async () => {
    const user = userEvent.setup()
    const onRefresh = vi.fn(async () => {})
    const onPin = vi.fn(async () => {})
    const available = buildEngram({
      engram_id: 'engram-available',
      title: 'Available Engram',
      abstract: '### Context\\nUse **memory** carry-forward.',
    })

    render(
      <ThemeProvider theme={lightTheme}>
        <PinnedEngramPanel
          selectedSessionId="session-1"
          pinnedEngrams={[]}
          availableEngrams={[available]}
          search=""
          loading={false}
          onSearchChange={vi.fn()}
          onRefresh={onRefresh}
          onPin={onPin}
          onUnpin={vi.fn(async () => {})}
          onCopyId={vi.fn(async () => {})}
        />
      </ThemeProvider>,
    )

    await user.click(screen.getByRole('button', { name: 'Refresh' }))
    expect(onRefresh).toHaveBeenCalledTimes(1)

    await user.click(screen.getByRole('button', { name: 'Pin to Session' }))
    expect(onPin).toHaveBeenCalledWith('engram-available')
  })
})
