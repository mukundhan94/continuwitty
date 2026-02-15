import { useMemo, useState } from 'react'
import type { ChangeEvent, FormEvent } from 'react'

import styled from 'styled-components'

import type { DocumentRecord } from '../api/types'
import {
  ErrorText,
  GlassPane,
  MutedText,
  PaneHeader,
  ScrollColumn,
  SectionDivider,
  SessionMeta,
} from '../styles/primitives'

const IngestionPane = styled(GlassPane)`
  overflow: auto;
`

const PanelBody = styled.div`
  flex: 1;
  min-height: 0;
  display: grid;
  gap: 0.6rem;
  align-content: start;
  overflow: auto;
  padding-right: 0.25rem;
`

const ModeTabs = styled.div`
  display: inline-flex;
  gap: 0.35rem;
`

const HeaderActions = styled.div`
  display: flex;
  align-items: center;
  gap: 0.4rem;
`

const SecondaryActions = styled.div`
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 0.5rem;
  flex-wrap: wrap;
`

const ModeButton = styled.button<{ $active: boolean }>`
  padding: 0.28rem 0.54rem;
  font-size: 0.76rem;
  border-radius: 8px;
  border: 1px solid var(--color-line);
  background: ${({ $active }) => ($active ? 'var(--session-active-bg)' : 'var(--surface-raised)')};
  color: var(--color-ink);
`

const CompactForm = styled.form`
  display: grid;
  gap: 0.45rem;
`

const SmallInput = styled.input`
  font-size: 0.82rem;
  padding: 0.4rem 0.48rem;
`

const SmallTextArea = styled.textarea`
  font-size: 0.82rem;
  padding: 0.45rem 0.52rem;
  min-height: 5rem;
  line-height: 1.4;
`

const DocList = styled(ScrollColumn)`
  min-height: 9rem;
  max-height: min(18rem, 38vh);
`

const DocCard = styled.article<{ $pinned: boolean }>`
  border: 1px solid var(--color-line);
  border-radius: 10px;
  padding: 0.45rem 0.52rem;
  background: var(--surface-raised);
  display: grid;
  gap: 0.18rem;

  ${({ $pinned }) =>
    $pinned
      ? `
    background: var(--session-active-bg);
    border-color: var(--session-active-border);
    box-shadow:
      inset 0 0 0 1px var(--session-active-border),
      0 0 0 1px var(--session-active-shadow);
  `
      : ''}
`

const DocActions = styled.div`
  display: flex;
  justify-content: flex-end;
  gap: 0.35rem;
`

const PinButton = styled.button`
  padding: 0.24rem 0.52rem;
  font-size: 0.72rem;
  line-height: 1.1;
`

interface DocumentIngestionPanelProps {
  projectId: string
  selectedSessionId: string | null
  documents: DocumentRecord[]
  pinnedDocumentIds: string[]
  loading: boolean
  submitting: boolean
  error: string | null
  onRefresh: () => Promise<void>
  onIngestText: (payload: {
    title: string
    text: string
    visibility_scope: 'private' | 'project'
    chunk_size_chars: number
    chunk_overlap_chars: number
  }) => Promise<void>
  onIngestFile: (payload: {
    title: string
    file: File
    visibility_scope: 'private' | 'project'
    chunk_size_chars: number
    chunk_overlap_chars: number
  }) => Promise<void>
  onPinDocument: (documentId: string) => Promise<void>
  onUnpinDocument: (documentId: string) => Promise<void>
}

