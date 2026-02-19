import { randomUUID } from 'node:crypto'

import type { Route } from '@playwright/test'

import { Given, When, Then, expect } from '../support/fixtures'

type McpFrame = {
  jsonrpc?: '2.0'
  id?: string | number
  result?: {
    tool_name?: string
    tools?: Array<{ name: string }>
    structuredContent?: {
      session?: { session_id: string }
      engram?: { engram_id: string }
    }
  }
  error?: {
    code: number
    message: string
    data?: Record<string, unknown>
  }
}

type MockToken = {
  tokenId: string
  token: string
  scope: 'read' | 'write'
  allowedTools: string[]
  allowedProjectIds: string[]
  name: string
  createdAt: string
  expiresAt: string
}

const mockTokens = new Map<string, MockToken>()

const readTools = ['chat.list_sessions', 'engram.query', 'engram.rehydrate', 'user.get_profile']
const writeTools = ['chat.create_session', 'chat.send_message', 'engram.create']

let latestBearerToken: string | null = null
let latestTokenScope: 'read' | 'write' | null = null
let latestVisibleToolNames: string[] = []
let latestWriteAttemptFrames: McpFrame[] = []
let latestCreatedSessionId: string | null = null
let latestCreatedEngramId: string | null = null

function asJson(route: Route, status: number, payload: unknown): Promise<void> {
  return route.fulfill({
    status,
    contentType: 'application/json',
    body: JSON.stringify(payload),
  })
}

function asSse(route: Route, frames: McpFrame[]): Promise<void> {
  const body = frames
    .map((frame) => `event: jsonrpc\ndata: ${JSON.stringify(frame)}\n\n`)
    .join('')
  return route.fulfill({
    status: 200,
    headers: {
      'content-type': 'text/event-stream',
      'cache-control': 'no-cache',
      connection: 'keep-alive',
    },
    body,
  })
}

function parseMcpFrames(rawSse: string): McpFrame[] {
  const frames: McpFrame[] = []
  for (const line of rawSse.split('\n')) {
    if (!line.startsWith('data: ')) {
      continue
    }
    frames.push(JSON.parse(line.slice(6)) as McpFrame)
  }
  return frames
}

function finalResultFrame(frames: McpFrame[]): McpFrame {
  const results = frames.filter((frame) => frame.result)
  if (results.length === 0) {
    throw new Error(`No MCP result frame found: ${JSON.stringify(frames)}`)
  }
  return results[results.length - 1]
}

function firstErrorFrame(frames: McpFrame[]): McpFrame {
  const errors = frames.filter((frame) => frame.error)
  if (errors.length === 0) {
    throw new Error(`No MCP error frame found: ${JSON.stringify(frames)}`)
  }
  return errors[0]
}

function tokenFromAuthHeader(authorization: string | undefined): string | null {
  if (!authorization) {
    return null
  }
  const trimmed = authorization.trim()
  if (!trimmed.toLowerCase().startsWith('bearer ')) {
    return null
  }
  const token = trimmed.slice(7).trim()
  return token.length > 0 ? token : null
}

function allowedToolsForToken(token: MockToken): string[] {
  const baseline = token.scope === 'write' ? [...new Set([...readTools, ...writeTools])] : readTools
  if (token.allowedTools.length === 0) {
    return baseline
  }
  return baseline.filter((toolName) => token.allowedTools.includes(toolName))
}

function projectAllowed(token: MockToken, projectId: string | undefined): boolean {
  if (token.allowedProjectIds.length === 0) {
    return true
  }
  if (!projectId) {
    return false
  }
  return token.allowedProjectIds.includes(projectId)
}

