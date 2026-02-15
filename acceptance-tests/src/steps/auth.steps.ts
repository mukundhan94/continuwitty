import { Given, Then, When } from '@cucumber/cucumber'
import { expect } from '@playwright/test'

import { acceptanceEnv } from '../support/env'
import { AcceptanceWorld } from '../support/world'

Given('I open the chat application', async function (this: AcceptanceWorld) {
  await this.getPage().goto(acceptanceEnv.webBaseUrl, { waitUntil: 'networkidle' })
})

When('I sign in with default local credentials', async function (this: AcceptanceWorld) {
  const page = this.getPage()
  const workbenchHeading = page.getByRole('heading', { name: /Memory Continuity Workbench/i })
  if (await workbenchHeading.isVisible({ timeout: 1500 }).catch(() => false)) {
    return
  }

  await expect(page.getByLabel('Username')).toBeVisible()
  await expect(page.getByLabel('Password')).toBeVisible()
  await page.getByLabel('Username').fill(acceptanceEnv.username)
  await page.getByLabel('Password').fill(acceptanceEnv.password)
  await page.getByRole('button', { name: /^Sign In$/i }).click()
})

Then('I should see the memory continuity workbench without refreshing', async function (this: AcceptanceWorld) {
  const heading = this
    .getPage()
    .getByRole('heading', { name: /Memory Continuity Workbench/i })
  await expect(heading).toBeVisible()
})
