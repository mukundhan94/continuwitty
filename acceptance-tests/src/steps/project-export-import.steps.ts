import { readFile } from 'node:fs/promises'

import type { APIResponse, Page } from '@playwright/test'

import { Then, When, expect } from '../support/fixtures'
import { acceptanceEnv } from '../support/env'

type ExportBundle = {
  project: {
    project_id: string
  }
  include_embeddings: boolean
  selected_collection_ids: string[]
  engrams: Array<{
    engram_id: string
    title: string
  }>
}

type ImportSummary = {
  target_project_id: string
  imported_engrams: number
  skipped_engrams: number
  overwritten_engrams: number
  imported_collections: number
  reused_collections: number
  imported_collection_items: number
  conflict_policy: 'skip' | 'overwrite' | 'rename'
}

type ExportDataset = {
  projectId: string
  engramIds: [string, string]
  collectionIds: [string, string]
}

type ImportDataset = {
  sourceProjectId: string
  targetProjectId: string
  duplicateTitle: string
  bundleBody: Buffer
}

let exportDataset: ExportDataset | null = null
let latestExportBundle: ExportBundle | null = null
let latestExportFilename: string | null = null

let importDataset: ImportDataset | null = null
let latestImportSummary: ImportSummary | null = null
let latestTargetTitles: string[] = []

function uniqueId(prefix: string): string {
  return `${prefix}-${Date.now()}-${Math.floor(Math.random() * 10_000)}`
}

async function expectOk(response: APIResponse, label: string): Promise<void> {
  if (response.ok()) {
    return
  }
  throw new Error(`${label} failed (${response.status()}): ${await response.text()}`)
}

async function createProject(page: Page, projectId: string): Promise<void> {
  const response = await page.request.post('/api/v1/projects', {
    data: {
      project_id: projectId,
      name: projectId,
      description: 'acceptance export/import dataset',
    },
  })
  await expectOk(response, 'create project')
}

async function createEngram(
  page: Page,
  params: {
    projectId: string
    title: string
    markdown: string
  },
): Promise<string> {
  const response = await page.request.post('/api/v1/engrams', {
    data: {
      project_id: params.projectId,
      title: params.title,
      abstract: `${params.title} abstract`,
      detailed_summary_markdown: params.markdown,
      tags: ['acceptance'],
      keywords: ['export', 'import'],
      visibility_scope: 'project',
    },
  })
  await expectOk(response, 'create engram')
  const body = (await response.json()) as { engram_id: string }
  return body.engram_id
}

async function createCollection(
  page: Page,
  params: { projectId: string; name: string },
): Promise<string> {
  const response = await page.request.post('/api/v1/admin/memory/collections', {
    data: {
      project_id: params.projectId,
      name: params.name,
      description: 'acceptance collection',
    },
  })
  await expectOk(response, 'create collection')
  const body = (await response.json()) as { collection_id: string }
  return body.collection_id
}

async function addCollectionItem(
  page: Page,
  params: { collectionId: string; engramId: string },
): Promise<void> {
  const response = await page.request.post(
    `/api/v1/admin/memory/collections/${params.collectionId}/items`,
    {
      data: { engram_ids: [params.engramId] },
    },
  )
  await expectOk(response, 'add collection item')
}

async function listEngramTitles(page: Page, projectId: string): Promise<string[]> {
  const response = await page.request.get('/api/v1/engrams', {
    params: {
      project_id: projectId,
      limit: '200',
      offset: '0',
    },
  })
  await expectOk(response, 'list engrams')
  const body = (await response.json()) as Array<{ title: string }>
  return body.map((item) => item.title)
}

async function exportBundleViaApi(page: Page, projectId: string): Promise<Buffer> {
  const response = await page.request.get(`/api/v1/projects/${projectId}/export`, {
    params: { format: 'json' },
  })
  await expectOk(response, 'export source project')
  return Buffer.from(await response.body())
}

