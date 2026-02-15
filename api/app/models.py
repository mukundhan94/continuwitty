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
    project_id: str
    thread_id: str | None = None
    title: str
    abstract: str
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


class UserRole(str, Enum):
    admin = "admin"
    analyst = "analyst"
    viewer = "viewer"


class VisibilityScope(str, Enum):
    private = "private"
    project = "project"


class ChatProvider(str, Enum):
    openai = "openai"
    anthropic = "anthropic"
    bedrock = "bedrock"


class UserRecord(BaseModel):
    user_id: UUID
    username: str
    role: UserRole
    is_active: bool
    created_at: datetime


class UserCreateRequest(BaseModel):
    username: str = Field(min_length=3, max_length=64, pattern=r"^[a-zA-Z0-9_.-]+$")
    password: str = Field(min_length=8, max_length=128)
    role: UserRole = UserRole.analyst
    is_active: bool = True


class UserUpdateRequest(BaseModel):
    role: UserRole | None = None
    is_active: bool | None = None
    password: str | None = Field(default=None, min_length=8, max_length=128)


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


class ChatSessionUpdateRequest(BaseModel):
    title: str | None = None
    provider: ChatProvider | None = None
    model_id: str | None = None
    system_prompt: str | None = None
    visibility_scope: VisibilityScope | None = None
    autosave_enabled: bool | None = None


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
    created_at: datetime
    updated_at: datetime


class ChatMessageCreateRequest(BaseModel):
    content_text: str


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


class ChatSendResponse(BaseModel):
    session_id: UUID
    message_id: UUID
    reply_message_id: UUID
    assistant_text: str
    used_engram_ids: list[UUID] = Field(default_factory=list)
    used_document_chunk_ids: list[UUID] = Field(default_factory=list)
    source_references: list[ChatSourceReference] = Field(default_factory=list)


class SaveSessionAsEngramRequest(BaseModel):
    title: str
    abstract: str
    visibility_scope: VisibilityScope = VisibilityScope.private
    tags: list[str] = Field(default_factory=list)
    keywords: list[str] = Field(default_factory=list)


class SaveSessionAsEngramResponse(BaseModel):
    engram_id: UUID
    session_id: UUID
    created_at: datetime


class ContinueSessionRequest(BaseModel):
    title: str | None = None


class ContinueSessionResponse(BaseModel):
    session: ChatSessionRecord
    carried_engram_ids: list[UUID] = Field(default_factory=list)


class McpJsonRpcRequest(BaseModel):
    jsonrpc: str = "2.0"
    id: str | int
    method: str
    params: dict = Field(default_factory=dict)


class McpJsonRpcResponse(BaseModel):
    jsonrpc: str = "2.0"
    id: str | int
    result: dict | None = None
    error: dict | None = None
