import { useMemo } from 'react'
import styled from 'styled-components'

import type {
  ChatSourceReference,
  EngramLinkRecord,
  EngramLinkSuggestion,
  EngramSummary,
  EngramTracePath,
} from '../api/types'
import {
  ErrorText,
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

const SummaryGrid = styled.div`
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 0.36rem;
`

const SummaryChip = styled.div`
  border: 1px solid var(--color-line);
  border-radius: ${({ theme }) => theme.radius.md};
  background: var(--surface-raised);
  padding: 0.36rem 0.42rem;
  font-size: 0.75rem;
  color: var(--color-ink-muted);
`

const LinkCard = styled.article`
  border: 1px solid var(--color-line);
  border-radius: ${({ theme }) => theme.radius.md};
  padding: 0.45rem 0.5rem;
  background: var(--surface-raised);
  display: grid;
  gap: 0.24rem;
`

const LinkHeading = styled.p`
  font-size: 0.8rem;
  font-weight: 700;
  color: var(--color-ink);
`

const LinkMeta = styled.p`
  font-size: 0.74rem;
  color: var(--color-ink-muted);
`

const SuggestionActions = styled.div`
  display: flex;
  gap: 0.35rem;
  flex-wrap: wrap;
`

function suggestionKey(suggestion: EngramLinkSuggestion): string {
  return `${suggestion.source_engram_id}::${suggestion.target_engram_id}`
}

function formatRelativeAge(value: string | null | undefined): string {
  if (!value) {
    return 'unknown age'
  }
  const timestamp = Date.parse(value)
  if (!Number.isFinite(timestamp)) {
    return 'unknown age'
  }
  const deltaSeconds = Math.max(0, Math.floor((Date.now() - timestamp) / 1000))
  if (deltaSeconds < 60) {
    return `${deltaSeconds}s ago`
  }
  if (deltaSeconds < 60 * 60) {
    return `${Math.floor(deltaSeconds / 60)}m ago`
  }
  if (deltaSeconds < 60 * 60 * 24) {
    return `${Math.floor(deltaSeconds / (60 * 60))}h ago`
  }
  return `${Math.floor(deltaSeconds / (60 * 60 * 24))}d ago`
}

function buildEngramTitleLookup(
  availableEngrams: EngramSummary[],
): Map<string, EngramSummary> {
  const lookup = new Map<string, EngramSummary>()
  for (const item of availableEngrams) {
    lookup.set(item.engram_id, item)
  }
  return lookup
}

function resolveEngramTitle(
  engramLookup: Map<string, EngramSummary>,
  engramId: string,
  fallback?: string,
): string {
  return engramLookup.get(engramId)?.title ?? fallback ?? engramId
}

interface LinkedEngramPanelProps {
  selectedSessionId: string | null
  loading: boolean
  error: string | null
  sourceEngramIds: string[]
  links: EngramLinkRecord[]
  suggestions: EngramLinkSuggestion[]
  pendingSuggestionKeys: string[]
  availableEngrams: EngramSummary[]
  sourceReferences: ChatSourceReference[]
  tracePaths: EngramTracePath[]
  onRefresh: () => Promise<void>
  onAcceptSuggestion: (suggestion: EngramLinkSuggestion) => Promise<void>
  onRejectSuggestion: (suggestion: EngramLinkSuggestion) => Promise<void>
}

export function LinkedEngramPanel({
  selectedSessionId,
  loading,
  error,
  sourceEngramIds,
  links,
  suggestions,
  pendingSuggestionKeys,
  availableEngrams,
  sourceReferences,
  tracePaths,
  onRefresh,
  onAcceptSuggestion,
  onRejectSuggestion,
}: LinkedEngramPanelProps) {
  const engramLookup = useMemo(
    () => buildEngramTitleLookup(availableEngrams),
    [availableEngrams],
  )

  return (
    <GlassPane>
      <PaneHeader>
        <h2>Linked Memory</h2>
        <button disabled={loading || !selectedSessionId} onClick={() => void onRefresh()} type="button">
          Refresh
        </button>
      </PaneHeader>

      {!selectedSessionId ? (
        <MutedText>Select a session to inspect linked-memory provenance.</MutedText>
      ) : (
        <>
          <SummaryGrid>
            <SummaryChip>Seed engrams: {sourceEngramIds.length}</SummaryChip>
            <SummaryChip>Trace paths: {tracePaths.length}</SummaryChip>
            <SummaryChip>Links in panel: {links.length}</SummaryChip>
            <SummaryChip>Citations: {sourceReferences.length}</SummaryChip>
          </SummaryGrid>

          {error ? <ErrorText>{error}</ErrorText> : null}
          {loading ? <MutedText>Loading linked-memory details...</MutedText> : null}

          <SectionDivider>
            <SectionTitle>Linked Engram Edges</SectionTitle>
          </SectionDivider>
          <ScrollColumn>
            {links.length === 0 ? (
              <MutedText>No linked edges captured yet for this response.</MutedText>
            ) : (
              links.map((link) => {
                const sourceTitle = resolveEngramTitle(engramLookup, link.source_engram_id)
                const targetTitle = resolveEngramTitle(engramLookup, link.target_engram_id)
                return (
                  <LinkCard data-testid={`linked-edge-${link.link_id}`} key={link.link_id}>
                    <LinkHeading>
                      {sourceTitle} {'->'} {targetTitle}
                    </LinkHeading>
                    <LinkMeta>
                      Relation: {link.relation_type} | Weight: {link.weight.toFixed(2)} | Confidence:{' '}
                      {link.confidence.toFixed(2)}
                    </LinkMeta>
                    <LinkMeta>
                      Status: {link.status} | Origin: {link.origin} | Updated{' '}
                      {formatRelativeAge(link.last_reinforced_at ?? link.updated_at)}
                    </LinkMeta>
                  </LinkCard>
                )
              })
            )}
          </ScrollColumn>

          <SectionDivider>
            <SectionTitle>Suggestion Queue</SectionTitle>
          </SectionDivider>
          <ScrollColumn>
            {suggestions.length === 0 ? (
              <MutedText>No new link suggestions for active response engrams.</MutedText>
            ) : (
              suggestions.map((suggestion) => {
                const key = suggestionKey(suggestion)
                const pending = pendingSuggestionKeys.includes(key)
                const sourceTitle = resolveEngramTitle(engramLookup, suggestion.source_engram_id)
                return (
                  <LinkCard data-testid={`link-suggestion-${key}`} key={key}>
                    <LinkHeading>
                      {sourceTitle} {'->'}{' '}
                      {resolveEngramTitle(
                        engramLookup,
                        suggestion.target_engram_id,
                        suggestion.target_title,
                      )}
                    </LinkHeading>
                    <LinkMeta>
                      Suggested relation: {suggestion.relation_type} | Score:{' '}
                      {suggestion.score.toFixed(2)} | Age{' '}
                      {formatRelativeAge(suggestion.target_created_at)}
                    </LinkMeta>
                    {suggestion.reasons.length > 0 ? (
                      <LinkMeta>Reasons: {suggestion.reasons.join(', ')}</LinkMeta>
                    ) : null}
                    <SuggestionActions>
                      <button
                        disabled={loading || pending}
                        onClick={() => void onAcceptSuggestion(suggestion)}
                        type="button"
                      >
                        Accept
                      </button>
                      <button
                        disabled={loading || pending}
                        onClick={() => void onRejectSuggestion(suggestion)}
                        type="button"
                      >
                        Reject
                      </button>
                    </SuggestionActions>
                  </LinkCard>
                )
              })
            )}
          </ScrollColumn>
        </>
      )}
    </GlassPane>
  )
}
