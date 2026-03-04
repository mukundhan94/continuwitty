import { useMemo, useState } from 'react'
import type { FormEvent } from 'react'

import styled from 'styled-components'

import type { ChatAutosaveStrategy, ChatProvider, ChatSession, VisibilityScope } from '../api/types'
import {
  GlassPane,
  MutedText,
  PaneHeader,
  ScrollColumn,
  SessionItemButton,
  SessionMeta,
  SplitGrid,
} from '../styles/primitives'
import { SmartIdDropdown } from './SmartIdDropdown'

const FieldBlock = styled.div`
  display: grid;
  gap: 0.3rem;
`

const CheckboxRow = styled.label`
  display: flex;
  align-items: center;
  gap: 0.5rem;
  text-transform: none;
  letter-spacing: normal;
  font-size: 0.82rem;

  input {
    width: auto;
    height: 1rem;
    width: 1rem;
  }
`

const SidebarBody = styled.div<{ $showCreator: boolean }>`
  flex: 1;
  min-height: 0;
  display: grid;
  gap: 0.7rem;
  grid-template-rows: ${({ $showCreator }) =>
    $showCreator ? 'minmax(18rem, 1fr) minmax(20rem, 1.6fr)' : 'auto minmax(0, 1fr)'};
`

const SectionLabel = styled.p`
  font-family: var(--font-display);
  font-size: 0.9rem;
  font-weight: 700;
  letter-spacing: 0.01em;
  color: var(--color-ink);
`

const CreatorPanel = styled.section`
  min-height: 0;
  display: grid;
  gap: 0.55rem;
`

const ToggleButton = styled.button`
  padding: 0.38rem 0.58rem;
  border: 1px solid var(--color-line);
  background: var(--surface-raised);
  color: var(--color-ink);
  font-size: 0.76rem;
  font-weight: 700;
  letter-spacing: 0.02em;
  text-transform: uppercase;
`

const CreatorForm = styled.form`
  min-height: 0;
  display: grid;
  grid-template-rows: minmax(0, 1fr) auto;
  border-top: 1px dashed var(--color-line);
`

const CreatorFields = styled.div`
  min-height: 0;
  overflow: auto;
  padding-right: 0.2rem;
  display: grid;
  align-content: start;
  gap: 0.55rem;
  padding-top: 0.55rem;
`

const CreatorStickyFooter = styled.div`
  position: sticky;
  bottom: 0;
  border-top: 1px dashed var(--color-line);
  padding-top: 0.55rem;
  background: var(--surface-glass);
`

const SessionListPanel = styled.section`
  min-height: 0;
  display: grid;
  grid-template-rows: auto minmax(0, 1fr);
  border-top: 1px dashed var(--color-line);
`

const SessionListHeader = styled.div`
  position: sticky;
  top: 0;
  z-index: 1;
  background: var(--surface-glass);
  padding: 0.5rem 0;
`

const SessionListSection = styled(ScrollColumn)`
  min-height: 0;
  padding-top: 0.2rem;
`

const SessionSearchRow = styled.div`
  display: grid;
  gap: 0.28rem;
  padding-bottom: 0.48rem;
  border-bottom: 1px dashed var(--color-line);
`

const DEFAULT_SESSION_TITLE = 'New chat session'
const DEFAULT_AUTOSAVE_INTERVAL_MINUTES = 30
const DEFAULT_AUTOSAVE_MIN_MESSAGES = 6
const DEFAULT_RETENTION_DAYS = 30
const DEFAULT_RETENTION_MAX_SNAPSHOTS = 60

interface CreateSessionRequest {
  project_id: string
  title: string
  provider: ChatProvider
  model_id: string
  system_prompt: string
  visibility_scope: VisibilityScope
  autosave_enabled: boolean
  autosave_strategy: ChatAutosaveStrategy
  autosave_interval_minutes: number
  autosave_min_messages: number
  retention_days: number
  retention_max_snapshots: number
}

interface SessionSidebarProps {
  sessions: ChatSession[]
  selectedSessionId: string | null
  projectId: string
  defaultProjectId: string | null
  settingDefaultProject?: boolean
  defaultProvider: ChatProvider
  defaultVisibilityScope: VisibilityScope
  modelDefaults: Record<ChatProvider, string>
  loading: boolean
  creating: boolean
  onProjectChange: (projectId: string) => void
  onSetDefaultProject?: () => Promise<void> | void
  onSelectSession: (sessionId: string) => void
  onCreateSession: (payload: CreateSessionRequest) => Promise<void>
}

