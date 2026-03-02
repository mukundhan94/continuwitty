package mcp

import (
	"testing"

	"engram/internal/repository"

	"github.com/google/uuid"
)

func TestCompatibilityServiceEngramLinkCreateParity(t *testing.T) {
	actorUserID := uuid.MustParse("00000000-0000-0000-0000-000000007001")
	sourceEngramID := uuid.MustParse("00000000-0000-0000-0000-000000007002")
	targetEngramID := uuid.MustParse("00000000-0000-0000-0000-000000007003")
	link := sampleCompatibilityLinkRecord(
		uuid.MustParse("00000000-0000-0000-0000-000000007004"),
		sourceEngramID,
		targetEngramID,
	)
	params := map[string]any{
		"engram_id":        sourceEngramID.String(),
		"target_engram_id": targetEngramID.String(),
		"relation_type":    "supports",
		"weight":           0.8,
		"temporal_weight":  0.7,
		"confidence":       0.6,
	}
	testCases := []struct {
		name            string
		request         StreamCallRequest
		asToolsCallPath bool
	}{
		{
			name:            "direct",
			request:         directToolRequest(actorUserID.String(), "engram.link_create", params),
			asToolsCallPath: false,
		},
		{
			name:            "tools call",
			request:         toolsCallRequest(actorUserID.String(), "engram_link_create", params),
			asToolsCallPath: true,
		},
	}
	for _, testCase := range testCases {
		testCase := testCase
		t.Run(testCase.name, func(t *testing.T) {
			service := &fakeEngramLinkCreateService{created: &link}
			frame := runCompatibilityRequestWithService(
				t,
				newEngramLinkCompatibilityService(
					CompatibilityServiceDependencies{EngramLinkCreate: service},
				),
				testCase.request,
			)
			created := engramLinkFromFrame(t, frame, testCase.asToolsCallPath)
			if created.LinkID != link.LinkID {
				t.Fatalf("expected created link id %s", link.LinkID)
			}
			if service.call.ActorUserID != actorUserID {
				t.Fatalf("expected actor user id in create call")
			}
			if service.call.SourceEngramID != sourceEngramID || service.call.TargetEngramID != targetEngramID {
				t.Fatalf("expected source/target ids in create call")
			}
		})
	}
}

func TestCompatibilityServiceEngramLinkCreateDuplicateErrorMapping(t *testing.T) {
	frame := runCompatibilityRequestWithService(
		t,
		newEngramLinkCompatibilityService(
			CompatibilityServiceDependencies{
				EngramLinkCreate: &fakeEngramLinkCreateService{err: repository.ErrEngramLinkExists},
			},
		),
		toolsCallRequest(
			"00000000-0000-0000-0000-000000007010",
			"engram_link_create",
			map[string]any{
				"engram_id":        "00000000-0000-0000-0000-000000007011",
				"target_engram_id": "00000000-0000-0000-0000-000000007012",
			},
		),
	)
	errorPayload := errorPayloadFromFrame(t, frame)
	requireErrorCode(t, errorPayload, -32602)
	assertErrorStatusCode(t, errorPayload, 409)
	assertErrorDetail(t, errorPayload, repository.ErrEngramLinkExists.Error())
}
