import { useEffect, useMemo, useState } from 'react'
import type { ChangeEvent, FormEvent, KeyboardEvent } from 'react'

import styled from 'styled-components'

import { exportProjectBundle, importProjectBundle } from '../api/export'
import { listCollections } from '../api/memoryAdmin'
import { listProjects } from '../api/projects'
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
import { SmartIdDropdown } from './SmartIdDropdown'

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

const CollectionPickerRow = styled.div`
  display: grid;
  grid-template-columns: minmax(0, 1fr) auto;
  gap: 0.45rem;
  align-items: start;
`

const ChipRow = styled.div`
  display: flex;
  flex-wrap: wrap;
  gap: 0.35rem;
`

const Chip = styled.span`
  display: inline-flex;
  align-items: center;
  gap: 0.28rem;
  border-radius: 9999px;
  border: 1px solid var(--color-line);
  background: var(--surface-mute);
  font-size: 0.78rem;
  padding: 0.2rem 0.5rem;

  button {
    border: 0;
    border-radius: 9999px;
    background: transparent;
    color: var(--color-ink-muted);
    width: 1rem;
    height: 1rem;
    line-height: 1rem;
    font-weight: 700;
    cursor: pointer;
    padding: 0;
  }
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

function uniqueIds(values: string[]): string[] {
  const unique = new Set<string>()
  const result: string[] = []
  for (const rawValue of values) {
    const normalized = rawValue.trim()
    if (!normalized || unique.has(normalized)) {
      continue
    }
    unique.add(normalized)
    result.push(normalized)
  }
  return result
}

function sortedUniqueIds(values: string[]): string[] {
  return uniqueIds(values).sort((left, right) => left.localeCompare(right))
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
  const [collectionSelectionIds, setCollectionSelectionIds] = useState<string[]>([])
  const [projectSuggestions, setProjectSuggestions] = useState<string[]>([])
  const [collectionSuggestions, setCollectionSuggestions] = useState<string[]>([])
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

  useEffect(() => {
    let cancelled = false
    const normalizedProjectId = projectId.trim()
    if (!normalizedProjectId) {
      setCollectionSuggestions([])
      return () => {
        cancelled = true
      }
    }

    const loadCollectionSuggestions = async () => {
      try {
        const collections = await listCollections({
          project_id: normalizedProjectId,
          include_deleted: false,
          limit: 200,
          offset: 0,
        })
        if (cancelled) {
          return
        }
        setCollectionSuggestions(collections.map((collection) => collection.collection_id))
      } catch {
        if (!cancelled) {
          setCollectionSuggestions([])
        }
      }
    }

    void loadCollectionSuggestions()
    return () => {
      cancelled = true
    }
  }, [projectId])

  const pendingCollectionIds = useMemo(
    () => parseCollectionIdInput(collectionInput),
    [collectionInput],
  )
  const selectedCollectionIds = useMemo(
    () => uniqueIds([...collectionSelectionIds, ...pendingCollectionIds]),
    [collectionSelectionIds, pendingCollectionIds],
  )
  const projectIdOptions = useMemo(
    () =>
      sortedUniqueIds([
        projectId,
        importProjectId,
        ...projectSuggestions,
      ]),
    [importProjectId, projectId, projectSuggestions],
  )
  const collectionIdOptions = useMemo(
    () =>
      sortedUniqueIds([
        ...collectionSuggestions,
        ...collectionSelectionIds,
        ...pendingCollectionIds,
      ]),
    [collectionSelectionIds, collectionSuggestions, pendingCollectionIds],
  )

  const addSelectedCollectionIds = () => {
    const parsedCollectionIds = parseCollectionIdInput(collectionInput)
    if (parsedCollectionIds.length === 0) {
      return
    }
    setCollectionSelectionIds((current) => uniqueIds([...current, ...parsedCollectionIds]))
    setCollectionInput('')
  }

  const removeSelectedCollectionId = (collectionId: string) => {
    setCollectionSelectionIds((current) => current.filter((item) => item !== collectionId))
  }

  const handleCollectionInputKeyDown = (event: KeyboardEvent<HTMLInputElement>) => {
    if (event.key !== 'Enter') {
      return
    }
    event.preventDefault()
    addSelectedCollectionIds()
  }

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
            <SmartIdDropdown
              id="export-project-id-field"
              value={projectId}
              options={projectIdOptions}
              onChange={onProjectChange}
              placeholder="engram-vault"
              inputTestId="export-project-id"
              optionsTestId="export-project-id-options"
              matchCountTestId="export-project-id-matches"
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
            <CollectionPickerRow>
              <SmartIdDropdown
                id="export-collection-id-field"
                value={collectionInput}
                options={collectionIdOptions}
                onChange={setCollectionInput}
                onKeyDown={handleCollectionInputKeyDown}
                placeholder="collection ID or comma-separated IDs"
                inputTestId="export-collection-ids"
                optionsTestId="export-collection-id-options"
                matchCountTestId="export-collection-id-matches"
              />
              <button
                type="button"
                disabled={!collectionInput.trim()}
                onClick={addSelectedCollectionIds}
                data-testid="export-collection-id-add"
              >
                Add
              </button>
            </CollectionPickerRow>
            {collectionSelectionIds.length > 0 ? (
              <ChipRow data-testid="export-selected-collection-ids">
                {collectionSelectionIds.map((collectionId) => (
                  <Chip key={collectionId}>
                    {collectionId}
                    <button
                      type="button"
                      aria-label={`remove collection ${collectionId}`}
                      onClick={() => removeSelectedCollectionId(collectionId)}
                    >
                      ×
                    </button>
                  </Chip>
                ))}
              </ChipRow>
            ) : null}
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
            <SmartIdDropdown
              id="import-project-id-field"
              value={importProjectId}
              options={projectIdOptions}
              onChange={setImportProjectId}
              placeholder="engram-vault"
              inputTestId="import-project-id"
              optionsTestId="import-project-id-options"
              matchCountTestId="import-project-id-matches"
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
