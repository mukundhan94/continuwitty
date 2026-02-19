from datetime import datetime
from enum import Enum
from uuid import UUID

from pydantic import BaseModel, Field


class SupportingSource(BaseModel):
    url: str
    title: str | None = None
    snippet: str
    captured_at: datetime


class Claim(BaseModel):
    claim: str
    supporting_sources: list[SupportingSource] = Field(default_factory=list)


class Decision(BaseModel):
    decision: str
    rationale: str


class ArtifactIn(BaseModel):
    artifact_type: str
    storage_uri: str
    metadata: dict = Field(default_factory=dict)


class MemoryEngramCreate(BaseModel):
    project_id: str = ""
    thread_id: str | None = None
    title: str
    abstract: str = ""
    detailed_summary_markdown: str
    decisions: list[Decision] = Field(default_factory=list)
    assumptions: list[str] = Field(default_factory=list)
    open_questions: list[str] = Field(default_factory=list)
    claims: list[Claim] = Field(default_factory=list)
    tags: list[str] = Field(default_factory=list)
    keywords: list[str] = Field(default_factory=list)
    artifacts: list[ArtifactIn] = Field(default_factory=list)
    retrieval_text: str | None = None
    visibility_scope: str = "private"
    source_session_id: UUID | None = None


class EngramSummary(BaseModel):
    engram_id: UUID
    project_id: str
    thread_id: str | None
    title: str
    abstract: str
    created_at: datetime
    tags: list[str] = Field(default_factory=list)
    keywords: list[str] = Field(default_factory=list)
    owner_user_id: UUID | None = None
    visibility_scope: str = "private"


class EngramCreateResponse(BaseModel):
    engram_id: UUID
    created_at: datetime
    resolved_project_id: str | None = None
    used_default_project: bool = False


class EngramQueryRequest(BaseModel):
    query: str
    top_k: int = Field(default=5, ge=1, le=50)
    project_id: str | None = None
    tags: list[str] = Field(default_factory=list)
    keywords: list[str] = Field(default_factory=list)
    created_after: datetime | None = None
    created_before: datetime | None = None


class EngramQueryResult(BaseModel):
    engram_id: UUID
    project_id: str
    title: str
    abstract: str
    created_at: datetime
    tags: list[str] = Field(default_factory=list)
    keywords: list[str] = Field(default_factory=list)
    owner_user_id: UUID | None = None
    visibility_scope: str = "private"
    distance: float


class RehydrationCitation(BaseModel):
    url: str
    title: str | None = None
    snippet: str
    captured_at: datetime


class EngramSourceRecord(BaseModel):
    source_id: UUID
    engram_id: UUID
    captured_at: datetime
    url: str
    title: str | None = None
    snippet: str | None = None


class AppVersionResponse(BaseModel):
    commit_id: str
    semantic_version: str
    release: str


class UserRole(str, Enum):
    admin = "admin"
    analyst = "analyst"
    viewer = "viewer"


class McpTokenScope(str, Enum):
    read = "read"
    write = "write"


class VisibilityScope(str, Enum):
    private = "private"
    project = "project"


class ChatProvider(str, Enum):
    openai = "openai"
    anthropic = "anthropic"
    bedrock = "bedrock"


class ChatAutosaveStrategy(str, Enum):
    off = "off"
    interval = "interval"
    message_count = "message_count"


class UserRecord(BaseModel):
    user_id: UUID
    username: str
    role: UserRole
    is_active: bool
    default_project_id: str | None = None
    created_at: datetime


class ProjectRecord(BaseModel):
    project_id: str
    name: str
    description: str = ""
    owner_user_id: UUID
    is_archived: bool = False
    created_at: datetime
    updated_at: datetime


class ProjectCreateRequest(BaseModel):
    project_id: str = Field(min_length=1, max_length=120)
    name: str = Field(min_length=1, max_length=120)
    description: str = Field(default="", max_length=2_000)
    owner_user_id: UUID | None = None


class ProjectDefaultResponse(BaseModel):
    default_project_id: str | None = None


class ProjectDefaultUpdateRequest(BaseModel):
    project_id: str = Field(min_length=1, max_length=120)


class UserCreateRequest(BaseModel):
    username: str = Field(min_length=3, max_length=64, pattern=r"^[a-zA-Z0-9_.-]+$")
    password: str = Field(min_length=8, max_length=128)
    role: UserRole = UserRole.analyst
    is_active: bool = True


