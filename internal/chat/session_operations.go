package chat

import (
	"context"

	"engram/internal/models"
	"engram/internal/repository"

	"github.com/google/uuid"
)

// ChatLifecyclePolicy captures effective lifecycle settings for a session.
type ChatLifecyclePolicy struct {
	AutosaveEnabled         bool                        `json:"autosave_enabled"`
	AutosaveStrategy        models.ChatAutosaveStrategy `json:"autosave_strategy"`
	AutosaveIntervalMinutes int                         `json:"autosave_interval_minutes"`
	AutosaveMinMessages     int                         `json:"autosave_min_messages"`
	RetentionDays           int                         `json:"retention_days"`
	RetentionMaxSnapshots   int                         `json:"retention_max_snapshots"`
}

// ChatLifecyclePolicyUpdateRequest captures mutable lifecycle policy fields.
type ChatLifecyclePolicyUpdateRequest struct {
	AutosaveEnabled         *bool                        `json:"autosave_enabled,omitempty"`
	AutosaveStrategy        *models.ChatAutosaveStrategy `json:"autosave_strategy,omitempty"`
	AutosaveIntervalMinutes *int                         `json:"autosave_interval_minutes,omitempty"`
	AutosaveMinMessages     *int                         `json:"autosave_min_messages,omitempty"`
	RetentionDays           *int                         `json:"retention_days,omitempty"`
	RetentionMaxSnapshots   *int                         `json:"retention_max_snapshots,omitempty"`
}

// SessionListRequest captures list-session filters.
type SessionListRequest struct {
	ActorUserID uuid.UUID
	ProjectID   *string
	Limit       int
	Offset      int
}

// SessionMessagesRequest captures list-messages filters.
type SessionMessagesRequest struct {
	ActorUserID uuid.UUID
	SessionID   uuid.UUID
	Limit       int
	Offset      int
}

// SessionPinEngramRequest captures pin/unpin engram request values.
type SessionPinEngramRequest struct {
	ActorUserID uuid.UUID
	SessionID   uuid.UUID
	EngramID    uuid.UUID
}

// SessionPinDocumentRequest captures pin/unpin document request values.
type SessionPinDocumentRequest struct {
	ActorUserID uuid.UUID
	SessionID   uuid.UUID
	DocumentID  uuid.UUID
}

type sessionPinOperation[Input any, Record any] struct {
	input               Input
	pin                 func(context.Context, repository.Queryer, Input) (*Record, error)
	unpin               func(context.Context, repository.Queryer, Input) (bool, error)
	notAccessibleDetail string
	notFoundDetail      string
}

type sessionPinAction[Input any, Record any] struct {
	pin                 func(context.Context, repository.Queryer, Input) (*Record, error)
	unpin               func(context.Context, repository.Queryer, Input) (bool, error)
	notAccessibleDetail string
	notFoundDetail      string
}

func (operation sessionPinOperation[Input, Record]) runPin(
	ctx context.Context,
	db repository.Queryer,
) (*Record, error) {
	return pinResource(
		func() (*Record, error) {
			return operation.pin(ctx, db, operation.input)
		},
		operation.notAccessibleDetail,
	)
}

func (operation sessionPinOperation[Input, Record]) runUnpin(
	ctx context.Context,
	db repository.Queryer,
) error {
	return unpinResource(
		func() (bool, error) {
			return operation.unpin(ctx, db, operation.input)
		},
		operation.notFoundDetail,
	)
}

