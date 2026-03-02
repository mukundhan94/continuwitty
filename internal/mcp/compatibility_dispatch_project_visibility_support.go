package mcp

import (
	"context"

	"engram/internal/models"
)

func (service *CompatibilityService) dispatchEngramShareTool(
	ctx context.Context,
	actor Actor,
	params map[string]any,
) (map[string]any, bool, *toolDispatchError) {
	return service.dispatchEngramVisibilityTool(
		ctx,
		actor,
		params,
		models.VisibilityScopeProject,
	)
}

func (service *CompatibilityService) dispatchEngramUnshareTool(
	ctx context.Context,
	actor Actor,
	params map[string]any,
) (map[string]any, bool, *toolDispatchError) {
	return service.dispatchEngramVisibilityTool(
		ctx,
		actor,
		params,
		models.VisibilityScopePrivate,
	)
}

func (service *CompatibilityService) dispatchEngramVisibilityTool(
	ctx context.Context,
	actor Actor,
	params map[string]any,
	scope models.VisibilityScope,
) (map[string]any, bool, *toolDispatchError) {
	if service.projectService == nil {
		return nil, false, nil
	}
	engramID, ok := requiredUUIDParam(params, "engram_id")
	if !ok {
		return nil, true, invalidParamError("engram_id")
	}
	var (
		updated *models.EngramVisibilityRecord
		err     error
	)
	if scope == models.VisibilityScopeProject {
		updated, err = service.projectService.ShareEngram(
			ctx,
			actor.UserID,
			normalizedActorRole(actor),
			engramID,
		)
	} else {
		updated, err = service.projectService.UnshareEngram(
			ctx,
			actor.UserID,
			normalizedActorRole(actor),
			engramID,
		)
	}
	if err != nil {
		return nil, true, mapProjectServiceError(err)
	}
	return map[string]any{"engram": updated}, true, nil
}