export function DocumentIngestionPanel({
  projectId,
  selectedSessionId,
  documents,
  pinnedDocumentIds,
  loading,
  submitting,
  error,
  onRefresh,
  onIngestText,
  onIngestFile,
  onPinDocument,
  onUnpinDocument,
}: DocumentIngestionPanelProps) {
  const [mode, setMode] = useState<'text' | 'file'>('file')
  const [showUploadForm, setShowUploadForm] = useState(false)
  const [title, setTitle] = useState('')
  const [textBody, setTextBody] = useState('')
  const [file, setFile] = useState<File | null>(null)
  const [visibilityScope, setVisibilityScope] = useState<'private' | 'project'>('private')

  const sortedDocuments = useMemo(
    () => [...documents].sort((a, b) => b.created_at.localeCompare(a.created_at)),
    [documents],
  )
  const pinnedDocumentIdSet = useMemo(() => new Set(pinnedDocumentIds), [pinnedDocumentIds])

  const resetInputState = () => {
    setTitle('')
    setTextBody('')
    setFile(null)
  }

  const handleFileChange = (event: ChangeEvent<HTMLInputElement>) => {
    const nextFile = event.target.files?.[0] || null
    setFile(nextFile)
  }

  const handleSubmitText = async (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault()
    if (!textBody.trim()) {
      return
    }
    await onIngestText({
      title: title.trim() || 'Untitled Text Document',
      text: textBody,
      visibility_scope: visibilityScope,
      chunk_size_chars: 1000,
      chunk_overlap_chars: 180,
    })
    resetInputState()
  }

  const handleSubmitFile = async (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault()
    if (!file) {
      return
    }
    await onIngestFile({
      title: title.trim(),
      file,
      visibility_scope: visibilityScope,
      chunk_size_chars: 1000,
      chunk_overlap_chars: 180,
    })
    resetInputState()
  }

  return (
    <IngestionPane as="aside" data-testid="document-ingestion-panel">
      <PaneHeader>
        <h2 className="font-display text-base font-semibold tracking-[0.02em] text-ink">Document Ingestion</h2>
        <HeaderActions>
          <button type="button" onClick={() => void onRefresh()} disabled={loading || submitting}>
            Refresh
          </button>
        </HeaderActions>
      </PaneHeader>

      <PanelBody>
        <MutedText>Project: {projectId}</MutedText>

        <SectionDivider>
          <p className="font-display text-sm font-semibold text-ink">Recent Documents ({sortedDocuments.length})</p>
        </SectionDivider>

        <DocList>
          {loading ? <MutedText>Loading documents...</MutedText> : null}
          {!loading && sortedDocuments.length === 0 ? (
            <MutedText>No documents ingested for this project.</MutedText>
          ) : null}

          {sortedDocuments.map((item) => (
            <DocCard
              key={item.document_id}
              data-testid={`document-card-${item.document_id}`}
              $pinned={pinnedDocumentIdSet.has(item.document_id)}
            >
              <p className="font-semibold text-ink">{item.title}</p>
              <SessionMeta>
                {item.source_type} · chunks {item.chunk_count} · {item.visibility_scope}
              </SessionMeta>
              {pinnedDocumentIdSet.has(item.document_id) ? (
                <MutedText data-testid={`document-pinned-${item.document_id}`}>Pinned to active chat</MutedText>
              ) : null}
              {item.source_name ? <MutedText>{item.source_name}</MutedText> : null}
              <DocActions>
                {pinnedDocumentIdSet.has(item.document_id) ? (
                  <PinButton
                    type="button"
                    onClick={() => void onUnpinDocument(item.document_id)}
                    disabled={!selectedSessionId}
                  >
                    Unpin
                  </PinButton>
                ) : (
                  <PinButton
                    type="button"
                    onClick={() => void onPinDocument(item.document_id)}
                    disabled={!selectedSessionId}
                  >
                    Pin to Chat
                  </PinButton>
                )}
              </DocActions>
            </DocCard>
          ))}
        </DocList>

        {!selectedSessionId ? (
          <MutedText>Select a chat session to pin uploaded documents into message context.</MutedText>
        ) : null}

        <SectionDivider>
          <SecondaryActions>
            <p className="font-display text-sm font-semibold text-ink">Upload Controls</p>
            <button type="button" onClick={() => setShowUploadForm((current) => !current)} disabled={submitting}>
              {showUploadForm ? 'Hide Upload Form' : 'Show Upload Form'}
            </button>
          </SecondaryActions>
        </SectionDivider>

        {showUploadForm ? (
          <>
            <ModeTabs>
              <ModeButton type="button" $active={mode === 'file'} onClick={() => setMode('file')}>
                File
              </ModeButton>
              <ModeButton type="button" $active={mode === 'text'} onClick={() => setMode('text')}>
                Text
              </ModeButton>
            </ModeTabs>

            {mode === 'file' ? (
              <CompactForm onSubmit={handleSubmitFile} data-testid="ingest-file-form">
                <label htmlFor="ingest-file-title">Title (optional)</label>
                <SmallInput
                  id="ingest-file-title"
                  value={title}
                  onChange={(event) => setTitle(event.target.value)}
                  placeholder="Use filename if empty"
                />

                <label htmlFor="ingest-file-input">Choose file</label>
                <SmallInput id="ingest-file-input" type="file" onChange={handleFileChange} />

                <label htmlFor="ingest-file-visibility">Visibility</label>
                <select
                  id="ingest-file-visibility"
                  value={visibilityScope}
                  onChange={(event) => setVisibilityScope(event.target.value as 'private' | 'project')}
                >
                  <option value="private">Private</option>
                  <option value="project">Project</option>
                </select>

                <button type="submit" disabled={submitting || !file}>
                  {submitting ? 'Uploading...' : 'Upload and Chunk'}
                </button>
              </CompactForm>
            ) : (
              <CompactForm onSubmit={handleSubmitText} data-testid="ingest-text-form">
                <label htmlFor="ingest-text-title">Title</label>
                <SmallInput
                  id="ingest-text-title"
                  value={title}
                  onChange={(event) => setTitle(event.target.value)}
                  placeholder="Incident Timeline"
                />

                <label htmlFor="ingest-text-body">Text</label>
                <SmallTextArea
                  id="ingest-text-body"
                  value={textBody}
                  onChange={(event) => setTextBody(event.target.value)}
                  placeholder="Paste runbook notes, docs, or postmortem timeline..."
                />

                <label htmlFor="ingest-text-visibility">Visibility</label>
                <select
                  id="ingest-text-visibility"
                  value={visibilityScope}
                  onChange={(event) => setVisibilityScope(event.target.value as 'private' | 'project')}
                >
                  <option value="private">Private</option>
                  <option value="project">Project</option>
                </select>

                <button type="submit" disabled={submitting || !textBody.trim()}>
                  {submitting ? 'Ingesting...' : 'Ingest Text'}
                </button>
              </CompactForm>
            )}
          </>
        ) : null}

        {error ? <ErrorText>{error}</ErrorText> : null}
      </PanelBody>
    </IngestionPane>
  )
}
