import { mkdir } from 'node:fs/promises'
import path from 'node:path'

import { expect, type Locator, type Page } from '@playwright/test'
import { createBdd, test as bddBase } from 'playwright-bdd'

import { acceptanceEnv } from './env'

type ScenarioState = {
  baselinePanelHeight: number | null
  latestPanelHeight: number | null
  latestSessionTitle: string | null
  latestSessionId: string | null
  latestModelId: string | null
  latestEngramId: string | null
  latestEngramTitle: string | null
  latestAssistantText: string | null
}

type AcceptanceFixtures = {
  scenarioState: ScenarioState
}

export const test = bddBase.extend<AcceptanceFixtures>({
  scenarioState: async ({}, use) => {
    await use({
      baselinePanelHeight: null,
      latestPanelHeight: null,
      latestSessionTitle: null,
      latestSessionId: null,
      latestModelId: null,
      latestEngramId: null,
      latestEngramTitle: null,
      latestAssistantText: null,
    })
  },
  page: async ({ page }, use) => {
    page.setDefaultTimeout(acceptanceEnv.timeoutMs)
    await use(page)
  },
})

export { expect }

function resolveWebURL(pathname: string): string {
  const base = acceptanceEnv.webBaseUrl.endsWith('/')
    ? acceptanceEnv.webBaseUrl
    : `${acceptanceEnv.webBaseUrl}/`
  return new URL(pathname, base).toString()
}

function usernameField(page: Page) {
  return page.getByLabel(/username/i)
}

function passwordField(page: Page) {
  return page.getByLabel(/password/i)
}

function signInButton(page: Page) {
  return page.getByRole('button', { name: /^Sign In$/i })
}

function logoutButton(page: Page) {
  return page.getByRole('button', { name: /^Logout$/i })
}

async function clickIfVisible(locator: Locator, timeout: number): Promise<boolean> {
  if (!(await locator.isVisible({ timeout }).catch(() => false))) {
    return false
  }
  await locator.click()
  return true
}

async function hoverIfVisible(locator: Locator, timeout: number): Promise<boolean> {
  if (!(await locator.isVisible({ timeout }).catch(() => false))) {
    return false
  }
  await locator.hover()
  return true
}

export async function waitForAppShell(page: Page): Promise<void> {
  await expect(page).toHaveURL(/\/app(\/|$)/, { timeout: 30000 })
  await expect(logoutButton(page)).toBeVisible({ timeout: 30000 })
}

export async function ensureSessionsWorkspace(page: Page): Promise<void> {
  await waitForAppShell(page)
  const isSessionsRoute = /\/app\/sessions(\/|$)/.test(page.url())
  if (!isSessionsRoute) {
    await page.goto(resolveWebURL('/app/sessions'), { waitUntil: 'domcontentloaded' })
  }
  await expect(page).toHaveURL(/\/app\/sessions(\/|$)/, { timeout: 30000 })
}

export async function ensureSessionCreatorVisible(page: Page): Promise<void> {
  await ensureSessionsWorkspace(page)
  const titleInput = page.locator('#session-title')
  if (await titleInput.isVisible({ timeout: 500 }).catch(() => false)) {
    return
  }

  await clickIfVisible(page.getByRole('button', { name: /Show Sessions Panel/i }), 1200)

  if (await titleInput.isVisible({ timeout: 800 }).catch(() => false)) {
    return
  }

  await hoverIfVisible(page.getByTestId('dock-hotzone-left'), 1200)

  if (!(await titleInput.isVisible({ timeout: 800 }).catch(() => false))) {
    await clickIfVisible(page.getByRole('button', { name: /Show Creator/i }), 5000)
  }
  await expect(titleInput).toBeVisible({ timeout: 15000 })
}

export async function openChatApplication(page: Page): Promise<void> {
  await page.goto(resolveWebURL('/login'), { waitUntil: 'domcontentloaded' })
}

async function ensureLoginForm(page: Page): Promise<void> {
  await openChatApplication(page)

  if (await logoutButton(page).isVisible({ timeout: 1500 }).catch(() => false)) {
    return
  }

  const startFlowingLink = page.getByRole('link', { name: /start flowing|login/i }).first()
  if (await startFlowingLink.isVisible({ timeout: 1200 }).catch(() => false)) {
    await startFlowingLink.click()
  }

  await expect(usernameField(page)).toBeVisible({ timeout: 15000 })
  await expect(passwordField(page)).toBeVisible({ timeout: 15000 })
  await expect(signInButton(page)).toBeVisible({ timeout: 15000 })
}

export async function signInWithCredentials(
  page: Page,
  username: string,
  password: string,
): Promise<void> {
  await ensureLoginForm(page)
  await usernameField(page).fill(username)
  await passwordField(page).fill(password)
  await signInButton(page).click()
  await waitForAppShell(page)
}

export async function signInIfNeeded(page: Page): Promise<void> {
  await openChatApplication(page)

  if (await logoutButton(page).isVisible({ timeout: 1500 }).catch(() => false)) {
    return
  }

  await signInWithCredentials(page, acceptanceEnv.username, acceptanceEnv.password)
}

export async function signOutIfNeeded(page: Page): Promise<void> {
  const logout = logoutButton(page)
  if (await logout.isVisible({ timeout: 1500 }).catch(() => false)) {
    await logout.click()
    await expect(logout).toBeHidden({ timeout: 15000 })
  }

  await page.context().clearCookies()
  await openChatApplication(page)
}

const { Given, When, Then, After } = createBdd(test)

After(async ({ page, $testInfo }) => {
  if ($testInfo.status === $testInfo.expectedStatus) {
    return
  }

  await mkdir('artifacts', { recursive: true })
  const sanitized = $testInfo.title.toLowerCase().replace(/[^a-z0-9]+/g, '-')
  const screenshotPath = path.join('artifacts', `${Date.now()}-${sanitized}.png`)
  await page.screenshot({ path: screenshotPath, fullPage: true })
  await $testInfo.attach('failure-screenshot', {
    body: Buffer.from(`Saved screenshot: ${screenshotPath}`),
    contentType: 'text/plain',
  })
})

export { Given, When, Then }
