package mcp

import (
	"context"

	"engram/internal/models"
)

func (service *CompatibilityService) dispatchChatContinueSessionTool(
	ctx context.Context,
	actor Actor,
	params map[string]any,
) (map[string]any, bool, *toolDispatchError) {
	if service.sessionContinue == nil {
		return nil, false, nil
	}
	sessionID, ok := requiredUUIDParam(params, "session_id")
	if !ok {
		return nil, true, invalidParamError("session_id")
	}
	title, ok := optionalStringPointerParam(params, "title")
	if !ok {
		return nil, true, invalidParamError("title")
	}
	continued, err := service.sessionContinue.ContinueSession(
		ctx,
		SessionContinueRequest{
			ActorUserID: actor.UserID,
			SessionID:   sessionID,
			Payload: models.ContinueSessionRequest{
				Title: title,
			},
		},
	)
	if err != nil {
		return nil, true, internalToolDispatchError()
	}
	if continued == nil {
		return nil, true, invalidParamsWithStatus(404, "Chat session not found")
	}
	return map[string]any{"continuation": *continued}, true, nil
}

func optionalStringPointerParam(params map[string]any, key string) (*string, bool) {
	value, found := optionalParamValue(params, key)
	if !found {
		return nil, true
	}
	text, ok := value.(string)
	if !ok {
		return nil, false
	}
	return &text, true
}
