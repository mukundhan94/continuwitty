import { useEffect, useMemo, useState } from 'react'
import type { FormEvent } from 'react'

import styled from 'styled-components'

import { createAgentRun, getAgentRun, resumeAgentRun } from '../api/agentRuns'
import type { AgentRunResponse } from '../api/types'
import { describeError } from '../utils/errors'
import {
  ErrorText,
  EyebrowText,
  GlassPane,
  MutedText,
  PaneHeader,
  ScrollColumn,
  SectionDivider,
  SplitGrid,
  SupportText,
} from '../styles/primitives'

const RECENT_AGENT_THREADS_STORAGE_KEY = 'engram.agent-run-recent-threads'
const DEFAULT_SNAPSHOT_EVERY = 3

const PageGrid = styled.main`
  display: grid;
  grid-template-columns: minmax(320px, 380px) minmax(0, 1fr);
  gap: 0.9rem;
  flex: 1;
  min-height: 0;

  @media (max-width: 1180px) {
    grid-template-columns: 1fr;
    overflow: auto;
  }
`

const Pane = styled(GlassPane)`
  min-height: 0;
`

const FormStack = styled.form`
  display: grid;
  gap: 0.72rem;
`

const Field = styled.label`
  display: grid;
  gap: 0.3rem;
`

const FieldHint = styled.span`
  font-size: 0.76rem;
  letter-spacing: 0.01em;
  text-transform: none;
  font-weight: 500;
  color: var(--color-ink-muted);
`

const CheckboxRow = styled.label`
  display: flex;
  align-items: center;
  gap: 0.5rem;
  text-transform: none;

  input {
    width: auto;
  }
`

const ThreadRow = styled.div`
  display: grid;
  grid-template-columns: minmax(0, 1fr) auto;
  gap: 0.45rem;
  align-items: start;

  @media (max-width: 680px) {
    grid-template-columns: 1fr;
  }
`

const RecentThreadRow = styled.div`
  display: flex;
  flex-wrap: wrap;
  gap: 0.45rem;
`

const GhostButton = styled.button`
  background: var(--surface-mute);
  color: var(--color-ink);
  box-shadow: none;
  border: 1px solid var(--color-line);

  &:hover {
    box-shadow: none;
  }
`

const ThreadChip = styled.button`
  background: var(--surface-raised);
  color: var(--color-ink);
  box-shadow: none;
  border: 1px solid var(--color-line);
  padding-inline: 0.75rem;

  &:hover {
    box-shadow: none;
  }
`

const SummaryGrid = styled.div`
  display: grid;
  grid-template-columns: repeat(4, minmax(0, 1fr));
  gap: 0.55rem;

  @media (max-width: 1040px) {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }

  @media (max-width: 560px) {
    grid-template-columns: 1fr;
  }
`

const SummaryCard = styled.div`
  border-radius: 16px;
  border: 1px solid var(--color-line);
  background: var(--surface-raised);
  padding: 0.8rem;
  display: grid;
  gap: 0.2rem;

  span {
    font-size: 0.72rem;
    text-transform: uppercase;
    letter-spacing: 0.08em;
    color: var(--color-ink-muted);
  }

  strong {
    font-size: 1rem;
    line-height: 1.3;
    color: var(--color-ink);
  }
`

const Badge = styled.span<{ $tone?: 'default' | 'success' }>`
  display: inline-flex;
  align-items: center;
  justify-content: center;
  min-height: 1.8rem;
  width: fit-content;
  border-radius: 9999px;
  padding: 0.2rem 0.68rem;
  border: 1px solid ${({ $tone }) => ($tone === 'success' ? 'var(--color-notice-border)' : 'var(--color-line)')};
  background: ${({ $tone }) => ($tone === 'success' ? 'var(--color-notice-bg)' : 'var(--surface-mute)')};
  color: ${({ $tone }) => ($tone === 'success' ? 'var(--color-success)' : 'var(--color-ink)')};
  font-size: 0.74rem;
  letter-spacing: 0.06em;
  text-transform: uppercase;
  font-weight: 700;
`

const ThreadMeta = styled.div`
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 0.8rem;
  flex-wrap: wrap;
`

