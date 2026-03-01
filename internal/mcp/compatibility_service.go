package mcp

import (
	"context"
	"strings"
	"time"

	"engram/internal/governance"
	"engram/internal/models"
	"engram/internal/projects"

	"github.com/google/uuid"
)

const (
	mcpProtocolVersion = "2024-11-05"
	mcpServerName      = "engram-vault-mcp"
)

// ProjectListService captures project listing behavior used by MCP compatibility tool dispatch.
type ProjectListService interface {
	ListProjects(
		ctx context.Context,
		actorUserID uuid.UUID,
		actorRole models.UserRole,
		includeArchived bool,
		limit int,
		offset int,
	) ([]models.ProjectRecord, error)
	CreateProject(
		ctx context.Context,
		actorUserID uuid.UUID,
		actorRole models.UserRole,
		request projects.CreateProjectRequest,
	) (*models.ProjectRecord, error)
	GetDefaultProjectID(ctx context.Context, actorUserID uuid.UUID) (*string, error)
	SetDefaultProjectID(
		ctx context.Context,
		actorUserID uuid.UUID,
		actorRole models.UserRole,
		projectID string,
	) (string, error)
	ListProjectMembers(
		ctx context.Context,
		actorUserID uuid.UUID,
		actorRole models.UserRole,
		projectID string,
		includeRevoked bool,
		limit int,
		offset int,
	) ([]models.ProjectMemberRecord, error)
	AddProjectMember(
		ctx context.Context,
		actorUserID uuid.UUID,
		actorRole models.UserRole,
		request projects.ProjectMemberCreateRequest,
	) (*models.ProjectMemberRecord, error)
	UpdateProjectMember(
		ctx context.Context,
		actorUserID uuid.UUID,
		actorRole models.UserRole,
		request projects.ProjectMemberUpdateRequest,
	) (*models.ProjectMemberRecord, error)
	RemoveProjectMember(
		ctx context.Context,
		actorUserID uuid.UUID,
		actorRole models.UserRole,
		request projects.ProjectMemberRemoveRequest,
	) error
	ShareEngram(
		ctx context.Context,
		actorUserID uuid.UUID,
		actorRole models.UserRole,
		engramID uuid.UUID,
	) (*models.EngramVisibilityRecord, error)
	UnshareEngram(
		ctx context.Context,
		actorUserID uuid.UUID,
		actorRole models.UserRole,
		engramID uuid.UUID,
	) (*models.EngramVisibilityRecord, error)
}

// ProjectExportService captures project export-bundle behavior used by MCP compatibility dispatch.
type ProjectExportService interface {
	ExportProjectBundle(
		ctx context.Context,
		request ProjectExportRequest,
	) (*ProjectExportResponse, error)
}

// ProjectImportService captures project import-bundle behavior used by MCP compatibility dispatch.
type ProjectImportService interface {
	ImportProjectBundle(
		ctx context.Context,
		request ProjectImportRequest,
	) (*ProjectImportResponse, error)
}

// SessionListService captures session listing behavior used by MCP compatibility user dispatch.
type SessionListService interface {
	ListSessions(
		ctx context.Context,
		request SessionListRequest,
	) ([]models.ChatSessionRecord, error)
}

// SessionGetService captures session lookup behavior used by MCP compatibility chat dispatch.
type SessionGetService interface {
	GetSession(
		ctx context.Context,
		request SessionGetRequest,
	) (*models.ChatSessionRecord, error)
}

// SessionCreateService captures session creation behavior used by MCP compatibility chat dispatch.
type SessionCreateService interface {
	CreateSession(
		ctx context.Context,
		request SessionCreateRequest,
	) (*models.ChatSessionRecord, error)
}

// SessionContinueService captures session continuation behavior used by MCP compatibility chat dispatch.
type SessionContinueService interface {
	ContinueSession(
		ctx context.Context,
		request SessionContinueRequest,
	) (*models.ContinueSessionResponse, error)
}

// SessionSaveAsEngramService captures save-as-engram behavior used by MCP compatibility chat dispatch.
type SessionSaveAsEngramService interface {
	SaveSessionAsEngram(
		ctx context.Context,
		request SessionSaveAsEngramRequest,
	) (*models.SaveSessionAsEngramResponse, error)
}

// SessionDeleteService captures session-delete behavior used by MCP compatibility chat dispatch.
type SessionDeleteService interface {
	DeleteSession(
		ctx context.Context,
		request SessionDeleteRequest,
	) (*SessionDeleteResponse, error)
}

// SessionRestoreService captures session-restore behavior used by MCP compatibility chat dispatch.
type SessionRestoreService interface {
	RestoreSession(
		ctx context.Context,
		request SessionRestoreRequest,
	) (*SessionRestoreResponse, error)
}

// LifecyclePolicyUpdateService captures lifecycle-policy updates used by MCP compatibility chat dispatch.
type LifecyclePolicyUpdateService interface {
	UpdateLifecyclePolicy(
		ctx context.Context,
		request SessionLifecyclePolicyUpdateRequest,
	) (*models.ChatSessionRecord, error)
}

