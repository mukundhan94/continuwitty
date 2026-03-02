package main

import (
	"context"

	internalapi "engram/internal/api"
	"engram/internal/models"
	"engram/internal/projects"
)

func (adapter projectRouteServiceAdapter) ListProjects(
	ctx context.Context,
	request internalapi.ProjectListRouteRequest,
) ([]models.ProjectRecord, error) {
	return adapter.service.ListProjects(
		ctx,
		projects.ActorContext{
			UserID: request.ActorUserID,
			Role:   request.ActorRole,
		},
		projects.ProjectListRequest{
			IncludeArchived: request.IncludeArchived,
			Limit:           request.Limit,
			Offset:          request.Offset,
		},
	)
}
