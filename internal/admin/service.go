package admin

import (
	"context"
	"errors"
	"strings"
	"time"

	"engram/internal/models"
	"engram/internal/repository"

	"github.com/google/uuid"
)

var (
	// ErrSessionNotFound indicates a requested session does not exist.
	ErrSessionNotFound = errors.New("session not found")
	// ErrEngramNotFound indicates a requested engram does not exist.
	ErrEngramNotFound = errors.New("engram not found")
	// ErrCollectionNotFound indicates a requested collection does not exist.
	ErrCollectionNotFound = errors.New("collection not found")
	// ErrEngramStale indicates expected update time no longer matches current row state.
	ErrEngramStale = errors.New("engram was updated by another operation")
	// ErrCollectionStale indicates expected update time no longer matches current row state.
	ErrCollectionStale = errors.New("collection was updated by another operation")
	// ErrProjectResolverNotConfigured indicates a write operation requiring project resolution cannot proceed.
	ErrProjectResolverNotConfigured = errors.New("project resolver is not configured")
	// ErrProjectIDRequired indicates that a write operation is missing project context.
	ErrProjectIDRequired = errors.New("project_id is required")
	// ErrConsolidationMinGroupSizeInvalid indicates invalid consolidation minimum group-size input.
	ErrConsolidationMinGroupSizeInvalid = errors.New("min_group_size must be at least 2")
	// ErrConsolidationSuggestionNotFound indicates a requested consolidation suggestion does not exist.
	ErrConsolidationSuggestionNotFound = errors.New("consolidation suggestion not found")
	// ErrConsolidationSuggestionActionInvalid indicates invalid consolidation action status.
	ErrConsolidationSuggestionActionInvalid = errors.New("status must be merged or rejected")
	// ErrContradictionAlertNotFound indicates a requested contradiction alert does not exist.
	ErrContradictionAlertNotFound = errors.New("contradiction alert not found")
	// ErrContradictionAlertResolveStatusInvalid indicates invalid contradiction alert resolve status.
	ErrContradictionAlertResolveStatusInvalid = errors.New("status must be resolved or dismissed")
	// ErrMemoryCurationSuggestionNotFound indicates a requested memory curation suggestion does not exist.
	ErrMemoryCurationSuggestionNotFound = errors.New("memory curation suggestion not found")
	// ErrMemoryCurationSuggestionActionInvalid indicates invalid memory curation suggestion action status.
	ErrMemoryCurationSuggestionActionInvalid = errors.New("status must be accepted, rejected, or applied")
	// ErrMemoryCurationSuggestionPayloadInvalid indicates apply action cannot parse required payload fields.
	ErrMemoryCurationSuggestionPayloadInvalid = errors.New("memory curation payload is invalid for apply action")
	// ErrMemoryCurationSuggestionApplyUnsupported indicates apply action cannot execute payload-specific workflow.
	ErrMemoryCurationSuggestionApplyUnsupported = errors.New("memory curation apply action is unsupported for suggestion payload")
)

// MemoryAdminListRequest captures shared admin list filters.
type MemoryAdminListRequest struct {
	ProjectID      *string    `json:"project_id,omitempty"`
	OwnerUserID    *uuid.UUID `json:"owner_user_id,omitempty"`
	IncludeDeleted bool       `json:"include_deleted"`
	Limit          int        `json:"limit"`
	Offset         int        `json:"offset"`
}

// MemoryAdminEngramListRequest captures admin engram list filters.
type MemoryAdminEngramListRequest struct {
	MemoryAdminListRequest
	SessionID *uuid.UUID `json:"session_id,omitempty"`
	QueryText *string    `json:"query_text,omitempty"`
}

// SessionDeleteRequest captures session delete settings.
type SessionDeleteRequest struct {
	DeleteLinkedEngrams bool    `json:"delete_linked_engrams"`
	Reason              *string `json:"reason,omitempty"`
}

// SessionDeleteResponse captures session delete results.
type SessionDeleteResponse struct {
	SessionID            uuid.UUID `json:"session_id"`
	Deleted              bool      `json:"deleted"`
	LinkedEngramsDeleted int       `json:"linked_engrams_deleted"`
}

// SessionRestoreResponse captures session restore results.
type SessionRestoreResponse struct {
	SessionID uuid.UUID `json:"session_id"`
	Restored  bool      `json:"restored"`
}