// MessageListService captures message listing behavior used by MCP compatibility chat dispatch.
type MessageListService interface {
	ListMessages(
		ctx context.Context,
		request MessageListRequest,
	) ([]models.ChatMessageRecord, error)
}

// MessageSendService captures message-send behavior used by MCP compatibility chat dispatch.
type MessageSendService interface {
	SendMessage(
		ctx context.Context,
		request SessionMessageSendRequest,
	) (*MessageSendResponse, error)
}

// MessageStreamService captures send-message stream behavior used by MCP compatibility transport.
type MessageStreamService interface {
	StreamMessageEvents(
		ctx context.Context,
		request SessionMessageSendRequest,
	) ([]MessageStreamEvent, error)
}

// TimelineListService captures timeline listing behavior used by MCP compatibility chat dispatch.
type TimelineListService interface {
	ListTimeline(
		ctx context.Context,
		request TimelineListRequest,
	) ([]models.ChatTimelineEvent, error)
}

// PinnedEngramListService captures pinned-engram listing behavior used by MCP compatibility chat dispatch.
type PinnedEngramListService interface {
	ListPinnedEngrams(
		ctx context.Context,
		request SessionScopedRequest,
	) ([]models.EngramSummary, error)
}

// PinnedDocumentListService captures pinned-document listing behavior used by MCP compatibility chat dispatch.
type PinnedDocumentListService interface {
	ListPinnedDocuments(
		ctx context.Context,
		request SessionScopedRequest,
	) ([]models.PinnedDocumentRecord, error)
}

// ProjectDocumentListService captures project-document listing behavior used by MCP compatibility chat dispatch.
type ProjectDocumentListService interface {
	ListProjectDocuments(
		ctx context.Context,
		request ProjectDocumentListRequest,
	) ([]models.DocumentRecord, error)
}

// EngramListService captures engram listing behavior used by MCP compatibility engram dispatch.
type EngramListService interface {
	ListEngrams(
		ctx context.Context,
		request EngramListRequest,
	) ([]models.AdminEngramRecord, error)
}

// EngramGetService captures engram lookup behavior used by MCP compatibility engram dispatch.
type EngramGetService interface {
	GetEngram(
		ctx context.Context,
		request EngramGetRequest,
	) (*models.AdminEngramRecord, error)
}

// EngramQueryService captures query behavior used by MCP compatibility engram dispatch.
type EngramQueryService interface {
	QueryEngrams(
		ctx context.Context,
		request EngramQueryDispatchRequest,
	) ([]models.EngramQueryResult, error)
}

// EngramRehydrateService captures rehydration behavior used by MCP compatibility engram dispatch.
type EngramRehydrateService interface {
	RehydrateEngram(
		ctx context.Context,
		request EngramRehydrateRequest,
	) (*models.RehydrationBundle, error)
}

// EngramCreateService captures engram create behavior used by MCP compatibility engram dispatch.
type EngramCreateService interface {
	CreateEngram(
		ctx context.Context,
		request EngramCreateRequest,
	) (*models.EngramCreateResponse, error)
}

// EngramCreateFromConversationService captures conversation-only create behavior used by MCP compatibility engram dispatch.
type EngramCreateFromConversationService interface {
	CreateEngramFromConversation(
		ctx context.Context,
		request EngramCreateFromConversationRequest,
	) (*EngramCreateFromConversationResponse, error)
}

// EngramUpdateService captures engram update behavior used by MCP compatibility engram dispatch.
type EngramUpdateService interface {
	UpdateEngram(
		ctx context.Context,
		request EngramUpdateRequest,
	) (*models.AdminEngramRecord, error)
}

// EngramMoveService captures engram move-project behavior used by MCP compatibility engram dispatch.
type EngramMoveService interface {
	MoveEngram(
		ctx context.Context,
		request EngramMoveRequest,
	) (*models.AdminEngramRecord, error)
}

// EngramDeleteService captures engram delete behavior used by MCP compatibility engram dispatch.
type EngramDeleteService interface {
	DeleteEngram(
		ctx context.Context,
		request EngramDeleteRequest,
	) (*EngramDeleteResponse, error)
}

// EngramRestoreService captures engram restore behavior used by MCP compatibility engram dispatch.
type EngramRestoreService interface {
	RestoreEngram(
		ctx context.Context,
		request EngramRestoreRequest,
	) (*EngramRestoreResponse, error)
}

// EngramCollectionListService captures collection-list behavior used by MCP compatibility engram dispatch.
type EngramCollectionListService interface {
	ListCollections(
		ctx context.Context,
		request EngramCollectionListRequest,
	) ([]models.EngramCollectionRecord, error)
}

// EngramCollectionGetService captures collection lookup behavior used by MCP compatibility policies.
type EngramCollectionGetService interface {
	GetCollection(
		ctx context.Context,
		request EngramCollectionGetRequest,
	) (*models.EngramCollectionRecord, error)
}

