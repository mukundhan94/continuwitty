package mcp

import "context"

func registerEngramReadToolHandlers(handlers map[string]implementedToolHandler) {
	handlers["engram.list"] = engramListHandler()
	handlers["engram.get"] = engramGetHandler()
	handlers["engram.collection_list"] = engramCollectionListHandler()
}

func engramListHandler() implementedToolHandler {
	return func(
		service *CompatibilityService,
		ctx context.Context,
		actor Actor,
		params map[string]any,
	) (map[string]any, bool, *toolDispatchError) {
		return service.dispatchEngramListTool(ctx, actor, params)
	}
}

func engramGetHandler() implementedToolHandler {
	return func(
		service *CompatibilityService,
		ctx context.Context,
		actor Actor,
		params map[string]any,
	) (map[string]any, bool, *toolDispatchError) {
		return service.dispatchEngramGetTool(ctx, actor, params)
	}
}

func engramCollectionListHandler() implementedToolHandler {
	return func(
		service *CompatibilityService,
		ctx context.Context,
		actor Actor,
		params map[string]any,
	) (map[string]any, bool, *toolDispatchError) {
		return service.dispatchEngramCollectionListTool(ctx, actor, params)
	}
}