class UserUpdateRequest(BaseModel):
    role: UserRole | None = None
    is_active: bool | None = None
    password: str | None = Field(default=None, min_length=8, max_length=128)


class McpTokenCreateRequest(BaseModel):
    name: str = Field(min_length=3, max_length=120)
    scope: McpTokenScope = McpTokenScope.read
    allowed_tools: list[str] = Field(default_factory=list)
    allowed_project_ids: list[str] = Field(default_factory=list)
    expires_in_days: int = Field(default=90, ge=1, le=3650)


class McpTokenCreateResponse(BaseModel):
    token_id: UUID
    name: str
    scope: McpTokenScope
    allowed_tools: list[str] = Field(default_factory=list)
    allowed_project_ids: list[str] = Field(default_factory=list)
    token_secret_hint: str
    token: str
    expires_at: datetime
    created_at: datetime


class McpTokenSummary(BaseModel):
    token_id: UUID
    name: str
    scope: McpTokenScope
    allowed_tools: list[str] = Field(default_factory=list)
    allowed_project_ids: list[str] = Field(default_factory=list)
    token_secret_hint: str
    expires_at: datetime
    last_used_at: datetime | None = None
    revoked_at: datetime | None = None
    created_at: datetime
    is_active: bool


class McpTokenRevokeRequest(BaseModel):
    reason: str | None = Field(default=None, max_length=240)


class RehydrationBundle(BaseModel):
    engram_id: UUID
    project_id: str
    title: str
    compact_summary: str
    detailed_summary_markdown: str = ""
    key_decisions: list[Decision] = Field(default_factory=list)
    open_questions: list[str] = Field(default_factory=list)
    top_citations: list[RehydrationCitation] = Field(default_factory=list)
    context_markdown: str
    owner_user_id: UUID | None = None
    visibility_scope: str = "private"


class ChatSessionCreateRequest(BaseModel):
    project_id: str
    title: str
    provider: ChatProvider = ChatProvider.openai
    model_id: str = "gpt-4o-mini"
    system_prompt: str = ""
    visibility_scope: VisibilityScope = VisibilityScope.private
    autosave_enabled: bool = False
    autosave_strategy: ChatAutosaveStrategy = ChatAutosaveStrategy.off
    autosave_interval_minutes: int = Field(default=30, ge=1, le=10_080)
    autosave_min_messages: int = Field(default=6, ge=1, le=500)
    retention_days: int = Field(default=30, ge=1, le=3_650)
    retention_max_snapshots: int = Field(default=60, ge=1, le=10_000)


class ChatSessionUpdateRequest(BaseModel):
    title: str | None = None
    provider: ChatProvider | None = None
    model_id: str | None = None
    system_prompt: str | None = None
    visibility_scope: VisibilityScope | None = None
    autosave_enabled: bool | None = None
    autosave_strategy: ChatAutosaveStrategy | None = None
    autosave_interval_minutes: int | None = Field(default=None, ge=1, le=10_080)
    autosave_min_messages: int | None = Field(default=None, ge=1, le=500)
    retention_days: int | None = Field(default=None, ge=1, le=3_650)
    retention_max_snapshots: int | None = Field(default=None, ge=1, le=10_000)


class ChatSessionRecord(BaseModel):
    session_id: UUID
    owner_user_id: UUID
    project_id: str
    title: str
    provider: ChatProvider
    model_id: str
    system_prompt: str
    visibility_scope: VisibilityScope
    autosave_enabled: bool
    autosave_strategy: ChatAutosaveStrategy = ChatAutosaveStrategy.off
    autosave_interval_minutes: int = 30
    autosave_min_messages: int = 6
    retention_days: int = 30
    retention_max_snapshots: int = 60
    created_at: datetime
    updated_at: datetime


class ChatMessageCreateRequest(BaseModel):
    content_text: str


class ChatLifecyclePolicy(BaseModel):
    autosave_enabled: bool
    autosave_strategy: ChatAutosaveStrategy
    autosave_interval_minutes: int = Field(ge=1)
    autosave_min_messages: int = Field(ge=1)
    retention_days: int = Field(ge=1)
    retention_max_snapshots: int = Field(ge=1)