// EngramCollectionCreateService captures collection-create behavior used by MCP compatibility engram dispatch.
type EngramCollectionCreateService interface {
	CreateCollection(
		ctx context.Context,
		request EngramCollectionCreateRequest,
	) (*EngramCollectionCreateResponse, error)
}

// EngramCollectionUpdateService captures collection-update behavior used by MCP compatibility engram dispatch.
type EngramCollectionUpdateService interface {
	UpdateCollection(
		ctx context.Context,
		request EngramCollectionUpdateRequest,
	) (*models.EngramCollectionRecord, error)
}

// EngramCollectionDeleteService captures collection-delete behavior used by MCP compatibility engram dispatch.
type EngramCollectionDeleteService interface {
	DeleteCollection(
		ctx context.Context,
		request EngramCollectionDeleteRequest,
	) (*EngramCollectionDeleteResponse, error)
}

// EngramCollectionAddItemsService captures collection item-add behavior used by MCP compatibility engram dispatch.
type EngramCollectionAddItemsService interface {
	AddCollectionItems(
		ctx context.Context,
		request EngramCollectionAddItemsRequest,
	) (*EngramCollectionAddItemsResponse, error)
}

// EngramCollectionRemoveItemService captures collection item-remove behavior used by MCP compatibility engram dispatch.
type EngramCollectionRemoveItemService interface {
	RemoveCollectionItem(
		ctx context.Context,
		request EngramCollectionRemoveItemRequest,
	) (*EngramCollectionRemoveItemResponse, error)
}

// PinEngramService captures engram pinning behavior used by MCP compatibility chat dispatch.
type PinEngramService interface {
	PinEngram(
		ctx context.Context,
		request SessionPinEngramRequest,
	) (*models.PinnedEngramRecord, error)
}

// UnpinEngramService captures engram unpinning behavior used by MCP compatibility chat dispatch.
type UnpinEngramService interface {
	UnpinEngram(
		ctx context.Context,
		request SessionPinEngramRequest,
	) (bool, error)
}

// PinDocumentService captures document pinning behavior used by MCP compatibility chat dispatch.
type PinDocumentService interface {
	PinDocument(
		ctx context.Context,
		request SessionPinDocumentRequest,
	) (*models.PinnedDocumentRecord, error)
}

// UnpinDocumentService captures document unpinning behavior used by MCP compatibility chat dispatch.
type UnpinDocumentService interface {
	UnpinDocument(
		ctx context.Context,
		request SessionPinDocumentRequest,
	) (bool, error)
}

// SessionListRequest captures compatibility-level session list inputs.
type SessionListRequest struct {
	ActorUserID uuid.UUID
	ProjectID   *string
	Limit       int
	Offset      int
}

// ProjectExportRequest captures compatibility-level project export inputs.
type ProjectExportRequest struct {
	ActorUserID       uuid.UUID
	ActorRole         models.UserRole
	ProjectID         string
	CollectionIDs     []uuid.UUID
	IncludeEmbeddings bool
}

// ProjectExportResponse captures compatibility-level project export outputs.
type ProjectExportResponse struct {
	Bundle map[string]any `json:"bundle"`
}

// ProjectImportRequest captures compatibility-level project import inputs.
type ProjectImportRequest struct {
	ActorUserID     uuid.UUID
	ActorRole       models.UserRole
	TargetProjectID string
	BundleBytes     []byte
	ConflictPolicy  string
}

// ProjectImportResponse captures compatibility-level project import outputs.
type ProjectImportResponse struct {
	Summary map[string]any `json:"summary"`
}

// SessionCreateRequest captures compatibility-level session create inputs.
type SessionCreateRequest struct {
	ActorUserID uuid.UUID
	Payload     models.ChatSessionCreateRequest
}

// SessionContinueRequest captures compatibility-level continue-session inputs.
type SessionContinueRequest struct {
	ActorUserID uuid.UUID
	SessionID   uuid.UUID
	Payload     models.ContinueSessionRequest
}

// SessionSaveAsEngramRequest captures compatibility-level save-session inputs.
type SessionSaveAsEngramRequest struct {
	ActorUserID uuid.UUID
	SessionID   uuid.UUID
	Payload     models.SaveSessionAsEngramRequest
}

// SessionDeleteRequest captures compatibility-level delete-session inputs.
type SessionDeleteRequest struct {
	ActorUserID         uuid.UUID
	ActorRole           models.UserRole
	SessionID           uuid.UUID
	DeleteLinkedEngrams bool
	Reason              *string
}

// SessionDeleteResponse captures delete-session outputs.
type SessionDeleteResponse struct {
	SessionID            uuid.UUID `json:"session_id"`
	Deleted              bool      `json:"deleted"`
	LinkedEngramsDeleted int       `json:"linked_engrams_deleted"`
}

