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

function toIsoNow() {
  return new Date().toISOString()
}

function parseCommaSeparated(value: string): string[] {
  return value
    .split(',')
    .map((item) => item.trim())
    .filter((item) => item.length > 0)
}

interface AdminMemoryPageProps {
  projectId: string
  onProjectChange: (projectId: string) => void
  onNotice: (message: string) => void
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
  const [sourceDraft, setSourceDraft] = useState<AdminEngramSourceInput>({
    captured_at: toIsoNow(),
    url: '',
    title: '',
    snippet: '',
    content_text: '',
    content_hash: '',
  })
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
      const [nextSessions, nextEngrams, nextCollections] = await Promise.all([
        listAdminSessions({
          project_id: projectId.trim() || undefined,
          include_deleted: includeDeleted,
          limit: 200,
          offset: 0,
        }),
        listAdminEngrams({
          project_id: projectId.trim() || undefined,
          q: queryText.trim() || undefined,
          include_deleted: includeDeleted,
          limit: 300,
          offset: 0,
        }),
        listCollections({
          project_id: projectId.trim() || undefined,
          include_deleted: includeDeleted,
          limit: 200,
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

  const runAction = useCallback(
    async (action: () => Promise<void>) => {
      setSubmitting(true)
      setError(null)
      try {
        await action()
      } catch (actionError) {
        setError(actionError instanceof Error ? actionError.message : 'Admin action failed.')
      } finally {
        setSubmitting(false)
      }
    },
    [],
  )

  return (
    <AdminLayout>
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
              onChange={(event) => setIncludeDeleted(event.target.checked)}
            />
            Include deleted
          </label>
          <button type="button" disabled={loading || submitting} onClick={() => void refresh()}>
            Refresh
          </button>
        </Toolbar>
        <label>
          <input
            type="checkbox"
            checked={deleteLinkedEngrams}
            onChange={(event) => setDeleteLinkedEngrams(event.target.checked)}
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
                    <button
                      type="button"
                      disabled={submitting}
                      onClick={() =>
                        void runAction(async () => {
                          await restoreAdminSession(session.session_id)
                          onNotice(`Restored session ${session.session_id}`)
                          await refresh()
                        })
                      }
                    >
                      Restore
                    </button>
                  ) : (
                    <button
                      type="button"
                      disabled={submitting}
                      onClick={() =>
                        void runAction(async () => {
                          await deleteAdminSession(session.session_id, {
                            delete_linked_engrams: deleteLinkedEngrams,
                            reason: 'admin-ui-cleanup',
                          })
                          onNotice(`Deleted session ${session.session_id}`)
                          await refresh()
                        })
                      }
                    >
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

      <GlassPane>
        <PaneHeader>
          <h2 className="font-display text-base font-semibold tracking-[0.02em] text-ink">Engram Management</h2>
        </PaneHeader>
        <Toolbar>
          <input
            value={queryText}
            onChange={(event) => setQueryText(event.target.value)}
            placeholder="search title / abstract / markdown"
          />
          <button type="button" disabled={loading || submitting} onClick={() => void refresh()}>
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
                onClick={() => void selectEngram(engram.engram_id)}
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
          <>
            <SectionDivider />
            <Field>
              <span>Title</span>
              <input value={editTitle} onChange={(event) => setEditTitle(event.target.value)} />
            </Field>
            <Field>
              <span>Abstract</span>
              <textarea rows={3} value={editAbstract} onChange={(event) => setEditAbstract(event.target.value)} />
            </Field>
            <Field>
              <span>Markdown</span>
              <textarea rows={6} value={editMarkdown} onChange={(event) => setEditMarkdown(event.target.value)} />
            </Field>
            <Field>
              <span>Tags (comma-separated)</span>
              <input value={editTags} onChange={(event) => setEditTags(event.target.value)} />
            </Field>
            <Field>
              <span>Keywords (comma-separated)</span>
              <input value={editKeywords} onChange={(event) => setEditKeywords(event.target.value)} />
            </Field>
            <Field>
              <span>Move target project</span>
              <input value={moveTargetProject} onChange={(event) => setMoveTargetProject(event.target.value)} />
            </Field>
            <SourceEditor>
              <strong className="font-display text-sm text-ink">Sources</strong>
              {sourceRows.map((source, index) => (
                <MutedText key={`${source.url}-${index}`}>{source.title || source.url}</MutedText>
              ))}
              <input
                value={sourceDraft.url}
                placeholder="source url"
                onChange={(event) =>
                  setSourceDraft((current) => ({
                    ...current,
                    url: event.target.value,
                  }))
                }
              />
              <input
                value={sourceDraft.title}
                placeholder="source title"
                onChange={(event) =>
                  setSourceDraft((current) => ({
                    ...current,
                    title: event.target.value,
                  }))
                }
              />
              <textarea
                rows={2}
                value={sourceDraft.snippet}
                placeholder="source snippet"
                onChange={(event) =>
                  setSourceDraft((current) => ({
                    ...current,
                    snippet: event.target.value,
                  }))
                }
              />
              <RowActions>
                <button
                  type="button"
                  onClick={() =>
                    setSourceRows((current) => [
                      ...current,
                      {
                        ...sourceDraft,
                        captured_at: sourceDraft.captured_at || toIsoNow(),
                      },
                    ])
                  }
                >
                  Add Source
                </button>
                <button type="button" onClick={() => setSourceRows([])}>
                  Clear Sources
                </button>
              </RowActions>
            </SourceEditor>
            <RowActions>
              <button
                type="button"
                disabled={submitting}
                onClick={() =>
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
              >
                Save
              </button>
              <button
                type="button"
                disabled={submitting || !moveTargetProject.trim()}
                onClick={() =>
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
              >
                Move
              </button>
              {selectedEngram.deleted_at ? (
                <button
                  type="button"
                  disabled={submitting}
                  onClick={() =>
                    void runAction(async () => {
                      await restoreAdminEngram(selectedEngram.engram_id)
                      onNotice(`Restored engram ${selectedEngram.engram_id}`)
                      await selectEngram(selectedEngram.engram_id)
                      await refresh()
                    })
                  }
                >
                  Restore
                </button>
              ) : (
                <button
                  type="button"
                  disabled={submitting}
                  onClick={() =>
                    void runAction(async () => {
                      await deleteAdminEngram(selectedEngram.engram_id, { reason: 'admin-ui-delete' })
                      onNotice(`Deleted engram ${selectedEngram.engram_id}`)
                      await selectEngram(selectedEngram.engram_id)
                      await refresh()
                    })
                  }
                >
                  Delete
                </button>
              )}
            </RowActions>
          </>
        ) : (
          <MutedText>Select an engram to edit metadata, move projects, or soft-delete/restore.</MutedText>
        )}
      </GlassPane>

      <GlassPane>
        <PaneHeader>
          <h2 className="font-display text-base font-semibold tracking-[0.02em] text-ink">Collections</h2>
        </PaneHeader>
        <Field>
          <span>Collection project</span>
          <input
            value={collectionProjectId}
            onChange={(event) => setCollectionProjectId(event.target.value)}
            placeholder="project id"
          />
        </Field>
        <Field>
          <span>Name</span>
          <input value={collectionName} onChange={(event) => setCollectionName(event.target.value)} />
        </Field>
        <Field>
          <span>Description</span>
          <textarea
            rows={2}
            value={collectionDescription}
            onChange={(event) => setCollectionDescription(event.target.value)}
          />
        </Field>
        <button
          type="button"
          disabled={submitting || !collectionProjectId.trim() || !collectionName.trim()}
          onClick={() =>
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
        >
          Create Collection
        </button>
        <SectionDivider />
        <Field>
          <span>Add engram to selected collection</span>
          <input
            value={collectionAddEngramId}
            onChange={(event) => setCollectionAddEngramId(event.target.value)}
            placeholder="engram id"
          />
        </Field>
        <RowActions>
          <button
            type="button"
            disabled={submitting || !selectedCollection || !collectionAddEngramId.trim()}
            onClick={() =>
              void runAction(async () => {
                const targetCollection = selectedCollection
                if (!targetCollection) {
                  return
                }
                await addCollectionItems(targetCollection.collection_id, {
                  engram_ids: [collectionAddEngramId.trim()],
                })
                onNotice(`Added engram ${collectionAddEngramId.trim()} to collection ${targetCollection.name}`)
                setCollectionAddEngramId('')
              })
            }
          >
            Add Item
          </button>
          <button
            type="button"
            disabled={submitting || !selectedCollection || !collectionAddEngramId.trim()}
            onClick={() =>
              void runAction(async () => {
                const targetCollection = selectedCollection
                if (!targetCollection) {
                  return
                }
                await removeCollectionItem(targetCollection.collection_id, collectionAddEngramId.trim())
                onNotice(`Removed engram ${collectionAddEngramId.trim()} from collection ${targetCollection.name}`)
                setCollectionAddEngramId('')
              })
            }
          >
            Remove Item
          </button>
        </RowActions>
        <ScrollColumn>
          <TableLike>
            {collections.map((collection) => (
              <RowCard
                key={collection.collection_id}
                $active={collection.collection_id === collectionSelectedId}
                onClick={() => setCollectionSelectedId(collection.collection_id)}
              >
                <div className="font-display text-sm font-semibold text-ink">{collection.name}</div>
                <MutedText>
                  {collection.project_id} · {collection.collection_id}
                </MutedText>
                <RowActions>
                  <button
                    type="button"
                    disabled={submitting}
                    onClick={() =>
                      void runAction(async () => {
                        await updateCollection(collection.collection_id, {
                          name: `${collection.name} (updated)`,
                          expected_updated_at: collection.updated_at,
                        })
                        onNotice(`Updated collection ${collection.collection_id}`)
                        await refresh()
                      })
                    }
                  >
                    Quick Rename
                  </button>
                  {collection.deleted_at ? (
                    <MutedText>Deleted</MutedText>
                  ) : (
                    <button
                      type="button"
                      disabled={submitting}
                      onClick={() =>
                        void runAction(async () => {
                          await deleteCollection(collection.collection_id, { reason: 'admin-ui-delete' })
                          onNotice(`Deleted collection ${collection.collection_id}`)
                          await refresh()
                        })
                      }
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
      {error ? <ErrorText>{error}</ErrorText> : null}
    </AdminLayout>
  )
}
