import { When, Then, expect } from '../support/fixtures'
import type { Page } from '@playwright/test'

type McpFrame = {
  result?: {
    structuredContent?: {
      engram?: { engram_id: string }
      enrichment_report?: {
        enrichment_applied?: boolean
        abstract_derived?: boolean
        auto_tags?: string[]
        auto_keywords?: string[]
      }
    }
  }
  error?: { code: number; message: string }
}

let latestEngramId: string | null = null
let latestProjectId: string | null = null
let latestEnrichmentReport: {
  enrichment_applied?: boolean
  abstract_derived?: boolean
  auto_tags?: string[]
  auto_keywords?: string[]
} | null = null
let expectedExplicit: {
  abstract: string
  tags: string[]
  keywords: string[]
} | null = null

function parseMcpFrames(rawSse: string): McpFrame[] {
  const frames: McpFrame[] = []
  for (const line of rawSse.split('\n')) {
    if (!line.startsWith('data: ')) {
      continue
    }
    frames.push(JSON.parse(line.slice(6)) as McpFrame)
  }
  return frames
}

function finalResultFrame(frames: McpFrame[]): McpFrame {
  const resultFrames = frames.filter((frame) => frame.result)
  if (resultFrames.length === 0) {
    throw new Error(`No MCP result frame found: ${JSON.stringify(frames)}`)
  }
  return resultFrames[resultFrames.length - 1]
}

async function callMcpToolsCall(
  page: Page,
  name: string,
  args: Record<string, unknown>,
): Promise<McpFrame> {
  const response = await page.request.post('/api/v1/mcp/stream', {
    data: {
      jsonrpc: '2.0',
      id: `acceptance-${Date.now()}`,
      method: 'tools/call',
      params: {
        name,
        arguments: args,
      },
    },
  })

  expect(response.ok()).toBeTruthy()
  const body = await response.text()
  const frames = parseMcpFrames(body)
  const finalFrame = finalResultFrame(frames)
  if (finalFrame.error) {
    throw new Error(`MCP tool call failed: ${JSON.stringify(finalFrame.error)}`)
  }
  return finalFrame
}

When('I persist a conversation through MCP with empty metadata fields', async ({ page }) => {
  latestProjectId = `project-mcp-auto-meta-${Date.now()}`
  expectedExplicit = null

  const frame = await callMcpToolsCall(page, 'engram.create_from_conversation', {
    project_id: latestProjectId,
    conversation_markdown:
      '## USER\\nSummarize incident impact.\\n\\n## ASSISTANT\\nOutage blast radius dropped after rollback and queue drain while support prepared customer updates.',
    title: 'Acceptance auto metadata seed',
    abstract: '',
    tags: [],
    keywords: [],
    visibility_scope: 'project',
  })

  const structured = frame.result?.structuredContent
  latestEngramId = structured?.engram?.engram_id ?? null
  latestEnrichmentReport = structured?.enrichment_report ?? null

  expect(latestEngramId).toBeTruthy()
})

Then('MCP enrichment report should indicate auto metadata was applied', async ({}) => {
  expect(latestEnrichmentReport?.enrichment_applied).toBe(true)
  expect(latestEnrichmentReport?.abstract_derived).toBe(true)
  expect((latestEnrichmentReport?.auto_tags || []).length).toBeGreaterThan(0)
  expect((latestEnrichmentReport?.auto_keywords || []).length).toBeGreaterThan(0)
})

Then('the persisted engram should expose generated abstract tags and keywords', async ({ page }) => {
  if (!latestProjectId || !latestEngramId) {
    throw new Error('Missing MCP-created engram context')
  }

  const response = await page.request.get(`/api/v1/engrams?project_id=${encodeURIComponent(latestProjectId)}`)
  expect(response.ok()).toBeTruthy()
  const rows = (await response.json()) as Array<{
    engram_id: string
    abstract: string
    tags: string[]
    keywords: string[]
  }>
  const created = rows.find((row) => row.engram_id === latestEngramId)

  expect(created).toBeTruthy()
  expect((created?.abstract || '').trim().length).toBeGreaterThan(0)
  expect((created?.tags || []).length).toBeGreaterThan(0)
  expect((created?.keywords || []).length).toBeGreaterThan(0)
})

When('I persist a conversation through MCP with explicit metadata fields', async ({ page }) => {
  latestProjectId = `project-mcp-explicit-meta-${Date.now()}`
  expectedExplicit = {
    abstract: 'Manual abstract from acceptance scenario',
    tags: ['manual-tag'],
    keywords: ['manual-keyword'],
  }

  const frame = await callMcpToolsCall(page, 'engram.create_from_conversation', {
    project_id: latestProjectId,
    conversation_markdown: '## ASSISTANT\\nRelease summary content.',
    title: 'Acceptance explicit metadata seed',
    abstract: expectedExplicit.abstract,
    tags: expectedExplicit.tags,
    keywords: expectedExplicit.keywords,
    visibility_scope: 'project',
  })

  const structured = frame.result?.structuredContent
  latestEngramId = structured?.engram?.engram_id ?? null
  latestEnrichmentReport = structured?.enrichment_report ?? null

  expect(latestEngramId).toBeTruthy()
})

Then('MCP enrichment report should indicate no metadata overwrite', async ({}) => {
  expect(latestEnrichmentReport?.enrichment_applied).toBe(false)
  expect(latestEnrichmentReport?.abstract_derived).toBe(false)
  expect(latestEnrichmentReport?.auto_tags || []).toEqual([])
  expect(latestEnrichmentReport?.auto_keywords || []).toEqual([])
})

Then('the persisted engram should keep caller abstract tags and keywords', async ({ page }) => {
  if (!latestProjectId || !latestEngramId || !expectedExplicit) {
    throw new Error('Missing explicit metadata scenario context')
  }

  const response = await page.request.get(`/api/v1/engrams?project_id=${encodeURIComponent(latestProjectId)}`)
  expect(response.ok()).toBeTruthy()
  const rows = (await response.json()) as Array<{
    engram_id: string
    abstract: string
    tags: string[]
    keywords: string[]
  }>
  const created = rows.find((row) => row.engram_id === latestEngramId)

  expect(created).toBeTruthy()
  expect(created?.abstract).toBe(expectedExplicit.abstract)
  expect(created?.tags).toEqual(expectedExplicit.tags)
  expect(created?.keywords).toEqual(expectedExplicit.keywords)
})
