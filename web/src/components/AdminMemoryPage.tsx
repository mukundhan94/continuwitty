import { useCallback, useEffect, useMemo, useState } from 'react'
import type { ChangeEvent } from 'react'

import styled from 'styled-components'

import {
  addCollectionItems,
  createCollection,
  deleteAdminEngram,
  deleteAdminSession,
  deleteCollection,
  getAdminEngram,
  listAdminEngrams,
  listAdminSessions,
  listCollections,
  moveAdminEngram,
  removeCollectionItem,
  restoreAdminEngram,
  restoreAdminSession,
  updateAdminEngram,
  updateCollection,
} from '../api/memoryAdmin'
import {
  addProjectMember,
  listProjectAuditEvents,
  listProjectMembers,
  listProjects,
  removeProjectMember,
  updateProjectMember,
} from '../api/projects'
import type {
  AdminChatSessionRecord,
  AdminEngramRecord,
  AdminEngramSourceInput,
  EngramCollectionRecord,
  ProjectAuditEventRecord,
  ProjectMemberRecord,
  ProjectMemberRole,
} from '../api/types'
import { ErrorText, GlassPane, MutedText, PaneHeader, ScrollColumn, SectionDivider } from '../styles/primitives'
import { SmartIdDropdown } from './SmartIdDropdown'

const AdminLayout = styled.main`
  display: grid;
  grid-template-columns: minmax(300px, 0.9fr) minmax(360px, 1.3fr) minmax(300px, 0.9fr);
  gap: 0.9rem;
  flex: 1;
  min-height: 0;

  @media (max-width: 1280px) {
    grid-template-columns: 1fr;
    overflow: auto;
  }
`

const Toolbar = styled.div`
  display: flex;
  align-items: center;
  gap: 0.55rem;
  flex-wrap: wrap;

  button,
  input {
    min-height: 2rem;
  }
`

const TableLike = styled.div`
  display: grid;
  gap: 0.45rem;
`

const RowCard = styled.div<{ $active?: boolean }>`
  border: 1px solid ${({ $active }) => ($active ? 'var(--session-active-border)' : 'var(--color-line)')};
  background: ${({ $active }) => ($active ? 'var(--session-active-bg)' : 'var(--surface-raised)')};
  border-radius: 11px;
  padding: 0.5rem 0.6rem;
  display: grid;
  gap: 0.3rem;
`

const RowActions = styled.div`
  display: flex;
  gap: 0.4rem;
  flex-wrap: wrap;
`

const Field = styled.label`
  display: grid;
  gap: 0.3rem;
`

const SourceEditor = styled.div`
  border: 1px dashed var(--color-line);
  border-radius: 10px;
  padding: 0.55rem;
  display: grid;
  gap: 0.4rem;
`

const SESSION_LIMIT = 200
const ENGRAM_LIMIT = 300
const COLLECTION_LIMIT = 200
const PROJECT_MEMBER_LIMIT = 500
const PROJECT_AUDIT_LIMIT = 200
const projectMemberRoles: ProjectMemberRole[] = ['owner', 'editor', 'viewer']
type TextInputChangeEvent = ChangeEvent<HTMLInputElement>
type TextAreaChangeEvent = ChangeEvent<HTMLTextAreaElement>
type EngramActionLabel = 'Restored' | 'Deleted'

function toIsoNow() {
  return new Date().toISOString()
}

function parseCommaSeparated(input: { value: string }): string[] {
  return input.value
    .split(',')
    .map((item) => item.trim())
    .filter((item) => item.length > 0)
}

function sortedUniqueProjectIds(values: string[]): string[] {
  const unique = new Set<string>()
  for (const rawValue of values) {
    const normalized = rawValue.trim()
    if (!normalized) {
      continue
    }
    unique.add(normalized)
  }
  return Array.from(unique).sort((left, right) => left.localeCompare(right))
}

function buildSourceDraft(): AdminEngramSourceInput {
  return {
    captured_at: toIsoNow(),
    url: '',
    title: '',
    snippet: '',
    content_text: '',
    content_hash: '',
  }
}

function withTimestampDraft(sourceDraft: AdminEngramSourceInput): AdminEngramSourceInput {
  return {
    ...sourceDraft,
    captured_at: sourceDraft.captured_at || toIsoNow(),
  }
}

interface AdminMemoryPageProps {
  projectId: string
  onProjectChange: (projectId: string) => void
  onNotice: (message: string) => void
}

interface SessionManagementSectionProps {
  projectId: string
  projectIdOptions: string[]
  includeDeleted: boolean
  deleteLinkedEngrams: boolean
  loading: boolean
  submitting: boolean
  sessions: AdminChatSessionRecord[]
  onProjectChange: (projectId: string) => void
  onIncludeDeletedChange: (nextValue: boolean) => void
  onDeleteLinkedEngramsChange: (nextValue: boolean) => void
  onRefresh: () => void
  onRestoreSession: (session: AdminChatSessionRecord) => void
  onDeleteSession: (session: AdminChatSessionRecord) => void
}

interface EngramDetailEditorProps {
  selectedEngram: AdminEngramRecord
  submitting: boolean
  editTitle: string
  editAbstract: string
  editMarkdown: string
  editTags: string
  editKeywords: string
  projectIdOptions: string[]
  moveTargetProject: string
  sourceDraft: AdminEngramSourceInput
  sourceRows: AdminEngramSourceInput[]
  onEditTitleChange: (event: TextInputChangeEvent) => void
  onEditAbstractChange: (event: TextAreaChangeEvent) => void
  onEditMarkdownChange: (event: TextAreaChangeEvent) => void
  onEditTagsChange: (event: TextInputChangeEvent) => void
  onEditKeywordsChange: (event: TextInputChangeEvent) => void
  onMoveTargetProjectChange: (value: string) => void
  onSourceUrlChange: (event: TextInputChangeEvent) => void
  onSourceTitleChange: (event: TextInputChangeEvent) => void
  onSourceSnippetChange: (event: TextAreaChangeEvent) => void
  onAddSource: () => void
  onClearSources: () => void
  onSave: () => void
  onMove: () => void
  onRestore: () => void
  onDelete: () => void
}

