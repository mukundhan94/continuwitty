package mcp

import "context"

func (service *CompatibilityService) dispatchChatPinEngramTool(
	ctx context.Context,
	actor Actor,
	params map[string]any,
) (map[string]any, bool, *toolDispatchError) {
	if service.pinEngramService == nil {
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
	pinned, err := service.pinEngramService.PinEngram(
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
	if pinned == nil {
		return nil, true, invalidParamsWithStatus(404, "Engram not found or session inaccessible")
	}
	return map[string]any{"pinned": *pinned}, true, nil
}
