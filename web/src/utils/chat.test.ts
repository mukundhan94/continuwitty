import { describe, expect, it } from 'vitest'

import type { ChatMessage } from '../api/types'
import { buildDefaultSaveAbstract } from './chat'

function buildMessage(overrides: Partial<ChatMessage>): ChatMessage {
  return {
    message_id: 'msg-1',
    session_id: 'session-1',
    role: 'assistant',
    content_text: '',
    provider: null,
    model_id: null,
    token_usage_json: {},
    used_engram_ids: [],
    created_at: '2026-02-15T00:00:00Z',
    ...overrides,
  }
}

describe('buildDefaultSaveAbstract', () => {
  it('returns full latest assistant message without truncation', () => {
    const longText = `Summary ${'x'.repeat(1200)}`
    const messages = [buildMessage({ content_text: longText })]
    expect(buildDefaultSaveAbstract(messages)).toBe(longText)
  })

  it('falls back to latest user message when assistant content is empty', () => {
    const messages = [
      buildMessage({ role: 'assistant', content_text: '   ' }),
      buildMessage({ role: 'user', content_text: 'User incident handoff note' }),
    ]
    expect(buildDefaultSaveAbstract(messages)).toBe('User incident handoff note')
  })
})
