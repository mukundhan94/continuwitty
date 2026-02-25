package projects

import (
	"context"
	"errors"
	"strings"

	"engram/internal/models"
	"engram/internal/repository"

	"github.com/google/uuid"
)

var (
	// ErrProjectIDMustNotBeBlank indicates project_id was empty after normalization.
	ErrProjectIDMustNotBeBlank = errors.New("project_id must not be blank")
	// ErrProjectNotFound indicates a project is not visible to the caller.
	ErrProjectNotFound = errors.New("Project not found")
	// ErrUserNotFound indicates no user row matched a default-project update.
	ErrUserNotFound = errors.New("User not found")
	// ErrProjectIDRequiredWhenNoDefaultProject indicates a write call needs explicit project context.
	ErrProjectIDRequiredWhenNoDefaultProject = errors.New("project_id is required when no default project is configured")
	// ErrDefaultProjectNotAccessible indicates a default project cannot be used by the caller.
	ErrDefaultProjectNotAccessible = errors.New("default project is not accessible; set a valid default project first")
)

// CreateProjectRequest captures project creation input.
type CreateProjectRequest struct {
	ProjectID   string
	Name        string
	Description string
	OwnerUserID *uuid.UUID
}

// Resolution captures resolved project context for write operations.
type Resolution struct {
	ProjectID          string
	UsedDefaultProject bool
}

type serviceDeps struct {
	listProjectsForActor func(ctx context.Context, db repository.Queryer, input repository.ProjectListInput) ([]models.ProjectRecord, error)
	getProjectForActor   func(ctx context.Context, db repository.Queryer, input repository.ProjectGetInput) (*models.ProjectRecord, error)
	createProject        func(ctx context.Context, db repository.Queryer, input repository.ProjectCreateInput) (*models.ProjectRecord, error)
	ensureProjectExists  func(ctx context.Context, db repository.Queryer, input repository.ProjectEnsureInput) (*models.ProjectRecord, error)
	getUserDefault       func(ctx context.Context, db repository.Queryer, userID uuid.UUID) (*string, error)
	setUserDefault       func(ctx context.Context, db repository.Queryer, userID uuid.UUID, projectID string) (bool, error)
}

func defaultServiceDeps() serviceDeps {
	return serviceDeps{
		listProjectsForActor: repository.ListProjectsForActor,
		getProjectForActor:   repository.GetProjectForActor,
		createProject:        repository.CreateProject,
		ensureProjectExists:  repository.EnsureProjectExists,
		getUserDefault:       repository.GetUserDefaultProjectID,
		setUserDefault:       repository.SetUserDefaultProjectID,
	}
}

// Service owns project business rules on top of repository operations.
type Service struct {
	db   repository.Queryer
	deps serviceDeps
}

// NewService builds a project service with repository defaults.
func NewService(db repository.Queryer) *Service {
	return &Service{db: db, deps: defaultServiceDeps()}
}

// ListProjects returns project rows visible to the actor.
func (service *Service) ListProjects(
	ctx context.Context,
	actorUserID uuid.UUID,
	actorRole models.UserRole,
	includeArchived bool,
	limit int,
	offset int,
) ([]models.ProjectRecord, error) {
	return service.deps.listProjectsForActor(
		ctx,
		service.db,
		repository.ProjectListInput{
			ActorUserID:     actorUserID,
			ActorRole:       string(actorRole),
			IncludeArchived: includeArchived,
			Limit:           limit,
			Offset:          offset,
		},
	)
}

// GetProject returns a visible project or nil when missing.
func (service *Service) GetProject(
	ctx context.Context,
	actorUserID uuid.UUID,
	actorRole models.UserRole,
	projectID string,
	includeArchived bool,
) (*models.ProjectRecord, error) {
	normalizedProjectID := normalizeProjectID(projectID)
	if normalizedProjectID == "" {
		return nil, nil
	}
	return service.getVisibleProject(ctx, actorUserID, actorRole, normalizedProjectID, includeArchived)
}

// CreateProject creates a project and applies owner rules.
func (service *Service) CreateProject(
	ctx context.Context,
	actorUserID uuid.UUID,
	actorRole models.UserRole,
	request CreateProjectRequest,
) (*models.ProjectRecord, error) {
	normalizedProjectID := normalizeProjectID(request.ProjectID)
	if normalizedProjectID == "" {
		return nil, ErrProjectIDMustNotBeBlank
	}
	ownerUserID := resolveOwnerUserID(actorUserID, actorRole, request.OwnerUserID)
	return service.deps.createProject(
		ctx,
		service.db,
		repository.ProjectCreateInput{
			ProjectID:   normalizedProjectID,
			Name:        strings.TrimSpace(request.Name),
			Description: strings.TrimSpace(request.Description),
			OwnerUserID: ownerUserID,
		},
	)
}

