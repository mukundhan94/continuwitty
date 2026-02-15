export interface SseFrame<T = unknown> {
  event: string
  data: T
}

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

  while (true) {
    const result = await reader.read()
    if (result.done) {
      break
    }

    buffer += decoder.decode(result.value, { stream: true })

    while (true) {
      const frameBoundary = buffer.indexOf('\n\n')
      if (frameBoundary === -1) {
        break
      }

      const rawFrame = buffer.slice(0, frameBoundary)
      buffer = buffer.slice(frameBoundary + 2)

      let eventName = 'message'
      const dataParts: string[] = []
      const lines = rawFrame.split('\n')
      for (const line of lines) {
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
}
