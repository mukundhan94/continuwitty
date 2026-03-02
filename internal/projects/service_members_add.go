package projects

import (
	"context"

	"engram/internal/models"
	"engram/internal/repository"

	"github.com/google/uuid"
)

// AddProjectMember adds or restores a project member role.
func (service *Service) AddProjectMember(
	ctx context.Context,
	actorUserID uuid.UUID,
	actorRole models.UserRole,
	request ProjectMemberCreateRequest,
) (*models.ProjectMemberRecord, error) {
	if err := validateProjectMemberRole(request.Role); err != nil {
		return nil, err
	}
	project, err := service.prepareProjectMemberMutation(
		ctx,
		ActorContext{UserID: actorUserID, Role: actorRole},
		request.ProjectID,
		request.UserID,
	)
	if err != nil {
		return nil, err
	}
	member, err := service.addOrRestoreProjectMember(
		ctx,
		memberMutationInput{
			ProjectID:   project.ProjectID,
			TargetUser:  request.UserID,
			ActorUserID: actorUserID,
			Role:        request.Role,
		},
	)
	if err != nil {
		return nil, err
	}
	if err := service.emitProjectMemberAuditEvent(
		ctx,
		memberAuditInput{
			ProjectID:   project.ProjectID,
			ActorUserID: actorUserID,
			TargetUser:  request.UserID,
			EventType:   "project.member.add",
			Metadata: map[string]any{
				"role": string(request.Role),
			},
		},
	); err != nil {
		return nil, err
	}
	return member, nil
}

func (service *Service) addOrRestoreProjectMember(
	ctx context.Context,
	input memberMutationInput,
) (*models.ProjectMemberRecord, error) {
	member, err := service.deps.addOrRestoreProjectMember(
		ctx,
		service.db,
		repository.ProjectMemberAddInput{
			ProjectID:     input.ProjectID,
			UserID:        input.TargetUser,
			Role:          input.Role,
			AddedByUserID: input.ActorUserID,
		},
	)
	if err != nil {
		return nil, err
	}
	if member == nil {
		return nil, ErrProjectMemberNotFound
	}
	return member, nil
}