// SessionRestoreRequest captures compatibility-level restore-session inputs.
type SessionRestoreRequest struct {
	ActorUserID uuid.UUID
	ActorRole   models.UserRole
	SessionID   uuid.UUID
}

// SessionRestoreResponse captures restore-session outputs.
type SessionRestoreResponse struct {
	SessionID uuid.UUID `json:"session_id"`
	Restored  bool      `json:"restored"`
}

// SessionLifecyclePolicyUpdateRequest captures compatibility-level lifecycle update inputs.
type SessionLifecyclePolicyUpdateRequest struct {
	ActorUserID             uuid.UUID
	SessionID               uuid.UUID
	AutosaveEnabled         *bool
	AutosaveStrategy        *models.ChatAutosaveStrategy
	AutosaveIntervalMinutes *int
	AutosaveMinMessages     *int
	RetentionDays           *int
	RetentionMaxSnapshots   *int
}

// SessionGetRequest captures compatibility-level session lookup inputs.
type SessionGetRequest struct {
	ActorUserID uuid.UUID
	SessionID   uuid.UUID
}

// MessageListRequest captures compatibility-level message list inputs.
type MessageListRequest struct {
	ActorUserID uuid.UUID
	SessionID   uuid.UUID
	Limit       int
	Offset      int
}

// SessionMessageSendRequest captures compatibility-level send-message inputs.
type SessionMessageSendRequest struct {
	ActorUserID uuid.UUID
	SessionID   uuid.UUID
	ContentText string
}

// MessageSendResponse captures send-message outputs.
type MessageSendResponse struct {
	SessionID            uuid.UUID      `json:"session_id"`
	MessageID            uuid.UUID      `json:"message_id"`
	ReplyMessageID       uuid.UUID      `json:"reply_message_id"`
	AssistantText        string         `json:"assistant_text"`
	PromptPolicyVersion  string         `json:"prompt_policy_version,omitempty"`
	UsedEngramIDs        []uuid.UUID    `json:"used_engram_ids"`
	UsedDocumentChunkIDs []uuid.UUID    `json:"used_document_chunk_ids"`
	SourceReferences     any            `json:"source_references"`
	DebugTrace           map[string]any `json:"debug_trace,omitempty"`
}

// MessageStreamEvent captures compatibility-level stream event payload emitted by chat.send_message.
type MessageStreamEvent struct {
	Type    string         `json:"type"`
	Payload map[string]any `json:"payload"`
}

// TimelineListRequest captures compatibility-level timeline list inputs.
type TimelineListRequest struct {
	ActorUserID uuid.UUID
	SessionID   uuid.UUID
	Limit       int
	Offset      int
}

// SessionScopedRequest captures compatibility-level actor/session identity inputs.
type SessionScopedRequest struct {
	ActorUserID uuid.UUID
	SessionID   uuid.UUID
}

// ProjectDocumentListRequest captures compatibility-level project document list inputs.
type ProjectDocumentListRequest struct {
	ActorUserID uuid.UUID
	ProjectID   *string
	Limit       int
	Offset      int
}

// SessionPinEngramRequest captures compatibility-level pin-engram inputs.
type SessionPinEngramRequest struct {
	ActorUserID uuid.UUID
	SessionID   uuid.UUID
	EngramID    uuid.UUID
}

// SessionPinDocumentRequest captures compatibility-level pin-document inputs.
type SessionPinDocumentRequest struct {
	ActorUserID uuid.UUID
	SessionID   uuid.UUID
	DocumentID  uuid.UUID
}

// EngramListRequest captures compatibility-level engram list inputs.
type EngramListRequest struct {
	ActorUserID    uuid.UUID
	ActorRole      models.UserRole
	ProjectID      *string
	SessionID      *uuid.UUID
	QueryText      *string
	IncludeDeleted bool
	Limit          int
	Offset         int
}

// EngramGetRequest captures compatibility-level engram get inputs.
type EngramGetRequest struct {
	ActorUserID    uuid.UUID
	ActorRole      models.UserRole
	EngramID       uuid.UUID
	IncludeDeleted bool
}

// EngramQueryDispatchRequest captures compatibility-level engram query inputs.
type EngramQueryDispatchRequest struct {
	ActorUserID uuid.UUID
	Payload     models.EngramQueryRequest
}

// EngramRehydrateRequest captures compatibility-level engram rehydration inputs.
type EngramRehydrateRequest struct {
	ActorUserID uuid.UUID
	EngramID    uuid.UUID
}

// EngramCreateRequest captures compatibility-level engram create inputs.
type EngramCreateRequest struct {
	ActorUserID uuid.UUID
	ActorRole   models.UserRole
	Payload     models.MemoryEngramCreate
}

// EngramCreateFromConversationRequest captures compatibility-level conversation create inputs.
type EngramCreateFromConversationRequest struct {
	ActorUserID          uuid.UUID
	ActorRole            models.UserRole
	ProjectID            string
	ThreadID             *string
	Title                string
	Abstract             string
	ConversationMarkdown string
	Tags                 []string
	Keywords             []string
	VisibilityScope      string
	RetrievalText        *string
	SourceSessionID      *uuid.UUID
	EnrichmentOrigin     string
}

