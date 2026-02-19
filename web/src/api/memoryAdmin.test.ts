import { afterEach, describe, expect, it, vi } from 'vitest'

import { listAdminEngrams, listAdminSessions, moveAdminEngram } from './memoryAdmin'

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

describe('memory admin api', () => {
  it('builds session list query params', async () => {
    const fetchMock = vi.spyOn(globalThis, 'fetch').mockResolvedValue(mockJsonResponse([]))

    await listAdminSessions({
      project_id: 'engram-vault',
      include_deleted: true,
      limit: 25,
      offset: 10,
    })

    const [path] = fetchMock.mock.calls[0]
    const resolved = String(path)
    expect(resolved).toContain('/api/v1/admin/memory/sessions?')
    expect(resolved).toContain('project_id=engram-vault')
    expect(resolved).toContain('include_deleted=true')
    expect(resolved).toContain('limit=25')
    expect(resolved).toContain('offset=10')
  })

  it('builds engram list query params', async () => {
    const fetchMock = vi.spyOn(globalThis, 'fetch').mockResolvedValue(mockJsonResponse([]))

    await listAdminEngrams({
      project_id: 'engram-vault',
      q: 'incident',
      include_deleted: false,
    })

    const [path] = fetchMock.mock.calls[0]
    const resolved = String(path)
    expect(resolved).toContain('/api/v1/admin/memory/engrams?')
    expect(resolved).toContain('project_id=engram-vault')
    expect(resolved).toContain('q=incident')
  })

  it('posts engram move payload', async () => {
    const fetchMock = vi.spyOn(globalThis, 'fetch').mockResolvedValue(
      mockJsonResponse({
        engram_id: 'id-1',
        project_id: 'target-project',
      }),
    )

    await moveAdminEngram('id-1', {
      target_project_id: 'target-project',
      reason: 'test move',
    })

    const [, init] = fetchMock.mock.calls[0]
    expect((init as RequestInit).method).toBe('POST')
    expect((init as RequestInit).body).toContain('target-project')
  })
})
