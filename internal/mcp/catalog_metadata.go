package mcp

type toolCatalogEntry struct {
	description string
	inputSchema map[string]any
}

func resolvedCatalogEntry(toolName string) toolCatalogEntry {
	entry, exists := toolCatalogEntries[toolName]
	if !exists {
		return defaultCatalogEntry(toolName)
	}
	description := entry.description
	if description == "" {
		description = "MCP tool: " + toolName
	}
	return toolCatalogEntry{
		description: description,
		inputSchema: cloneAnyMap(entry.inputSchema),
	}
}

func defaultCatalogEntry(toolName string) toolCatalogEntry {
	return toolCatalogEntry{
		description: "MCP tool: " + toolName,
		inputSchema: map[string]any{
			"type":       "object",
			"properties": map[string]any{},
		},
	}
}

func cloneAnyMap(source map[string]any) map[string]any {
	if source == nil {
		return map[string]any{}
	}
	cloned := make(map[string]any, len(source))
	for key, value := range source {
		cloned[key] = cloneAnyValue(value)
	}
	return cloned
}

func cloneAnySlice(source []any) []any {
	cloned := make([]any, len(source))
	for index, item := range source {
		cloned[index] = cloneAnyValue(item)
	}
	return cloned
}

func cloneAnyValue(value any) any {
	switch typed := value.(type) {
	case map[string]any:
		return cloneAnyMap(typed)
	case []any:
		return cloneAnySlice(typed)
	case []string:
		cloned := make([]string, len(typed))
		copy(cloned, typed)
		return cloned
	default:
		return typed
	}
}
