import { randomUUID } from 'node:crypto'

import { type APIRequestContext, type APIResponse } from '@playwright/test'

import { Then, When, expect } from '../support/fixtures'

type CurationSuggestion = {
  suggestion_id: string
  project_id: string
  suggestion_type: string
  status: string
  payload_json?: Record<string, unknown>
  actioned_at?: string | null
  action_taken_by?: string | null
}

type RefreshResponse = {
  updated_count: number
}

type ConsolidationSuggestion = {
  suggestion_id: string
  status: string
}

type ContradictionAlert = {
  alert_id: string
  status: string
  resolved_at?: string | null
  resolved_by?: string | null
}

let seededProjectID: string | null = null
let listedSuggestions: CurationSuggestion[] = []
let selectedSuggestionID: string | null = null
let selectedConsolidationCurationSuggestionID: string | null = null
let selectedContradictionCurationSuggestionID: string | null = null
let selectedLinkCurationSuggestionID: string | null = null
let targetConsolidationSuggestionID: string | null = null
let targetContradictionAlertID: string | null = null
let seededLinkSourceEngramID: string | null = null

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

type CreateContradictionLinkInput = {
  request: APIRequestContext
  sourceEngramID: string
  targetEngramID: string
  relationType: string
}

type ActionCurationSuggestionInput = {
  request: APIRequestContext
  projectID: string
  suggestionID: string
  status: 'accepted' | 'applied' | 'rejected'
}

type AppliedCurationIDs = {
  projectID: string
  consolidationCurationSuggestionID: string
  contradictionCurationSuggestionID: string
}

type DownstreamTargets = {
  projectID: string
  consolidationSuggestionID: string
  contradictionAlertID: string
}

type SelectedCurationContext = {
  projectID: string
  suggestionID: string
}

type ActionedCurationAssertionInput = {
  request: APIRequestContext
  context: SelectedCurationContext
  status: 'accepted' | 'applied'
  listContext: string
}

async function expectJSON<T>(input: ExpectJSONInput): Promise<T> {
  const payload = await input.response.text()
  if (!input.response.ok()) {
    throw new Error(`${input.context} failed (${input.response.status()}): ${payload}`)
  }
  return JSON.parse(payload) as T
}

async function createProject(input: CreateProjectInput): Promise<void> {
  const response = await input.request.post('/api/v1/projects', {
    data: {
      project_id: input.projectID,
      name: input.projectID,
      description: 'Phase40 curation acceptance project',
    },
  })
  await expectJSON<Record<string, unknown>>({ response, context: 'phase40 project create' })
}

async function createEngram(input: CreateEngramInput): Promise<string> {
  const response = await input.request.post('/api/v1/engrams', {
    data: {
      project_id: input.projectID,
      thread_id: `phase40-thread-${randomUUID()}`,
      title: input.title,
      abstract: `Phase40 curation check for ${input.title}`,
      detailed_summary_markdown: `Phase40 deterministic entry for ${input.title}`,
      tags: ['phase40', 'curation'],
      keywords: ['phase40', 'curation'],
      visibility_scope: 'project',
    },
  })
  const payload = await expectJSON<{ engram_id: string }>({
    response,
    context: 'phase40 engram create',
  })
  return payload.engram_id
}

async function createContradictionLink(input: CreateContradictionLinkInput): Promise<void> {
  const response = await input.request.post(`/api/v1/engrams/${input.sourceEngramID}/links`, {
    data: {
      target_engram_id: input.targetEngramID,
      relation_type: input.relationType,
      weight: 0.9,
      temporal_weight: 0.8,
      confidence: 0.95,
    },
  })
  await expectJSON<Record<string, unknown>>({
    response,
    context: 'phase40 contradiction link create',
  })
}

async function actionCurationSuggestion(input: ActionCurationSuggestionInput): Promise<CurationSuggestion> {
  const response = await input.request.post(
    `/api/v1/admin/memory/engrams/curation/suggestions/${input.suggestionID}/action`,
    {
      data: {
        project_id: input.projectID,
        status: input.status,
      },
    },
  )
  return expectJSON<CurationSuggestion>({
    response,
    context: 'phase40 curation action',
  })
}

