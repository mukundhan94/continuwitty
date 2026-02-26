package mcp

import (
	"context"
	"strings"

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

// SessionCreateRequest captures compatibility-level session create inputs.
type SessionCreateRequest struct {
	ActorUserID uuid.UUID
	Payload     models.ChatSessionCreateRequest
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

// CompatibilityServiceDependencies captures optional service dependencies for compatibility dispatch.
type CompatibilityServiceDependencies struct {
	ProjectService         ProjectListService
	SessionService         SessionListService
	SessionGet             SessionGetService
	SessionCreate          SessionCreateService
	LifecyclePolicyUpdate  LifecyclePolicyUpdateService
	MessageService         MessageListService
	TimelineService        TimelineListService
	PinnedEngramService    PinnedEngramListService
	PinnedDocumentService  PinnedDocumentListService
	ProjectDocumentService ProjectDocumentListService
	PinEngramService       PinEngramService
	UnpinEngramService     UnpinEngramService
	PinDocumentService     PinDocumentService
	UnpinDocumentService   UnpinDocumentService
}

// CompatibilityService provides baseline MCP interop behavior while the full tool catalog migrates.
type CompatibilityService struct {
	serverVersion          string
	projectService         ProjectListService
	sessionService         SessionListService
	sessionGet             SessionGetService
	sessionCreate          SessionCreateService
	lifecyclePolicyUpdate  LifecyclePolicyUpdateService
	messageService         MessageListService
	timelineService        TimelineListService
	pinnedEngramService    PinnedEngramListService
	pinnedDocumentService  PinnedDocumentListService
	projectDocumentService ProjectDocumentListService
	pinEngramService       PinEngramService
	unpinEngramService     UnpinEngramService
	pinDocumentService     PinDocumentService
	unpinDocumentService   UnpinDocumentService
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
	return &CompatibilityService{
		serverVersion:          trimmed,
		projectService:         dependencies.ProjectService,
		sessionService:         dependencies.SessionService,
		sessionGet:             dependencies.SessionGet,
		sessionCreate:          dependencies.SessionCreate,
		lifecyclePolicyUpdate:  dependencies.LifecyclePolicyUpdate,
		messageService:         dependencies.MessageService,
		timelineService:        dependencies.TimelineService,
		pinnedEngramService:    dependencies.PinnedEngramService,
		pinnedDocumentService:  dependencies.PinnedDocumentService,
		projectDocumentService: dependencies.ProjectDocumentService,
		pinEngramService:       dependencies.PinEngramService,
		unpinEngramService:     dependencies.UnpinEngramService,
		pinDocumentService:     dependencies.PinDocumentService,
		unpinDocumentService:   dependencies.UnpinDocumentService,
	}
}

// HandleNotification accepts JSON-RPC notifications and intentionally no-ops.
func (service *CompatibilityService) HandleNotification(_ context.Context, _ JSONRPCRequest) error {
	return nil
}

// StreamCall emits a single terminal response frame for the request.
func (service *CompatibilityService) StreamCall(ctx context.Context, request StreamCallRequest) <-chan Frame {
	frames := make(chan Frame, 1)
	go func() {
		defer close(frames)
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
	if policyError := authorizeToolCall(dottedName, input.tokenAuth); policyError != nil {
		return errorFrame(input.requestID, policyError.code, policyError.message, policyError.data)
	}
	payload, handled, dispatchError := service.dispatchImplementedTool(
		input.ctx,
		canonicalToolName(dottedName),
		input.actor,
		arguments,
	)
	if dispatchError != nil {
		return errorFrame(input.requestID, dispatchError.code, dispatchError.message, dispatchError.data)
	}
	if handled {
		return successFrame(input.requestID, buildToolCallSuccessResult(name, payload))
	}
	return errorFrame(
		input.requestID,
		-32000,
		"Tool not implemented",
		map[string]any{"method": canonicalToolName(dottedName)},
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
	if policyError := authorizeToolCall(dottedMethod, input.tokenAuth); policyError != nil {
		return errorFrame(input.requestID, policyError.code, policyError.message, policyError.data)
	}
	payload, handled, dispatchError := service.dispatchImplementedTool(
		input.ctx,
		canonicalToolName(dottedMethod),
		input.actor,
		input.params,
	)
	if dispatchError != nil {
		return errorFrame(input.requestID, dispatchError.code, dispatchError.message, dispatchError.data)
	}
	if handled {
		return successFrame(input.requestID, payload)
	}
	return errorFrame(
		input.requestID,
		-32000,
		"Tool not implemented",
		map[string]any{"method": canonicalToolName(dottedMethod)},
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
