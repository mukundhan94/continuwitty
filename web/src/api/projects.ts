import { apiJson } from './http'
import type { ProjectDefaultResponse, ProjectRecord } from './types'

export interface CreateProjectPayload {
  project_id: string
  name: string
  description?: string
  owner_user_id?: string
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
