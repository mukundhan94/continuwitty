import { render, screen } from '@testing-library/react'
import type { ComponentProps } from 'react'
import userEvent from '@testing-library/user-event'
import { ThemeProvider } from 'styled-components'
import { describe, expect, it, vi } from 'vitest'

import type { DocumentRecord } from '../api/types'
import { lightTheme } from '../styles/theme'
import { DocumentIngestionPanel } from './DocumentIngestionPanel'

function buildDocument(overrides: Partial<DocumentRecord> = {}): DocumentRecord {
  return {
    document_id: 'doc-1',
    owner_user_id: '00000000-0000-0000-0000-000000000001',
    project_id: 'engram-vault',
    title: 'Incident Timeline',
    source_type: 'text',
    source_name: null,
    mime_type: 'text/plain',
    visibility_scope: 'project',
    content_hash: 'abc123',
    chunk_count: 4,
    created_at: '2026-02-15T00:00:00Z',
    updated_at: '2026-02-15T00:00:00Z',
    ...overrides,
  }
}

function renderPanel(overrides: Partial<ComponentProps<typeof DocumentIngestionPanel>> = {}) {
  const props: ComponentProps<typeof DocumentIngestionPanel> = {
    projectId: 'engram-vault',
    selectedSessionId: 'session-1',
    documents: [buildDocument()],
    pinnedDocumentIds: [],
    loading: false,
    submitting: false,
    error: null,
    onRefresh: vi.fn(async () => {}),
    onIngestText: vi.fn(async () => {}),
    onIngestFile: vi.fn(async () => {}),
    onPinDocument: vi.fn(async () => {}),
    onUnpinDocument: vi.fn(async () => {}),
    ...overrides,
  }

  render(
    <ThemeProvider theme={lightTheme}>
      <DocumentIngestionPanel {...props} />
    </ThemeProvider>,
  )
  return props
}

describe('DocumentIngestionPanel', () => {
  it('renders ingested documents', () => {
    renderPanel()

    expect(screen.getByTestId('document-card-doc-1')).toBeInTheDocument()
    expect(screen.getByText('Incident Timeline')).toBeInTheDocument()
  })

  it('submits text ingestion payload', async () => {
    const user = userEvent.setup()
    const onIngestText = vi.fn(async () => {})
    renderPanel({ onIngestText })

    await user.click(screen.getByRole('button', { name: 'Show Upload Form' }))
    await user.click(screen.getByRole('button', { name: 'Text' }))
    await user.type(screen.getByLabelText('Title'), 'Triage Notes')
    await user.type(screen.getByLabelText('Text'), 'Queue depth crossed threshold for 20 minutes.')
    await user.click(screen.getByRole('button', { name: 'Ingest Text' }))

    expect(onIngestText).toHaveBeenCalledTimes(1)
    expect(onIngestText).toHaveBeenCalledWith(
      expect.objectContaining({
        title: 'Triage Notes',
        text: 'Queue depth crossed threshold for 20 minutes.',
      }),
    )
  })

  it('submits file ingestion payload when file is selected', async () => {
    const user = userEvent.setup()
    const onIngestFile = vi.fn(async () => {})
    renderPanel({ onIngestFile })

    await user.click(screen.getByRole('button', { name: 'Show Upload Form' }))
    const input = screen.getByLabelText('Choose file')
    const file = new File(['hello world'], 'notes.txt', { type: 'text/plain' })
    await user.upload(input, file)
    await user.click(screen.getByRole('button', { name: 'Upload and Chunk' }))

    expect(onIngestFile).toHaveBeenCalledTimes(1)
    expect(onIngestFile).toHaveBeenCalledWith(
      expect.objectContaining({
        file,
      }),
    )
  })

  it('keeps upload form collapsed by default and can expand it', async () => {
    const user = userEvent.setup()
    renderPanel()

    expect(screen.queryByTestId('ingest-file-form')).not.toBeInTheDocument()
    await user.click(screen.getByRole('button', { name: 'Show Upload Form' }))
    expect(screen.getByTestId('ingest-file-form')).toBeInTheDocument()
  })

  it('pins and unpins documents from recent list', async () => {
    const user = userEvent.setup()
    const onPinDocument = vi.fn(async () => {})
    const onUnpinDocument = vi.fn(async () => {})

    const { rerender } = render(
      <ThemeProvider theme={lightTheme}>
        <DocumentIngestionPanel
          projectId="engram-vault"
          selectedSessionId="session-1"
          documents={[buildDocument()]}
          pinnedDocumentIds={[]}
          loading={false}
          submitting={false}
          error={null}
          onRefresh={vi.fn(async () => {})}
          onIngestText={vi.fn(async () => {})}
          onIngestFile={vi.fn(async () => {})}
          onPinDocument={onPinDocument}
          onUnpinDocument={onUnpinDocument}
        />
      </ThemeProvider>,
    )

    await user.click(screen.getByRole('button', { name: 'Pin to Chat' }))
    expect(onPinDocument).toHaveBeenCalledWith('doc-1')

    rerender(
      <ThemeProvider theme={lightTheme}>
        <DocumentIngestionPanel
          projectId="engram-vault"
          selectedSessionId="session-1"
          documents={[buildDocument()]}
          pinnedDocumentIds={['doc-1']}
          loading={false}
          submitting={false}
          error={null}
          onRefresh={vi.fn(async () => {})}
          onIngestText={vi.fn(async () => {})}
          onIngestFile={vi.fn(async () => {})}
          onPinDocument={onPinDocument}
          onUnpinDocument={onUnpinDocument}
        />
      </ThemeProvider>,
    )

    await user.click(screen.getByRole('button', { name: 'Unpin' }))
    expect(onUnpinDocument).toHaveBeenCalledWith('doc-1')
  })
})
