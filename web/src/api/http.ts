export class ApiError extends Error {
  status: number
  detail: string

  constructor(status: number, detail: string) {
    super(detail)
    this.status = status
    this.detail = detail
  }
}

async function parseError(response: Response): Promise<ApiError> {
  const fallback = `${response.status} ${response.statusText}`
  const contentType = response.headers.get('content-type') || ''

  if (contentType.includes('application/json')) {
    try {
      const body = (await response.json()) as { detail?: string }
      return new ApiError(response.status, body.detail || fallback)
    } catch {
      return new ApiError(response.status, fallback)
    }
  }

  try {
    const text = (await response.text()).trim()
    return new ApiError(response.status, text || fallback)
  } catch {
    return new ApiError(response.status, fallback)
  }
}

export async function apiJson<T>(path: string, init: RequestInit = {}): Promise<T> {
  const headers = new Headers(init.headers)
  if (!headers.has('Content-Type')) {
    headers.set('Content-Type', 'application/json')
  }

  const response = await fetch(path, {
    ...init,
    credentials: 'include',
    headers,
  })

  if (!response.ok) {
    throw await parseError(response)
  }

  return (await response.json()) as T
}

export async function apiVoid(path: string, init: RequestInit = {}): Promise<void> {
  const response = await fetch(path, {
    credentials: 'include',
    ...init,
  })

  if (!response.ok) {
    throw await parseError(response)
  }
}

export async function parseApiError(response: Response): Promise<ApiError> {
  return parseError(response)
}