// EngramCreateFromConversationResponse captures compatibility-level conversation create outputs.
type EngramCreateFromConversationResponse struct {
	Engram           models.EngramCreateResponse `json:"engram"`
	EnrichmentReport map[string]any              `json:"enrichment_report"`
}

// EngramUpdateRequest captures compatibility-level engram update inputs.
type EngramUpdateRequest struct {
	ActorUserID             uuid.UUID
	ActorRole               models.UserRole
	EngramID                uuid.UUID
	ExpectedUpdatedAt       *time.Time
	Title                   *string
	Abstract                *string
	DetailedSummaryMarkdown *string
	Tags                    *[]string
	Keywords                *[]string
	VisibilityScope         *models.VisibilityScope
	Sources                 *[]models.AdminEngramSourceInput
}

// EngramMoveRequest captures compatibility-level engram move inputs.
type EngramMoveRequest struct {
	ActorUserID       uuid.UUID
	ActorRole         models.UserRole
	EngramID          uuid.UUID
	TargetProjectID   string
	Reason            *string
	ExpectedUpdatedAt *time.Time
}

// EngramDeleteRequest captures compatibility-level engram delete inputs.
type EngramDeleteRequest struct {
	ActorUserID uuid.UUID
	ActorRole   models.UserRole
	EngramID    uuid.UUID
	Reason      *string
}

// EngramDeleteResponse captures compatibility-level engram delete outputs.
type EngramDeleteResponse struct {
	EngramID uuid.UUID `json:"engram_id"`
	Deleted  bool      `json:"deleted"`
}

// EngramRestoreRequest captures compatibility-level engram restore inputs.
type EngramRestoreRequest struct {
	ActorUserID uuid.UUID
	ActorRole   models.UserRole
	EngramID    uuid.UUID
}

// EngramRestoreResponse captures compatibility-level engram restore outputs.
type EngramRestoreResponse struct {
	EngramID uuid.UUID `json:"engram_id"`
	Restored bool      `json:"restored"`
}

// EngramCollectionListRequest captures compatibility-level collection list inputs.
type EngramCollectionListRequest struct {
	ActorUserID    uuid.UUID
	ActorRole      models.UserRole
	ProjectID      *string
	IncludeDeleted bool
	Limit          int
	Offset         int
}

// EngramCollectionGetRequest captures compatibility-level collection lookup inputs.
type EngramCollectionGetRequest struct {
	ActorUserID    uuid.UUID
	ActorRole      models.UserRole
	CollectionID   uuid.UUID
	IncludeDeleted bool
}

// EngramCollectionCreateRequest captures compatibility-level collection create inputs.
type EngramCollectionCreateRequest struct {
	ActorUserID uuid.UUID
	ActorRole   models.UserRole
	ProjectID   string
	Name        string
	Description string
}

// EngramCollectionCreateResponse captures compatibility-level collection create outputs.
type EngramCollectionCreateResponse struct {
	Collection         models.EngramCollectionRecord `json:"collection"`
	ResolvedProjectID  string                        `json:"resolved_project_id"`
	UsedDefaultProject bool                          `json:"used_default_project"`
}

// EngramCollectionUpdateRequest captures compatibility-level collection update inputs.
type EngramCollectionUpdateRequest struct {
	ActorUserID       uuid.UUID
	ActorRole         models.UserRole
	CollectionID      uuid.UUID
	ExpectedUpdatedAt *time.Time
	Name              *string
	Description       *string
}

// EngramCollectionDeleteRequest captures compatibility-level collection delete inputs.
type EngramCollectionDeleteRequest struct {
	ActorUserID  uuid.UUID
	ActorRole    models.UserRole
	CollectionID uuid.UUID
	Reason       *string
}

// EngramCollectionDeleteResponse captures compatibility-level collection delete outputs.
type EngramCollectionDeleteResponse struct {
	Deleted bool `json:"deleted"`
}

// EngramCollectionAddItemsRequest captures compatibility-level collection item-add inputs.
type EngramCollectionAddItemsRequest struct {
	ActorUserID  uuid.UUID
	ActorRole    models.UserRole
	CollectionID uuid.UUID
	EngramIDs    []uuid.UUID
}

// EngramCollectionAddItemsResponse captures compatibility-level collection item-add outputs.
type EngramCollectionAddItemsResponse struct {
	Added int `json:"added"`
}

// EngramCollectionRemoveItemRequest captures compatibility-level collection item-remove inputs.
type EngramCollectionRemoveItemRequest struct {
	ActorUserID  uuid.UUID
	ActorRole    models.UserRole
	CollectionID uuid.UUID
	EngramID     uuid.UUID
}

// EngramCollectionRemoveItemResponse captures compatibility-level collection item-remove outputs.
type EngramCollectionRemoveItemResponse struct {
	Removed bool `json:"removed"`
}