async function applyCurationSuggestionStatus(
  request: APIRequestContext,
  context: SelectedCurationContext,
  status: 'accepted' | 'applied',
): Promise<void> {
  const payload = await actionCurationSuggestion({
    request,
    projectID: context.projectID,
    suggestionID: context.suggestionID,
    status,
  })
  expect(payload.suggestion_id).toBe(context.suggestionID)
  expect(payload.status).toBe(status)
}

async function assertActionedCurationSuggestion(input: ActionedCurationAssertionInput): Promise<void> {
  const response = await input.request.get(
    `/api/v1/admin/memory/engrams/curation/suggestions?project_id=${input.context.projectID}&status=${input.status}&limit=50&offset=0`,
  )
  const listed = await expectJSON<CurationSuggestion[]>({
    response,
    context: input.listContext,
  })
  const actioned = listed.find((entry) => entry.suggestion_id === input.context.suggestionID)
  expect(actioned).toBeTruthy()
  expect(actioned?.status).toBe(input.status)
  expect(actioned?.actioned_at).toBeTruthy()
  expect(actioned?.action_taken_by).toBeTruthy()
}

function requirePayloadString(suggestion: CurationSuggestion, key: string): string {
  const value = suggestion.payload_json?.[key]
  if (typeof value !== 'string' || value.trim() === '') {
    throw new Error(`Missing ${key} in curation suggestion payload`)
  }
  return value
}

function requireSeededProjectID(): string {
  if (!seededProjectID) {
    throw new Error('Missing seeded project id')
  }
  return seededProjectID
}

function requireAppliedCurationIDs(): AppliedCurationIDs {
  return {
    projectID: requireSeededProjectID(),
    consolidationCurationSuggestionID: requireStringID(
      selectedConsolidationCurationSuggestionID,
      'Missing consolidation curation suggestion id',
    ),
    contradictionCurationSuggestionID: requireStringID(
      selectedContradictionCurationSuggestionID,
      'Missing contradiction curation suggestion id',
    ),
  }
}

function requireSelectedCurationContext(): SelectedCurationContext {
  return {
    projectID: requireSeededProjectID(),
    suggestionID: requireStringID(selectedSuggestionID, 'Missing selected curation suggestion id'),
  }
}

function requireLinkCurationContext(): SelectedCurationContext {
  return {
    projectID: requireSeededProjectID(),
    suggestionID: requireStringID(
      selectedLinkCurationSuggestionID,
      'Missing link curation suggestion id',
    ),
  }
}

function requireDownstreamTargets(): DownstreamTargets {
  return {
    projectID: requireSeededProjectID(),
    consolidationSuggestionID: requireStringID(
      targetConsolidationSuggestionID,
      'Missing consolidation suggestion id',
    ),
    contradictionAlertID: requireStringID(targetContradictionAlertID, 'Missing contradiction alert id'),
  }
}

function requireStringID(value: string | null, message: string): string {
  if (!value) {
    throw new Error(message)
  }
  return value
}

async function assertMergedConsolidationSuggestion(
  request: APIRequestContext,
  projectID: string,
  suggestionID: string,
): Promise<void> {
  const consolidationResponse = await request.get(
    `/api/v1/admin/memory/engrams/consolidation/suggestions?project_id=${projectID}&status=merged&limit=50&offset=0`,
  )
  const merged = await expectJSON<ConsolidationSuggestion[]>({
    response: consolidationResponse,
    context: 'phase40 consolidation list merged',
  })
  const mergedRecord = merged.find((entry) => entry.suggestion_id === suggestionID)
  expect(mergedRecord).toBeTruthy()
  expect(mergedRecord?.status).toBe('merged')
}

async function assertResolvedContradictionAlert(
  request: APIRequestContext,
  projectID: string,
  alertID: string,
): Promise<void> {
  const contradictionResponse = await request.get(
    `/api/v1/admin/memory/engrams/contradictions/alerts?project_id=${projectID}&status=resolved&limit=50&offset=0`,
  )
  const resolved = await expectJSON<ContradictionAlert[]>({
    response: contradictionResponse,
    context: 'phase40 contradiction list resolved',
  })
  const resolvedRecord = resolved.find((entry) => entry.alert_id === alertID)
  expect(resolvedRecord).toBeTruthy()
  expect(resolvedRecord?.status).toBe('resolved')
  expect(resolvedRecord?.resolved_at).toBeTruthy()
  expect(resolvedRecord?.resolved_by).toBeTruthy()
}

