import { render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
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

vi.mock('../api/export', () => ({
  exportProjectBundle: exportApiMocks.exportProjectBundle,
  importProjectBundle: exportApiMocks.importProjectBundle,
}))

function renderTransferPage(overrides: Partial<{ projectId: string }> = {}) {
  const onProjectChange = vi.fn()
  const onNotice = vi.fn()

  render(
    <ThemeProvider theme={lightTheme}>
      <ProjectTransferPage
        projectId={overrides.projectId ?? 'engram-vault'}
        onProjectChange={onProjectChange}
        onNotice={onNotice}
      />
    </ThemeProvider>,
  )

  return {
    onProjectChange,
    onNotice,
  }
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

  it('renders export and import sections', () => {
    renderTransferPage()

    expect(screen.getByText('Project Export')).toBeInTheDocument()
    expect(screen.getByText('Project Import')).toBeInTheDocument()
    expect(screen.getByTestId('project-transfer-page')).toBeInTheDocument()
  })

  it('exports bundle using selected options and shows notice', async () => {
    const user = userEvent.setup()
    const { onNotice } = renderTransferPage()

    await user.selectOptions(screen.getByTestId('export-format'), 'zip')
    await user.type(
      screen.getByTestId('export-collection-ids'),
      'collection-a, collection-b  collection-a',
    )
    await user.click(screen.getByLabelText('Include embeddings (optional)'))
    await user.click(screen.getByRole('button', { name: 'Export Bundle' }))

    await waitFor(() => {
      expect(exportApiMocks.exportProjectBundle).toHaveBeenCalledWith({
        projectId: 'engram-vault',
        format: 'zip',
        collectionIds: ['collection-a', 'collection-b'],
        includeEmbeddings: true,
      })
      expect(onNotice).toHaveBeenCalledWith(
        'Export ready: engram-export-engram-vault.json',
      )
    })
  })

  it('imports bundle and renders summary counts', async () => {
    const user = userEvent.setup()
    const importResponse: ProjectImportResponse = {
      target_project_id: 'phase33-target',
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
    await user.type(screen.getByTestId('import-project-id'), 'phase33-target')
    await user.selectOptions(screen.getByTestId('import-conflict-policy'), 'rename')

    const fileInput = screen.getByTestId('import-bundle-file')
    const file = new File(['{"schema_version":"1.0"}'], 'export.json', {
      type: 'application/json',
    })
    await user.upload(fileInput, file)
    await user.click(screen.getByRole('button', { name: 'Import Bundle' }))

    await waitFor(() => {
      expect(exportApiMocks.importProjectBundle).toHaveBeenCalledWith({
        projectId: 'phase33-target',
        file,
        conflictPolicy: 'rename',
      })
      expect(onNotice).toHaveBeenCalledWith(
        'Import completed: 4 engrams into phase33-target',
      )
    })

    expect(screen.getByTestId('import-result-summary')).toBeInTheDocument()
    expect(screen.getByText('Imported engrams: 4')).toBeInTheDocument()
    expect(screen.getByText('Conflict policy: rename')).toBeInTheDocument()
  })
})