// CompatibilityServiceDependencies captures optional service dependencies for compatibility dispatch.
type CompatibilityServiceDependencies struct {
	MCPToolPolicyVersion     string
	ProjectService           ProjectListService
	ProjectExport            ProjectExportService
	ProjectImport            ProjectImportService
	SessionService           SessionListService
	SessionGet               SessionGetService
	SessionCreate            SessionCreateService
	SessionContinue          SessionContinueService
	SessionSaveAsEngram      SessionSaveAsEngramService
	SessionDelete            SessionDeleteService
	SessionRestore           SessionRestoreService
	LifecyclePolicyUpdate    LifecyclePolicyUpdateService
	MessageService           MessageListService
	MessageSend              MessageSendService
	MessageStream            MessageStreamService
	TimelineService          TimelineListService
	PinnedEngramService      PinnedEngramListService
	PinnedDocumentService    PinnedDocumentListService
	ProjectDocumentService   ProjectDocumentListService
	EngramList               EngramListService
	EngramGet                EngramGetService
	EngramQuery              EngramQueryService
	EngramRehydrate          EngramRehydrateService
	EngramCreate             EngramCreateService
	EngramCreateConversation EngramCreateFromConversationService
	EngramUpdate             EngramUpdateService
	EngramMove               EngramMoveService
	EngramDelete             EngramDeleteService
	EngramRestore            EngramRestoreService
	EngramCollectionList     EngramCollectionListService
	EngramCollectionGet      EngramCollectionGetService
	EngramCollectionCreate   EngramCollectionCreateService
	EngramCollectionUpdate   EngramCollectionUpdateService
	EngramCollectionDelete   EngramCollectionDeleteService
	EngramCollectionAddItems EngramCollectionAddItemsService
	EngramCollectionRemove   EngramCollectionRemoveItemService
	PinEngramService         PinEngramService
	UnpinEngramService       UnpinEngramService
	PinDocumentService       PinDocumentService
	UnpinDocumentService     UnpinDocumentService
}

// CompatibilityService provides baseline MCP interop behavior while the full tool catalog migrates.
type CompatibilityService struct {
	serverVersion            string
	mcpToolPolicyVersion     string
	projectService           ProjectListService
	projectExport            ProjectExportService
	projectImport            ProjectImportService
	sessionService           SessionListService
	sessionGet               SessionGetService
	sessionCreate            SessionCreateService
	sessionContinue          SessionContinueService
	sessionSaveAsEngram      SessionSaveAsEngramService
	sessionDelete            SessionDeleteService
	sessionRestore           SessionRestoreService
	lifecyclePolicyUpdate    LifecyclePolicyUpdateService
	messageService           MessageListService
	messageSend              MessageSendService
	messageStream            MessageStreamService
	timelineService          TimelineListService
	pinnedEngramService      PinnedEngramListService
	pinnedDocumentService    PinnedDocumentListService
	projectDocumentService   ProjectDocumentListService
	engramList               EngramListService
	engramGet                EngramGetService
	engramQuery              EngramQueryService
	engramRehydrate          EngramRehydrateService
	engramCreate             EngramCreateService
	engramCreateConversation EngramCreateFromConversationService
	engramUpdate             EngramUpdateService
	engramMove               EngramMoveService
	engramDelete             EngramDeleteService
	engramRestore            EngramRestoreService
	engramCollectionList     EngramCollectionListService
	engramCollectionGet      EngramCollectionGetService
	engramCollectionCreate   EngramCollectionCreateService
	engramCollectionUpdate   EngramCollectionUpdateService
	engramCollectionDelete   EngramCollectionDeleteService
	engramCollectionAddItems EngramCollectionAddItemsService
	engramCollectionRemove   EngramCollectionRemoveItemService
	pinEngramService         PinEngramService
	unpinEngramService       UnpinEngramService
	pinDocumentService       PinDocumentService
	unpinDocumentService     UnpinDocumentService
}

// NewCompatibilityService builds a compatibility MCP service with stable initialize/tool-list behavior.
func NewCompatibilityService(serverVersion string) *CompatibilityService {
	return NewCompatibilityServiceWithDependencies(
		serverVersion,
		CompatibilityServiceDependencies{},
	)
}

