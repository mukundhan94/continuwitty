import { After, Before, Status, setDefaultTimeout } from '@cucumber/cucumber'
import { mkdir } from 'node:fs/promises'

import { acceptanceEnv } from './env'
import { AcceptanceWorld } from './world'

setDefaultTimeout(acceptanceEnv.timeoutMs)

Before(async function (this: AcceptanceWorld) {
  await this.initBrowser()
})

After(async function (this: AcceptanceWorld, scenario) {
  try {
    if (scenario.result?.status === Status.FAILED && this.page) {
      await mkdir('artifacts', { recursive: true })
      const sanitized = scenario.pickle.name.toLowerCase().replace(/[^a-z0-9]+/g, '-')
      const screenshotPath = `artifacts/${Date.now()}-${sanitized}.png`
      await this.page.screenshot({ path: screenshotPath, fullPage: true })
      this.attach(`Saved screenshot: ${screenshotPath}`)
    }
  } finally {
    await this.closeBrowser()
  }
})
