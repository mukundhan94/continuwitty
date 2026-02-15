import { afterEach, describe, expect, it, vi } from 'vitest'

import {
  McpProtocolError,
  finalMcpErrorFrame,
  finalMcpResultFrame,
  parseMcpJsonRpcFrame,
  streamMcpCall,
} from './mcpClient'

function createStream(payload: string): ReadableStream<Uint8Array> {
  const encoder = new TextEncoder()
  return new ReadableStream({
    start(controller) {
      controller.enqueue(encoder.encode(payload))
      controller.close()
    },
  })
}

function createResponse(params: {
  ok: boolean
  status: number
  body?: ReadableStream<Uint8Array> | null
  headers?: Record<string, string>
}): Response {
  return {
    ok: params.ok,
    status: params.status,
    statusText: '',
    body: params.body ?? null,
    headers: new Headers(params.headers ?? { 'content-type': 'text/event-stream' }),
    text: async () => '',
    json: async () => ({}),
  } as Response
}

afterEach(() => {
  vi.restoreAllMocks()
})

describe('parseMcpJsonRpcFrame', () => {
  it('parses result, event, and error frames', () => {
    const result = parseMcpJsonRpcFrame({
      jsonrpc: '2.0',
      id: 'abc',
      result: { session: { session_id: '1' } },
    })
    expect('result' in result).toBe(true)

    const event = parseMcpJsonRpcFrame({
      jsonrpc: '2.0',
      method: 'mcp.event',
      params: { id: 'abc', tool: 'chat.send_message', event: 'chunk', data: { text: 'hi' } },
    })
    expect('method' in event && event.method === 'mcp.event').toBe(true)

    const error = parseMcpJsonRpcFrame({
      jsonrpc: '2.0',
      id: 'abc',
      error: { code: -32010, message: 'bad' },
    })
    expect('error' in error).toBe(true)
  })

  it('rejects malformed payloads', () => {
    expect(() => parseMcpJsonRpcFrame('not-an-object')).toThrow(McpProtocolError)
    expect(() => parseMcpJsonRpcFrame({ jsonrpc: '1.0' })).toThrow(McpProtocolError)
  })
})

describe('streamMcpCall', () => {
  it('parses JSON-RPC frames from SSE payloads', async () => {
    const ssePayload =
      'event: ping\ndata: {"ok":true}\n\n' +
      'event: jsonrpc\ndata: {"jsonrpc":"2.0","method":"mcp.event","params":{"id":"1","tool":"chat.send_message","event":"chunk","data":{"text":"he"}}}\n\n' +
      'event: jsonrpc\ndata: {"jsonrpc":"2.0","id":"1","result":{"message":{"assistant_text":"hello"}}}\n\n'

    vi.spyOn(globalThis, 'fetch').mockResolvedValue(
      createResponse({
        ok: true,
        status: 200,
        body: createStream(ssePayload),
      }),
    )

    const frames = await streamMcpCall({
      jsonrpc: '2.0',
      id: '1',
      method: 'chat.send_message',
      params: { session_id: 'session-1', content_text: 'hello', stream: true },
    })

    expect(frames.length).toBe(2)
    expect(finalMcpErrorFrame(frames)).toBeNull()
    expect(finalMcpResultFrame(frames)?.result).toEqual({
      message: { assistant_text: 'hello' },
    })
  })

  it('fails when stream has no jsonrpc frames', async () => {
    vi.spyOn(globalThis, 'fetch').mockResolvedValue(
      createResponse({
        ok: true,
        status: 200,
        body: createStream('event: ping\ndata: {"ok":true}\n\n'),
      }),
    )

    await expect(
      streamMcpCall({
        jsonrpc: '2.0',
        id: 'empty',
        method: 'user.get_profile',
      }),
    ).rejects.toThrow(McpProtocolError)
  })
})