interface CreatorState {
  title: string
  provider: ChatProvider
  modelId: string
  systemPrompt: string
  visibilityScope: VisibilityScope
  autosaveEnabled: boolean
  autosaveStrategy: ChatAutosaveStrategy
  autosaveIntervalMinutes: number
  autosaveMinMessages: number
  retentionDays: number
  retentionMaxSnapshots: number
}

interface AutosaveControlsProps {
  autosaveStrategy: ChatAutosaveStrategy
  autosaveIntervalMinutes: number
  autosaveMinMessages: number
  retentionDays: number
  retentionMaxSnapshots: number
  onAutosaveStrategyChange: (value: ChatAutosaveStrategy) => void
  onAutosaveIntervalMinutesChange: (value: number) => void
  onAutosaveMinMessagesChange: (value: number) => void
  onRetentionDaysChange: (value: number) => void
  onRetentionMaxSnapshotsChange: (value: number) => void
}

interface SessionCreatorPanelProps {
  showCreator: boolean
  projectId: string
  projectIdOptions: string[]
  defaultProjectId: string | null
  settingDefaultProject: boolean
  creating: boolean
  creatorState: CreatorState
  onProjectChange: (projectId: string) => void
  onSetDefaultProject?: () => Promise<void> | void
  onTitleChange: (value: string) => void
  onProviderChange: (value: ChatProvider) => void
  onVisibilityScopeChange: (value: VisibilityScope) => void
  onModelIdChange: (value: string) => void
  onSystemPromptChange: (value: string) => void
  onAutosaveEnabledChange: (value: boolean) => void
  onAutosaveStrategyChange: (value: ChatAutosaveStrategy) => void
  onAutosaveIntervalMinutesChange: (value: number) => void
  onAutosaveMinMessagesChange: (value: number) => void
  onRetentionDaysChange: (value: number) => void
  onRetentionMaxSnapshotsChange: (value: number) => void
  onSubmit: (event: FormEvent<HTMLFormElement>) => Promise<void>
}

interface SessionHistoryPanelProps {
  loading: boolean
  sessions: ChatSession[]
  selectedSessionId: string | null
  onSelectSession: (sessionId: string) => void
}

interface SessionHistoryResultsProps {
  loading: boolean
  sessionsCount: number
  filteredSessions: ChatSession[]
  visibleSessions: ChatSession[]
  selectedSessionId: string | null
  maxVisibleSessions: number
  onSelectSession: (sessionId: string) => void
}

interface ProjectSelectionFieldProps {
  projectId: string
  projectIdOptions: string[]
  defaultProjectId: string | null
  settingDefaultProject: boolean
  onProjectChange: (projectId: string) => void
  onSetDefaultProject?: () => Promise<void> | void
}

interface SessionCreatorFieldsProps {
  creatorState: CreatorState
  onTitleChange: (value: string) => void
  onProviderChange: (value: ChatProvider) => void
  onVisibilityScopeChange: (value: VisibilityScope) => void
  onModelIdChange: (value: string) => void
  onSystemPromptChange: (value: string) => void
  onAutosaveEnabledChange: (value: boolean) => void
  onAutosaveStrategyChange: (value: ChatAutosaveStrategy) => void
  onAutosaveIntervalMinutesChange: (value: number) => void
  onAutosaveMinMessagesChange: (value: number) => void
  onRetentionDaysChange: (value: number) => void
  onRetentionMaxSnapshotsChange: (value: number) => void
}

function parsePositiveNumber(value: string, fallback: number): number {
  const parsedValue = Number(value)
  return Number.isFinite(parsedValue) && parsedValue > 0 ? parsedValue : fallback
}

function sortedByNewest(sessions: ChatSession[]): ChatSession[] {
  return [...sessions].sort((left, right) => right.created_at.localeCompare(left.created_at))
}

