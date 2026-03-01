import { useCallback, useEffect, useMemo, useState } from 'react'

import { createEngramLink, listEngramLinks, suggestEngramLinks } from '../api/chat'
import type {
  EngramLinkRecord,
  EngramLinkStatus,
  EngramLinkSuggestion,
  EngramTracePath,
} from '../api/types'

const DEFAULT_LINK_LIST_LIMIT = 120
const DEFAULT_SUGGEST_LIMIT = 4
const DEFAULT_MAX_CANDIDATES = 20
const DEFAULT_MINIMUM_SCORE = 0.25
const MAX_SOURCE_ENGRAMS = 4
const MAX_PANEL_LINKS = 12
const MAX_PANEL_SUGGESTIONS = 10

interface LinkedEngramInsightsConfig {
  selectedSessionId: string | null
  usedEngramIds: string[]
  usedEngramLinkIds: string[]
  engramTracePaths: EngramTracePath[]
  setNotice: (value: string | null) => void
  setChatError: (value: string | null) => void
  describeError: (error: unknown) => string
}

interface SourceLoadResult {
  links: EngramLinkRecord[]
  suggestions: EngramLinkSuggestion[]
}

function uniqueOrdered(values: string[]): string[] {
  const seen = new Set<string>()
  const ordered: string[] = []
  for (const value of values) {
    const normalized = value.trim()
    if (!normalized || seen.has(normalized)) {
      continue
    }
    seen.add(normalized)
    ordered.push(normalized)
  }
  return ordered
}

function joinedSignature(values: string[]): string {
  return uniqueOrdered(values).join('|')
}

function buildSourceEngramIds(
  engramTracePaths: EngramTracePath[],
  usedEngramIds: string[],
): string[] {
  const rootsFromTrace = engramTracePaths.map((path) => path.root_engram_id)
  return uniqueOrdered([...rootsFromTrace, ...usedEngramIds]).slice(0, MAX_SOURCE_ENGRAMS)
}

function recencyUnix(value: string | null | undefined): number {
  if (!value) {
    return 0
  }
  const parsed = Date.parse(value)
  if (!Number.isFinite(parsed)) {
    return 0
  }
  return parsed
}

function sortLinksByFreshnessAndWeight(records: EngramLinkRecord[]): EngramLinkRecord[] {
  return [...records].sort((left, right) => {
    const leftRecency = Math.max(
      recencyUnix(left.last_reinforced_at),
      recencyUnix(left.updated_at),
      recencyUnix(left.created_at),
    )
    const rightRecency = Math.max(
      recencyUnix(right.last_reinforced_at),
      recencyUnix(right.updated_at),
      recencyUnix(right.created_at),
    )
    if (leftRecency !== rightRecency) {
      return rightRecency - leftRecency
    }
    if (left.weight !== right.weight) {
      return right.weight - left.weight
    }
    return right.confidence - left.confidence
  })
}

function mergeAndSelectLinks(
  records: EngramLinkRecord[],
  usedEngramLinkIds: string[],
): EngramLinkRecord[] {
  const deduped = new Map<string, EngramLinkRecord>()
  for (const link of records) {
    deduped.set(link.link_id, link)
  }
  const all = sortLinksByFreshnessAndWeight(Array.from(deduped.values()))
  if (all.length === 0) {
    return []
  }
  const usedLinkSet = new Set(usedEngramLinkIds)
  if (usedLinkSet.size === 0) {
    return all.slice(0, MAX_PANEL_LINKS)
  }
  const usedOnly = all.filter((link) => usedLinkSet.has(link.link_id))
  if (usedOnly.length > 0) {
    return usedOnly.slice(0, MAX_PANEL_LINKS)
  }
  return all.slice(0, MAX_PANEL_LINKS)
}

function suggestionKey(suggestion: EngramLinkSuggestion): string {
  return `${suggestion.source_engram_id}::${suggestion.target_engram_id}`
}

function mergeAndSelectSuggestions(
  suggestions: EngramLinkSuggestion[],
): EngramLinkSuggestion[] {
  const deduped = new Map<string, EngramLinkSuggestion>()
  for (const suggestion of suggestions) {
    deduped.set(suggestionKey(suggestion), suggestion)
  }
  const ordered = [...deduped.values()].sort((left, right) => {
    if (left.score !== right.score) {
      return right.score - left.score
    }
    return recencyUnix(right.target_created_at) - recencyUnix(left.target_created_at)
  })
  return ordered.slice(0, MAX_PANEL_SUGGESTIONS)
}

