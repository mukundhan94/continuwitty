import { apiJson } from './http'
import type {
  EngramLinkOrigin,
  EngramLinkRecord,
  EngramLinkRelationType,
  EngramLinkStatus,
  EngramLinkSuggestion,
} from './types'

export interface CreateEngramLinkPayload {
  target_engram_id: string
  relation_type?: EngramLinkRelationType
  weight?: number
  temporal_weight?: number
  confidence?: number
  origin?: EngramLinkOrigin
  status?: EngramLinkStatus
  evidence_json?: Record<string, unknown>
  last_reinforced_at?: string
}

export interface ListEngramLinksQuery {
  relation_type?: EngramLinkRelationType
  include_archived?: boolean
  limit?: number
  offset?: number
}

export interface SuggestEngramLinksPayload {
  limit?: number
  max_candidates?: number
  minimum_score?: number
  include_archived?: boolean
}

function appendOptionalBoolean(
  query: URLSearchParams,
  key: string,
  value: boolean | undefined,
): void {
  if (typeof value === 'boolean') {
    query.set(key, value ? 'true' : 'false')
  }
}

function appendOptionalNumber(
  query: URLSearchParams,
  key: string,
  value: number | undefined,
): void {
  if (typeof value === 'number' && Number.isFinite(value)) {
    query.set(key, String(value))
  }
}

export async function createEngramLink(
  sourceEngramId: string,
  payload: CreateEngramLinkPayload,
): Promise<EngramLinkRecord> {
  return apiJson<EngramLinkRecord>(`/api/v1/engrams/${sourceEngramId}/links`, {
    method: 'POST',
    body: JSON.stringify(payload),
  })
}

export async function listEngramLinks(
  sourceEngramId: string,
  query: ListEngramLinksQuery = {},
): Promise<EngramLinkRecord[]> {
  const searchParams = new URLSearchParams()
  if (query.relation_type) {
    searchParams.set('relation_type', query.relation_type)
  }
  appendOptionalBoolean(searchParams, 'include_archived', query.include_archived)
  appendOptionalNumber(searchParams, 'limit', query.limit)
  appendOptionalNumber(searchParams, 'offset', query.offset)
  const querySuffix = searchParams.toString()
  const path = querySuffix
    ? `/api/v1/engrams/${sourceEngramId}/links?${querySuffix}`
    : `/api/v1/engrams/${sourceEngramId}/links`
  return apiJson<EngramLinkRecord[]>(path)
}

export async function suggestEngramLinks(
  sourceEngramId: string,
  payload: SuggestEngramLinksPayload = {},
): Promise<EngramLinkSuggestion[]> {
  return apiJson<EngramLinkSuggestion[]>(`/api/v1/engrams/${sourceEngramId}/links/suggest`, {
    method: 'POST',
    body: JSON.stringify(payload),
  })
}
