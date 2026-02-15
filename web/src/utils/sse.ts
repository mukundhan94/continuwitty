export interface SseFrame<T = unknown> {
  event: string
  data: T
}

const FRAME_SEPARATOR = /\n\n/

function parseDataLine(rawData: string): unknown {
  try {
    return JSON.parse(rawData)
  } catch {
    return rawData
  }
}

export async function* parseSseStream(stream: ReadableStream<Uint8Array>): AsyncGenerator<SseFrame> {
  const reader = stream.getReader()
  const decoder = new TextDecoder()
  let buffer = ''

  const consumeFrames = function* (): Generator<SseFrame> {
    while (true) {
      const match = FRAME_SEPARATOR.exec(buffer)
      if (!match || match.index < 0) {
        break
      }
      const frameBoundary = match.index
      const rawFrame = buffer.slice(0, frameBoundary)
      buffer = buffer.slice(frameBoundary + match[0].length)

      let eventName = 'message'
      const dataParts: string[] = []
      const lines = rawFrame.split('\n')
      for (const rawLine of lines) {
        const line = rawLine.trimEnd()
        if (line.startsWith('event:')) {
          eventName = line.slice(6).trim() || 'message'
        }
        if (line.startsWith('data:')) {
          dataParts.push(line.slice(5).trim())
        }
      }

      if (dataParts.length === 0) {
        continue
      }

      const rawData = dataParts.join('\n')
      yield {
        event: eventName,
        data: parseDataLine(rawData),
      }
    }
  }

  while (true) {
    const result = await reader.read()
    if (result.done) {
      break
    }

    buffer += decoder.decode(result.value, { stream: true })
    buffer = buffer.replace(/\r\n/g, '\n')
    buffer = buffer.replace(/\r/g, '\n')

    for (const frame of consumeFrames()) {
      yield frame
    }
  }

  buffer += decoder.decode()
  buffer = buffer.replace(/\r\n/g, '\n')
  buffer = buffer.replace(/\r/g, '\n')
  if (buffer.trim().length > 0 && !buffer.endsWith('\n\n')) {
    buffer += '\n\n'
  }
  for (const frame of consumeFrames()) {
    yield frame
  }
}