class ChatLifecyclePolicyUpdateRequest(BaseModel):
    autosave_enabled: bool | None = None
    autosave_strategy: ChatAutosaveStrategy | None = None
    autosave_interval_minutes: int | None = Field(default=None, ge=1, le=10_080)
    autosave_min_messages: int | None = Field(default=None, ge=1, le=500)
    retention_days: int | None = Field(default=None, ge=1, le=3_650)
    retention_max_snapshots: int | None = Field(default=None, ge=1, le=10_000)


class ChatTimelineEvent(BaseModel):
    event_id: UUID
    session_id: UUID
    event_type: str
    title: str
    abstract: str
    tags: list[str] = Field(default_factory=list)
    created_at: datetime


class ChatMessageRecord(BaseModel):
    message_id: UUID
    session_id: UUID
    role: str
    content_text: str
    provider: str | None = None
    model_id: str | None = None
    token_usage_json: dict = Field(default_factory=dict)
    used_engram_ids: list[UUID] = Field(default_factory=list)
    created_at: datetime


class PinEngramRequest(BaseModel):
    engram_id: UUID


class PinnedEngramRecord(BaseModel):
    session_id: UUID
    engram_id: UUID
    pinned_by_user_id: UUID
    created_at: datetime


class PinDocumentRequest(BaseModel):
    document_id: UUID


class PinnedDocumentRecord(BaseModel):
    session_id: UUID
    document_id: UUID
    pinned_by_user_id: UUID
    created_at: datetime


class ChatSourceReference(BaseModel):
    source_type: str = "engram_source"
    engram_id: UUID
    engram_title: str
    url: str
    title: str | None = None
    snippet: str
    captured_at: datetime
    document_id: UUID | None = None
    chunk_id: UUID | None = None
    chunk_index: int | None = None


class ChatDebugEmbeddingCall(BaseModel):
    operation: str
    provider_id: str
    duration_ms: float
    item_count: int
    text_chars: int
    dim: int
    used_fallback: bool = False


class ChatDebugLLMCall(BaseModel):
    provider: str
    model_id: str
    call_type: str = "generate"
    duration_ms: float
    input_chars: int
    output_chars: int
    token_usage: dict[str, int] = Field(default_factory=dict)
    token_usage_is_estimated: bool = False


class ChatDebugProviderMessage(BaseModel):
    role: str
    content_preview: str
    char_count: int


class ChatDebugTrace(BaseModel):
    total_duration_ms: float
    prepare_duration_ms: float
    context_duration_ms: float
    history_load_duration_ms: float
    llm_call_duration_ms: float
    persistence_duration_ms: float
    used_engram_count: int
    used_document_chunk_count: int
    source_reference_count: int
    provider: str
    model_id: str
    request_input_text: str
    response_output_text: str
    provider_system_prompt_preview: str
    provider_messages: list[ChatDebugProviderMessage] = Field(default_factory=list)
    embedding_calls: list[ChatDebugEmbeddingCall] = Field(default_factory=list)
    llm_calls: list[ChatDebugLLMCall] = Field(default_factory=list)


class ChatSendResponse(BaseModel):
    session_id: UUID
    message_id: UUID
    reply_message_id: UUID
    assistant_text: str
    used_engram_ids: list[UUID] = Field(default_factory=list)
    used_document_chunk_ids: list[UUID] = Field(default_factory=list)
    source_references: list[ChatSourceReference] = Field(default_factory=list)
    debug_trace: ChatDebugTrace | None = None


class SaveSessionAsEngramRequest(BaseModel):
    title: str
    abstract: str = ""
    visibility_scope: VisibilityScope = VisibilityScope.private
    tags: list[str] = Field(default_factory=list)
    keywords: list[str] = Field(default_factory=list)


class EngramCreateFromConversationRequest(BaseModel):
    project_id: str = ""
    conversation_markdown: str
    thread_id: str | None = None
    title: str = "Conversation Snapshot"
    abstract: str = ""
    visibility_scope: VisibilityScope = VisibilityScope.private
    tags: list[str] = Field(default_factory=list)
    keywords: list[str] = Field(default_factory=list)
    retrieval_text: str | None = None
    source_session_id: UUID | None = None


class SaveSessionAsEngramResponse(BaseModel):
    engram_id: UUID
    session_id: UUID
    created_at: datetime


class ContinueSessionRequest(BaseModel):
    title: str | None = None


class ContinueSessionResponse(BaseModel):
    session: ChatSessionRecord
    carried_engram_ids: list[UUID] = Field(default_factory=list)


