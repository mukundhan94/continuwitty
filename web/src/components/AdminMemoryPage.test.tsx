import { render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { useState } from 'react'
import { ThemeProvider } from 'styled-components'
import { beforeEach, describe, expect, it, vi } from 'vitest'

import type {
  AdminChatSessionRecord,
  AdminEngramRecord,
  EngramCollectionRecord,
  ProjectAuditEventRecord,
  ProjectMemberRecord,
} from '../api/types'
import { lightTheme } from '../styles/theme'
import { AdminMemoryPage } from './AdminMemoryPage'

const memoryAdminMocks = vi.hoisted(() => ({
  listAdminSessions: vi.fn<() => Promise<AdminChatSessionRecord[]>>(async () => []),
  listAdminEngrams: vi.fn<() => Promise<AdminEngramRecord[]>>(async () => []),
  listCollections: vi.fn<() => Promise<EngramCollectionRecord[]>>(async () => []),
  getAdminEngram: vi.fn<() => Promise<AdminEngramRecord>>(async () => buildAdminEngram()),
}))

const projectApiMocks = vi.hoisted(() => ({
  listProjects: vi.fn(async () => [
    {
      project_id: 'engram-vault',
      name: 'Engram Vault',
      description: '',
      owner_user_id: 'owner-1',
      is_archived: false,
      created_at: '2026-01-01T00:00:00Z',
      updated_at: '2026-01-01T00:00:00Z',
    },
    {
      project_id: 'ops-notes',
      name: 'Ops Notes',
      description: '',
      owner_user_id: 'owner-1',
      is_archived: false,
      created_at: '2026-01-01T00:00:00Z',
      updated_at: '2026-01-01T00:00:00Z',
    },
  ]),
  listProjectMembers: vi.fn<() => Promise<ProjectMemberRecord[]>>(async () => []),
  addProjectMember: vi.fn<() => Promise<ProjectMemberRecord>>(async () => ({
    project_id: 'engram-vault',
    user_id: '00000000-0000-0000-0000-000000000191',
    role: 'viewer',
    added_by_user_id: '00000000-0000-0000-0000-000000000001',
    created_at: '2026-01-01T00:00:00Z',
    updated_at: '2026-01-01T00:00:00Z',
    revoked_at: null,
    revoked_by_user_id: null,
  })),
  updateProjectMember: vi.fn<() => Promise<ProjectMemberRecord>>(async () => ({
    project_id: 'engram-vault',
    user_id: '00000000-0000-0000-0000-000000000191',
    role: 'editor',
    added_by_user_id: '00000000-0000-0000-0000-000000000001',
    created_at: '2026-01-01T00:00:00Z',
    updated_at: '2026-01-01T00:00:00Z',
    revoked_at: null,
    revoked_by_user_id: null,
  })),
  removeProjectMember: vi.fn<() => Promise<{ removed: boolean }>>(async () => ({ removed: true })),
  listProjectAuditEvents: vi.fn<() => Promise<ProjectAuditEventRecord[]>>(async () => []),
}))

vi.mock('../api/projects', () => ({
  listProjects: projectApiMocks.listProjects,
  listProjectMembers: projectApiMocks.listProjectMembers,
  addProjectMember: projectApiMocks.addProjectMember,
  updateProjectMember: projectApiMocks.updateProjectMember,
  removeProjectMember: projectApiMocks.removeProjectMember,
  listProjectAuditEvents: projectApiMocks.listProjectAuditEvents,
}))

vi.mock('../api/memoryAdmin', async () => {
  const actual = await vi.importActual<typeof import('../api/memoryAdmin')>('../api/memoryAdmin')
  return {
    ...actual,
    listAdminSessions: memoryAdminMocks.listAdminSessions,
    listAdminEngrams: memoryAdminMocks.listAdminEngrams,
    listCollections: memoryAdminMocks.listCollections,
    getAdminEngram: memoryAdminMocks.getAdminEngram,
  }
})

function buildAdminEngram(overrides: Partial<AdminEngramRecord> = {}): AdminEngramRecord {
  return {
    engram_id: 'engram-1',
    project_id: 'engram-vault',
    thread_id: null,
    title: 'Alpha',
    abstract: 'Alpha abstract',
    detailed_summary_markdown: 'Alpha markdown',
    tags: ['ops'],
    keywords: ['latency'],
    owner_user_id: null,
    visibility_scope: 'project',
    source_session_id: null,
    created_at: '2026-01-01T00:00:00Z',
    updated_at: '2026-01-01T00:00:00Z',
    deleted_at: null,
    deleted_by_user_id: null,
    delete_reason: null,
    sources: [],
    ...overrides,
  }
}

function optionValues(testId: string): string[] {
  const list = screen.getByTestId(testId)
  const options = list.querySelectorAll('option')
  return [...options].map((item) => item.getAttribute('value') || '')
}

function renderAdminPage(overrides: Partial<{ projectId: string }> = {}) {
  const onProjectChange = vi.fn()
  const onNotice = vi.fn()

  function Harness() {
    const [projectId, setProjectId] = useState(overrides.projectId ?? 'engram-vault')
    return (
      <AdminMemoryPage
        projectId={projectId}
        onProjectChange={(nextProjectId) => {
          onProjectChange(nextProjectId)
          setProjectId(nextProjectId)
        }}
        onNotice={onNotice}
      />
    )
  }

  render(
    <ThemeProvider theme={lightTheme}>
      <Harness />
    </ThemeProvider>,
  )

  return {
    onProjectChange,
    onNotice,
  }
}

describe('AdminMemoryPage', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    memoryAdminMocks.getAdminEngram.mockResolvedValue(buildAdminEngram())
    projectApiMocks.listProjectMembers.mockResolvedValue([
      {
        project_id: 'engram-vault',
        user_id: '00000000-0000-0000-0000-000000000001',
        role: 'owner',
        added_by_user_id: '00000000-0000-0000-0000-000000000001',
        created_at: '2026-01-01T00:00:00Z',
        updated_at: '2026-01-01T00:00:00Z',
        revoked_at: null,
        revoked_by_user_id: null,
      },
    ])
    projectApiMocks.listProjectAuditEvents.mockResolvedValue([
      {
        event_id: '00000000-0000-0000-0000-000000000911',
        project_id: 'engram-vault',
        actor_user_id: '00000000-0000-0000-0000-000000000001',
        event_type: 'engram.share',
        target_type: 'engram',
        target_user_id: null,
        target_engram_id: '00000000-0000-0000-0000-000000000812',
        metadata: { from_visibility_scope: 'private', to_visibility_scope: 'project' },
        created_at: '2026-01-01T00:00:00Z',
      },
    ])
  })

  it('renders management sections and loads admin datasets', async () => {
    renderAdminPage()

    expect(screen.getByText('Session Management')).toBeInTheDocument()
    expect(screen.getByText('Engram Management')).toBeInTheDocument()
    expect(screen.getByText('Collections')).toBeInTheDocument()
    expect(screen.getByText('Project Members')).toBeInTheDocument()
    expect(screen.getByText('Project Audit Timeline')).toBeInTheDocument()

    await waitFor(() => {
      expect(memoryAdminMocks.listAdminSessions).toHaveBeenCalled()
      expect(memoryAdminMocks.listAdminEngrams).toHaveBeenCalled()
      expect(memoryAdminMocks.listCollections).toHaveBeenCalled()
      expect(projectApiMocks.listProjects).toHaveBeenCalled()
      expect(projectApiMocks.listProjectMembers).toHaveBeenCalled()
      expect(projectApiMocks.listProjectAuditEvents).toHaveBeenCalled()
    })
  })

  it('shows smart project suggestions in filter and collection project fields', async () => {
    const user = userEvent.setup()
    renderAdminPage()

    await waitFor(() => {
      expect(projectApiMocks.listProjects).toHaveBeenCalled()
    })

    await user.clear(screen.getByTestId('admin-session-project-id-input'))
    await user.type(screen.getByTestId('admin-session-project-id-input'), 'ops')
    expect(optionValues('admin-session-project-id-options')).toEqual(
      expect.arrayContaining(['ops-notes']),
    )

    await user.clear(screen.getByTestId('admin-collection-project-id-input'))
    await user.type(screen.getByTestId('admin-collection-project-id-input'), 'ops')
    expect(optionValues('admin-collection-project-id-options')).toEqual(
      expect.arrayContaining(['ops-notes']),
    )
  })

  it('shows smart project suggestions in move target project field', async () => {
    const user = userEvent.setup()
    memoryAdminMocks.listAdminEngrams.mockResolvedValueOnce([
      buildAdminEngram({
        engram_id: 'engram-42',
        title: 'Investigate latency',
      }),
    ])
    memoryAdminMocks.getAdminEngram.mockResolvedValueOnce(
      buildAdminEngram({
        engram_id: 'engram-42',
        title: 'Investigate latency',
      }),
    )

    renderAdminPage()

    await waitFor(() => {
      expect(screen.getByText('Investigate latency')).toBeInTheDocument()
    })

    await user.click(screen.getByText('Investigate latency'))

    await waitFor(() => {
      expect(screen.getByTestId('admin-move-target-project-id-input')).toBeInTheDocument()
    })

    await user.clear(screen.getByTestId('admin-move-target-project-id-input'))
    await user.type(screen.getByTestId('admin-move-target-project-id-input'), 'ops')
    expect(optionValues('admin-move-target-project-id-options')).toEqual(
      expect.arrayContaining(['ops-notes']),
    )
  })

  it('reloads datasets with include_deleted when toggled', async () => {
    const user = userEvent.setup()
    renderAdminPage()

    await waitFor(() => {
      expect(memoryAdminMocks.listAdminSessions).toHaveBeenCalled()
    })

    await user.click(screen.getByLabelText(/include deleted/i))

    await waitFor(() => {
      expect(memoryAdminMocks.listAdminSessions).toHaveBeenLastCalledWith(
        expect.objectContaining({ include_deleted: true }),
      )
      expect(memoryAdminMocks.listAdminEngrams).toHaveBeenLastCalledWith(
        expect.objectContaining({ include_deleted: true }),
      )
      expect(memoryAdminMocks.listCollections).toHaveBeenLastCalledWith(
        expect.objectContaining({ include_deleted: true }),
      )
    })
  })

  it('applies engram query filter when search is submitted', async () => {
    const user = userEvent.setup()
    renderAdminPage()

    await waitFor(() => {
      expect(memoryAdminMocks.listAdminEngrams).toHaveBeenCalled()
    })

    await user.type(screen.getByPlaceholderText(/search title \/ abstract \/ markdown/i), 'latency')
    await user.click(screen.getByRole('button', { name: /^search$/i }))

    await waitFor(() => {
      expect(memoryAdminMocks.listAdminEngrams).toHaveBeenLastCalledWith(
        expect.objectContaining({ q: 'latency' }),
      )
    })
  })

  it('adds a project member from the members panel', async () => {
    const user = userEvent.setup()
    renderAdminPage()

    await waitFor(() => {
      expect(projectApiMocks.listProjectMembers).toHaveBeenCalled()
    })

    await user.type(screen.getByTestId('admin-member-user-id-input'), '00000000-0000-0000-0000-000000000191')
    await user.selectOptions(screen.getByTestId('admin-member-role-select'), 'viewer')
    await user.click(screen.getByRole('button', { name: /add member/i }))

    await waitFor(() => {
      expect(projectApiMocks.addProjectMember).toHaveBeenCalledWith('engram-vault', {
        user_id: '00000000-0000-0000-0000-000000000191',
        role: 'viewer',
      })
    })
  })
})
