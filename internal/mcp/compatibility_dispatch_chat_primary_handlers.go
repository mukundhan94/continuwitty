package mcp

import "context"

func registerChatPrimaryToolHandlers(handlers map[string]implementedToolHandler) {
	handlers["chat.send_message"] = chatSendMessageHandler()
	handlers["chat.continue_session"] = chatContinueSessionHandler()
}

func chatSendMessageHandler() implementedToolHandler {
	return func(
		service *CompatibilityService,
		ctx context.Context,
		actor Actor,
		params map[string]any,
	) (map[string]any, bool, *toolDispatchError) {
		return service.dispatchChatSendMessageTool(ctx, actor, params)
	}
}

func chatContinueSessionHandler() implementedToolHandler {
	return func(
		service *CompatibilityService,
		ctx context.Context,
		actor Actor,
		params map[string]any,
	) (map[string]any, bool, *toolDispatchError) {
		return service.dispatchChatContinueSessionTool(ctx, actor, params)
	}
}
