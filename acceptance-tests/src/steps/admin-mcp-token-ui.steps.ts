import type { Locator, Page } from '@playwright/test'

import { When, Then, expect, signInWithCredentials, waitForAppShell } from '../support/fixtures'
import { acceptanceEnv } from '../support/env'

type OptionSnapshot = {
  tools: string[]
  projects: string[]
}

type MockTokenSummary = {
  token_id: string
  name: string
  scope: 'read' | 'write'
  allowed_tools: string[]
  allowed_project_ids: string[]
  token_secret_hint: string
  expires_at: string
  last_used_at: string | null
  revoked_at: string | null
  created_at: string
  is_active: boolean
}

let optionSnapshot: OptionSnapshot = { tools: [], projects: [] }
let mockTokens: MockTokenSummary[] = []
let createdTokenName: string | null = null
let createdViewerUsername: string | null = null
const createdViewerPassword = 'viewerpass123'
const adminUsername = 'admin'
const adminPassword = 'admin123'

async function optionValues(select: Locator): Promise<string[]> {
  return select.evaluate((element) => {
    if (!(element instanceof HTMLSelectElement)) {
      return []
    }
    return [...element.options]
      .map((item) => item.value)
      .filter((value) => value.trim().length > 0)
  })
}

async function ensureAdminSession(page: Page) {
  const meResponse = await page.request.get('/api/v1/me')
  if (meResponse.ok()) {
    const me = (await meResponse.json()) as { role?: string }
    if (me.role === 'admin') {
      return
    }
  }

  const logoutButton = page.getByRole('button', { name: /^Logout$/i })
  if (await logoutButton.isVisible().catch(() => false)) {
    await logoutButton.click()
  }
  await page.context().clearCookies()

  await signInWithCredentials(page, adminUsername, adminPassword)
}

async function installMcpTokenMocks(page: Page): Promise<void> {
  mockTokens = []
  await page.route('**/api/v1/mcp/tokens', async (route) => {
    const request = route.request()
    if (request.method() === 'GET') {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify(mockTokens),
      })
      return
    }
    if (request.method() === 'POST') {
      const payload = (request.postDataJSON() || {}) as {
        name?: string
        scope?: 'read' | 'write'
        allowed_tools?: string[]
        allowed_project_ids?: string[]
      }
      const tokenId = `token-${Date.now()}-${Math.floor(Math.random() * 10_000)}`
      const createdAt = new Date().toISOString()
      const expiresAt = new Date(Date.now() + 90 * 24 * 60 * 60 * 1000).toISOString()
      const summary: MockTokenSummary = {
        token_id: tokenId,
        name: (payload.name || '').trim(),
        scope: payload.scope === 'write' ? 'write' : 'read',
        allowed_tools: Array.isArray(payload.allowed_tools) ? payload.allowed_tools : [],
        allowed_project_ids: Array.isArray(payload.allowed_project_ids) ? payload.allowed_project_ids : [],
        token_secret_hint: 'demo...hint',
        expires_at: expiresAt,
        last_used_at: null,
        revoked_at: null,
        created_at: createdAt,
        is_active: true,
      }
      mockTokens.unshift(summary)
      await route.fulfill({
        status: 201,
        contentType: 'application/json',
        body: JSON.stringify({
          ...summary,
          token: `engram_mcp_${tokenId.replace(/-/g, '')}_secret`,
        }),
      })
      return
    }
    await route.fallback()
  })

  await page.route('**/api/v1/mcp/tokens/*/revoke', async (route) => {
    const request = route.request()
    if (request.method() !== 'POST') {
      await route.fallback()
      return
    }
    const tokenId = request.url().split('/').slice(-2)[0]
    mockTokens = mockTokens.map((item) =>
      item.token_id === tokenId
        ? {
            ...item,
            revoked_at: new Date().toISOString(),
            is_active: false,
          }
        : item,
    )
    const updated = mockTokens.find((item) => item.token_id === tokenId)
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify(updated),
    })
  })
}

