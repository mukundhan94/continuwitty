import { ApiError, apiJson, parseApiError } from './http'
import type { UserProfile } from './types'

export function extractCsrfTokenFromHtml(html: string): string {
  const match = html.match(/name="csrf_token" value="([^"]+)"/)
  if (!match || !match[1]) {
    throw new ApiError(500, 'Unable to extract CSRF token from HTML response')
  }
  return match[1]
}

async function fetchCsrfToken(pagePath: '/login' | '/ui'): Promise<string> {
  const response = await fetch(pagePath, {
    credentials: 'include',
    method: 'GET',
  })
  if (!response.ok) {
    throw await parseApiError(response)
  }
  return extractCsrfTokenFromHtml(await response.text())
}

export async function getSessionProfile(): Promise<UserProfile> {
  return apiJson<UserProfile>('/api/v1/me')
}

export async function loginWithPassword(username: string, password: string): Promise<void> {
  const csrfToken = await fetchCsrfToken('/login')
  const body = new URLSearchParams({
    username,
    password,
    csrf_token: csrfToken,
  })

  const response = await fetch('/login', {
    credentials: 'include',
    method: 'POST',
    headers: {
      'Content-Type': 'application/x-www-form-urlencoded',
    },
    body: body.toString(),
    redirect: 'manual',
  })

  if (response.status === 303) {
    return
  }

  throw await parseApiError(response)
}

export async function logoutCurrentUser(): Promise<void> {
  const csrfToken = await fetchCsrfToken('/ui')
  const body = new URLSearchParams({ csrf_token: csrfToken })

  const response = await fetch('/logout', {
    credentials: 'include',
    method: 'POST',
    headers: {
      'Content-Type': 'application/x-www-form-urlencoded',
    },
    body: body.toString(),
    redirect: 'manual',
  })

  if (response.status === 303) {
    return
  }

  throw await parseApiError(response)
}
