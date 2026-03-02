package mcp

import (
	"errors"
	"testing"

	"github.com/google/uuid"
)

func TestCompatibilityServiceEngramLinkUpdateAndArchive(t *testing.T) {
	actorUserID := uuid.MustParse("00000000-0000-0000-0000-000000007030")
	linkID := uuid.MustParse("00000000-0000-0000-0000-000000007031")
	updatedRecord := sampleCompatibilityLinkRecord(
		linkID,
		uuid.MustParse("00000000-0000-0000-0000-000000007032"),
		uuid.MustParse("00000000-0000-0000-0000-000000007033"),
	)
	updateService := &fakeEngramLinkUpdateService{updated: &updatedRecord}
	archiveService := &fakeEngramLinkArchiveService{archived: &updatedRecord}
	service := newEngramLinkCompatibilityService(
		CompatibilityServiceDependencies{
			EngramLinkUpdate:  updateService,
			EngramLinkArchive: archiveService,
		},
	)

	updateFrame := runCompatibilityRequestWithService(
		t,
		service,
		directToolRequest(
			actorUserID.String(),
			"engram.link_update",
			map[string]any{
				"link_id":       linkID.String(),
				"status":        "rejected",
				"confidence":    0.2,
				"evidence_json": map[string]any{"reviewed": true},
			},
		),
	)
	_ = engramLinkFromFrame(t, updateFrame, false)
	if updateService.call.LinkID != linkID {
		t.Fatalf("expected update call link id")
	}

	archiveFrame := runCompatibilityRequestWithService(
		t,
		service,
		directToolRequest(
			actorUserID.String(),
			"engram.link_archive",
			map[string]any{"link_id": linkID.String()},
		),
	)
	_ = engramLinkFromFrame(t, archiveFrame, false)
	if archiveService.call.LinkID != linkID {
		t.Fatalf("expected archive call link id")
	}

	validationFrame := runCompatibilityRequestWithService(
		t,
		service,
		toolsCallRequest(
			actorUserID.String(),
			"engram_link_update",
			map[string]any{"link_id": linkID.String()},
		),
	)
	errorPayload := errorPayloadFromFrame(t, validationFrame)
	requireErrorCode(t, errorPayload, -32602)
	assertErrorStatusCode(t, errorPayload, 422)
}

func TestCompatibilityServiceEngramLinkUpdateInternalError(t *testing.T) {
	frame := runCompatibilityRequestWithService(
		t,
		newEngramLinkCompatibilityService(
			CompatibilityServiceDependencies{
				EngramLinkUpdate: &fakeEngramLinkUpdateService{err: errors.New("boom")},
			},
		),
		directToolRequest(
			"00000000-0000-0000-0000-000000007040",
			"engram.link_update",
			map[string]any{
				"link_id":    "00000000-0000-0000-0000-000000007041",
				"confidence": 0.3,
			},
		),
	)
	requireErrorCode(t, errorPayloadFromFrame(t, frame), -32603)
}