interface EngramManagementSectionProps {
  queryText: string
  loading: boolean
  submitting: boolean
  engrams: AdminEngramRecord[]
  selectedEngramId: string | null
  selectedEngram: AdminEngramRecord | null
  editTitle: string
  editAbstract: string
  editMarkdown: string
  editTags: string
  editKeywords: string
  projectIdOptions: string[]
  moveTargetProject: string
  sourceDraft: AdminEngramSourceInput
  sourceRows: AdminEngramSourceInput[]
  onQueryTextChange: (event: TextInputChangeEvent) => void
  onRefresh: () => void
  onSelectEngram: (engram: AdminEngramRecord) => void
  onEditTitleChange: (event: TextInputChangeEvent) => void
  onEditAbstractChange: (event: TextAreaChangeEvent) => void
  onEditMarkdownChange: (event: TextAreaChangeEvent) => void
  onEditTagsChange: (event: TextInputChangeEvent) => void
  onEditKeywordsChange: (event: TextInputChangeEvent) => void
  onMoveTargetProjectChange: (value: string) => void
  onSourceUrlChange: (event: TextInputChangeEvent) => void
  onSourceTitleChange: (event: TextInputChangeEvent) => void
  onSourceSnippetChange: (event: TextAreaChangeEvent) => void
  onAddSource: () => void
  onClearSources: () => void
  onSaveEngram: () => void
  onMoveEngram: () => void
  onRestoreEngram: () => void
  onDeleteEngram: () => void
}

interface CollectionSectionProps {
  collections: EngramCollectionRecord[]
  selectedCollectionId: string | null
  selectedCollection: EngramCollectionRecord | null
  projectIdOptions: string[]
  collectionProjectId: string
  collectionName: string
  collectionDescription: string
  collectionAddEngramId: string
  submitting: boolean
  onCollectionProjectIdChange: (value: string) => void
  onCollectionNameChange: (event: TextInputChangeEvent) => void
  onCollectionDescriptionChange: (event: TextAreaChangeEvent) => void
  onCollectionAddEngramIdChange: (event: TextInputChangeEvent) => void
  onSelectCollection: (collection: EngramCollectionRecord) => void
  onCreateCollection: () => void
  onAddItem: () => void
  onRemoveItem: () => void
  onQuickRenameCollection: (collection: EngramCollectionRecord) => void
  onDeleteCollection: (collection: EngramCollectionRecord) => void
}

interface CollectionListProps {
  collections: EngramCollectionRecord[]
  selectedCollectionId: string | null
  submitting: boolean
  onSelectCollection: (collection: EngramCollectionRecord) => void
  onQuickRenameCollection: (collection: EngramCollectionRecord) => void
  onDeleteCollection: (collection: EngramCollectionRecord) => void
}

interface ProjectMembersSectionProps {
  projectId: string
  projectIdOptions: string[]
  submitting: boolean
  members: ProjectMemberRecord[]
  memberUserId: string
  memberRole: ProjectMemberRole
  memberRoleDrafts: Record<string, ProjectMemberRole>
  onProjectChange: (projectId: string) => void
  onMemberUserIdChange: (event: TextInputChangeEvent) => void
  onMemberRoleChange: (event: ChangeEvent<HTMLSelectElement>) => void
  onMemberRoleDraftChange: (userId: string, role: ProjectMemberRole) => void
  onAddMember: () => void
  onUpdateMember: (member: ProjectMemberRecord) => void
  onRemoveMember: (member: ProjectMemberRecord) => void
}

interface ProjectAuditSectionProps {
  projectId: string
  projectIdOptions: string[]
  events: ProjectAuditEventRecord[]
  loading: boolean
  onProjectChange: (projectId: string) => void
  onRefresh: () => void
}

function SessionManagementSection({
  projectId,
  projectIdOptions,
  includeDeleted,
  deleteLinkedEngrams,
  loading,
  submitting,
  sessions,
  onProjectChange,
  onIncludeDeletedChange,
  onDeleteLinkedEngramsChange,
  onRefresh,
  onRestoreSession,
  onDeleteSession,
}: SessionManagementSectionProps) {
  return (
    <GlassPane>
      <PaneHeader>
        <h2 className="font-display text-base font-semibold tracking-[0.02em] text-ink">Session Management</h2>
      </PaneHeader>
      <Toolbar>
        <SmartIdDropdown
          id="admin-session-project-id-field"
          value={projectId}
          options={projectIdOptions}
          onChange={onProjectChange}
          placeholder="project-id filter"
          inputTestId="admin-session-project-id-input"
          optionsTestId="admin-session-project-id-options"
          showMatchCount={false}
        />
        <label>
          <input
            type="checkbox"
            checked={includeDeleted}
            onChange={(event) => onIncludeDeletedChange(event.target.checked)}
          />
          Include deleted
        </label>
        <button type="button" disabled={loading || submitting} onClick={onRefresh}>
          Refresh
        </button>
      </Toolbar>
      <label>
        <input
          type="checkbox"
          checked={deleteLinkedEngrams}
          onChange={(event) => onDeleteLinkedEngramsChange(event.target.checked)}
        />
        Delete linked engrams when deleting a session
      </label>
      <SectionDivider />
      <ScrollColumn>
        <TableLike>
          {sessions.map((session) => (
            <RowCard key={session.session_id}>
              <div className="font-display text-sm font-semibold text-ink">{session.title}</div>
              <MutedText>
                {session.project_id} · {session.provider}/{session.model_id}
              </MutedText>
              <MutedText>{session.session_id}</MutedText>
              {session.deleted_at ? <MutedText>Deleted: {session.deleted_at}</MutedText> : null}
              <RowActions>
                {session.deleted_at ? (
                  <button type="button" disabled={submitting} onClick={() => onRestoreSession(session)}>
                    Restore
                  </button>
                ) : (
                  <button type="button" disabled={submitting} onClick={() => onDeleteSession(session)}>
                    Delete
                  </button>
                )}
              </RowActions>
            </RowCard>
          ))}
          {sessions.length === 0 ? <MutedText>No sessions found for selected filters.</MutedText> : null}
        </TableLike>
      </ScrollColumn>
    </GlassPane>
  )
}

