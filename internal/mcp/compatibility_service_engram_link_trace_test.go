package mcp

import (
	"testing"

	"engram/internal/models"
)

func TestCompatibilityServiceEngramTracePathParity(t *testing.T) {
	fixture := newEngramLinkQueryFixture()
	traceFrame := runCompatibilityRequestWithService(
		t,
		fixture.service,
		directToolRequest(
			fixture.actorUserID.String(),
			"engram.trace_path",
			map[string]any{
				"engram_id":     fixture.sourceEngramID.String(),
				"max_depth":     3,
				"max_neighbors": 8,
			},
		),
	)
	steps := engramLinkFrameValue[[]models.EngramLinkTraversalStep](t, traceFrame, false, "steps")
	if len(steps) != 1 {
		t.Fatalf("expected one trace step")
	}
	if fixture.traceService.call.RootEngramID != fixture.sourceEngramID {
		t.Fatalf("expected trace root engram id %s, got %s", fixture.sourceEngramID, fixture.traceService.call.RootEngramID)
	}
	if fixture.traceService.call.MaxDepth != 3 {
		t.Fatalf("expected trace max depth 3, got %d", fixture.traceService.call.MaxDepth)
	}
}
