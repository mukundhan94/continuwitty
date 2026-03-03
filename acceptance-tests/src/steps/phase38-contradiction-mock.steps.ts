import { randomUUID } from 'node:crypto'

import { type APIRequestContext, type APIResponse } from '@playwright/test'

import { Then, When, expect } from '../support/fixtures'

type ContradictionAlert = {
  alert_id: string
  source_engram_id: string
  target_engram_id: string
  status: string
  resolved_at?: string | null
  resolved_by?: string | null
}

type ContradictionRefreshResponse = {
  updated_count: number
}

type ContradictionBenchmark = {
  precision: number
  recall: number
  truePositiveCount: number
  expectedCount: number
  predictedCount: number
}

const contradictionPairs: Array<[string, string]> = [
  ['Deploy exclusively with blue/green strategy.', 'Never use blue/green deployments in production.'],
  ['Store PII with one-year retention.', 'Purge all PII immediately after processing.'],
]

let seededProjectID: string | null = null
let expectedPairKeys = new Set<string>()
let listedAlerts: ContradictionAlert[] = []
let selectedAlertID: string | null = null
let contradictionBenchmark: ContradictionBenchmark | null = null

type PairKeyInput = {
  leftID: string
  rightID: string
}

type ExpectJSONInput = {
  response: APIResponse
  context: string
}

type CreateProjectInput = {
  projectID: string
  request: APIRequestContext
}

type CreateEngramInput = {
  request: APIRequestContext
  projectID: string
  title: string
}

type CreateLinkInput = {
  request: APIRequestContext
  sourceEngramID: string
  targetEngramID: string
  relationType: 'contradicts' | 'related_to'
}

function pairKey(input: PairKeyInput): string {
  return [input.leftID, input.rightID].sort().join('|')
}

async function expectJSON<T>(input: ExpectJSONInput): Promise<T> {
  const payload = await input.response.text()
  if (!input.response.ok()) {
    throw new Error(`${input.context} failed (${input.response.status()}): ${payload}`)
  }
  return JSON.parse(payload) as T
}

async function createPhase38Project(input: CreateProjectInput): Promise<void> {
  const response = await input.request.post('/api/v1/projects', {
    data: {
      project_id: input.projectID,
      name: input.projectID,
      description: 'Phase38 contradiction quality acceptance project',
    },
  })
  await expectJSON<Record<string, unknown>>({ response, context: 'phase38 project create' })
}

async function createEngram(input: CreateEngramInput): Promise<string> {
  const response = await input.request.post('/api/v1/engrams', {
    data: {
      project_id: input.projectID,
      thread_id: `phase38-thread-${randomUUID()}`,
      title: input.title,
      abstract: `Phase38 contradiction check for ${input.title}`,
      detailed_summary_markdown: `Phase38 deterministic entry for ${input.title}`,
      tags: ['phase38', 'contradiction'],
      keywords: ['phase38', 'contradiction'],
      visibility_scope: 'project',
    },
  })
  const payload = await expectJSON<{ engram_id: string }>({
    response,
    context: 'phase38 engram create',
  })
  return payload.engram_id
}

async function createLink(input: CreateLinkInput): Promise<void> {
  const response = await input.request.post(`/api/v1/engrams/${input.sourceEngramID}/links`, {
    data: {
      target_engram_id: input.targetEngramID,
      relation_type: input.relationType,
      weight: 0.9,
      temporal_weight: 0.8,
      confidence: 0.95,
    },
  })
  await expectJSON<Record<string, unknown>>({ response, context: 'phase38 link create' })
}

function calculateContradictionBenchmark(
  alerts: ContradictionAlert[],
  expectedKeys: Set<string>,
): ContradictionBenchmark {
  const predictedKeys = new Set<string>()
  for (const alert of alerts) {
    predictedKeys.add(pairKey({ leftID: alert.source_engram_id, rightID: alert.target_engram_id }))
  }
  let truePositiveCount = 0
  for (const expectedKey of expectedKeys) {
    if (predictedKeys.has(expectedKey)) {
      truePositiveCount++
    }
  }
  const predictedCount = predictedKeys.size
  const expectedCount = expectedKeys.size
  return {
    precision: predictedCount === 0 ? 0 : truePositiveCount / predictedCount,
    recall: expectedCount === 0 ? 0 : truePositiveCount / expectedCount,
    truePositiveCount,
    expectedCount,
    predictedCount,
  }
}

