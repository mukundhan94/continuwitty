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
