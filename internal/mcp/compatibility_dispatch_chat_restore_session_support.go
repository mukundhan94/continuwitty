package mcp

import "context"

func (service *CompatibilityService) dispatchChatRestoreSessionTool(
	ctx context.Context,
	actor Actor,
	params map[string]any,
) (map[string]any, bool, *toolDispatchError) {
	if service.sessionRestore == nil {
		return nil, false, nil
	}
	sessionID, ok := requiredUUIDParam(params, "session_id")
	if !ok {
		return nil, true, invalidParamError("session_id")
	}
	restored, err := service.sessionRestore.RestoreSession(
		ctx,
		SessionRestoreRequest{
			ActorUserID: actor.UserID,
			ActorRole:   normalizedActorRole(actor),
			SessionID:   sessionID,
		},
	)
	if err != nil {
		return nil, true, internalToolDispatchError()
	}
	if restored == nil {
		return nil, true, invalidParamsWithStatus(404, "Session not found")
	}
	return map[string]any{"result": *restored}, true, nil
}
