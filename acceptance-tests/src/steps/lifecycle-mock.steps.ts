import { randomUUID } from 'node:crypto'

import type { Route } from '@playwright/test'

import { Given, Then, When, expect } from '../support/fixtures'
import { uniqueTitle, waitForAssistantResponseText } from '../support/chat'

type MockLifecycleSession = {
  session: Record<string, unknown>
  autosaveEnabled: boolean
  autosaveStrategy: 'off' | 'interval' | 'message_count'
  autosaveMinMessages: number
  autosaveIntervalMinutes: number
  assistantMessageCount: number
  messages: Array<Record<string, unknown>>
  timeline: Array<Record<string, unknown>>
}

const mockSessions = new Map<string, MockLifecycleSession>()
let latestCreatePayload: Record<string, unknown> | null = null

function asJson(route: Route, status: number, payload: unknown): Promise<void> {
  return route.fulfill({
    status,
    contentType: 'application/json',
    body: JSON.stringify(payload),
  })
}

function newTimelineEvent(title: string, abstract: string): Record<string, unknown> {
  return {
    event_id: randomUUID(),
    event_type: 'autosave_snapshot',
    engram_id: randomUUID(),
    title,
    abstract,
    created_at: new Date().toISOString(),
  }
}

function buildSseFrame(eventName: string, data: unknown): string {
  return `event: ${eventName}\ndata: ${JSON.stringify(data)}\n\n`
}

Given('mocked lifecycle backend is enabled', async ({ page }) => {
  mockSessions.clear()
  latestCreatePayload = null

  await page.route('**/api/v1/engrams?**', async (route) => {
    if (route.request().method() === 'GET') {
      await asJson(route, 200, [])
      return
    }
    await route.fallback()
  })

  await page.route('**/api/v1/chat/**', async (route) => {
    const request = route.request()
    const method = request.method()
    const url = new URL(request.url())
    const path = url.pathname

    if (method === 'GET' && path === '/api/v1/chat/sessions') {
      const sessions = [...mockSessions.values()].map((entry) => entry.session)
      await asJson(route, 200, sessions)
      return
    }

    if (method === 'POST' && path === '/api/v1/chat/sessions') {
      const payload = request.postDataJSON() as Record<string, unknown>
      latestCreatePayload = payload

      const sessionId = randomUUID()
      const nowIso = new Date().toISOString()
      const createdSession = {
        session_id: sessionId,
        owner_user_id: '4e23dee1-4ae7-4d2a-a022-c0c071cd817a',
        project_id: payload.project_id,
        title: payload.title,
        provider: payload.provider,
        model_id: payload.model_id,
        system_prompt: payload.system_prompt || '',
        visibility_scope: payload.visibility_scope || 'private',
        autosave_enabled: Boolean(payload.autosave_enabled),
        autosave_strategy: (payload.autosave_strategy as string) || 'off',
        autosave_interval_minutes: Number(payload.autosave_interval_minutes || 30),
        autosave_min_messages: Number(payload.autosave_min_messages || 6),
        retention_days: Number(payload.retention_days || 30),
        retention_max_snapshots: Number(payload.retention_max_snapshots || 60),
        created_at: nowIso,
        updated_at: nowIso,
      }

      mockSessions.set(sessionId, {
        session: createdSession,
        autosaveEnabled: Boolean(createdSession.autosave_enabled),
        autosaveStrategy: createdSession.autosave_strategy as 'off' | 'interval' | 'message_count',
        autosaveMinMessages: Number(createdSession.autosave_min_messages),
        autosaveIntervalMinutes: Number(createdSession.autosave_interval_minutes),
        assistantMessageCount: 0,
        messages: [],
        timeline: [],
      })

      await asJson(route, 201, createdSession)
      return
    }

    const sessionMatch = path.match(/^\/api\/v1\/chat\/sessions\/([^/]+)$/)
    if (method === 'GET' && sessionMatch) {
      const session = mockSessions.get(sessionMatch[1])
      await asJson(route, session ? 200 : 404, session ? session.session : { detail: 'Not found' })
      return
    }

    const messagesMatch = path.match(/^\/api\/v1\/chat\/sessions\/([^/]+)\/messages$/)
    if (method === 'GET' && messagesMatch) {
      const session = mockSessions.get(messagesMatch[1])
      await asJson(route, 200, session?.messages ?? [])
      return
    }

    const timelineMatch = path.match(/^\/api\/v1\/chat\/sessions\/([^/]+)\/timeline$/)
    if (method === 'GET' && timelineMatch) {
      const session = mockSessions.get(timelineMatch[1])
      await asJson(route, 200, session?.timeline ?? [])
      return
    }

    const pinnedEngramsMatch = path.match(/^\/api\/v1\/chat\/sessions\/([^/]+)\/engrams$/)
    if (method === 'GET' && pinnedEngramsMatch) {
      await asJson(route, 200, [])
      return
    }

    const pinnedDocumentsMatch = path.match(/^\/api\/v1\/chat\/sessions\/([^/]+)\/documents$/)
    if (method === 'GET' && pinnedDocumentsMatch) {
      await asJson(route, 200, [])
      return
    }

    const streamMatch = path.match(/^\/api\/v1\/chat\/sessions\/([^/]+)\/messages\/stream$/)
    if (method === 'POST' && streamMatch) {
      const sessionId = streamMatch[1]
      const session = mockSessions.get(sessionId)
      if (!session) {
        await asJson(route, 404, { detail: 'Session not found' })
        return
      }

      const payload = request.postDataJSON() as { content_text?: string }
      const userText = (payload.content_text || '').trim()
      const assistantText = `mocked assistant response for: ${userText}`

      const userMessage = {
        message_id: randomUUID(),
        session_id: sessionId,
        role: 'user',
        content_text: userText,
        provider: null,
        model_id: null,
        token_usage_json: {},
        used_engram_ids: [],
        created_at: new Date().toISOString(),
      }
      const assistantMessage = {
        message_id: randomUUID(),
        session_id: sessionId,
        role: 'assistant',
        content_text: assistantText,
        provider: session.session.provider,
        model_id: session.session.model_id,
        token_usage_json: { input_tokens: 10, output_tokens: 12, total_tokens: 22 },
        used_engram_ids: [],
        created_at: new Date().toISOString(),
      }

      session.messages.push(userMessage, assistantMessage)
      session.assistantMessageCount += 1

      if (session.autosaveEnabled) {
        if (
          session.autosaveStrategy === 'message_count' &&
          session.assistantMessageCount % Math.max(session.autosaveMinMessages, 1) === 0
        ) {
          session.timeline.unshift(
            newTimelineEvent(
              `Autosave Snapshot #${Math.floor(session.assistantMessageCount / Math.max(session.autosaveMinMessages, 1))}`,
              'Mock autosave snapshot generated at message-count threshold.',
            ),
          )
        }
        if (session.autosaveStrategy === 'interval' && session.timeline.length === 0) {
          session.timeline.unshift(
            newTimelineEvent('Autosave Snapshot #1', 'Mock interval autosave snapshot.'),
          )
        }
      }

      const metaPayload = {
        session_id: sessionId,
        message_id: userMessage.message_id,
        used_engram_ids: [],
        used_document_chunk_ids: [],
        source_references: [],
      }
      const donePayload = {
        session_id: sessionId,
        message_id: userMessage.message_id,
        reply_message_id: assistantMessage.message_id,
        assistant_text: assistantText,
        used_engram_ids: [],
        used_document_chunk_ids: [],
        source_references: [],
      }

      const sseBody = [
        buildSseFrame('meta', metaPayload),
        buildSseFrame('chunk', { text: assistantText }),
        buildSseFrame('done', donePayload),
      ].join('')

      await route.fulfill({
        status: 200,
        headers: {
          'content-type': 'text/event-stream',
          'cache-control': 'no-cache',
          connection: 'keep-alive',
        },
        body: sseBody,
      })
      return
    }

    await route.fallback()
  })
})