When('I seed deterministic contradiction links for warning quality checks', async ({ page }) => {
  seededProjectID = `phase38-contradiction-${Date.now()}`
  expectedPairKeys = new Set<string>()
  listedAlerts = []
  selectedAlertID = null
  contradictionBenchmark = null

  await createPhase38Project({ projectID: seededProjectID, request: page.request })

  for (const [leftText, rightText] of contradictionPairs) {
    const leftID = await createEngram({ request: page.request, projectID: seededProjectID, title: leftText })
    const rightID = await createEngram({
      request: page.request,
      projectID: seededProjectID,
      title: rightText,
    })
    expectedPairKeys.add(pairKey({ leftID, rightID }))
    await createLink({
      request: page.request,
      sourceEngramID: leftID,
      targetEngramID: rightID,
      relationType: 'contradicts',
    })
  }

  const unrelatedLeft = await createEngram({
    request: page.request,
    projectID: seededProjectID,
    title: 'Roll out with staged canary.',
  })
  const unrelatedRight = await createEngram({
    request: page.request,
    projectID: seededProjectID,
    title: 'Collect rollout metrics in Grafana.',
  })
  await createLink({
    request: page.request,
    sourceEngramID: unrelatedLeft,
    targetEngramID: unrelatedRight,
    relationType: 'related_to',
  })
})

When('I refresh contradiction alerts for the seeded project', async ({ page }) => {
  if (!seededProjectID) {
    throw new Error('Missing seeded project id')
  }
  const refresh = await page.request.post('/api/v1/admin/memory/engrams/contradictions/refresh', {
    data: { project_id: seededProjectID },
  })
  const refreshPayload = await expectJSON<ContradictionRefreshResponse>({
    response: refresh,
    context: 'phase38 contradiction refresh',
  })
  expect(refreshPayload.updated_count).toBeGreaterThanOrEqual(expectedPairKeys.size)

  const list = await page.request.get(
    `/api/v1/admin/memory/engrams/contradictions/alerts?project_id=${seededProjectID}&status=open&limit=25&offset=0`,
  )
  listedAlerts = await expectJSON<ContradictionAlert[]>({
    response: list,
    context: 'phase38 contradiction list',
  })
  expect(listedAlerts.length).toBeGreaterThanOrEqual(expectedPairKeys.size)
})

Then('contradiction alert precision and recall should meet threshold', async () => {
  contradictionBenchmark = calculateContradictionBenchmark(listedAlerts, expectedPairKeys)
  expect(contradictionBenchmark.expectedCount).toBeGreaterThanOrEqual(2)
  expect(contradictionBenchmark.truePositiveCount).toBe(contradictionBenchmark.expectedCount)
  expect(contradictionBenchmark.precision).toBeGreaterThanOrEqual(0.95)
  expect(contradictionBenchmark.recall).toBeGreaterThanOrEqual(0.95)

  const selected = listedAlerts.find((alert) =>
    expectedPairKeys.has(pairKey({ leftID: alert.source_engram_id, rightID: alert.target_engram_id })),
  )
  selectedAlertID = selected?.alert_id ?? null
  expect(selectedAlertID).toBeTruthy()
})

When('I resolve one contradiction alert for the seeded project', async ({ page }) => {
  if (!seededProjectID || !selectedAlertID) {
    throw new Error('Missing seeded project id or selected alert id')
  }
  const response = await page.request.post(
    `/api/v1/admin/memory/engrams/contradictions/alerts/${selectedAlertID}/resolve`,
    {
      data: {
        project_id: seededProjectID,
        status: 'resolved',
      },
    },
  )
  const resolved = await expectJSON<ContradictionAlert>({
    response,
    context: 'phase38 contradiction resolve',
  })
  expect(resolved.alert_id).toBe(selectedAlertID)
  expect(resolved.status).toBe('resolved')
})

Then('resolved contradiction alerts should include the actioned record', async ({ page }) => {
  if (!seededProjectID || !selectedAlertID) {
    throw new Error('Missing seeded project id or selected alert id')
  }
  const response = await page.request.get(
    `/api/v1/admin/memory/engrams/contradictions/alerts?project_id=${seededProjectID}&status=resolved&limit=25&offset=0`,
  )
  const resolved = await expectJSON<ContradictionAlert[]>({
    response,
    context: 'phase38 resolved contradiction list',
  })
  const actioned = resolved.find((entry) => entry.alert_id === selectedAlertID)
  expect(actioned).toBeTruthy()
  expect(actioned?.status).toBe('resolved')
  expect(actioned?.resolved_at).toBeTruthy()
  expect(actioned?.resolved_by).toBeTruthy()
})
