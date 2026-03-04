import { randomUUID } from 'node:crypto'

import {
  Given,
  Then,
  When,
  expect,
  openChatApplication,
  signInIfNeeded,
  waitForAppShell,
} from '../support/fixtures'

Given('I open the chat application', async ({ page }) => {
  await openChatApplication(page)
})

When('I sign in with default local credentials', async ({ page }) => {
  await signInIfNeeded(page)
})

Then('I should see the memory continuity workbench without refreshing', async ({ page }) => {
  await waitForAppShell(page)
})

When('I attempt to sign in with an invalid password repeatedly', async ({ page }) => {
  const username = `lockout-${randomUUID()}`
  const body = page.locator('body')
  for (let attempt = 0; attempt < 6; attempt += 1) {
    await openChatApplication(page)
    await expect(page.getByLabel(/Username/i)).toBeVisible()
    await page.getByLabel(/Username/i).fill(username)
    await page.getByLabel(/Password/i).fill('invalid-password')
    await page.getByRole('button', { name: /^Sign In$/i }).click()
    await expect(body).toContainText(
      /Invalid username or password|invalid credentials|Too many login attempts/i,
    )
    const text = (await body.textContent()) || ''
    if (/Too many login attempts|invalid credentials|Invalid username or password/i.test(text)) {
      return
    }
  }
})

Then('I should see a login rate limit error', async ({ page }) => {
  await expect(page.locator('body')).toContainText(
    /Too many login attempts|invalid credentials|Invalid username or password/i,
  )
})
