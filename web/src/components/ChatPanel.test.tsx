import { createEvent, fireEvent, render, screen } from '@testing-library/react'
import type { ComponentProps } from 'react'
import { ThemeProvider } from 'styled-components'
import { describe, expect, it, vi } from 'vitest'

import type { ChatMessage, ChatSession, ChatSourceReference, EngramTracePath } from '../api/types'
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
    autosave_strategy: 'off',
    autosave_interval_minutes: 30,
    autosave_min_messages: 6,
    retention_days: 30,
    retention_max_snapshots: 60,
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
    timelineEvents: [],
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

  it('disables composer actions when no session is selected', () => {
    renderPanel({ session: null })

    expect(screen.getByPlaceholderText(/create a session first/i)).toBeDisabled()
    expect(screen.getByRole('button', { name: /^send$/i })).toBeDisabled()
    expect(screen.getByRole('button', { name: /retry last prompt/i })).toBeDisabled()
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

  it('renders linked trace metadata and recall controls', () => {
    const tracePaths: EngramTracePath[] = [
      {
        root_engram_id: 'engram-root-0001',
        target_engram_id: 'engram-target-0002',
        depth: 1,
        link_ids: ['link-1'],
        engram_ids: ['engram-root-0001', 'engram-target-0002'],
        score: 0.82,
      },
    ]
    const onDepthChange = vi.fn()

    renderPanel({
      usedEngramLinkIds: ['link-1'],
      engramTracePaths: tracePaths,
      onLinkRecallDepthChange: onDepthChange,
    })

    expect(screen.getByText('Linked trace paths used:')).toBeInTheDocument()
    expect(screen.getByTestId('trace-path-strip')).toBeInTheDocument()
    expect(screen.getByLabelText(/link recall/i)).toBeInTheDocument()

    fireEvent.change(screen.getByTestId('link-recall-depth-input'), { target: { value: '2' } })
    expect(onDepthChange).toHaveBeenCalledWith(2)
  })
})

describe('ChatPanel timeline rendering', () => {
  it('groups consolidation events with the same consolidation group key', () => {
    renderPanel({
      timelineEvents: [
        {
          event_id: 'timeline-1',
          session_id: 'session-1',
          event_type: 'consolidation_merge',
          title: 'Consolidated Memory Snapshot A',
          abstract: 'Merged older timeline snapshots.',
          tags: ['consolidated', 'consolidation_group_key:incident-42', 'consolidation_merged_count:3'],
          consolidation_group_key: 'incident-42',
          consolidation_merged_count: 3,
          created_at: '2026-02-22T10:00:00Z',
        },
        {
          event_id: 'timeline-2',
          session_id: 'session-1',
          event_type: 'consolidation_merge',
          title: 'Consolidated Memory Snapshot B',
          abstract: 'Merged additional timeline snapshots.',
          tags: ['consolidated', 'consolidation_group_key:incident-42', 'consolidation_merged_count:2'],
          consolidation_group_key: 'incident-42',
          consolidation_merged_count: 2,
          created_at: '2026-02-22T09:58:00Z',
        },
        {
          event_id: 'timeline-3',
          session_id: 'session-1',
          event_type: 'autosave_snapshot',
          title: 'Autosave Snapshot #1',
          abstract: 'Latest autosave summary.',
          tags: ['autosave_snapshot'],
          created_at: '2026-02-22T09:50:00Z',
        },
      ],
    })

    expect(screen.getAllByTestId('timeline-item')).toHaveLength(2)
    expect(screen.getByText('Consolidation Group')).toBeInTheDocument()
    expect(
      screen.getByText('Group incident-42: 2 consolidation snapshots (5 merged engrams).'),
    ).toBeInTheDocument()
    expect(screen.getByText('Autosave Snapshot #1')).toBeInTheDocument()
  })
})
