import { useEffect, useMemo, useState } from 'react'
import type { ChangeEvent, FormEvent } from 'react'

import styled from 'styled-components'

import { exportProjectBundle, importProjectBundle } from '../api/export'
import type {
  ProjectExportFormat,
  ProjectImportConflictPolicy,
  ProjectImportResponse,
} from '../api/types'
import { describeError } from '../utils/errors'
import {
  ErrorText,
  GlassPane,
  MutedText,
  PaneHeader,
  ScrollColumn,
  SectionDivider,
  SessionMeta,
} from '../styles/primitives'

const TransferLayout = styled.main`
  display: grid;
  grid-template-columns: minmax(300px, 1fr) minmax(300px, 1fr);
  gap: 0.9rem;
  flex: 1;
  min-height: 0;

  @media (max-width: 1180px) {
    grid-template-columns: 1fr;
    overflow: auto;
  }
`

const TransferPane = styled(GlassPane)`
  overflow: auto;
`

const FormStack = styled.form`
  display: grid;
  gap: 0.55rem;
`

const Field = styled.label`
  display: grid;
  gap: 0.28rem;
`

const CheckboxLabel = styled.label`
  display: flex;
  align-items: center;
  gap: 0.45rem;
  font-size: 0.8rem;
  letter-spacing: 0.02em;
  text-transform: none;
  font-weight: 600;
`

const ResultGrid = styled.div`
  display: grid;
  gap: 0.35rem;
  border: 1px dashed var(--color-line);
  border-radius: 10px;
  padding: 0.55rem;
  background: var(--surface-raised);
`

function parseCollectionIdInput(input: string): string[] {
  const values: string[] = []
  const seen = new Set<string>()

  for (const rawValue of input.split(/[\s,]+/)) {
    const normalized = rawValue.trim()
    if (!normalized || seen.has(normalized)) {
      continue
    }
    seen.add(normalized)
    values.push(normalized)
  }

  return values
}

function triggerDownload(blob: Blob, filename: string): void {
  const objectUrl = window.URL.createObjectURL(blob)
  const anchor = document.createElement('a')
  anchor.href = objectUrl
  anchor.download = filename
  document.body.append(anchor)
  anchor.click()
  anchor.remove()
  window.URL.revokeObjectURL(objectUrl)
}

interface ProjectTransferPageProps {
  projectId: string
  onProjectChange: (projectId: string) => void
  onNotice: (message: string) => void
}

