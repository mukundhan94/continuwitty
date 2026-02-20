import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'

import {
  createMcpToken,
  listAvailableMcpProjects,
  listAvailableMcpTools,
  listMcpTokens,
  revokeMcpToken,
} from './mcpTokens'

const mcpClientMocks = vi.hoisted(() => ({
  streamMcpCall: vi.fn(),
  finalMcpErrorFrame: vi.fn(),
  finalMcpResultFrame: vi.fn(),
}))

vi.mock('./mcpClient', async () => {
  const actual = await vi.importActual<typeof import('./mcpClient')>('./mcpClient')
  return {
    ...actual,
    streamMcpCall: mcpClientMocks.streamMcpCall,
    finalMcpErrorFrame: mcpClientMocks.finalMcpErrorFrame,
    finalMcpResultFrame: mcpClientMocks.finalMcpResultFrame,
  }
})

function mockJsonResponse(body: unknown): Response {
  return {
    ok: true,
    status: 200,
    statusText: 'OK',
    headers: new Headers({ 'content-type': 'application/json' }),
    json: async () => body,
  } as Response
}

function mockMcpResult(result: Record<string, unknown>, frameId: string): void {
  mcpClientMocks.streamMcpCall.mockResolvedValue([{ id: frameId }])
  mcpClientMocks.finalMcpResultFrame.mockReturnValue({
    jsonrpc: '2.0',
    id: frameId,
    result,
  })
}

function expectMcpMethodCalled(method: string): void {
  expect(mcpClientMocks.streamMcpCall).toHaveBeenCalledWith(
    expect.objectContaining({
      method,
    }),
  )
}

beforeEach(() => {
  mcpClientMocks.streamMcpCall.mockReset()
  mcpClientMocks.finalMcpErrorFrame.mockReset()
  mcpClientMocks.finalMcpResultFrame.mockReset()
  mcpClientMocks.finalMcpErrorFrame.mockReturnValue(null)
})

afterEach(() => {
  vi.restoreAllMocks()
})

describe('mcp token api', () => {
  it('lists token summaries', async () => {
    const fetchMock = vi.spyOn(globalThis, 'fetch').mockResolvedValue(
      mockJsonResponse([
        {
          token_id: 'token-1',
          name: 'read token',
          scope: 'read',
          allowed_tools: [],
          allowed_project_ids: [],
          token_secret_hint: 'tok_****',
          expires_at: '2026-03-01T00:00:00Z',
          last_used_at: null,
          revoked_at: null,
          created_at: '2026-02-01T00:00:00Z',
          is_active: true,
        },
      ]),
    )

    const result = await listMcpTokens()

    expect(result).toHaveLength(1)
    expect(result[0].token_id).toBe('token-1')
    expect(fetchMock).toHaveBeenCalledWith('/api/v1/mcp/tokens', expect.any(Object))
  })

  it('creates token with posted payload', async () => {
    const fetchMock = vi.spyOn(globalThis, 'fetch').mockResolvedValue(
      mockJsonResponse({
        token_id: 'token-2',
        name: 'write token',
        scope: 'write',
        allowed_tools: ['engram.create'],
        allowed_project_ids: ['engram-vault'],
        token_secret_hint: 'tok_****',
        token: 'token-secret',
        expires_at: '2026-04-01T00:00:00Z',
        last_used_at: null,
        revoked_at: null,
        created_at: '2026-02-01T00:00:00Z',
        is_active: true,
      }),
    )

    await createMcpToken({
      name: 'write token',
      scope: 'write',
      allowed_tools: ['engram.create'],
      allowed_project_ids: ['engram-vault'],
      expires_in_days: 30,
    })

    const [, init] = fetchMock.mock.calls[0]
    const requestInit = init as RequestInit
    expect(requestInit.method).toBe('POST')
    expect(requestInit.body).toContain('"name":"write token"')
    expect(requestInit.body).toContain('"expires_in_days":30')
  })

  it('revokes token with default reason when none is provided', async () => {
    const fetchMock = vi.spyOn(globalThis, 'fetch').mockResolvedValue(
      mockJsonResponse({
        token_id: 'token-3',
        name: 'rotated token',
        scope: 'read',
        allowed_tools: [],
        allowed_project_ids: [],
        token_secret_hint: 'tok_****',
        expires_at: '2026-05-01T00:00:00Z',
        last_used_at: null,
        revoked_at: '2026-02-20T00:00:00Z',
        created_at: '2026-02-01T00:00:00Z',
        is_active: false,
      }),
    )

    await revokeMcpToken('token-3')

    const [, init] = fetchMock.mock.calls[0]
    const requestInit = init as RequestInit
    expect(requestInit.method).toBe('POST')
    expect(requestInit.body).toBe(JSON.stringify({ reason: 'rotation' }))
  })

  it('lists available tools from MCP stream result and sorts names', async () => {
    mockMcpResult(
      {
        tools: [{ name: 'zeta' }, { name: 'alpha' }, { invalid: true }, { name: 'beta' }],
      },
      'tools-1',
    )

    const tools = await listAvailableMcpTools()

    expect(tools).toEqual(['alpha', 'beta', 'zeta'])
    expectMcpMethodCalled('tools/list')
  })

  it('lists available project ids from MCP stream result and sorts values', async () => {
    mockMcpResult(
      {
        project_ids: ['engram-b', '', 'engram-a', '  ', 1],
      },
      'projects-1',
    )

    const projects = await listAvailableMcpProjects()

    expect(projects).toEqual(['engram-a', 'engram-b'])
    expectMcpMethodCalled('user.list_projects')
  })

  it('throws when MCP stream returns an error frame', async () => {
    mcpClientMocks.streamMcpCall.mockResolvedValue([{ id: 'projects-2' }])
    mcpClientMocks.finalMcpErrorFrame.mockReturnValue({
      jsonrpc: '2.0',
      id: 'projects-2',
      error: {
        code: -32010,
        message: 'forbidden',
      },
    })

    await expect(listAvailableMcpProjects()).rejects.toThrow(
      'MCP user.list_projects failed: forbidden',
    )
  })
})
