import { apiJson } from './http'
import type {
  ChatLifecyclePolicy,
  ChatMessage,
  ChatSession,
  ChatTimelineEvent,
  EngramSummary,
} from './types'
import type { CreateSessionPayload } from './chatTypes'

export async function listChatSessions(projectId: string): Promise<ChatSession[]> {
  const query = new URLSearchParams({ project_id: projectId })
  return apiJson<ChatSession[]>(`/api/v1/chat/sessions?${query.toString()}`)
}

export async function createChatSession(payload: CreateSessionPayload): Promise<ChatSession> {
  return apiJson<ChatSession>('/api/v1/chat/sessions', {
    method: 'POST',
    body: JSON.stringify(payload),
  })
}

export async function getLifecyclePolicy(sessionId: string): Promise<ChatLifecyclePolicy> {
  return apiJson<ChatLifecyclePolicy>(`/api/v1/chat/sessions/${sessionId}/lifecycle-policy`)
}

export async function updateLifecyclePolicy(
  sessionId: string,
  payload: Partial<ChatLifecyclePolicy>,
): Promise<ChatLifecyclePolicy> {
  return apiJson<ChatLifecyclePolicy>(`/api/v1/chat/sessions/${sessionId}/lifecycle-policy`, {
    method: 'PATCH',
    body: JSON.stringify(payload),
  })
}

export async function updateChatSession(
  sessionId: string,
  payload: Partial<CreateSessionPayload>,
): Promise<ChatSession> {
  return apiJson<ChatSession>(`/api/v1/chat/sessions/${sessionId}`, {
    method: 'PATCH',
    body: JSON.stringify(payload),
  })
}

export async function listSessionMessages(sessionId: string): Promise<ChatMessage[]> {
  return apiJson<ChatMessage[]>(`/api/v1/chat/sessions/${sessionId}/messages`)
}

export async function listSessionTimeline(sessionId: string): Promise<ChatTimelineEvent[]> {
  return apiJson<ChatTimelineEvent[]>(`/api/v1/chat/sessions/${sessionId}/timeline`)
}

export async function listEngrams(projectId: string): Promise<EngramSummary[]> {
  const query = new URLSearchParams({ project_id: projectId, limit: '200', offset: '0' })
  return apiJson<EngramSummary[]>(`/api/v1/engrams?${query.toString()}`)
}
