package projects

import (
	"context"
	"errors"

	"engram/internal/models"
	"engram/internal/repository"

	"github.com/google/uuid"
)

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

func (service *Service) resolveExplicitProjectID(
	ctx context.Context,
	actorUserID uuid.UUID,
	actorRole models.UserRole,
	projectID string,
) (Resolution, error) {
	actor := ActorContext{UserID: actorUserID, Role: actorRole}
	err := service.ensureWritableExplicitProject(ctx, actor, projectID)
	switch {
	case err == nil:
		return Resolution{ProjectID: projectID, UsedDefaultProject: false}, nil
	case !errors.Is(err, ErrProjectNotFound):
		return Resolution{}, err
	}
	if err := service.ensureCreatableExplicitProject(ctx, actorUserID, actorRole, projectID); err != nil {
		return Resolution{}, err
	}
	return Resolution{ProjectID: projectID, UsedDefaultProject: false}, nil
}

func (service *Service) ensureWritableExplicitProject(
	ctx context.Context,
	actor ActorContext,
	projectID string,
) error {
	visible, err := service.getVisibleProject(
		ctx,
		projectVisibilityLookup{
			Actor:           actor,
			ProjectID:       projectID,
			IncludeArchived: false,
		},
	)
	if err != nil {
		return err
	}
	if visible == nil {
		return ErrProjectNotFound
	}
	if !canWriteVisibleProject(actor.Role, visible.MembershipRole) {
		return ErrProjectWriteForbidden
	}
	return nil
}

func (service *Service) ensureCreatableExplicitProject(
	ctx context.Context,
	actorUserID uuid.UUID,
	actorRole models.UserRole,
	projectID string,
) error {
	existing, err := service.lookupProjectAsAdmin(ctx, actorUserID, projectID)
	if err != nil {
		return err
	}
	if existing != nil {
		return ErrProjectWriteForbidden
	}
	if !canCreateProject(actorRole) {
		return ErrProjectWriteForbidden
	}
	return service.ensureProjectExistsWithOwnerMembership(ctx, projectID, actorUserID)
}

func (service *Service) lookupProjectAsAdmin(
	ctx context.Context,
	actorUserID uuid.UUID,
	projectID string,
) (*models.ProjectRecord, error) {
	return service.getVisibleProject(
		ctx,
		projectVisibilityLookup{
			Actor:           ActorContext{UserID: actorUserID, Role: models.UserRoleAdmin},
			ProjectID:       projectID,
			IncludeArchived: true,
		},
	)
}

func (service *Service) ensureProjectExistsWithOwnerMembership(
	ctx context.Context,
	projectID string,
	ownerUserID uuid.UUID,
) error {
	created, err := service.deps.ensureProjectExists(
		ctx,
		service.db,
		repository.ProjectEnsureInput{
			ProjectID:   projectID,
			OwnerUserID: ownerUserID,
		},
	)
	if err != nil {
		return err
	}
	if created == nil {
		return nil
	}
	return service.ensureCanonicalOwnerMembership(ctx, *created)
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
	visible, err := service.getVisibleProject(
		ctx,
		projectVisibilityLookup{
			Actor:           ActorContext{UserID: actorUserID, Role: actorRole},
			ProjectID:       normalizedDefaultProjectID,
			IncludeArchived: false,
		},
	)
	if err != nil {
		return Resolution{}, err
	}
	if visible == nil {
		return Resolution{}, ErrDefaultProjectNotAccessible
	}
	if !canWriteVisibleProject(actorRole, visible.MembershipRole) {
		return Resolution{}, ErrProjectWriteForbidden
	}
	return Resolution{ProjectID: normalizedDefaultProjectID, UsedDefaultProject: true}, nil
}

func normalizeOptionalDefaultProjectID(defaultProjectID *string) string {
	if defaultProjectID == nil {
		return ""
	}
	return normalizeProjectID(*defaultProjectID)
}
