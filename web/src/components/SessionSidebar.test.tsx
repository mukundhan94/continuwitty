import { fireEvent, render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { ThemeProvider } from 'styled-components'
import { describe, expect, it, vi } from 'vitest'

import type { ChatSession, ChatSessionFormPayload } from '../api/types'
import { lightTheme } from '../styles/theme'
import { SessionSidebar } from './SessionSidebar'

function buildSession(overrides: Partial<ChatSession>): ChatSession {
  return {
    session_id: 'session-1',
    owner_user_id: 'user-1',
    project_id: 'engram-vault',
    title: 'Session One',
    provider: 'openai',
    model_id: 'gpt-4o-mini',
    system_prompt: '',
    visibility_scope: 'private',
    autosave_enabled: false,
    autosave_strategy: 'off',
    autosave_interval_minutes: 30,
    autosave_min_messages: 6,
    retention_days: 30,
    retention_max_snapshots: 60,
    created_at: '2026-02-15T00:00:00Z',
    updated_at: '2026-02-15T00:00:00Z',
    ...overrides,
  }
}

function renderSidebar(
  overrides: Partial<{
    onCreateSession: (payload: ChatSessionFormPayload) => Promise<void>
    onSelectSession: (sessionId: string) => void
  }> = {},
) {
  const onCreateSession = overrides.onCreateSession ?? vi.fn(async () => {})
  const onSelectSession = overrides.onSelectSession ?? vi.fn()
  render(
    <ThemeProvider theme={lightTheme}>
      <SessionSidebar
        sessions={[
          buildSession({ session_id: 'session-active', title: 'Current Session' }),
          buildSession({ session_id: 'session-older', title: 'Older Session' }),
        ]}
        selectedSessionId="session-active"
        projectId="engram-vault"
        defaultProjectId="engram-vault"
        defaultProvider="openai"
        defaultVisibilityScope="private"
        modelDefaults={{
          openai: 'gpt-4o-mini',
          anthropic: 'claude-3-5-haiku-20241022',
          bedrock: 'eu.anthropic.claude-haiku-4-5-20251001-v1:0',
        }}
        loading={false}
        creating={false}
        onProjectChange={vi.fn()}
        onSetDefaultProject={vi.fn(async () => {})}
        onSelectSession={onSelectSession}
        onCreateSession={onCreateSession}
      />
    </ThemeProvider>,
  )
  return { onCreateSession, onSelectSession }
}

describe('SessionSidebar', () => {
  it('shows previous sessions section label', () => {
    renderSidebar()
    expect(screen.getByText('Previous Sessions')).toBeInTheDocument()
    const options = screen
      .getByTestId('session-project-id-options')
      .querySelectorAll('option')
    const values = [...options].map((item) => item.getAttribute('value'))
    expect(values).toContain('engram-vault')
  })

  it('supports hiding and showing creator section', async () => {
    const user = userEvent.setup()
    renderSidebar()

    expect(screen.getByLabelText(/project id/i)).toBeInTheDocument()
    await user.click(screen.getByRole('button', { name: /hide creator/i }))
    expect(screen.queryByLabelText(/project id/i)).not.toBeInTheDocument()
    await user.click(screen.getByRole('button', { name: /show creator/i }))
    expect(screen.getByLabelText(/project id/i)).toBeInTheDocument()
  })

  it('marks selected session for active highlighting', () => {
    renderSidebar()
    const activeSessionButton = screen.getByRole('button', { name: /current session/i })
    expect(activeSessionButton).toHaveAttribute('aria-current', 'true')
  })

  it('submits normalized create-session payload', async () => {
    const user = userEvent.setup()
    const onCreateSession = vi.fn(async () => {})
    renderSidebar({ onCreateSession })

    const titleInput = screen.getByLabelText(/title/i)
    const modelInput = screen.getByLabelText(/model/i)
    const systemPromptInput = screen.getByLabelText(/system prompt/i)
    await user.clear(titleInput)
    await user.type(titleInput, '  Incident Analysis  ')
    await user.clear(modelInput)
    await user.type(modelInput, '  gpt-4o-mini  ')
    await user.type(systemPromptInput, '  keep concise  ')
    await user.click(screen.getByRole('button', { name: /create session/i }))

    await waitFor(() => {
      expect(onCreateSession).toHaveBeenCalledWith(
        expect.objectContaining({
          project_id: 'engram-vault',
          title: 'Incident Analysis',
          provider: 'openai',
          model_id: 'gpt-4o-mini',
          system_prompt: 'keep concise',
          visibility_scope: 'private',
          autosave_enabled: false,
          autosave_strategy: 'off',
          autosave_interval_minutes: 30,
          autosave_min_messages: 6,
          retention_days: 30,
          retention_max_snapshots: 60,
        }),
      )
    })
  })

  it('submits message_count autosave payload when autosave is enabled', async () => {
    const user = userEvent.setup()
    const onCreateSession = vi.fn(async () => {})
    renderSidebar({ onCreateSession })

    await user.click(screen.getByLabelText(/enable autosave snapshots/i))
    await user.selectOptions(screen.getByLabelText(/autosave strategy/i), 'message_count')

    const minMessagesInput = screen.getByLabelText(/autosave every n assistant messages/i)
    fireEvent.change(minMessagesInput, { target: { value: '3' } })

    await user.click(screen.getByRole('button', { name: /create session/i }))

    await waitFor(() => {
      expect(onCreateSession).toHaveBeenCalledWith(
        expect.objectContaining({
          autosave_enabled: true,
          autosave_strategy: 'message_count',
          autosave_min_messages: 3,
        }),
      )
    })
  })

  it('filters session list by search input', async () => {
    const user = userEvent.setup()
    renderSidebar()

    await user.type(screen.getByLabelText(/find session/i), 'older')

    expect(screen.getByRole('button', { name: /older session/i })).toBeInTheDocument()
    expect(screen.queryByRole('button', { name: /current session/i })).not.toBeInTheDocument()
  })
})