async function loadInsightsForSource(sourceEngramId: string): Promise<SourceLoadResult> {
  const [links, suggestions] = await Promise.all([
    listEngramLinks(sourceEngramId, {
      include_archived: false,
      limit: DEFAULT_LINK_LIST_LIMIT,
      offset: 0,
    }),
    suggestEngramLinks(sourceEngramId, {
      limit: DEFAULT_SUGGEST_LIMIT,
      max_candidates: DEFAULT_MAX_CANDIDATES,
      minimum_score: DEFAULT_MINIMUM_SCORE,
      include_archived: false,
    }),
  ])
  return { links, suggestions }
}

function removePendingKey(current: string[], key: string): string[] {
  return current.filter((item) => item !== key)
}

export function useLinkedEngramInsights(config: LinkedEngramInsightsConfig) {
  const {
    selectedSessionId,
    usedEngramIds,
    usedEngramLinkIds,
    engramTracePaths,
    setNotice,
    setChatError,
    describeError,
  } = config

  const [links, setLinks] = useState<EngramLinkRecord[]>([])
  const [suggestions, setSuggestions] = useState<EngramLinkSuggestion[]>([])
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const [pendingSuggestionKeys, setPendingSuggestionKeys] = useState<string[]>([])

  const sourceEngramIdsSignature = useMemo(
    () => joinedSignature(buildSourceEngramIds(engramTracePaths, usedEngramIds)),
    [engramTracePaths, usedEngramIds],
  )
  const sourceEngramIds = useMemo(
    () =>
      sourceEngramIdsSignature
        ? sourceEngramIdsSignature.split('|')
        : [],
    [sourceEngramIdsSignature],
  )
  const usedEngramLinkIdsSignature = useMemo(
    () => joinedSignature(usedEngramLinkIds),
    [usedEngramLinkIds],
  )

  const refreshLinkInsights = useCallback(async () => {
    const sourceIds = sourceEngramIdsSignature
      ? sourceEngramIdsSignature.split('|')
      : []
    if (!selectedSessionId || sourceIds.length === 0) {
      setLinks([])
      setSuggestions([])
      setError(null)
      return
    }
    setLoading(true)
    try {
      const loaded = await Promise.all(sourceIds.map(loadInsightsForSource))
      const mergedLinks = mergeAndSelectLinks(
        loaded.flatMap((result) => result.links),
        usedEngramLinkIdsSignature ? usedEngramLinkIdsSignature.split('|') : [],
      )
      const mergedSuggestions = mergeAndSelectSuggestions(
        loaded.flatMap((result) => result.suggestions),
      )
      setLinks(mergedLinks)
      setSuggestions(mergedSuggestions)
      setError(null)
    } catch (nextError) {
      setError(describeError(nextError))
    } finally {
      setLoading(false)
    }
  }, [
    describeError,
    selectedSessionId,
    sourceEngramIdsSignature,
    usedEngramLinkIdsSignature,
  ])

  useEffect(() => {
    void refreshLinkInsights()
  }, [refreshLinkInsights])

  const runSuggestionMutation = useCallback(
    async (suggestion: EngramLinkSuggestion, status: EngramLinkStatus) => {
      const key = suggestionKey(suggestion)
      setPendingSuggestionKeys((current) => [...current, key])
      try {
        await createEngramLink(suggestion.source_engram_id, {
          target_engram_id: suggestion.target_engram_id,
          relation_type: suggestion.relation_type,
          weight: suggestion.weight,
          temporal_weight: suggestion.temporal_weight,
          confidence: suggestion.confidence,
          origin: 'suggested',
          status,
          evidence_json: suggestion.evidence_json,
        })
        setNotice(
          status === 'active'
            ? `Accepted link suggestion to ${suggestion.target_title}`
            : `Rejected link suggestion to ${suggestion.target_title}`,
        )
        await refreshLinkInsights()
      } catch (nextError) {
        setChatError(describeError(nextError))
      } finally {
        setPendingSuggestionKeys((current) => removePendingKey(current, key))
      }
    },
    [describeError, refreshLinkInsights, setChatError, setNotice],
  )

  const handleAcceptSuggestion = useCallback(
    async (suggestion: EngramLinkSuggestion) => {
      await runSuggestionMutation(suggestion, 'active')
    },
    [runSuggestionMutation],
  )

  const handleRejectSuggestion = useCallback(
    async (suggestion: EngramLinkSuggestion) => {
      await runSuggestionMutation(suggestion, 'rejected')
    },
    [runSuggestionMutation],
  )

  return {
    sourceEngramIds,
    links,
    suggestions,
    loading,
    error,
    pendingSuggestionKeys,
    refreshLinkInsights,
    handleAcceptSuggestion,
    handleRejectSuggestion,
  }
}
