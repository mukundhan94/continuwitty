package mcp

import (
	"testing"

	"engram/internal/models"
)

func TestCompatibilityServiceEngramLinkSuggestParity(t *testing.T) {
	fixture := newEngramLinkQueryFixture()
	suggestFrame := runCompatibilityRequestWithService(
		t,
		fixture.service,
		toolsCallRequest(
			fixture.actorUserID.String(),
			"engram_link_suggest",
			map[string]any{
				"engram_id":      fixture.sourceEngramID.String(),
				"limit":          4,
				"max_candidates": 12,
				"minimum_score":  0.3,
			},
		),
	)
	suggestions := engramLinkFrameValue[[]models.EngramLinkSuggestion](t, suggestFrame, true, "suggestions")
	if len(suggestions) != 1 {
		t.Fatalf("expected one suggestion response")
	}
	if fixture.suggestService.call.SourceEngramID != fixture.sourceEngramID {
		t.Fatalf("expected suggest source engram id %s, got %s", fixture.sourceEngramID, fixture.suggestService.call.SourceEngramID)
	}
	if fixture.suggestService.call.Limit != 4 {
		t.Fatalf("expected suggest limit 4, got %d", fixture.suggestService.call.Limit)
	}
}