const DetailBlock = styled.div`
  display: grid;
  gap: 0.38rem;
  border: 1px solid var(--color-line);
  border-radius: 16px;
  background: var(--surface-raised);
  padding: 0.85rem;

  h3 {
    margin: 0;
    font-size: 0.92rem;
    color: var(--color-ink);
  }

  ul {
    margin: 0;
    padding-left: 1rem;
    display: grid;
    gap: 0.28rem;
    color: var(--color-ink-muted);
  }

  p {
    margin: 0;
    color: var(--color-ink-muted);
    white-space: pre-wrap;
  }
`

const IdList = styled.div`
  display: flex;
  flex-wrap: wrap;
  gap: 0.42rem;
`

const IdPill = styled.code`
  display: inline-flex;
  align-items: center;
  border-radius: 9999px;
  border: 1px solid var(--color-line);
  background: var(--surface-mute);
  padding: 0.28rem 0.55rem;
  font-size: 0.75rem;
  color: var(--color-ink);
`

function buildThreadIdSeed(projectId: string): string {
  const prefix = projectId.trim().toLowerCase().replace(/[^a-z0-9]+/g, '-').replace(/(^-|-$)/g, '') || 'agent'
  const stamp = new Date().toISOString().slice(0, 16).replace(/[:T]/g, '-').toLowerCase()
  return `${prefix}-run-${stamp}`
}

function parseLineList(input: string): string[] {
  return input
    .split('\n')
    .map((item) => item.trim())
    .filter(Boolean)
}

function parseTokenList(input: string): string[] {
  return input
    .split(/[\n,]+/)
    .map((item) => item.trim())
    .filter(Boolean)
}

function loadRecentThreads(): string[] {
  if (typeof window === 'undefined') {
    return []
  }
  try {
    const rawValue = window.localStorage.getItem(RECENT_AGENT_THREADS_STORAGE_KEY)
    if (!rawValue) {
      return []
    }
    const parsed = JSON.parse(rawValue) as unknown
    if (!Array.isArray(parsed)) {
      return []
    }
    return parsed.filter((item): item is string => typeof item === 'string' && item.trim().length > 0)
  } catch {
    return []
  }
}

function writeRecentThreads(nextThreads: string[]): void {
  if (typeof window === 'undefined') {
    return
  }
  window.localStorage.setItem(RECENT_AGENT_THREADS_STORAGE_KEY, JSON.stringify(nextThreads.slice(0, 8)))
}

interface AgentRunsPageProps {
  projectId: string
  onProjectChange: (projectId: string) => void
  selectedThreadId?: string | null
  onThreadSelect: (threadId: string) => void
  onNotice: (message: string) => void
}

