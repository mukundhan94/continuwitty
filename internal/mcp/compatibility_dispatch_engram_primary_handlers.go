package mcp

func registerEngramPrimaryToolHandlers(handlers map[string]implementedToolHandler) {
	handlers["engram.create"] = bindEngramDispatch((*CompatibilityService).dispatchEngramCreateTool)
	handlers["engram.create_from_conversation"] = bindEngramDispatch(
		(*CompatibilityService).dispatchEngramCreateFromConversationTool,
	)
}