type sessionOperationsDeps struct {
	ensureProjectExists func(ctx context.Context, db repository.Queryer, input repository.ProjectEnsureInput) (*models.ProjectRecord, error)
	createChatSession   func(ctx context.Context, db repository.Queryer, input repository.ChatSessionCreateInput) (*models.ChatSessionRecord, error)
	listChatSessions    func(ctx context.Context, db repository.Queryer, input repository.ChatSessionListInput) ([]models.ChatSessionRecord, error)
	getChatSession      func(ctx context.Context, db repository.Queryer, input repository.ChatSessionGetInput) (*models.ChatSessionRecord, error)
	updateChatSession   func(ctx context.Context, db repository.Queryer, input repository.ChatSessionUpdateInput) (*models.ChatSessionRecord, error)
	listChatMessages    func(ctx context.Context, db repository.Queryer, input repository.ChatMessageListInput) ([]models.ChatMessageRecord, error)
	listPinnedEngrams   func(ctx context.Context, db repository.Queryer, input repository.ChatPinnedListInput) ([]models.EngramSummary, error)
	listPinnedDocuments func(ctx context.Context, db repository.Queryer, input repository.ChatPinnedListInput) ([]models.PinnedDocumentRecord, error)
	pinEngram           func(ctx context.Context, db repository.Queryer, input repository.ChatPinEngramInput) (*models.PinnedEngramRecord, error)
	pinDocument         func(ctx context.Context, db repository.Queryer, input repository.ChatPinDocumentInput) (*models.PinnedDocumentRecord, error)
	unpinEngram         func(ctx context.Context, db repository.Queryer, input repository.ChatPinEngramInput) (bool, error)
	unpinDocument       func(ctx context.Context, db repository.Queryer, input repository.ChatPinDocumentInput) (bool, error)
}

func defaultSessionOperationsDeps() sessionOperationsDeps {
	return sessionOperationsDeps{
		ensureProjectExists: repository.EnsureProjectExists,
		createChatSession:   repository.CreateChatSession,
		listChatSessions:    repository.ListChatSessions,
		getChatSession:      repository.GetChatSession,
		updateChatSession:   repository.UpdateChatSession,
		listChatMessages:    repository.ListChatMessages,
		listPinnedEngrams:   repository.ListPinnedEngramSummaries,
		listPinnedDocuments: repository.ListPinnedDocuments,
		pinEngram:           repository.PinEngramToSession,
		pinDocument:         repository.PinDocumentToSession,
		unpinEngram:         repository.UnpinEngramFromSession,
		unpinDocument:       repository.UnpinDocumentFromSession,
	}
}

// SessionOperationsService ports chat session operations from session_operations.py.
type SessionOperationsService struct {
	db   repository.Queryer
	deps sessionOperationsDeps
}

// NewSessionOperationsService constructs a session operations service.
func NewSessionOperationsService(db repository.Queryer) *SessionOperationsService {
	return &SessionOperationsService{db: db, deps: defaultSessionOperationsDeps()}
}

// NormalizeSessionCreatePayload applies backward-compatible autosave policy normalization.
func NormalizeSessionCreatePayload(payload models.ChatSessionCreateRequest) models.ChatSessionCreateRequest {
	autosaveEnabled, autosaveStrategy := NormalizeAutosavePolicy(payload.AutosaveEnabled, payload.AutosaveStrategy)
	payload.AutosaveEnabled = autosaveEnabled
	payload.AutosaveStrategy = autosaveStrategy
	return payload
}

// NormalizeSessionUpdatePayload applies partial-update autosave compatibility semantics.
func NormalizeSessionUpdatePayload(payload models.ChatSessionUpdateRequest) models.ChatSessionUpdateRequest {
	if payload.AutosaveEnabled == nil && payload.AutosaveStrategy == nil {
		return payload
	}
	autosaveEnabled, autosaveStrategy := resolveAutosaveUpdate(payload.AutosaveEnabled, payload.AutosaveStrategy)
	normalizedEnabled, normalizedStrategy := NormalizeAutosavePolicy(autosaveEnabled, autosaveStrategy)
	payload.AutosaveEnabled = boolRef(normalizedEnabled)
	payload.AutosaveStrategy = autosaveStrategyRef(normalizedStrategy)
	return payload
}

// BuildLifecyclePolicy maps a session record into lifecycle policy output.
func BuildLifecyclePolicy(session models.ChatSessionRecord) ChatLifecyclePolicy {
	return ChatLifecyclePolicy{
		AutosaveEnabled:         session.AutosaveEnabled,
		AutosaveStrategy:        session.AutosaveStrategy,
		AutosaveIntervalMinutes: session.AutosaveIntervalMinutes,
		AutosaveMinMessages:     session.AutosaveMinMessages,
		RetentionDays:           session.RetentionDays,
		RetentionMaxSnapshots:   session.RetentionMaxSnapshots,
	}
}

