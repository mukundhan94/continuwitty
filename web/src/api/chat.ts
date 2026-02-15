import { apiJson, apiVoid, parseApiError } from './http'
import type {
  ChatMessage,
  ChatSendResponse,
  ChatSession,
  ChatSourceReference,
  ContinueSessionResponse,
  EngramSummary,
  SaveSessionAsEngramResponse,
  VisibilityScope,
} from './types'
import { parseSseStream } from '../utils/sse'

export interface CreateSessionPayload {
  project_id: string
  title: string
  provider: 'openai' | 'anthropic' | 'bedrock'
  model_id: string
  system_prompt: string
  visibility_scope: VisibilityScope
  autosave_enabled: boolean
}

export interface SaveEngramPayload {
  title: string
  abstract: string
  visibility_scope: VisibilityScope
  tags: string[]
  keywords: string[]
}

export interface StreamMetaPayload {
  session_id: string
  message_id: string
  used_engram_ids: string[]
  source_references: ChatSourceReference[]
}

export interface StreamChunkPayload {
  text: string
}

export interface StreamDonePayload {
  session_id: string
  message_id: string
  reply_message_id: string
  assistant_text: string
  used_engram_ids: string[]
  source_references: ChatSourceReference[]
}

export interface StreamErrorPayload {
  detail: string
  status_code: number
  error_code: string
}

export type ChatStreamEvent =
  | { event: 'meta'; data: StreamMetaPayload }
  | { event: 'chunk'; data: StreamChunkPayload }
  | { event: 'done'; data: StreamDonePayload }
  | { event: 'error'; data: StreamErrorPayload }

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

export async function listPinnedEngrams(sessionId: string): Promise<EngramSummary[]> {
  return apiJson<EngramSummary[]>(`/api/v1/chat/sessions/${sessionId}/engrams`)
}

export async function sendChatMessage(sessionId: string, contentText: string): Promise<ChatSendResponse> {
  return apiJson<ChatSendResponse>(`/api/v1/chat/sessions/${sessionId}/messages`, {
    method: 'POST',
    body: JSON.stringify({ content_text: contentText }),
  })
}

export async function* streamChatMessage(
  sessionId: string,
  contentText: string,
): AsyncGenerator<ChatStreamEvent> {
  const response = await fetch(`/api/v1/chat/sessions/${sessionId}/messages/stream`, {
    credentials: 'include',
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
    },
    body: JSON.stringify({ content_text: contentText }),
  })

  if (!response.ok) {
    throw await parseApiError(response)
  }

  if (!response.body) {
    throw new Error('Streaming response body is not available')
  }

  for await (const frame of parseSseStream(response.body)) {
    if (frame.event === 'meta' || frame.event === 'chunk' || frame.event === 'done' || frame.event === 'error') {
      yield { event: frame.event, data: frame.data } as ChatStreamEvent
    }
  }
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

export async function saveSessionAsEngram(
  sessionId: string,
  payload: SaveEngramPayload,
): Promise<SaveSessionAsEngramResponse> {
  return apiJson<SaveSessionAsEngramResponse>(`/api/v1/chat/sessions/${sessionId}/save-engram`, {
    method: 'POST',
    body: JSON.stringify(payload),
  })
}

export async function continueSession(
  sessionId: string,
  title?: string,
): Promise<ContinueSessionResponse> {
  return apiJson<ContinueSessionResponse>(`/api/v1/chat/sessions/${sessionId}/continue`, {
    method: 'POST',
    body: JSON.stringify({ title }),
  })
}

export async function listEngrams(projectId: string): Promise<EngramSummary[]> {
  const query = new URLSearchParams({ project_id: projectId, limit: '200', offset: '0' })
  return apiJson<EngramSummary[]>(`/api/v1/engrams?${query.toString()}`)
}
