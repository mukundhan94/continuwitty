import { Given, Then, When, expect, signInIfNeeded } from '../support/fixtures'
import { uniqueTitle, waitForAssistantResponseText } from '../support/chat'
import { acceptanceEnv } from '../support/env'

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
  const assistantText = await waitForAssistantResponseText(
    page,
    acceptanceEnv.bedrockLiveMinResponseChars,
    120000,
  )
  scenarioState.latestAssistantText = assistantText
  expect(assistantText.toLowerCase()).not.toContain('bedrock credentials are not configured')
  expect(assistantText.toLowerCase()).not.toContain('provider error')
})
