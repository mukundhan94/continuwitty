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
	handlers["engram.link_update"] = bindEngramDispatch((*CompatibilityService).dispatchEngramLinkUpdateTool)
	handlers["engram.link_archive"] = bindEngramDispatch((*CompatibilityService).dispatchEngramLinkArchiveTool)
	handlers["engram.feedback"] = bindEngramDispatch((*CompatibilityService).dispatchEngramFeedbackTool)
	handlers["engram.refresh_freshness"] = bindEngramDispatch((*CompatibilityService).dispatchEngramRefreshFreshnessTool)
	handlers["engram.refresh_consolidation"] = bindEngramDispatch((*CompatibilityService).dispatchEngramRefreshConsolidationTool)
	handlers["engram.consolidation_action"] = bindEngramDispatch((*CompatibilityService).dispatchEngramConsolidationActionTool)
	handlers["engram.delete"] = bindEngramDispatch((*CompatibilityService).dispatchEngramDeleteTool)
	handlers["engram.restore"] = bindEngramDispatch((*CompatibilityService).dispatchEngramRestoreTool)
}