When('I seed deterministic memory curation prerequisites', async ({ page }) => {
  seededProjectID = `phase40-curation-${Date.now()}`
  listedSuggestions = []
  selectedSuggestionID = null
  selectedConsolidationCurationSuggestionID = null
  selectedContradictionCurationSuggestionID = null
  selectedLinkCurationSuggestionID = null
  targetConsolidationSuggestionID = null
  targetContradictionAlertID = null
  seededLinkSourceEngramID = null

  await createProject({ projectID: seededProjectID, request: page.request })

  // Consolidation prerequisites: duplicate titles in the same project.
  await createEngram({
    request: page.request,
    projectID: seededProjectID,
    title: 'Phase40 Duplicate Insight',
  })
  await createEngram({
    request: page.request,
    projectID: seededProjectID,
    title: ' phase40 duplicate insight ',
  })

  // Contradiction prerequisites: one contradicts link pair.
  const sourceID = await createEngram({
    request: page.request,
    projectID: seededProjectID,
    title: 'Always deploy with blue/green strategy.',
  })
  const targetID = await createEngram({
    request: page.request,
    projectID: seededProjectID,
    title: 'Never deploy with blue/green strategy.',
  })
  await createContradictionLink({
    request: page.request,
    sourceEngramID: sourceID,
    targetEngramID: targetID,
    relationType: 'contradicts',
  })
  await createContradictionLink({
    request: page.request,
    sourceEngramID: sourceID,
    targetEngramID: targetID,
    relationType: 'supports',
  })
  seededLinkSourceEngramID = sourceID
})

When('I refresh consolidation and contradiction workflows for memory curation', async ({ page }) => {
  const projectID = requireSeededProjectID()

  const consolidationRefresh = await page.request.post(
    '/api/v1/admin/memory/engrams/consolidation/refresh',
    {
      data: {
        project_id: projectID,
        min_group_size: 2,
      },
    },
  )
  const consolidationPayload = await expectJSON<RefreshResponse>({
    response: consolidationRefresh,
    context: 'phase40 consolidation refresh',
  })
  expect(consolidationPayload.updated_count).toBeGreaterThanOrEqual(1)

  const contradictionRefresh = await page.request.post(
    '/api/v1/admin/memory/engrams/contradictions/refresh',
    {
      data: { project_id: projectID },
    },
  )
  const contradictionPayload = await expectJSON<RefreshResponse>({
    response: contradictionRefresh,
    context: 'phase40 contradiction refresh',
  })
  expect(contradictionPayload.updated_count).toBeGreaterThanOrEqual(1)

  const listed = await page.request.get(
    `/api/v1/admin/memory/engrams/curation/suggestions?project_id=${projectID}&status=suggested&limit=50&offset=0`,
  )
  listedSuggestions = await expectJSON<CurationSuggestion[]>({
    response: listed,
    context: 'phase40 curation list suggested',
  })
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
  const context = requireSelectedCurationContext()
  await applyCurationSuggestionStatus(page.request, context, 'accepted')
})

Then('accepted curation suggestions should include the actioned record', async ({ page }) => {
  await assertActionedCurationSuggestion({
    request: page.request,
    context: requireSelectedCurationContext(),
    status: 'accepted',
    listContext: 'phase40 curation list accepted',
  })
})

Then('curation suggestion payloads should include consolidation and contradiction references', async () => {
  const consolidation = listedSuggestions.find(
    (entry) => entry.suggestion_type === 'consolidate' && entry.status === 'suggested',
  )
  const contradiction = listedSuggestions.find(
    (entry) => entry.suggestion_type === 'contradiction' && entry.status === 'suggested',
  )
  expect(consolidation).toBeTruthy()
  expect(contradiction).toBeTruthy()
  if (!consolidation || !contradiction) {
    throw new Error('Missing actionable consolidation or contradiction curation suggestions')
  }

  selectedConsolidationCurationSuggestionID = consolidation.suggestion_id
  selectedContradictionCurationSuggestionID = contradiction.suggestion_id
  targetConsolidationSuggestionID = requirePayloadString(
    consolidation,
    'consolidation_suggestion_id',
  )
  targetContradictionAlertID = requirePayloadString(contradiction, 'contradiction_alert_id')
})