function EngramDetailEditor({
  selectedEngram,
  submitting,
  editTitle,
  editAbstract,
  editMarkdown,
  editTags,
  editKeywords,
  projectIdOptions,
  moveTargetProject,
  sourceDraft,
  sourceRows,
  onEditTitleChange,
  onEditAbstractChange,
  onEditMarkdownChange,
  onEditTagsChange,
  onEditKeywordsChange,
  onMoveTargetProjectChange,
  onSourceUrlChange,
  onSourceTitleChange,
  onSourceSnippetChange,
  onAddSource,
  onClearSources,
  onSave,
  onMove,
  onRestore,
  onDelete,
}: EngramDetailEditorProps) {
  return (
    <>
      <SectionDivider />
      <Field>
        <span>Title</span>
        <input value={editTitle} onChange={onEditTitleChange} />
      </Field>
      <Field>
        <span>Abstract</span>
        <textarea rows={3} value={editAbstract} onChange={onEditAbstractChange} />
      </Field>
      <Field>
        <span>Markdown</span>
        <textarea rows={6} value={editMarkdown} onChange={onEditMarkdownChange} />
      </Field>
      <Field>
        <span>Tags (comma-separated)</span>
        <input value={editTags} onChange={onEditTagsChange} />
      </Field>
      <Field>
        <span>Keywords (comma-separated)</span>
        <input value={editKeywords} onChange={onEditKeywordsChange} />
      </Field>
      <Field>
        <span>Move target project</span>
        <SmartIdDropdown
          id="admin-move-target-project-id-field"
          value={moveTargetProject}
          options={projectIdOptions}
          onChange={onMoveTargetProjectChange}
          placeholder="target project id"
          inputTestId="admin-move-target-project-id-input"
          optionsTestId="admin-move-target-project-id-options"
          showMatchCount={false}
        />
      </Field>
      <SourceEditor>
        <strong className="font-display text-sm text-ink">Sources</strong>
        {sourceRows.map((source, index) => (
          <MutedText key={`${source.url}-${index}`}>{source.title || source.url}</MutedText>
        ))}
        <input
          value={sourceDraft.url}
          placeholder="source url"
          onChange={onSourceUrlChange}
        />
        <input
          value={sourceDraft.title}
          placeholder="source title"
          onChange={onSourceTitleChange}
        />
        <textarea
          rows={2}
          value={sourceDraft.snippet}
          placeholder="source snippet"
          onChange={onSourceSnippetChange}
        />
        <RowActions>
          <button type="button" onClick={onAddSource}>
            Add Source
          </button>
          <button type="button" onClick={onClearSources}>
            Clear Sources
          </button>
        </RowActions>
      </SourceEditor>
      <RowActions>
        <button type="button" disabled={submitting} onClick={onSave}>
          Save
        </button>
        <button type="button" disabled={submitting || !moveTargetProject.trim()} onClick={onMove}>
          Move
        </button>
        {selectedEngram.deleted_at ? (
          <button type="button" disabled={submitting} onClick={onRestore}>
            Restore
          </button>
        ) : (
          <button type="button" disabled={submitting} onClick={onDelete}>
            Delete
          </button>
        )}
      </RowActions>
    </>
  )
}

function EngramManagementSection({
  queryText,
  loading,
  submitting,
  engrams,
  selectedEngramId,
  selectedEngram,
  editTitle,
  editAbstract,
  editMarkdown,
  editTags,
  editKeywords,
  projectIdOptions,
  moveTargetProject,
  sourceDraft,
  sourceRows,
  onQueryTextChange,
  onRefresh,
  onSelectEngram,
  onEditTitleChange,
  onEditAbstractChange,
  onEditMarkdownChange,
  onEditTagsChange,
  onEditKeywordsChange,
  onMoveTargetProjectChange,
  onSourceUrlChange,
  onSourceTitleChange,
  onSourceSnippetChange,
  onAddSource,
  onClearSources,
  onSaveEngram,
  onMoveEngram,
  onRestoreEngram,
  onDeleteEngram,
}: EngramManagementSectionProps) {
  return (
    <GlassPane>
      <PaneHeader>
        <h2 className="font-display text-base font-semibold tracking-[0.02em] text-ink">Engram Management</h2>
      </PaneHeader>
      <Toolbar>
        <input
          value={queryText}
          onChange={onQueryTextChange}
          placeholder="search title / abstract / markdown"
        />
        <button type="button" disabled={loading || submitting} onClick={onRefresh}>
          Search
        </button>
      </Toolbar>
      <SectionDivider />
      <ScrollColumn>
        <TableLike>
          {engrams.map((engram) => (
            <RowCard
              key={engram.engram_id}
              $active={engram.engram_id === selectedEngramId}
              onClick={() => onSelectEngram(engram)}
            >
              <div className="font-display text-sm font-semibold text-ink">{engram.title}</div>
              <MutedText>{engram.project_id}</MutedText>
              <MutedText>{engram.engram_id}</MutedText>
              {engram.deleted_at ? <MutedText>Deleted: {engram.deleted_at}</MutedText> : null}
            </RowCard>
          ))}
          {engrams.length === 0 ? <MutedText>No engrams found for selected filters.</MutedText> : null}
        </TableLike>
      </ScrollColumn>

      {selectedEngram ? (
        <EngramDetailEditor
          selectedEngram={selectedEngram}
          submitting={submitting}
          editTitle={editTitle}
          editAbstract={editAbstract}
          editMarkdown={editMarkdown}
          editTags={editTags}
          editKeywords={editKeywords}
          projectIdOptions={projectIdOptions}
          moveTargetProject={moveTargetProject}
          sourceDraft={sourceDraft}
          sourceRows={sourceRows}
          onEditTitleChange={onEditTitleChange}
          onEditAbstractChange={onEditAbstractChange}
          onEditMarkdownChange={onEditMarkdownChange}
          onEditTagsChange={onEditTagsChange}
          onEditKeywordsChange={onEditKeywordsChange}
          onMoveTargetProjectChange={onMoveTargetProjectChange}
          onSourceUrlChange={onSourceUrlChange}
          onSourceTitleChange={onSourceTitleChange}
          onSourceSnippetChange={onSourceSnippetChange}
          onAddSource={onAddSource}
          onClearSources={onClearSources}
          onSave={onSaveEngram}
          onMove={onMoveEngram}
          onRestore={onRestoreEngram}
          onDelete={onDeleteEngram}
        />
      ) : (
        <MutedText>Select an engram to edit metadata, move projects, or soft-delete/restore.</MutedText>
      )}
    </GlassPane>
  )
}

