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
