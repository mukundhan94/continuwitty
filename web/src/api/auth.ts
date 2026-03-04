import { ApiError, apiJson, parseApiError } from './http'
import type { UserProfile } from './types'

interface SessionCsrfResponse {
  csrf_token: string
}

async function fetchCsrfToken(): Promise<string> {
  const payload = await apiJson<SessionCsrfResponse>('/api/v1/session/csrf', { method: 'GET' })
  const token = payload.csrf_token?.trim()
  if (!token) {
    throw new ApiError(500, 'Missing csrf token in session response')
  }
  return token
}

export async function getSessionProfile(): Promise<UserProfile> {
  return apiJson<UserProfile>('/api/v1/me')
}

export async function loginWithPassword(username: string, password: string): Promise<void> {
  const csrfToken = await fetchCsrfToken()

  const response = await fetch('/api/v1/session/login', {
    credentials: 'include',
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
    },
    body: JSON.stringify({
      username,
      password,
      csrf_token: csrfToken,
    }),
  })

  if (response.ok) {
    return
  }

  throw await parseApiError(response)
}

export async function logoutCurrentUser(): Promise<void> {
  const csrfToken = await fetchCsrfToken()

  const response = await fetch('/api/v1/session/logout', {
    credentials: 'include',
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
    },
    body: JSON.stringify({ csrf_token: csrfToken }),
  })

  if (response.ok) {
    return
  }

  throw await parseApiError(response)
}
