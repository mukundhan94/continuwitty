import { apiJson, apiVoid } from './http'
import type { EngramSummary, PinnedDocumentRecord } from './types'

export async function listPinnedEngrams(sessionId: string): Promise<EngramSummary[]> {
  return apiJson<EngramSummary[]>(`/api/v1/chat/sessions/${sessionId}/engrams`)
}

export async function listPinnedDocuments(sessionId: string): Promise<PinnedDocumentRecord[]> {
  return apiJson<PinnedDocumentRecord[]>(`/api/v1/chat/sessions/${sessionId}/documents`)
}

export async function pinEngramToSession(sessionId: string, engramId: string): Promise<void> {
  await apiJson(`/api/v1/chat/sessions/${sessionId}/engrams/pin`, {
    method: 'POST',
    body: JSON.stringify({ engram_id: engramId }),
  })
}

export async function unpinEngramFromSession(sessionId: string, engramId: string): Promise<void> {
  await apiVoid(`/api/v1/chat/sessions/${sessionId}/engrams/${engramId}`, {
    method: 'DELETE',
  })
}

export async function pinDocumentToSession(sessionId: string, documentId: string): Promise<void> {
  await apiJson(`/api/v1/chat/sessions/${sessionId}/documents/pin`, {
    method: 'POST',
    body: JSON.stringify({ document_id: documentId }),
  })
}

export async function unpinDocumentFromSession(sessionId: string, documentId: string): Promise<void> {
  await apiVoid(`/api/v1/chat/sessions/${sessionId}/documents/${documentId}`, {
    method: 'DELETE',
  })
}