function buildCreateSessionPayload(projectId: string, state: CreatorState): CreateSessionRequest {
  return {
    project_id: projectId,
    title: state.title.trim(),
    provider: state.provider,
    model_id: state.modelId.trim(),
    system_prompt: state.systemPrompt.trim(),
    visibility_scope: state.visibilityScope,
    autosave_enabled: state.autosaveEnabled,
    autosave_strategy: state.autosaveEnabled ? state.autosaveStrategy : 'off',
    autosave_interval_minutes: state.autosaveIntervalMinutes,
    autosave_min_messages: state.autosaveMinMessages,
    retention_days: state.retentionDays,
    retention_max_snapshots: state.retentionMaxSnapshots,
  }
}

function AutosaveControls({
  autosaveStrategy,
  autosaveIntervalMinutes,
  autosaveMinMessages,
  retentionDays,
  retentionMaxSnapshots,
  onAutosaveStrategyChange,
  onAutosaveIntervalMinutesChange,
  onAutosaveMinMessagesChange,
  onRetentionDaysChange,
  onRetentionMaxSnapshotsChange,
}: AutosaveControlsProps) {
  return (
    <>
      <FieldBlock>
        <label htmlFor="autosave-strategy">Autosave Strategy</label>
        <select
          id="autosave-strategy"
          value={autosaveStrategy}
          onChange={(event) => onAutosaveStrategyChange(event.target.value as ChatAutosaveStrategy)}
        >
          <option value="interval">Interval</option>
          <option value="message_count">Message Count</option>
        </select>
      </FieldBlock>

      {autosaveStrategy === 'interval' ? (
        <FieldBlock>
          <label htmlFor="autosave-interval-minutes">Autosave Interval (minutes)</label>
          <input
            id="autosave-interval-minutes"
            type="number"
            min={1}
            value={autosaveIntervalMinutes}
            onChange={(event) => onAutosaveIntervalMinutesChange(parsePositiveNumber(event.target.value, 1))}
          />
        </FieldBlock>
      ) : (
        <FieldBlock>
          <label htmlFor="autosave-min-messages">Autosave Every N Assistant Messages</label>
          <input
            id="autosave-min-messages"
            type="number"
            min={1}
            value={autosaveMinMessages}
            onChange={(event) => onAutosaveMinMessagesChange(parsePositiveNumber(event.target.value, 1))}
          />
        </FieldBlock>
      )}

      <SplitGrid>
        <FieldBlock>
          <label htmlFor="retention-days">Retention Days</label>
          <input
            id="retention-days"
            type="number"
            min={1}
            value={retentionDays}
            onChange={(event) => onRetentionDaysChange(parsePositiveNumber(event.target.value, 1))}
          />
        </FieldBlock>

        <FieldBlock>
          <label htmlFor="retention-max-snapshots">Max Snapshots</label>
          <input
            id="retention-max-snapshots"
            type="number"
            min={1}
            value={retentionMaxSnapshots}
            onChange={(event) => onRetentionMaxSnapshotsChange(parsePositiveNumber(event.target.value, 1))}
          />
        </FieldBlock>
      </SplitGrid>
    </>
  )
}

function ProjectSelectionField({
  projectId,
  projectIdOptions,
  defaultProjectId,
  settingDefaultProject,
  onProjectChange,
  onSetDefaultProject,
}: ProjectSelectionFieldProps) {
  return (
    <FieldBlock>
      <label htmlFor="project-id">Project ID</label>
      <SplitGrid>
        <SmartIdDropdown
          id="project-id"
          value={projectId}
          options={projectIdOptions}
          onChange={onProjectChange}
          placeholder="project-id"
          inputTestId="session-project-id-input"
          optionsTestId="session-project-id-options"
          matchCountTestId="session-project-id-matches"
        />
        <button
          type="button"
          disabled={settingDefaultProject || !projectId.trim() || !onSetDefaultProject}
          onClick={() => {
            void onSetDefaultProject?.()
          }}
        >
          {settingDefaultProject ? 'Saving…' : 'Set Default'}
        </button>
      </SplitGrid>
      <MutedText>
        Default Project: <strong>{defaultProjectId || 'not configured'}</strong>
      </MutedText>
    </FieldBlock>
  )
}

