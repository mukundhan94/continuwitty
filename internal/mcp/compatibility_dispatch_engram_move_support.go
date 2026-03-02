package mcp

import (
	"context"
	"errors"

	"engram/internal/admin"
	"engram/internal/projects"

	"github.com/google/uuid"
)

func (service *CompatibilityService) dispatchEngramMoveProjectTool(
	ctx context.Context,
	actor Actor,
	params map[string]any,
) (map[string]any, bool, *toolDispatchError) {
	if service.engramMove == nil {
		return nil, false, nil
	}
	request, dispatchErr := parseEngramMoveRequest(actor, params)
	if dispatchErr != nil {
		return nil, true, dispatchErr
	}
	moved, err := service.engramMove.MoveEngram(ctx, request)
	if err != nil {
		return nil, true, mapEngramMoveError(err, request.EngramID)
	}
	if moved == nil {
		return nil, true, engramNotFoundDispatchError(request.EngramID)
	}
	return map[string]any{"engram": *moved}, true, nil
}

func parseEngramMoveRequest(
	actor Actor,
	params map[string]any,
) (EngramMoveRequest, *toolDispatchError) {
	engramID, ok := requiredUUIDParam(params, "engram_id")
	if !ok {
		return EngramMoveRequest{}, invalidParamError("engram_id")
	}
	reason, ok := optionalStringPointerParam(params, "reason")
	if !ok {
		return EngramMoveRequest{}, invalidParamError("reason")
	}
	expectedUpdatedAt, ok := optionalRFC3339TimePointerParam(params, "expected_updated_at")
	if !ok {
		return EngramMoveRequest{}, invalidParamError("expected_updated_at")
	}
	return EngramMoveRequest{
		ActorUserID:       actor.UserID,
		ActorRole:         normalizedActorRole(actor),
		EngramID:          engramID,
		TargetProjectID:   stringParamWithDefault(params, "target_project_id", ""),
		Reason:            reason,
		ExpectedUpdatedAt: expectedUpdatedAt,
	}, nil
}

func mapEngramMoveError(err error, engramID uuid.UUID) *toolDispatchError {
	switch {
	case errors.Is(err, admin.ErrEngramNotFound):
		return engramNotFoundDispatchError(engramID)
	case errors.Is(err, admin.ErrEngramStale):
		return invalidParamsWithStatus(409, admin.ErrEngramStale.Error())
	case errors.Is(err, projects.ErrProjectIDMustNotBeBlank):
		return invalidParamError("target_project_id")
	case errors.Is(err, projects.ErrProjectNotFound):
		return invalidParamsWithStatus(404, "Project not found")
	case errors.Is(err, projects.ErrProjectIDRequiredWhenNoDefaultProject):
		return invalidParamsWithStatus(422, projects.ErrProjectIDRequiredWhenNoDefaultProject.Error())
	case errors.Is(err, projects.ErrDefaultProjectNotAccessible):
		return invalidParamsWithStatus(422, projects.ErrDefaultProjectNotAccessible.Error())
	default:
		return internalToolDispatchError()
	}
}
