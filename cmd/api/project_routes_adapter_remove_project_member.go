package main

import (
	"context"

	internalapi "engram/internal/api"
	"engram/internal/projects"
)

func (adapter projectRouteServiceAdapter) RemoveProjectMember(
	ctx context.Context,
	request internalapi.ProjectMemberDeleteRouteRequest,
) error {
	return adapter.service.RemoveProjectMember(
		ctx,
		request.ActorUserID,
		request.ActorRole,
		projects.ProjectMemberRemoveRequest{
			ProjectID: request.ProjectID,
			UserID:    request.UserID,
		},
	)
}
