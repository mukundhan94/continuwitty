package mcp

func registerEngramMutationToolHandlers(handlers map[string]implementedToolHandler) {
	handlers["engram.collection_update"] = bindEngramDispatch((*CompatibilityService).dispatchEngramCollectionUpdateTool)
	handlers["engram.collection_delete"] = bindEngramDispatch((*CompatibilityService).dispatchEngramCollectionDeleteTool)
	handlers["engram.collection_add_items"] = bindEngramDispatch(
		(*CompatibilityService).dispatchEngramCollectionAddItemsTool,
	)
	handlers["engram.collection_remove_items"] = bindEngramDispatch(
		(*CompatibilityService).dispatchEngramCollectionRemoveItemTool,
	)
	handlers["engram.delete"] = bindEngramDispatch((*CompatibilityService).dispatchEngramDeleteTool)
	handlers["engram.restore"] = bindEngramDispatch((*CompatibilityService).dispatchEngramRestoreTool)
}
