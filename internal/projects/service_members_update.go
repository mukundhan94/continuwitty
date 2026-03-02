package projects

import (
	"context"

	"engram/internal/models"
	"engram/internal/repository"

	"github.com/google/uuid"
)

// UpdateProjectMember updates an existing member role.
func (service *Service) UpdateProjectMember(
	ctx context.Context,
	actorUserID uuid.UUID,
	actorRole models.UserRole,
	request ProjectMemberUpdateRequest,
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
	current, err := service.getActiveProjectMember(ctx, project.ProjectID, request.UserID)
	if err != nil {
		return nil, err
	}
	updated, err := service.updateProjectMemberRole(
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
			EventType:   "project.member.update",
			Metadata: map[string]any{
				"from_role": string(current.Role),
				"to_role":   string(request.Role),
			},
		},
	); err != nil {
		return nil, err
	}
	return updated, nil
}

func (service *Service) getActiveProjectMember(
	ctx context.Context,
	projectID string,
	userID uuid.UUID,
) (*models.ProjectMemberRecord, error) {
	current, err := service.deps.getProjectMember(
		ctx,
		service.db,
		repository.ProjectMemberGetInput{
			ProjectID:      projectID,
			UserID:         userID,
			IncludeRevoked: false,
		},
	)
	if err != nil {
		return nil, err
	}
	if current == nil {
		return nil, ErrProjectMemberNotFound
	}
	return current, nil
}

func (service *Service) updateProjectMemberRole(
	ctx context.Context,
	input memberMutationInput,
) (*models.ProjectMemberRecord, error) {
	updated, err := service.deps.updateProjectMemberRole(
		ctx,
		service.db,
		repository.ProjectMemberUpdateInput{
			ProjectID:       input.ProjectID,
			UserID:          input.TargetUser,
			Role:            input.Role,
			UpdatedByUserID: input.ActorUserID,
		},
	)
	if err != nil {
		return nil, err
	}
	if updated == nil {
		return nil, ErrProjectMemberNotFound
	}
	return updated, nil
}
