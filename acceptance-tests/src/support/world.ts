import { World, type IWorldOptions, setWorldConstructor } from '@cucumber/cucumber'
import { type Browser, type BrowserContext, type Page, chromium } from 'playwright'

import { acceptanceEnv } from './env'

export class AcceptanceWorld extends World {
  browser: Browser | null = null
  context: BrowserContext | null = null
  page: Page | null = null
  baselinePanelHeight: number | null = null
  latestPanelHeight: number | null = null
  latestSessionTitle: string | null = null

  constructor(options: IWorldOptions) {
    super(options)
  }

  async initBrowser(): Promise<void> {
    this.browser = await chromium.launch({
      headless: acceptanceEnv.headless,
    })
    this.context = await this.browser.newContext({
      viewport: { width: 1660, height: 1024 },
    })
    this.page = await this.context.newPage()
    this.page.setDefaultTimeout(acceptanceEnv.timeoutMs)
  }

  async closeBrowser(): Promise<void> {
    await this.context?.close()
    await this.browser?.close()
    this.context = null
    this.browser = null
    this.page = null
  }

  getPage(): Page {
    if (!this.page) {
      throw new Error('Playwright page is not initialized')
    }
    return this.page
  }

  async ensureLoggedIn(): Promise<void> {
    const page = this.getPage()
    await page.goto(acceptanceEnv.webBaseUrl, { waitUntil: 'networkidle' })

    const workbenchHeading = page.getByRole('heading', { name: /Memory Continuity Workbench/i })
    if (await workbenchHeading.isVisible({ timeout: 1500 }).catch(() => false)) {
      return
    }

    const loginButton = page.getByRole('button', { name: /^Sign In$/i })
    await page.getByLabel('Username').waitFor()
    await page.getByLabel('Password').waitFor()
    if (await loginButton.isVisible()) {
      await page.getByLabel('Username').fill(acceptanceEnv.username)
      await page.getByLabel('Password').fill(acceptanceEnv.password)
      await loginButton.click()
    }

    await workbenchHeading.waitFor()
  }
}

setWorldConstructor(AcceptanceWorld)
