package mcp

import "context"

func (service *CompatibilityService) dispatchChatUnpinEngramTool(
	ctx context.Context,
	actor Actor,
	params map[string]any,
) (map[string]any, bool, *toolDispatchError) {
	if service.unpinEngramService == nil {
		return nil, false, nil
	}
	sessionID, ok := requiredUUIDParam(params, "session_id")
	if !ok {
		return nil, true, invalidParamError("session_id")
	}
	engramID, ok := requiredUUIDParam(params, "engram_id")
	if !ok {
		return nil, true, invalidParamError("engram_id")
	}
	removed, err := service.unpinEngramService.UnpinEngram(
		ctx,
		SessionPinEngramRequest{
			ActorUserID: actor.UserID,
			SessionID:   sessionID,
			EngramID:    engramID,
		},
	)
	if err != nil {
		return nil, true, internalToolDispatchError()
	}
	if !removed {
		return nil, true, invalidParamsWithStatus(404, "Pinned engram not found for session")
	}
	return map[string]any{"removed": true}, true, nil
}
