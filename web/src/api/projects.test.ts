import { afterEach, describe, expect, it, vi } from 'vitest'

import {
  addProjectMember,
  getDefaultProject,
  listProjectAuditEvents,
  listProjectMembers,
  listProjects,
  removeProjectMember,
  setDefaultProject,
  updateProjectMember,
} from './projects'

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

    const payload = await setDefaultProject({ project_id: 'project-phase-31' })
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

  it('lists project members', async () => {
    const fetchMock = vi.spyOn(globalThis, 'fetch').mockResolvedValue(
      mockJsonResponse([
        {
          project_id: 'engram-vault',
          user_id: '00000000-0000-0000-0000-000000000101',
          role: 'viewer',
          added_by_user_id: null,
          created_at: '2026-01-01T00:00:00Z',
          updated_at: '2026-01-01T00:00:00Z',
          revoked_at: null,
          revoked_by_user_id: null,
        },
      ]),
    )

    const payload = await listProjectMembers({
      project_id: 'engram-vault',
      params: { include_revoked: false, limit: 250, offset: 2 },
    })
    expect(payload).toHaveLength(1)
    expect(payload[0].role).toBe('viewer')
    expect(String(fetchMock.mock.calls[0][0])).toContain('/api/v1/projects/engram-vault/members?')
    expect(String(fetchMock.mock.calls[0][0])).toContain('limit=250')
    expect(String(fetchMock.mock.calls[0][0])).toContain('offset=2')
  })

  it('posts project member create/update/delete routes', async () => {
    const fetchMock = vi
      .spyOn(globalThis, 'fetch')
      .mockResolvedValue(mockJsonResponse({ removed: true }))
      .mockResolvedValueOnce(
        mockJsonResponse({
          project_id: 'engram-vault',
          user_id: '00000000-0000-0000-0000-000000000101',
          role: 'viewer',
          added_by_user_id: null,
          created_at: '2026-01-01T00:00:00Z',
          updated_at: '2026-01-01T00:00:00Z',
          revoked_at: null,
          revoked_by_user_id: null,
        }),
      )
      .mockResolvedValueOnce(
        mockJsonResponse({
          project_id: 'engram-vault',
          user_id: '00000000-0000-0000-0000-000000000101',
          role: 'editor',
          added_by_user_id: null,
          created_at: '2026-01-01T00:00:00Z',
          updated_at: '2026-01-01T00:00:00Z',
          revoked_at: null,
          revoked_by_user_id: null,
        }),
      )

    await addProjectMember({
      project_id: 'engram-vault',
      payload: {
        user_id: '00000000-0000-0000-0000-000000000101',
        role: 'viewer',
      },
    })
    await updateProjectMember({
      project_id: 'engram-vault',
      user_id: '00000000-0000-0000-0000-000000000101',
      payload: {
        role: 'editor',
      },
    })
    await removeProjectMember({
      project_id: 'engram-vault',
      user_id: '00000000-0000-0000-0000-000000000101',
    })

    expect(String(fetchMock.mock.calls[0][0])).toContain('/api/v1/projects/engram-vault/members')
    expect((fetchMock.mock.calls[0][1] as RequestInit).method).toBe('POST')
    expect((fetchMock.mock.calls[1][1] as RequestInit).method).toBe('PATCH')
    expect(String(fetchMock.mock.calls[1][0])).toContain('/members/00000000-0000-0000-0000-000000000101')
    expect((fetchMock.mock.calls[2][1] as RequestInit).method).toBe('DELETE')
  })

  it('lists project audit events', async () => {
    const fetchMock = vi.spyOn(globalThis, 'fetch').mockResolvedValue(
      mockJsonResponse([
        {
          event_id: '00000000-0000-0000-0000-000000000911',
          project_id: 'engram-vault',
          actor_user_id: '00000000-0000-0000-0000-000000000001',
          event_type: 'engram.share',
          target_type: 'engram',
          target_user_id: null,
          target_engram_id: '00000000-0000-0000-0000-000000000812',
          metadata: {},
          created_at: '2026-01-01T00:00:00Z',
        },
      ]),
    )

    const payload = await listProjectAuditEvents({
      project_id: 'engram-vault',
      params: { limit: 20, offset: 5 },
    })
    expect(payload).toHaveLength(1)
    expect(payload[0].event_type).toBe('engram.share')
    expect(String(fetchMock.mock.calls[0][0])).toContain('/api/v1/projects/engram-vault/audit-events?')
    expect(String(fetchMock.mock.calls[0][0])).toContain('limit=20')
    expect(String(fetchMock.mock.calls[0][0])).toContain('offset=5')
  })
})