function CollectionList({
  collections,
  selectedCollectionId,
  submitting,
  onSelectCollection,
  onQuickRenameCollection,
  onDeleteCollection,
}: CollectionListProps) {
  return (
    <ScrollColumn>
      <TableLike>
        {collections.map((collection) => (
          <RowCard
            key={collection.collection_id}
            $active={collection.collection_id === selectedCollectionId}
            onClick={() => onSelectCollection(collection)}
          >
            <div className="font-display text-sm font-semibold text-ink">{collection.name}</div>
            <MutedText>
              {collection.project_id} · {collection.collection_id}
            </MutedText>
            <RowActions>
              <button
                type="button"
                disabled={submitting}
                onClick={() => onQuickRenameCollection(collection)}
              >
                Quick Rename
              </button>
              {collection.deleted_at ? (
                <MutedText>Deleted</MutedText>
              ) : (
                <button
                  type="button"
                  disabled={submitting}
                  onClick={() => onDeleteCollection(collection)}
                >
                  Delete
                </button>
              )}
            </RowActions>
          </RowCard>
        ))}
        {collections.length === 0 ? <MutedText>No collections found for selected filters.</MutedText> : null}
      </TableLike>
    </ScrollColumn>
  )
}

function CollectionSection({
  collections,
  selectedCollectionId,
  selectedCollection,
  projectIdOptions,
  collectionProjectId,
  collectionName,
  collectionDescription,
  collectionAddEngramId,
  submitting,
  onCollectionProjectIdChange,
  onCollectionNameChange,
  onCollectionDescriptionChange,
  onCollectionAddEngramIdChange,
  onSelectCollection,
  onCreateCollection,
  onAddItem,
  onRemoveItem,
  onQuickRenameCollection,
  onDeleteCollection,
}: CollectionSectionProps) {
  return (
    <GlassPane>
      <PaneHeader>
        <h2 className="font-display text-base font-semibold tracking-[0.02em] text-ink">Collections</h2>
      </PaneHeader>
      <Field>
        <span>Collection project</span>
        <SmartIdDropdown
          id="admin-collection-project-id-field"
          value={collectionProjectId}
          options={projectIdOptions}
          onChange={onCollectionProjectIdChange}
          placeholder="project id"
          inputTestId="admin-collection-project-id-input"
          optionsTestId="admin-collection-project-id-options"
          showMatchCount={false}
        />
      </Field>
      <Field>
        <span>Name</span>
        <input value={collectionName} onChange={onCollectionNameChange} />
      </Field>
      <Field>
        <span>Description</span>
        <textarea
          rows={2}
          value={collectionDescription}
          onChange={onCollectionDescriptionChange}
        />
      </Field>
      <button
        type="button"
        disabled={submitting || !collectionProjectId.trim() || !collectionName.trim()}
        onClick={onCreateCollection}
      >
        Create Collection
      </button>
      <SectionDivider />
      <Field>
        <span>Add engram to selected collection</span>
        <input
          value={collectionAddEngramId}
          onChange={onCollectionAddEngramIdChange}
          placeholder="engram id"
        />
      </Field>
      <RowActions>
        <button
          type="button"
          disabled={submitting || !selectedCollection || !collectionAddEngramId.trim()}
          onClick={onAddItem}
        >
          Add Item
        </button>
        <button
          type="button"
          disabled={submitting || !selectedCollection || !collectionAddEngramId.trim()}
          onClick={onRemoveItem}
        >
          Remove Item
        </button>
      </RowActions>
      <CollectionList
        collections={collections}
        selectedCollectionId={selectedCollectionId}
        submitting={submitting}
        onSelectCollection={onSelectCollection}
        onQuickRenameCollection={onQuickRenameCollection}
        onDeleteCollection={onDeleteCollection}
      />
    </GlassPane>
  )
}

function renderAuditEventTarget(event: ProjectAuditEventRecord): string {
  if (event.target_user_id) {
    return event.target_user_id
  }
  if (event.target_engram_id) {
    return event.target_engram_id
  }
  return event.target_type
}

