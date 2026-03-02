package mcp

import (
	"context"

	"engram/internal/projects"
)

func (service *CompatibilityService) dispatchProjectListTool(
	ctx context.Context,
	actor Actor,
	params map[string]any,
) (map[string]any, bool, *toolDispatchError) {
	if service.projectService == nil {
		return nil, false, nil
	}
	includeArchived, ok := optionalBoolParam(params, "include_archived", false)
	if !ok {
		return nil, true, invalidParamError("include_archived")
	}
	limit, ok := optionalIntParam(params, "limit", defaultProjectListLimit)
	if !ok {
		return nil, true, invalidParamError("limit")
	}
	offset, ok := optionalIntParam(params, "offset", defaultProjectListOffset)
	if !ok {
		return nil, true, invalidParamError("offset")
	}

	projectRows, err := service.projectService.ListProjects(
		ctx,
		projects.ActorContext{
			UserID: actor.UserID,
			Role:   normalizedActorRole(actor),
		},
		projects.ProjectListRequest{
			IncludeArchived: includeArchived,
			Limit:           limit,
			Offset:          offset,
		},
	)
	if err != nil {
		return nil, true, internalToolDispatchError()
	}
	return map[string]any{"projects": projectRows}, true, nil
}

func (service *CompatibilityService) dispatchProjectCreateTool(
	ctx context.Context,
	actor Actor,
	params map[string]any,
) (map[string]any, bool, *toolDispatchError) {
	if service.projectService == nil {
		return nil, false, nil
	}
	ownerUserID, ok := optionalUUIDParam(params, "owner_user_id")
	if !ok {
		return nil, true, invalidParamError("owner_user_id")
	}
	created, err := service.projectService.CreateProject(
		ctx,
		actor.UserID,
		normalizedActorRole(actor),
		projects.CreateProjectRequest{
			ProjectID:   stringParamWithDefault(params, "project_id", ""),
			Name:        stringParamWithDefault(params, "name", ""),
			Description: stringParamWithDefault(params, "description", ""),
			OwnerUserID: ownerUserID,
		},
	)
	if err != nil {
		return nil, true, mapProjectCreateError(err)
	}
	return map[string]any{"project": created}, true, nil
}

func (service *CompatibilityService) dispatchProjectGetDefaultTool(
	ctx context.Context,
	actor Actor,
) (map[string]any, bool, *toolDispatchError) {
	if service.projectService == nil {
		return nil, false, nil
	}
	defaultProjectID, err := service.projectService.GetDefaultProjectID(ctx, actor.UserID)
	if err != nil {
		return nil, true, internalToolDispatchError()
	}
	return map[string]any{"default_project_id": optionalString(defaultProjectID)}, true, nil
}

func (service *CompatibilityService) dispatchProjectSetDefaultTool(
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
	defaultProjectID, err := service.projectService.SetDefaultProjectID(
		ctx,
		actor.UserID,
		normalizedActorRole(actor),
		projectID,
	)
	if err != nil {
		return nil, true, mapProjectServiceError(err)
	}
	return map[string]any{"default_project_id": defaultProjectID}, true, nil
}