// EngramUpdateRequest captures mutable admin engram fields.
type EngramUpdateRequest struct {
	ExpectedUpdatedAt       *time.Time                       `json:"expected_updated_at,omitempty"`
	Title                   *string                          `json:"title,omitempty"`
	Abstract                *string                          `json:"abstract,omitempty"`
	DetailedSummaryMarkdown *string                          `json:"detailed_summary_markdown,omitempty"`
	Tags                    *[]string                        `json:"tags,omitempty"`
	Keywords                *[]string                        `json:"keywords,omitempty"`
	VisibilityScope         *models.VisibilityScope          `json:"visibility_scope,omitempty"`
	Sources                 *[]models.AdminEngramSourceInput `json:"sources,omitempty"`
}

// EngramMoveRequest captures admin engram move settings.
type EngramMoveRequest struct {
	TargetProjectID   string     `json:"target_project_id"`
	Reason            *string    `json:"reason,omitempty"`
	ExpectedUpdatedAt *time.Time `json:"expected_updated_at,omitempty"`
}

// EngramDeleteRequest captures delete reason for an engram.
type EngramDeleteRequest struct {
	Reason *string `json:"reason,omitempty"`
}

// EngramDeleteResponse captures engram delete results.
type EngramDeleteResponse struct {
	EngramID uuid.UUID `json:"engram_id"`
	Deleted  bool      `json:"deleted"`
}

// EngramRestoreResponse captures engram restore results.
type EngramRestoreResponse struct {
	EngramID uuid.UUID `json:"engram_id"`
	Restored bool      `json:"restored"`
}

// EngramFreshnessRefreshRequest captures freshness maintenance refresh options.
type EngramFreshnessRefreshRequest struct {
	ProjectID    *string  `json:"project_id,omitempty"`
	HalfLifeDays *float64 `json:"half_life_days,omitempty"`
}

// EngramFreshnessRefreshResponse captures freshness maintenance refresh results.
type EngramFreshnessRefreshResponse struct {
	ProjectID     *string   `json:"project_id,omitempty"`
	HalfLifeDays  float64   `json:"half_life_days"`
	ReferenceTime time.Time `json:"reference_time"`
	UpdatedCount  int       `json:"updated_count"`
}

// EngramConsolidationSuggestionRefreshRequest captures refresh options for consolidation suggestions.
type EngramConsolidationSuggestionRefreshRequest struct {
	ProjectID    *string `json:"project_id,omitempty"`
	MinGroupSize *int    `json:"min_group_size,omitempty"`
}

// EngramConsolidationSuggestionRefreshResponse captures refresh results for consolidation suggestions.
type EngramConsolidationSuggestionRefreshResponse struct {
	ProjectID    *string   `json:"project_id,omitempty"`
	MinGroupSize int       `json:"min_group_size"`
	SuggestedAt  time.Time `json:"suggested_at"`
	UpdatedCount int       `json:"updated_count"`
}

// EngramConsolidationSuggestionListRequest captures list filters for consolidation suggestions.
type EngramConsolidationSuggestionListRequest struct {
	ProjectID *string                               `json:"project_id,omitempty"`
	Status    *models.ConsolidationSuggestionStatus `json:"status,omitempty"`
	Limit     int                                   `json:"limit"`
	Offset    int                                   `json:"offset"`
}

// EngramConsolidationSuggestionActionRequest captures action payload for a suggestion.
type EngramConsolidationSuggestionActionRequest struct {
	ProjectID *string                              `json:"project_id,omitempty"`
	Status    models.ConsolidationSuggestionStatus `json:"status"`
}

// EngramContradictionAlertRefreshRequest captures refresh options for contradiction alerts.
type EngramContradictionAlertRefreshRequest struct {
	ProjectID *string `json:"project_id,omitempty"`
}

// EngramContradictionAlertRefreshResponse captures refresh results for contradiction alerts.
type EngramContradictionAlertRefreshResponse struct {
	ProjectID    *string   `json:"project_id,omitempty"`
	DetectedAt   time.Time `json:"detected_at"`
	UpdatedCount int       `json:"updated_count"`
}

// EngramContradictionAlertListRequest captures list filters for contradiction alerts.
type EngramContradictionAlertListRequest struct {
	ProjectID *string                          `json:"project_id,omitempty"`
	Status    *models.ContradictionAlertStatus `json:"status,omitempty"`
	Limit     int                              `json:"limit"`
	Offset    int                              `json:"offset"`
}

