import { describe, expect, it } from 'vitest'

import { parseSseStream } from './sse'

function createStream(payload: string): ReadableStream<Uint8Array> {
  const encoder = new TextEncoder()
  return new ReadableStream({
    start(controller) {
      controller.enqueue(encoder.encode(payload))
      controller.close()
    },
  })
}

function createChunkedStream(chunks: string[]): ReadableStream<Uint8Array> {
  const encoder = new TextEncoder()
  return new ReadableStream({
    start(controller) {
      for (const chunk of chunks) {
        controller.enqueue(encoder.encode(chunk))
      }
      controller.close()
    },
  })
}

describe('parseSseStream', () => {
  it('parses named events with json payloads', async () => {
    const stream = createStream(
      'event: meta\ndata: {"kind":"meta"}\n\n' + 'event: chunk\ndata: {"text":"hello"}\n\n',
    )

    const frames = []
    for await (const frame of parseSseStream(stream)) {
      frames.push(frame)
    }

    expect(frames).toEqual([
      { event: 'meta', data: { kind: 'meta' } },
      { event: 'chunk', data: { text: 'hello' } },
    ])
  })

  it('falls back to message event when name is omitted', async () => {
    const stream = createStream('data: plain-text\n\n')

    const frames = []
    for await (const frame of parseSseStream(stream)) {
      frames.push(frame)
    }

    expect(frames).toEqual([{ event: 'message', data: 'plain-text' }])
  })

  it('parses CRLF-separated frames across chunk boundaries', async () => {
    const stream = createChunkedStream([
      'event: meta\r\n',
      'data: {"kind":"meta"}\r\n',
      '\r\n',
      'event: chunk\r\n',
      'data: {"text":"hel',
      'lo"}\r\n',
      '\r\n',
    ])

    const frames = []
    for await (const frame of parseSseStream(stream)) {
      frames.push(frame)
    }

    expect(frames).toEqual([
      { event: 'meta', data: { kind: 'meta' } },
      { event: 'chunk', data: { text: 'hello' } },
    ])
  })
})