function SessionCreatorFields({
  creatorState,
  onTitleChange,
  onProviderChange,
  onVisibilityScopeChange,
  onModelIdChange,
  onSystemPromptChange,
  onAutosaveEnabledChange,
  onAutosaveStrategyChange,
  onAutosaveIntervalMinutesChange,
  onAutosaveMinMessagesChange,
  onRetentionDaysChange,
  onRetentionMaxSnapshotsChange,
}: SessionCreatorFieldsProps) {
  return (
    <>
      <label htmlFor="session-title">Title</label>
      <input
        id="session-title"
        value={creatorState.title}
        onChange={(event) => onTitleChange(event.target.value)}
        required
      />

      <SplitGrid>
        <FieldBlock>
          <label htmlFor="provider">Provider</label>
          <select
            id="provider"
            value={creatorState.provider}
            onChange={(event) => onProviderChange(event.target.value as ChatProvider)}
          >
            <option value="openai">OpenAI</option>
            <option value="anthropic">Anthropic</option>
            <option value="bedrock">Bedrock</option>
          </select>
        </FieldBlock>

        <FieldBlock>
          <label htmlFor="visibility">Visibility</label>
          <select
            id="visibility"
            value={creatorState.visibilityScope}
            onChange={(event) => onVisibilityScopeChange(event.target.value as VisibilityScope)}
          >
            <option value="private">Private</option>
            <option value="project">Project</option>
          </select>
        </FieldBlock>
      </SplitGrid>

      <label htmlFor="model-id">Model</label>
      <input
        id="model-id"
        value={creatorState.modelId}
        onChange={(event) => onModelIdChange(event.target.value)}
        required
      />

      <label htmlFor="system-prompt">System Prompt</label>
      <textarea
        id="system-prompt"
        value={creatorState.systemPrompt}
        onChange={(event) => onSystemPromptChange(event.target.value)}
        rows={3}
        placeholder="Optional guidance for the assistant"
      />

      <CheckboxRow htmlFor="autosave-enabled">
        <input
          id="autosave-enabled"
          type="checkbox"
          checked={creatorState.autosaveEnabled}
          onChange={(event) => onAutosaveEnabledChange(event.target.checked)}
        />
        <span>Enable autosave snapshots</span>
      </CheckboxRow>

      {creatorState.autosaveEnabled ? (
        <AutosaveControls
          autosaveStrategy={creatorState.autosaveStrategy}
          autosaveIntervalMinutes={creatorState.autosaveIntervalMinutes}
          autosaveMinMessages={creatorState.autosaveMinMessages}
          retentionDays={creatorState.retentionDays}
          retentionMaxSnapshots={creatorState.retentionMaxSnapshots}
          onAutosaveStrategyChange={onAutosaveStrategyChange}
          onAutosaveIntervalMinutesChange={onAutosaveIntervalMinutesChange}
          onAutosaveMinMessagesChange={onAutosaveMinMessagesChange}
          onRetentionDaysChange={onRetentionDaysChange}
          onRetentionMaxSnapshotsChange={onRetentionMaxSnapshotsChange}
        />
      ) : null}
    </>
  )
}

function SessionCreatorPanel({
  showCreator,
  projectId,
  projectIdOptions,
  defaultProjectId,
  settingDefaultProject,
  creating,
  creatorState,
  onProjectChange,
  onSetDefaultProject,
  onTitleChange,
  onProviderChange,
  onVisibilityScopeChange,
  onModelIdChange,
  onSystemPromptChange,
  onAutosaveEnabledChange,
  onAutosaveStrategyChange,
  onAutosaveIntervalMinutesChange,
  onAutosaveMinMessagesChange,
  onRetentionDaysChange,
  onRetentionMaxSnapshotsChange,
  onSubmit,
}: SessionCreatorPanelProps) {
  return (
    <CreatorPanel data-testid="session-create-panel">
      {showCreator ? (
        <CreatorForm onSubmit={(event) => void onSubmit(event)}>
          <CreatorFields>
            <ProjectSelectionField
              projectId={projectId}
              projectIdOptions={projectIdOptions}
              defaultProjectId={defaultProjectId}
              settingDefaultProject={settingDefaultProject}
              onProjectChange={onProjectChange}
              onSetDefaultProject={onSetDefaultProject}
            />
            <SessionCreatorFields
              creatorState={creatorState}
              onTitleChange={onTitleChange}
              onProviderChange={onProviderChange}
              onVisibilityScopeChange={onVisibilityScopeChange}
              onModelIdChange={onModelIdChange}
              onSystemPromptChange={onSystemPromptChange}
              onAutosaveEnabledChange={onAutosaveEnabledChange}
              onAutosaveStrategyChange={onAutosaveStrategyChange}
              onAutosaveIntervalMinutesChange={onAutosaveIntervalMinutesChange}
              onAutosaveMinMessagesChange={onAutosaveMinMessagesChange}
              onRetentionDaysChange={onRetentionDaysChange}
              onRetentionMaxSnapshotsChange={onRetentionMaxSnapshotsChange}
            />
          </CreatorFields>

          <CreatorStickyFooter>
            <button type="submit" disabled={creating || !projectId.trim() || !creatorState.title.trim()}>
              {creating ? 'Creating...' : 'Create Session'}
            </button>
          </CreatorStickyFooter>
        </CreatorForm>
      ) : null}
    </CreatorPanel>
  )
}

