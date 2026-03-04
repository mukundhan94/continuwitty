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

interface LinkedInsightsStateSetters {
  setLinks: (value: EngramLinkRecord[]) => void
  setSuggestions: (value: EngramLinkSuggestion[]) => void
  setError: (value: string | null) => void
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

interface RecencyUnixInput {
  value: string | null | undefined
}

function recencyUnix(input: RecencyUnixInput): number {
  if (!input.value) {
    return 0
  }
  const parsed = Date.parse(input.value)
  if (!Number.isFinite(parsed)) {
    return 0
  }
  return parsed
}

function sortLinksByFreshnessAndWeight(records: EngramLinkRecord[]): EngramLinkRecord[] {
  return [...records].sort((left, right) => {
    const leftRecency = Math.max(
      recencyUnix({ value: left.last_reinforced_at }),
      recencyUnix({ value: left.updated_at }),
      recencyUnix({ value: left.created_at }),
    )
    const rightRecency = Math.max(
      recencyUnix({ value: right.last_reinforced_at }),
      recencyUnix({ value: right.updated_at }),
      recencyUnix({ value: right.created_at }),
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
    return recencyUnix({ value: right.target_created_at }) - recencyUnix({ value: left.target_created_at })
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

interface RemovePendingKeyInput {
  current: string[]
  key: string
}

function removePendingKey(input: RemovePendingKeyInput): string[] {
  return input.current.filter((item) => item !== input.key)
}

interface IdsFromSignatureInput {
  signature: string
}

function idsFromSignature(input: IdsFromSignatureInput): string[] {
  return input.signature ? input.signature.split('|') : []
}

function clearLinkedInsights(state: LinkedInsightsStateSetters): void {
  state.setLinks([])
  state.setSuggestions([])
  state.setError(null)
}

async function loadMergedInsightsForSources(input: {
  sourceIds: string[]
  usedLinkIds: string[]
}): Promise<SourceLoadResult> {
  const loaded = await Promise.all(input.sourceIds.map(loadInsightsForSource))
  return {
    links: mergeAndSelectLinks(
      loaded.flatMap((result) => result.links),
      input.usedLinkIds,
    ),
    suggestions: mergeAndSelectSuggestions(
      loaded.flatMap((result) => result.suggestions),
    ),
  }
}

interface SuggestionNoticeInput {
  status: EngramLinkStatus
  targetTitle: string
}

function noticeForSuggestionStatus(input: SuggestionNoticeInput): string {
  return input.status === 'active'
    ? `Accepted link suggestion to ${input.targetTitle}`
    : `Rejected link suggestion to ${input.targetTitle}`
}

function buildSuggestionCreatePayload(
  suggestion: EngramLinkSuggestion,
  status: EngramLinkStatus,
) {
  return {
    target_engram_id: suggestion.target_engram_id,
    relation_type: suggestion.relation_type,
    weight: suggestion.weight,
    temporal_weight: suggestion.temporal_weight,
    confidence: suggestion.confidence,
    origin: 'suggested' as const,
    status,
    evidence_json: suggestion.evidence_json,
  }
}

export function useLinkedEngramInsights(config: LinkedEngramInsightsConfig) {
  const {
    selectedSessionId,
    usedEngramIds,
    usedEngramLinkIds,
    engramTracePaths,
    describeError,
  } = config

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

  const {
    links,
    suggestions,
    loading,
    error,
    refreshLinkInsights,
  } = useLinkedEngramLoader({
    selectedSessionId,
    sourceEngramIdsSignature,
    usedEngramLinkIdsSignature,
    describeError,
  })
  const {
    pendingSuggestionKeys,
    handleAcceptSuggestion,
    handleRejectSuggestion,
  } = useLinkedEngramSuggestionActions({
    describeError,
    refreshLinkInsights,
    setChatError: config.setChatError,
    setNotice: config.setNotice,
  })

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

interface LinkedEngramLoaderConfig {
  selectedSessionId: string | null
  sourceEngramIdsSignature: string
  usedEngramLinkIdsSignature: string
  describeError: (error: unknown) => string
}

function useLinkedEngramLoader(config: LinkedEngramLoaderConfig) {
  const [links, setLinks] = useState<EngramLinkRecord[]>([])
  const [suggestions, setSuggestions] = useState<EngramLinkSuggestion[]>([])
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState<string | null>(null)

  const refreshLinkInsights = useCallback(async () => {
    const sourceIds = idsFromSignature({ signature: config.sourceEngramIdsSignature })
    if (!config.selectedSessionId || sourceIds.length === 0) {
      clearLinkedInsights({
        setLinks,
        setSuggestions,
        setError,
      })
      return
    }
    setLoading(true)
    try {
      const merged = await loadMergedInsightsForSources({
        sourceIds,
        usedLinkIds: idsFromSignature({ signature: config.usedEngramLinkIdsSignature }),
      })
      setLinks(merged.links)
      setSuggestions(merged.suggestions)
      setError(null)
    } catch (nextError) {
      setError(config.describeError(nextError))
    } finally {
      setLoading(false)
    }
  }, [config.describeError, config.selectedSessionId, config.sourceEngramIdsSignature, config.usedEngramLinkIdsSignature])

  useEffect(() => {
    void refreshLinkInsights()
  }, [refreshLinkInsights])

  return {
    links,
    suggestions,
    loading,
    error,
    refreshLinkInsights,
  }
}

interface LinkedEngramSuggestionActionsConfig {
  describeError: (error: unknown) => string
  refreshLinkInsights: () => Promise<void>
  setChatError: (value: string | null) => void
  setNotice: (value: string | null) => void
}

function useLinkedEngramSuggestionActions(config: LinkedEngramSuggestionActionsConfig) {
  const [pendingSuggestionKeys, setPendingSuggestionKeys] = useState<string[]>([])
  const { describeError, refreshLinkInsights, setChatError, setNotice } = config

  const runSuggestionMutation = useCallback(
    async (suggestion: EngramLinkSuggestion, status: EngramLinkStatus) => {
      const key = suggestionKey(suggestion)
      setPendingSuggestionKeys((current) => [...current, key])
      try {
        await createEngramLink(
          suggestion.source_engram_id,
          buildSuggestionCreatePayload(suggestion, status),
        )
        setNotice(noticeForSuggestionStatus({ status, targetTitle: suggestion.target_title }))
        await refreshLinkInsights()
      } catch (nextError) {
        setChatError(describeError(nextError))
      } finally {
        setPendingSuggestionKeys((current) => removePendingKey({ current, key }))
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
    pendingSuggestionKeys,
    handleAcceptSuggestion,
    handleRejectSuggestion,
  }
}
