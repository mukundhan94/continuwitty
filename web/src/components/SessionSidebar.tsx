import { useMemo, useState } from 'react'
import type { FormEvent } from 'react'

import styled from 'styled-components'

import type { ChatProvider, ChatSession, VisibilityScope } from '../api/types'
import {
  FormGrid,
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

interface CreateSessionRequest {
  project_id: string
  title: string
  provider: ChatProvider
  model_id: string
  system_prompt: string
  visibility_scope: VisibilityScope
  autosave_enabled: boolean
}

interface SessionSidebarProps {
  sessions: ChatSession[]
  selectedSessionId: string | null
  projectId: string
  defaultProvider: ChatProvider
  defaultVisibilityScope: VisibilityScope
  modelDefaults: Record<ChatProvider, string>
  loading: boolean
  creating: boolean
  onProjectChange: (projectId: string) => void
  onSelectSession: (sessionId: string) => void
  onCreateSession: (payload: CreateSessionRequest) => Promise<void>
}

export function SessionSidebar({
  sessions,
  selectedSessionId,
  projectId,
  defaultProvider,
  defaultVisibilityScope,
  modelDefaults,
  loading,
  creating,
  onProjectChange,
  onSelectSession,
  onCreateSession,
}: SessionSidebarProps) {
  const [title, setTitle] = useState('New chat session')
  const [provider, setProvider] = useState<ChatProvider>(defaultProvider)
  const [modelId, setModelId] = useState(modelDefaults[defaultProvider])
  const [systemPrompt, setSystemPrompt] = useState('')
  const [visibilityScope, setVisibilityScope] = useState<VisibilityScope>(defaultVisibilityScope)
  const [autosaveEnabled, setAutosaveEnabled] = useState(false)

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
    })
    setTitle('New chat session')
  }

  return (
    <GlassPane as="aside">
      <PaneHeader>
        <h2 className="font-display text-base font-semibold tracking-[0.02em] text-ink">Sessions</h2>
      </PaneHeader>

      <FieldBlock>
        <label htmlFor="project-id">Project ID</label>
        <input
          id="project-id"
          value={projectId}
          onChange={(event) => onProjectChange(event.target.value)}
          placeholder="project-id"
        />
      </FieldBlock>

      <FormGrid onSubmit={handleSubmit}>
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

        <button type="submit" disabled={creating || !projectId.trim() || !title.trim()}>
          {creating ? 'Creating...' : 'Create Session'}
        </button>
      </FormGrid>

      <ScrollColumn className="flex-1">
        {loading ? <MutedText>Loading sessions...</MutedText> : null}
        {!loading && sortedSessions.length === 0 ? <MutedText>No sessions for this project.</MutedText> : null}

        {sortedSessions.map((session) => (
          <SessionItemButton
            key={session.session_id}
            $active={selectedSessionId === session.session_id}
            type="button"
            onClick={() => onSelectSession(session.session_id)}
          >
            <span className="font-semibold">{session.title}</span>
            <SessionMeta>
              {session.provider}/{session.model_id}
            </SessionMeta>
          </SessionItemButton>
        ))}
      </ScrollColumn>
    </GlassPane>
  )
}