When('I apply consolidation and contradiction curation suggestions for the seeded project', async ({ page }) => {
  const ids = requireAppliedCurationIDs()

  const consolidationApplied = await actionCurationSuggestion({
    request: page.request,
    projectID: ids.projectID,
    suggestionID: ids.consolidationCurationSuggestionID,
    status: 'applied',
  })
  expect(consolidationApplied.status).toBe('applied')

  const contradictionApplied = await actionCurationSuggestion({
    request: page.request,
    projectID: ids.projectID,
    suggestionID: ids.contradictionCurationSuggestionID,
    status: 'applied',
  })
  expect(contradictionApplied.status).toBe('applied')
})

Then('applied curation suggestions should include both actioned records', async ({ page }) => {
  const ids = requireAppliedCurationIDs()
  const response = await page.request.get(
    `/api/v1/admin/memory/engrams/curation/suggestions?project_id=${ids.projectID}&status=applied&limit=50&offset=0`,
  )
  const applied = await expectJSON<CurationSuggestion[]>({
    response,
    context: 'phase40 curation list applied',
  })
  const consolidation = applied.find(
    (entry) => entry.suggestion_id === ids.consolidationCurationSuggestionID,
  )
  const contradiction = applied.find(
    (entry) => entry.suggestion_id === ids.contradictionCurationSuggestionID,
  )
  expect(consolidation).toBeTruthy()
  expect(contradiction).toBeTruthy()
  expect(consolidation?.actioned_at).toBeTruthy()
  expect(contradiction?.actioned_at).toBeTruthy()
})

Then('downstream consolidation and contradiction records should be actioned', async ({ page }) => {
  const targets = requireDownstreamTargets()
  await assertMergedConsolidationSuggestion(
    page.request,
    targets.projectID,
    targets.consolidationSuggestionID,
  )
  await assertResolvedContradictionAlert(
    page.request,
    targets.projectID,
    targets.contradictionAlertID,
  )
})

When(
  'I refresh link hygiene curation suggestions for the seeded source engram',
  async ({ page }) => {
    const sourceEngramID = requireStringID(
      seededLinkSourceEngramID,
      'Missing seeded link source engram id',
    )
    const refresh = await page.request.post(
      `/api/v1/admin/memory/engrams/${sourceEngramID}/links/curation/refresh`,
      { data: { include_archived: false, limit: 50 } },
    )
    const refreshPayload = await expectJSON<RefreshResponse>({
      response: refresh,
      context: 'phase40 link curation refresh',
    })
    expect(refreshPayload.updated_count).toBeGreaterThanOrEqual(1)

    const projectID = requireSeededProjectID()
    const listed = await page.request.get(
      `/api/v1/admin/memory/engrams/curation/suggestions?project_id=${projectID}&suggestion_type=link&status=suggested&limit=50&offset=0`,
    )
    listedSuggestions = await expectJSON<CurationSuggestion[]>({
      response: listed,
      context: 'phase40 link curation list suggested',
    })
    const selected = listedSuggestions.find(
      (entry) => entry.suggestion_type === 'link' && entry.status === 'suggested',
    )
    selectedLinkCurationSuggestionID = selected?.suggestion_id ?? null
    expect(selectedLinkCurationSuggestionID).toBeTruthy()
  },
)

Then('curation suggestion type coverage should include link', async () => {
  const observedTypes = new Set<string>(listedSuggestions.map((item) => item.suggestion_type))
  expect(observedTypes.has('link')).toBeTruthy()
})

When('I apply one link curation suggestion for the seeded project', async ({ page }) => {
  const context = requireLinkCurationContext()
  await applyCurationSuggestionStatus(page.request, context, 'applied')
})

Then('applied curation suggestions should include the link actioned record', async ({ page }) => {
  await assertActionedCurationSuggestion({
    request: page.request,
    context: requireLinkCurationContext(),
    status: 'applied',
    listContext: 'phase40 link curation list applied',
  })
})
