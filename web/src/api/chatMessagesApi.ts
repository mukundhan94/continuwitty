import { apiJson, parseApiError } from './http'
import type { ContinueSessionResponse, ChatSendResponse, SaveSessionAsEngramResponse } from './types'
import { parseSseStream, type SseFrame } from '../utils/sse'
import type { ChatStreamEvent, SaveEngramPayload } from './chatTypes'

function isChatStreamEventName(eventName: string): eventName is ChatStreamEvent['event'] {
  return eventName === 'meta' || eventName === 'chunk' || eventName === 'done' || eventName === 'error'
}

function toChatStreamEvent(frame: SseFrame): ChatStreamEvent | null {
  if (!isChatStreamEventName(frame.event)) {
    return null
  }
  return {
    event: frame.event,
    data: frame.data,
  } as ChatStreamEvent
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
    const streamEvent = toChatStreamEvent(frame)
    if (streamEvent) {
      yield streamEvent
    }
  }
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
