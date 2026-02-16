import { createEvent, fireEvent, render, screen } from '@testing-library/react'
import type { ComponentProps } from 'react'
import { ThemeProvider } from 'styled-components'
import { describe, expect, it, vi } from 'vitest'

import type { ChatMessage, ChatSession, ChatSourceReference } from '../api/types'
import { lightTheme } from '../styles/theme'
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
    debugTrace: null,
    onComposerChange: vi.fn(),
    onSend,
    onRetry: vi.fn(async () => {}),
    onOpenSaveModal: vi.fn(),
    onContinueSession: vi.fn(async () => {}),
    ...overrides,
  }

  render(
    <ThemeProvider theme={lightTheme}>
      <ChatPanel {...props} />
    </ThemeProvider>,
  )
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

describe('ChatPanel markdown rendering', () => {
  it('renders markdown formatting and links in assistant messages', () => {
    renderPanel({
      messages: [
        {
          message_id: 'msg-1',
          session_id: 'session-1',
          role: 'assistant',
          content_text: '**Incident Summary**\n\n- DB latency spike\n- Cache rollback\n\n[Runbook](https://example.com/runbook)',
          provider: 'bedrock',
          model_id: 'eu.anthropic.claude-haiku-4-5-20251001-v1:0',
          token_usage_json: {},
          used_engram_ids: [],
          created_at: '2026-02-15T00:00:00Z',
        },
      ],
    })

    expect(screen.getByText('Incident Summary', { selector: 'strong' })).toBeInTheDocument()
    expect(screen.getByText('DB latency spike')).toBeInTheDocument()

    const runbookLink = screen.getByRole('link', { name: 'Runbook' })
    expect(runbookLink).toHaveAttribute('href', 'https://example.com/runbook')
    expect(runbookLink).toHaveAttribute('target', '_blank')
  })

  it('renders debug trace details when provided', () => {
    renderPanel({
      debugTrace: {
        total_duration_ms: 120.5,
        prepare_duration_ms: 30.1,
        context_duration_ms: 20.2,
        history_load_duration_ms: 4.2,
        llm_call_duration_ms: 70.2,
        persistence_duration_ms: 2.4,
        used_engram_count: 1,
        used_document_chunk_count: 2,
        source_reference_count: 2,
        provider: 'openai',
        model_id: 'gpt-4o-mini',
        request_input_text: 'hello',
        response_output_text: 'world',
        provider_system_prompt_preview: 'be concise',
        provider_messages: [{ role: 'user', content_preview: 'hello', char_count: 5 }],
        embedding_calls: [
          {
            operation: 'embed',
            provider_id: 'local',
            duration_ms: 2.3,
            item_count: 1,
            text_chars: 5,
            dim: 256,
            used_fallback: false,
          },
        ],
        llm_calls: [
          {
            provider: 'openai',
            model_id: 'gpt-4o-mini',
            call_type: 'generate',
            duration_ms: 70.2,
            input_chars: 10,
            output_chars: 5,
            token_usage: { input_tokens: 3, output_tokens: 2, total_tokens: 5 },
            token_usage_is_estimated: false,
          },
        ],
      },
    })

    expect(screen.getByText('Debug Trace')).toBeInTheDocument()
    expect(screen.getByText(/Input Tokens:/)).toBeInTheDocument()
    expect(screen.getByText(/Output Tokens:/)).toBeInTheDocument()
  })
})
