export type UserRole = 'admin' | 'analyst' | 'viewer'
export type ChatProvider = 'openai' | 'anthropic' | 'bedrock'
export type VisibilityScope = 'private' | 'project'
export type ChatAutosaveStrategy = 'off' | 'interval' | 'message_count'

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

export interface EngramVisibilityRecord {
  engram_id: string
  project_id: string
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

export interface ChatSessionFormPayload {
  project_id: string
  title: string
  provider: ChatProvider
  model_id: string
  system_prompt: string
  visibility_scope: VisibilityScope
  autosave_enabled: boolean
  autosave_strategy: ChatAutosaveStrategy
  autosave_interval_minutes: number
  autosave_min_messages: number
  retention_days: number
  retention_max_snapshots: number
}

export interface ChatSession extends ChatSessionFormPayload {
  session_id: string
  owner_user_id: string
  created_at: string
  updated_at: string
}

export interface ChatLifecyclePolicy {
  autosave_enabled: boolean
  autosave_strategy: ChatAutosaveStrategy
  autosave_interval_minutes: number
  autosave_min_messages: number
  retention_days: number
  retention_max_snapshots: number
}

export interface ChatTimelineEvent {
  event_id: string
  session_id: string
  event_type:
    | 'autosave_snapshot'
    | 'manual_snapshot'
    | 'consolidation'
    | 'consolidation_merge'
    | 'consolidation_group'
    | string
  title: string
  abstract: string
  tags: string[]
  consolidation_group_key?: string | null
  consolidation_merged_count?: number | null
  created_at: string
}

export interface AgentSourceInput {
  url: string
  title?: string | null
  snippet: string
  captured_at?: string | null
}

export interface AgentDecision {
  decision: string
  rationale: string
}

export interface AgentState {
  project_id: string
  thread_id: string
  objective: string
  notes: string[]
  assumptions: string[]
  tags: string[]
  keywords: string[]
  sources: AgentSourceInput[]
  synthesis_title: string
  synthesis_abstract: string
  synthesis_markdown: string
  decisions: AgentDecision[]
  open_questions: string[]
  status: string
  auto_persist_engram: boolean
  engram_id: string | null
  snapshot_enabled: boolean
  snapshot_every_n_notes: number
  snapshot_count: number
  snapshot_engram_ids: string[]
}

export interface AgentRunResponse {
  thread_id: string
  status: string
  engram_id: string | null
  snapshot_engram_ids: string[]
  state: AgentState
}

export interface AgentRunCreateInput {
  project_id: string
  thread_id: string
  objective: string
  notes?: string[]
  assumptions?: string[]
  tags?: string[]
  keywords?: string[]
  sources?: AgentSourceInput[]
  auto_persist_engram?: boolean
  snapshot_enabled?: boolean
  snapshot_every_n_notes?: number
}

export interface AgentRunResumeInput {
  notes?: string[]
  assumptions?: string[]
  tags?: string[]
  keywords?: string[]
  sources?: AgentSourceInput[]
  auto_persist_engram?: boolean
  snapshot_enabled?: boolean
  snapshot_every_n_notes?: number
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

export interface EngramTracePath {
  root_engram_id: string
  target_engram_id: string
  depth: number
  link_ids: string[]
  engram_ids: string[]
  score: number
}

export type EngramLinkRelationType =
  | 'supports'
  | 'depends_on'
  | 'contradicts'
  | 'related_to'
  | 'derived_from'

export type EngramLinkOrigin = 'manual' | 'suggested' | 'inferred' | 'system'

export type EngramLinkStatus = 'active' | 'suggested' | 'archived' | 'rejected'

export interface EngramLinkRecord {
  link_id: string
  project_id: string
  source_engram_id: string
  target_engram_id: string
  relation_type: EngramLinkRelationType
  weight: number
  temporal_weight: number
  confidence: number
  origin: EngramLinkOrigin
  status: EngramLinkStatus
  evidence_json: Record<string, unknown>
  created_by_user_id: string
  last_reinforced_at?: string | null
  created_at: string
  updated_at: string
}

export interface EngramLinkSuggestion {
  source_engram_id: string
  target_engram_id: string
  project_id: string
  target_title: string
  target_abstract: string
  target_created_at: string
  relation_type: EngramLinkRelationType
  weight: number
  temporal_weight: number
  confidence: number
  score: number
  origin: EngramLinkOrigin
  status: EngramLinkStatus
  reasons: string[]
  evidence_json: Record<string, unknown>
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
  used_engram_link_ids?: string[]
  engram_trace_paths?: EngramTracePath[]
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

export type McpTokenScope = 'read' | 'write'

export interface McpTokenSummary {
  token_id: string
  name: string
  scope: McpTokenScope
  allowed_tools: string[]
  allowed_project_ids: string[]
  token_secret_hint: string
  expires_at: string
  last_used_at: string | null
  revoked_at: string | null
  created_at: string
  is_active: boolean
}

export interface McpTokenCreateRequest {
  name: string
  scope: McpTokenScope
  allowed_tools: string[]
  allowed_project_ids: string[]
  expires_in_days: number
}

export interface McpTokenCreateResponse extends McpTokenSummary {
  token: string
}

export interface ProjectRecord {
  project_id: string
  name: string
  description: string
  owner_user_id: string
  membership_role?: ProjectMemberRole
  is_archived: boolean
  created_at: string
  updated_at: string
}

export interface ProjectDefaultResponse {
  default_project_id: string | null
}

export type ProjectMemberRole = 'owner' | 'editor' | 'viewer'

export interface ProjectMemberRecord {
  project_id: string
  user_id: string
  role: ProjectMemberRole
  added_by_user_id: string | null
  created_at: string
  updated_at: string
  revoked_at: string | null
  revoked_by_user_id: string | null
}

export interface ProjectAuditEventRecord {
  event_id: string
  project_id: string
  actor_user_id: string | null
  event_type: string
  target_type: string
  target_user_id: string | null
  target_engram_id: string | null
  metadata: Record<string, unknown>
  created_at: string
}

export type ProjectExportFormat = 'json' | 'zip'
export type ProjectImportConflictPolicy = 'skip' | 'overwrite' | 'rename'

export interface ProjectImportResponse {
  target_project_id: string
  imported_engrams: number
  skipped_engrams: number
  overwritten_engrams: number
  imported_collections: number
  reused_collections: number
  imported_collection_items: number
  conflict_policy: ProjectImportConflictPolicy
}

export interface AdminChatSessionRecord {
  session_id: string
  owner_user_id: string
  project_id: string
  title: string
  provider: ChatProvider
  model_id: string
  system_prompt: string
  visibility_scope: VisibilityScope
  autosave_enabled: boolean
  autosave_strategy: ChatAutosaveStrategy
  autosave_interval_minutes: number
  autosave_min_messages: number
  retention_days: number
  retention_max_snapshots: number
  created_at: string
  updated_at: string
  deleted_at: string | null
  deleted_by_user_id: string | null
  delete_reason: string | null
}

export interface AdminEngramSourceInput {
  captured_at: string
  url: string
  title: string
  snippet: string
  content_text?: string | null
  content_hash?: string | null
}

export interface AdminEngramRecord {
  engram_id: string
  project_id: string
  thread_id: string | null
  title: string
  abstract: string
  detailed_summary_markdown: string
  tags: string[]
  keywords: string[]
  owner_user_id: string | null
  visibility_scope: VisibilityScope
  source_session_id: string | null
  created_at: string
  updated_at: string
  deleted_at: string | null
  deleted_by_user_id: string | null
  delete_reason: string | null
  sources: AdminEngramSourceInput[]
}

export interface AdminSessionDeleteResponse {
  session_id: string
  deleted: boolean
  linked_engrams_deleted: number
}

export interface AdminSessionRestoreResponse {
  session_id: string
  restored: boolean
}

export interface AdminEngramDeleteResponse {
  engram_id: string
  deleted: boolean
}

export interface AdminEngramRestoreResponse {
  engram_id: string
  restored: boolean
}

export interface EngramCollectionRecord {
  collection_id: string
  project_id: string
  owner_user_id: string
  name: string
  description: string
  created_at: string
  updated_at: string
  deleted_at: string | null
  deleted_by_user_id: string | null
  delete_reason: string | null
}