// EngramContradictionAlertResolveRequest captures resolve payload for a contradiction alert.
type EngramContradictionAlertResolveRequest struct {
	ProjectID *string                         `json:"project_id,omitempty"`
	Status    models.ContradictionAlertStatus `json:"status"`
}

// MemoryCurationSuggestionListRequest captures list filters for memory curation suggestions.
type MemoryCurationSuggestionListRequest struct {
	ProjectID      *string                                `json:"project_id,omitempty"`
	SessionID      *uuid.UUID                             `json:"session_id,omitempty"`
	SuggestionType *models.MemoryCurationSuggestionType   `json:"suggestion_type,omitempty"`
	Status         *models.MemoryCurationSuggestionStatus `json:"status,omitempty"`
	Limit          int                                    `json:"limit"`
	Offset         int                                    `json:"offset"`
}

// MemoryCurationSuggestionActionRequest captures action payload for a memory curation suggestion.
type MemoryCurationSuggestionActionRequest struct {
	ProjectID *string                               `json:"project_id,omitempty"`
	Status    models.MemoryCurationSuggestionStatus `json:"status"`
}

// CollectionCreateRequest captures collection create payload values.
type CollectionCreateRequest struct {
	ProjectID   string `json:"project_id"`
	Name        string `json:"name"`
	Description string `json:"description"`
}

// CollectionUpdateRequest captures mutable collection fields.
type CollectionUpdateRequest struct {
	ExpectedUpdatedAt *time.Time `json:"expected_updated_at,omitempty"`
	Name              *string    `json:"name,omitempty"`
	Description       *string    `json:"description,omitempty"`
}

// CollectionDeleteRequest captures collection delete settings.
type CollectionDeleteRequest struct {
	Reason *string `json:"reason,omitempty"`
}

// CollectionItemsUpdateRequest captures collection item update payload.
type CollectionItemsUpdateRequest struct {
	EngramIDs []uuid.UUID `json:"engram_ids"`
}

// CollectionDeleteResponse captures collection delete results.
type CollectionDeleteResponse struct {
	Deleted bool `json:"deleted"`
}

// CollectionItemsAddResponse captures add-item operation results.
type CollectionItemsAddResponse struct {
	Added int `json:"added"`
}

// CollectionItemRemoveResponse captures remove-item operation results.
type CollectionItemRemoveResponse struct {
	Removed bool `json:"removed"`
}

// WriteActor captures actor identity used for write authorization.
type WriteActor struct {
	UserID uuid.UUID
	Role   string
}

// ResolveProjectWriteInput captures project-resolution parameters for write operations.
type ResolveProjectWriteInput struct {
	ActorUserID uuid.UUID
	ActorRole   string
	ProjectID   string
}

// ResolveProjectWriteResult captures the resolved project id for write operations.
type ResolveProjectWriteResult struct {
	ProjectID          string
	UsedDefaultProject bool
}

// ProjectResolver resolves and authorizes target project ids for writes.
type ProjectResolver interface {
	ResolveProjectIDForWrite(ctx context.Context, input ResolveProjectWriteInput) (ResolveProjectWriteResult, error)
}

