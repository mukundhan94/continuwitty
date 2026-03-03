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

function pairKey(leftID: string, rightID: string): string {
  return [leftID, rightID].sort().join('|')
}

async function expectJSON<T>(response: APIResponse, context: string): Promise<T> {
  const payload = await response.text()
  if (!response.ok()) {
    throw new Error(`${context} failed (${response.status()}): ${payload}`)
  }
  return JSON.parse(payload) as T
}

async function createPhase38Project(
  projectID: string,
  pageRequest: APIRequestContext,
): Promise<void> {
  const response = await pageRequest.post('/api/v1/projects', {
    data: {
      project_id: projectID,
      name: projectID,
      description: 'Phase38 contradiction quality acceptance project',
    },
  })
  await expectJSON<Record<string, unknown>>(response, 'phase38 project create')
}

async function createEngram(
  pageRequest: APIRequestContext,
  projectID: string,
  title: string,
): Promise<string> {
  const response = await pageRequest.post('/api/v1/engrams', {
    data: {
      project_id: projectID,
      thread_id: `phase38-thread-${randomUUID()}`,
      title,
      abstract: `Phase38 contradiction check for ${title}`,
      detailed_summary_markdown: `Phase38 deterministic entry for ${title}`,
      tags: ['phase38', 'contradiction'],
      keywords: ['phase38', 'contradiction'],
      visibility_scope: 'project',
    },
  })
  const payload = await expectJSON<{ engram_id: string }>(response, 'phase38 engram create')
  return payload.engram_id
}

async function createLink(
  pageRequest: APIRequestContext,
  sourceEngramID: string,
  targetEngramID: string,
  relationType: 'contradicts' | 'related_to',
): Promise<void> {
  const response = await pageRequest.post(`/api/v1/engrams/${sourceEngramID}/links`, {
    data: {
      target_engram_id: targetEngramID,
      relation_type: relationType,
      weight: 0.9,
      temporal_weight: 0.8,
      confidence: 0.95,
    },
  })
  await expectJSON<Record<string, unknown>>(response, 'phase38 link create')
}

function calculateContradictionBenchmark(
  alerts: ContradictionAlert[],
  expectedKeys: Set<string>,
): ContradictionBenchmark {
  const predictedKeys = new Set<string>()
  for (const alert of alerts) {
    predictedKeys.add(pairKey(alert.source_engram_id, alert.target_engram_id))
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

  await createPhase38Project(seededProjectID, page.request)

  for (const [leftText, rightText] of contradictionPairs) {
    const leftID = await createEngram(page.request, seededProjectID, leftText)
    const rightID = await createEngram(page.request, seededProjectID, rightText)
    expectedPairKeys.add(pairKey(leftID, rightID))
    await createLink(page.request, leftID, rightID, 'contradicts')
  }

  const unrelatedLeft = await createEngram(page.request, seededProjectID, 'Roll out with staged canary.')
  const unrelatedRight = await createEngram(page.request, seededProjectID, 'Collect rollout metrics in Grafana.')
  await createLink(page.request, unrelatedLeft, unrelatedRight, 'related_to')
})

When('I refresh contradiction alerts for the seeded project', async ({ page }) => {
  if (!seededProjectID) {
    throw new Error('Missing seeded project id')
  }
  const refresh = await page.request.post('/api/v1/admin/memory/engrams/contradictions/refresh', {
    data: { project_id: seededProjectID },
  })
  const refreshPayload = await expectJSON<ContradictionRefreshResponse>(
    refresh,
    'phase38 contradiction refresh',
  )
  expect(refreshPayload.updated_count).toBeGreaterThanOrEqual(expectedPairKeys.size)

  const list = await page.request.get(
    `/api/v1/admin/memory/engrams/contradictions/alerts?project_id=${seededProjectID}&status=open&limit=25&offset=0`,
  )
  listedAlerts = await expectJSON<ContradictionAlert[]>(list, 'phase38 contradiction list')
  expect(listedAlerts.length).toBeGreaterThanOrEqual(expectedPairKeys.size)
})

Then('contradiction alert precision and recall should meet threshold', async () => {
  contradictionBenchmark = calculateContradictionBenchmark(listedAlerts, expectedPairKeys)
  expect(contradictionBenchmark.expectedCount).toBeGreaterThanOrEqual(2)
  expect(contradictionBenchmark.truePositiveCount).toBe(contradictionBenchmark.expectedCount)
  expect(contradictionBenchmark.precision).toBeGreaterThanOrEqual(0.95)
  expect(contradictionBenchmark.recall).toBeGreaterThanOrEqual(0.95)

  const selected = listedAlerts.find((alert) =>
    expectedPairKeys.has(pairKey(alert.source_engram_id, alert.target_engram_id)),
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
  const resolved = await expectJSON<ContradictionAlert>(response, 'phase38 contradiction resolve')
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
  const resolved = await expectJSON<ContradictionAlert[]>(response, 'phase38 resolved contradiction list')
  const actioned = resolved.find((entry) => entry.alert_id === selectedAlertID)
  expect(actioned).toBeTruthy()
  expect(actioned?.status).toBe('resolved')
  expect(actioned?.resolved_at).toBeTruthy()
  expect(actioned?.resolved_by).toBeTruthy()
})
