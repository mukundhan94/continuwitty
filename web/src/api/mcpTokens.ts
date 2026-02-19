import { apiJson } from './http'
import {
  finalMcpErrorFrame,
  finalMcpResultFrame,
  streamMcpCall,
} from './mcpClient'
import type {
  McpTokenCreateRequest,
  McpTokenCreateResponse,
  McpTokenSummary,
} from './types'

export function listMcpTokens(): Promise<McpTokenSummary[]> {
  return apiJson<McpTokenSummary[]>('/api/v1/mcp/tokens')
}

export function createMcpToken(payload: McpTokenCreateRequest): Promise<McpTokenCreateResponse> {
  return apiJson<McpTokenCreateResponse>('/api/v1/mcp/tokens', {
    method: 'POST',
    body: JSON.stringify(payload),
  })
}

export function revokeMcpToken(tokenId: string, reason = 'rotation'): Promise<McpTokenSummary> {
  return apiJson<McpTokenSummary>(`/api/v1/mcp/tokens/${tokenId}/revoke`, {
    method: 'POST',
    body: JSON.stringify({ reason }),
  })
}

async function mcpResult(method: string, params: Record<string, unknown> = {}): Promise<Record<string, unknown>> {
  const frames = await streamMcpCall({
    jsonrpc: '2.0',
    id: `mcp-token-ui-${Date.now()}-${Math.floor(Math.random() * 1000)}`,
    method,
    params,
  })
  const error = finalMcpErrorFrame(frames)
  if (error) {
    throw new Error(`MCP ${method} failed: ${error.error.message}`)
  }
  const result = finalMcpResultFrame(frames)
  if (!result) {
    throw new Error(`MCP ${method} returned no result frame`)
  }
  return result.result
}

export async function listAvailableMcpTools(): Promise<string[]> {
  const result = await mcpResult('tools/list')
  const toolsRaw = result.tools
  if (!Array.isArray(toolsRaw)) {
    return []
  }
  return toolsRaw
    .map((item) => {
      if (typeof item === 'object' && item && typeof (item as { name?: unknown }).name === 'string') {
        return (item as { name: string }).name
      }
      return ''
    })
    .filter((item) => item.length > 0)
    .sort()
}

export async function listAvailableMcpProjects(): Promise<string[]> {
  const result = await mcpResult('user.list_projects')
  const projectsRaw = result.project_ids
  if (!Array.isArray(projectsRaw)) {
    return []
  }
  return projectsRaw
    .filter((item): item is string => typeof item === 'string' && item.trim().length > 0)
    .sort()
}