// NewCompatibilityServiceWithDependencies builds a compatibility MCP service with dependency-backed tool dispatch.
func NewCompatibilityServiceWithDependencies(
	serverVersion string,
	dependencies CompatibilityServiceDependencies,
) *CompatibilityService {
	trimmed := strings.TrimSpace(serverVersion)
	if trimmed == "" {
		trimmed = "0.1.0"
	}
	policyVersion := strings.TrimSpace(dependencies.MCPToolPolicyVersion)
	if policyVersion == "" {
		policyVersion = governance.DefaultMCPToolPolicyVersion
	}
	return &CompatibilityService{
		serverVersion:            trimmed,
		mcpToolPolicyVersion:     policyVersion,
		projectService:           dependencies.ProjectService,
		projectExport:            dependencies.ProjectExport,
		projectImport:            dependencies.ProjectImport,
		sessionService:           dependencies.SessionService,
		sessionGet:               dependencies.SessionGet,
		sessionCreate:            dependencies.SessionCreate,
		sessionContinue:          dependencies.SessionContinue,
		sessionSaveAsEngram:      dependencies.SessionSaveAsEngram,
		sessionDelete:            dependencies.SessionDelete,
		sessionRestore:           dependencies.SessionRestore,
		lifecyclePolicyUpdate:    dependencies.LifecyclePolicyUpdate,
		messageService:           dependencies.MessageService,
		messageSend:              dependencies.MessageSend,
		messageStream:            dependencies.MessageStream,
		timelineService:          dependencies.TimelineService,
		pinnedEngramService:      dependencies.PinnedEngramService,
		pinnedDocumentService:    dependencies.PinnedDocumentService,
		projectDocumentService:   dependencies.ProjectDocumentService,
		engramList:               dependencies.EngramList,
		engramGet:                dependencies.EngramGet,
		engramQuery:              dependencies.EngramQuery,
		engramRehydrate:          dependencies.EngramRehydrate,
		engramCreate:             dependencies.EngramCreate,
		engramCreateConversation: dependencies.EngramCreateConversation,
		engramUpdate:             dependencies.EngramUpdate,
		engramMove:               dependencies.EngramMove,
		engramDelete:             dependencies.EngramDelete,
		engramRestore:            dependencies.EngramRestore,
		engramCollectionList:     dependencies.EngramCollectionList,
		engramCollectionGet:      dependencies.EngramCollectionGet,
		engramCollectionCreate:   dependencies.EngramCollectionCreate,
		engramCollectionUpdate:   dependencies.EngramCollectionUpdate,
		engramCollectionDelete:   dependencies.EngramCollectionDelete,
		engramCollectionAddItems: dependencies.EngramCollectionAddItems,
		engramCollectionRemove:   dependencies.EngramCollectionRemove,
		pinEngramService:         dependencies.PinEngramService,
		unpinEngramService:       dependencies.UnpinEngramService,
		pinDocumentService:       dependencies.PinDocumentService,
		unpinDocumentService:     dependencies.UnpinDocumentService,
	}
}

// HandleNotification accepts JSON-RPC notifications and intentionally no-ops.
func (service *CompatibilityService) HandleNotification(_ context.Context, _ JSONRPCRequest) error {
	return nil
}

// StreamCall emits MCP event frames plus a terminal response frame for the request.
func (service *CompatibilityService) StreamCall(ctx context.Context, request StreamCallRequest) <-chan Frame {
	frames := make(chan Frame, 1)
	go func() {
		defer close(frames)
		if service.emitStreamCallFrames(ctx, request, frames) {
			return
		}
		response := service.dispatch(ctx, request.Request, request.Actor, request.TokenAuth)
		select {
		case <-ctx.Done():
			return
		case frames <- response:
		}
	}()
	return frames
}

func (service *CompatibilityService) dispatch(
	ctx context.Context,
	request JSONRPCRequest,
	actor Actor,
	tokenAuth *models.MCPTokenAuthContext,
) Frame {
	if strings.TrimSpace(request.Method) == "" {
		return invalidRequestFrame(request.ID)
	}

	switch request.Method {
	case "initialize":
		return successFrame(request.ID, map[string]any{
			"protocolVersion": mcpProtocolVersion,
			"serverInfo": map[string]any{
				"name":    mcpServerName,
				"version": service.serverVersion,
			},
			"policy": map[string]any{
				"tool_policy_version": service.mcpToolPolicyVersion,
			},
			"capabilities": map[string]any{
				"tools": map[string]any{"listChanged": false},
			},
		})
	case "tools/list":
		return successFrame(request.ID, map[string]any{
			"tools": buildVisiblePublicToolCatalog(tokenAuth),
		})
	case "tools/call":
		return service.dispatchToolsCall(toolsCallInput{
			ctx:       ctx,
			requestID: request.ID,
			params:    request.Params,
			actor:     actor,
			tokenAuth: tokenAuth,
		})
	default:
		return service.dispatchDirectToolMethod(directToolCallInput{
			ctx:       ctx,
			requestID: request.ID,
			method:    request.Method,
			params:    request.Params,
			actor:     actor,
			tokenAuth: tokenAuth,
		})
	}
}

type toolsCallInput struct {
	ctx       context.Context
	requestID any
	params    map[string]any
	actor     Actor
	tokenAuth *models.MCPTokenAuthContext
}