type serviceDeps struct {
	listAdminSessions       func(ctx context.Context, db repository.Queryer, input repository.AdminSessionListInput) ([]models.AdminChatSessionRecord, error)
	getAdminSession         func(ctx context.Context, db repository.Queryer, sessionID uuid.UUID, includeDeleted bool) (*models.AdminChatSessionRecord, error)
	softDeleteSession       func(ctx context.Context, db repository.Queryer, input repository.AdminSessionSoftDeleteInput) (*models.AdminChatSessionRecord, error)
	restoreSession          func(ctx context.Context, db repository.Queryer, sessionID uuid.UUID) (bool, error)
	softDeleteLinkedEngrams func(ctx context.Context, db repository.Queryer, input repository.SoftDeleteLinkedEngramsInput) (int, error)

	listAdminEngrams       func(ctx context.Context, db repository.Queryer, input repository.AdminEngramListInput) ([]models.AdminEngramRecord, error)
	getAdminEngram         func(ctx context.Context, db repository.Queryer, engramID uuid.UUID, includeDeleted bool) (*models.AdminEngramRecord, error)
	updateAdminEngram      func(ctx context.Context, db repository.Queryer, input repository.AdminEngramUpdateInput) (*models.AdminEngramRecord, error)
	moveAdminEngramProject func(ctx context.Context, db repository.Queryer, input repository.AdminEngramMoveProjectInput) (*models.AdminEngramRecord, error)
	softDeleteEngram       func(ctx context.Context, db repository.Queryer, input repository.AdminEngramSoftDeleteInput) (bool, error)
	restoreEngram          func(ctx context.Context, db repository.Queryer, engramID uuid.UUID) (bool, error)
	refreshEngramFreshness func(
		ctx context.Context,
		db repository.Queryer,
		input repository.EngramFreshnessRefreshInput,
	) (repository.EngramFreshnessRefreshInput, error)
	refreshConsolidationSuggestions func(
		ctx context.Context,
		db repository.Queryer,
		input repository.ConsolidationSuggestionRefreshInput,
	) (repository.ConsolidationSuggestionRefreshInput, error)
	listConsolidationSuggestions func(
		ctx context.Context,
		db repository.Queryer,
		input repository.ConsolidationSuggestionListInput,
	) ([]models.EngramConsolidationSuggestion, error)
	applyConsolidationSuggestionAction func(
		ctx context.Context,
		db repository.Queryer,
		input repository.ConsolidationSuggestionActionInput,
	) (*models.EngramConsolidationSuggestion, error)
	refreshContradictionAlerts func(
		ctx context.Context,
		db repository.Queryer,
		input repository.ContradictionAlertRefreshInput,
	) (repository.ContradictionAlertRefreshInput, error)
	listContradictionAlerts func(
		ctx context.Context,
		db repository.Queryer,
		input repository.ContradictionAlertListInput,
	) ([]models.EngramContradictionAlert, error)
	resolveContradictionAlert func(
		ctx context.Context,
		db repository.Queryer,
		input repository.ContradictionAlertResolveInput,
	) (*models.EngramContradictionAlert, error)
	createMemoryCurationSuggestion func(
		ctx context.Context,
		db repository.Queryer,
		input repository.MemoryCurationSuggestionCreateInput,
	) (*models.MemoryCurationSuggestion, error)
	resetMemoryCurationSuggestions func(
		ctx context.Context,
		db repository.Queryer,
		input repository.MemoryCurationSuggestionResetInput,
	) (int, error)
	getMemoryCurationSuggestion func(
		ctx context.Context,
		db repository.Queryer,
		suggestionID uuid.UUID,
		projectID *string,
	) (*models.MemoryCurationSuggestion, error)
	listMemoryCurationSuggestions func(
		ctx context.Context,
		db repository.Queryer,
		input repository.MemoryCurationSuggestionListInput,
	) ([]models.MemoryCurationSuggestion, error)
	applyMemoryCurationSuggestionAction func(
		ctx context.Context,
		db repository.Queryer,
		input repository.MemoryCurationSuggestionActionInput,
	) (*models.MemoryCurationSuggestion, error)
	archiveEngramLink func(
		ctx context.Context,
		db repository.Queryer,
		input repository.EngramLinkArchiveInput,
	) (*models.EngramLinkRecord, error)

	listCollections      func(ctx context.Context, db repository.Queryer, input repository.CollectionListInput) ([]models.EngramCollectionRecord, error)
	getCollection        func(ctx context.Context, db repository.Queryer, collectionID uuid.UUID, includeDeleted bool) (*models.EngramCollectionRecord, error)
	createCollection     func(ctx context.Context, db repository.Queryer, input repository.CollectionCreateInput) (*models.EngramCollectionRecord, error)
	updateCollection     func(ctx context.Context, db repository.Queryer, input repository.CollectionUpdateInput) (*models.EngramCollectionRecord, error)
	softDeleteCollection func(ctx context.Context, db repository.Queryer, input repository.CollectionSoftDeleteInput) (bool, error)
	addCollectionItems   func(ctx context.Context, db repository.Queryer, input repository.CollectionAddItemsInput) (int, error)
	removeCollectionItem func(ctx context.Context, db repository.Queryer, input repository.CollectionRemoveItemInput) (bool, error)
}

