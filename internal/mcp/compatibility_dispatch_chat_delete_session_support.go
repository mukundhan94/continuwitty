package mcp

import "context"

func (service *CompatibilityService) dispatchChatDeleteSessionTool(
	ctx context.Context,
	actor Actor,
	params map[string]any,
) (map[string]any, bool, *toolDispatchError) {
	if service.sessionDelete == nil {
		return nil, false, nil
	}
	sessionID, ok := requiredUUIDParam(params, "session_id")
	if !ok {
		return nil, true, invalidParamError("session_id")
	}
	deleteLinkedEngrams, ok := optionalBoolParam(params, "delete_linked_engrams", false)
	if !ok {
		return nil, true, invalidParamError("delete_linked_engrams")
	}
	reason, ok := optionalStringPointerParam(params, "reason")
	if !ok {
		return nil, true, invalidParamError("reason")
	}
	deleted, err := service.sessionDelete.DeleteSession(
		ctx,
		SessionDeleteRequest{
			ActorUserID:         actor.UserID,
			SessionID:           sessionID,
			DeleteLinkedEngrams: deleteLinkedEngrams,
			Reason:              reason,
		},
	)
	if err != nil {
		return nil, true, internalToolDispatchError()
	}
	if deleted == nil {
		return nil, true, invalidParamsWithStatus(404, "Session not found")
	}
	return map[string]any{"result": *deleted}, true, nil
}