// CreateSession creates a normalized session after ensuring project accessibility.
func (service *SessionOperationsService) CreateSession(
	ctx context.Context,
	actorUserID uuid.UUID,
	payload models.ChatSessionCreateRequest,
) (*models.ChatSessionRecord, error) {
	normalizedPayload := NormalizeSessionCreatePayload(payload)
	_, err := service.deps.ensureProjectExists(
		ctx,
		service.db,
		repository.ProjectEnsureInput{ProjectID: normalizedPayload.ProjectID, OwnerUserID: actorUserID},
	)
	if err != nil {
		return nil, err
	}
	return service.deps.createChatSession(
		ctx,
		service.db,
		repository.ChatSessionCreateInput{OwnerUserID: actorUserID, Payload: normalizedPayload},
	)
}

// ListSessions returns sessions visible to the actor.
func (service *SessionOperationsService) ListSessions(
	ctx context.Context,
	request SessionListRequest,
) ([]models.ChatSessionRecord, error) {
	return service.deps.listChatSessions(
		ctx,
		service.db,
		repository.ChatSessionListInput{
			ActorUserID: request.ActorUserID,
			ProjectID:   request.ProjectID,
			Limit:       request.Limit,
			Offset:      request.Offset,
		},
	)
}

// GetSession returns a session or a typed not-found error.
func (service *SessionOperationsService) GetSession(
	ctx context.Context,
	actorUserID uuid.UUID,
	sessionID uuid.UUID,
) (*models.ChatSessionRecord, error) {
	session, err := service.deps.getChatSession(
		ctx,
		service.db,
		repository.ChatSessionGetInput{SessionID: sessionID, ActorUserID: actorUserID},
	)
	if err != nil {
		return nil, err
	}
	if session == nil {
		return nil, NewChatSessionNotFoundError("")
	}
	return session, nil
}

// UpdateSession updates mutable session fields and enforces not-found behavior.
func (service *SessionOperationsService) UpdateSession(
	ctx context.Context,
	actorUserID uuid.UUID,
	sessionID uuid.UUID,
	payload models.ChatSessionUpdateRequest,
) (*models.ChatSessionRecord, error) {
	normalizedPayload := NormalizeSessionUpdatePayload(payload)
	updated, err := service.deps.updateChatSession(
		ctx,
		service.db,
		repository.ChatSessionUpdateInput{SessionID: sessionID, ActorUserID: actorUserID, Payload: normalizedPayload},
	)
	if err != nil {
		return nil, err
	}
	if updated == nil {
		return nil, NewChatSessionNotFoundError("")
	}
	return updated, nil
}

// GetLifecyclePolicy returns effective lifecycle policy for a session.
func (service *SessionOperationsService) GetLifecyclePolicy(
	ctx context.Context,
	actorUserID uuid.UUID,
	sessionID uuid.UUID,
) (ChatLifecyclePolicy, error) {
	session, err := service.GetSession(ctx, actorUserID, sessionID)
	if err != nil {
		return ChatLifecyclePolicy{}, err
	}
	return BuildLifecyclePolicy(*session), nil
}

// UpdateLifecyclePolicy updates lifecycle policy fields and returns the effective policy.
func (service *SessionOperationsService) UpdateLifecyclePolicy(
	ctx context.Context,
	actorUserID uuid.UUID,
	sessionID uuid.UUID,
	payload ChatLifecyclePolicyUpdateRequest,
) (ChatLifecyclePolicy, error) {
	if payload.isEmpty() {
		return service.GetLifecyclePolicy(ctx, actorUserID, sessionID)
	}
	updated, err := service.UpdateSession(
		ctx,
		actorUserID,
		sessionID,
		models.ChatSessionUpdateRequest{
			AutosaveEnabled:         payload.AutosaveEnabled,
			AutosaveStrategy:        payload.AutosaveStrategy,
			AutosaveIntervalMinutes: payload.AutosaveIntervalMinutes,
			AutosaveMinMessages:     payload.AutosaveMinMessages,
			RetentionDays:           payload.RetentionDays,
			RetentionMaxSnapshots:   payload.RetentionMaxSnapshots,
		},
	)
	if err != nil {
		return ChatLifecyclePolicy{}, err
	}
	return BuildLifecyclePolicy(*updated), nil
}

