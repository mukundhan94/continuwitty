import { createEvent, fireEvent, render, screen } from '@testing-library/react'
import type { ComponentProps } from 'react'
import { describe, expect, it, vi } from 'vitest'

import type { ChatMessage, ChatSession, ChatSourceReference } from '../api/types'
import { ChatPanel } from './ChatPanel'

function buildSession(): ChatSession {
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
  }
}

function renderPanel(overrides: Partial<ComponentProps<typeof ChatPanel>> = {}) {
  const onSend = vi.fn(async () => {})
  const props: ComponentProps<typeof ChatPanel> = {
    session: buildSession(),
    messages: [] as ChatMessage[],
    pendingUserText: null,
    streamingAssistantText: '',
    composerText: 'Hello world',
    sending: false,
    error: null,
    sourceReferences: [] as ChatSourceReference[],
    onComposerChange: vi.fn(),
    onSend,
    onRetry: vi.fn(async () => {}),
    onOpenSaveModal: vi.fn(),
    onContinueSession: vi.fn(async () => {}),
    ...overrides,
  }

  render(<ChatPanel {...props} />)
  return { props, onSend }
}

describe('ChatPanel keyboard behavior', () => {
  it('submits on Enter', () => {
    const { onSend } = renderPanel()
    const composer = screen.getByPlaceholderText(/ask something and stream a response/i)

    const event = createEvent.keyDown(composer, { key: 'Enter', code: 'Enter' })
    fireEvent(composer, event)

    expect(event.defaultPrevented).toBe(true)
    expect(onSend).toHaveBeenCalledTimes(1)
  })

  it('keeps newline behavior on Shift+Enter', () => {
    const { onSend } = renderPanel()
    const composer = screen.getByPlaceholderText(/ask something and stream a response/i)

    const event = createEvent.keyDown(composer, { key: 'Enter', code: 'Enter', shiftKey: true })
    fireEvent(composer, event)

    expect(event.defaultPrevented).toBe(false)
    expect(onSend).not.toHaveBeenCalled()
  })
})
