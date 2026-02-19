import { randomUUID } from 'node:crypto'

import { Then, When, expect } from '../support/fixtures'

type McpFrame = {
  result?: {
    structuredContent?: {
      engram?: {
        engram_id: string
        resolved_project_id?: string
        used_default_project?: boolean
      }
    }
  }
  error?: {
    code: number
    message: string
    data?: Record<string, unknown>
  }
}

let phase31ProjectId: string | null = null
let mcpCreatedEngram: {
  engram_id: string
  resolved_project_id?: string
  used_default_project?: boolean
} | null = null
let lifecycleSessionId: string | null = null
let lifecycleEngramId: string | null = null

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

When('I configure a default project for the phase31 scenario', async ({ page }) => {
  phase31ProjectId = `phase31-default-${Date.now()}`
  const createProject = await page.request.post('/api/v1/projects', {
    data: {
      project_id: phase31ProjectId,
      name: phase31ProjectId,
      description: 'Phase31 acceptance project',
    },
  })
  expect(createProject.ok()).toBeTruthy()

  const setDefault = await page.request.patch('/api/v1/projects/default', {
    data: { project_id: phase31ProjectId },
  })
  expect(setDefault.ok()).toBeTruthy()
})

When('I create an engram through MCP without project_id', async ({ page }) => {
  const response = await page.request.post('/api/v1/mcp/stream', {
    data: {
      jsonrpc: '2.0',
      id: `phase31-mcp-${Date.now()}`,
      method: 'tools/call',
      params: {
        name: 'engram.create',
        arguments: {
          title: 'Phase31 MCP default project create',
          abstract: 'project omitted intentionally',
          detailed_summary_markdown: 'MCP should resolve via default project.',
        },
      },
    },
  })
  expect(response.ok()).toBeTruthy()
  const body = await response.text()
  const frames = parseMcpFrames(body)
  const finalFrame = finalResultFrame(frames)
  if (finalFrame.error) {
    throw new Error(`MCP create failed: ${JSON.stringify(finalFrame.error)}`)
  }
  mcpCreatedEngram = finalFrame.result?.structuredContent?.engram ?? null
})

Then('the MCP response should report default-project resolution', async () => {
  expect(phase31ProjectId).toBeTruthy()
  expect(mcpCreatedEngram?.engram_id).toBeTruthy()
  expect(mcpCreatedEngram?.resolved_project_id).toBe(phase31ProjectId)
  expect(mcpCreatedEngram?.used_default_project).toBe(true)
})

When('I create a session and a linked engram for admin lifecycle checks', async ({ page }) => {
  const projectId = `phase31-admin-${Date.now()}`
  const createSession = await page.request.post('/api/v1/chat/sessions', {
    data: {
      project_id: projectId,
      title: 'Phase31 Admin Lifecycle Session',
      provider: 'openai',
      model_id: 'gpt-4o-mini',
      system_prompt: '',
      visibility_scope: 'private',
      autosave_enabled: false,
      autosave_strategy: 'off',
      autosave_interval_minutes: 30,
      autosave_min_messages: 6,
      retention_days: 30,
      retention_max_snapshots: 60,
    },
  })
  expect(createSession.ok()).toBeTruthy()
  const sessionBody = (await createSession.json()) as { session_id: string }
  lifecycleSessionId = sessionBody.session_id

  const createEngram = await page.request.post('/api/v1/engrams', {
    data: {
      project_id: projectId,
      thread_id: `phase31-thread-${randomUUID()}`,
      title: 'Phase31 linked engram',
      abstract: 'linked to source session',
      detailed_summary_markdown: 'Linked engram markdown',
      tags: ['phase31'],
      keywords: ['admin'],
      source_session_id: lifecycleSessionId,
      visibility_scope: 'project',
    },
  })
  expect(createEngram.ok()).toBeTruthy()
  const engramBody = (await createEngram.json()) as { engram_id: string }
  lifecycleEngramId = engramBody.engram_id
})

When('I soft-delete that session without deleting linked engrams', async ({ page }) => {
  if (!lifecycleSessionId) {
    throw new Error('Missing lifecycle session id')
  }
  const deleted = await page.request.delete(`/api/v1/admin/memory/sessions/${lifecycleSessionId}`, {
    data: { delete_linked_engrams: false, reason: 'phase31-acceptance' },
  })
  expect(deleted.ok()).toBeTruthy()
})

Then('I should be able to restore the session and still query the linked engram', async ({ page }) => {
  if (!lifecycleSessionId || !lifecycleEngramId) {
    throw new Error('Missing lifecycle identifiers')
  }

  const restore = await page.request.post(`/api/v1/admin/memory/sessions/${lifecycleSessionId}/restore`)
  expect(restore.ok()).toBeTruthy()

  const listLinked = await page.request.get(`/api/v1/admin/memory/engrams?session_id=${lifecycleSessionId}`)
  expect(listLinked.ok()).toBeTruthy()
  const engrams = (await listLinked.json()) as Array<{ engram_id: string; deleted_at: string | null }>
  const linked = engrams.find((item) => item.engram_id === lifecycleEngramId)
  expect(linked).toBeTruthy()
  expect(linked?.deleted_at).toBeNull()
})
