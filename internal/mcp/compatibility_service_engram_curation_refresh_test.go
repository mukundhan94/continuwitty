package mcp

import (
	"context"
	"errors"
	"reflect"
	"testing"
	"time"

	"engram/internal/admin"

	"github.com/google/uuid"
)

func TestCompatibilityServiceEngramCurationRefreshParity(t *testing.T) {
	actorUserID := uuid.MustParse("74470000-0000-0000-0000-000000000001")
	sourceEngramID := uuid.MustParse("74470000-0000-0000-0000-000000000002")
	projectID := "engram-vault"
	response := EngramCurationRefreshResponse{
		ProjectID:      &projectID,
		SourceEngramID: sourceEngramID,
		SuggestedAt:    time.Date(2026, 3, 3, 22, 5, 0, 0, time.UTC),
		UpdatedCount:   3,
	}
	service := &fakeEngramCurationRefreshService{response: &response}
	params := map[string]any{
		"project_id":          projectID,
		"source_engram_id":    sourceEngramID.String(),
		"include_archived":    true,
		"limit":               40,
		"stale_after_days":    30,
		"low_value_threshold": 0.45,
	}
	testCases := []struct {
		name            string
		request         StreamCallRequest
		asToolsCallPath bool
	}{
		{
			name: "direct",
			request: asAdminActor(
				directToolRequest(actorUserID.String(), "engram.curation_refresh_links", params),
			),
			asToolsCallPath: false,
		},
		{
			name: "tools call",
			request: asAdminActor(
				toolsCallRequest(actorUserID.String(), "engram_curation_refresh_links", params),
			),
			asToolsCallPath: true,
		},
	}
	for _, testCase := range testCases {
		testCase := testCase
		t.Run(testCase.name, func(t *testing.T) {
			frame := runCompatibilityRequestWithService(
				t,
				newEngramCurationRefreshCompatibilityService(service),
				testCase.request,
			)
			got := curationRefreshFromFrame(t, frame, testCase.asToolsCallPath)
			if !reflect.DeepEqual(response, got) {
				t.Fatalf("expected curation_refresh payload to match service output")
			}
			if !reflect.DeepEqual(
				EngramCurationRefreshRequest{
					ActorUserID:       actorUserID,
					ActorRole:         "admin",
					ProjectID:         &projectID,
					SourceEngramID:    sourceEngramID,
					IncludeArchived:   true,
					Limit:             40,
					StaleAfterDays:    30,
					LowValueThreshold: 0.45,
				},
				service.call,
			) {
				t.Fatalf("expected curation refresh request to match")
			}
		})
	}
}

func TestCompatibilityServiceEngramCurationRefreshValidationAndErrors(t *testing.T) {
	actorUserID := uuid.MustParse("74480000-0000-0000-0000-000000000001")
	validParams := map[string]any{
		"source_engram_id": "74480000-0000-0000-0000-000000000002",
	}
	service := newEngramCurationRefreshCompatibilityService(&fakeEngramCurationRefreshService{})

	forbiddenFrame := runCompatibilityRequestWithService(
		t,
		service,
		toolsCallRequest(actorUserID.String(), "engram_curation_refresh_links", validParams),
	)
	requireErrorCode(t, errorPayloadFromFrame(t, forbiddenFrame), -32602)

	invalidSourceIDFrame := runCompatibilityRequestWithService(
		t,
		service,
		asAdminActor(
			toolsCallRequest(
				actorUserID.String(),
				"engram_curation_refresh_links",
				map[string]any{"source_engram_id": "invalid"},
			),
		),
	)
	requireErrorCode(t, errorPayloadFromFrame(t, invalidSourceIDFrame), -32602)

	invalidIncludeArchivedFrame := runCompatibilityRequestWithService(
		t,
		service,
		asAdminActor(
			toolsCallRequest(
				actorUserID.String(),
				"engram_curation_refresh_links",
				map[string]any{
					"source_engram_id": "74480000-0000-0000-0000-000000000002",
					"include_archived": "bad",
				},
			),
		),
	)
	requireErrorCode(t, errorPayloadFromFrame(t, invalidIncludeArchivedFrame), -32602)

	notFoundFrame := runCompatibilityRequestWithService(
		t,
		newEngramCurationRefreshCompatibilityService(
			&fakeEngramCurationRefreshService{err: admin.ErrEngramNotFound},
		),
		asAdminActor(
			toolsCallRequest(actorUserID.String(), "engram_curation_refresh_links", validParams),
		),
	)
	requireErrorCode(t, errorPayloadFromFrame(t, notFoundFrame), -32602)

	scopeMismatchFrame := runCompatibilityRequestWithService(
		t,
		newEngramCurationRefreshCompatibilityService(
			&fakeEngramCurationRefreshService{err: admin.ErrProjectScopeMismatch},
		),
		asAdminActor(
			toolsCallRequest(actorUserID.String(), "engram_curation_refresh_links", validParams),
		),
	)
	requireErrorCode(t, errorPayloadFromFrame(t, scopeMismatchFrame), -32602)

	internalFrame := runCompatibilityRequestWithService(
		t,
		newEngramCurationRefreshCompatibilityService(
			&fakeEngramCurationRefreshService{err: errors.New("boom")},
		),
		asAdminActor(
			toolsCallRequest(actorUserID.String(), "engram_curation_refresh_links", validParams),
		),
	)
	requireErrorCode(t, errorPayloadFromFrame(t, internalFrame), -32603)
}

func newEngramCurationRefreshCompatibilityService(service EngramCurationRefreshService) Service {
	return NewCompatibilityServiceWithDependencies(
		"1.2.3",
		CompatibilityServiceDependencies{EngramCurationRefresh: service},
	)
}

func curationRefreshFromFrame(
	t *testing.T,
	frame Frame,
	asToolsCallPath bool,
) EngramCurationRefreshResponse {
	t.Helper()
	result := resultPayloadFromFrame(t, frame)
	payload := result
	if asToolsCallPath {
		payload = mapFromMap(t, result, "structuredContent")
	}
	refreshed, ok := payload["curation_refresh"].(EngramCurationRefreshResponse)
	if !ok {
		t.Fatalf("expected curation_refresh payload")
	}
	return refreshed
}

type fakeEngramCurationRefreshService struct {
	response *EngramCurationRefreshResponse
	err      error
	call     EngramCurationRefreshRequest
}

func (service *fakeEngramCurationRefreshService) RefreshEngramLinkCurationSuggestions(
	_ context.Context,
	request EngramCurationRefreshRequest,
) (*EngramCurationRefreshResponse, error) {
	service.call = request
	if service.err != nil {
		return nil, service.err
	}
	if service.response == nil {
		return nil, nil
	}
	refreshed := *service.response
	return &refreshed, nil
}
