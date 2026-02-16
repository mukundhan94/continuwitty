export type UserRole = 'admin' | 'analyst' | 'viewer'
export type ChatProvider = 'openai' | 'anthropic' | 'bedrock'
export type VisibilityScope = 'private' | 'project'

export interface UserProfile {
  user_id: string
  username: string
  role: UserRole
}

export interface EngramSummary {
  engram_id: string
  project_id: string
  thread_id: string | null
  title: string
  abstract: string
  created_at: string
  tags: string[]
  keywords: string[]
  owner_user_id: string | null
  visibility_scope: VisibilityScope
}

export interface DocumentRecord {
  document_id: string
  owner_user_id: string
  project_id: string
  title: string
  source_type: 'text' | 'file'
  source_name: string | null
  mime_type: string | null
  visibility_scope: VisibilityScope
  content_hash: string
  chunk_count: number
  created_at: string
  updated_at: string
}

export interface ChatSession {
  session_id: string
  owner_user_id: string
  project_id: string
  title: string
  provider: ChatProvider
  model_id: string
  system_prompt: string
  visibility_scope: VisibilityScope
  autosave_enabled: boolean
  created_at: string
  updated_at: string
}

export interface ChatMessage {
  message_id: string
  session_id: string
  role: string
  content_text: string
  provider: string | null
  model_id: string | null
  token_usage_json: Record<string, number>
  used_engram_ids: string[]
  created_at: string
}

export interface ChatSourceReference {
  engram_id: string
  engram_title: string
  url: string
  title: string | null
  snippet: string
  captured_at: string
  source_type?: string
  document_id?: string | null
  chunk_id?: string | null
  chunk_index?: number | null
}

export interface ChatDebugEmbeddingCall {
  operation: string
  provider_id: string
  duration_ms: number
  item_count: number
  text_chars: number
  dim: number
  used_fallback: boolean
}

export interface ChatDebugLlmCall {
  provider: string
  model_id: string
  call_type: string
  duration_ms: number
  input_chars: number
  output_chars: number
  token_usage: Record<string, number>
  token_usage_is_estimated: boolean
}

export interface ChatDebugProviderMessage {
  role: string
  content_preview: string
  char_count: number
}

export interface ChatDebugTrace {
  total_duration_ms: number
  prepare_duration_ms: number
  context_duration_ms: number
  history_load_duration_ms: number
  llm_call_duration_ms: number
  persistence_duration_ms: number
  used_engram_count: number
  used_document_chunk_count: number
  source_reference_count: number
  provider: string
  model_id: string
  request_input_text: string
  response_output_text: string
  provider_system_prompt_preview: string
  provider_messages: ChatDebugProviderMessage[]
  embedding_calls: ChatDebugEmbeddingCall[]
  llm_calls: ChatDebugLlmCall[]
}

export interface ChatSendResponse {
  session_id: string
  message_id: string
  reply_message_id: string
  assistant_text: string
  used_engram_ids: string[]
  used_document_chunk_ids: string[]
  source_references: ChatSourceReference[]
  debug_trace?: ChatDebugTrace | null
}

export interface ContinueSessionResponse {
  session: ChatSession
  carried_engram_ids: string[]
}

export interface PinnedDocumentRecord {
  session_id: string
  document_id: string
  pinned_by_user_id: string
  created_at: string
}

export interface SaveSessionAsEngramResponse {
  engram_id: string
  session_id: string
  created_at: string
}
