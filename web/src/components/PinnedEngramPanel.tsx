import type { EngramSummary } from '../api/types'
import { useCallback, useEffect, useRef, useState, type FocusEvent, type MouseEvent } from 'react'
import type { Components } from 'react-markdown'
import ReactMarkdown from 'react-markdown'
import remarkBreaks from 'remark-breaks'
import remarkGfm from 'remark-gfm'
import styled, { css } from 'styled-components'

import {
  EngramCard,
  EngramTitle,
  GlassPane,
  MessageText,
  MutedText,
  PaneHeader,
  ScrollColumn,
  SectionDivider,
} from '../styles/primitives'
import { normalizeTooltipMarkdown } from '../utils/markdownTooltip'

const SectionTitle = styled.h3`
  font-family: var(--font-display);
  font-size: 0.98rem;
  font-weight: 700;
  color: var(--color-ink);
`

const EngramActions = styled.div`
  display: flex;
  justify-content: flex-end;
  gap: 0.35rem;
  flex-wrap: wrap;
`

const CardActionButton = styled.button`
  padding: 0.25rem 0.52rem;
  border-radius: 9px;
  font-size: 0.74rem;
  line-height: 1.15;

  &:hover {
    transform: none;
    filter: brightness(1.03);
  }
`

const SplitSections = styled.div`
  flex: 1;
  min-height: 0;
  display: grid;
  grid-template-rows: minmax(0, 1fr) auto minmax(0, 1fr);
  gap: 0.55rem;
`

const SectionPanel = styled.section`
  min-height: 0;
  display: grid;
  grid-template-rows: auto minmax(0, 1fr);
  gap: 0.45rem;
`

const SearchPanel = styled(SectionPanel)`
  grid-template-rows: auto auto minmax(0, 1fr);
`

const MidDottedDivider = styled.div`
  border-top: 1px dashed var(--color-line);
`

const SectionScroll = styled(ScrollColumn)`
  min-height: 0;
`

const EngramCardSelectable = styled(EngramCard)<{ $pinned: boolean }>`
  position: relative;

  ${({ $pinned }) =>
    $pinned
      ? css`
          background: var(--session-active-bg);
          border-color: var(--session-active-border);
          box-shadow:
            inset 0 0 0 1px var(--session-active-border),
            0 0 0 1px var(--session-active-shadow);
        `
      : null}
`

const FloatingTooltip = styled.aside`
  position: fixed;
  z-index: 60;
  max-width: calc(100vw - 1.2rem);
  background: var(--color-card);
  border: 1px solid var(--color-line);
  border-radius: ${({ theme }) => theme.radius.md};
  box-shadow: var(--shadow-modal);
  padding: 0.72rem 0.78rem;
  display: grid;
  gap: 0.52rem;
`

const TooltipTitle = styled.p`
  font-size: 0.78rem;
  font-weight: 700;
  letter-spacing: 0.04em;
  text-transform: uppercase;
  color: var(--color-ink-muted);
`

const TooltipBody = styled(MessageText)`
  font-family: var(--font-body);
  font-size: 0.74rem;
  line-height: 1.58;
  max-height: min(33.8rem, calc(100vh - 3rem));
  overflow: auto;
  padding-right: 0.42rem;
  text-align: left;
  overflow-wrap: anywhere;
  word-break: break-word;

  & > :first-child {
    margin-top: 0;
  }

  & > :last-child {
    margin-bottom: 0;
  }

  h1,
  h2,
  h3,
  h4 {
    font-family: var(--font-body);
    font-weight: 700;
    letter-spacing: 0;
    line-height: 1.38;
    margin: 0.5rem 0 0.3rem;
  }

  h1 {
    font-size: 0.88rem;
  }

  h2 {
    font-size: 0.82rem;
  }

  h3,
  h4 {
    font-size: 0.79rem;
  }

  p,
  ul,
  ol,
  pre,
  blockquote,
  table,
  hr {
    margin: 0.42rem 0;
  }

  ul,
  ol {
    padding-left: 0.95rem;
  }

  li + li {
    margin-top: 0.16rem;
  }

  table {
    width: 100%;
    border-collapse: collapse;
    border: 1px solid var(--markdown-table-border);
    border-radius: 8px;
    overflow: hidden;
    background: var(--surface-raised);
  }

  th,
  td {
    text-align: left;
    vertical-align: top;
    padding: 0.22rem 0.3rem;
    border-bottom: 1px solid var(--markdown-table-border);
  }

  th {
    background: var(--markdown-table-header-bg);
    font-weight: 700;
  }

  tr:last-child td {
    border-bottom: none;
  }

  code {
    font-size: 0.72rem;
  }
`