// ListMessages returns visible chat messages for a session.
func (service *SessionOperationsService) ListMessages(
	ctx context.Context,
	request SessionMessagesRequest,
) ([]models.ChatMessageRecord, error) {
	if _, err := service.GetSession(ctx, request.ActorUserID, request.SessionID); err != nil {
		return nil, err
	}
	return service.deps.listChatMessages(
		ctx,
		service.db,
		repository.ChatMessageListInput{
			SessionID:   request.SessionID,
			ActorUserID: request.ActorUserID,
			Limit:       request.Limit,
			Offset:      request.Offset,
		},
	)
}

// ListPinnedEngrams returns pinned engram summaries for a visible session.
func (service *SessionOperationsService) ListPinnedEngrams(
	ctx context.Context,
	actorUserID uuid.UUID,
	sessionID uuid.UUID,
) ([]models.EngramSummary, error) {
	return listPinnedByInput(ctx, service, actorUserID, sessionID, service.deps.listPinnedEngrams)
}

// ListPinnedDocuments returns pinned document rows for a visible session.
func (service *SessionOperationsService) ListPinnedDocuments(
	ctx context.Context,
	actorUserID uuid.UUID,
	sessionID uuid.UUID,
) ([]models.PinnedDocumentRecord, error) {
	return listPinnedByInput(ctx, service, actorUserID, sessionID, service.deps.listPinnedDocuments)
}

// PinEngram pins an engram to a session when both resources are accessible.
func (service *SessionOperationsService) PinEngram(
	ctx context.Context,
	request SessionPinEngramRequest,
) (*models.PinnedEngramRecord, error) {
	return runPinAction(ctx, service, request.pinEngramInput(), service.engramPinAction())
}

// PinDocument pins a document to a session when both resources are accessible.
func (service *SessionOperationsService) PinDocument(
	ctx context.Context,
	request SessionPinDocumentRequest,
) (*models.PinnedDocumentRecord, error) {
	return runPinAction(ctx, service, request.pinDocumentInput(), service.documentPinAction())
}

// UnpinEngram removes a pinned engram or returns typed not-found behavior.
func (service *SessionOperationsService) UnpinEngram(
	ctx context.Context,
	request SessionPinEngramRequest,
) error {
	return runUnpinAction(ctx, service, request.pinEngramInput(), service.engramPinAction())
}

// UnpinDocument removes a pinned document or returns typed not-found behavior.
func (service *SessionOperationsService) UnpinDocument(
	ctx context.Context,
	request SessionPinDocumentRequest,
) error {
	return runUnpinAction(ctx, service, request.pinDocumentInput(), service.documentPinAction())
}

func resolveAutosaveUpdate(
	autosaveEnabled *bool,
	autosaveStrategy *models.ChatAutosaveStrategy,
) (bool, models.ChatAutosaveStrategy) {
	resolvedEnabled := deriveAutosaveEnabled(autosaveEnabled, autosaveStrategy)
	resolvedStrategy := deriveAutosaveStrategy(autosaveStrategy, resolvedEnabled)
	return resolvedEnabled, resolvedStrategy
}

func (payload ChatLifecyclePolicyUpdateRequest) isEmpty() bool {
	switch {
	case payload.AutosaveEnabled != nil:
		return false
	case payload.AutosaveStrategy != nil:
		return false
	case payload.AutosaveIntervalMinutes != nil:
		return false
	case payload.AutosaveMinMessages != nil:
		return false
	case payload.RetentionDays != nil:
		return false
	case payload.RetentionMaxSnapshots != nil:
		return false
	default:
		return true
	}
}

func deriveAutosaveEnabled(
	autosaveEnabled *bool,
	autosaveStrategy *models.ChatAutosaveStrategy,
) bool {
	if autosaveEnabled != nil {
		return *autosaveEnabled
	}
	if autosaveStrategy != nil {
		return *autosaveStrategy != models.ChatAutosaveStrategyOff
	}
	return false
}

func deriveAutosaveStrategy(
	autosaveStrategy *models.ChatAutosaveStrategy,
	autosaveEnabled bool,
) models.ChatAutosaveStrategy {
	if autosaveStrategy != nil {
		return *autosaveStrategy
	}
	if autosaveEnabled {
		return models.ChatAutosaveStrategyInterval
	}
	return models.ChatAutosaveStrategyOff
}

