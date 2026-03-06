import { useCallback, useEffect, useMemo, useState } from 'react'
import type { FormEvent } from 'react'

import styled from 'styled-components'

import {
  addProjectMember,
  listProjectAuditEvents,
  listProjectMembers,
  removeProjectMember,
  updateProjectMember,
} from '../api/projects'
import type { ProjectAuditEventRecord, ProjectMemberRecord, ProjectMemberRole, ProjectRecord } from '../api/types'
import { describeError } from '../utils/errors'
import { ErrorText, GlassPane, MutedText, PaneHeader, ScrollColumn, SectionDivider, SessionMeta } from '../styles/primitives'
import { SmartIdDropdown } from './SmartIdDropdown'

const memberRoles: ProjectMemberRole[] = ['owner', 'editor', 'viewer']

const Layout = styled.main`
  display: grid;
  grid-template-columns: minmax(280px, 360px) minmax(0, 1fr);
  gap: 0.9rem;
  flex: 1;
  min-height: 0;

  @media (max-width: 1180px) {
    grid-template-columns: 1fr;
    overflow: auto;
  }
`

const Pane = styled(GlassPane)`
  min-height: 0;
`

const FormStack = styled.form`
  display: grid;
  gap: 0.65rem;
`

const Field = styled.label`
  display: grid;
  gap: 0.28rem;
`

const ActionRow = styled.div`
  display: flex;
  flex-wrap: wrap;
  gap: 0.45rem;
`

const SubtleButton = styled.button`
  background: var(--surface-mute);
  color: var(--color-ink);
  border: 1px solid var(--color-line);
  box-shadow: none;

  &:hover {
    box-shadow: none;
  }
`

const RecordCard = styled.article`
  display: grid;
  gap: 0.4rem;
  border-radius: 16px;
  border: 1px solid var(--color-line);
  background: var(--surface-raised);
  padding: 0.85rem;

  h3,
  p {
    margin: 0;
  }
`

const RoleRow = styled.div`
  display: grid;
  grid-template-columns: minmax(0, 1fr) auto auto;
  gap: 0.45rem;
  align-items: end;

  @media (max-width: 680px) {
    grid-template-columns: 1fr;
  }
`

const MetadataBlock = styled.pre`
  margin: 0;
  border-radius: 12px;
  border: 1px dashed var(--color-line);
  background: var(--surface-mute);
  padding: 0.65rem;
  overflow: auto;
  font-size: 0.75rem;
  color: var(--color-ink-muted);
`

const StatusPill = styled.span<{ $tone?: 'muted' | 'success' }>`
  display: inline-flex;
  align-items: center;
  width: fit-content;
  border-radius: 9999px;
  padding: 0.2rem 0.6rem;
  border: 1px solid ${({ $tone }) => ($tone === 'success' ? 'var(--color-notice-border)' : 'var(--color-line)')};
  background: ${({ $tone }) => ($tone === 'success' ? 'var(--color-notice-bg)' : 'var(--surface-mute)')};
  color: ${({ $tone }) => ($tone === 'success' ? 'var(--color-success)' : 'var(--color-ink-muted)')};
  font-size: 0.72rem;
  text-transform: uppercase;
  letter-spacing: 0.08em;
  font-weight: 700;
`

function formatWhen(value: string | null): string {
  if (!value) {
    return 'Active'
  }
  return new Date(value).toLocaleString()
}

function formatAuditTarget(event: ProjectAuditEventRecord): string {
  if (event.target_engram_id) {
    return `${event.target_type} ${event.target_engram_id}`
  }
  if (event.target_user_id) {
    return `${event.target_type} ${event.target_user_id}`
  }
  return event.target_type
}

interface ProjectGovernancePageProps {
  action: 'members' | 'audit'
  projectId: string
  project: ProjectRecord | null
  projectSuggestions: string[]
  onProjectChange: (projectId: string) => void
  onNotice: (message: string) => void
}

