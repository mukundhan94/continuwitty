package projects

import (
	"context"

	"engram/internal/models"
	"engram/internal/repository"

	"github.com/google/uuid"
)

// ShareEngram sets engram visibility_scope=project.
func (service *Service) ShareEngram(
	ctx context.Context,
	actorUserID uuid.UUID,
	actorRole models.UserRole,
	engramID uuid.UUID,
) (*models.EngramVisibilityRecord, error) {
	return service.setEngramVisibilityForActor(
		ctx,
		engramActorVisibilityInput{
			Actor:       ActorContext{UserID: actorUserID, Role: actorRole},
			EngramID:    engramID,
			TargetScope: models.VisibilityScopeProject,
			EventType:   "engram.share",
		},
	)
}

// UnshareEngram sets engram visibility_scope=private.
func (service *Service) UnshareEngram(
	ctx context.Context,
	actorUserID uuid.UUID,
	actorRole models.UserRole,
	engramID uuid.UUID,
) (*models.EngramVisibilityRecord, error) {
	return service.setEngramVisibilityForActor(
		ctx,
		engramActorVisibilityInput{
			Actor:       ActorContext{UserID: actorUserID, Role: actorRole},
			EngramID:    engramID,
			TargetScope: models.VisibilityScopePrivate,
			EventType:   "engram.unshare",
		},
	)
}

func (service *Service) setEngramVisibilityForActor(
	ctx context.Context,
	input engramActorVisibilityInput,
) (*models.EngramVisibilityRecord, error) {
	return service.setEngramVisibility(
		ctx,
		engramVisibilityMutationInput{
			Actor:       input.Actor,
			EngramID:    input.EngramID,
			TargetScope: input.TargetScope,
			EventType:   input.EventType,
		},
	)
}

func (service *Service) setEngramVisibility(
	ctx context.Context,
	input engramVisibilityMutationInput,
) (*models.EngramVisibilityRecord, error) {
	current, err := service.deps.getEngramShareRecord(ctx, service.db, input.EngramID, false)
	if err != nil {
		return nil, err
	}
	if current == nil {
		return nil, ErrEngramNotFound
	}
	allowed, err := service.canShareEngram(ctx, input.Actor.UserID, input.Actor.Role, *current)
	if err != nil {
		return nil, err
	}
	if !allowed {
		return nil, ErrEngramShareForbidden
	}
	updated, err := service.deps.updateEngramVisibility(
		ctx,
		service.db,
		repository.EngramVisibilityUpdateInput{
			EngramID:        input.EngramID,
			VisibilityScope: input.TargetScope,
			ActorUserID:     input.Actor.UserID,
		},
	)
	if err != nil {
		return nil, err
	}
	if updated == nil {
		return nil, ErrEngramNotFound
	}
	targetEngramID := updated.EngramID
	if err := service.emitProjectAuditEvent(
		ctx,
		repository.ProjectAuditEventCreateInput{
			ProjectID:      updated.ProjectID,
			ActorUserID:    &input.Actor.UserID,
			EventType:      input.EventType,
			TargetType:     "engram",
			TargetEngramID: &targetEngramID,
			Metadata: map[string]any{
				"from_visibility_scope": string(current.VisibilityScope),
				"to_visibility_scope":   string(updated.VisibilityScope),
			},
		},
	); err != nil {
		return nil, err
	}
	return &models.EngramVisibilityRecord{
		EngramID:        updated.EngramID,
		ProjectID:       updated.ProjectID,
		VisibilityScope: updated.VisibilityScope,
	}, nil
}

func (service *Service) canShareEngram(
	ctx context.Context,
	actorUserID uuid.UUID,
	actorRole models.UserRole,
	engram repository.EngramShareRecord,
) (bool, error) {
	if actorRole == models.UserRoleAdmin {
		return true, nil
	}
	projectRole, err := service.deps.resolveActorProjectRole(
		ctx,
		service.db,
		repository.ActorProjectRoleInput{
			ProjectID:   engram.ProjectID,
			ActorUserID: actorUserID,
		},
	)
	if err != nil {
		return false, err
	}
	if projectRole == nil {
		return false, nil
	}
	switch *projectRole {
	case models.ProjectMemberRoleOwner:
		return true, nil
	case models.ProjectMemberRoleEditor:
		return engram.OwnerUserID != nil && *engram.OwnerUserID == actorUserID, nil
	default:
		return false, nil
	}
}
