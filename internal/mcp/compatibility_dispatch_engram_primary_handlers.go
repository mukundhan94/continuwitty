package mcp

func registerEngramPrimaryToolHandlers(handlers map[string]implementedToolHandler) {
	handlers["engram.create"] = bindEngramDispatch((*CompatibilityService).dispatchEngramCreateTool)
}
