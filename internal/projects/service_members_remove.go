package projects

import (
	"context"

	"engram/internal/models"
	"engram/internal/repository"

	"github.com/google/uuid"
)

// RemoveProjectMember revokes an active member role.
func (service *Service) RemoveProjectMember(
	ctx context.Context,
	actorUserID uuid.UUID,
	actorRole models.UserRole,
	request ProjectMemberRemoveRequest,
) error {
	project, err := service.prepareProjectMemberMutation(
		ctx,
		ActorContext{UserID: actorUserID, Role: actorRole},
		request.ProjectID,
		request.UserID,
	)
	if err != nil {
		return err
	}
	if err := service.removeProjectMember(ctx, project.ProjectID, request.UserID, actorUserID); err != nil {
		return err
	}
	return service.emitProjectMemberAuditEvent(
		ctx,
		memberAuditInput{
			ProjectID:   project.ProjectID,
			ActorUserID: actorUserID,
			TargetUser:  request.UserID,
			EventType:   "project.member.remove",
		},
	)
}

func (service *Service) removeProjectMember(
	ctx context.Context,
	projectID string,
	userID uuid.UUID,
	actorUserID uuid.UUID,
) error {
	removed, err := service.deps.removeProjectMember(
		ctx,
		service.db,
		repository.ProjectMemberRemoveInput{
			ProjectID:       projectID,
			UserID:          userID,
			RevokedByUserID: actorUserID,
		},
	)
	if err != nil {
		return err
	}
	if !removed {
		return ErrProjectMemberNotFound
	}
	return nil
}
