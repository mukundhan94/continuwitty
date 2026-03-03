import { randomUUID } from 'node:crypto'

import { type APIRequestContext, type APIResponse } from '@playwright/test'

import { Then, When, expect } from '../support/fixtures'

type ConsolidationSuggestion = {
  suggestion_id: string
  source_engram_ids: string[]
  status: string
  actioned_at?: string | null
  action_taken_by?: string | null
}

type ConsolidationRefreshResponse = {
  updated_count: number
}

type GroupingBenchmark = {
  precision: number
  recall: number
  truePositiveCount: number
  expectedCount: number
  predictedCount: number
}

const duplicateTitleGroups = [
  ['Incident Response Playbook', 'incident response playbook', ' Incident Response Playbook  '],
  ['Rollback Checklist', 'rollback checklist'],
]

const uniqueTitles = ['Paging Escalation Matrix', 'SLO Burn Alert Routing']

let seededProjectID: string | null = null
let expectedGroupKeys = new Set<string>()
let seededTitleByEngramID = new Map<string, string>()
let listedSuggestions: ConsolidationSuggestion[] = []
let selectedSuggestionID: string | null = null
let groupingBenchmark: GroupingBenchmark | null = null

function normalizeTitle(title: string): string {
  return title.trim().toLowerCase()
}

async function expectJSON<T>(response: APIResponse, context: string): Promise<T> {
  const payload = await response.text()
  if (!response.ok()) {
    throw new Error(`${context} failed (${response.status()}): ${payload}`)
  }
  return JSON.parse(payload) as T
}

async function createPhase37Project(
  projectID: string,
  pageRequest: APIRequestContext,
): Promise<void> {
  const response = await pageRequest.post('/api/v1/projects', {
    data: {
      project_id: projectID,
      name: projectID,
      description: 'Phase37 consolidation quality acceptance project',
    },
  })
  await expectJSON<Record<string, unknown>>(response, 'phase37 project create')
}

async function createEngram(pageRequest: APIRequestContext, projectID: string, title: string): Promise<string> {
  const response = await pageRequest.post('/api/v1/engrams', {
    data: {
      project_id: projectID,
      thread_id: `phase37-thread-${randomUUID()}`,
      title,
      abstract: `Phase37 duplicate check for ${title.trim()}`,
      detailed_summary_markdown: `Phase37 deterministic entry for ${title.trim()}`,
      tags: ['phase37', 'consolidation'],
      keywords: ['phase37', 'duplicate'],
      visibility_scope: 'project',
    },
  })
  const payload = await expectJSON<{ engram_id: string }>(response, 'phase37 engram create')
  return payload.engram_id
}

function suggestionGroupKey(
  suggestion: ConsolidationSuggestion,
  titleByEngramID: Map<string, string>,
): string | null {
  const titles = new Set<string>()
  for (const sourceID of suggestion.source_engram_ids) {
    const normalized = titleByEngramID.get(sourceID)
    if (normalized) {
      titles.add(normalized)
    }
  }
  if (titles.size !== 1) {
    return null
  }
  return [...titles][0]
}

function calculateGroupingBenchmark(
  suggestions: ConsolidationSuggestion[],
  expectedKeys: Set<string>,
  titleByEngramID: Map<string, string>,
): GroupingBenchmark {
  const predictedKeys = new Set<string>()
  for (const suggestion of suggestions) {
    const key = suggestionGroupKey(suggestion, titleByEngramID)
    if (key) {
      predictedKeys.add(key)
    }
  }
  let truePositiveCount = 0
  for (const key of expectedKeys) {
    if (predictedKeys.has(key)) {
      truePositiveCount++
    }
  }
  const predictedCount = predictedKeys.size
  const expectedCount = expectedKeys.size
  return {
    precision: predictedCount == 0 ? 0 : truePositiveCount / predictedCount,
    recall: expectedCount == 0 ? 0 : truePositiveCount / expectedCount,
    truePositiveCount,
    expectedCount,
    predictedCount,
  }
}

