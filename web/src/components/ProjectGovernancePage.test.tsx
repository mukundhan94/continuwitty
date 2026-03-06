import { render, screen, waitFor, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { ThemeProvider } from 'styled-components'
import { beforeEach, describe, expect, it, vi } from 'vitest'

import type { ProjectAuditEventRecord, ProjectMemberRecord, ProjectRecord } from '../api/types'
import { lightTheme } from '../styles/theme'
import { ProjectGovernancePage } from './ProjectGovernancePage'

const projectApiMocks = vi.hoisted(() => ({
  listProjectMembers: vi.fn(),
  addProjectMember: vi.fn(),
  updateProjectMember: vi.fn(),
  removeProjectMember: vi.fn(),
  listProjectAuditEvents: vi.fn(),
}))

vi.mock('../api/projects', () => ({
  listProjectMembers: projectApiMocks.listProjectMembers,
  addProjectMember: projectApiMocks.addProjectMember,
  updateProjectMember: projectApiMocks.updateProjectMember,
  removeProjectMember: projectApiMocks.removeProjectMember,
  listProjectAuditEvents: projectApiMocks.listProjectAuditEvents,
}))

const project: ProjectRecord = {
  project_id: 'engram-vault',
  name: 'Engram Vault',
  description: 'Primary memory workspace',
  owner_user_id: 'owner-1',
  membership_role: 'owner',
  is_archived: false,
  created_at: '2026-03-01T10:00:00Z',
  updated_at: '2026-03-06T10:00:00Z',
}

const memberRecord: ProjectMemberRecord = {
  project_id: 'engram-vault',
  user_id: 'user-1',
  role: 'editor',
  added_by_user_id: 'owner-1',
  created_at: '2026-03-01T10:00:00Z',
  updated_at: '2026-03-06T10:00:00Z',
  revoked_at: null,
  revoked_by_user_id: null,
}

const auditRecord: ProjectAuditEventRecord = {
  event_id: 'audit-1',
  project_id: 'engram-vault',
  actor_user_id: 'owner-1',
  event_type: 'member_added',
  target_type: 'user',
  target_user_id: 'user-1',
  target_engram_id: null,
  metadata: { role: 'editor' },
  created_at: '2026-03-06T10:05:00Z',
}

function renderGovernancePage(action: 'members' | 'audit') {
  const onProjectChange = vi.fn()
  const onNotice = vi.fn()

  render(
    <ThemeProvider theme={lightTheme}>
      <ProjectGovernancePage
        action={action}
        projectId="engram-vault"
        project={project}
        projectSuggestions={['engram-vault', 'ops-notes']}
        onProjectChange={onProjectChange}
        onNotice={onNotice}
      />
    </ThemeProvider>,
  )

  return { onProjectChange, onNotice }
}

describe('ProjectGovernancePage', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    projectApiMocks.listProjectMembers.mockResolvedValue([memberRecord])
    projectApiMocks.addProjectMember.mockResolvedValue({ ...memberRecord, user_id: 'user-2', role: 'viewer' })
    projectApiMocks.updateProjectMember.mockResolvedValue({ ...memberRecord, role: 'viewer' })
    projectApiMocks.removeProjectMember.mockResolvedValue({ removed: true })
    projectApiMocks.listProjectAuditEvents.mockResolvedValue([auditRecord])
  })

  it('manages project members from a dedicated page', async () => {
    const user = userEvent.setup()
    const { onNotice } = renderGovernancePage('members')

    expect(await screen.findByTestId('project-member-user-1')).toBeInTheDocument()

    await user.type(screen.getByTestId('project-member-user-id'), 'user-2')
    await user.selectOptions(screen.getByTestId('project-member-role'), 'viewer')
    await user.click(screen.getByRole('button', { name: 'Add Member' }))

    await waitFor(() => {
      expect(projectApiMocks.addProjectMember).toHaveBeenCalledWith({
        project_id: 'engram-vault',
        payload: {
          user_id: 'user-2',
          role: 'viewer',
        },
      })
    })

    const memberCard = screen.getByTestId('project-member-user-1')
    await user.selectOptions(screen.getByTestId('project-member-role-user-1'), 'viewer')
    await user.click(within(memberCard).getByRole('button', { name: 'Save Role' }))

    await waitFor(() => {
      expect(projectApiMocks.updateProjectMember).toHaveBeenCalledWith({
        project_id: 'engram-vault',
        user_id: 'user-1',
        payload: { role: 'viewer' },
      })
    })

    await user.click(within(memberCard).getByRole('button', { name: 'Remove' }))

    await waitFor(() => {
      expect(projectApiMocks.removeProjectMember).toHaveBeenCalledWith({
        project_id: 'engram-vault',
        user_id: 'user-1',
      })
    })

    expect(onNotice).toHaveBeenCalled()
  })

  it('renders project audit events on a dedicated page', async () => {
    renderGovernancePage('audit')

    expect(await screen.findByTestId('project-audit-audit-1')).toBeInTheDocument()
    expect(screen.getByText('member_added')).toBeInTheDocument()
    expect(screen.getByText(/"role": "editor"/)).toBeInTheDocument()
  })
})
