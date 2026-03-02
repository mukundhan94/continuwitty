package mcp

import "context"

func registerEngramReadToolHandlers(handlers map[string]implementedToolHandler) {
	handlers["engram.list"] = bindEngramDispatch((*CompatibilityService).dispatchEngramListTool)
	handlers["engram.get"] = bindEngramDispatch((*CompatibilityService).dispatchEngramGetTool)
	handlers["engram.query"] = bindEngramDispatch((*CompatibilityService).dispatchEngramQueryTool)
	handlers["engram.rehydrate"] = bindEngramDispatch((*CompatibilityService).dispatchEngramRehydrateTool)
	handlers["engram.link_list"] = bindEngramDispatch((*CompatibilityService).dispatchEngramLinkListTool)
	handlers["engram.link_suggest"] = bindEngramDispatch((*CompatibilityService).dispatchEngramLinkSuggestTool)
	handlers["engram.trace_path"] = bindEngramDispatch((*CompatibilityService).dispatchEngramTracePathTool)
	handlers["engram.collection_list"] = bindEngramDispatch((*CompatibilityService).dispatchEngramCollectionListTool)
}

type engramDispatchFunc func(
	service *CompatibilityService,
	ctx context.Context,
	actor Actor,
	params map[string]any,
) (map[string]any, bool, *toolDispatchError)

func bindEngramDispatch(dispatch engramDispatchFunc) implementedToolHandler {
	return func(
		service *CompatibilityService,
		ctx context.Context,
		actor Actor,
		params map[string]any,
	) (map[string]any, bool, *toolDispatchError) {
		return dispatch(service, ctx, actor, params)
	}
}
