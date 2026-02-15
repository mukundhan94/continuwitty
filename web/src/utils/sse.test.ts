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
})