function SessionHistoryPanel({ loading, sessions, selectedSessionId, onSelectSession }: SessionHistoryPanelProps) {
  const [sessionSearch, setSessionSearch] = useState('')
  const normalizedSearch = sessionSearch.trim().toLowerCase()
  const filteredSessions = useMemo(() => {
    if (!normalizedSearch) {
      return sessions
    }
    return sessions.filter((session) => {
      const searchable = `${session.title} ${session.provider} ${session.model_id} ${session.session_id}`.toLowerCase()
      return searchable.includes(normalizedSearch)
    })
  }, [normalizedSearch, sessions])

  const MAX_VISIBLE_SESSIONS = 120
  const visibleSessions = filteredSessions.slice(0, MAX_VISIBLE_SESSIONS)

  return (
    <SessionListPanel data-testid="session-list-panel">
      <SessionListHeader>
        <SectionLabel>Previous Sessions</SectionLabel>
        <SessionSearchRow>
          <label htmlFor="session-search">Find Session</label>
          <input
            id="session-search"
            value={sessionSearch}
            onChange={(event) => setSessionSearch(event.target.value)}
            placeholder="Search sessions by title, model, or id"
          />
          <SessionMeta>
            Showing {visibleSessions.length} of {filteredSessions.length} sessions ({sessions.length} total)
          </SessionMeta>
        </SessionSearchRow>
      </SessionListHeader>

      <SessionListSection>
        <SessionHistoryResults
          loading={loading}
          sessionsCount={sessions.length}
          filteredSessions={filteredSessions}
          visibleSessions={visibleSessions}
          selectedSessionId={selectedSessionId}
          maxVisibleSessions={MAX_VISIBLE_SESSIONS}
          onSelectSession={onSelectSession}
        />
      </SessionListSection>
    </SessionListPanel>
  )
}

function SessionHistoryResults({
  loading,
  sessionsCount,
  filteredSessions,
  visibleSessions,
  selectedSessionId,
  maxVisibleSessions,
  onSelectSession,
}: SessionHistoryResultsProps) {
  if (loading) {
    return <MutedText>Loading sessions...</MutedText>
  }

  if (sessionsCount === 0) {
    return <MutedText>No sessions for this project.</MutedText>
  }

  if (filteredSessions.length === 0) {
    return <MutedText>No sessions match this search.</MutedText>
  }

  return (
    <>
      {visibleSessions.map((session) => (
        <SessionItemButton
          key={session.session_id}
          $active={selectedSessionId === session.session_id}
          aria-current={selectedSessionId === session.session_id ? 'true' : undefined}
          type="button"
          onClick={() => onSelectSession(session.session_id)}
        >
          <span className="font-semibold">{session.title}</span>
          <SessionMeta>
            {session.provider}/{session.model_id}
          </SessionMeta>
        </SessionItemButton>
      ))}
      {filteredSessions.length > maxVisibleSessions ? (
        <MutedText>
          More than {maxVisibleSessions} matches found. Add more search terms to narrow this list.
        </MutedText>
      ) : null}
    </>
  )
}