func defaultServiceDeps() serviceDeps {
	return serviceDeps{
		listAdminSessions:       repository.ListAdminSessions,
		getAdminSession:         repository.GetAdminSession,
		softDeleteSession:       repository.SoftDeleteSession,
		restoreSession:          repository.RestoreSession,
		softDeleteLinkedEngrams: repository.SoftDeleteLinkedEngrams,

		listAdminEngrams:                    repository.ListAdminEngrams,
		getAdminEngram:                      repository.GetAdminEngram,
		updateAdminEngram:                   repository.UpdateAdminEngram,
		moveAdminEngramProject:              repository.MoveAdminEngramProject,
		softDeleteEngram:                    repository.SoftDeleteEngram,
		restoreEngram:                       repository.RestoreEngram,
		refreshEngramFreshness:              repository.RefreshEngramFreshnessScores,
		refreshConsolidationSuggestions:     repository.RefreshExactDuplicateConsolidationSuggestions,
		listConsolidationSuggestions:        repository.ListEngramConsolidationSuggestions,
		applyConsolidationSuggestionAction:  repository.ApplyEngramConsolidationSuggestionAction,
		refreshContradictionAlerts:          repository.RefreshContradictionAlerts,
		listContradictionAlerts:             repository.ListContradictionAlerts,
		resolveContradictionAlert:           repository.ResolveContradictionAlert,
		createMemoryCurationSuggestion:      repository.CreateMemoryCurationSuggestion,
		resetMemoryCurationSuggestions:      repository.ResetSuggestedMemoryCurationSuggestions,
		getMemoryCurationSuggestion:         repository.GetMemoryCurationSuggestion,
		listMemoryCurationSuggestions:       repository.ListMemoryCurationSuggestions,
		applyMemoryCurationSuggestionAction: repository.ApplyMemoryCurationSuggestionAction,
		archiveEngramLink:                   repository.ArchiveEngramLink,

		listCollections:      repository.ListCollections,
		getCollection:        repository.GetCollection,
		createCollection:     repository.CreateCollection,
		updateCollection:     repository.UpdateCollection,
		softDeleteCollection: repository.SoftDeleteCollection,
		addCollectionItems:   repository.AddCollectionItems,
		removeCollectionItem: repository.RemoveCollectionItem,
	}
}

// Service contains memory-admin business logic on top of repository operations.
type Service struct {
	db              repository.Queryer
	embeddingDim    int
	projectResolver ProjectResolver
	deps            serviceDeps
}

type sharedListRequestInput struct {
	ProjectID      *string
	OwnerUserID    *uuid.UUID
	IncludeDeleted bool
	Limit          int
	Offset         int
}

// NewService creates a memory-admin service.
func NewService(db repository.Queryer, embeddingDim int, projectResolver ProjectResolver) *Service {
	return &Service{
		db:              db,
		embeddingDim:    embeddingDim,
		projectResolver: projectResolver,
		deps:            defaultServiceDeps(),
	}
}

// ListSessions returns sessions filtered by request fields.
func (s *Service) ListSessions(ctx context.Context, request MemoryAdminListRequest) ([]models.AdminChatSessionRecord, error) {
	return s.deps.listAdminSessions(ctx, s.db, toSharedListRequestInput(request).sessionInput())
}

// GetSession returns a session by id when present.
func (s *Service) GetSession(ctx context.Context, sessionID uuid.UUID, includeDeleted bool) (*models.AdminChatSessionRecord, error) {
	return s.deps.getAdminSession(ctx, s.db, sessionID, includeDeleted)
}

// DeleteSession soft-deletes a session and optionally linked engrams.
func (s *Service) DeleteSession(ctx context.Context, sessionID, actorUserID uuid.UUID, payload SessionDeleteRequest) (SessionDeleteResponse, error) {
	deleted, err := s.deps.softDeleteSession(
		ctx,
		s.db,
		repository.AdminSessionSoftDeleteInput{
			SessionID:       sessionID,
			DeletedByUserID: actorUserID,
			Reason:          payload.Reason,
		},
	)
	if err != nil {
		return SessionDeleteResponse{}, err
	}
	if deleted == nil {
		return SessionDeleteResponse{}, ErrSessionNotFound
	}
	linkedDeleted := 0
	if payload.DeleteLinkedEngrams {
		count, err := s.deps.softDeleteLinkedEngrams(
			ctx,
			s.db,
			repository.SoftDeleteLinkedEngramsInput{
				SessionID:       sessionID,
				DeletedByUserID: actorUserID,
				Reason:          payload.Reason,
			},
		)
		if err != nil {
			return SessionDeleteResponse{}, err
		}
		linkedDeleted = count
	}
	return SessionDeleteResponse{SessionID: sessionID, Deleted: true, LinkedEngramsDeleted: linkedDeleted}, nil
}

