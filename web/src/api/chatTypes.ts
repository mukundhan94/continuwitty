import type {
  ChatDebugTrace,
  ChatSessionFormPayload,
  ChatSourceReference,
  VisibilityScope,
} from './types'

export type CreateSessionPayload = ChatSessionFormPayload

export interface SaveEngramPayload {
  title: string
  abstract: string
  visibility_scope: VisibilityScope
  tags: string[]
  keywords: string[]
}

export interface StreamMetaPayload {
  session_id: string
  message_id: string
  used_engram_ids: string[]
  used_document_chunk_ids: string[]
  source_references: ChatSourceReference[]
  debug_trace?: ChatDebugTrace | null
}

export interface StreamChunkPayload {
  text: string
}

export interface StreamDonePayload {
  session_id: string
  message_id: string
  reply_message_id: string
  assistant_text: string
  used_engram_ids: string[]
  used_document_chunk_ids: string[]
  source_references: ChatSourceReference[]
  debug_trace?: ChatDebugTrace | null
}

export interface StreamErrorPayload {
  detail: string
  status_code: number
  error_code: string
}

export type ChatStreamEvent =
  | { event: 'meta'; data: StreamMetaPayload }
  | { event: 'chunk'; data: StreamChunkPayload }
  | { event: 'done'; data: StreamDonePayload }
  | { event: 'error'; data: StreamErrorPayload }
