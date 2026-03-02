package projects

import (
	"context"

	"engram/internal/models"
	"engram/internal/repository"
)

// ListProjectMembers lists members for a project visible to owner/admin.
func (service *Service) ListProjectMembers(
	ctx context.Context,
	actor ActorContext,
	request ProjectMemberListRequest,
) ([]models.ProjectMemberRecord, error) {
	project, err := service.requireProjectMemberManager(ctx, actor.UserID, actor.Role, request.ProjectID)
	if err != nil {
		return nil, err
	}
	if err := service.ensureCanonicalOwnerMembership(ctx, *project); err != nil {
		return nil, err
	}
	return service.deps.listProjectMembers(
		ctx,
		service.db,
		repository.ProjectMemberListInput{
			ProjectID:      project.ProjectID,
			IncludeRevoked: request.IncludeRevoked,
			Limit:          request.Limit,
			Offset:         request.Offset,
		},
	)
}
