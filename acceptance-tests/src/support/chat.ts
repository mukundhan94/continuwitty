import type { Page } from '@playwright/test'

import { expect } from './fixtures'

export function uniqueTitle(base: string): string {
  const suffix = `${Date.now()}-${Math.floor(Math.random() * 10000)}`
  return `${base} ${suffix}`
}

function stripAssistantRolePrefix(value: string): string {
  return value.replace(/^assistant\s*/i, '').trim()
}

export async function waitForAssistantResponseText(
  page: Page,
  minChars: number,
  timeoutMs = 120000,
): Promise<string> {
  const chatPanel = page.getByTestId('chat-panel')
  const assistantBubble = chatPanel.locator('article').filter({ hasText: /\bassistant\b/i }).last()
  await expect(assistantBubble).toBeVisible({ timeout: timeoutMs })

  await expect
    .poll(
      async () => {
        const raw = (await assistantBubble.innerText()).trim()
        const text = stripAssistantRolePrefix(raw)
        return text.length
      },
      { timeout: timeoutMs },
    )
    .toBeGreaterThanOrEqual(minChars)

  await expect(chatPanel.getByRole('button', { name: /^Send$/i })).toBeVisible({ timeout: timeoutMs })

  const raw = (await assistantBubble.innerText()).trim()
  return stripAssistantRolePrefix(raw)
}