When('I create a message-count autosave session named {string}', async ({ page, scenarioState }, title: string) => {
  const actualTitle = uniqueTitle(title)
  scenarioState.latestSessionTitle = actualTitle

  const showCreator = page.getByRole('button', { name: /Show Creator/i })
  if (await showCreator.isVisible().catch(() => false)) {
    await showCreator.click()
  }

  await page.locator('#session-title').fill(actualTitle)
  await page.locator('#autosave-enabled').check()
  await page.locator('#autosave-strategy').selectOption('message_count')
  await page.locator('#autosave-min-messages').fill('2')
  await page.getByRole('button', { name: /Create Session/i }).click()

  await expect(page.getByRole('heading', { name: actualTitle })).toBeVisible()
})

When('I create an autosave-off session named {string}', async ({ page, scenarioState }, title: string) => {
  const actualTitle = uniqueTitle(title)
  scenarioState.latestSessionTitle = actualTitle

  const showCreator = page.getByRole('button', { name: /Show Creator/i })
  if (await showCreator.isVisible().catch(() => false)) {
    await showCreator.click()
  }

  await page.locator('#session-title').fill(actualTitle)
  await page.locator('#autosave-enabled').uncheck()
  await page.getByRole('button', { name: /Create Session/i }).click()

  await expect(page.getByRole('heading', { name: actualTitle })).toBeVisible()
})

When('I send a mocked lifecycle prompt {string}', async ({ page }, prompt: string) => {
  const composer = page.getByTestId('chat-panel').locator('textarea')
  await composer.fill(prompt)
  await composer.press('Enter')
  await waitForAssistantResponseText(page, 24, 30000)
})

Then('lifecycle timeline should show {int} autosave snapshots', async ({ page }, expectedCount: number) => {
  const timeline = page.getByTestId('chat-panel').locator('li').filter({ hasText: /Autosave Snapshot/i })
  await expect(timeline).toHaveCount(expectedCount)
})

Then(
  'the create session request should include autosave strategy {string} with minimum messages {int}',
  async ({}, expectedStrategy: string, expectedMinMessages: number) => {
    expect(latestCreatePayload).not.toBeNull()
    expect(latestCreatePayload?.autosave_enabled).toBe(true)
    expect(latestCreatePayload?.autosave_strategy).toBe(expectedStrategy)
    expect(Number(latestCreatePayload?.autosave_min_messages)).toBe(expectedMinMessages)
  },
)

Then('the create session request should disable autosave', async ({}) => {
  expect(latestCreatePayload).not.toBeNull()
  expect(latestCreatePayload?.autosave_enabled).toBe(false)
  expect(latestCreatePayload?.autosave_strategy).toBe('off')
})
