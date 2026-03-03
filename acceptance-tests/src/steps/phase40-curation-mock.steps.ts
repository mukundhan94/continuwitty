import { randomUUID } from 'node:crypto'

import { type APIRequestContext, type APIResponse } from '@playwright/test'

import { Then, When, expect } from '../support/fixtures'

type CurationSuggestion = {
  suggestion_id: string
  project_id: string
  suggestion_type: string
  status: string
  actioned_at?: string | null
  action_taken_by?: string | null
}

type RefreshResponse = {
  updated_count: number
}

let seededProjectID: string | null = null
let listedSuggestions: CurationSuggestion[] = []
let selectedSuggestionID: string | null = null

async function expectJSON<T>(response: APIResponse, context: string): Promise<T> {
  const payload = await response.text()
  if (!response.ok()) {
    throw new Error(`${context} failed (${response.status()}): ${payload}`)
  }
  return JSON.parse(payload) as T
}

async function createProject(projectID: string, request: APIRequestContext): Promise<void> {
  const response = await request.post('/api/v1/projects', {
    data: {
      project_id: projectID,
      name: projectID,
      description: 'Phase40 curation acceptance project',
    },
  })
  await expectJSON<Record<string, unknown>>(response, 'phase40 project create')
}

async function createEngram(
  request: APIRequestContext,
  projectID: string,
  title: string,
): Promise<string> {
  const response = await request.post('/api/v1/engrams', {
    data: {
      project_id: projectID,
      thread_id: `phase40-thread-${randomUUID()}`,
      title,
      abstract: `Phase40 curation check for ${title}`,
      detailed_summary_markdown: `Phase40 deterministic entry for ${title}`,
      tags: ['phase40', 'curation'],
      keywords: ['phase40', 'curation'],
      visibility_scope: 'project',
    },
  })
  const payload = await expectJSON<{ engram_id: string }>(response, 'phase40 engram create')
  return payload.engram_id
}

async function createContradictionLink(
  request: APIRequestContext,
  sourceEngramID: string,
  targetEngramID: string,
): Promise<void> {
  const response = await request.post(`/api/v1/engrams/${sourceEngramID}/links`, {
    data: {
      target_engram_id: targetEngramID,
      relation_type: 'contradicts',
      weight: 0.9,
      temporal_weight: 0.8,
      confidence: 0.95,
    },
  })
  await expectJSON<Record<string, unknown>>(response, 'phase40 contradiction link create')
}

When('I seed deterministic memory curation prerequisites', async ({ page }) => {
  seededProjectID = `phase40-curation-${Date.now()}`
  listedSuggestions = []
  selectedSuggestionID = null

  await createProject(seededProjectID, page.request)

  // Consolidation prerequisites: duplicate titles in the same project.
  await createEngram(page.request, seededProjectID, 'Phase40 Duplicate Insight')
  await createEngram(page.request, seededProjectID, ' phase40 duplicate insight ')

  // Contradiction prerequisites: one contradicts link pair.
  const sourceID = await createEngram(
    page.request,
    seededProjectID,
    'Always deploy with blue/green strategy.',
  )
  const targetID = await createEngram(
    page.request,
    seededProjectID,
    'Never deploy with blue/green strategy.',
  )
  await createContradictionLink(page.request, sourceID, targetID)
})

When('I refresh consolidation and contradiction workflows for memory curation', async ({ page }) => {
  if (!seededProjectID) {
    throw new Error('Missing seeded project id')
  }

  const consolidationRefresh = await page.request.post(
    '/api/v1/admin/memory/engrams/consolidation/refresh',
    {
      data: {
        project_id: seededProjectID,
        min_group_size: 2,
      },
    },
  )
  const consolidationPayload = await expectJSON<RefreshResponse>(
    consolidationRefresh,
    'phase40 consolidation refresh',
  )
  expect(consolidationPayload.updated_count).toBeGreaterThanOrEqual(1)

  const contradictionRefresh = await page.request.post(
    '/api/v1/admin/memory/engrams/contradictions/refresh',
    {
      data: { project_id: seededProjectID },
    },
  )
  const contradictionPayload = await expectJSON<RefreshResponse>(
    contradictionRefresh,
    'phase40 contradiction refresh',
  )
  expect(contradictionPayload.updated_count).toBeGreaterThanOrEqual(1)

  const listed = await page.request.get(
    `/api/v1/admin/memory/engrams/curation/suggestions?project_id=${seededProjectID}&status=suggested&limit=50&offset=0`,
  )
  listedSuggestions = await expectJSON<CurationSuggestion[]>(listed, 'phase40 curation list suggested')
  expect(listedSuggestions.length).toBeGreaterThanOrEqual(2)
})

Then('curation suggestion type coverage should include consolidate and contradiction', async () => {
  const observedTypes = new Set<string>(listedSuggestions.map((item) => item.suggestion_type))
  expect(observedTypes.has('consolidate')).toBeTruthy()
  expect(observedTypes.has('contradiction')).toBeTruthy()

  const actionable = listedSuggestions.find((item) => item.status === 'suggested')
  selectedSuggestionID = actionable?.suggestion_id ?? null
  expect(selectedSuggestionID).toBeTruthy()
})

When('I accept one curation suggestion for the seeded project', async ({ page }) => {
  if (!seededProjectID || !selectedSuggestionID) {
    throw new Error('Missing seeded project id or selected suggestion id')
  }
  const response = await page.request.post(
    `/api/v1/admin/memory/engrams/curation/suggestions/${selectedSuggestionID}/action`,
    {
      data: {
        project_id: seededProjectID,
        status: 'accepted',
      },
    },
  )
  const payload = await expectJSON<CurationSuggestion>(response, 'phase40 curation action')
  expect(payload.suggestion_id).toBe(selectedSuggestionID)
  expect(payload.status).toBe('accepted')
})

Then('accepted curation suggestions should include the actioned record', async ({ page }) => {
  if (!seededProjectID || !selectedSuggestionID) {
    throw new Error('Missing seeded project id or selected suggestion id')
  }
  const response = await page.request.get(
    `/api/v1/admin/memory/engrams/curation/suggestions?project_id=${seededProjectID}&status=accepted&limit=50&offset=0`,
  )
  const accepted = await expectJSON<CurationSuggestion[]>(response, 'phase40 curation list accepted')
  const actioned = accepted.find((entry) => entry.suggestion_id === selectedSuggestionID)
  expect(actioned).toBeTruthy()
  expect(actioned?.status).toBe('accepted')
  expect(actioned?.actioned_at).toBeTruthy()
  expect(actioned?.action_taken_by).toBeTruthy()
})