func (service *CompatibilityService) dispatchToolsCall(input toolsCallInput) Frame {
	name, ok := requiredToolName(input.params)
	if !ok {
		return invalidParamsFrame(input.requestID, map[string]any{"missing": "name"})
	}
	arguments, ok := toolCallArguments(input.params)
	if !ok {
		return invalidParamsFrame(input.requestID, map[string]any{"invalid": "arguments"})
	}
	dottedName := toDottedToolName(name)
	if !toolExists(dottedName) {
		return methodNotFoundFrame(input.requestID, name)
	}
	return service.dispatchAuthorizedToolCall(
		authorizedToolDispatchInput{
			ctx:              input.ctx,
			requestID:        input.requestID,
			policyToolName:   dottedName,
			responseToolName: name,
			canonicalMethod:  canonicalToolName(dottedName),
			actor:            input.actor,
			params:           arguments,
			tokenAuth:        input.tokenAuth,
			asToolsCall:      true,
		},
	)
}

type directToolCallInput struct {
	ctx       context.Context
	requestID any
	method    string
	params    map[string]any
	actor     Actor
	tokenAuth *models.MCPTokenAuthContext
}

func (service *CompatibilityService) dispatchDirectToolMethod(input directToolCallInput) Frame {
	dottedMethod := toDottedToolName(input.method)
	if !toolExists(dottedMethod) {
		return methodNotFoundFrame(input.requestID, input.method)
	}
	return service.dispatchAuthorizedToolCall(
		authorizedToolDispatchInput{
			ctx:             input.ctx,
			requestID:       input.requestID,
			policyToolName:  dottedMethod,
			canonicalMethod: canonicalToolName(dottedMethod),
			actor:           input.actor,
			params:          input.params,
			tokenAuth:       input.tokenAuth,
			asToolsCall:     false,
		},
	)
}

type authorizedToolDispatchInput struct {
	ctx              context.Context
	requestID        any
	policyToolName   string
	responseToolName string
	canonicalMethod  string
	actor            Actor
	params           map[string]any
	tokenAuth        *models.MCPTokenAuthContext
	asToolsCall      bool
}

func (service *CompatibilityService) dispatchAuthorizedToolCall(
	input authorizedToolDispatchInput,
) Frame {
	if policyError := authorizeToolCall(input.policyToolName, input.tokenAuth); policyError != nil {
		return errorFrame(input.requestID, policyError.code, policyError.message, policyError.data)
	}
	normalizedParams, policyError := normalizeTokenToolParams(
		input.policyToolName,
		input.params,
		input.tokenAuth,
	)
	if policyError != nil {
		return errorFrame(input.requestID, policyError.code, policyError.message, policyError.data)
	}
	if policyError := service.enforceTokenProjectPolicy(
		tokenProjectPolicyRequest{
			ctx:       input.ctx,
			actor:     input.actor,
			toolName:  input.policyToolName,
			params:    normalizedParams,
			tokenAuth: input.tokenAuth,
		},
	); policyError != nil {
		return errorFrame(input.requestID, policyError.code, policyError.message, policyError.data)
	}
	payload, handled, dispatchError := service.dispatchImplementedTool(
		input.ctx,
		input.canonicalMethod,
		input.actor,
		normalizedParams,
	)
	if dispatchError != nil {
		return errorFrame(input.requestID, dispatchError.code, dispatchError.message, dispatchError.data)
	}
	if handled {
		return buildAuthorizedToolSuccessFrame(input, payload)
	}
	return errorFrame(
		input.requestID,
		-32000,
		"Tool not implemented",
		map[string]any{"method": input.canonicalMethod},
	)
}

func buildAuthorizedToolSuccessFrame(
	input authorizedToolDispatchInput,
	payload map[string]any,
) Frame {
	if !input.asToolsCall {
		return successFrame(input.requestID, payload)
	}
	return successFrame(
		input.requestID,
		buildToolCallSuccessResult(input.responseToolName, payload),
	)
}

func requiredToolName(params map[string]any) (string, bool) {
	rawName, ok := params["name"]
	if !ok {
		return "", false
	}
	name, ok := rawName.(string)
	if !ok {
		return "", false
	}
	name = strings.TrimSpace(name)
	return name, name != ""
}

func toolCallArguments(params map[string]any) (map[string]any, bool) {
	rawArguments, ok := params["arguments"]
	if !ok || rawArguments == nil {
		return map[string]any{}, true
	}
	arguments, ok := rawArguments.(map[string]any)
	return arguments, ok
}

func successFrame(id any, result map[string]any) Frame {
	return Frame{
		"jsonrpc": "2.0",
		"id":      id,
		"result":  result,
	}
}

func invalidRequestFrame(id any) Frame {
	return errorFrame(id, -32600, "Invalid Request", nil)
}

func invalidParamsFrame(id any, data map[string]any) Frame {
	return errorFrame(id, -32602, "Invalid params", data)
}

func methodNotFoundFrame(id any, method string) Frame {
	return errorFrame(id, -32601, "Method not found", map[string]any{"method": method})
}

func errorFrame(id any, code int, message string, data map[string]any) Frame {
	payload := map[string]any{
		"code":    code,
		"message": message,
	}
	if data != nil {
		payload["data"] = data
	}
	return Frame{
		"jsonrpc": "2.0",
		"id":      id,
		"error":   payload,
	}
}
