package main

import (
	"context"

	internalapi "engram/internal/api"
	"engram/internal/models"
	"engram/internal/projects"
)

func (adapter projectRouteServiceAdapter) AddProjectMember(
	ctx context.Context,
	request internalapi.ProjectMemberCreateRouteRequest,
) (*models.ProjectMemberRecord, error) {
	return adapter.service.AddProjectMember(
		ctx,
		request.ActorUserID,
		request.ActorRole,
		projects.ProjectMemberCreateRequest{
			ProjectID: request.ProjectID,
			UserID:    request.Payload.UserID,
			Role:      request.Payload.Role,
		},
	)
}
