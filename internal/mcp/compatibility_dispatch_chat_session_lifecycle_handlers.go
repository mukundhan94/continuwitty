package mcp

import "context"

func registerChatSessionLifecycleToolHandlers(handlers map[string]implementedToolHandler) {
	handlers["chat.delete_session"] = chatDeleteSessionHandler()
	handlers["chat.restore_session"] = chatRestoreSessionHandler()
}

func chatDeleteSessionHandler() implementedToolHandler {
	return func(
		service *CompatibilityService,
		ctx context.Context,
		actor Actor,
		params map[string]any,
	) (map[string]any, bool, *toolDispatchError) {
		return service.dispatchChatDeleteSessionTool(ctx, actor, params)
	}
}

func chatRestoreSessionHandler() implementedToolHandler {
	return func(
		service *CompatibilityService,
		ctx context.Context,
		actor Actor,
		params map[string]any,
	) (map[string]any, bool, *toolDispatchError) {
		return service.dispatchChatRestoreSessionTool(ctx, actor, params)
	}
}
