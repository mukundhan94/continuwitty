import { randomUUID } from 'node:crypto'

import { Given, Then, When, expect, openChatApplication, signInIfNeeded } from '../support/fixtures'

Given('I open the chat application', async ({ page }) => {
  await openChatApplication(page)
})

When('I sign in with default local credentials', async ({ page }) => {
  await signInIfNeeded(page)
})

Then('I should see the memory continuity workbench without refreshing', async ({ page }) => {
  const heading = page.getByRole('heading', { name: /Memory Continuity Workbench/i })
  await expect(heading).toBeVisible()
})

When('I attempt to sign in with an invalid password repeatedly', async ({ page }) => {
  const username = `lockout-${randomUUID()}`
  const body = page.locator('body')
  for (let attempt = 0; attempt < 6; attempt += 1) {
    await openChatApplication(page)
    await expect(page.getByLabel('Username')).toBeVisible()
    await page.getByLabel('Username').fill(username)
    await page.getByLabel('Password').fill('invalid-password')
    await page.getByRole('button', { name: /^Sign In$/i }).click()
    await expect(body).toContainText(/Invalid username or password|Too many login attempts/i)
    const text = (await body.textContent()) || ''
    if (/Too many login attempts/i.test(text)) {
      return
    }
  }
})

Then('I should see a login rate limit error', async ({ page }) => {
  await expect(page.locator('body')).toContainText(/Too many login attempts/i)
})
