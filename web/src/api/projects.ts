import { apiJson } from './http'
import type {
  ProjectAuditEventRecord,
  ProjectDefaultResponse,
  ProjectMemberRecord,
  ProjectMemberRole,
  ProjectRecord,
} from './types'

export interface CreateProjectPayload {
  project_id: string
  name: string
  description?: string
  owner_user_id?: string
}

export interface ListProjectMembersParams {
  include_revoked?: boolean
  limit?: number
  offset?: number
}

export interface AddProjectMemberPayload {
  user_id: string
  role: ProjectMemberRole
}

export interface UpdateProjectMemberPayload {
  role: ProjectMemberRole
}

export interface ListProjectAuditEventsParams {
  limit?: number
  offset?: number
}

export interface SetDefaultProjectInput {
  project_id: string
}

export interface ListProjectMembersInput {
  project_id: string
  params?: ListProjectMembersParams
}

export interface AddProjectMemberInput {
  project_id: string
  payload: AddProjectMemberPayload
}

export interface UpdateProjectMemberInput {
  project_id: string
  user_id: string
  payload: UpdateProjectMemberPayload
}

export interface RemoveProjectMemberInput {
  project_id: string
  user_id: string
}

export interface ListProjectAuditEventsInput {
  project_id: string
  params?: ListProjectAuditEventsParams
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

export async function listProjects(includeArchived = false): Promise<ProjectRecord[]> {
  const query = new URLSearchParams({
    include_archived: includeArchived ? 'true' : 'false',
    limit: '500',
    offset: '0',
  })
  return apiJson<ProjectRecord[]>(`/api/v1/projects?${query.toString()}`)
}

export async function createProject(payload: CreateProjectPayload): Promise<ProjectRecord> {
  return apiJson<ProjectRecord>('/api/v1/projects', {
    method: 'POST',
    body: JSON.stringify({
      project_id: payload.project_id,
      name: payload.name,
      description: payload.description ?? '',
      owner_user_id: payload.owner_user_id,
    }),
  })
}

export async function getDefaultProject(): Promise<ProjectDefaultResponse> {
  return apiJson<ProjectDefaultResponse>('/api/v1/projects/default')
}

export async function setDefaultProject(input: SetDefaultProjectInput): Promise<ProjectDefaultResponse> {
  return apiJson<ProjectDefaultResponse>('/api/v1/projects/default', {
    method: 'PATCH',
    body: JSON.stringify({ project_id: input.project_id }),
  })
}

export async function listProjectMembers(
  input: ListProjectMembersInput,
): Promise<ProjectMemberRecord[]> {
  const params = input.params ?? {}
  const query = buildQuery({
    include_revoked: params.include_revoked ?? false,
    limit: params.limit ?? 500,
    offset: params.offset ?? 0,
  })
  return apiJson<ProjectMemberRecord[]>(`/api/v1/projects/${input.project_id}/members?${query}`)
}

export async function addProjectMember(input: AddProjectMemberInput): Promise<ProjectMemberRecord> {
  return upsertProjectMember(`/api/v1/projects/${input.project_id}/members`, 'POST', input.payload)
}

export async function updateProjectMember(input: UpdateProjectMemberInput): Promise<ProjectMemberRecord> {
  return upsertProjectMember(
    `/api/v1/projects/${input.project_id}/members/${input.user_id}`,
    'PATCH',
    input.payload,
  )
}

async function upsertProjectMember(
  path: string,
  method: 'POST' | 'PATCH',
  payload: AddProjectMemberPayload | UpdateProjectMemberPayload,
): Promise<ProjectMemberRecord> {
  return apiJson<ProjectMemberRecord>(path, {
    method,
    body: JSON.stringify(payload),
  })
}

export async function removeProjectMember(input: RemoveProjectMemberInput): Promise<{ removed: boolean }> {
  return apiJson<{ removed: boolean }>(
    `/api/v1/projects/${input.project_id}/members/${input.user_id}`,
    {
    method: 'DELETE',
    },
  )
}

export async function listProjectAuditEvents(
  input: ListProjectAuditEventsInput,
): Promise<ProjectAuditEventRecord[]> {
  const params = input.params ?? {}
  const query = buildQuery({
    limit: params.limit ?? 200,
    offset: params.offset ?? 0,
  })
  return apiJson<ProjectAuditEventRecord[]>(`/api/v1/projects/${input.project_id}/audit-events?${query}`)
}