// RestoreSession restores a soft-deleted session.
func (s *Service) RestoreSession(ctx context.Context, sessionID uuid.UUID) (SessionRestoreResponse, error) {
	restored, err := s.deps.restoreSession(ctx, s.db, sessionID)
	if err := boolResultNotFoundError(restored, err, ErrSessionNotFound); err != nil {
		return SessionRestoreResponse{}, err
	}
	return SessionRestoreResponse{SessionID: sessionID, Restored: true}, nil
}

// ListEngrams returns engrams filtered by request fields.
func (s *Service) ListEngrams(ctx context.Context, request MemoryAdminEngramListRequest) ([]models.AdminEngramRecord, error) {
	sharedInput := toSharedListRequestInput(request.MemoryAdminListRequest)
	return s.deps.listAdminEngrams(ctx, s.db, sharedInput.engramInput(request))
}

// FindEngram returns an engram by id, or nil when absent.
func (s *Service) FindEngram(ctx context.Context, engramID uuid.UUID, includeDeleted bool) (*models.AdminEngramRecord, error) {
	return s.deps.getAdminEngram(ctx, s.db, engramID, includeDeleted)
}

// GetEngram returns an engram by id or ErrEngramNotFound.
func (s *Service) GetEngram(ctx context.Context, engramID uuid.UUID, includeDeleted bool) (*models.AdminEngramRecord, error) {
	engram, err := s.FindEngram(ctx, engramID, includeDeleted)
	if err != nil {
		return nil, err
	}
	if engram == nil {
		return nil, ErrEngramNotFound
	}
	return engram, nil
}

// UpdateEngram updates mutable engram fields and validates stale writes.
func (s *Service) UpdateEngram(ctx context.Context, engramID, actorUserID uuid.UUID, payload EngramUpdateRequest) (*models.AdminEngramRecord, error) {
	current, err := s.GetEngram(ctx, engramID, false)
	if err != nil {
		return nil, err
	}
	if payload.ExpectedUpdatedAt != nil && !payload.ExpectedUpdatedAt.Equal(current.UpdatedAt) {
		return nil, ErrEngramStale
	}

	updated, err := s.deps.updateAdminEngram(
		ctx,
		s.db,
		repository.AdminEngramUpdateInput{
			EngramID:                engramID,
			ActorUserID:             actorUserID,
			Title:                   payload.Title,
			Abstract:                payload.Abstract,
			DetailedSummaryMarkdown: payload.DetailedSummaryMarkdown,
			Tags:                    payload.Tags,
			Keywords:                payload.Keywords,
			VisibilityScope:         payload.VisibilityScope,
			Sources:                 payload.Sources,
			EmbeddingDim:            s.embeddingDim,
		},
	)
	if err != nil {
		return nil, err
	}
	if updated == nil {
		return nil, ErrEngramNotFound
	}
	return updated, nil
}

// MoveEngram moves an engram to another project after project resolution.
func (s *Service) MoveEngram(
	ctx context.Context,
	engramID uuid.UUID,
	actor WriteActor,
	payload EngramMoveRequest,
) (*models.AdminEngramRecord, error) {
	return s.moveEngramWithActor(ctx, engramID, actor, payload)
}

func (s *Service) moveEngramWithActor(
	ctx context.Context,
	engramID uuid.UUID,
	actor WriteActor,
	payload EngramMoveRequest,
) (*models.AdminEngramRecord, error) {
	current, err := s.GetEngram(ctx, engramID, false)
	if err != nil {
		return nil, err
	}
	if payload.ExpectedUpdatedAt != nil && !payload.ExpectedUpdatedAt.Equal(current.UpdatedAt) {
		return nil, ErrEngramStale
	}
	targetProjectID, err := s.resolveProjectIDForWrite(ctx, actor, payload.TargetProjectID)
	if err != nil {
		return nil, err
	}
	moved, err := s.deps.moveAdminEngramProject(
		ctx,
		s.db,
		repository.AdminEngramMoveProjectInput{
			EngramID:        engramID,
			TargetProjectID: targetProjectID,
			ActorUserID:     actor.UserID,
		},
	)
	if err != nil {
		return nil, err
	}
	if moved == nil {
		return nil, ErrEngramNotFound
	}
	return moved, nil
}

// DeleteEngram soft-deletes an engram.
func (s *Service) DeleteEngram(ctx context.Context, engramID, actorUserID uuid.UUID, payload EngramDeleteRequest) (EngramDeleteResponse, error) {
	deleted, err := s.deps.softDeleteEngram(
		ctx,
		s.db,
		repository.AdminEngramSoftDeleteInput{
			EngramID:        engramID,
			DeletedByUserID: actorUserID,
			Reason:          payload.Reason,
		},
	)
	if err := boolResultNotFoundError(deleted, err, ErrEngramNotFound); err != nil {
		return EngramDeleteResponse{}, err
	}
	return EngramDeleteResponse{EngramID: engramID, Deleted: true}, nil
}