export function ProjectTransferPage({
  projectId,
  onProjectChange,
  onNotice,
}: ProjectTransferPageProps) {
  const [exportFormat, setExportFormat] = useState<ProjectExportFormat>('json')
  const [includeEmbeddings, setIncludeEmbeddings] = useState(false)
  const [collectionInput, setCollectionInput] = useState('')
  const [exporting, setExporting] = useState(false)
  const [exportError, setExportError] = useState<string | null>(null)

  const [importProjectId, setImportProjectId] = useState(projectId)
  const [conflictPolicy, setConflictPolicy] =
    useState<ProjectImportConflictPolicy>('skip')
  const [importFile, setImportFile] = useState<File | null>(null)
  const [importing, setImporting] = useState(false)
  const [importError, setImportError] = useState<string | null>(null)
  const [importResult, setImportResult] = useState<ProjectImportResponse | null>(null)

  useEffect(() => {
    setImportProjectId((current) => (current.trim().length > 0 ? current : projectId))
  }, [projectId])

  const selectedCollectionIds = useMemo(
    () => parseCollectionIdInput(collectionInput),
    [collectionInput],
  )

  const handleExport = async (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault()
    const normalizedProjectId = projectId.trim()
    if (!normalizedProjectId) {
      setExportError('Workspace project ID is required for export.')
      return
    }

    setExportError(null)
    setExporting(true)
    try {
      const bundle = await exportProjectBundle({
        projectId: normalizedProjectId,
        format: exportFormat,
        collectionIds: selectedCollectionIds,
        includeEmbeddings,
      })
      triggerDownload(bundle.blob, bundle.filename)
      onNotice(`Export ready: ${bundle.filename}`)
    } catch (error) {
      setExportError(describeError(error))
    } finally {
      setExporting(false)
    }
  }

  const handleImportFileChange = (event: ChangeEvent<HTMLInputElement>) => {
    setImportFile(event.target.files?.[0] ?? null)
  }

  const handleImport = async (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault()
    const normalizedProjectId = importProjectId.trim()

    if (!normalizedProjectId) {
      setImportError('Target project ID is required for import.')
      return
    }
    if (!importFile) {
      setImportError('Select an export bundle file (.json or .zip).')
      return
    }

    setImportError(null)
    setImporting(true)
    try {
      const result = await importProjectBundle({
        projectId: normalizedProjectId,
        file: importFile,
        conflictPolicy,
      })
      setImportResult(result)
      onNotice(
        `Import completed: ${result.imported_engrams} engrams into ${result.target_project_id}`,
      )
    } catch (error) {
      setImportError(describeError(error))
    } finally {
      setImporting(false)
    }
  }

  return (
    <TransferLayout data-testid="project-transfer-page">
      <TransferPane>
        <PaneHeader>
          <h2 className="font-display text-base font-semibold tracking-[0.02em] text-ink">
            Project Export
          </h2>
        </PaneHeader>
        <MutedText>
          Create a portable memory bundle for this project. Leave collection IDs empty to
          export the full project.
        </MutedText>
        <SectionDivider />
        <FormStack onSubmit={(event) => void handleExport(event)}>
          <Field>
            <span>Workspace Project ID</span>
            <input
              value={projectId}
              onChange={(event) => onProjectChange(event.target.value)}
              placeholder="engram-vault"
              data-testid="export-project-id"
            />
          </Field>
          <Field>
            <span>Bundle Format</span>
            <select
              value={exportFormat}
              onChange={(event) =>
                setExportFormat(event.target.value as ProjectExportFormat)
              }
              data-testid="export-format"
            >
              <option value="json">JSON</option>
              <option value="zip">ZIP</option>
            </select>
          </Field>
          <Field>
            <span>Collection IDs (optional)</span>
            <textarea
              rows={3}
              value={collectionInput}
              onChange={(event) => setCollectionInput(event.target.value)}
              placeholder="UUIDs separated by commas or whitespace"
              data-testid="export-collection-ids"
            />
            <SessionMeta>
              {selectedCollectionIds.length === 0
                ? 'Full project export selected.'
                : `${selectedCollectionIds.length} collection IDs selected.`}
            </SessionMeta>
          </Field>
          <CheckboxLabel>
            <input
              type="checkbox"
              checked={includeEmbeddings}
              onChange={(event) => setIncludeEmbeddings(event.target.checked)}
            />
            Include embeddings (optional)
          </CheckboxLabel>
          <button type="submit" disabled={exporting}>
            {exporting ? 'Exporting...' : 'Export Bundle'}
          </button>
          {exportError ? <ErrorText>{exportError}</ErrorText> : null}
        </FormStack>
      </TransferPane>

      <TransferPane>
        <PaneHeader>
          <h2 className="font-display text-base font-semibold tracking-[0.02em] text-ink">
            Project Import
          </h2>
        </PaneHeader>
        <MutedText>
          Import a JSON or ZIP export bundle into a target project with deterministic conflict
          handling.
        </MutedText>
        <SectionDivider />
        <FormStack onSubmit={(event) => void handleImport(event)}>
          <Field>
            <span>Target Project ID</span>
            <input
              value={importProjectId}
              onChange={(event) => setImportProjectId(event.target.value)}
              placeholder="engram-vault"
              data-testid="import-project-id"
            />
          </Field>
          <Field>
            <span>Conflict Policy</span>
            <select
              value={conflictPolicy}
              onChange={(event) =>
                setConflictPolicy(event.target.value as ProjectImportConflictPolicy)
              }
              data-testid="import-conflict-policy"
            >
              <option value="skip">Skip duplicates</option>
              <option value="overwrite">Overwrite duplicates</option>
              <option value="rename">Rename duplicates</option>
            </select>
          </Field>
          <Field>
            <span>Bundle File</span>
            <input
              type="file"
              accept=".json,.zip,application/json,application/zip"
              onChange={handleImportFileChange}
              data-testid="import-bundle-file"
            />
            <SessionMeta>
              {importFile ? importFile.name : 'No file selected.'}
            </SessionMeta>
          </Field>
          <button type="submit" disabled={importing}>
            {importing ? 'Importing...' : 'Import Bundle'}
          </button>
          {importError ? <ErrorText>{importError}</ErrorText> : null}
        </FormStack>
        {importResult ? (
          <>
            <SectionDivider />
            <ScrollColumn>
              <ResultGrid data-testid="import-result-summary">
                <p className="font-display text-sm font-semibold text-ink">
                  Import Summary
                </p>
                <MutedText>Target: {importResult.target_project_id}</MutedText>
                <MutedText>Imported engrams: {importResult.imported_engrams}</MutedText>
                <MutedText>Skipped engrams: {importResult.skipped_engrams}</MutedText>
                <MutedText>
                  Overwritten engrams: {importResult.overwritten_engrams}
                </MutedText>
                <MutedText>
                  Imported collections: {importResult.imported_collections}
                </MutedText>
                <MutedText>Reused collections: {importResult.reused_collections}</MutedText>
                <MutedText>
                  Imported collection items: {importResult.imported_collection_items}
                </MutedText>
                <MutedText>Conflict policy: {importResult.conflict_policy}</MutedText>
              </ResultGrid>
            </ScrollColumn>
          </>
        ) : null}
      </TransferPane>
    </TransferLayout>
  )
}
