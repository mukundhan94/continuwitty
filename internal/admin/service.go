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

		listAdminEngrams:       repository.ListAdminEngrams,
		getAdminEngram:         repository.GetAdminEngram,
		updateAdminEngram:      repository.UpdateAdminEngram,
		moveAdminEngramProject: repository.MoveAdminEngramProject,
		softDeleteEngram:       repository.SoftDeleteEngram,
		restoreEngram:          repository.RestoreEngram,

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
	return s.deps.listAdminSessions(
		ctx,
		s.db,
		repository.AdminSessionListInput{
			ProjectID:      request.ProjectID,
			OwnerUserID:    request.OwnerUserID,
			IncludeDeleted: request.IncludeDeleted,
			Limit:          request.Limit,
			Offset:         request.Offset,
		},
	)
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
	if err != nil {
		return SessionRestoreResponse{}, err
	}
	if !restored {
		return SessionRestoreResponse{}, ErrSessionNotFound
	}
	return SessionRestoreResponse{SessionID: sessionID, Restored: true}, nil
}

// ListEngrams returns engrams filtered by request fields.
func (s *Service) ListEngrams(ctx context.Context, request MemoryAdminEngramListRequest) ([]models.AdminEngramRecord, error) {
	return s.deps.listAdminEngrams(
		ctx,
		s.db,
		repository.AdminEngramListInput{
			ProjectID:      request.ProjectID,
			OwnerUserID:    request.OwnerUserID,
			IncludeDeleted: request.IncludeDeleted,
			Limit:          request.Limit,
			Offset:         request.Offset,
			SessionID:      request.SessionID,
			QueryText:      request.QueryText,
		},
	)
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
func (s *Service) MoveEngram(ctx context.Context, engramID, actorUserID uuid.UUID, actorRole string, payload EngramMoveRequest) (*models.AdminEngramRecord, error) {
	current, err := s.GetEngram(ctx, engramID, false)
	if err != nil {
		return nil, err
	}
	if payload.ExpectedUpdatedAt != nil && !payload.ExpectedUpdatedAt.Equal(current.UpdatedAt) {
		return nil, ErrEngramStale
	}
	if s.projectResolver == nil {
		return nil, ErrProjectResolverNotConfigured
	}
	resolution, err := s.projectResolver.ResolveProjectIDForWrite(
		ctx,
		ResolveProjectWriteInput{
			ActorUserID: actorUserID,
			ActorRole:   actorRole,
			ProjectID:   payload.TargetProjectID,
		},
	)
	if err != nil {
		return nil, err
	}
	moved, err := s.deps.moveAdminEngramProject(
		ctx,
		s.db,
		repository.AdminEngramMoveProjectInput{
			EngramID:        engramID,
			TargetProjectID: resolution.ProjectID,
			ActorUserID:     actorUserID,
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
	if err != nil {
		return EngramDeleteResponse{}, err
	}
	if !deleted {
		return EngramDeleteResponse{}, ErrEngramNotFound
	}
	return EngramDeleteResponse{EngramID: engramID, Deleted: true}, nil
}

// RestoreEngram restores a soft-deleted engram.
func (s *Service) RestoreEngram(ctx context.Context, engramID uuid.UUID) (EngramRestoreResponse, error) {
	restored, err := s.deps.restoreEngram(ctx, s.db, engramID)
	if err != nil {
		return EngramRestoreResponse{}, err
	}
	if !restored {
		return EngramRestoreResponse{}, ErrEngramNotFound
	}
	return EngramRestoreResponse{EngramID: engramID, Restored: true}, nil
}

// ListCollections returns collections filtered by request fields.
func (s *Service) ListCollections(ctx context.Context, request MemoryAdminListRequest) ([]models.EngramCollectionRecord, error) {
	return s.deps.listCollections(
		ctx,
		s.db,
		repository.CollectionListInput{
			ProjectID:      request.ProjectID,
			OwnerUserID:    request.OwnerUserID,
			IncludeDeleted: request.IncludeDeleted,
			Limit:          request.Limit,
			Offset:         request.Offset,
		},
	)
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
	if s.projectResolver == nil {
		return nil, ErrProjectResolverNotConfigured
	}
	resolution, err := s.projectResolver.ResolveProjectIDForWrite(
		ctx,
		ResolveProjectWriteInput{
			ActorUserID: actorUserID,
			ActorRole:   actorRole,
			ProjectID:   payload.ProjectID,
		},
	)
	if err != nil {
		return nil, err
	}
	return s.deps.createCollection(
		ctx,
		s.db,
		repository.CollectionCreateInput{
			ProjectID:   resolution.ProjectID,
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
	if err != nil {
		return CollectionDeleteResponse{}, err
	}
	if !deleted {
		return CollectionDeleteResponse{}, ErrCollectionNotFound
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