func (request SessionPinEngramRequest) pinEngramInput() repository.ChatPinEngramInput {
	return repository.ChatPinEngramInput{
		SessionID:   request.SessionID,
		EngramID:    request.EngramID,
		ActorUserID: request.ActorUserID,
	}
}

func (request SessionPinDocumentRequest) pinDocumentInput() repository.ChatPinDocumentInput {
	return repository.ChatPinDocumentInput{
		SessionID:   request.SessionID,
		DocumentID:  request.DocumentID,
		ActorUserID: request.ActorUserID,
	}
}

func pinResource[T any](
	execute func() (*T, error),
	notAccessibleDetail string,
) (*T, error) {
	record, err := execute()
	if err != nil {
		return nil, err
	}
	if record == nil {
		return nil, NewChatValidationError(notAccessibleDetail)
	}
	return record, nil
}

func unpinResource(execute func() (bool, error), notFoundDetail string) error {
	removed, err := execute()
	if err != nil {
		return err
	}
	if !removed {
		return NewChatSessionNotFoundError(notFoundDetail)
	}
	return nil
}

func listPinnedByInput[Record any](
	ctx context.Context,
	service *SessionOperationsService,
	actorUserID uuid.UUID,
	sessionID uuid.UUID,
	listResources func(context.Context, repository.Queryer, repository.ChatPinnedListInput) ([]Record, error),
) ([]Record, error) {
	if _, err := service.GetSession(ctx, actorUserID, sessionID); err != nil {
		return nil, err
	}
	return listResources(
		ctx,
		service.db,
		repository.ChatPinnedListInput{SessionID: sessionID, ActorUserID: actorUserID},
	)
}

func newSessionPinOperation[Input any, Record any](
	input Input,
	pin func(context.Context, repository.Queryer, Input) (*Record, error),
	unpin func(context.Context, repository.Queryer, Input) (bool, error),
	notAccessibleDetail string,
	notFoundDetail string,
) sessionPinOperation[Input, Record] {
	return sessionPinOperation[Input, Record]{
		input:               input,
		pin:                 pin,
		unpin:               unpin,
		notAccessibleDetail: notAccessibleDetail,
		notFoundDetail:      notFoundDetail,
	}
}

func runPinAction[Input any, Record any](
	ctx context.Context,
	service *SessionOperationsService,
	input Input,
	action sessionPinAction[Input, Record],
) (*Record, error) {
	return newSessionPinOperation(
		input,
		action.pin,
		action.unpin,
		action.notAccessibleDetail,
		action.notFoundDetail,
	).runPin(ctx, service.db)
}

func runUnpinAction[Input any, Record any](
	ctx context.Context,
	service *SessionOperationsService,
	input Input,
	action sessionPinAction[Input, Record],
) error {
	return newSessionPinOperation(
		input,
		action.pin,
		action.unpin,
		action.notAccessibleDetail,
		action.notFoundDetail,
	).runUnpin(ctx, service.db)
}

func (service *SessionOperationsService) engramPinAction() sessionPinAction[repository.ChatPinEngramInput, models.PinnedEngramRecord] {
	return sessionPinAction[repository.ChatPinEngramInput, models.PinnedEngramRecord]{
		pin:                 service.deps.pinEngram,
		unpin:               service.deps.unpinEngram,
		notAccessibleDetail: "Session or engram is not accessible for pinning",
		notFoundDetail:      "Pinned engram not found for session",
	}
}

func (service *SessionOperationsService) documentPinAction() sessionPinAction[repository.ChatPinDocumentInput, models.PinnedDocumentRecord] {
	return sessionPinAction[repository.ChatPinDocumentInput, models.PinnedDocumentRecord]{
		pin:                 service.deps.pinDocument,
		unpin:               service.deps.unpinDocument,
		notAccessibleDetail: "Session or document is not accessible for pinning",
		notFoundDetail:      "Pinned document not found for session",
	}
}

func boolRef(value bool) *bool {
	copy := value
	return &copy
}

func autosaveStrategyRef(value models.ChatAutosaveStrategy) *models.ChatAutosaveStrategy {
	copy := value
	return &copy
}
