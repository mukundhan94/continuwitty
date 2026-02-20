import { afterEach, describe, expect, it, vi } from 'vitest'

import { ApiError, apiJson, apiVoid, parseApiError } from './http'

function mockResponse(params: {
  ok: boolean
  status: number
  statusText?: string
  contentType?: string
  jsonBody?: unknown
  textBody?: string
  jsonError?: Error
  textError?: Error
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
    json: async () => {
      if (params.jsonError) {
        throw params.jsonError
      }
      return params.jsonBody ?? {}
    },
    text: async () => {
      if (params.textError) {
        throw params.textError
      }
      return params.textBody ?? ''
    },
  } as Response
}

afterEach(() => {
  vi.restoreAllMocks()
})

describe('http api helpers', () => {
  it('apiJson includes credentials and merged headers', async () => {
    const fetchMock = vi.spyOn(globalThis, 'fetch').mockResolvedValue(
      mockResponse({
        ok: true,
        status: 200,
        contentType: 'application/json',
        jsonBody: { ok: true },
      }),
    )

    const result = await apiJson<{ ok: boolean }>('/api/v1/test-json', {
      method: 'POST',
      headers: { 'X-Trace-Id': 'trace-1' },
      body: '{"hello":"world"}',
    })

    expect(result.ok).toBe(true)
    const [, init] = fetchMock.mock.calls[0]
    const requestInit = init as RequestInit
    expect(requestInit.credentials).toBe('include')
    if (requestInit.headers instanceof Headers) {
      expect(requestInit.headers.get('Content-Type')).toBe('application/json')
      expect(requestInit.headers.get('X-Trace-Id')).toBe('trace-1')
    } else {
      const headers = requestInit.headers as Record<string, string>
      expect(headers['Content-Type'] ?? headers['content-type']).toBe('application/json')
      expect(headers['X-Trace-Id'] ?? headers['x-trace-id']).toBe('trace-1')
    }
  })

  it('apiJson throws ApiError with JSON detail payload', async () => {
    vi.spyOn(globalThis, 'fetch').mockResolvedValue(
      mockResponse({
        ok: false,
        status: 401,
        statusText: 'Unauthorized',
        contentType: 'application/json',
        jsonBody: { detail: 'invalid session' },
      }),
    )

    await expect(apiJson('/api/v1/protected')).rejects.toBeInstanceOf(ApiError)
    await expect(apiJson('/api/v1/protected')).rejects.toMatchObject({
      status: 401,
      detail: 'invalid session',
    })
  })

  it('parseApiError falls back to status line when JSON parsing fails', async () => {
    const response = mockResponse({
      ok: false,
      status: 500,
      statusText: 'Server Error',
      contentType: 'application/json',
      jsonError: new Error('invalid-json'),
    })

    const parsed = await parseApiError(response)

    expect(parsed.status).toBe(500)
    expect(parsed.detail).toBe('500 Server Error')
  })

  it('apiVoid throws ApiError using text response detail', async () => {
    vi.spyOn(globalThis, 'fetch').mockResolvedValue(
      mockResponse({
        ok: false,
        status: 403,
        statusText: 'Forbidden',
        contentType: 'text/plain',
        textBody: 'write scope required',
      }),
    )

    await expect(apiVoid('/api/v1/mcp/tokens/token-1/revoke', { method: 'POST' })).rejects.toMatchObject({
      status: 403,
      detail: 'write scope required',
    })
  })
})
