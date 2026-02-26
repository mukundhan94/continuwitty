package mcp

func registerChatToolHandlers(handlers map[string]implementedToolHandler) {
	registerChatSessionToolHandlers(handlers)
	registerChatCollectionToolHandlers(handlers)
	registerChatPrimaryToolHandlers(handlers)
	registerChatMutationToolHandlers(handlers)
}
