package mcp

import (
	"context"

	"engram/internal/models"
	"engram/internal/projects"

	"github.com/google/uuid"
)

type projectMemberMutationParams struct {
	projectID string
	userID    uuid.UUID
	role      models.ProjectMemberRole
}

func (service *CompatibilityService) dispatchProjectMemberListTool(
	ctx context.Context,
	actor Actor,
	params map[string]any,
) (map[string]any, bool, *toolDispatchError) {
	if service.projectService == nil {
		return nil, false, nil
	}
	projectID, ok := requiredStringParam(params, "project_id")
	if !ok {
		return nil, true, invalidParamError("project_id")
	}
	includeRevoked, ok := optionalBoolParam(params, "include_revoked", false)
	if !ok {
		return nil, true, invalidParamError("include_revoked")
	}
	limit, ok := optionalIntParam(params, "limit", defaultProjectListLimit)
	if !ok {
		return nil, true, invalidParamError("limit")
	}
	offset, ok := optionalIntParam(params, "offset", defaultProjectListOffset)
	if !ok {
		return nil, true, invalidParamError("offset")
	}
	members, err := service.projectService.ListProjectMembers(
		ctx,
		projects.ActorContext{
			UserID: actor.UserID,
			Role:   normalizedActorRole(actor),
		},
		projects.ProjectMemberListRequest{
			ProjectID:      projectID,
			IncludeRevoked: includeRevoked,
			Limit:          limit,
			Offset:         offset,
		},
	)
	if err != nil {
		return nil, true, mapProjectServiceError(err)
	}
	return map[string]any{"members": members}, true, nil
}

func (service *CompatibilityService) dispatchProjectMemberAddTool(
	ctx context.Context,
	actor Actor,
	params map[string]any,
) (map[string]any, bool, *toolDispatchError) {
	return service.dispatchProjectMemberRoleMutationTool(ctx, actor, params, false)
}

func (service *CompatibilityService) dispatchProjectMemberUpdateTool(
	ctx context.Context,
	actor Actor,
	params map[string]any,
) (map[string]any, bool, *toolDispatchError) {
	return service.dispatchProjectMemberRoleMutationTool(ctx, actor, params, true)
}

func (service *CompatibilityService) dispatchProjectMemberRoleMutationTool(
	ctx context.Context,
	actor Actor,
	params map[string]any,
	update bool,
) (map[string]any, bool, *toolDispatchError) {
	if service.projectService == nil {
		return nil, false, nil
	}
	parsed, parseErr := parseProjectMemberMutationParams(params)
	if parseErr != nil {
		return nil, true, parseErr
	}
	var (
		member *models.ProjectMemberRecord
		err    error
	)
	if update {
		member, err = service.projectService.UpdateProjectMember(
			ctx,
			actor.UserID,
			normalizedActorRole(actor),
			projects.ProjectMemberUpdateRequest{
				ProjectID: parsed.projectID,
				UserID:    parsed.userID,
				Role:      parsed.role,
			},
		)
	} else {
		member, err = service.projectService.AddProjectMember(
			ctx,
			actor.UserID,
			normalizedActorRole(actor),
			projects.ProjectMemberCreateRequest{
				ProjectID: parsed.projectID,
				UserID:    parsed.userID,
				Role:      parsed.role,
			},
		)
	}
	if err != nil {
		return nil, true, mapProjectServiceError(err)
	}
	return map[string]any{"member": member}, true, nil
}

func (service *CompatibilityService) dispatchProjectMemberRemoveTool(
	ctx context.Context,
	actor Actor,
	params map[string]any,
) (map[string]any, bool, *toolDispatchError) {
	if service.projectService == nil {
		return nil, false, nil
	}
	projectID, ok := requiredStringParam(params, "project_id")
	if !ok {
		return nil, true, invalidParamError("project_id")
	}
	userID, ok := requiredUUIDParam(params, "user_id")
	if !ok {
		return nil, true, invalidParamError("user_id")
	}
	if err := service.projectService.RemoveProjectMember(
		ctx,
		actor.UserID,
		normalizedActorRole(actor),
		projects.ProjectMemberRemoveRequest{
			ProjectID: projectID,
			UserID:    userID,
		},
	); err != nil {
		return nil, true, mapProjectServiceError(err)
	}
	return map[string]any{"removed": true}, true, nil
}

func parseProjectMemberMutationParams(params map[string]any) (projectMemberMutationParams, *toolDispatchError) {
	projectID, ok := requiredStringParam(params, "project_id")
	if !ok {
		return projectMemberMutationParams{}, invalidParamError("project_id")
	}
	userID, ok := requiredUUIDParam(params, "user_id")
	if !ok {
		return projectMemberMutationParams{}, invalidParamError("user_id")
	}
	role, ok := requiredProjectMemberRoleParam(params, "role")
	if !ok {
		return projectMemberMutationParams{}, invalidParamError("role")
	}
	return projectMemberMutationParams{
		projectID: projectID,
		userID:    userID,
		role:      role,
	}, nil
}