export function ProjectGovernancePage({
  action,
  projectId,
  project,
  projectSuggestions,
  onProjectChange,
  onNotice,
}: ProjectGovernancePageProps) {
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [members, setMembers] = useState<ProjectMemberRecord[]>([])
  const [auditEvents, setAuditEvents] = useState<ProjectAuditEventRecord[]>([])
  const [creatingMember, setCreatingMember] = useState(false)
  const [memberError, setMemberError] = useState<string | null>(null)
  const [newMemberUserId, setNewMemberUserId] = useState('')
  const [newMemberRole, setNewMemberRole] = useState<ProjectMemberRole>('viewer')
  const [draftRoles, setDraftRoles] = useState<Record<string, ProjectMemberRole>>({})
  const [savingMemberId, setSavingMemberId] = useState<string | null>(null)
  const [removingMemberId, setRemovingMemberId] = useState<string | null>(null)

  const loadProjectSurface = useCallback(async () => {
    if (!projectId.trim()) {
      setMembers([])
      setAuditEvents([])
      setDraftRoles({})
      setLoading(false)
      return
    }

    setLoading(true)
    setError(null)
    try {
      if (action === 'members') {
        const records = await listProjectMembers({
          project_id: projectId,
          params: { include_revoked: true, limit: 200, offset: 0 },
        })
        setMembers(records)
        setDraftRoles(
          records.reduce<Record<string, ProjectMemberRole>>((accumulator, record) => {
            accumulator[record.user_id] = record.role
            return accumulator
          }, {}),
        )
        setAuditEvents([])
      } else {
        const records = await listProjectAuditEvents({
          project_id: projectId,
          params: { limit: 100, offset: 0 },
        })
        setAuditEvents(records)
        setMembers([])
      }
    } catch (loadError) {
      setError(describeError(loadError))
      setMembers([])
      setAuditEvents([])
    } finally {
      setLoading(false)
    }
  }, [action, projectId])

  useEffect(() => {
    void loadProjectSurface()
  }, [loadProjectSurface])

  const memberOptions = useMemo(() => members.map((member) => member.user_id), [members])

  const handleAddMember = async (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault()
    setCreatingMember(true)
    setMemberError(null)
    try {
      const created = await addProjectMember({
        project_id: projectId,
        payload: {
          user_id: newMemberUserId.trim(),
          role: newMemberRole,
        },
      })
      setMembers((current) => [created, ...current.filter((record) => record.user_id !== created.user_id)])
      setDraftRoles((current) => ({ ...current, [created.user_id]: created.role }))
      setNewMemberUserId('')
      onNotice(`Added ${created.user_id} to ${projectId}.`)
    } catch (createError) {
      setMemberError(describeError(createError))
    } finally {
      setCreatingMember(false)
    }
  }

  const handleUpdateMember = async (member: ProjectMemberRecord) => {
    setSavingMemberId(member.user_id)
    setMemberError(null)
    try {
      const updated = await updateProjectMember({
        project_id: projectId,
        user_id: member.user_id,
        payload: { role: draftRoles[member.user_id] ?? member.role },
      })
      setMembers((current) => current.map((record) => (record.user_id === updated.user_id ? updated : record)))
      setDraftRoles((current) => ({ ...current, [updated.user_id]: updated.role }))
      onNotice(`Updated ${updated.user_id} in ${projectId}.`)
    } catch (updateError) {
      setMemberError(describeError(updateError))
    } finally {
      setSavingMemberId(null)
    }
  }

  const handleRemoveMember = async (member: ProjectMemberRecord) => {
    setRemovingMemberId(member.user_id)
    setMemberError(null)
    try {
      await removeProjectMember({ project_id: projectId, user_id: member.user_id })
      setMembers((current) =>
        current.map((record) =>
          record.user_id === member.user_id
            ? {
                ...record,
                revoked_at: new Date().toISOString(),
              }
            : record,
        ),
      )
      onNotice(`Removed ${member.user_id} from ${projectId}.`)
    } catch (removeError) {
      setMemberError(describeError(removeError))
    } finally {
      setRemovingMemberId(null)
    }
  }

  return (
    <Layout data-testid="project-governance-page">
      <Pane>
        <PaneHeader>
          <div>
            <h2 className="font-display text-base font-semibold tracking-[0.02em] text-ink">
              {action === 'members' ? 'Project Members' : 'Project Audit'}
            </h2>
            <MutedText>
              {action === 'members'
                ? 'Manage explicit project access with focused member operations.'
                : 'Review governance, collaboration, and memory events without chat noise.'}
            </MutedText>
          </div>
          <StatusPill $tone="success">Dedicated Route</StatusPill>
        </PaneHeader>

        <Field>
          Project ID
          <SmartIdDropdown
            value={projectId}
            options={[projectId, ...projectSuggestions]}
            onChange={onProjectChange}
            inputTestId="project-governance-project-id"
            optionsTestId="project-governance-project-options"
            matchCountTestId="project-governance-project-matches"
          />
        </Field>

        <SectionDivider>
          <h3 className="font-display text-lg font-semibold text-ink">{project?.name || projectId || 'Project overview'}</h3>
          <MutedText>{project?.description || 'Use this view to keep governance tasks separate from active chat work.'}</MutedText>
          <SessionMeta>
            Owner {project?.owner_user_id || 'unknown'} · Updated {project ? new Date(project.updated_at).toLocaleString() : 'n/a'}
          </SessionMeta>
        </SectionDivider>

        {action === 'members' ? (
          <FormStack onSubmit={handleAddMember}>
            <Field>
              User ID
              <SmartIdDropdown
                value={newMemberUserId}
                options={memberOptions}
                onChange={setNewMemberUserId}
                placeholder="user-123"
                inputTestId="project-member-user-id"
                optionsTestId="project-member-user-options"
                matchCountTestId="project-member-user-matches"
              />
            </Field>
            <Field>
              Role
              <select
                data-testid="project-member-role"
                value={newMemberRole}
                onChange={(event) => setNewMemberRole(event.target.value as ProjectMemberRole)}
              >
                {memberRoles.map((role) => (
                  <option key={role} value={role}>{role}</option>
                ))}
              </select>
            </Field>
            {memberError ? <ErrorText>{memberError}</ErrorText> : null}
            <ActionRow>
              <button type="submit" disabled={creatingMember || !projectId.trim() || !newMemberUserId.trim()}>
                {creatingMember ? 'Adding…' : 'Add Member'}
              </button>
              <SubtleButton type="button" onClick={() => void loadProjectSurface()} disabled={loading}>
                Refresh Members
              </SubtleButton>
            </ActionRow>
          </FormStack>
        ) : (
          <ActionRow>
            <SubtleButton type="button" onClick={() => void loadProjectSurface()} disabled={loading}>
              Refresh Audit
            </SubtleButton>
          </ActionRow>
        )}
      </Pane>

      <Pane>
        <PaneHeader>
          <div>
            <h2 className="font-display text-base font-semibold tracking-[0.02em] text-ink">
              {action === 'members' ? 'Access Directory' : 'Audit Timeline'}
            </h2>
            <MutedText>
              {action === 'members'
                ? 'Update member roles or revoke access from a dedicated operational view.'
                : 'Audit events remain isolated so operators can review them with full context.'}
            </MutedText>
          </div>
        </PaneHeader>

        {loading ? <MutedText>Loading project {action}…</MutedText> : null}
        {error ? <ErrorText>{error}</ErrorText> : null}

        <ScrollColumn>
          {!loading && !error && action === 'members' && members.length === 0 ? (
            <MutedText>No members found for this project.</MutedText>
          ) : null}
          {!loading && !error && action === 'audit' && auditEvents.length === 0 ? (
            <MutedText>No audit events recorded for this project yet.</MutedText>
          ) : null}

          {action === 'members'
            ? members.map((member) => (
                <RecordCard key={member.user_id} data-testid={`project-member-${member.user_id}`}>
                  <div>
                    <h3>{member.user_id}</h3>
                    <SessionMeta>
                      Added {new Date(member.created_at).toLocaleString()} · Last updated {new Date(member.updated_at).toLocaleString()}
                    </SessionMeta>
                  </div>
                  <StatusPill>{member.revoked_at ? 'Revoked' : 'Active'}</StatusPill>
                  <SessionMeta>
                    Added by {member.added_by_user_id || 'system'} · Status {formatWhen(member.revoked_at)}
                  </SessionMeta>
                  <RoleRow>
                    <Field>
                      Role
                      <select
                        data-testid={`project-member-role-${member.user_id}`}
                        value={draftRoles[member.user_id] ?? member.role}
                        onChange={(event) =>
                          setDraftRoles((current) => ({
                            ...current,
                            [member.user_id]: event.target.value as ProjectMemberRole,
                          }))
                        }
                      >
                        {memberRoles.map((role) => (
                          <option key={role} value={role}>{role}</option>
                        ))}
                      </select>
                    </Field>
                    <SubtleButton
                      type="button"
                      onClick={() => void handleUpdateMember(member)}
                      disabled={savingMemberId === member.user_id}
                    >
                      {savingMemberId === member.user_id ? 'Saving…' : 'Save Role'}
                    </SubtleButton>
                    <SubtleButton
                      type="button"
                      onClick={() => void handleRemoveMember(member)}
                      disabled={Boolean(member.revoked_at) || removingMemberId === member.user_id}
                    >
                      {removingMemberId === member.user_id ? 'Removing…' : 'Remove'}
                    </SubtleButton>
                  </RoleRow>
                </RecordCard>
              ))
            : auditEvents.map((event) => (
                <RecordCard key={event.event_id} data-testid={`project-audit-${event.event_id}`}>
                  <div>
                    <h3>{event.event_type}</h3>
                    <SessionMeta>{new Date(event.created_at).toLocaleString()}</SessionMeta>
                  </div>
                  <SessionMeta>
                    Actor {event.actor_user_id || 'system'} · Target {formatAuditTarget(event)}
                  </SessionMeta>
                  <SessionMeta>Event scope {event.target_type}</SessionMeta>
                  <MetadataBlock>{JSON.stringify(event.metadata, null, 2)}</MetadataBlock>
                </RecordCard>
              ))}
        </ScrollColumn>
      </Pane>
    </Layout>
  )
}
