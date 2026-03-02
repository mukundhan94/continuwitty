package projects

import (
	"context"
	"errors"

	"engram/internal/models"
	"engram/internal/repository"

	"github.com/google/uuid"
)

// ListProjectAuditEvents lists project audit rows for owner/admin.
func (service *Service) ListProjectAuditEvents(
	ctx context.Context,
	actorUserID uuid.UUID,
	actorRole models.UserRole,
	request ProjectAuditListRequest,
) ([]models.ProjectAuditEventRecord, error) {
	project, err := service.requireProjectMemberManager(ctx, actorUserID, actorRole, request.ProjectID)
	if err != nil {
		if errors.Is(err, ErrProjectMemberManagementForbidden) {
			return nil, ErrProjectAuditForbidden
		}
		return nil, err
	}
	return service.deps.listProjectAuditEvents(
		ctx,
		service.db,
		repository.ProjectAuditEventListInput{
			ProjectID: project.ProjectID,
			Limit:     request.Limit,
			Offset:    request.Offset,
		},
	)
}

func (service *Service) emitProjectAuditEvent(
	ctx context.Context,
	input repository.ProjectAuditEventCreateInput,
) error {
	_, err := service.deps.createProjectAuditEvent(ctx, service.db, input)
	return err
}

func (service *Service) emitProjectMemberAuditEvent(
	ctx context.Context,
	input memberAuditInput,
) error {
	return service.emitProjectAuditEvent(
		ctx,
		repository.ProjectAuditEventCreateInput{
			ProjectID:    input.ProjectID,
			ActorUserID:  &input.ActorUserID,
			EventType:    input.EventType,
			TargetType:   "user",
			TargetUserID: &input.TargetUser,
			Metadata:     input.Metadata,
		},
	)
}