async function ensureProjectTransferPage(page: Page): Promise<void> {
  const transferPage = page.getByTestId('project-transfer-page')
  const onTransferPage = await transferPage
    .isVisible({ timeout: 300 })
    .catch(() => false)
  if (!onTransferPage) {
    const base = acceptanceEnv.webBaseUrl.endsWith('/')
      ? acceptanceEnv.webBaseUrl
      : `${acceptanceEnv.webBaseUrl}/`
    await page.goto(new URL('/app/projects/transfer/export', base).toString(), {
      waitUntil: 'domcontentloaded',
    })
  }
  await expect(transferPage).toBeVisible()
}

async function setWorkspaceProject(page: Page, projectId: string): Promise<void> {
  const showCreator = page.getByRole('button', { name: /Show Creator/i })
  if (await showCreator.isVisible().catch(() => false)) {
    await showCreator.click()
  }
  await page.getByTestId('session-project-id-input').fill(projectId)
  await expect(page.getByTestId('session-project-id-input')).toHaveValue(projectId)
}

async function datalistOptionValues(page: Page, listSelector: string): Promise<string[]> {
  return page.locator(`${listSelector} option`).evaluateAll((nodes) =>
    nodes
      .map((node) => (node as HTMLOptionElement).value.trim())
      .filter((value) => value.length > 0),
  )
}

async function downloadExportFromUi(page: Page): Promise<{ bundle: ExportBundle; filename: string }> {
  const downloadPromise = page.waitForEvent('download')
  await page.getByRole('button', { name: /^Export Bundle$/i }).click()
  const download = await downloadPromise
  const downloadPath = await download.path()
  if (!downloadPath) {
    throw new Error('Missing downloaded export file path')
  }
  const raw = await readFile(downloadPath, 'utf-8')
  return {
    bundle: JSON.parse(raw) as ExportBundle,
    filename: download.suggestedFilename(),
  }
}

async function setIncludeEmbeddingsToggle(page: Page, enabled: boolean): Promise<void> {
  const checkbox = page.getByLabel('Include embeddings (optional)')
  const checked = await checkbox.isChecked()
  if (checked !== enabled) {
    await checkbox.click()
  }
}

async function prepareExportForm(
  page: Page,
  params: {
    projectId: string
    format: 'json' | 'zip'
    collectionIdsText: string
    includeEmbeddings: boolean
  },
): Promise<void> {
  await page.getByTestId('export-project-id').fill(params.projectId)
  await page.getByTestId('export-format').selectOption(params.format)
  await page.getByTestId('export-collection-ids').fill(params.collectionIdsText)
  await setIncludeEmbeddingsToggle(page, params.includeEmbeddings)
}

async function importBundleFromUi(
  page: Page,
  params: { targetProjectId: string; policy: 'skip' | 'overwrite' | 'rename'; bundleBody: Buffer },
): Promise<ImportSummary> {
  await page.getByTestId('import-project-id').fill(params.targetProjectId)
  await page.getByTestId('import-conflict-policy').selectOption(params.policy)
  await page.getByTestId('import-bundle-file').setInputFiles({
    name: `export-${params.policy}.json`,
    mimeType: 'application/json',
    buffer: params.bundleBody,
  })

  const responsePromise = page.waitForResponse((response) => {
    return (
      response.request().method() === 'POST' &&
      response.url().includes(`/api/v1/projects/${params.targetProjectId}/import`)
    )
  })

  await page.getByRole('button', { name: /^Import Bundle$/i }).click()
  const response = await responsePromise
  expect(response.ok()).toBeTruthy()
  return (await response.json()) as ImportSummary
}

When('I open the project transfer page', async ({ page }) => {
  await ensureProjectTransferPage(page)
})