function ProjectMembersSection({
  projectId,
  projectIdOptions,
  submitting,
  members,
  memberUserId,
  memberRole,
  memberRoleDrafts,
  onProjectChange,
  onMemberUserIdChange,
  onMemberRoleChange,
  onMemberRoleDraftChange,
  onAddMember,
  onUpdateMember,
  onRemoveMember,
}: ProjectMembersSectionProps) {
  return (
    <GlassPane>
      <PaneHeader>
        <h2 className="font-display text-base font-semibold tracking-[0.02em] text-ink">Project Members</h2>
      </PaneHeader>
      <Field>
        <span>Project</span>
        <SmartIdDropdown
          id="admin-member-project-id-field"
          value={projectId}
          options={projectIdOptions}
          onChange={onProjectChange}
          placeholder="project id"
          inputTestId="admin-member-project-id-input"
          optionsTestId="admin-member-project-id-options"
          showMatchCount={false}
        />
      </Field>
      <RowActions>
        <input
          value={memberUserId}
          onChange={onMemberUserIdChange}
          placeholder="user id"
          data-testid="admin-member-user-id-input"
        />
        <select value={memberRole} onChange={onMemberRoleChange} data-testid="admin-member-role-select">
          {projectMemberRoles.map((role) => (
            <option key={role} value={role}>
              {role}
            </option>
          ))}
        </select>
        <button
          type="button"
          disabled={submitting || !projectId.trim() || !memberUserId.trim()}
          onClick={onAddMember}
        >
          Add member
        </button>
      </RowActions>
      <SectionDivider />
      <ScrollColumn>
        <TableLike>
          {members.map((member) => {
            const draftRole = memberRoleDrafts[member.user_id] ?? member.role
            const isOwnerRow = member.role === 'owner'
            return (
              <RowCard key={member.user_id}>
                <div className="font-display text-sm font-semibold text-ink">{member.user_id}</div>
                <MutedText>{member.role}</MutedText>
                <RowActions>
                  <select
                    value={draftRole}
                    disabled={submitting || isOwnerRow}
                    onChange={(event) =>
                      onMemberRoleDraftChange(member.user_id, event.target.value as ProjectMemberRole)
                    }
                  >
                    {projectMemberRoles.map((role) => (
                      <option key={role} value={role}>
                        {role}
                      </option>
                    ))}
                  </select>
                  <button type="button" disabled={submitting || isOwnerRow} onClick={() => onUpdateMember(member)}>
                    Update role
                  </button>
                  <button type="button" disabled={submitting || isOwnerRow} onClick={() => onRemoveMember(member)}>
                    Remove
                  </button>
                </RowActions>
              </RowCard>
            )
          })}
          {members.length === 0 ? <MutedText>No members found for selected project.</MutedText> : null}
        </TableLike>
      </ScrollColumn>
    </GlassPane>
  )
}

function ProjectAuditSection({
  projectId,
  projectIdOptions,
  events,
  loading,
  onProjectChange,
  onRefresh,
}: ProjectAuditSectionProps) {
  return (
    <GlassPane>
      <PaneHeader>
        <h2 className="font-display text-base font-semibold tracking-[0.02em] text-ink">Project Audit Timeline</h2>
      </PaneHeader>
      <Toolbar>
        <SmartIdDropdown
          id="admin-audit-project-id-field"
          value={projectId}
          options={projectIdOptions}
          onChange={onProjectChange}
          placeholder="project id"
          inputTestId="admin-audit-project-id-input"
          optionsTestId="admin-audit-project-id-options"
          showMatchCount={false}
        />
        <button type="button" disabled={loading} onClick={onRefresh}>
          Refresh
        </button>
      </Toolbar>
      <SectionDivider />
      <ScrollColumn>
        <TableLike>
          {events.map((event) => (
            <RowCard key={event.event_id}>
              <div className="font-display text-sm font-semibold text-ink">{event.event_type}</div>
              <MutedText>{event.created_at}</MutedText>
              <MutedText>Target: {renderAuditEventTarget(event)}</MutedText>
              {event.actor_user_id ? <MutedText>Actor: {event.actor_user_id}</MutedText> : null}
              {Object.keys(event.metadata).length > 0 ? (
                <MutedText>{JSON.stringify(event.metadata)}</MutedText>
              ) : null}
            </RowCard>
          ))}
          {events.length === 0 ? <MutedText>No audit events for selected project.</MutedText> : null}
        </TableLike>
      </ScrollColumn>
    </GlassPane>
  )
}

