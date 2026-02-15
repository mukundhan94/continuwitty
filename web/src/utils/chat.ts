import type { ChatMessage } from '../api/types'

function normalizeSpaces(value: string): string {
  return value.trim().replace(/\s+/g, ' ')
}

export function buildDefaultSaveAbstract(messages: ChatMessage[]): string {
  for (const role of ['assistant', 'user'] as const) {
    for (let index = messages.length - 1; index >= 0; index -= 1) {
      const message = messages[index]
      if (message.role !== role) {
        continue
      }
      const normalized = normalizeSpaces(message.content_text || '')
      if (!normalized) {
        continue
      }
      return normalized
    }
  }
  return 'Captured session insights from chat transcript.'
}
