import { afterEach, describe, expect, it, vi } from 'vitest'

import { getDefaultProject, listProjects, setDefaultProject } from './projects'

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

describe('projects api', () => {
  it('loads default project id', async () => {
    const fetchMock = vi.spyOn(globalThis, 'fetch').mockResolvedValue(
      mockJsonResponse({ default_project_id: 'engram-vault' }),
    )

    const payload = await getDefaultProject()
    expect(payload.default_project_id).toBe('engram-vault')
    expect(fetchMock).toHaveBeenCalledWith('/api/v1/projects/default', expect.any(Object))
  })

  it('patches default project id', async () => {
    const fetchMock = vi.spyOn(globalThis, 'fetch').mockResolvedValue(
      mockJsonResponse({ default_project_id: 'project-phase-31' }),
    )

    const payload = await setDefaultProject('project-phase-31')
    expect(payload.default_project_id).toBe('project-phase-31')
    const [, init] = fetchMock.mock.calls[0]
    expect((init as RequestInit).method).toBe('PATCH')
  })

  it('lists projects', async () => {
    const fetchMock = vi.spyOn(globalThis, 'fetch').mockResolvedValue(
      mockJsonResponse([
        {
          project_id: 'engram-vault',
          name: 'engram-vault',
          description: '',
          owner_user_id: 'owner',
          is_archived: false,
          created_at: '2026-02-01T00:00:00Z',
          updated_at: '2026-02-01T00:00:00Z',
        },
      ]),
    )

    const payload = await listProjects()
    expect(payload).toHaveLength(1)
    expect(payload[0].project_id).toBe('engram-vault')
    expect(String(fetchMock.mock.calls[0][0])).toContain('/api/v1/projects?')
  })
})
