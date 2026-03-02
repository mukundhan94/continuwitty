package main

import (
	"context"

	internalapi "engram/internal/api"
	"engram/internal/models"
	"engram/internal/projects"
)

func (adapter projectRouteServiceAdapter) ListProjectMembers(
	ctx context.Context,
	request internalapi.ProjectMemberListRouteRequest,
) ([]models.ProjectMemberRecord, error) {
	return adapter.service.ListProjectMembers(
		ctx,
		projects.ActorContext{
			UserID: request.ActorUserID,
			Role:   request.ActorRole,
		},
		projects.ProjectMemberListRequest{
			ProjectID:      request.ProjectID,
			IncludeRevoked: request.IncludeRevoked,
			Limit:          request.Limit,
			Offset:         request.Offset,
		},
	)
}
