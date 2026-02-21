import { useCallback, useEffect, useMemo, useState } from 'react'

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
import type {
  AdminChatSessionRecord,
  AdminEngramRecord,
  AdminEngramSourceInput,
  EngramCollectionRecord,
} from '../api/types'
import { ErrorText, GlassPane, MutedText, PaneHeader, ScrollColumn, SectionDivider } from '../styles/primitives'

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

function toIsoNow() {
  return new Date().toISOString()
}

function parseCommaSeparated(value: string): string[] {
  return value
    .split(',')
    .map((item) => item.trim())
    .filter((item) => item.length > 0)
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
  includeDeleted: boolean
  deleteLinkedEngrams: boolean
  loading: boolean
  submitting: boolean
  sessions: AdminChatSessionRecord[]
  onProjectChange: (projectId: string) => void
  onIncludeDeletedChange: (nextValue: boolean) => void
  onDeleteLinkedEngramsChange: (nextValue: boolean) => void
  onRefresh: () => void
  onRestoreSession: (sessionId: string) => void
  onDeleteSession: (sessionId: string) => void
}

interface EngramDetailEditorProps {
  selectedEngram: AdminEngramRecord
  submitting: boolean
  editTitle: string
  editAbstract: string
  editMarkdown: string
  editTags: string
  editKeywords: string
  moveTargetProject: string
  sourceDraft: AdminEngramSourceInput
  sourceRows: AdminEngramSourceInput[]
  onEditTitleChange: (value: string) => void
  onEditAbstractChange: (value: string) => void
  onEditMarkdownChange: (value: string) => void
  onEditTagsChange: (value: string) => void
  onEditKeywordsChange: (value: string) => void
  onMoveTargetProjectChange: (value: string) => void
  onSourceDraftFieldChange: (field: keyof AdminEngramSourceInput, value: string) => void
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
  moveTargetProject: string
  sourceDraft: AdminEngramSourceInput
  sourceRows: AdminEngramSourceInput[]
  onQueryTextChange: (value: string) => void
  onRefresh: () => void
  onSelectEngram: (engramId: string) => void
  onEditTitleChange: (value: string) => void
  onEditAbstractChange: (value: string) => void
  onEditMarkdownChange: (value: string) => void
  onEditTagsChange: (value: string) => void
  onEditKeywordsChange: (value: string) => void
  onMoveTargetProjectChange: (value: string) => void
  onSourceDraftFieldChange: (field: keyof AdminEngramSourceInput, value: string) => void
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
  collectionProjectId: string
  collectionName: string
  collectionDescription: string
  collectionAddEngramId: string
  submitting: boolean
  onCollectionProjectIdChange: (value: string) => void
  onCollectionNameChange: (value: string) => void
  onCollectionDescriptionChange: (value: string) => void
  onCollectionAddEngramIdChange: (value: string) => void
  onSelectCollection: (collectionId: string) => void
  onCreateCollection: () => void
  onAddItem: () => void
  onRemoveItem: () => void
  onQuickRenameCollection: (collection: EngramCollectionRecord) => void
  onDeleteCollection: (collectionId: string) => void
}

