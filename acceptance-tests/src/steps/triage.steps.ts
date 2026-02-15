import { When, Then, expect } from '../support/fixtures'
import { waitForAssistantResponseText } from '../support/chat'

const TRIAGE_SEED_PROMPT = [
  'You are the incident commander for the fictional "NebulaPay orbital payments" outage.',
  'Generate a concise but operational triage memo in markdown.',
  'Include sections: Severity, Blast Radius, Working Hypotheses, Mitigations in progress, and Next 30 Minutes.',
  'Keep it practical and action-oriented for a war-room handoff.',
].join(' ')

const TRIAGE_HANDOFF_PROMPT = [
  'Use pinned engram memory from this session and draft a commander handoff brief.',
  'Format with markdown headings and bullets for:',
  'Situation now, What changed, Immediate asks by role (SRE/AppSec/DBA), and Risks if no action in 30 minutes.',
  'Do not ask for more data. Provide a decisive operational brief.',
].join(' ')

function countKeywordHits(text: string, keywords: string[]): number {
  const normalized = text.toLowerCase()
  return keywords.filter((keyword) => normalized.includes(keyword.toLowerCase())).length
}

When('I send a live triage seed prompt', async ({ page, scenarioState }) => {
  const composer = page.getByTestId('chat-panel').locator('textarea')
  await composer.fill(TRIAGE_SEED_PROMPT)
  await composer.press('Enter')

  const assistantText = await waitForAssistantResponseText(page, 160, 120000)
  scenarioState.latestAssistantText = assistantText
})

Then('I should receive a triage-oriented assistant response', async ({ scenarioState }) => {
  const assistantText = scenarioState.latestAssistantText || ''
  expect(assistantText.length).toBeGreaterThanOrEqual(160)

  const incidentSignals = [
    'severity',
    'blast radius',
    'hypothesis',
    'mitigation',
    'next 30',
    'rollback',
    'incident',
  ]
  expect(countKeywordHits(assistantText, incidentSignals)).toBeGreaterThanOrEqual(2)
})

When('I save the current triage response as a project engram', async ({ page, scenarioState }) => {
  const title = `Nebula P1 Triage Snapshot ${Date.now()}`
  scenarioState.latestEngramTitle = title

  await page.getByTestId('chat-panel').getByRole('button', { name: /^Save as Engram$/i }).click()
  await expect(page.getByRole('dialog')).toBeVisible()
  await page.locator('#save-title').fill(title)
  await page.locator('#save-abstract').fill(
    'Live triage snapshot for commander handoff validation with continuity memory.',
  )
  await page.locator('#save-visibility').selectOption('project')
  await page.locator('#save-tags').fill('incident,triage,nebula,p1')
  await page.locator('#save-keywords').fill('harness,continuity,handoff,rollback')
  await page.getByRole('button', { name: /^Save Engram$/i }).click()

  const notice = page.getByText(/^Saved session as engram /i)
  await expect(notice).toBeVisible({ timeout: 30000 })
  const noticeText = (await notice.innerText()).trim()
  const match = noticeText.match(/[0-9a-f]{8}-[0-9a-f]{4}-[1-5][0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}/i)
  if (!match) {
    throw new Error(`Unable to parse engram id from notice: ${noticeText}`)
  }
  scenarioState.latestEngramId = match[0]
})

When('I pin the saved triage engram to the active session', async ({ page, scenarioState }) => {
  if (!scenarioState.latestEngramId) {
    throw new Error('No saved engram id available to pin')
  }
  if (!scenarioState.latestEngramTitle) {
    throw new Error('No saved engram title available to pin')
  }

  const panel = page.getByTestId('pinned-engrams-panel')
  await panel.getByPlaceholder('Search title, abstract, or engram id').fill(scenarioState.latestEngramId)

  const pinButton = panel.getByRole('button', { name: /^Pin to Session$/i }).first()
  await expect(pinButton).toBeVisible({ timeout: 30000 })
  await pinButton.click()

  const sessionPins = panel.getByRole('heading', { name: /^Session Pins$/i }).locator('..')
  await expect(sessionPins.getByText(scenarioState.latestEngramTitle)).toBeVisible({ timeout: 30000 })
  await expect(sessionPins.getByRole('button', { name: /^Unpin$/i })).toBeVisible()
})

When('I request a commander handoff brief from pinned triage memory', async ({ page, scenarioState }) => {
  const composer = page.getByTestId('chat-panel').locator('textarea')
  await composer.fill(TRIAGE_HANDOFF_PROMPT)
  await composer.press('Enter')

  const assistantText = await waitForAssistantResponseText(page, 180, 120000)
  scenarioState.latestAssistantText = assistantText
})

Then('I should receive a continuity-aware triage handoff response', async ({ scenarioState }) => {
  const assistantText = scenarioState.latestAssistantText || ''
  expect(assistantText.length).toBeGreaterThanOrEqual(180)

  const handoffSignals = [
    'situation',
    'what changed',
    'immediate asks',
    'risk',
    'next 30',
    'handoff',
    'incident',
  ]
  expect(countKeywordHits(assistantText, handoffSignals)).toBeGreaterThanOrEqual(3)
})