Given('mocked MCP token backend is enabled', async ({ page }) => {
  mockTokens.clear()
  latestBearerToken = null
  latestTokenScope = null
  latestVisibleToolNames = []
  latestWriteAttemptFrames = []
  latestCreatedSessionId = null
  latestCreatedEngramId = null

  await page.route('**/api/v1/mcp/tokens', async (route) => {
    const request = route.request()
    if (request.method() !== 'POST') {
      await route.fallback()
      return
    }

    const payload = request.postDataJSON() as {
      name?: string
      scope?: 'read' | 'write'
      allowed_tools?: string[]
      allowed_project_ids?: string[]
      expires_in_days?: number
    }

    const tokenId = randomUUID()
    const secret = randomUUID().replace(/-/g, '')
    const token = `engram_mcp_${tokenId.replace(/-/g, '')}_${secret}`
    const createdAt = new Date().toISOString()
    const expiresAt = new Date(Date.now() + (payload.expires_in_days ?? 90) * 24 * 60 * 60 * 1000)
      .toISOString()

    const record: MockToken = {
      tokenId,
      token,
      name: payload.name || 'mock-token',
      scope: payload.scope || 'read',
      allowedTools: payload.allowed_tools || [],
      allowedProjectIds: payload.allowed_project_ids || [],
      createdAt,
      expiresAt,
    }
    mockTokens.set(token, record)

    await asJson(route, 201, {
      token_id: record.tokenId,
      name: record.name,
      scope: record.scope,
      allowed_tools: record.allowedTools,
      allowed_project_ids: record.allowedProjectIds,
      token_secret_hint: `${secret.slice(0, 6)}...${secret.slice(-4)}`,
      token,
      expires_at: record.expiresAt,
      created_at: record.createdAt,
    })
  })

  await page.route('**/api/v1/mcp/stream', async (route) => {
    const request = route.request()
    if (request.method() !== 'POST') {
      await route.fallback()
      return
    }

    const authToken = tokenFromAuthHeader((await request.headerValue('authorization')) || undefined)
    if (!authToken || !mockTokens.has(authToken)) {
      await asJson(route, 401, { detail: 'Authentication required' })
      return
    }

    const tokenPolicy = mockTokens.get(authToken)
    if (!tokenPolicy) {
      await asJson(route, 401, { detail: 'Authentication required' })
      return
    }

    const rpc = request.postDataJSON() as {
      id?: string | number
      method?: string
      params?: Record<string, unknown>
    }
    const requestId = rpc.id || 'mock-request'
    const method = rpc.method || ''
    const params = rpc.params || {}

    if (method === 'tools/list') {
      const tools = allowedToolsForToken(tokenPolicy).map((name) => ({ name }))
      await asSse(route, [{ jsonrpc: '2.0', id: requestId, result: { tools } }])
      return
    }

    if (method === 'tools/call') {
      const toolName = String(params.name || '')
      const args = (params.arguments as Record<string, unknown> | undefined) || {}
      const projectId = typeof args.project_id === 'string' ? args.project_id : undefined
      const isWriteTool = writeTools.includes(toolName)

      if (tokenPolicy.scope === 'read' && isWriteTool) {
        await asSse(route, [
          {
            jsonrpc: '2.0',
            id: requestId,
            error: {
              code: -32003,
              message: 'Token scope does not allow this tool',
              data: {
                tool: toolName,
                required_scope: 'write',
                token_scope: 'read',
              },
            },
          },
        ])
        return
      }

      if (!projectAllowed(tokenPolicy, projectId)) {
        await asSse(route, [
          {
            jsonrpc: '2.0',
            id: requestId,
            error: {
              code: -32003,
              message: 'Project not allowed by token policy',
              data: {
                tool: toolName,
                required_scope: isWriteTool ? 'write' : 'read',
                token_scope: tokenPolicy.scope,
                project_id: projectId,
              },
            },
          },
        ])
        return
      }

      if (toolName === 'chat.create_session') {
        await asSse(route, [
          {
            jsonrpc: '2.0',
            id: requestId,
            result: {
              tool_name: toolName,
              structuredContent: {
                session: { session_id: randomUUID() },
              },
            },
          },
        ])
        return
      }

      if (toolName === 'engram.create') {
        await asSse(route, [
          {
            jsonrpc: '2.0',
            id: requestId,
            result: {
              tool_name: toolName,
              structuredContent: {
                engram: { engram_id: randomUUID() },
              },
            },
          },
        ])
        return
      }

      await asSse(route, [{ jsonrpc: '2.0', id: requestId, result: { tool_name: toolName } }])
      return
    }

    await asSse(route, [
      {
        jsonrpc: '2.0',
        id: requestId,
        error: {
          code: -32601,
          message: 'Method not found',
          data: { method },
        },
      },
    ])
  })
})

async function callMcpWithBearer(
  page: import('@playwright/test').Page,
  token: string,
  method: string,
  params: Record<string, unknown>,
): Promise<McpFrame[]> {
  const response = await page.evaluate(
    async ({ tokenArg, methodArg, paramsArg }) => {
      const rpcRequest = {
        jsonrpc: '2.0',
        id: `acceptance-token-${Date.now()}-${Math.floor(Math.random() * 1000)}`,
        method: methodArg,
        params: paramsArg,
      }
      const httpResponse = await fetch('/api/v1/mcp/stream', {
        method: 'POST',
        headers: {
          Accept: 'text/event-stream',
          'Content-Type': 'application/json',
          Authorization: `Bearer ${tokenArg}`,
        },
        body: JSON.stringify(rpcRequest),
      })
      return {
        ok: httpResponse.ok,
        status: httpResponse.status,
        body: await httpResponse.text(),
      }
    },
    { tokenArg: token, methodArg: method, paramsArg: params },
  )

  expect(response.ok, `status=${response.status}\n${response.body}`).toBeTruthy()
  const body = response.body
  const frames = parseMcpFrames(body)
  expect(frames.length).toBeGreaterThan(0)
  return frames
}