When('I prepare a project dataset for export tests', async ({ page }) => {
  const projectId = uniqueId('export-src')
  await createProject(page, projectId)

  const engramA = await createEngram(page, {
    projectId,
    title: 'Export Alpha',
    markdown: 'Markdown A',
  })
  const engramB = await createEngram(page, {
    projectId,
    title: 'Export Beta',
    markdown: 'Markdown B',
  })

  const collectionA = await createCollection(page, {
    projectId,
    name: uniqueId('collection-a'),
  })
  const collectionB = await createCollection(page, {
    projectId,
    name: uniqueId('collection-b'),
  })
  await addCollectionItem(page, { collectionId: collectionA, engramId: engramA })
  await addCollectionItem(page, { collectionId: collectionB, engramId: engramB })

  exportDataset = {
    projectId,
    engramIds: [engramA, engramB],
    collectionIds: [collectionA, collectionB],
  }
  await setWorkspaceProject(page, projectId)
  latestExportBundle = null
  latestExportFilename = null
})

Then(
  'the transfer form should expose search suggestions for the prepared dataset',
  async ({ page }) => {
    if (!exportDataset) {
      throw new Error('Missing export dataset')
    }

    await expect.poll(async () => datalistOptionValues(page, '#export-project-id-field-list')).toEqual(
      expect.arrayContaining([exportDataset.projectId]),
    )
    await expect.poll(async () => datalistOptionValues(page, '#export-collection-id-field-list')).toEqual(
      expect.arrayContaining(exportDataset.collectionIds),
    )
  },
)

When('I export the prepared project as JSON', async ({ page }) => {
  if (!exportDataset) {
    throw new Error('Missing export dataset')
  }
  await prepareExportForm(page, {
    projectId: exportDataset.projectId,
    format: 'json',
    collectionIdsText: '',
    includeEmbeddings: false,
  })

  const exported = await downloadExportFromUi(page)
  latestExportBundle = exported.bundle
  latestExportFilename = exported.filename
})

Then(
  'the exported bundle should include full project data with embeddings disabled',
  async () => {
    if (!exportDataset || !latestExportBundle) {
      throw new Error('Missing export assertions context')
    }

    expect(latestExportFilename?.endsWith('.json')).toBe(true)
    expect(latestExportBundle.project.project_id).toBe(exportDataset.projectId)
    expect(latestExportBundle.include_embeddings).toBe(false)
    expect(new Set(latestExportBundle.selected_collection_ids)).toEqual(
      new Set(exportDataset.collectionIds),
    )
    expect(new Set(latestExportBundle.engrams.map((item) => item.engram_id))).toEqual(
      new Set(exportDataset.engramIds),
    )
  },
)

When(
  'I export the prepared project using only the first collection ID',
  async ({ page }) => {
    if (!exportDataset) {
      throw new Error('Missing export dataset')
    }
    const [collectionId] = exportDataset.collectionIds
    await prepareExportForm(page, {
      projectId: exportDataset.projectId,
      format: 'json',
      collectionIdsText: collectionId,
      includeEmbeddings: false,
    })

    const exported = await downloadExportFromUi(page)
    latestExportBundle = exported.bundle
    latestExportFilename = exported.filename
  },
)

Then(
  'the exported bundle should only include engrams from the selected collection',
  async () => {
    if (!exportDataset || !latestExportBundle) {
      throw new Error('Missing export assertions context')
    }
    expect(latestExportBundle.selected_collection_ids).toEqual([
      exportDataset.collectionIds[0],
    ])
    expect(latestExportBundle.engrams.map((item) => item.engram_id)).toEqual([
      exportDataset.engramIds[0],
    ])
  },
)

When('I export the prepared project with embeddings enabled', async ({ page }) => {
  if (!exportDataset) {
    throw new Error('Missing export dataset')
  }
  await prepareExportForm(page, {
    projectId: exportDataset.projectId,
    format: 'json',
    collectionIdsText: '',
    includeEmbeddings: true,
  })

  const exported = await downloadExportFromUi(page)
  latestExportBundle = exported.bundle
  latestExportFilename = exported.filename
})

Then('the exported bundle should mark embeddings as included', async () => {
  if (!latestExportBundle) {
    throw new Error('Missing export bundle')
  }
  expect(latestExportBundle.include_embeddings).toBe(true)
})

