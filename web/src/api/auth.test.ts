import { afterEach, describe, expect, it, vi } from 'vitest'

import { ApiError } from './http'
import { extractCsrfTokenFromHtml, loginWithPassword, logoutCurrentUser } from './auth'

function mockResponse(params: {
  ok: boolean
  status: number
  type?: ResponseType
  text?: string
  json?: unknown
}): Response {
  return {
    ok: params.ok,
    status: params.status,
    statusText: '',
    type: params.type ?? 'basic',
    headers: new Headers({
      'content-type': 'text/html',
    }),
    text: async () => params.text ?? '',
    json: async () => params.json ?? {},
  } as Response
}

afterEach(() => {
  vi.restoreAllMocks()
})

describe('extractCsrfTokenFromHtml', () => {
  it('extracts csrf token from login markup', () => {
    const html = '<form><input type="hidden" name="csrf_token" value="abc123" /></form>'
    expect(extractCsrfTokenFromHtml(html)).toBe('abc123')
  })

  it('throws when token is missing', () => {
    expect(() => extractCsrfTokenFromHtml('<html></html>')).toThrow(ApiError)
  })
})

describe('auth redirect handling', () => {
  it('treats manual redirect opaque login response as success', async () => {
    const fetchMock = vi
      .spyOn(globalThis, 'fetch')
      .mockResolvedValueOnce(
        mockResponse({
          ok: true,
          status: 200,
          text: '<form><input name="csrf_token" value="token-1" /></form>',
        }),
      )
      .mockResolvedValueOnce(
        mockResponse({
          ok: false,
          status: 0,
          type: 'opaqueredirect',
        }),
      )

    await expect(loginWithPassword('admin', 'admin123')).resolves.toBeUndefined()
    expect(fetchMock).toHaveBeenCalledTimes(2)
  })

  it('treats manual redirect opaque logout response as success', async () => {
    const fetchMock = vi
      .spyOn(globalThis, 'fetch')
      .mockResolvedValueOnce(
        mockResponse({
          ok: true,
          status: 200,
          text: '<form><input name="csrf_token" value="token-2" /></form>',
        }),
      )
      .mockResolvedValueOnce(
        mockResponse({
          ok: false,
          status: 0,
          type: 'opaqueredirect',
        }),
      )

    await expect(logoutCurrentUser()).resolves.toBeUndefined()
    expect(fetchMock).toHaveBeenCalledTimes(2)
  })

  it('still throws for non-redirect login failures', async () => {
    vi.spyOn(globalThis, 'fetch')
      .mockResolvedValueOnce(
        mockResponse({
          ok: true,
          status: 200,
          text: '<form><input name="csrf_token" value="token-3" /></form>',
        }),
      )
      .mockResolvedValueOnce(
        mockResponse({
          ok: false,
          status: 401,
          text: 'Invalid username or password.',
        }),
      )

    await expect(loginWithPassword('admin', 'wrong')).rejects.toThrow(ApiError)
  })
})
