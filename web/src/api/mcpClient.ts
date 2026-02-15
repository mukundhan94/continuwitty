import { parseApiError } from './http'
import { parseSseStream } from '../utils/sse'

export interface McpJsonRpcRequest {
  jsonrpc: '2.0'
  id: string | number
  method: string
  params?: Record<string, unknown>
}

export interface McpJsonRpcErrorObject {
  code: number
  message: string
  data?: Record<string, unknown>
}

export interface McpJsonRpcResultFrame {
  jsonrpc: '2.0'
  id: string | number
  result: Record<string, unknown>
}

export interface McpJsonRpcErrorFrame {
  jsonrpc: '2.0'
  id: string | number
  error: McpJsonRpcErrorObject
}

export interface McpJsonRpcEventPayload {
  id: string | number
  tool: string
  event: string
  data: Record<string, unknown>
}

export interface McpJsonRpcEventFrame {
  jsonrpc: '2.0'
  method: 'mcp.event'
  params: McpJsonRpcEventPayload
}

export type McpJsonRpcFrame = McpJsonRpcResultFrame | McpJsonRpcErrorFrame | McpJsonRpcEventFrame

export class McpProtocolError extends Error {}

function isRecord(value: unknown): value is Record<string, unknown> {
  return typeof value === 'object' && value !== null && !Array.isArray(value)
}

export function parseMcpJsonRpcFrame(raw: unknown): McpJsonRpcFrame {
  // Keep runtime validation explicit because external MCP clients/tools can send
  // loosely typed payloads. This parser is the contract boundary for UI/tooling code.
  if (!isRecord(raw)) {
    throw new McpProtocolError('MCP frame must be a JSON object')
  }

  if (raw.jsonrpc !== '2.0') {
    throw new McpProtocolError('Invalid jsonrpc version in MCP frame')
  }

  if (raw.method === 'mcp.event') {
    if (!isRecord(raw.params)) {
      throw new McpProtocolError('MCP event frame missing params object')
    }
    const params = raw.params
    if (typeof params.tool !== 'string' || typeof params.event !== 'string' || !isRecord(params.data)) {
      throw new McpProtocolError('MCP event frame params are malformed')
    }
    if (typeof params.id !== 'string' && typeof params.id !== 'number') {
      throw new McpProtocolError('MCP event frame params.id is invalid')
    }
    return {
      jsonrpc: '2.0',
      method: 'mcp.event',
      params: {
        id: params.id,
        tool: params.tool,
        event: params.event,
        data: params.data,
      },
    }
  }

  if (typeof raw.id !== 'string' && typeof raw.id !== 'number') {
    throw new McpProtocolError('MCP result/error frame id is invalid')
  }

  if ('error' in raw) {
    if (!isRecord(raw.error) || typeof raw.error.code !== 'number' || typeof raw.error.message !== 'string') {
      throw new McpProtocolError('MCP error frame payload is malformed')
    }
    return {
      jsonrpc: '2.0',
      id: raw.id,
      error: {
        code: raw.error.code,
        message: raw.error.message,
        data: isRecord(raw.error.data) ? raw.error.data : undefined,
      },
    }
  }

  if ('result' in raw) {
    if (!isRecord(raw.result)) {
      throw new McpProtocolError('MCP result frame payload is malformed')
    }
    return {
      jsonrpc: '2.0',
      id: raw.id,
      result: raw.result,
    }
  }

  throw new McpProtocolError('Unsupported MCP JSON-RPC frame shape')
}

export async function streamMcpCall(
  request: McpJsonRpcRequest,
  path = '/api/v1/mcp/stream',
): Promise<McpJsonRpcFrame[]> {
  // `credentials: include` keeps parity with the app's cookie-session auth model.
  const response = await fetch(path, {
    credentials: 'include',
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
    },
    body: JSON.stringify({ ...request, params: request.params ?? {} }),
  })

  if (!response.ok) {
    throw await parseApiError(response)
  }

  if (!response.body) {
    throw new McpProtocolError('MCP stream response body is not available')
  }

  const frames: McpJsonRpcFrame[] = []
  for await (const sseFrame of parseSseStream(response.body)) {
    // Ignore non-JSON-RPC events so this helper is robust to future keep-alive
    // or telemetry event names in the same stream.
    if (sseFrame.event !== 'jsonrpc') {
      continue
    }
    frames.push(parseMcpJsonRpcFrame(sseFrame.data))
  }

  if (frames.length === 0) {
    throw new McpProtocolError('MCP stream returned no jsonrpc frames')
  }
  return frames
}

export function finalMcpResultFrame(frames: McpJsonRpcFrame[]): McpJsonRpcResultFrame | null {
  // Later frames supersede earlier ones; walk from the tail for the effective result.
  for (let index = frames.length - 1; index >= 0; index -= 1) {
    const frame = frames[index]
    if ('result' in frame) {
      return frame
    }
  }
  return null
}

export function finalMcpErrorFrame(frames: McpJsonRpcFrame[]): McpJsonRpcErrorFrame | null {
  // Mirrors `finalMcpResultFrame` semantics for deterministic conflict resolution.
  for (let index = frames.length - 1; index >= 0; index -= 1) {
    const frame = frames[index]
    if ('error' in frame) {
      return frame
    }
  }
  return null
}
