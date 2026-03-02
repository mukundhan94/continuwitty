package projects

import (
	"context"

	"engram/internal/models"

	"github.com/google/uuid"
)

func validateProjectMemberRole(role models.ProjectMemberRole) error {
	if role == models.ProjectMemberRoleOwner {
		return ErrProjectOwnerRoleNotAssignable
	}
	return nil
}

func (service *Service) prepareProjectMemberMutation(
	ctx context.Context,
	actor ActorContext,
	projectID string,
	targetUserID uuid.UUID,
) (*models.ProjectRecord, error) {
	project, err := service.requireProjectMemberManager(
		ctx,
		actor.UserID,
		actor.Role,
		projectID,
	)
	if err != nil {
		return nil, err
	}
	if targetUserID == project.OwnerUserID {
		return nil, ErrProjectOwnerMembershipImmutable
	}
	return project, nil
}
