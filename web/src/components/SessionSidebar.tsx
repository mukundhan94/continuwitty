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
  const [title, setTitle] = useState('New chat session')
  const [provider, setProvider] = useState<ChatProvider>(defaultProvider)
  const [modelId, setModelId] = useState(modelDefaults[defaultProvider])
  const [systemPrompt, setSystemPrompt] = useState('')
  const [visibilityScope, setVisibilityScope] = useState<VisibilityScope>(defaultVisibilityScope)
  const [autosaveEnabled, setAutosaveEnabled] = useState(false)
  const [autosaveStrategy, setAutosaveStrategy] = useState<ChatAutosaveStrategy>('interval')
  const [autosaveIntervalMinutes, setAutosaveIntervalMinutes] = useState(30)
  const [autosaveMinMessages, setAutosaveMinMessages] = useState(6)
  const [retentionDays, setRetentionDays] = useState(30)
  const [retentionMaxSnapshots, setRetentionMaxSnapshots] = useState(60)
  const [showCreator, setShowCreator] = useState(true)

  const sortedSessions = useMemo(
    () => [...sessions].sort((a, b) => b.created_at.localeCompare(a.created_at)),
    [sessions],
  )

  const handleProviderChange = (nextProvider: ChatProvider) => {
    setProvider(nextProvider)
    setModelId(modelDefaults[nextProvider])
  }

  const handleSubmit = async (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault()
    await onCreateSession({
      project_id: projectId,
      title: title.trim(),
      provider,
      model_id: modelId.trim(),
      system_prompt: systemPrompt.trim(),
      visibility_scope: visibilityScope,
      autosave_enabled: autosaveEnabled,
      autosave_strategy: autosaveEnabled ? autosaveStrategy : 'off',
      autosave_interval_minutes: autosaveIntervalMinutes,
      autosave_min_messages: autosaveMinMessages,
      retention_days: retentionDays,
      retention_max_snapshots: retentionMaxSnapshots,
    })
    setTitle('New chat session')
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
        <CreatorPanel data-testid="session-create-panel">
          {showCreator ? (
            <CreatorForm onSubmit={handleSubmit}>
              <CreatorFields>
                <FieldBlock>
                  <label htmlFor="project-id">Project ID</label>
                  <SplitGrid>
                    <input
                      id="project-id"
                      value={projectId}
                      onChange={(event) => onProjectChange(event.target.value)}
                      placeholder="project-id"
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

                <label htmlFor="session-title">Title</label>
                <input
                  id="session-title"
                  value={title}
                  onChange={(event) => setTitle(event.target.value)}
                  required
                />

                <SplitGrid>
                  <FieldBlock>
                    <label htmlFor="provider">Provider</label>
                    <select
                      id="provider"
                      value={provider}
                      onChange={(event) => handleProviderChange(event.target.value as ChatProvider)}
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
                      value={visibilityScope}
                      onChange={(event) => setVisibilityScope(event.target.value as VisibilityScope)}
                    >
                      <option value="private">Private</option>
                      <option value="project">Project</option>
                    </select>
                  </FieldBlock>
                </SplitGrid>

                <label htmlFor="model-id">Model</label>
                <input id="model-id" value={modelId} onChange={(event) => setModelId(event.target.value)} required />

                <label htmlFor="system-prompt">System Prompt</label>
                <textarea
                  id="system-prompt"
                  value={systemPrompt}
                  onChange={(event) => setSystemPrompt(event.target.value)}
                  rows={3}
                  placeholder="Optional guidance for the assistant"
                />

                <CheckboxRow htmlFor="autosave-enabled">
                  <input
                    id="autosave-enabled"
                    type="checkbox"
                    checked={autosaveEnabled}
                    onChange={(event) => setAutosaveEnabled(event.target.checked)}
                  />
                  <span>Enable autosave snapshots</span>
                </CheckboxRow>

                {autosaveEnabled ? (
                  <>
                    <FieldBlock>
                      <label htmlFor="autosave-strategy">Autosave Strategy</label>
                      <select
                        id="autosave-strategy"
                        value={autosaveStrategy}
                        onChange={(event) => setAutosaveStrategy(event.target.value as ChatAutosaveStrategy)}
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
                          onChange={(event) => setAutosaveIntervalMinutes(Number(event.target.value) || 1)}
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
                          onChange={(event) => setAutosaveMinMessages(Number(event.target.value) || 1)}
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
                          onChange={(event) => setRetentionDays(Number(event.target.value) || 1)}
                        />
                      </FieldBlock>

                      <FieldBlock>
                        <label htmlFor="retention-max-snapshots">Max Snapshots</label>
                        <input
                          id="retention-max-snapshots"
                          type="number"
                          min={1}
                          value={retentionMaxSnapshots}
                          onChange={(event) => setRetentionMaxSnapshots(Number(event.target.value) || 1)}
                        />
                      </FieldBlock>
                    </SplitGrid>
                  </>
                ) : null}
              </CreatorFields>

              <CreatorStickyFooter>
                <button type="submit" disabled={creating || !projectId.trim() || !title.trim()}>
                  {creating ? 'Creating...' : 'Create Session'}
                </button>
              </CreatorStickyFooter>
            </CreatorForm>
          ) : null}
        </CreatorPanel>

        <SessionListPanel data-testid="session-list-panel">
          <SessionListHeader>
            <SectionLabel>Previous Sessions</SectionLabel>
          </SessionListHeader>

          <SessionListSection>
            {loading ? <MutedText>Loading sessions...</MutedText> : null}
            {!loading && sortedSessions.length === 0 ? <MutedText>No sessions for this project.</MutedText> : null}

            {sortedSessions.map((session) => (
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
          </SessionListSection>
        </SessionListPanel>
      </SidebarBody>
    </GlassPane>
  )
}
