package main

import (
	"context"

	internalapi "engram/internal/api"
	"engram/internal/models"
	"engram/internal/projects"

	"github.com/google/uuid"
)

type projectRouteServiceAdapter struct {
	service *projects.Service
}

func newProjectRouteServiceAdapter(service *projects.Service) internalapi.ProjectService {
	if service == nil {
		return nil
	}
	return projectRouteServiceAdapter{service: service}
}

func (adapter projectRouteServiceAdapter) CreateProject(
	ctx context.Context,
	request internalapi.ProjectCreateRouteRequest,
) (*models.ProjectRecord, error) {
	return adapter.service.CreateProject(
		ctx,
		request.ActorUserID,
		request.ActorRole,
		projects.CreateProjectRequest{
			ProjectID:   request.Payload.ProjectID,
			Name:        request.Payload.Name,
			Description: request.Payload.Description,
			OwnerUserID: request.Payload.OwnerUserID,
		},
	)
}

func (adapter projectRouteServiceAdapter) GetDefaultProjectID(
	ctx context.Context,
	actorUserID uuid.UUID,
) (*string, error) {
	return adapter.service.GetDefaultProjectID(ctx, actorUserID)
}

func (adapter projectRouteServiceAdapter) SetDefaultProjectID(
	ctx context.Context,
	request internalapi.ProjectDefaultUpdateRouteRequest,
) (string, error) {
	return adapter.service.SetDefaultProjectID(
		ctx,
		request.ActorUserID,
		request.ActorRole,
		request.ProjectID,
	)
}
