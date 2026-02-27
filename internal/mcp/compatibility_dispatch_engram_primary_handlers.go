package mcp

func registerEngramPrimaryToolHandlers(handlers map[string]implementedToolHandler) {
	handlers["engram.create"] = bindEngramDispatch((*CompatibilityService).dispatchEngramCreateTool)
	handlers["engram.create_from_conversation"] = bindEngramDispatch(
		(*CompatibilityService).dispatchEngramCreateFromConversationTool,
	)
	handlers["engram.collection_create"] = bindEngramDispatch((*CompatibilityService).dispatchEngramCollectionCreateTool)
	handlers["engram.update"] = bindEngramDispatch((*CompatibilityService).dispatchEngramUpdateTool)
	handlers["engram.move_project"] = bindEngramDispatch((*CompatibilityService).dispatchEngramMoveProjectTool)
}