function SessionManagementSection({
  projectId,
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
        <input
          value={projectId}
          onChange={(event) => onProjectChange(event.target.value)}
          placeholder="project-id filter"
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
                  <button type="button" disabled={submitting} onClick={() => onRestoreSession(session.session_id)}>
                    Restore
                  </button>
                ) : (
                  <button type="button" disabled={submitting} onClick={() => onDeleteSession(session.session_id)}>
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
  moveTargetProject,
  sourceDraft,
  sourceRows,
  onEditTitleChange,
  onEditAbstractChange,
  onEditMarkdownChange,
  onEditTagsChange,
  onEditKeywordsChange,
  onMoveTargetProjectChange,
  onSourceDraftFieldChange,
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
        <input value={editTitle} onChange={(event) => onEditTitleChange(event.target.value)} />
      </Field>
      <Field>
        <span>Abstract</span>
        <textarea rows={3} value={editAbstract} onChange={(event) => onEditAbstractChange(event.target.value)} />
      </Field>
      <Field>
        <span>Markdown</span>
        <textarea rows={6} value={editMarkdown} onChange={(event) => onEditMarkdownChange(event.target.value)} />
      </Field>
      <Field>
        <span>Tags (comma-separated)</span>
        <input value={editTags} onChange={(event) => onEditTagsChange(event.target.value)} />
      </Field>
      <Field>
        <span>Keywords (comma-separated)</span>
        <input value={editKeywords} onChange={(event) => onEditKeywordsChange(event.target.value)} />
      </Field>
      <Field>
        <span>Move target project</span>
        <input value={moveTargetProject} onChange={(event) => onMoveTargetProjectChange(event.target.value)} />
      </Field>
      <SourceEditor>
        <strong className="font-display text-sm text-ink">Sources</strong>
        {sourceRows.map((source, index) => (
          <MutedText key={`${source.url}-${index}`}>{source.title || source.url}</MutedText>
        ))}
        <input
          value={sourceDraft.url}
          placeholder="source url"
          onChange={(event) => onSourceDraftFieldChange('url', event.target.value)}
        />
        <input
          value={sourceDraft.title}
          placeholder="source title"
          onChange={(event) => onSourceDraftFieldChange('title', event.target.value)}
        />
        <textarea
          rows={2}
          value={sourceDraft.snippet}
          placeholder="source snippet"
          onChange={(event) => onSourceDraftFieldChange('snippet', event.target.value)}
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
  onSourceDraftFieldChange,
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
          onChange={(event) => onQueryTextChange(event.target.value)}
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
              onClick={() => onSelectEngram(engram.engram_id)}
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
          moveTargetProject={moveTargetProject}
          sourceDraft={sourceDraft}
          sourceRows={sourceRows}
          onEditTitleChange={onEditTitleChange}
          onEditAbstractChange={onEditAbstractChange}
          onEditMarkdownChange={onEditMarkdownChange}
          onEditTagsChange={onEditTagsChange}
          onEditKeywordsChange={onEditKeywordsChange}
          onMoveTargetProjectChange={onMoveTargetProjectChange}
          onSourceDraftFieldChange={onSourceDraftFieldChange}
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

function CollectionSection({
  collections,
  selectedCollectionId,
  selectedCollection,
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
        <input
          value={collectionProjectId}
          onChange={(event) => onCollectionProjectIdChange(event.target.value)}
          placeholder="project id"
        />
      </Field>
      <Field>
        <span>Name</span>
        <input value={collectionName} onChange={(event) => onCollectionNameChange(event.target.value)} />
      </Field>
      <Field>
        <span>Description</span>
        <textarea
          rows={2}
          value={collectionDescription}
          onChange={(event) => onCollectionDescriptionChange(event.target.value)}
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
          onChange={(event) => onCollectionAddEngramIdChange(event.target.value)}
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
      <ScrollColumn>
        <TableLike>
          {collections.map((collection) => (
            <RowCard
              key={collection.collection_id}
              $active={collection.collection_id === selectedCollectionId}
              onClick={() => onSelectCollection(collection.collection_id)}
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
                    onClick={() => onDeleteCollection(collection.collection_id)}
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
    </GlassPane>
  )
}

export function AdminMemoryPage({ projectId, onProjectChange, onNotice }: AdminMemoryPageProps) {
  const [includeDeleted, setIncludeDeleted] = useState(false)
  const [queryText, setQueryText] = useState('')
  const [sessions, setSessions] = useState<AdminChatSessionRecord[]>([])
  const [engrams, setEngrams] = useState<AdminEngramRecord[]>([])
  const [collections, setCollections] = useState<EngramCollectionRecord[]>([])
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
  const [collectionName, setCollectionName] = useState('')
  const [collectionDescription, setCollectionDescription] = useState('')
  const [collectionSelectedId, setCollectionSelectedId] = useState<string | null>(null)
  const [collectionAddEngramId, setCollectionAddEngramId] = useState('')
  const [deleteLinkedEngrams, setDeleteLinkedEngrams] = useState(false)
  const [loading, setLoading] = useState(false)
  const [submitting, setSubmitting] = useState(false)
  const [error, setError] = useState<string | null>(null)

  const selectedCollection = useMemo(
    () => collections.find((item) => item.collection_id === collectionSelectedId) ?? null,
    [collectionSelectedId, collections],
  )

  const refresh = useCallback(async () => {
    setLoading(true)
    setError(null)
    try {
      const projectFilter = projectId.trim() || undefined
      const [nextSessions, nextEngrams, nextCollections] = await Promise.all([
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
      ])
      setSessions(nextSessions)
      setEngrams(nextEngrams)
      setCollections(nextCollections)
    } catch (refreshError) {
      setError(refreshError instanceof Error ? refreshError.message : 'Failed to load admin memory data.')
    } finally {
      setLoading(false)
    }
  }, [includeDeleted, projectId, queryText])

  useEffect(() => {
    void refresh()
  }, [refresh])

  const selectEngram = useCallback(async (engramId: string) => {
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

  const updateSourceDraftField = (field: keyof AdminEngramSourceInput, value: string) => {
    setSourceDraft((current) => ({
      ...current,
      [field]: value,
    }))
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
        tags: parseCommaSeparated(editTags),
        keywords: parseCommaSeparated(editKeywords),
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
    action: (engramId: string) => Promise<unknown>,
    noticePrefix: string,
  ) => {
    if (!selectedEngram) {
      return
    }
    const engramId = selectedEngram.engram_id
    void runAction(async () => {
      await action(engramId)
      onNotice(`${noticePrefix} engram ${engramId}`)
      await selectEngram(engramId)
      await refresh()
    })
  }

  const restoreSelectedEngram = () => {
    runSelectedEngramLifecycleAction(
      async (engramId) => {
        await restoreAdminEngram(engramId)
      },
      'Restored',
    )
  }

  const deleteSelectedEngram = () => {
    runSelectedEngramLifecycleAction(
      async (engramId) => {
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
    action: (collectionId: string, engramId: string) => Promise<void>,
    noticePrefix: string,
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
      await action(collectionId, engramId)
      onNotice(`${noticePrefix} engram ${engramId} ${noticePrefix === 'Added' ? 'to' : 'from'} collection ${collectionName}`)
      setCollectionAddEngramId('')
    })
  }

  const addItemToSelectedCollection = () => {
    runSelectedCollectionItemAction(
      async (collectionId, engramId) => {
        await addCollectionItems(collectionId, {
          engram_ids: [engramId],
        })
      },
      'Added',
    )
  }

  const removeItemFromSelectedCollection = () => {
    runSelectedCollectionItemAction(
      async (collectionId, engramId) => {
        await removeCollectionItem(collectionId, engramId)
      },
      'Removed',
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

  const deleteCollectionRecord = (collectionId: string) => {
    void runAction(async () => {
      await deleteCollection(collectionId, { reason: 'admin-ui-delete' })
      onNotice(`Deleted collection ${collectionId}`)
      await refresh()
    })
  }

  const refreshView = () => {
    void refresh()
  }

  const restoreSessionById = (sessionId: string) => {
    void runAction(async () => {
      await restoreAdminSession(sessionId)
      onNotice(`Restored session ${sessionId}`)
      await refresh()
    })
  }

  const deleteSessionById = (sessionId: string) => {
    void runAction(async () => {
      await deleteAdminSession(sessionId, {
        delete_linked_engrams: deleteLinkedEngrams,
        reason: 'admin-ui-cleanup',
      })
      onNotice(`Deleted session ${sessionId}`)
      await refresh()
    })
  }

  const selectEngramById = (engramId: string) => {
    void selectEngram(engramId)
  }

  return (
    <AdminLayout>
      <SessionManagementSection
        projectId={projectId}
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
        moveTargetProject={moveTargetProject}
        sourceDraft={sourceDraft}
        sourceRows={sourceRows}
        onQueryTextChange={setQueryText}
        onRefresh={refreshView}
        onSelectEngram={selectEngramById}
        onEditTitleChange={setEditTitle}
        onEditAbstractChange={setEditAbstract}
        onEditMarkdownChange={setEditMarkdown}
        onEditTagsChange={setEditTags}
        onEditKeywordsChange={setEditKeywords}
        onMoveTargetProjectChange={setMoveTargetProject}
        onSourceDraftFieldChange={updateSourceDraftField}
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
        collectionProjectId={collectionProjectId}
        collectionName={collectionName}
        collectionDescription={collectionDescription}
        collectionAddEngramId={collectionAddEngramId}
        submitting={submitting}
        onCollectionProjectIdChange={setCollectionProjectId}
        onCollectionNameChange={setCollectionName}
        onCollectionDescriptionChange={setCollectionDescription}
        onCollectionAddEngramIdChange={setCollectionAddEngramId}
        onSelectCollection={setCollectionSelectedId}
        onCreateCollection={createCollectionRecord}
        onAddItem={addItemToSelectedCollection}
        onRemoveItem={removeItemFromSelectedCollection}
        onQuickRenameCollection={renameCollectionQuickly}
        onDeleteCollection={deleteCollectionRecord}
      />

      {error ? <ErrorText>{error}</ErrorText> : null}
    </AdminLayout>
  )
}