function adminTokensRoute(): string {
  const base = acceptanceEnv.webBaseUrl.endsWith('/')
    ? acceptanceEnv.webBaseUrl
    : `${acceptanceEnv.webBaseUrl}/`
  return new URL('/app/admin/tokens', base).toString()
}

When('I open the admin MCP token manager', async ({ page }) => {
  await ensureAdminSession(page)
  await installMcpTokenMocks(page)
  await page.goto(adminTokensRoute(), { waitUntil: 'domcontentloaded' })
  await expect(page.getByTestId('admin-mcp-token-page')).toBeVisible({ timeout: 30000 })
})

Then('I should see selectable MCP tool and project options', async ({ page }) => {
  const toolSelect = page.getByTestId('admin-tool-options')
  const projectSelect = page.getByTestId('admin-project-options')

  await expect(toolSelect).toBeVisible()
  await expect(projectSelect).toBeVisible()
  await expect
    .poll(async () => {
      const tools = await optionValues(toolSelect)
      const projects = await optionValues(projectSelect)
      return tools.length > 0 && projects.length > 0
    })
    .toBe(true)

  const tools = await optionValues(toolSelect)
  const projects = await optionValues(projectSelect)

  expect(tools.length).toBeGreaterThan(0)
  expect(projects.length).toBeGreaterThan(0)

  optionSnapshot = { tools, projects }
})

When('I add tool and project chips and create a new MCP token', async ({ page }) => {
  createdTokenName = `acceptance-admin-token-${Date.now()}`

  const toolCandidate = optionSnapshot.tools.includes('engram.query')
    ? 'engram.query'
    : optionSnapshot.tools[0]
  const projectCandidate = optionSnapshot.projects.includes('engram-vault')
    ? 'engram-vault'
    : optionSnapshot.projects[0]

  await page.getByLabel('Token Name').fill(createdTokenName)
  await page.getByLabel('Scope').selectOption('write')

  await page.getByTestId('admin-tool-options').selectOption(toolCandidate)
  await page.getByTestId('admin-add-tool-chip').click()
  await expect(page.getByLabel(`Remove tool ${toolCandidate}`)).toBeVisible()

  await page.getByTestId('admin-project-options').selectOption(projectCandidate)
  await page.getByTestId('admin-add-project-chip').click()
  await expect(page.getByLabel(`Remove project ${projectCandidate}`)).toBeVisible()

  await page.getByRole('button', { name: /^Create Token$/i }).click()
})

Then('I should see the one-time MCP token value and a persisted token row', async ({ page }) => {
  if (!createdTokenName) {
    throw new Error('Missing created token name')
  }

  await expect(page.getByText(/^New token created$/i)).toBeVisible({ timeout: 30000 })
  await expect(page.getByText(/engram_mcp_/i)).toBeVisible({ timeout: 30000 })

  const tokenRow = page.locator('tbody tr').filter({ hasText: createdTokenName }).first()
  await expect(tokenRow).toBeVisible({ timeout: 30000 })
})

Then('I should not see the MCP Tokens admin action', async ({ page }) => {
  await page.goto(adminTokensRoute(), { waitUntil: 'domcontentloaded' })
  await expect(page.getByTestId('admin-mcp-token-page')).toHaveCount(0)
  await expect(page).toHaveURL(/\/app\/workspace(?:\?|$)/)
})

When('I create a viewer user for admin token UI checks', async ({ page }) => {
  createdViewerUsername = `viewer_acceptance_${Date.now()}`

  const response = await page.request.post('/api/v1/users', {
    data: {
      username: createdViewerUsername,
      password: createdViewerPassword,
      role: 'viewer',
      is_active: true,
    },
  })
  expect(response.ok()).toBeTruthy()
})

When('I sign out and sign in as the created viewer', async ({ page }) => {
  if (!createdViewerUsername) {
    throw new Error('Missing created viewer username')
  }

  const logoutButton = page.getByRole('button', { name: /^Logout$/i })
  if (await logoutButton.isVisible().catch(() => false)) {
    await logoutButton.click()
  }
  await page.context().clearCookies()
  await signInWithCredentials(page, createdViewerUsername, createdViewerPassword)
  await waitForAppShell(page)
})
