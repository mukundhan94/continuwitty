package mcp

import "context"

func registerChatMutationToolHandlers(handlers map[string]implementedToolHandler) {
	handlers["chat.pin_engram"] = chatPinEngramHandler()
	handlers["chat.unpin_engram"] = chatUnpinEngramHandler()
}

func chatPinEngramHandler() implementedToolHandler {
	return func(
		service *CompatibilityService,
		ctx context.Context,
		actor Actor,
		params map[string]any,
	) (map[string]any, bool, *toolDispatchError) {
		return service.dispatchChatPinEngramTool(ctx, actor, params)
	}
}

func chatUnpinEngramHandler() implementedToolHandler {
	return func(
		service *CompatibilityService,
		ctx context.Context,
		actor Actor,
		params map[string]any,
	) (map[string]any, bool, *toolDispatchError) {
		return service.dispatchChatUnpinEngramTool(ctx, actor, params)
	}
}
