import { Given, Then, When, expect, signInIfNeeded } from '../support/fixtures'

function uniqueTitle(base: string): string {
  const suffix = `${Date.now()}-${Math.floor(Math.random() * 10000)}`
  return `${base} ${suffix}`
}

Given('I am signed in', async ({ page }) => {
  await signInIfNeeded(page)
})

When('I create a session named {string}', async ({ page, scenarioState }, title: string) => {
  const actualTitle = uniqueTitle(title)
  scenarioState.latestSessionTitle = actualTitle

  await page.locator('#session-title').fill(actualTitle)
  await page.getByRole('button', { name: /Create Session/i }).click()

  await expect(page.getByRole('heading', { name: actualTitle })).toBeVisible()
})

When('I capture the current chat pane height as baseline', async ({ page, scenarioState }) => {
  const panel = page.getByTestId('chat-panel')
  const box = await panel.boundingBox()
  if (!box) {
    throw new Error('Unable to capture chat panel bounding box')
  }
  scenarioState.baselinePanelHeight = box.height
})

Then(
  'the chat pane height drift should be at most {int} pixels',
  async ({ page, scenarioState }, tolerance: number) => {
    if (scenarioState.baselinePanelHeight == null) {
      throw new Error('Baseline chat pane height not captured')
    }

    const panel = page.getByTestId('chat-panel')
    const box = await panel.boundingBox()
    if (!box) {
      throw new Error('Unable to capture chat panel bounding box')
    }

    scenarioState.latestPanelHeight = box.height
    const drift = Math.abs(scenarioState.latestPanelHeight - scenarioState.baselinePanelHeight)
    expect(
      drift,
      `expected chat pane drift <= ${tolerance}px, got ${drift}px (baseline=${scenarioState.baselinePanelHeight}, current=${scenarioState.latestPanelHeight})`,
    ).toBeLessThanOrEqual(tolerance)
  },
)

When('I click continue in new chat', async ({ page }) => {
  await page.getByRole('button', { name: /Continue in New Chat/i }).click()
})

Then('a continued session should become active', async ({ page }) => {
  const heading = page.getByTestId('chat-panel').getByRole('heading', { level: 2 })
  await expect(heading).toContainText('(continued)')
})