export function AgentRunsPage({
  projectId,
  onProjectChange,
  selectedThreadId,
  onThreadSelect,
  onNotice,
}: AgentRunsPageProps) {
  const [threadIdInput, setThreadIdInput] = useState(() => buildThreadIdSeed(projectId))
  const [objective, setObjective] = useState('Capture durable work state for a multi-step agent run')
  const [notesInput, setNotesInput] = useState('')
  const [assumptionsInput, setAssumptionsInput] = useState('')
  const [tagsInput, setTagsInput] = useState('research, durable-memory')
  const [keywordsInput, setKeywordsInput] = useState('continuity, checkpoint, engram')
  const [snapshotEnabled, setSnapshotEnabled] = useState(true)
  const [snapshotEvery, setSnapshotEvery] = useState(DEFAULT_SNAPSHOT_EVERY)
  const [autoPersistEngram, setAutoPersistEngram] = useState(true)
  const [creating, setCreating] = useState(false)
  const [createError, setCreateError] = useState<string | null>(null)

  const [lookupThreadId, setLookupThreadId] = useState(selectedThreadId ?? '')
  const [resumeNotesInput, setResumeNotesInput] = useState('')
  const [resumeAssumptionsInput, setResumeAssumptionsInput] = useState('')
  const [resumeTagsInput, setResumeTagsInput] = useState('')
  const [resumeKeywordsInput, setResumeKeywordsInput] = useState('')
  const [resumeSnapshotEnabled, setResumeSnapshotEnabled] = useState<boolean | ''>('')
  const [resumeAutoPersistEngram, setResumeAutoPersistEngram] = useState<boolean | ''>('')
  const [resumeSnapshotEvery, setResumeSnapshotEvery] = useState<number | ''>('')
  const [loadingRun, setLoadingRun] = useState(false)
  const [runError, setRunError] = useState<string | null>(null)
  const [resuming, setResuming] = useState(false)
  const [resumeError, setResumeError] = useState<string | null>(null)
  const [runResponse, setRunResponse] = useState<AgentRunResponse | null>(null)
  const [recentThreads, setRecentThreads] = useState<string[]>(() => loadRecentThreads())

  useEffect(() => {
    setLookupThreadId(selectedThreadId ?? '')
  }, [selectedThreadId])

  useEffect(() => {
    if (!selectedThreadId) {
      setRunResponse(null)
      setRunError(null)
      return
    }

    let cancelled = false

    const load = async () => {
      setLoadingRun(true)
      setRunError(null)
      try {
        const response = await getAgentRun(selectedThreadId)
        if (!cancelled) {
          setRunResponse(response)
        }
      } catch (error) {
        if (!cancelled) {
          setRunError(describeError(error))
          setRunResponse(null)
        }
      } finally {
        if (!cancelled) {
          setLoadingRun(false)
        }
      }
    }

    void load()
    return () => {
      cancelled = true
    }
  }, [selectedThreadId])

  const hasActiveRun = Boolean(runResponse)
  const activeState = runResponse?.state ?? null
  const activeDecisions = activeState?.decisions ?? []
  const activeOpenQuestions = activeState?.open_questions ?? []
  const activeSnapshotEngramIds = activeState?.snapshot_engram_ids ?? runResponse?.snapshot_engram_ids ?? []
  const recentThreadItems = useMemo(() => recentThreads.filter((threadId) => threadId !== selectedThreadId), [recentThreads, selectedThreadId])

  const rememberThread = (threadId: string) => {
    const normalized = threadId.trim()
    if (!normalized) {
      return
    }
    setRecentThreads((current) => {
      const next = [normalized, ...current.filter((item) => item !== normalized)].slice(0, 8)
      writeRecentThreads(next)
      return next
    })
  }

  const handleCreateRun = async (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault()
    setCreating(true)
    setCreateError(null)
    try {
      const response = await createAgentRun({
        project_id: projectId.trim(),
        thread_id: threadIdInput.trim(),
        objective: objective.trim(),
        notes: parseLineList(notesInput),
        assumptions: parseLineList(assumptionsInput),
        tags: parseTokenList(tagsInput),
        keywords: parseTokenList(keywordsInput),
        snapshot_enabled: snapshotEnabled,
        snapshot_every_n_notes: snapshotEvery,
        auto_persist_engram: autoPersistEngram,
      })
      rememberThread(response.thread_id)
      setRunResponse(response)
      setLookupThreadId(response.thread_id)
      onThreadSelect(response.thread_id)
      onNotice(`Agent run ${response.thread_id} is ready.`)
      setNotesInput('')
      setAssumptionsInput('')
      setThreadIdInput(buildThreadIdSeed(projectId))
    } catch (error) {
      setCreateError(describeError(error))
    } finally {
      setCreating(false)
    }
  }

  const handleOpenThread = () => {
    const normalized = lookupThreadId.trim()
    if (!normalized) {
      setRunError('Enter a thread ID to inspect an existing durable run.')
      return
    }
    setRunError(null)
    rememberThread(normalized)
    onThreadSelect(normalized)
  }

  const handleResumeRun = async (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault()
    if (!activeState) {
      setResumeError('Open a durable run before sending resume notes.')
      return
    }
    setResuming(true)
    setResumeError(null)
    try {
      const response = await resumeAgentRun(activeState.thread_id, {
        notes: parseLineList(resumeNotesInput),
        assumptions: parseLineList(resumeAssumptionsInput),
        tags: parseTokenList(resumeTagsInput),
        keywords: parseTokenList(resumeKeywordsInput),
        snapshot_enabled: resumeSnapshotEnabled === '' ? undefined : resumeSnapshotEnabled,
        auto_persist_engram: resumeAutoPersistEngram === '' ? undefined : resumeAutoPersistEngram,
        snapshot_every_n_notes: resumeSnapshotEvery === '' ? undefined : resumeSnapshotEvery,
      })
      rememberThread(response.thread_id)
      setRunResponse(response)
      onNotice(`Agent run ${response.thread_id} resumed.`)
      setResumeNotesInput('')
      setResumeAssumptionsInput('')
      setResumeTagsInput('')
      setResumeKeywordsInput('')
      setResumeSnapshotEnabled('')
      setResumeAutoPersistEngram('')
      setResumeSnapshotEvery('')
    } catch (error) {
      setResumeError(describeError(error))
    } finally {
      setResuming(false)
    }
  }

  return (
    <PageGrid data-testid="agent-runs-page">
      <Pane>
        <PaneHeader>
          <div>
            <h2 className="font-display text-base font-semibold tracking-[0.02em] text-ink">Durable Agent Runs</h2>
            <MutedText>Start long-running threads that can checkpoint, resume, and persist engrams.</MutedText>
          </div>
          <Badge $tone="success">Agent-ready</Badge>
        </PaneHeader>

        <FormStack onSubmit={handleCreateRun}>
          <Field>
            Project ID
            <input
              data-testid="agent-run-project-id"
              value={projectId}
              onChange={(event) => onProjectChange(event.target.value)}
              placeholder="engram-vault"
            />
          </Field>
          <Field>
            Thread ID
            <input
              data-testid="agent-run-thread-id"
              value={threadIdInput}
              onChange={(event) => setThreadIdInput(event.target.value)}
              placeholder="engram-vault-run-2026-03-06-10-00"
            />
            <FieldHint>Stable thread IDs make resume and external agent coordination predictable.</FieldHint>
          </Field>
          <Field>
            Objective
            <textarea
              data-testid="agent-run-objective"
              value={objective}
              onChange={(event) => setObjective(event.target.value)}
              placeholder="What should this run achieve?"
            />
          </Field>
          <Field>
            Initial Notes
            <textarea
              data-testid="agent-run-notes"
              value={notesInput}
              onChange={(event) => setNotesInput(event.target.value)}
              placeholder="One note per line"
            />
          </Field>
          <SplitGrid>
            <Field>
              Assumptions
              <textarea
                data-testid="agent-run-assumptions"
                value={assumptionsInput}
                onChange={(event) => setAssumptionsInput(event.target.value)}
                placeholder="One assumption per line"
              />
            </Field>
            <div>
              <Field>
                Tags
                <input
                  data-testid="agent-run-tags"
                  value={tagsInput}
                  onChange={(event) => setTagsInput(event.target.value)}
                  placeholder="comma,separated,tags"
                />
              </Field>
              <Field>
                Keywords
                <input
                  data-testid="agent-run-keywords"
                  value={keywordsInput}
                  onChange={(event) => setKeywordsInput(event.target.value)}
                  placeholder="checkpoint,rehydration,continuity"
                />
              </Field>
            </div>
          </SplitGrid>
          <SplitGrid>
            <CheckboxRow>
              <input
                data-testid="agent-run-snapshot-enabled"
                type="checkbox"
                checked={snapshotEnabled}
                onChange={(event) => setSnapshotEnabled(event.target.checked)}
              />
              Enable periodic snapshots
            </CheckboxRow>
            <CheckboxRow>
              <input
                data-testid="agent-run-auto-persist"
                type="checkbox"
                checked={autoPersistEngram}
                onChange={(event) => setAutoPersistEngram(event.target.checked)}
              />
              Auto-persist final engram
            </CheckboxRow>
          </SplitGrid>
          <Field>
            Snapshot Every N Notes
            <input
              data-testid="agent-run-snapshot-every"
              type="number"
              min={1}
              value={snapshotEvery}
              onChange={(event) => setSnapshotEvery(Math.max(1, Number(event.target.value) || DEFAULT_SNAPSHOT_EVERY))}
            />
          </Field>
          {createError ? <ErrorText>{createError}</ErrorText> : null}
          <button type="submit" disabled={creating}>{creating ? 'Starting Run...' : 'Start Durable Run'}</button>
        </FormStack>
      </Pane>

      <Pane>
        <PaneHeader>
          <div>
            <h2 className="font-display text-base font-semibold tracking-[0.02em] text-ink">Thread Inspector</h2>
            <MutedText>Open any durable run by thread ID, review synthesis state, and append resume notes.</MutedText>
          </div>
        </PaneHeader>

        <ThreadRow>
          <Field>
            Thread ID
            <input
              data-testid="agent-run-lookup"
              value={lookupThreadId}
              onChange={(event) => setLookupThreadId(event.target.value)}
              placeholder="thread-manual-001"
            />
          </Field>
          <GhostButton type="button" onClick={handleOpenThread}>Open Thread</GhostButton>
        </ThreadRow>

        {recentThreadItems.length > 0 ? (
          <div>
            <EyebrowText>Recent Threads</EyebrowText>
            <RecentThreadRow>
              {recentThreadItems.map((threadId) => (
                <ThreadChip key={threadId} type="button" onClick={() => {
                  setLookupThreadId(threadId)
                  rememberThread(threadId)
                  onThreadSelect(threadId)
                }}>
                  {threadId}
                </ThreadChip>
              ))}
            </RecentThreadRow>
          </div>
        ) : null}

        {runError ? <ErrorText>{runError}</ErrorText> : null}
        {loadingRun ? <MutedText>Loading durable run state...</MutedText> : null}
        {!loadingRun && !hasActiveRun && !runError ? (
          <SupportText>Open a thread to inspect checkpointed state, generated synthesis, and snapshot engrams.</SupportText>
        ) : null}

        {activeState ? (
          <>
            <SectionDivider>
              <ThreadMeta>
                <div>
                  <EyebrowText>{activeState.project_id}</EyebrowText>
                  <h2 className="font-display text-xl font-semibold text-ink">{activeState.objective}</h2>
                  <MutedText>{activeState.thread_id}</MutedText>
                </div>
                <Badge>{runResponse?.status ?? activeState.status}</Badge>
              </ThreadMeta>
            </SectionDivider>

            <SummaryGrid>
              <SummaryCard>
                <span>Snapshots</span>
                <strong>{String(activeState.snapshot_count)}</strong>
              </SummaryCard>
              <SummaryCard>
                <span>Auto Persist</span>
                <strong>{activeState.auto_persist_engram ? 'On' : 'Off'}</strong>
              </SummaryCard>
              <SummaryCard>
                <span>Final Engram</span>
                <strong>{runResponse?.engram_id ?? 'Pending'}</strong>
              </SummaryCard>
              <SummaryCard>
                <span>Snapshot Cadence</span>
                <strong>Every {activeState.snapshot_every_n_notes}</strong>
              </SummaryCard>
            </SummaryGrid>

            <ScrollColumn>
              {activeState.synthesis_abstract ? (
                <DetailBlock>
                  <h3>Synthesis Abstract</h3>
                  <p>{activeState.synthesis_abstract}</p>
                </DetailBlock>
              ) : null}

              {activeState.synthesis_markdown ? (
                <DetailBlock>
                  <h3>Synthesis Markdown</h3>
                  <p>{activeState.synthesis_markdown}</p>
                </DetailBlock>
              ) : null}

              {activeDecisions.length > 0 ? (
                <DetailBlock>
                  <h3>Decisions</h3>
                  <ul>
                    {activeDecisions.map((decision) => (
                      <li key={`${decision.decision}-${decision.rationale}`}>
                        <strong>{decision.decision}</strong>: {decision.rationale}
                      </li>
                    ))}
                  </ul>
                </DetailBlock>
              ) : null}

              {activeOpenQuestions.length > 0 ? (
                <DetailBlock>
                  <h3>Open Questions</h3>
                  <ul>
                    {activeOpenQuestions.map((question) => (
                      <li key={question}>{question}</li>
                    ))}
                  </ul>
                </DetailBlock>
              ) : null}

              {activeSnapshotEngramIds.length > 0 ? (
                <DetailBlock>
                  <h3>Snapshot Engrams</h3>
                  <IdList>
                    {activeSnapshotEngramIds.map((engramId) => (
                      <IdPill key={engramId}>{engramId}</IdPill>
                    ))}
                  </IdList>
                </DetailBlock>
              ) : null}
            </ScrollColumn>

            <SectionDivider>
              <PaneHeader>
                <div>
                  <h2 className="font-display text-base font-semibold tracking-[0.02em] text-ink">Resume Thread</h2>
                  <MutedText>Append new notes and optionally override persistence behavior for the next checkpoint.</MutedText>
                </div>
              </PaneHeader>
              <FormStack onSubmit={handleResumeRun}>
                <Field>
                  Resume Notes
                  <textarea
                    data-testid="agent-run-resume-notes"
                    value={resumeNotesInput}
                    onChange={(event) => setResumeNotesInput(event.target.value)}
                    placeholder="One note per line"
                  />
                </Field>
                <SplitGrid>
                  <Field>
                    Resume Assumptions
                    <textarea
                      data-testid="agent-run-resume-assumptions"
                      value={resumeAssumptionsInput}
                      onChange={(event) => setResumeAssumptionsInput(event.target.value)}
                      placeholder="Only include new assumptions"
                    />
                  </Field>
                  <div>
                    <Field>
                      Resume Tags
                      <input
                        data-testid="agent-run-resume-tags"
                        value={resumeTagsInput}
                        onChange={(event) => setResumeTagsInput(event.target.value)}
                        placeholder="comma,separated,tags"
                      />
                    </Field>
                    <Field>
                      Resume Keywords
                      <input
                        data-testid="agent-run-resume-keywords"
                        value={resumeKeywordsInput}
                        onChange={(event) => setResumeKeywordsInput(event.target.value)}
                        placeholder="new,keywords"
                      />
                    </Field>
                  </div>
                </SplitGrid>
                <SplitGrid>
                  <Field>
                    Snapshot Override
                    <select
                      data-testid="agent-run-resume-snapshot-enabled"
                      value={resumeSnapshotEnabled === '' ? 'inherit' : resumeSnapshotEnabled ? 'enabled' : 'disabled'}
                      onChange={(event) => {
                        const value = event.target.value
                        setResumeSnapshotEnabled(value === 'inherit' ? '' : value === 'enabled')
                      }}
                    >
                      <option value="inherit">Inherit current</option>
                      <option value="enabled">Enable snapshots</option>
                      <option value="disabled">Disable snapshots</option>
                    </select>
                  </Field>
                  <Field>
                    Auto Persist Override
                    <select
                      data-testid="agent-run-resume-auto-persist"
                      value={resumeAutoPersistEngram === '' ? 'inherit' : resumeAutoPersistEngram ? 'enabled' : 'disabled'}
                      onChange={(event) => {
                        const value = event.target.value
                        setResumeAutoPersistEngram(value === 'inherit' ? '' : value === 'enabled')
                      }}
                    >
                      <option value="inherit">Inherit current</option>
                      <option value="enabled">Auto-persist final engram</option>
                      <option value="disabled">Skip final engram</option>
                    </select>
                  </Field>
                </SplitGrid>
                <Field>
                  Snapshot Every N Notes Override
                  <input
                    data-testid="agent-run-resume-snapshot-every"
                    type="number"
                    min={1}
                    value={resumeSnapshotEvery}
                    onChange={(event) => {
                      const nextValue = event.target.value
                      setResumeSnapshotEvery(nextValue === '' ? '' : Math.max(1, Number(nextValue) || DEFAULT_SNAPSHOT_EVERY))
                    }}
                    placeholder={String(activeState.snapshot_every_n_notes)}
                  />
                </Field>
                {resumeError ? <ErrorText>{resumeError}</ErrorText> : null}
                <button type="submit" disabled={resuming}>{resuming ? 'Resuming...' : 'Resume Thread'}</button>
              </FormStack>
            </SectionDivider>
          </>
        ) : null}
      </Pane>
    </PageGrid>
  )
}
