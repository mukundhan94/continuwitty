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

export async function setDefaultProject(projectId: string): Promise<ProjectDefaultResponse> {
  return apiJson<ProjectDefaultResponse>('/api/v1/projects/default', {
    method: 'PATCH',
    body: JSON.stringify({ project_id: projectId }),
  })
}

export async function listProjectMembers(
  projectId: string,
  params: ListProjectMembersParams = {},
): Promise<ProjectMemberRecord[]> {
  const query = buildQuery({
    include_revoked: params.include_revoked ?? false,
    limit: params.limit ?? 500,
    offset: params.offset ?? 0,
  })
  return apiJson<ProjectMemberRecord[]>(`/api/v1/projects/${projectId}/members?${query}`)
}

export async function addProjectMember(
  projectId: string,
  payload: AddProjectMemberPayload,
): Promise<ProjectMemberRecord> {
  return apiJson<ProjectMemberRecord>(`/api/v1/projects/${projectId}/members`, {
    method: 'POST',
    body: JSON.stringify(payload),
  })
}

export async function updateProjectMember(
  projectId: string,
  userId: string,
  payload: UpdateProjectMemberPayload,
): Promise<ProjectMemberRecord> {
  return apiJson<ProjectMemberRecord>(`/api/v1/projects/${projectId}/members/${userId}`, {
    method: 'PATCH',
    body: JSON.stringify(payload),
  })
}

export async function removeProjectMember(projectId: string, userId: string): Promise<{ removed: boolean }> {
  return apiJson<{ removed: boolean }>(`/api/v1/projects/${projectId}/members/${userId}`, {
    method: 'DELETE',
  })
}

export async function listProjectAuditEvents(
  projectId: string,
  params: ListProjectAuditEventsParams = {},
): Promise<ProjectAuditEventRecord[]> {
  const query = buildQuery({
    limit: params.limit ?? 200,
    offset: params.offset ?? 0,
  })
  return apiJson<ProjectAuditEventRecord[]>(`/api/v1/projects/${projectId}/audit-events?${query}`)
}