interface TooltipState {
  engram: EngramSummary
  left: number
  top: number
  width: number
  maxHeight: number
}

interface PinnedEngramPanelProps {
  selectedSessionId: string | null
  pinnedEngrams: EngramSummary[]
  availableEngrams: EngramSummary[]
  search: string
  loading: boolean
  onSearchChange: (value: string) => void
  onRefresh: () => Promise<void>
  onPin: (engramId: string) => Promise<void>
  onUnpin: (engramId: string) => Promise<void>
  onCopyId: (engramId: string) => Promise<void>
}

const markdownComponents: Components = {
  a: ({ node, ...props }) => {
    void node
    return <a {...props} rel="noreferrer" target="_blank" />
  },
}

export function PinnedEngramPanel({
  selectedSessionId,
  pinnedEngrams,
  availableEngrams,
  search,
  loading,
  onSearchChange,
  onRefresh,
  onPin,
  onUnpin,
  onCopyId,
}: PinnedEngramPanelProps) {
  const pinnedIds = new Set(pinnedEngrams.map((item) => item.engram_id))
  const normalized = search.trim().toLowerCase()
  const [tooltip, setTooltip] = useState<TooltipState | null>(null)
  const hideTimerRef = useRef<number | null>(null)

  const clearHideTimer = useCallback(() => {
    if (hideTimerRef.current !== null) {
      window.clearTimeout(hideTimerRef.current)
      hideTimerRef.current = null
    }
  }, [])

  const scheduleHideTooltip = useCallback(() => {
    clearHideTimer()
    hideTimerRef.current = window.setTimeout(() => {
      setTooltip(null)
    }, 140)
  }, [clearHideTimer])

  useEffect(
    () => () => {
      if (hideTimerRef.current !== null) {
        window.clearTimeout(hideTimerRef.current)
      }
    },
    [],
  )

  const showTooltip = useCallback(
    (engram: EngramSummary, target: HTMLElement) => {
      if (!engram.abstract.trim()) {
        setTooltip(null)
        return
      }

      clearHideTimer()
      const rect = target.getBoundingClientRect()
      const viewportWidth = window.innerWidth || 1280
      const viewportHeight = window.innerHeight || 800
      const edgePadding = 10
      const anchorOffset = 6
      const tooltipWidth = Math.max(420, Math.min(viewportWidth * 0.68, viewportWidth - edgePadding * 2))
      const tooltipHeight = Math.max(320, Math.min(viewportHeight * 0.82, viewportHeight - edgePadding * 2))

      let left = rect.left - tooltipWidth - anchorOffset
      if (left < edgePadding) {
        left = rect.right + anchorOffset
      }
      left = Math.max(edgePadding, Math.min(left, viewportWidth - tooltipWidth - edgePadding))

      const minTop = edgePadding
      const maxTop = viewportHeight - tooltipHeight - edgePadding
      const topAligned = rect.top + 2
      const bottomAligned = rect.bottom - tooltipHeight - 2

      let top = topAligned
      if (topAligned > maxTop) {
        const candidates = [maxTop]
        if (bottomAligned >= minTop && bottomAligned <= maxTop) {
          candidates.push(bottomAligned)
        }
        top = candidates.reduce((closest, candidate) =>
          Math.abs(candidate - rect.top) < Math.abs(closest - rect.top) ? candidate : closest,
        )
      }
      top = Math.max(minTop, Math.min(top, maxTop))

      setTooltip({ engram, left, top, width: tooltipWidth, maxHeight: tooltipHeight })
    },
    [clearHideTimer],
  )

  const handleCardMouseEnter = useCallback(
    (engram: EngramSummary) => (event: MouseEvent<HTMLElement>) => {
      showTooltip(engram, event.currentTarget)
    },
    [showTooltip],
  )

  const handleCardFocus = useCallback(
    (engram: EngramSummary) => (event: FocusEvent<HTMLElement>) => {
      showTooltip(engram, event.currentTarget)
    },
    [showTooltip],
  )

  const filteredAvailable = availableEngrams.filter((item) => {
    if (!normalized) {
      return true
    }
    return (
      item.title.toLowerCase().includes(normalized) ||
      item.abstract.toLowerCase().includes(normalized) ||
      item.engram_id.toLowerCase().includes(normalized)
    )
  })

  return (
    <GlassPane as="aside" className="overflow-hidden" data-testid="pinned-engrams-panel">
      <PaneHeader>
        <h2 className="font-display text-base font-semibold tracking-[0.02em] text-ink">Pinned Engrams</h2>
        <button type="button" onClick={onRefresh} disabled={!selectedSessionId || loading}>
          Refresh
        </button>
      </PaneHeader>

      {!selectedSessionId ? <MutedText>Select a session to manage engrams.</MutedText> : null}

      <SectionDivider />
      <SplitSections>
        <SectionPanel data-testid="pinned-section">
          <SectionTitle>Session Pins</SectionTitle>
          <SectionScroll>
            {pinnedEngrams.length === 0 ? <MutedText>No pinned engrams yet.</MutedText> : null}

            {pinnedEngrams.map((engram) => (
              <EngramCardSelectable
                key={engram.engram_id}
                $pinned={true}
                aria-selected="true"
                data-testid={`engram-card-${engram.engram_id}`}
                onMouseEnter={handleCardMouseEnter(engram)}
                onMouseLeave={scheduleHideTooltip}
                onFocus={handleCardFocus(engram)}
                onBlur={scheduleHideTooltip}
              >
                <EngramTitle>{engram.title}</EngramTitle>
                <EngramActions>
                  <CardActionButton type="button" onClick={() => onCopyId(engram.engram_id)}>
                    Copy ID
                  </CardActionButton>
                  <CardActionButton type="button" onClick={() => onUnpin(engram.engram_id)}>
                    Unpin
                  </CardActionButton>
                </EngramActions>
              </EngramCardSelectable>
            ))}
          </SectionScroll>
        </SectionPanel>

        <MidDottedDivider data-testid="engram-mid-divider" />

        <SearchPanel data-testid="unpinned-section">
          <SectionTitle>Search Project Engrams</SectionTitle>
          <input
            value={search}
            onChange={(event) => onSearchChange(event.target.value)}
            placeholder="Search title, abstract, or engram id"
          />

          <SectionScroll>
            {filteredAvailable.map((engram) => {
              const isPinned = pinnedIds.has(engram.engram_id)
              return (
                <EngramCardSelectable
                  key={engram.engram_id}
                  $pinned={isPinned}
                  aria-selected={isPinned ? 'true' : 'false'}
                  data-testid={`engram-card-${engram.engram_id}`}
                  onMouseEnter={handleCardMouseEnter(engram)}
                  onMouseLeave={scheduleHideTooltip}
                  onFocus={handleCardFocus(engram)}
                  onBlur={scheduleHideTooltip}
                >
                  <EngramTitle>{engram.title}</EngramTitle>
                  <EngramActions>
                    <CardActionButton type="button" onClick={() => onCopyId(engram.engram_id)}>
                      Copy ID
                    </CardActionButton>
                    <CardActionButton
                      type="button"
                      onClick={() => onPin(engram.engram_id)}
                      disabled={!selectedSessionId || isPinned}
                    >
                      {isPinned ? 'Pinned' : 'Pin to Session'}
                    </CardActionButton>
                  </EngramActions>
                </EngramCardSelectable>
              )
            })}

            {filteredAvailable.length === 0 ? <MutedText>No matching engrams found.</MutedText> : null}
          </SectionScroll>
        </SearchPanel>
      </SplitSections>

      {tooltip ? (
        <FloatingTooltip
          data-testid="engram-tooltip"
          onMouseEnter={clearHideTimer}
          onMouseLeave={scheduleHideTooltip}
          style={{ left: `${tooltip.left}px`, top: `${tooltip.top}px`, width: `${tooltip.width}px` }}
        >
          <TooltipTitle>Abstract Preview</TooltipTitle>
          <TooltipBody style={{ maxHeight: `${tooltip.maxHeight}px` }}>
            <ReactMarkdown components={markdownComponents} remarkPlugins={[remarkGfm, remarkBreaks]}>
              {normalizeTooltipMarkdown(tooltip.engram.abstract)}
            </ReactMarkdown>
          </TooltipBody>
        </FloatingTooltip>
      ) : null}
    </GlassPane>
  )
}
