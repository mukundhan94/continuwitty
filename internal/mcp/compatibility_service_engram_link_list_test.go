package mcp

import (
	"testing"

	"engram/internal/models"

	"github.com/google/uuid"
)

func TestCompatibilityServiceEngramLinkListParity(t *testing.T) {
	fixture := newEngramLinkQueryFixture()
	listFrame := runCompatibilityRequestWithService(
		t,
		fixture.service,
		directToolRequest(
			fixture.actorUserID.String(),
			"engram.link_list",
			map[string]any{
				"engram_id":        fixture.sourceEngramID.String(),
				"relation_type":    "supports",
				"include_archived": true,
				"limit":            3,
				"offset":           1,
			},
		),
	)
	links := engramLinkFrameValue[[]models.EngramLinkRecord](t, listFrame, false, "links")
	if len(links) != 1 {
		t.Fatalf("expected one link response")
	}
	assertEngramLinkListCall(t, fixture.listService.call, engramLinkListCallExpectation{
		sourceEngramID: fixture.sourceEngramID,
		limit:          3,
		offset:         1,
	})
}

type engramLinkListCallExpectation struct {
	sourceEngramID uuid.UUID
	limit          int
	offset         int
}

func assertEngramLinkListCall(
	t *testing.T,
	call EngramLinkListRequest,
	expectation engramLinkListCallExpectation,
) {
	t.Helper()
	if call.SourceEngramID != expectation.sourceEngramID {
		t.Fatalf("expected list source engram id %s, got %s", expectation.sourceEngramID, call.SourceEngramID)
	}
	if call.Limit != expectation.limit {
		t.Fatalf("expected list limit %d, got %d", expectation.limit, call.Limit)
	}
	if call.Offset != expectation.offset {
		t.Fatalf("expected list offset %d, got %d", expectation.offset, call.Offset)
	}
}