// GetDefaultProjectID returns the caller's default project id when present.
func (service *Service) GetDefaultProjectID(ctx context.Context, actorUserID uuid.UUID) (*string, error) {
	return service.deps.getUserDefault(ctx, service.db, actorUserID)
}

// SetDefaultProjectID updates the caller's default project id after visibility checks.
func (service *Service) SetDefaultProjectID(
	ctx context.Context,
	actorUserID uuid.UUID,
	actorRole models.UserRole,
	projectID string,
) (string, error) {
	normalizedProjectID := normalizeProjectID(projectID)
	if normalizedProjectID == "" {
		return "", ErrProjectIDMustNotBeBlank
	}
	visible, err := service.getVisibleProject(ctx, actorUserID, actorRole, normalizedProjectID, false)
	if err != nil {
		return "", err
	}
	if visible == nil {
		return "", ErrProjectNotFound
	}
	updated, err := service.deps.setUserDefault(ctx, service.db, actorUserID, normalizedProjectID)
	if err != nil {
		return "", err
	}
	if !updated {
		return "", ErrUserNotFound
	}
	return normalizedProjectID, nil
}

// ResolveProjectIDForWrite resolves explicit or default project context for a write call.
func (service *Service) ResolveProjectIDForWrite(
	ctx context.Context,
	actorUserID uuid.UUID,
	actorRole models.UserRole,
	projectID string,
) (Resolution, error) {
	explicitProjectID := normalizeProjectID(projectID)
	if explicitProjectID != "" {
		return service.resolveExplicitProjectID(ctx, actorUserID, actorRole, explicitProjectID)
	}
	return service.resolveDefaultProjectID(ctx, actorUserID, actorRole)
}

func resolveOwnerUserID(actorUserID uuid.UUID, actorRole models.UserRole, requestedOwnerUserID *uuid.UUID) uuid.UUID {
	if actorRole == models.UserRoleAdmin && requestedOwnerUserID != nil {
		return *requestedOwnerUserID
	}
	return actorUserID
}

func normalizeProjectID(projectID string) string {
	return strings.TrimSpace(projectID)
}

func (service *Service) getVisibleProject(
	ctx context.Context,
	actorUserID uuid.UUID,
	actorRole models.UserRole,
	projectID string,
	includeArchived bool,
) (*models.ProjectRecord, error) {
	return service.deps.getProjectForActor(
		ctx,
		service.db,
		repository.ProjectGetInput{
			ProjectID:       projectID,
			ActorUserID:     actorUserID,
			ActorRole:       string(actorRole),
			IncludeArchived: includeArchived,
		},
	)
}

func (service *Service) resolveExplicitProjectID(
	ctx context.Context,
	actorUserID uuid.UUID,
	actorRole models.UserRole,
	projectID string,
) (Resolution, error) {
	visible, err := service.getVisibleProject(ctx, actorUserID, actorRole, projectID, false)
	if err != nil {
		return Resolution{}, err
	}
	if visible == nil {
		if _, err := service.deps.ensureProjectExists(
			ctx,
			service.db,
			repository.ProjectEnsureInput{ProjectID: projectID, OwnerUserID: actorUserID},
		); err != nil {
			return Resolution{}, err
		}
	}
	return Resolution{ProjectID: projectID, UsedDefaultProject: false}, nil
}

func (service *Service) resolveDefaultProjectID(
	ctx context.Context,
	actorUserID uuid.UUID,
	actorRole models.UserRole,
) (Resolution, error) {
	defaultProjectID, err := service.deps.getUserDefault(ctx, service.db, actorUserID)
	if err != nil {
		return Resolution{}, err
	}
	normalizedDefaultProjectID := normalizeOptionalDefaultProjectID(defaultProjectID)
	if normalizedDefaultProjectID == "" {
		return Resolution{}, ErrProjectIDRequiredWhenNoDefaultProject
	}
	visible, err := service.getVisibleProject(ctx, actorUserID, actorRole, normalizedDefaultProjectID, false)
	if err != nil {
		return Resolution{}, err
	}
	if visible == nil {
		return Resolution{}, ErrDefaultProjectNotAccessible
	}
	return Resolution{ProjectID: normalizedDefaultProjectID, UsedDefaultProject: true}, nil
}

func normalizeOptionalDefaultProjectID(defaultProjectID *string) string {
	if defaultProjectID == nil {
		return ""
	}
	return normalizeProjectID(*defaultProjectID)
}
