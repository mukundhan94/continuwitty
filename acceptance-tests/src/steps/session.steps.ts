import { Given, Then, When } from '@cucumber/cucumber'
import { expect } from '@playwright/test'

import { AcceptanceWorld } from '../support/world'

function uniqueTitle(base: string): string {
  const suffix = `${Date.now()}-${Math.floor(Math.random() * 10000)}`
  return `${base} ${suffix}`
}

Given('I am signed in', async function (this: AcceptanceWorld) {
  await this.ensureLoggedIn()
})

When('I create a session named {string}', async function (this: AcceptanceWorld, title: string) {
  const page = this.getPage()
  const actualTitle = uniqueTitle(title)
  this.latestSessionTitle = actualTitle

  await page.locator('#session-title').fill(actualTitle)
  await page.getByRole('button', { name: /Create Session/i }).click()

  await expect(page.getByRole('heading', { name: actualTitle })).toBeVisible()
})

When('I capture the current chat pane height as baseline', async function (this: AcceptanceWorld) {
  const panel = this.getPage().getByTestId('chat-panel')
  const box = await panel.boundingBox()
  if (!box) {
    throw new Error('Unable to capture chat panel bounding box')
  }
  this.baselinePanelHeight = box.height
})

Then(
  'the chat pane height drift should be at most {int} pixels',
  async function (this: AcceptanceWorld, tolerance: number) {
    if (this.baselinePanelHeight == null) {
      throw new Error('Baseline chat pane height not captured')
    }

    const panel = this.getPage().getByTestId('chat-panel')
    const box = await panel.boundingBox()
    if (!box) {
      throw new Error('Unable to capture chat panel bounding box')
    }

    this.latestPanelHeight = box.height
    const drift = Math.abs(this.latestPanelHeight - this.baselinePanelHeight)
    expect(
      drift,
      `expected chat pane drift <= ${tolerance}px, got ${drift}px (baseline=${this.baselinePanelHeight}, current=${this.latestPanelHeight})`,
    ).toBeLessThanOrEqual(tolerance)
  },
)

When('I click continue in new chat', async function (this: AcceptanceWorld) {
  const page = this.getPage()
  await page.getByRole('button', { name: /Continue in New Chat/i }).click()
})

Then('a continued session should become active', async function (this: AcceptanceWorld) {
  const heading = this.getPage().getByTestId('chat-panel').getByRole('heading', { level: 2 })
  await expect(heading).toContainText('(continued)')
})
