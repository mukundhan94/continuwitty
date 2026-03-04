import { afterEach, describe, expect, it, vi } from 'vitest'

import { ApiError } from './http'
import { loginWithPassword, logoutCurrentUser } from './auth'

function mockResponse(params: {
  ok: boolean
  status: number
  type?: ResponseType
  text?: string
  json?: unknown
  contentType?: string
}): Response {
  return {
    ok: params.ok,
    status: params.status,
    statusText: '',
    type: params.type ?? 'basic',
    headers: new Headers({
      'content-type': params.contentType ?? 'application/json',
    }),
    text: async () => params.text ?? '',
    json: async () => params.json ?? {},
  } as Response
}

afterEach(() => {
  vi.restoreAllMocks()
})

describe('session auth APIs', () => {
  it('logs in using session csrf + json login endpoints', async () => {
    const fetchMock = vi
      .spyOn(globalThis, 'fetch')
      .mockResolvedValueOnce(
        mockResponse({
          ok: true,
          status: 200,
          json: { csrf_token: 'token-1' },
        }),
      )
      .mockResolvedValueOnce(
        mockResponse({
          ok: true,
          status: 200,
          json: { username: 'admin' },
        }),
      )

    await expect(loginWithPassword('admin', 'admin123')).resolves.toBeUndefined()
    expect(fetchMock).toHaveBeenCalledTimes(2)
    expect(fetchMock).toHaveBeenNthCalledWith(
      1,
      '/api/v1/session/csrf',
      expect.objectContaining({ method: 'GET', credentials: 'include' }),
    )
    expect(fetchMock).toHaveBeenNthCalledWith(
      2,
      '/api/v1/session/login',
      expect.objectContaining({
        method: 'POST',
        credentials: 'include',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          username: 'admin',
          password: 'admin123',
          csrf_token: 'token-1',
        }),
      }),
    )
  })

  it('logs out using session csrf + json logout endpoints', async () => {
    const fetchMock = vi
      .spyOn(globalThis, 'fetch')
      .mockResolvedValueOnce(
        mockResponse({
          ok: true,
          status: 200,
          json: { csrf_token: 'token-2' },
        }),
      )
      .mockResolvedValueOnce(
        mockResponse({
          ok: true,
          status: 200,
          json: { logged_out: true },
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
          json: { csrf_token: 'token-3' },
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

  it('throws when csrf endpoint does not return a token', async () => {
    vi.spyOn(globalThis, 'fetch').mockResolvedValueOnce(
      mockResponse({
        ok: true,
        status: 200,
        json: {},
      }),
    )

    await expect(loginWithPassword('admin', 'admin123')).rejects.toThrow(ApiError)
  })
})
