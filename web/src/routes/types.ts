export type NavSection =
  | 'marketing'
  | 'workspace'
  | 'sessions'
  | 'engrams'
  | 'links'
  | 'documents'
  | 'projects'
  | 'transfer'
  | 'admin'

export type AppRouteId =
  | 'landing'
  | 'for_enterprise'
  | 'for_developers'
  | 'product'
  | 'how_it_works'
  | 'pricing'
  | 'login'
  | 'app_workspace'
  | 'app_sessions'
  | 'app_sessions_new'
  | 'app_session_chat'
  | 'app_session_lifecycle'
  | 'app_session_timeline'
  | 'app_session_save_engram'
  | 'app_session_continue'
  | 'app_engrams'
  | 'app_engrams_query'
  | 'app_engrams_detail'
  | 'app_engrams_sources'
  | 'app_engrams_rehydrate'
  | 'app_engrams_links'
  | 'app_engrams_trace'
  | 'app_session_pins_engrams'
  | 'app_session_pins_documents'
  | 'app_documents'
  | 'app_documents_ingest_text'
  | 'app_documents_ingest_file'
  | 'app_projects'
  | 'app_projects_members'
  | 'app_projects_audit'
  | 'app_projects_transfer_export'
  | 'app_projects_transfer_import'
  | 'app_admin_sessions'
  | 'app_admin_engrams'
  | 'app_admin_collections'
  | 'app_admin_curation'
  | 'app_admin_contradictions'
  | 'app_admin_tokens'
  | 'app_admin_observability'

export interface BreadcrumbSpec {
  label: string
  to?: string
}

export interface PricingPlanTeaser {
  id: 'starter' | 'team' | 'enterprise'
  name: string
  headline: string
  audience: string
  cta: string
  points: string[]
}
