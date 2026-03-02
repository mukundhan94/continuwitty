package main

import (
	"context"

	internalapi "engram/internal/api"
	"engram/internal/models"
)

func (adapter projectRouteServiceAdapter) ListProjects(
	ctx context.Context,
	request internalapi.ProjectListRouteRequest,
) ([]models.ProjectRecord, error) {
	return adapter.service.ListProjects(
		ctx,
		request.ActorUserID,
		request.ActorRole,
		request.IncludeArchived,
		request.Limit,
		request.Offset,
	)
}
