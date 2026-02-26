package mcp

import "context"

func registerChatPrimaryToolHandlers(handlers map[string]implementedToolHandler) {
	handlers["chat.continue_session"] = chatContinueSessionHandler()
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
