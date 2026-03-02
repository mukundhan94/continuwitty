package projects

import (
	"context"
	"strings"

	"engram/internal/models"
	"engram/internal/repository"

	"github.com/google/uuid"
)

// ListProjects returns project rows visible to the actor.
func (service *Service) ListProjects(
	ctx context.Context,
	actor ActorContext,
	request ProjectListRequest,
) ([]models.ProjectRecord, error) {
	return service.deps.listProjectsForActor(
		ctx,
		service.db,
		repository.ProjectListInput{
			ActorUserID:     actor.UserID,
			ActorRole:       string(actor.Role),
			IncludeArchived: request.IncludeArchived,
			Limit:           request.Limit,
			Offset:          request.Offset,
		},
	)
}

// GetProject returns a visible project or nil when missing.
func (service *Service) GetProject(
	ctx context.Context,
	actor ActorContext,
	request ProjectGetRequest,
) (*models.ProjectRecord, error) {
	normalizedProjectID := normalizeProjectID(request.ProjectID)
	if normalizedProjectID == "" {
		return nil, nil
	}
	return service.getVisibleProject(
		ctx,
		projectVisibilityLookup{
			Actor:           actor,
			ProjectID:       normalizedProjectID,
			IncludeArchived: request.IncludeArchived,
		},
	)
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
	if !canCreateProject(actorRole) {
		return nil, ErrProjectWriteForbidden
	}
	ownerUserID := resolveOwnerUserID(actorUserID, actorRole, request.OwnerUserID)
	created, err := service.deps.createProject(
		ctx,
		service.db,
		repository.ProjectCreateInput{
			ProjectID:   normalizedProjectID,
			Name:        strings.TrimSpace(request.Name),
			Description: strings.TrimSpace(request.Description),
			OwnerUserID: ownerUserID,
		},
	)
	if err != nil {
		return nil, err
	}
	if created == nil {
		return nil, nil
	}
	if err := service.ensureCanonicalOwnerMembership(ctx, *created); err != nil {
		return nil, err
	}
	return created, nil
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
	visible, err := service.getVisibleProject(
		ctx,
		projectVisibilityLookup{
			Actor:           ActorContext{UserID: actorUserID, Role: actorRole},
			ProjectID:       normalizedProjectID,
			IncludeArchived: false,
		},
	)
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
