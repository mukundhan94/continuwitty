package mcp

func registerEngramMutationToolHandlers(handlers map[string]implementedToolHandler) {
	handlers["engram.delete"] = bindEngramDispatch((*CompatibilityService).dispatchEngramDeleteTool)
	handlers["engram.restore"] = bindEngramDispatch((*CompatibilityService).dispatchEngramRestoreTool)
}
