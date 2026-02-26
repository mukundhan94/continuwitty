package mcp

func registerEngramToolHandlers(handlers map[string]implementedToolHandler) {
	registerEngramPrimaryToolHandlers(handlers)
	registerEngramReadToolHandlers(handlers)
	registerEngramMutationToolHandlers(handlers)
}
