import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { ThemeProvider } from 'styled-components'
import { describe, expect, it, vi } from 'vitest'

import type { ChatSession } from '../api/types'
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
    created_at: '2026-02-15T00:00:00Z',
    updated_at: '2026-02-15T00:00:00Z',
    ...overrides,
  }
}

function renderSidebar() {
  render(
    <ThemeProvider theme={lightTheme}>
      <SessionSidebar
        sessions={[
          buildSession({ session_id: 'session-active', title: 'Current Session' }),
          buildSession({ session_id: 'session-older', title: 'Older Session' }),
        ]}
        selectedSessionId="session-active"
        projectId="engram-vault"
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
        onSelectSession={vi.fn()}
        onCreateSession={vi.fn(async () => {})}
      />
    </ThemeProvider>,
  )
}

describe('SessionSidebar', () => {
  it('shows previous sessions section label', () => {
    renderSidebar()
    expect(screen.getByText('Previous Sessions')).toBeInTheDocument()
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
})
