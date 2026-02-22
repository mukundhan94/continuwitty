import { render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { useState } from 'react'
import { ThemeProvider } from 'styled-components'
import { beforeEach, describe, expect, it, vi } from 'vitest'

import type { ProjectImportResponse } from '../api/types'
import { lightTheme } from '../styles/theme'
import { ProjectTransferPage } from './ProjectTransferPage'

const exportApiMocks = vi.hoisted(() => ({
  exportProjectBundle: vi.fn(async () => ({
    blob: new Blob(['{}'], { type: 'application/json' }),
    filename: 'engram-export-engram-vault.json',
    contentType: 'application/json',
  })),
  importProjectBundle: vi.fn(async () => ({
    target_project_id: 'engram-vault',
    imported_engrams: 1,
    skipped_engrams: 0,
    overwritten_engrams: 0,
    imported_collections: 1,
    reused_collections: 0,
    imported_collection_items: 1,
    conflict_policy: 'skip',
  })),
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
}))

const memoryAdminMocks = vi.hoisted(() => ({
  listCollections: vi.fn(async () => [
    {
      collection_id: 'collection-a',
      project_id: 'engram-vault',
      owner_user_id: 'owner-1',
      name: 'A',
      description: '',
      created_at: '2026-01-01T00:00:00Z',
      updated_at: '2026-01-01T00:00:00Z',
      deleted_at: null,
      deleted_by_user_id: null,
      delete_reason: null,
    },
    {
      collection_id: 'collection-b',
      project_id: 'engram-vault',
      owner_user_id: 'owner-1',
      name: 'B',
      description: '',
      created_at: '2026-01-01T00:00:00Z',
      updated_at: '2026-01-01T00:00:00Z',
      deleted_at: null,
      deleted_by_user_id: null,
      delete_reason: null,
    },
  ]),
}))

vi.mock('../api/export', () => ({
  exportProjectBundle: exportApiMocks.exportProjectBundle,
  importProjectBundle: exportApiMocks.importProjectBundle,
}))

vi.mock('../api/projects', () => ({
  listProjects: projectApiMocks.listProjects,
}))

vi.mock('../api/memoryAdmin', () => ({
  listCollections: memoryAdminMocks.listCollections,
}))

function optionValues(testId: string): string[] {
  const list = screen.getByTestId(testId)
  const options = list.querySelectorAll('option')
  return [...options].map((item) => item.getAttribute('value') || '')
}

function renderTransferPage(overrides: Partial<{ projectId: string }> = {}) {
  const onProjectChange = vi.fn()
  const onNotice = vi.fn()

  function Harness() {
    const [projectId, setProjectId] = useState(overrides.projectId ?? 'engram-vault')
    return (
      <ProjectTransferPage
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

async function submitExport(
  params: Partial<{
    format: 'json' | 'zip'
    includeEmbeddings: boolean
    collectionIdsText: string
  }> = {},
) {
  const user = userEvent.setup()
  renderTransferPage()

  if (params.format) {
    await user.selectOptions(screen.getByTestId('export-format'), params.format)
  }
  if (params.collectionIdsText) {
    await user.type(screen.getByTestId('export-collection-ids'), params.collectionIdsText)
  }
  if (params.includeEmbeddings) {
    await user.click(screen.getByLabelText('Include embeddings (optional)'))
  }
  await user.click(screen.getByRole('button', { name: 'Export Bundle' }))
}

describe('ProjectTransferPage', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    Object.defineProperty(window.URL, 'createObjectURL', {
      configurable: true,
      writable: true,
      value: vi.fn(() => 'blob:project-export'),
    })
    Object.defineProperty(window.URL, 'revokeObjectURL', {
      configurable: true,
      writable: true,
      value: vi.fn(),
    })
    vi.spyOn(HTMLAnchorElement.prototype, 'click').mockImplementation(() => {})
  })

  it('renders export and import sections with smart project suggestions', async () => {
    const user = userEvent.setup()
    renderTransferPage()

    expect(screen.getByText('Project Export')).toBeInTheDocument()
    expect(screen.getByText('Project Import')).toBeInTheDocument()
    expect(screen.getByTestId('project-transfer-page')).toBeInTheDocument()

    await waitFor(() => {
      expect(projectApiMocks.listProjects).toHaveBeenCalled()
    })

    await user.clear(screen.getByTestId('import-project-id'))
    await user.type(screen.getByTestId('import-project-id'), 'ops')
    expect(optionValues('import-project-id-options')).toEqual(
      expect.arrayContaining(['ops-notes']),
    )
  })

  it('exports JSON with collection ID filters', async () => {
    const user = userEvent.setup()
    const { onNotice } = renderTransferPage()

    await user.type(screen.getByTestId('export-collection-ids'), 'collection-a, collection-b')
    await user.click(screen.getByRole('button', { name: 'Export Bundle' }))

    await waitFor(() => {
      expect(exportApiMocks.exportProjectBundle).toHaveBeenCalledWith({
        projectId: 'engram-vault',
        format: 'json',
        collectionIds: ['collection-a', 'collection-b'],
        includeEmbeddings: false,
      })
      expect(onNotice).toHaveBeenCalledWith('Export ready: engram-export-engram-vault.json')
    })
  })

  it('exports with include embeddings enabled', async () => {
    await submitExport({ includeEmbeddings: true })

    await waitFor(() => {
      expect(exportApiMocks.exportProjectBundle).toHaveBeenCalledWith(
        expect.objectContaining({
          includeEmbeddings: true,
          collectionIds: [],
        }),
      )
    })
  })

  it('exports ZIP format', async () => {
    await submitExport({ format: 'zip' })

    await waitFor(() => {
      expect(exportApiMocks.exportProjectBundle).toHaveBeenCalledWith(
        expect.objectContaining({
          format: 'zip',
        }),
      )
    })
  })

  it('imports bundle and renders summary counts', async () => {
    const user = userEvent.setup()
    const importResponse: ProjectImportResponse = {
      target_project_id: 'target-project',
      imported_engrams: 4,
      skipped_engrams: 1,
      overwritten_engrams: 0,
      imported_collections: 2,
      reused_collections: 1,
      imported_collection_items: 6,
      conflict_policy: 'rename',
    }
    exportApiMocks.importProjectBundle.mockResolvedValueOnce(importResponse)
    const { onNotice } = renderTransferPage()

    await user.clear(screen.getByTestId('import-project-id'))
    await user.type(screen.getByTestId('import-project-id'), 'target-project')
    await user.selectOptions(screen.getByTestId('import-conflict-policy'), 'rename')

    const fileInput = screen.getByTestId('import-bundle-file')
    const file = new File(['{"schema_version":"1.0"}'], 'export.json', {
      type: 'application/json',
    })
    await user.upload(fileInput, file)
    await user.click(screen.getByRole('button', { name: 'Import Bundle' }))

    await waitFor(() => {
      expect(exportApiMocks.importProjectBundle).toHaveBeenCalledWith({
        projectId: 'target-project',
        file,
        conflictPolicy: 'rename',
      })
      expect(onNotice).toHaveBeenCalledWith('Import completed: 4 engrams into target-project')
    })

    expect(screen.getByTestId('import-result-summary')).toBeInTheDocument()
    expect(screen.getByText('Imported engrams: 4')).toBeInTheDocument()
    expect(screen.getByText('Conflict policy: rename')).toBeInTheDocument()
  })
})