export function AdminMemoryPage({ projectId, onProjectChange, onNotice }: AdminMemoryPageProps) {
  const [includeDeleted, setIncludeDeleted] = useState(false)
  const [queryText, setQueryText] = useState('')
  const [sessions, setSessions] = useState<AdminChatSessionRecord[]>([])
  const [engrams, setEngrams] = useState<AdminEngramRecord[]>([])
  const [collections, setCollections] = useState<EngramCollectionRecord[]>([])
  const [members, setMembers] = useState<ProjectMemberRecord[]>([])
  const [auditEvents, setAuditEvents] = useState<ProjectAuditEventRecord[]>([])
  const [selectedEngramId, setSelectedEngramId] = useState<string | null>(null)
  const [selectedEngram, setSelectedEngram] = useState<AdminEngramRecord | null>(null)
  const [editTitle, setEditTitle] = useState('')
  const [editAbstract, setEditAbstract] = useState('')
  const [editMarkdown, setEditMarkdown] = useState('')
  const [editTags, setEditTags] = useState('')
  const [editKeywords, setEditKeywords] = useState('')
  const [moveTargetProject, setMoveTargetProject] = useState('')
  const [sourceDraft, setSourceDraft] = useState<AdminEngramSourceInput>(buildSourceDraft)
  const [sourceRows, setSourceRows] = useState<AdminEngramSourceInput[]>([])
  const [collectionProjectId, setCollectionProjectId] = useState(projectId)
  const [projectSuggestions, setProjectSuggestions] = useState<string[]>([])
  const [collectionName, setCollectionName] = useState('')
  const [collectionDescription, setCollectionDescription] = useState('')
  const [collectionSelectedId, setCollectionSelectedId] = useState<string | null>(null)
  const [collectionAddEngramId, setCollectionAddEngramId] = useState('')
  const [memberUserId, setMemberUserId] = useState('')
  const [memberRole, setMemberRole] = useState<ProjectMemberRole>('viewer')
  const [memberRoleDrafts, setMemberRoleDrafts] = useState<Record<string, ProjectMemberRole>>({})
  const [deleteLinkedEngrams, setDeleteLinkedEngrams] = useState(false)
  const [loading, setLoading] = useState(false)
  const [submitting, setSubmitting] = useState(false)
  const [error, setError] = useState<string | null>(null)

  const selectedCollection = useMemo(
    () => collections.find((item) => item.collection_id === collectionSelectedId) ?? null,
    [collectionSelectedId, collections],
  )

  useEffect(() => {
    const drafts: Record<string, ProjectMemberRole> = {}
    for (const member of members) {
      drafts[member.user_id] = member.role
    }
    setMemberRoleDrafts(drafts)
  }, [members])

  useEffect(() => {
    setCollectionProjectId((current) => (current.trim() ? current : projectId))
  }, [projectId])

  useEffect(() => {
    let cancelled = false

    const loadProjectSuggestions = async () => {
      try {
        const projects = await listProjects(true)
        if (cancelled) {
          return
        }
        setProjectSuggestions(projects.map((project) => project.project_id))
      } catch {
        if (!cancelled) {
          setProjectSuggestions([])
        }
      }
    }

    void loadProjectSuggestions()
    return () => {
      cancelled = true
    }
  }, [])

  const projectIdOptions = useMemo(
    () =>
      sortedUniqueProjectIds([
        projectId,
        collectionProjectId,
        moveTargetProject,
        ...projectSuggestions,
        ...sessions.map((session) => session.project_id),
        ...engrams.map((engram) => engram.project_id),
        ...collections.map((collection) => collection.project_id),
      ]),
    [
      collectionProjectId,
      collections,
      engrams,
      moveTargetProject,
      projectId,
      projectSuggestions,
      sessions,
    ],
  )

  const refresh = useCallback(async () => {
    setLoading(true)
    setError(null)
    try {
      const projectFilter = projectId.trim() || undefined
      const [nextSessions, nextEngrams, nextCollections, nextMembers, nextAuditEvents] = await Promise.all([
        listAdminSessions({
          project_id: projectFilter,
          include_deleted: includeDeleted,
          limit: SESSION_LIMIT,
          offset: 0,
        }),
        listAdminEngrams({
          project_id: projectFilter,
          q: queryText.trim() || undefined,
          include_deleted: includeDeleted,
          limit: ENGRAM_LIMIT,
          offset: 0,
        }),
        listCollections({
          project_id: projectFilter,
          include_deleted: includeDeleted,
          limit: COLLECTION_LIMIT,
          offset: 0,
        }),
        projectFilter
          ? listProjectMembers({
              project_id: projectFilter,
              params: {
                include_revoked: false,
                limit: PROJECT_MEMBER_LIMIT,
                offset: 0,
              },
            })
          : Promise.resolve<ProjectMemberRecord[]>([]),
        projectFilter
          ? listProjectAuditEvents({
              project_id: projectFilter,
              params: {
                limit: PROJECT_AUDIT_LIMIT,
                offset: 0,
              },
            })
          : Promise.resolve<ProjectAuditEventRecord[]>([]),
      ])
      setSessions(nextSessions)
      setEngrams(nextEngrams)
      setCollections(nextCollections)
      setMembers(nextMembers)
      setAuditEvents(nextAuditEvents)
    } catch (refreshError) {
      setError(refreshError instanceof Error ? refreshError.message : 'Failed to load admin memory data.')
    } finally {
      setLoading(false)
    }
  }, [includeDeleted, projectId, queryText])

  useEffect(() => {
    void refresh()
  }, [refresh])

  const selectEngram = useCallback(async (input: { engramId: string }) => {
    const { engramId } = input
    setSelectedEngramId(engramId)
    setError(null)
    try {
      const detail = await getAdminEngram(engramId, true)
      setSelectedEngram(detail)
      setEditTitle(detail.title)
      setEditAbstract(detail.abstract)
      setEditMarkdown(detail.detailed_summary_markdown)
      setEditTags(detail.tags.join(', '))
      setEditKeywords(detail.keywords.join(', '))
      setSourceRows(detail.sources)
      setMoveTargetProject(detail.project_id)
    } catch (detailError) {
      setError(detailError instanceof Error ? detailError.message : 'Failed to load engram detail.')
    }
  }, [])

  const runAction = useCallback(async (action: () => Promise<void>) => {
    setSubmitting(true)
    setError(null)
    try {
      await action()
    } catch (actionError) {
      setError(actionError instanceof Error ? actionError.message : 'Admin action failed.')
    } finally {
      setSubmitting(false)
    }
  }, [])

  const updateSourceDraftField = (input: { field: keyof AdminEngramSourceInput; value: string }) => {
    const { field, value } = input
    setSourceDraft((current) => ({
      ...current,
      [field]: value,
    }))
  }

  const handleSourceUrlChange = (event: TextInputChangeEvent) => {
    updateSourceDraftField({
      field: 'url',
      value: event.target.value,
    })
  }

  const handleSourceTitleChange = (event: TextInputChangeEvent) => {
    updateSourceDraftField({
      field: 'title',
      value: event.target.value,
    })
  }

  const handleSourceSnippetChange = (event: TextAreaChangeEvent) => {
    updateSourceDraftField({
      field: 'snippet',
      value: event.target.value,
    })
  }

  const addSourceRow = () => {
    setSourceRows((current) => [...current, withTimestampDraft(sourceDraft)])
  }

  const clearSourceRows = () => {
    setSourceRows([])
  }

  const saveSelectedEngram = () => {
    if (!selectedEngram) {
      return
    }
    void runAction(async () => {
      const updated = await updateAdminEngram(selectedEngram.engram_id, {
        title: editTitle.trim(),
        abstract: editAbstract.trim(),
        detailed_summary_markdown: editMarkdown,
        tags: parseCommaSeparated({ value: editTags }),
        keywords: parseCommaSeparated({ value: editKeywords }),
        expected_updated_at: selectedEngram.updated_at,
        sources: sourceRows,
      })
      setSelectedEngram(updated)
      onNotice(`Updated engram ${updated.engram_id}`)
      await refresh()
    })
  }

  const moveSelectedEngram = () => {
    if (!selectedEngram || !moveTargetProject.trim()) {
      return
    }
    void runAction(async () => {
      const moved = await moveAdminEngram(selectedEngram.engram_id, {
        target_project_id: moveTargetProject.trim(),
        expected_updated_at: selectedEngram.updated_at,
        reason: 'admin-ui-move',
      })
      setSelectedEngram(moved)
      onNotice(`Moved engram ${moved.engram_id} to ${moved.project_id}`)
      await refresh()
    })
  }

  const runSelectedEngramLifecycleAction = (
    action: (input: { engramId: string }) => Promise<unknown>,
    noticePrefix: EngramActionLabel,
  ) => {
    if (!selectedEngram) {
      return
    }
    const engramId = selectedEngram.engram_id
    void runAction(async () => {
      await action({ engramId })
      onNotice(`${noticePrefix} engram ${engramId}`)
      await selectEngram({ engramId })
      await refresh()
    })
  }

  const restoreSelectedEngram = () => {
    runSelectedEngramLifecycleAction(
      async ({ engramId }) => {
        await restoreAdminEngram(engramId)
      },
      'Restored',
    )
  }

  const deleteSelectedEngram = () => {
    runSelectedEngramLifecycleAction(
      async ({ engramId }) => {
        await deleteAdminEngram(engramId, { reason: 'admin-ui-delete' })
      },
      'Deleted',
    )
  }

  const createCollectionRecord = () => {
    if (!collectionProjectId.trim() || !collectionName.trim()) {
      return
    }
    void runAction(async () => {
      const created = await createCollection({
        project_id: collectionProjectId.trim(),
        name: collectionName.trim(),
        description: collectionDescription.trim(),
      })
      setCollectionSelectedId(created.collection_id)
      setCollectionName('')
      setCollectionDescription('')
      onNotice(`Created collection ${created.name}`)
      await refresh()
    })
  }

  const runSelectedCollectionItemAction = (
    action: (input: { collectionId: string; engramId: string }) => Promise<void>,
    isAddAction: boolean,
  ) => {
    if (!selectedCollection) {
      return
    }
    const engramId = collectionAddEngramId.trim()
    if (!engramId) {
      return
    }
    const collectionName = selectedCollection.name
    const collectionId = selectedCollection.collection_id
    void runAction(async () => {
      await action({ collectionId, engramId })
      onNotice(`${isAddAction ? 'Added' : 'Removed'} engram ${engramId} ${isAddAction ? 'to' : 'from'} collection ${collectionName}`)
      setCollectionAddEngramId('')
    })
  }

  const addItemToSelectedCollection = () => {
    runSelectedCollectionItemAction(
      async ({ collectionId, engramId }) => {
        await addCollectionItems(collectionId, {
          engram_ids: [engramId],
        })
      },
      true,
    )
  }

  const removeItemFromSelectedCollection = () => {
    runSelectedCollectionItemAction(
      async ({ collectionId, engramId }) => {
        await removeCollectionItem(collectionId, engramId)
      },
      false,
    )
  }

  const renameCollectionQuickly = (collection: EngramCollectionRecord) => {
    void runAction(async () => {
      await updateCollection(collection.collection_id, {
        name: `${collection.name} (updated)`,
        expected_updated_at: collection.updated_at,
      })
      onNotice(`Updated collection ${collection.collection_id}`)
      await refresh()
    })
  }

  const deleteCollectionRecord = (collection: EngramCollectionRecord) => {
    const collectionId = collection.collection_id
    void runAction(async () => {
      await deleteCollection(collectionId, { reason: 'admin-ui-delete' })
      onNotice(`Deleted collection ${collectionId}`)
      await refresh()
    })
  }

  const addMemberToProject = () => {
    const normalizedProjectID = projectId.trim()
    const normalizedUserID = memberUserId.trim()
    if (!normalizedProjectID || !normalizedUserID) {
      return
    }
    void runAction(async () => {
      const added = await addProjectMember({
        project_id: normalizedProjectID,
        payload: {
          user_id: normalizedUserID,
          role: memberRole,
        },
      })
      setMemberUserId('')
      onNotice(`Added member ${added.user_id} (${added.role})`)
      await refresh()
    })
  }

  const updateProjectMemberRoleByRecord = (member: ProjectMemberRecord) => {
    const normalizedProjectID = projectId.trim()
    if (!normalizedProjectID) {
      return
    }
    const targetRole = memberRoleDrafts[member.user_id] ?? member.role
    void runAction(async () => {
      const updated = await updateProjectMember({
        project_id: normalizedProjectID,
        user_id: member.user_id,
        payload: {
          role: targetRole,
        },
      })
      onNotice(`Updated member ${updated.user_id} to ${updated.role}`)
      await refresh()
    })
  }

  const removeProjectMemberByRecord = (member: ProjectMemberRecord) => {
    const normalizedProjectID = projectId.trim()
    if (!normalizedProjectID) {
      return
    }
    void runAction(async () => {
      await removeProjectMember({
        project_id: normalizedProjectID,
        user_id: member.user_id,
      })
      onNotice(`Removed member ${member.user_id}`)
      await refresh()
    })
  }

  const refreshView = () => {
    void refresh()
  }

  const restoreSessionById = (session: AdminChatSessionRecord) => {
    const sessionId = session.session_id
    void runAction(async () => {
      await restoreAdminSession(sessionId)
      onNotice(`Restored session ${sessionId}`)
      await refresh()
    })
  }

  const deleteSessionById = (session: AdminChatSessionRecord) => {
    const sessionId = session.session_id
    void runAction(async () => {
      await deleteAdminSession(sessionId, {
        delete_linked_engrams: deleteLinkedEngrams,
        reason: 'admin-ui-cleanup',
      })
      onNotice(`Deleted session ${sessionId}`)
      await refresh()
    })
  }

  const selectEngramRecord = (engram: AdminEngramRecord) => {
    void selectEngram({ engramId: engram.engram_id })
  }

  return (
    <AdminLayout>
      <SessionManagementSection
        projectId={projectId}
        projectIdOptions={projectIdOptions}
        includeDeleted={includeDeleted}
        deleteLinkedEngrams={deleteLinkedEngrams}
        loading={loading}
        submitting={submitting}
        sessions={sessions}
        onProjectChange={onProjectChange}
        onIncludeDeletedChange={setIncludeDeleted}
        onDeleteLinkedEngramsChange={setDeleteLinkedEngrams}
        onRefresh={refreshView}
        onRestoreSession={restoreSessionById}
        onDeleteSession={deleteSessionById}
      />

      <EngramManagementSection
        queryText={queryText}
        loading={loading}
        submitting={submitting}
        engrams={engrams}
        selectedEngramId={selectedEngramId}
        selectedEngram={selectedEngram}
        editTitle={editTitle}
        editAbstract={editAbstract}
        editMarkdown={editMarkdown}
        editTags={editTags}
        editKeywords={editKeywords}
        projectIdOptions={projectIdOptions}
        moveTargetProject={moveTargetProject}
        sourceDraft={sourceDraft}
        sourceRows={sourceRows}
        onQueryTextChange={(event) => setQueryText(event.target.value)}
        onRefresh={refreshView}
        onSelectEngram={selectEngramRecord}
        onEditTitleChange={(event) => setEditTitle(event.target.value)}
        onEditAbstractChange={(event) => setEditAbstract(event.target.value)}
        onEditMarkdownChange={(event) => setEditMarkdown(event.target.value)}
        onEditTagsChange={(event) => setEditTags(event.target.value)}
        onEditKeywordsChange={(event) => setEditKeywords(event.target.value)}
        onMoveTargetProjectChange={setMoveTargetProject}
        onSourceUrlChange={handleSourceUrlChange}
        onSourceTitleChange={handleSourceTitleChange}
        onSourceSnippetChange={handleSourceSnippetChange}
        onAddSource={addSourceRow}
        onClearSources={clearSourceRows}
        onSaveEngram={saveSelectedEngram}
        onMoveEngram={moveSelectedEngram}
        onRestoreEngram={restoreSelectedEngram}
        onDeleteEngram={deleteSelectedEngram}
      />

      <CollectionSection
        collections={collections}
        selectedCollectionId={collectionSelectedId}
        selectedCollection={selectedCollection}
        projectIdOptions={projectIdOptions}
        collectionProjectId={collectionProjectId}
        collectionName={collectionName}
        collectionDescription={collectionDescription}
        collectionAddEngramId={collectionAddEngramId}
        submitting={submitting}
        onCollectionProjectIdChange={setCollectionProjectId}
        onCollectionNameChange={(event) => setCollectionName(event.target.value)}
        onCollectionDescriptionChange={(event) => setCollectionDescription(event.target.value)}
        onCollectionAddEngramIdChange={(event) => setCollectionAddEngramId(event.target.value)}
        onSelectCollection={(collection) => setCollectionSelectedId(collection.collection_id)}
        onCreateCollection={createCollectionRecord}
        onAddItem={addItemToSelectedCollection}
        onRemoveItem={removeItemFromSelectedCollection}
        onQuickRenameCollection={renameCollectionQuickly}
        onDeleteCollection={deleteCollectionRecord}
      />

      <ProjectMembersSection
        projectId={projectId}
        projectIdOptions={projectIdOptions}
        submitting={submitting}
        members={members}
        memberUserId={memberUserId}
        memberRole={memberRole}
        memberRoleDrafts={memberRoleDrafts}
        onProjectChange={onProjectChange}
        onMemberUserIdChange={(event) => setMemberUserId(event.target.value)}
        onMemberRoleChange={(event) => setMemberRole(event.target.value as ProjectMemberRole)}
        onMemberRoleDraftChange={(userId, role) =>
          setMemberRoleDrafts((current) => ({
            ...current,
            [userId]: role,
          }))
        }
        onAddMember={addMemberToProject}
        onUpdateMember={updateProjectMemberRoleByRecord}
        onRemoveMember={removeProjectMemberByRecord}
      />

      <ProjectAuditSection
        projectId={projectId}
        projectIdOptions={projectIdOptions}
        events={auditEvents}
        loading={loading}
        onProjectChange={onProjectChange}
        onRefresh={refreshView}
      />

      {error ? <ErrorText>{error}</ErrorText> : null}
    </AdminLayout>
  )
}