export function SessionSidebar({
  sessions,
  selectedSessionId,
  projectId,
  defaultProjectId,
  settingDefaultProject = false,
  defaultProvider,
  defaultVisibilityScope,
  modelDefaults,
  loading,
  creating,
  onProjectChange,
  onSetDefaultProject,
  onSelectSession,
  onCreateSession,
}: SessionSidebarProps) {
  const [title, setTitle] = useState(DEFAULT_SESSION_TITLE)
  const [provider, setProvider] = useState<ChatProvider>(defaultProvider)
  const [modelId, setModelId] = useState(modelDefaults[defaultProvider] ?? '')
  const [systemPrompt, setSystemPrompt] = useState('')
  const [visibilityScope, setVisibilityScope] = useState<VisibilityScope>(defaultVisibilityScope)
  const [autosaveEnabled, setAutosaveEnabled] = useState(false)
  const [autosaveStrategy, setAutosaveStrategy] = useState<ChatAutosaveStrategy>('interval')
  const [autosaveIntervalMinutes, setAutosaveIntervalMinutes] = useState(DEFAULT_AUTOSAVE_INTERVAL_MINUTES)
  const [autosaveMinMessages, setAutosaveMinMessages] = useState(DEFAULT_AUTOSAVE_MIN_MESSAGES)
  const [retentionDays, setRetentionDays] = useState(DEFAULT_RETENTION_DAYS)
  const [retentionMaxSnapshots, setRetentionMaxSnapshots] = useState(DEFAULT_RETENTION_MAX_SNAPSHOTS)
  const [showCreator, setShowCreator] = useState(true)

  const sortedSessions = useMemo(() => sortedByNewest(sessions), [sessions])
  const projectIdOptions = useMemo(() => {
    const unique = new Set<string>()
    const candidates = [projectId, defaultProjectId ?? '', ...sessions.map((item) => item.project_id)]
    for (const candidate of candidates) {
      const normalized = candidate.trim()
      if (!normalized) {
        continue
      }
      unique.add(normalized)
    }
    return Array.from(unique).sort((left, right) => left.localeCompare(right))
  }, [defaultProjectId, projectId, sessions])

  const creatorState: CreatorState = {
    title,
    provider,
    modelId,
    systemPrompt,
    visibilityScope,
    autosaveEnabled,
    autosaveStrategy,
    autosaveIntervalMinutes,
    autosaveMinMessages,
    retentionDays,
    retentionMaxSnapshots,
  }

  const handleProviderChange = (nextProvider: ChatProvider) => {
    setProvider(nextProvider)
    setModelId(modelDefaults[nextProvider] ?? '')
  }

  const handleSubmit = async (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault()
    await onCreateSession(buildCreateSessionPayload(projectId, creatorState))
    setTitle(DEFAULT_SESSION_TITLE)
  }

  return (
    <GlassPane as="aside" data-testid="session-sidebar">
      <PaneHeader>
        <h2 className="font-display text-base font-semibold tracking-[0.02em] text-ink">Create Session</h2>
        <ToggleButton type="button" onClick={() => setShowCreator((current) => !current)}>
          {showCreator ? 'Hide Creator' : 'Show Creator'}
        </ToggleButton>
      </PaneHeader>

      <SidebarBody $showCreator={showCreator}>
        <SessionCreatorPanel
          showCreator={showCreator}
          projectId={projectId}
          projectIdOptions={projectIdOptions}
          defaultProjectId={defaultProjectId}
          settingDefaultProject={settingDefaultProject}
          creating={creating}
          creatorState={creatorState}
          onProjectChange={onProjectChange}
          onSetDefaultProject={onSetDefaultProject}
          onTitleChange={setTitle}
          onProviderChange={handleProviderChange}
          onVisibilityScopeChange={setVisibilityScope}
          onModelIdChange={setModelId}
          onSystemPromptChange={setSystemPrompt}
          onAutosaveEnabledChange={setAutosaveEnabled}
          onAutosaveStrategyChange={setAutosaveStrategy}
          onAutosaveIntervalMinutesChange={setAutosaveIntervalMinutes}
          onAutosaveMinMessagesChange={setAutosaveMinMessages}
          onRetentionDaysChange={setRetentionDays}
          onRetentionMaxSnapshotsChange={setRetentionMaxSnapshots}
          onSubmit={handleSubmit}
        />

        <SessionHistoryPanel
          loading={loading}
          sessions={sortedSessions}
          selectedSessionId={selectedSessionId}
          onSelectSession={onSelectSession}
        />
      </SidebarBody>
    </GlassPane>
  )
}
