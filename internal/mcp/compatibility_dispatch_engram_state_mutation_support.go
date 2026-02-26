package mcp

import (
	"context"

	"github.com/google/uuid"
)

func (service *CompatibilityService) dispatchEngramDeleteTool(
	ctx context.Context,
	actor Actor,
	params map[string]any,
) (map[string]any, bool, *toolDispatchError) {
	if service.engramDelete == nil {
		return nil, false, nil
	}
	engramID, ok := requiredUUIDParam(params, "engram_id")
	if !ok {
		return nil, true, invalidParamError("engram_id")
	}
	reason, ok := optionalStringPointerParam(params, "reason")
	if !ok {
		return nil, true, invalidParamError("reason")
	}
	deleted, err := service.engramDelete.DeleteEngram(
		ctx,
		EngramDeleteRequest{
			ActorUserID: actor.UserID,
			ActorRole:   normalizedActorRole(actor),
			EngramID:    engramID,
			Reason:      reason,
		},
	)
	if err != nil {
		return nil, true, internalToolDispatchError()
	}
	if deleted == nil {
		return nil, true, engramNotFoundDispatchError(engramID)
	}
	return map[string]any{"result": *deleted}, true, nil
}

func (service *CompatibilityService) dispatchEngramRestoreTool(
	ctx context.Context,
	actor Actor,
	params map[string]any,
) (map[string]any, bool, *toolDispatchError) {
	if service.engramRestore == nil {
		return nil, false, nil
	}
	engramID, ok := requiredUUIDParam(params, "engram_id")
	if !ok {
		return nil, true, invalidParamError("engram_id")
	}
	restored, err := service.engramRestore.RestoreEngram(
		ctx,
		EngramRestoreRequest{
			ActorUserID: actor.UserID,
			ActorRole:   normalizedActorRole(actor),
			EngramID:    engramID,
		},
	)
	if err != nil {
		return nil, true, internalToolDispatchError()
	}
	if restored == nil {
		return nil, true, engramNotFoundDispatchError(engramID)
	}
	return map[string]any{"result": *restored}, true, nil
}

func engramNotFoundDispatchError(engramID uuid.UUID) *toolDispatchError {
	return &toolDispatchError{
		code:    -32004,
		message: "Engram not found",
		data: map[string]any{
			"engram_id": engramID.String(),
		},
	}
}
