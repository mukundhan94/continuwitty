import { Given, Then, When, expect, signInIfNeeded } from '../support/fixtures'
import { acceptanceEnv } from '../support/env'

function uniqueTitle(base: string): string {
  const suffix = `${Date.now()}-${Math.floor(Math.random() * 10000)}`
  return `${base} ${suffix}`
}

Given('I am signed in for bedrock live testing', async ({ page }) => {
  await signInIfNeeded(page)
})

When(
  'I create a bedrock session named {string} using the default bedrock model',
  async ({ page, scenarioState }, title: string) => {
    const actualTitle = uniqueTitle(title)
    scenarioState.latestSessionTitle = actualTitle

    await page.locator('#session-title').fill(actualTitle)
    await page.locator('#provider').selectOption('bedrock')

    const modelId = (await page.locator('#model-id').inputValue()).trim()
    if (!modelId) {
      throw new Error('Bedrock model input was empty after provider selection')
    }
    scenarioState.latestModelId = modelId

    await page.getByRole('button', { name: /Create Session/i }).click()
    await expect(page.getByTestId('chat-panel').getByRole('heading', { name: actualTitle })).toBeVisible()

    const metadata = page.getByTestId('chat-panel').getByText(/^bedrock\//i)
    await expect(metadata).toContainText(`bedrock/${modelId}`)
  },
)

Then('the selected bedrock model should match the configured default when provided', async ({ scenarioState }) => {
  if (!scenarioState.latestModelId) {
    throw new Error('No bedrock model captured from session creation')
  }
  if (acceptanceEnv.bedrockLiveExpectedModel) {
    expect(scenarioState.latestModelId).toBe(acceptanceEnv.bedrockLiveExpectedModel)
  }
})

When('I send a live bedrock prompt', async ({ page }) => {
  const chatPanel = page.getByTestId('chat-panel')
  const composer = chatPanel.locator('textarea')
  await composer.fill(acceptanceEnv.bedrockLivePrompt)
  await composer.press('Enter')
})

Then('I should receive a non-empty assistant response from bedrock', async ({ page, scenarioState }) => {
  const chatPanel = page.getByTestId('chat-panel')
  const assistantBubble = chatPanel.locator('article').filter({ hasText: /\bassistant\b/i }).last()
  await expect(assistantBubble).toBeVisible({ timeout: 120000 })

  const assistantTextBlock = assistantBubble.locator('p').nth(1)
  await expect
    .poll(
      async () => {
        const text = (await assistantTextBlock.innerText()).trim()
        return text.length
      },
      { timeout: 120000 },
    )
    .toBeGreaterThanOrEqual(acceptanceEnv.bedrockLiveMinResponseChars)

  await expect(chatPanel.getByRole('button', { name: /^Send$/i })).toBeVisible({ timeout: 120000 })

  const assistantText = (await assistantTextBlock.innerText()).trim()
  scenarioState.latestAssistantText = assistantText
  expect(assistantText.toLowerCase()).not.toContain('bedrock credentials are not configured')
  expect(assistantText.toLowerCase()).not.toContain('provider error')
})
