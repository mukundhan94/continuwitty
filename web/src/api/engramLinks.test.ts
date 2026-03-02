import { afterEach, describe, expect, it, vi } from 'vitest'

import { createEngramLink, listEngramLinks, suggestEngramLinks } from './engramLinks'

function mockJsonResponse(body: unknown): Response {
  return {
    ok: true,
    status: 200,
    statusText: 'OK',
    headers: new Headers({ 'content-type': 'application/json' }),
    json: async () => body,
  } as Response
}

afterEach(() => {
  vi.restoreAllMocks()
})

describe('engram link api', () => {
  it('builds list link query params', async () => {
    const fetchMock = vi.spyOn(globalThis, 'fetch').mockResolvedValue(mockJsonResponse([]))

    await listEngramLinks('engram-1', {
      relation_type: 'supports',
      include_archived: true,
      limit: 20,
      offset: 5,
    })

    const [path] = fetchMock.mock.calls[0]
    const resolvedPath = String(path)
    expect(resolvedPath).toContain('/api/v1/engrams/engram-1/links?')
    expect(resolvedPath).toContain('relation_type=supports')
    expect(resolvedPath).toContain('include_archived=true')
    expect(resolvedPath).toContain('limit=20')
    expect(resolvedPath).toContain('offset=5')
  })

  it('posts create link payload', async () => {
    const fetchMock = vi.spyOn(globalThis, 'fetch').mockResolvedValue(
      mockJsonResponse({ link_id: 'link-1' }),
    )

    await createEngramLink('engram-1', {
      target_engram_id: 'engram-2',
      relation_type: 'related_to',
      weight: 0.7,
      origin: 'suggested',
      status: 'active',
    })

    const [, init] = fetchMock.mock.calls[0]
    const requestInit = init as RequestInit
    expect(requestInit.method).toBe('POST')
    expect(requestInit.body).toContain('"target_engram_id":"engram-2"')
    expect(requestInit.body).toContain('"status":"active"')
  })

  it('posts suggest payload with tuning controls', async () => {
    const fetchMock = vi.spyOn(globalThis, 'fetch').mockResolvedValue(mockJsonResponse([]))

    await suggestEngramLinks('engram-1', {
      limit: 4,
      max_candidates: 12,
      minimum_score: 0.3,
      include_archived: false,
    })

    const [path, init] = fetchMock.mock.calls[0]
    expect(String(path)).toContain('/api/v1/engrams/engram-1/links/suggest')
    expect((init as RequestInit).method).toBe('POST')
    expect((init as RequestInit).body).toContain('"minimum_score":0.3')
  })
})
