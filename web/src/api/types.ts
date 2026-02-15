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
}

export interface ChatSendResponse {
  session_id: string
  message_id: string
  reply_message_id: string
  assistant_text: string
  used_engram_ids: string[]
  used_document_chunk_ids: string[]
  source_references: ChatSourceReference[]
}

export interface ContinueSessionResponse {
  session: ChatSession
  carried_engram_ids: string[]
}

export interface SaveSessionAsEngramResponse {
  engram_id: string
  session_id: string
  created_at: string
}
