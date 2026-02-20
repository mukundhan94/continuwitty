import { afterEach, describe, expect, it, vi } from 'vitest'

import { ApiError } from './http'
import {
  createChatSession,
  listChatSessions,
  streamChatMessage,
  type ChatStreamEvent,
  type CreateSessionPayload,
} from './chat'
import * as sse from '../utils/sse'

function mockResponse(params: {
  ok: boolean
  status: number
  statusText?: string
  contentType?: string
  jsonBody?: unknown
  body?: ReadableStream<Uint8Array> | null
}): Response {
  return {
    ok: params.ok,
    status: params.status,
    statusText: params.statusText ?? '',
    headers: new Headers(
      params.contentType
        ? {
            'content-type': params.contentType,
          }
        : {},
    ),
    body: params.body ?? null,
    json: async () => params.jsonBody ?? {},
    text: async () => '',
  } as Response
}

function emptyStream(): ReadableStream<Uint8Array> {
  return new ReadableStream<Uint8Array>({
    start(controller) {
      controller.close()
    },
  })
}

async function collectStream(stream: AsyncGenerator<ChatStreamEvent>): Promise<ChatStreamEvent[]> {
  const events: ChatStreamEvent[] = []
  for await (const event of stream) {
    events.push(event)
  }
  return events
}

function sessionPayload(): CreateSessionPayload {
  return {
    project_id: 'engram-vault',
    title: 'Incident timeline',
    provider: 'openai',
    model_id: 'gpt-4o-mini',
    system_prompt: '',
    visibility_scope: 'private',
    autosave_enabled: true,
    autosave_strategy: 'message_count',
    autosave_interval_minutes: 30,
    autosave_min_messages: 4,
    retention_days: 30,
    retention_max_snapshots: 60,
  }
}

afterEach(() => {
  vi.restoreAllMocks()
})

describe('chat api', () => {
  it('builds list session query parameters', async () => {
    const fetchMock = vi.spyOn(globalThis, 'fetch').mockResolvedValue(
      mockResponse({
        ok: true,
        status: 200,
        contentType: 'application/json',
        jsonBody: [],
      }),
    )

    await listChatSessions('engram-vault')

    const [path] = fetchMock.mock.calls[0]
    const resolvedPath = String(path)
    expect(resolvedPath).toContain('/api/v1/chat/sessions?')
    expect(resolvedPath).toContain('project_id=engram-vault')
  })

  it('posts create session payload', async () => {
    const payload = sessionPayload()
    const fetchMock = vi.spyOn(globalThis, 'fetch').mockResolvedValue(
      mockResponse({
        ok: true,
        status: 200,
        contentType: 'application/json',
        jsonBody: { session_id: 'session-1' },
      }),
    )

    await createChatSession(payload)

    const [, init] = fetchMock.mock.calls[0]
    const requestInit = init as RequestInit
    expect(requestInit.method).toBe('POST')
    expect(requestInit.body).toBe(JSON.stringify(payload))
  })

  it('streams only supported chat stream events', async () => {
    vi.spyOn(globalThis, 'fetch').mockResolvedValue(
      mockResponse({
        ok: true,
        status: 200,
        body: emptyStream(),
      }),
    )

    vi.spyOn(sse, 'parseSseStream').mockReturnValue(
      (async function* () {
        yield {
          event: 'meta',
          data: {
            session_id: 'session-1',
            message_id: 'message-1',
            used_engram_ids: [],
            used_document_chunk_ids: [],
            source_references: [],
          },
        }
        yield { event: 'unknown', data: { ignored: true } }
        yield { event: 'chunk', data: { text: 'hello' } }
        yield {
          event: 'done',
          data: {
            session_id: 'session-1',
            message_id: 'message-1',
            reply_message_id: 'message-2',
            assistant_text: 'hello',
            used_engram_ids: [],
            used_document_chunk_ids: [],
            source_references: [],
          },
        }
        yield {
          event: 'error',
          data: {
            detail: 'upstream unavailable',
            status_code: 503,
            error_code: 'provider_unavailable',
          },
        }
      })(),
    )

    const events = await collectStream(streamChatMessage('session-1', 'hi'))

    expect(events.map((event) => event.event)).toEqual(['meta', 'chunk', 'done', 'error'])
  })

  it('raises ApiError when stream endpoint returns non-ok response', async () => {
    vi.spyOn(globalThis, 'fetch').mockResolvedValue(
      mockResponse({
        ok: false,
        status: 400,
        statusText: 'Bad Request',
        contentType: 'application/json',
        jsonBody: { detail: 'invalid request' },
      }),
    )

    await expect(collectStream(streamChatMessage('session-1', 'hi'))).rejects.toBeInstanceOf(ApiError)
    await expect(collectStream(streamChatMessage('session-1', 'hi'))).rejects.toMatchObject({
      status: 400,
      detail: 'invalid request',
    })
  })

  it('fails when streaming response body is missing', async () => {
    vi.spyOn(globalThis, 'fetch').mockResolvedValue(
      mockResponse({
        ok: true,
        status: 200,
        body: null,
      }),
    )

    await expect(collectStream(streamChatMessage('session-1', 'hi'))).rejects.toThrow(
      'Streaming response body is not available',
    )
  })
})
