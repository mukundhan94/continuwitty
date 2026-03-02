import { apiJson } from './http'
import type {
  AdminChatSessionRecord,
  AdminEngramDeleteResponse,
  AdminEngramRecord,
  AdminEngramRestoreResponse,
  AdminSessionDeleteResponse,
  AdminSessionRestoreResponse,
  EngramVisibilityRecord,
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

export interface DeleteAdminSessionInput {
  session_id: string
  payload: AdminDeleteSessionPayload
}

export interface RestoreAdminSessionInput {
  session_id: string
}

export interface GetAdminEngramInput {
  engram_id: string
  include_deleted?: boolean
}

export interface UpdateAdminEngramInput {
  engram_id: string
  payload: UpdateAdminEngramPayload
}

export interface MoveAdminEngramInput {
  engram_id: string
  payload: MoveAdminEngramPayload
}

export interface DeleteAdminEngramInput {
  engram_id: string
  payload: AdminDeletePayload
}

export interface RestoreAdminEngramInput {
  engram_id: string
}

export interface UpdateCollectionInput {
  collection_id: string
  payload: UpdateCollectionPayload
}

export interface DeleteCollectionInput {
  collection_id: string
  payload: AdminDeletePayload
}

export interface AddCollectionItemsInput {
  collection_id: string
  payload: UpdateCollectionItemsPayload
}

export interface RemoveCollectionItemInput {
  collection_id: string
  engram_id: string
}

export interface ShareEngramInput {
  engram_id: string
}

export interface UnshareEngramInput {
  engram_id: string
}

function buildQuery(params: Record<string, string | number | boolean | undefined>): string {
  const query = new URLSearchParams()
  for (const [key, value] of Object.entries(params)) {
    if (!isPresentQueryValue(value)) {
      continue
    }
    query.set(key, String(value))
  }
  return query.toString()
}

function isPresentQueryValue(value: string | number | boolean | undefined): boolean {
  return value !== undefined && value !== null && value !== ''
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

export async function deleteAdminSession(input: DeleteAdminSessionInput): Promise<AdminSessionDeleteResponse> {
  return apiJson<AdminSessionDeleteResponse>(`/api/v1/admin/memory/sessions/${input.session_id}`, {
    method: 'DELETE',
    body: JSON.stringify({
      delete_linked_engrams: input.payload.delete_linked_engrams ?? false,
      reason: input.payload.reason ?? null,
    }),
  })
}

export async function restoreAdminSession(input: RestoreAdminSessionInput): Promise<AdminSessionRestoreResponse> {
  return apiJson<AdminSessionRestoreResponse>(`/api/v1/admin/memory/sessions/${input.session_id}/restore`, {
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

export async function getAdminEngram(input: GetAdminEngramInput): Promise<AdminEngramRecord> {
  const query = buildQuery({ include_deleted: input.include_deleted ?? true })
  return apiJson<AdminEngramRecord>(`/api/v1/admin/memory/engrams/${input.engram_id}?${query}`)
}

export async function updateAdminEngram(input: UpdateAdminEngramInput): Promise<AdminEngramRecord> {
  return apiJson<AdminEngramRecord>(`/api/v1/admin/memory/engrams/${input.engram_id}`, {
    method: 'PATCH',
    body: JSON.stringify(input.payload),
  })
}

export async function moveAdminEngram(input: MoveAdminEngramInput): Promise<AdminEngramRecord> {
  return apiJson<AdminEngramRecord>(`/api/v1/admin/memory/engrams/${input.engram_id}/move`, {
    method: 'POST',
    body: JSON.stringify(input.payload),
  })
}

export async function deleteAdminEngram(input: DeleteAdminEngramInput): Promise<AdminEngramDeleteResponse> {
  return apiJson<AdminEngramDeleteResponse>(`/api/v1/admin/memory/engrams/${input.engram_id}`, {
    method: 'DELETE',
    body: JSON.stringify({ reason: input.payload.reason ?? null }),
  })
}

export async function restoreAdminEngram(input: RestoreAdminEngramInput): Promise<AdminEngramRestoreResponse> {
  return apiJson<AdminEngramRestoreResponse>(`/api/v1/admin/memory/engrams/${input.engram_id}/restore`, {
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

export async function updateCollection(input: UpdateCollectionInput): Promise<EngramCollectionRecord> {
  return apiJson<EngramCollectionRecord>(`/api/v1/admin/memory/collections/${input.collection_id}`, {
    method: 'PATCH',
    body: JSON.stringify(input.payload),
  })
}

export async function deleteCollection(input: DeleteCollectionInput): Promise<{ deleted: boolean }> {
  return apiJson<{ deleted: boolean }>(`/api/v1/admin/memory/collections/${input.collection_id}`, {
    method: 'DELETE',
    body: JSON.stringify({ reason: input.payload.reason ?? null }),
  })
}

export async function addCollectionItems(input: AddCollectionItemsInput): Promise<{ added: number }> {
  return apiJson<{ added: number }>(`/api/v1/admin/memory/collections/${input.collection_id}/items`, {
    method: 'POST',
    body: JSON.stringify(input.payload),
  })
}

export async function removeCollectionItem(input: RemoveCollectionItemInput): Promise<{ removed: boolean }> {
  return apiJson<{ removed: boolean }>(
    `/api/v1/admin/memory/collections/${input.collection_id}/items/${input.engram_id}`,
    { method: 'DELETE' },
  )
}

export async function shareEngram(input: ShareEngramInput): Promise<EngramVisibilityRecord> {
  return apiJson<EngramVisibilityRecord>(`/api/v1/engrams/${input.engram_id}/share`, {
    method: 'POST',
  })
}

export async function unshareEngram(input: UnshareEngramInput): Promise<EngramVisibilityRecord> {
  return apiJson<EngramVisibilityRecord>(`/api/v1/engrams/${input.engram_id}/unshare`, {
    method: 'POST',
  })
}
