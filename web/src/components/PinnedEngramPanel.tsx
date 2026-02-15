import type { EngramSummary } from '../api/types'
import styled from 'styled-components'

import {
  EngramAbstract,
  EngramCard,
  EngramTitle,
  GlassPane,
  MutedText,
  PaneHeader,
  ScrollColumn,
  SectionDivider,
} from '../styles/primitives'

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
    <GlassPane as="aside" className="overflow-hidden">
      <PaneHeader>
        <h2 className="font-display text-base font-semibold tracking-[0.02em] text-ink">Pinned Engrams</h2>
        <button type="button" onClick={onRefresh} disabled={!selectedSessionId || loading}>
          Refresh
        </button>
      </PaneHeader>

      {!selectedSessionId ? <MutedText>Select a session to manage engrams.</MutedText> : null}

      <SectionDivider />
      <ScrollColumn>
        <SectionTitle>Session Pins</SectionTitle>
        {pinnedEngrams.length === 0 ? <MutedText>No pinned engrams yet.</MutedText> : null}

        {pinnedEngrams.map((engram) => (
          <EngramCard key={engram.engram_id}>
            <EngramTitle>{engram.title}</EngramTitle>
            <EngramAbstract>{engram.abstract}</EngramAbstract>
            <EngramActions>
              <button type="button" onClick={() => onCopyId(engram.engram_id)}>
                Copy ID
              </button>
              <button type="button" onClick={() => onUnpin(engram.engram_id)}>
                Unpin
              </button>
            </EngramActions>
          </EngramCard>
        ))}
      </ScrollColumn>

      <SectionDivider />
      <ScrollColumn>
        <SectionTitle>Search Project Engrams</SectionTitle>
        <input
          value={search}
          onChange={(event) => onSearchChange(event.target.value)}
          placeholder="Search title, abstract, or engram id"
        />

        {filteredAvailable.map((engram) => (
          <EngramCard key={engram.engram_id}>
            <EngramTitle>{engram.title}</EngramTitle>
            <EngramAbstract>{engram.abstract}</EngramAbstract>
            <EngramActions>
              <button type="button" onClick={() => onCopyId(engram.engram_id)}>
                Copy ID
              </button>
              <button
                type="button"
                onClick={() => onPin(engram.engram_id)}
                disabled={!selectedSessionId || pinnedIds.has(engram.engram_id)}
              >
                {pinnedIds.has(engram.engram_id) ? 'Pinned' : 'Pin to Session'}
              </button>
            </EngramActions>
          </EngramCard>
        ))}

        {filteredAvailable.length === 0 ? <MutedText>No matching engrams found.</MutedText> : null}
      </ScrollColumn>
    </GlassPane>
  )
}
