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
  for (let attempt = 0; attempt < 6; attempt += 1) {
    await openChatApplication(page)
    await expect(page.getByLabel('Username')).toBeVisible()
    await page.getByLabel('Username').fill('admin')
    await page.getByLabel('Password').fill('invalid-password')
    await page.getByRole('button', { name: /^Sign In$/i }).click()
  }
})

Then('I should see a login rate limit error', async ({ page }) => {
  await expect(page.locator('body')).toContainText(/Too many login attempts/i)
})