// RestoreEngram restores a soft-deleted engram.
func (s *Service) RestoreEngram(ctx context.Context, engramID uuid.UUID) (EngramRestoreResponse, error) {
	restored, err := s.deps.restoreEngram(ctx, s.db, engramID)
	if err := boolResultNotFoundError(restored, err, ErrEngramNotFound); err != nil {
		return EngramRestoreResponse{}, err
	}
	return EngramRestoreResponse{EngramID: engramID, Restored: true}, nil
}

// RefreshEngramFreshness recomputes freshness scores for active engrams.
func (s *Service) RefreshEngramFreshness(
	ctx context.Context,
	request EngramFreshnessRefreshRequest,
) (EngramFreshnessRefreshResponse, error) {
	refreshInput := repository.EngramFreshnessRefreshInput{
		ProjectID: request.ProjectID,
	}
	if request.HalfLifeDays != nil {
		refreshInput.HalfLifeDays = *request.HalfLifeDays
	}
	refreshed, err := s.deps.refreshEngramFreshness(ctx, s.db, refreshInput)
	if err != nil {
		return EngramFreshnessRefreshResponse{}, err
	}
	return EngramFreshnessRefreshResponse{
		ProjectID:     refreshed.ProjectID,
		HalfLifeDays:  refreshed.HalfLifeDays,
		ReferenceTime: refreshed.ReferenceTime,
		UpdatedCount:  refreshed.UpdatedCount,
	}, nil
}

// ListCollections returns collections filtered by request fields.
func (s *Service) ListCollections(ctx context.Context, request MemoryAdminListRequest) ([]models.EngramCollectionRecord, error) {
	return s.deps.listCollections(ctx, s.db, toSharedListRequestInput(request).collectionInput())
}

// FindCollection returns a collection by id, or nil when absent.
func (s *Service) FindCollection(ctx context.Context, collectionID uuid.UUID, includeDeleted bool) (*models.EngramCollectionRecord, error) {
	return s.deps.getCollection(ctx, s.db, collectionID, includeDeleted)
}

// GetCollection returns a collection by id or ErrCollectionNotFound.
func (s *Service) GetCollection(ctx context.Context, collectionID uuid.UUID, includeDeleted bool) (*models.EngramCollectionRecord, error) {
	collection, err := s.FindCollection(ctx, collectionID, includeDeleted)
	if err != nil {
		return nil, err
	}
	if collection == nil {
		return nil, ErrCollectionNotFound
	}
	return collection, nil
}

// CreateCollection creates a collection in a resolved project.
func (s *Service) CreateCollection(ctx context.Context, actorUserID uuid.UUID, actorRole string, payload CollectionCreateRequest) (*models.EngramCollectionRecord, error) {
	projectID, err := s.resolveProjectIDForWrite(ctx, WriteActor{UserID: actorUserID, Role: actorRole}, payload.ProjectID)
	if err != nil {
		return nil, err
	}
	return s.deps.createCollection(
		ctx,
		s.db,
		repository.CollectionCreateInput{
			ProjectID:   projectID,
			OwnerUserID: actorUserID,
			Name:        strings.TrimSpace(payload.Name),
			Description: strings.TrimSpace(payload.Description),
		},
	)
}

// UpdateCollection updates mutable collection fields and validates stale writes.
func (s *Service) UpdateCollection(ctx context.Context, collectionID uuid.UUID, payload CollectionUpdateRequest) (*models.EngramCollectionRecord, error) {
	current, err := s.GetCollection(ctx, collectionID, false)
	if err != nil {
		return nil, err
	}
	if payload.ExpectedUpdatedAt != nil && !payload.ExpectedUpdatedAt.Equal(current.UpdatedAt) {
		return nil, ErrCollectionStale
	}

	var name *string
	if payload.Name != nil {
		trimmed := strings.TrimSpace(*payload.Name)
		name = &trimmed
	}
	var description *string
	if payload.Description != nil {
		trimmed := strings.TrimSpace(*payload.Description)
		description = &trimmed
	}
	updated, err := s.deps.updateCollection(
		ctx,
		s.db,
		repository.CollectionUpdateInput{
			CollectionID: collectionID,
			Name:         name,
			Description:  description,
		},
	)
	if err != nil {
		return nil, err
	}
	if updated == nil {
		return nil, ErrCollectionNotFound
	}
	return updated, nil
}

