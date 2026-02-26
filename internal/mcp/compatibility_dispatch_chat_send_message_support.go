package mcp

import (
	"context"
	"errors"

	"engram/internal/chat"
)

func (service *CompatibilityService) dispatchChatSendMessageTool(
	ctx context.Context,
	actor Actor,
	params map[string]any,
) (map[string]any, bool, *toolDispatchError) {
	if service.messageSend == nil {
		return nil, false, nil
	}
	sessionID, ok := requiredUUIDParam(params, "session_id")
	if !ok {
		return nil, true, invalidParamError("session_id")
	}
	response, err := service.messageSend.SendMessage(
		ctx,
		SessionMessageSendRequest{
			ActorUserID: actor.UserID,
			SessionID:   sessionID,
			ContentText: stringParamWithDefault(params, "content_text", ""),
		},
	)
	if err != nil {
		return nil, true, mapChatSendError(err)
	}
	if response == nil {
		return nil, true, internalToolDispatchError()
	}
	return map[string]any{"message": *response}, true, nil
}

func mapChatSendError(err error) *toolDispatchError {
	var providerErr *chat.ChatProviderExecutionError
	if errors.As(err, &providerErr) {
		return &toolDispatchError{
			code:    -32020,
			message: providerErr.Detail(),
			data: map[string]any{
				"status_code": providerErr.StatusCode(),
				"error_code":  providerErr.ErrorCode(),
			},
		}
	}
	var serviceErr *chat.ChatServiceError
	if errors.As(err, &serviceErr) {
		return &toolDispatchError{
			code:    -32010,
			message: serviceErr.Detail(),
			data: map[string]any{
				"status_code": serviceErr.StatusCode(),
			},
		}
	}
	return internalToolDispatchError()
}
