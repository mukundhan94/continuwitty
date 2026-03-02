package main

import (
	"context"

	internalapi "engram/internal/api"
	"engram/internal/models"
	"engram/internal/projects"
)

func (adapter projectRouteServiceAdapter) UpdateProjectMember(
	ctx context.Context,
	request internalapi.ProjectMemberUpdateRouteRequest,
) (*models.ProjectMemberRecord, error) {
	return adapter.service.UpdateProjectMember(
		ctx,
		request.ActorUserID,
		request.ActorRole,
		projects.ProjectMemberUpdateRequest{
			ProjectID: request.ProjectID,
			UserID:    request.UserID,
			Role:      request.Payload.Role,
		},
	)
}