class AdminChatSessionRecord(BaseModel):
    session_id: UUID
    owner_user_id: UUID
    project_id: str
    title: str
    provider: ChatProvider
    model_id: str
    system_prompt: str
    visibility_scope: VisibilityScope
    autosave_enabled: bool
    autosave_strategy: ChatAutosaveStrategy = ChatAutosaveStrategy.off
    autosave_interval_minutes: int = 30
    autosave_min_messages: int = 6
    retention_days: int = 30
    retention_max_snapshots: int = 60
    created_at: datetime
    updated_at: datetime
    deleted_at: datetime | None = None
    deleted_by_user_id: UUID | None = None
    delete_reason: str | None = None


class AdminSessionDeleteRequest(BaseModel):
    delete_linked_engrams: bool = False
    reason: str | None = Field(default=None, max_length=240)


class AdminSessionDeleteResponse(BaseModel):
    session_id: UUID
    deleted: bool
    linked_engrams_deleted: int = 0


class AdminSessionRestoreResponse(BaseModel):
    session_id: UUID
    restored: bool


class AdminEngramSourceInput(BaseModel):
    captured_at: datetime
    url: str
    title: str | None = None
    snippet: str | None = None
    content_text: str | None = None
    content_hash: str | None = None


class AdminEngramSourceRecord(AdminEngramSourceInput):
    source_id: UUID


class AdminEngramRecord(BaseModel):
    engram_id: UUID
    project_id: str
    thread_id: str | None = None
    title: str
    abstract: str
    detailed_summary_markdown: str
    tags: list[str] = Field(default_factory=list)
    keywords: list[str] = Field(default_factory=list)
    owner_user_id: UUID | None = None
    visibility_scope: str = "private"
    source_session_id: UUID | None = None
    created_at: datetime
    updated_at: datetime
    deleted_at: datetime | None = None
    deleted_by_user_id: UUID | None = None
    delete_reason: str | None = None
    sources: list[AdminEngramSourceRecord] = Field(default_factory=list)


class AdminEngramUpdateRequest(BaseModel):
    title: str | None = Field(default=None, min_length=1, max_length=300)
    abstract: str | None = Field(default=None, max_length=5_000)
    detailed_summary_markdown: str | None = None
    tags: list[str] | None = None
    keywords: list[str] | None = None
    visibility_scope: VisibilityScope | None = None
    sources: list[AdminEngramSourceInput] | None = None
    expected_updated_at: datetime | None = None


class AdminEngramMoveRequest(BaseModel):
    target_project_id: str = Field(min_length=1, max_length=120)
    reason: str | None = Field(default=None, max_length=240)
    expected_updated_at: datetime | None = None


class AdminEngramDeleteRequest(BaseModel):
    reason: str | None = Field(default=None, max_length=240)


class AdminEngramDeleteResponse(BaseModel):
    engram_id: UUID
    deleted: bool


class AdminEngramRestoreResponse(BaseModel):
    engram_id: UUID
    restored: bool


class EngramCollectionRecord(BaseModel):
    collection_id: UUID
    project_id: str
    owner_user_id: UUID
    name: str
    description: str = ""
    created_at: datetime
    updated_at: datetime
    deleted_at: datetime | None = None
    deleted_by_user_id: UUID | None = None
    delete_reason: str | None = None


class EngramCollectionCreateRequest(BaseModel):
    project_id: str = Field(min_length=1, max_length=120)
    name: str = Field(min_length=1, max_length=240)
    description: str = Field(default="", max_length=2_000)


class EngramCollectionUpdateRequest(BaseModel):
    name: str | None = Field(default=None, min_length=1, max_length=240)
    description: str | None = Field(default=None, max_length=2_000)
    expected_updated_at: datetime | None = None


class EngramCollectionDeleteRequest(BaseModel):
    reason: str | None = Field(default=None, max_length=240)


class EngramCollectionItemsUpdateRequest(BaseModel):
    engram_ids: list[UUID] = Field(default_factory=list)


class McpJsonRpcRequest(BaseModel):
    jsonrpc: str = "2.0"
    id: str | int | None = None
    method: str
    params: dict = Field(default_factory=dict)


class McpJsonRpcResponse(BaseModel):
    jsonrpc: str = "2.0"
    id: str | int | None = None
    result: dict | None = None
    error: dict | None = None
