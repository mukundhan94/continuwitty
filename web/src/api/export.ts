import { parseApiError } from './http'
import type {
  ProjectExportFormat,
  ProjectImportConflictPolicy,
  ProjectImportResponse,
} from './types'

export interface ExportProjectBundleParams {
  projectId: string
  format?: ProjectExportFormat
  collectionIds?: string[]
  includeEmbeddings?: boolean
}

export interface ExportProjectBundleResult {
  blob: Blob
  filename: string
  contentType: string
}

export interface ImportProjectBundleParams {
  projectId: string
  file: File
  conflictPolicy: ProjectImportConflictPolicy
}

function sanitizeProjectId(projectId: string): string {
  return projectId.trim()
}

function buildExportFilename(projectId: string, format: ProjectExportFormat): string {
  const suffix = format === 'zip' ? 'zip' : 'json'
  const safeProjectId = projectId.replaceAll('/', '-').replaceAll(' ', '-')
  return `engram-export-${safeProjectId}.${suffix}`
}

function parseFilenameFromDisposition(headerValue: string | null): string | null {
  if (!headerValue) {
    return null
  }
  const match = /filename="([^"]+)"/i.exec(headerValue)
  if (!match?.[1]) {
    return null
  }
  return match[1]
}

function buildExportQuery(params: ExportProjectBundleParams): string {
  const query = new URLSearchParams()
  query.set('format', params.format ?? 'json')
  query.set('include_embeddings', params.includeEmbeddings ? 'true' : 'false')
  for (const collectionId of params.collectionIds ?? []) {
    const normalized = collectionId.trim()
    if (!normalized) {
      continue
    }
    query.append('collection_ids', normalized)
  }
  return query.toString()
}

export async function exportProjectBundle(
  params: ExportProjectBundleParams,
): Promise<ExportProjectBundleResult> {
  const projectId = sanitizeProjectId(params.projectId)
  const format = params.format ?? 'json'
  const query = buildExportQuery(params)
  const response = await fetch(
    `/api/v1/projects/${encodeURIComponent(projectId)}/export?${query}`,
    { credentials: 'include' },
  )

  if (!response.ok) {
    throw await parseApiError(response)
  }

  const filename =
    parseFilenameFromDisposition(response.headers.get('content-disposition')) ??
    buildExportFilename(projectId, format)

  return {
    blob: await response.blob(),
    filename,
    contentType: response.headers.get('content-type') || 'application/octet-stream',
  }
}

export async function importProjectBundle(
  params: ImportProjectBundleParams,
): Promise<ProjectImportResponse> {
  const projectId = sanitizeProjectId(params.projectId)
  const query = new URLSearchParams({
    conflict_policy: params.conflictPolicy,
  })
  const formData = new FormData()
  formData.append('file', params.file)

  const response = await fetch(
    `/api/v1/projects/${encodeURIComponent(projectId)}/import?${query.toString()}`,
    {
      method: 'POST',
      credentials: 'include',
      body: formData,
    },
  )

  if (!response.ok) {
    throw await parseApiError(response)
  }

  return (await response.json()) as ProjectImportResponse
}
