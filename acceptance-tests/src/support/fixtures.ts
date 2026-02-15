import { mkdir } from 'node:fs/promises'
import path from 'node:path'

import { expect, type Page } from '@playwright/test'
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

export async function openChatApplication(page: Page): Promise<void> {
  await page.goto(acceptanceEnv.webBaseUrl, { waitUntil: 'networkidle' })
}

export async function signInIfNeeded(page: Page): Promise<void> {
  await openChatApplication(page)

  const workbenchHeading = page.getByRole('heading', { name: /Memory Continuity Workbench/i })
  if (await workbenchHeading.isVisible({ timeout: 1500 }).catch(() => false)) {
    return
  }

  await expect(page.getByLabel('Username')).toBeVisible()
  await expect(page.getByLabel('Password')).toBeVisible()
  await page.getByLabel('Username').fill(acceptanceEnv.username)
  await page.getByLabel('Password').fill(acceptanceEnv.password)
  await page.getByRole('button', { name: /^Sign In$/i }).click()
  await expect(workbenchHeading).toBeVisible()
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