When('I seed deterministic duplicate engrams for consolidation quality checks', async ({ page }) => {
  seededProjectID = `phase37-consolidation-${Date.now()}`
  expectedGroupKeys = new Set<string>()
  seededTitleByEngramID = new Map<string, string>()
  listedSuggestions = []
  selectedSuggestionID = null
  groupingBenchmark = null

  await createPhase37Project(seededProjectID, page.request)

  for (const group of duplicateTitleGroups) {
    const expectedKey = normalizeTitle(group[0])
    expectedGroupKeys.add(expectedKey)
    for (const rawTitle of group) {
      const engramID = await createEngram(page.request, seededProjectID, rawTitle)
      seededTitleByEngramID.set(engramID, normalizeTitle(rawTitle))
    }
  }

  for (const rawTitle of uniqueTitles) {
    const engramID = await createEngram(page.request, seededProjectID, rawTitle)
    seededTitleByEngramID.set(engramID, normalizeTitle(rawTitle))
  }
})

When('I refresh consolidation suggestions for the seeded project', async ({ page }) => {
  if (!seededProjectID) {
    throw new Error('Missing seeded project id')
  }
  const refresh = await page.request.post('/api/v1/admin/memory/engrams/consolidation/refresh', {
    data: {
      project_id: seededProjectID,
      min_group_size: 2,
    },
  })
  const refreshPayload = await expectJSON<ConsolidationRefreshResponse>(
    refresh,
    'phase37 consolidation refresh',
  )
  expect(refreshPayload.updated_count).toBeGreaterThanOrEqual(expectedGroupKeys.size)

  const list = await page.request.get(
    `/api/v1/admin/memory/engrams/consolidation/suggestions?project_id=${seededProjectID}&status=suggested&limit=25&offset=0`,
  )
  listedSuggestions = await expectJSON<ConsolidationSuggestion[]>(list, 'phase37 consolidation list')
  expect(listedSuggestions.length).toBeGreaterThanOrEqual(expectedGroupKeys.size)
})

Then('consolidation grouping precision and recall should meet threshold', async () => {
  groupingBenchmark = calculateGroupingBenchmark(
    listedSuggestions,
    expectedGroupKeys,
    seededTitleByEngramID,
  )
  expect(groupingBenchmark.expectedCount).toBeGreaterThanOrEqual(2)
  expect(groupingBenchmark.truePositiveCount).toBe(groupingBenchmark.expectedCount)
  expect(groupingBenchmark.precision).toBeGreaterThanOrEqual(0.95)
  expect(groupingBenchmark.recall).toBeGreaterThanOrEqual(0.95)

  for (const suggestion of listedSuggestions) {
    const key = suggestionGroupKey(suggestion, seededTitleByEngramID)
    if (key && expectedGroupKeys.has(key)) {
      selectedSuggestionID = suggestion.suggestion_id
      break
    }
  }
  expect(selectedSuggestionID).toBeTruthy()
})

When('I merge one consolidation suggestion for the seeded project', async ({ page }) => {
  if (!seededProjectID || !selectedSuggestionID) {
    throw new Error('Missing seeded project id or selected suggestion id')
  }
  const action = await page.request.post(
    `/api/v1/admin/memory/engrams/consolidation/suggestions/${selectedSuggestionID}/action`,
    {
      data: {
        project_id: seededProjectID,
        status: 'merged',
      },
    },
  )
  const payload = await expectJSON<ConsolidationSuggestion>(action, 'phase37 consolidation action')
  expect(payload.suggestion_id).toBe(selectedSuggestionID)
  expect(payload.status).toBe('merged')
})

Then('merged consolidation suggestions should include the actioned record', async ({ page }) => {
  if (!seededProjectID || !selectedSuggestionID) {
    throw new Error('Missing seeded project id or selected suggestion id')
  }
  const response = await page.request.get(
    `/api/v1/admin/memory/engrams/consolidation/suggestions?project_id=${seededProjectID}&status=merged&limit=25&offset=0`,
  )
  const merged = await expectJSON<ConsolidationSuggestion[]>(
    response,
    'phase37 merged consolidation list',
  )
  const actioned = merged.find((entry) => entry.suggestion_id === selectedSuggestionID)
  expect(actioned).toBeTruthy()
  expect(actioned?.status).toBe('merged')
  expect(actioned?.actioned_at).toBeTruthy()
  expect(actioned?.action_taken_by).toBeTruthy()
})