When('I export the prepared project as ZIP', async ({ page }) => {
  if (!exportDataset) {
    throw new Error('Missing export dataset')
  }
  await prepareExportForm(page, {
    projectId: exportDataset.projectId,
    format: 'zip',
    collectionIdsText: '',
    includeEmbeddings: false,
  })

  const downloadPromise = page.waitForEvent('download')
  await page.getByRole('button', { name: /^Export Bundle$/i }).click()
  const download = await downloadPromise
  latestExportFilename = download.suggestedFilename()
})

Then('the exported download should be a ZIP bundle', async () => {
  expect(latestExportFilename?.endsWith('.zip')).toBe(true)
})

When(
  'I prepare source and target projects for import policy {string}',
  async ({ page }, policy: 'skip' | 'overwrite' | 'rename') => {
    const sourceProjectId = uniqueId(`import-src-${policy}`)
    const targetProjectId = uniqueId(`import-dst-${policy}`)
    const duplicateTitle = `Collision ${policy}`
    const duplicateMarkdown = `Shared markdown ${policy}`

    await createProject(page, sourceProjectId)
    await createProject(page, targetProjectId)

    const sourceEngram = await createEngram(page, {
      projectId: sourceProjectId,
      title: duplicateTitle,
      markdown: duplicateMarkdown,
    })
    const sourceCollection = await createCollection(page, {
      projectId: sourceProjectId,
      name: uniqueId(`import-collection-${policy}`),
    })
    await addCollectionItem(page, {
      collectionId: sourceCollection,
      engramId: sourceEngram,
    })

    await createEngram(page, {
      projectId: targetProjectId,
      title: duplicateTitle,
      markdown: duplicateMarkdown,
    })

    importDataset = {
      sourceProjectId,
      targetProjectId,
      duplicateTitle,
      bundleBody: await exportBundleViaApi(page, sourceProjectId),
    }
    latestImportSummary = null
    latestTargetTitles = []
  },
)

When(
  'I import the prepared bundle with policy {string}',
  async ({ page }, policy: 'skip' | 'overwrite' | 'rename') => {
    if (!importDataset) {
      throw new Error('Missing import dataset')
    }
    latestImportSummary = await importBundleFromUi(page, {
      targetProjectId: importDataset.targetProjectId,
      policy,
      bundleBody: importDataset.bundleBody,
    })
    await expect(page.getByTestId('import-result-summary')).toBeVisible()
    latestTargetTitles = await listEngramTitles(page, importDataset.targetProjectId)
  },
)

Then(
  'the import result should match {string} policy expectations',
  async ({}, policy: 'skip' | 'overwrite' | 'rename') => {
    if (!importDataset || !latestImportSummary) {
      throw new Error('Missing import assertions context')
    }
    const dataset = importDataset

    expect(latestImportSummary.conflict_policy).toBe(policy)
    expect(latestImportSummary.imported_collections).toBeGreaterThanOrEqual(1)

    if (policy === 'skip') {
      expect(latestImportSummary.imported_engrams).toBe(0)
      expect(latestImportSummary.skipped_engrams).toBe(1)
      expect(latestImportSummary.overwritten_engrams).toBe(0)
      expect(
        latestTargetTitles.filter((title) => title === dataset.duplicateTitle),
      ).toHaveLength(1)
      return
    }

    if (policy === 'rename') {
      expect(latestImportSummary.imported_engrams).toBe(1)
      expect(latestImportSummary.skipped_engrams).toBe(0)
      expect(latestImportSummary.overwritten_engrams).toBe(0)
      expect(latestTargetTitles).toContain(dataset.duplicateTitle)
      expect(
        latestTargetTitles.some((title) =>
          title.startsWith(`${dataset.duplicateTitle} (imported`),
        ),
      ).toBe(true)
      return
    }

    expect(latestImportSummary.imported_engrams).toBe(1)
    expect(latestImportSummary.skipped_engrams).toBe(0)
    expect(latestImportSummary.overwritten_engrams).toBe(1)
    expect(
      latestTargetTitles.filter((title) => title === dataset.duplicateTitle),
    ).toHaveLength(1)
    expect(
      latestTargetTitles.some((title) =>
        title.startsWith(`${dataset.duplicateTitle} (imported`),
      ),
    ).toBe(false)
  },
)
