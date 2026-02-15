import { apiJson, parseApiError } from './http'
import type { DocumentRecord, VisibilityScope } from './types'

interface IngestTextPayload {
  project_id: string
  title: string
  text: string
  visibility_scope: VisibilityScope
  chunk_size_chars: number
  chunk_overlap_chars: number
  metadata?: Record<string, unknown>
}

interface IngestFilePayload {
  project_id: string
  title?: string
  visibility_scope: VisibilityScope
  chunk_size_chars: number
  chunk_overlap_chars: number
  metadata?: Record<string, unknown>
  file: File
}

interface IngestResponse {
  document: DocumentRecord
}

export async function listProjectDocuments(projectId: string): Promise<DocumentRecord[]> {
  const query = new URLSearchParams({ project_id: projectId, limit: '200', offset: '0' })
  return apiJson<DocumentRecord[]>(`/api/v1/ingestion/documents?${query.toString()}`)
}

export async function ingestTextDocument(payload: IngestTextPayload): Promise<DocumentRecord> {
  const response = await apiJson<IngestResponse>('/api/v1/ingestion/text', {
    method: 'POST',
    body: JSON.stringify(payload),
  })
  return response.document
}

export async function ingestFileDocument(payload: IngestFilePayload): Promise<DocumentRecord> {
  const body = new FormData()
  body.set('project_id', payload.project_id)
  body.set('visibility_scope', payload.visibility_scope)
  body.set('chunk_size_chars', String(payload.chunk_size_chars))
  body.set('chunk_overlap_chars', String(payload.chunk_overlap_chars))
  if (payload.title && payload.title.trim()) {
    body.set('title', payload.title.trim())
  }
  if (payload.metadata) {
    body.set('metadata_json', JSON.stringify(payload.metadata))
  }
  body.set('file', payload.file)

  const response = await fetch('/api/v1/ingestion/file', {
    method: 'POST',
    credentials: 'include',
    body,
  })

  if (!response.ok) {
    throw await parseApiError(response)
  }

  const parsed = (await response.json()) as IngestResponse
  return parsed.document
}
