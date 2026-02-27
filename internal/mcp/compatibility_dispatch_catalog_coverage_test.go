package mcp

import "testing"

func TestImplementedToolHandlersCoverToolCatalogOrder(t *testing.T) {
	for _, toolName := range toolCatalogOrder {
		canonicalName := canonicalToolName(toolName)
		if _, exists := implementedToolHandlers[canonicalName]; exists {
			continue
		}
		t.Fatalf("missing implemented handler for tool %q (canonical=%q)", toolName, canonicalName)
	}
}
