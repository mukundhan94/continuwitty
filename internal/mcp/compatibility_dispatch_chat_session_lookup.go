package mcp

import (
	"context"

	"engram/internal/models"
)

func (service *CompatibilityService) lookupChatSession(
	ctx context.Context,
	actor Actor,
	params map[string]any,
) (*models.ChatSessionRecord, bool, *toolDispatchError) {
	if service.sessionGet == nil {
		return nil, false, nil
	}
	sessionID, ok := requiredUUIDParam(params, "session_id")
	if !ok {
		return nil, true, invalidParamError("session_id")
	}
	session, err := service.sessionGet.GetSession(
		ctx,
		SessionGetRequest{
			ActorUserID: actor.UserID,
			SessionID:   sessionID,
		},
	)
	if err != nil {
		return nil, true, internalToolDispatchError()
	}
	if session == nil {
		return nil, true, invalidParamsWithStatus(404, "Chat session not found")
	}
	return session, true, nil
}

func (service *CompatibilityService) dispatchSessionPayloadTool(
	ctx context.Context,
	actor Actor,
	params map[string]any,
	buildPayload func(models.ChatSessionRecord) map[string]any,
) (map[string]any, bool, *toolDispatchError) {
	session, handled, dispatchErr := service.lookupChatSession(ctx, actor, params)
	if dispatchErr != nil || !handled {
		return nil, handled, dispatchErr
	}
	return buildPayload(*session), true, nil
}