// DeleteCollection soft-deletes a collection.
func (s *Service) DeleteCollection(ctx context.Context, collectionID, actorUserID uuid.UUID, payload CollectionDeleteRequest) (CollectionDeleteResponse, error) {
	deleted, err := s.deps.softDeleteCollection(
		ctx,
		s.db,
		repository.CollectionSoftDeleteInput{
			CollectionID:    collectionID,
			DeletedByUserID: actorUserID,
			Reason:          payload.Reason,
		},
	)
	if err := boolResultNotFoundError(deleted, err, ErrCollectionNotFound); err != nil {
		return CollectionDeleteResponse{}, err
	}
	return CollectionDeleteResponse{Deleted: true}, nil
}

// AddCollectionItems adds engrams to a collection.
func (s *Service) AddCollectionItems(ctx context.Context, collectionID, actorUserID uuid.UUID, payload CollectionItemsUpdateRequest) (CollectionItemsAddResponse, error) {
	added, err := s.deps.addCollectionItems(
		ctx,
		s.db,
		repository.CollectionAddItemsInput{
			CollectionID: collectionID,
			ActorUserID:  actorUserID,
			EngramIDs:    payload.EngramIDs,
		},
	)
	if err != nil {
		return CollectionItemsAddResponse{}, err
	}
	return CollectionItemsAddResponse{Added: added}, nil
}

// RemoveCollectionItem removes one engram from a collection.
func (s *Service) RemoveCollectionItem(ctx context.Context, collectionID, engramID uuid.UUID) (CollectionItemRemoveResponse, error) {
	removed, err := s.deps.removeCollectionItem(
		ctx,
		s.db,
		repository.CollectionRemoveItemInput{
			CollectionID: collectionID,
			EngramID:     engramID,
		},
	)
	if err != nil {
		return CollectionItemRemoveResponse{}, err
	}
	if !removed {
		return CollectionItemRemoveResponse{}, ErrCollectionNotFound
	}
	return CollectionItemRemoveResponse{Removed: true}, nil
}

func toSharedListRequestInput(request MemoryAdminListRequest) sharedListRequestInput {
	return sharedListRequestInput{
		ProjectID:      request.ProjectID,
		OwnerUserID:    request.OwnerUserID,
		IncludeDeleted: request.IncludeDeleted,
		Limit:          request.Limit,
		Offset:         request.Offset,
	}
}

func (input sharedListRequestInput) sessionInput() repository.AdminSessionListInput {
	return repository.AdminSessionListInput{
		ProjectID:      input.ProjectID,
		OwnerUserID:    input.OwnerUserID,
		IncludeDeleted: input.IncludeDeleted,
		Limit:          input.Limit,
		Offset:         input.Offset,
	}
}

func (input sharedListRequestInput) collectionInput() repository.CollectionListInput {
	return repository.CollectionListInput{
		ProjectID:      input.ProjectID,
		OwnerUserID:    input.OwnerUserID,
		IncludeDeleted: input.IncludeDeleted,
		Limit:          input.Limit,
		Offset:         input.Offset,
	}
}

func (input sharedListRequestInput) engramInput(request MemoryAdminEngramListRequest) repository.AdminEngramListInput {
	return repository.AdminEngramListInput{
		ProjectID:      input.ProjectID,
		OwnerUserID:    input.OwnerUserID,
		IncludeDeleted: input.IncludeDeleted,
		Limit:          input.Limit,
		Offset:         input.Offset,
		SessionID:      request.SessionID,
		QueryText:      request.QueryText,
	}
}

func boolResultNotFoundError(result bool, opErr error, notFoundErr error) error {
	if opErr != nil {
		return opErr
	}
	if !result {
		return notFoundErr
	}
	return nil
}

func (s *Service) resolveProjectIDForWrite(ctx context.Context, actor WriteActor, projectID string) (string, error) {
	if s.projectResolver == nil {
		return "", ErrProjectResolverNotConfigured
	}
	resolution, err := s.projectResolver.ResolveProjectIDForWrite(
		ctx,
		ResolveProjectWriteInput{
			ActorUserID: actor.UserID,
			ActorRole:   actor.Role,
			ProjectID:   projectID,
		},
	)
	if err != nil {
		return "", err
	}
	return resolution.ProjectID, nil
}
