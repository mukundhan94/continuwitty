package main

import (
	"context"

	internalapi "engram/internal/api"
	"engram/internal/models"
)

func (adapter projectRouteServiceAdapter) ListProjectMembers(
	ctx context.Context,
	request internalapi.ProjectMemberListRouteRequest,
) ([]models.ProjectMemberRecord, error) {
	return adapter.service.ListProjectMembers(
		ctx,
		request.ActorUserID,
		request.ActorRole,
		request.ProjectID,
		request.IncludeRevoked,
		request.Limit,
		request.Offset,
	)
}
