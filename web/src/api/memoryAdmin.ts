import { apiJson } from './http'
import type {
  AdminChatSessionRecord,
  AdminEngramDeleteResponse,
  AdminEngramRecord,
  AdminEngramRestoreResponse,
  AdminSessionDeleteResponse,
  AdminSessionRestoreResponse,
  EngramCollectionRecord,
} from './types'

export interface ListAdminSessionsParams {
  project_id?: string
  owner_user_id?: string
  include_deleted?: boolean
  limit?: number
  offset?: number
}

export interface ListAdminEngramsParams {
  project_id?: string
  session_id?: string
  q?: string
  include_deleted?: boolean
  limit?: number
  offset?: number
}

export interface ListCollectionsParams {
  project_id?: string
  include_deleted?: boolean
  limit?: number
  offset?: number
}

export interface AdminDeletePayload {
  reason?: string
}

export interface AdminDeleteSessionPayload extends AdminDeletePayload {
  delete_linked_engrams?: boolean
}

export interface UpdateAdminEngramPayload {
  title?: string
  abstract?: string
  detailed_summary_markdown?: string
  tags?: string[]
  keywords?: string[]
  visibility_scope?: 'private' | 'project'
  expected_updated_at?: string
  sources?: Array<{
    captured_at: string
    url: string
    title: string
    snippet: string
    content_text?: string | null
    content_hash?: string | null
  }>
}

export interface MoveAdminEngramPayload {
  target_project_id: string
  expected_updated_at?: string
  reason?: string
}

export interface CreateCollectionPayload {
  project_id: string
  name: string
  description?: string
}

export interface UpdateCollectionPayload {
  name?: string
  description?: string
  expected_updated_at?: string
}

export interface UpdateCollectionItemsPayload {
  engram_ids: string[]
}

function buildQuery(params: Record<string, string | number | boolean | undefined>): string {
  const query = new URLSearchParams()
  for (const [key, value] of Object.entries(params)) {
    if (value === undefined || value === null || value === '') {
      continue
    }
    query.set(key, String(value))
  }
  return query.toString()
}

export async function listAdminSessions(
  params: ListAdminSessionsParams = {},
): Promise<AdminChatSessionRecord[]> {
  const query = buildQuery({
    project_id: params.project_id,
    owner_user_id: params.owner_user_id,
    include_deleted: params.include_deleted ?? false,
    limit: params.limit ?? 200,
    offset: params.offset ?? 0,
  })
  return apiJson<AdminChatSessionRecord[]>(`/api/v1/admin/memory/sessions?${query}`)
}

export async function deleteAdminSession(
  sessionId: string,
  payload: AdminDeleteSessionPayload,
): Promise<AdminSessionDeleteResponse> {
  return apiJson<AdminSessionDeleteResponse>(`/api/v1/admin/memory/sessions/${sessionId}`, {
    method: 'DELETE',
    body: JSON.stringify({
      delete_linked_engrams: payload.delete_linked_engrams ?? false,
      reason: payload.reason ?? null,
    }),
  })
}

export async function restoreAdminSession(sessionId: string): Promise<AdminSessionRestoreResponse> {
  return apiJson<AdminSessionRestoreResponse>(`/api/v1/admin/memory/sessions/${sessionId}/restore`, {
    method: 'POST',
  })
}

export async function listAdminEngrams(
  params: ListAdminEngramsParams = {},
): Promise<AdminEngramRecord[]> {
  const query = buildQuery({
    project_id: params.project_id,
    session_id: params.session_id,
    q: params.q,
    include_deleted: params.include_deleted ?? false,
    limit: params.limit ?? 200,
    offset: params.offset ?? 0,
  })
  return apiJson<AdminEngramRecord[]>(`/api/v1/admin/memory/engrams?${query}`)
}

export async function getAdminEngram(
  engramId: string,
  includeDeleted = true,
): Promise<AdminEngramRecord> {
  const query = buildQuery({ include_deleted: includeDeleted })
  return apiJson<AdminEngramRecord>(`/api/v1/admin/memory/engrams/${engramId}?${query}`)
}

export async function updateAdminEngram(
  engramId: string,
  payload: UpdateAdminEngramPayload,
): Promise<AdminEngramRecord> {
  return apiJson<AdminEngramRecord>(`/api/v1/admin/memory/engrams/${engramId}`, {
    method: 'PATCH',
    body: JSON.stringify(payload),
  })
}

export async function moveAdminEngram(
  engramId: string,
  payload: MoveAdminEngramPayload,
): Promise<AdminEngramRecord> {
  return apiJson<AdminEngramRecord>(`/api/v1/admin/memory/engrams/${engramId}/move`, {
    method: 'POST',
    body: JSON.stringify(payload),
  })
}

export async function deleteAdminEngram(
  engramId: string,
  payload: AdminDeletePayload,
): Promise<AdminEngramDeleteResponse> {
  return apiJson<AdminEngramDeleteResponse>(`/api/v1/admin/memory/engrams/${engramId}`, {
    method: 'DELETE',
    body: JSON.stringify({ reason: payload.reason ?? null }),
  })
}

export async function restoreAdminEngram(engramId: string): Promise<AdminEngramRestoreResponse> {
  return apiJson<AdminEngramRestoreResponse>(`/api/v1/admin/memory/engrams/${engramId}/restore`, {
    method: 'POST',
  })
}

export async function listCollections(
  params: ListCollectionsParams = {},
): Promise<EngramCollectionRecord[]> {
  const query = buildQuery({
    project_id: params.project_id,
    include_deleted: params.include_deleted ?? false,
    limit: params.limit ?? 200,
    offset: params.offset ?? 0,
  })
  return apiJson<EngramCollectionRecord[]>(`/api/v1/admin/memory/collections?${query}`)
}

export async function createCollection(payload: CreateCollectionPayload): Promise<EngramCollectionRecord> {
  return apiJson<EngramCollectionRecord>('/api/v1/admin/memory/collections', {
    method: 'POST',
    body: JSON.stringify({
      project_id: payload.project_id,
      name: payload.name,
      description: payload.description ?? '',
    }),
  })
}

export async function updateCollection(
  collectionId: string,
  payload: UpdateCollectionPayload,
): Promise<EngramCollectionRecord> {
  return apiJson<EngramCollectionRecord>(`/api/v1/admin/memory/collections/${collectionId}`, {
    method: 'PATCH',
    body: JSON.stringify(payload),
  })
}

export async function deleteCollection(collectionId: string, payload: AdminDeletePayload): Promise<{ deleted: boolean }> {
  return apiJson<{ deleted: boolean }>(`/api/v1/admin/memory/collections/${collectionId}`, {
    method: 'DELETE',
    body: JSON.stringify({ reason: payload.reason ?? null }),
  })
}

export async function addCollectionItems(
  collectionId: string,
  payload: UpdateCollectionItemsPayload,
): Promise<{ added: number }> {
  return apiJson<{ added: number }>(`/api/v1/admin/memory/collections/${collectionId}/items`, {
    method: 'POST',
    body: JSON.stringify(payload),
  })
}

export async function removeCollectionItem(
  collectionId: string,
  engramId: string,
): Promise<{ removed: boolean }> {
  return apiJson<{ removed: boolean }>(
    `/api/v1/admin/memory/collections/${collectionId}/items/${engramId}`,
    { method: 'DELETE' },
  )
}
