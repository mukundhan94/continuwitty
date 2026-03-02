package main

import (
	"context"

	internalapi "engram/internal/api"
	"engram/internal/models"
	"engram/internal/projects"
)

func (adapter projectRouteServiceAdapter) ListProjectAuditEvents(
	ctx context.Context,
	request internalapi.ProjectAuditListRouteRequest,
) ([]models.ProjectAuditEventRecord, error) {
	return adapter.service.ListProjectAuditEvents(
		ctx,
		request.ActorUserID,
		request.ActorRole,
		projects.ProjectAuditListRequest{
			ProjectID: request.ProjectID,
			Limit:     request.Limit,
			Offset:    request.Offset,
		},
	)
}