async function createTokenViaMockApi(
  page: import('@playwright/test').Page,
  scope: 'read' | 'write',
  projectId: string,
): Promise<void> {
  latestTokenScope = scope
  const response = await page.evaluate(
    async ({ scopeArg, projectIdArg }) => {
      const httpResponse = await fetch('/api/v1/mcp/tokens', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          name: `acceptance-${scopeArg}-${Date.now()}`,
          scope: scopeArg,
          allowed_tools: [],
          allowed_project_ids: [projectIdArg],
          expires_in_days: 90,
        }),
      })
      return {
        ok: httpResponse.ok,
        status: httpResponse.status,
        body: await httpResponse.text(),
      }
    },
    { scopeArg: scope, projectIdArg: projectId },
  )

  const responseText = response.body
  expect(response.ok, `status=${response.status}\n${responseText}`).toBeTruthy()
  const payload = JSON.parse(responseText) as { token: string }
  latestBearerToken = payload.token
  expect(latestBearerToken.startsWith('engram_mcp_')).toBeTruthy()
  latestVisibleToolNames = []
  latestWriteAttemptFrames = []
  latestCreatedSessionId = null
  latestCreatedEngramId = null
}

When('I create a read MCP token for project {string}', async ({ page }, projectId: string) => {
  await createTokenViaMockApi(page, 'read', projectId)
})

When('I create a write MCP token for project {string}', async ({ page }, projectId: string) => {
  await createTokenViaMockApi(page, 'write', projectId)
})

When('I call MCP tools list using bearer token only', async ({ page }) => {
  if (!latestBearerToken) {
    throw new Error('Missing bearer token')
  }

  const frames = await callMcpWithBearer(page, latestBearerToken, 'tools/list', {})
  const finalResult = finalResultFrame(frames)
  const tools = finalResult.result?.tools || []
  latestVisibleToolNames = tools.map((tool) => tool.name)
})

Then('only read-safe tools should be visible for the bearer token', async () => {
  expect(latestTokenScope).toBe('read')
  expect(latestVisibleToolNames.length).toBeGreaterThan(0)

  expect(latestVisibleToolNames).toContain('user.get_profile')
  expect(latestVisibleToolNames).toContain('engram.query')
  expect(latestVisibleToolNames).not.toContain('chat.create_session')
  expect(latestVisibleToolNames).not.toContain('chat.send_message')
  expect(latestVisibleToolNames).not.toContain('engram.create')
})

When('I attempt a write MCP tool call for project {string}', async ({ page }, projectId: string) => {
  if (!latestBearerToken) {
    throw new Error('Missing bearer token')
  }

  latestWriteAttemptFrames = await callMcpWithBearer(page, latestBearerToken, 'tools/call', {
    name: 'chat.create_session',
    arguments: {
      project_id: projectId,
      title: `read-token-denied-${Date.now()}`,
      provider: 'openai',
      model_id: 'gpt-4o-mini',
      visibility_scope: 'private',
      autosave_enabled: false,
    },
  })
})

Then('the MCP response should deny the write action for token scope', async () => {
  const errorFrame = firstErrorFrame(latestWriteAttemptFrames)
  expect(errorFrame.error?.code).toBe(-32003)
  expect(errorFrame.error?.data?.required_scope).toBe('write')
  expect(errorFrame.error?.data?.token_scope).toBe('read')
})

When('I execute a bearer write workflow for project {string}', async ({ page }, projectId: string) => {
  if (!latestBearerToken) {
    throw new Error('Missing bearer token')
  }

  const sessionFrames = await callMcpWithBearer(page, latestBearerToken, 'tools/call', {
    name: 'chat.create_session',
    arguments: {
      project_id: projectId,
      title: `write-token-session-${Date.now()}`,
      provider: 'openai',
      model_id: 'gpt-4o-mini',
      visibility_scope: 'private',
      autosave_enabled: false,
    },
  })
  latestCreatedSessionId =
    finalResultFrame(sessionFrames).result?.structuredContent?.session?.session_id || null

  const engramFrames = await callMcpWithBearer(page, latestBearerToken, 'tools/call', {
    name: 'engram.create',
    arguments: {
      project_id: projectId,
      title: `write-token-engram-${Date.now()}`,
      abstract: 'Created from acceptance write-token workflow.',
      detailed_summary_markdown:
        'This engram validates MCP bearer token write scope with project allowlist enforcement.',
      tags: ['acceptance', 'mcp'],
      keywords: ['token', 'write'],
    },
  })
  latestCreatedEngramId =
    finalResultFrame(engramFrames).result?.structuredContent?.engram?.engram_id || null
})

Then('the bearer write workflow should create a session and engram', async () => {
  expect(latestTokenScope).toBe('write')
  expect(latestCreatedSessionId).toBeTruthy()
  expect(latestCreatedEngramId).toBeTruthy()
})
