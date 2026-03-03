package mcp

import (
	"context"
	"errors"
	"reflect"
	"testing"

	"engram/internal/models"

	"github.com/google/uuid"
)

func TestCompatibilityServiceEngramConsolidationListParity(t *testing.T) {
	actorUserID := uuid.MustParse("63330000-0000-0000-0000-000000000001")
	projectID := "engram-vault"
	status := models.ConsolidationSuggestionStatusSuggested
	suggestionID := uuid.MustParse("63330000-0000-0000-0000-000000000010")
	service := &fakeEngramConsolidationListService{
		response: []models.EngramConsolidationSuggestion{
			{
				SuggestionID: suggestionID,
				ProjectID:    projectID,
				Status:       models.ConsolidationSuggestionStatusSuggested,
			},
		},
	}
	params := map[string]any{
		"project_id": projectID,
		"status":     string(status),
		"limit":      15,
		"offset":     2,
	}
	testCases := []struct {
		name            string
		request         StreamCallRequest
		asToolsCallPath bool
	}{
		{
			name: "direct",
			request: asAdminActor(
				directToolRequest(actorUserID.String(), "engram.consolidation_list", params),
			),
			asToolsCallPath: false,
		},
		{
			name: "tools call",
			request: asAdminActor(
				toolsCallRequest(actorUserID.String(), "engram_consolidation_list", params),
			),
			asToolsCallPath: true,
		},
	}
	for _, testCase := range testCases {
		testCase := testCase
		t.Run(testCase.name, func(t *testing.T) {
			frame := runCompatibilityRequestWithService(
				t,
				newEngramConsolidationListCompatibilityService(service),
				testCase.request,
			)
			got := consolidationListFromFrame(t, frame, testCase.asToolsCallPath)
			if !reflect.DeepEqual(service.response, got) {
				t.Fatalf("expected consolidation_suggestions payload to match service output")
			}
			if !reflect.DeepEqual(
				EngramConsolidationListRequest{
					ActorUserID: actorUserID,
					ActorRole:   "admin",
					ProjectID:   &projectID,
					Status:      &status,
					Limit:       15,
					Offset:      2,
				},
				service.call,
			) {
				t.Fatalf("expected list request to match")
			}
		})
	}
}

func TestCompatibilityServiceEngramConsolidationListValidationAndErrors(t *testing.T) {
	actorUserID := uuid.MustParse("63340000-0000-0000-0000-000000000001")
	service := newEngramConsolidationListCompatibilityService(&fakeEngramConsolidationListService{})
	testCases := []struct {
		name    string
		params  map[string]any
		asAdmin bool
	}{
		{
			name: "forbidden non admin",
			params: map[string]any{
				"project_id": "engram-vault",
			},
			asAdmin: false,
		},
		{
			name: "invalid status",
			params: map[string]any{
				"status": "bad",
			},
			asAdmin: true,
		},
		{
			name: "invalid limit",
			params: map[string]any{
				"limit": "bad",
			},
			asAdmin: true,
		},
	}
	for _, testCase := range testCases {
		testCase := testCase
		t.Run(testCase.name, func(t *testing.T) {
			request := toolsCallRequest(actorUserID.String(), "engram_consolidation_list", testCase.params)
			if testCase.asAdmin {
				request = asAdminActor(request)
			}
			frame := runCompatibilityRequestWithService(t, service, request)
			requireErrorCode(t, errorPayloadFromFrame(t, frame), -32602)
		})
	}

	internalFrame := runCompatibilityRequestWithService(
		t,
		newEngramConsolidationListCompatibilityService(
			&fakeEngramConsolidationListService{err: errors.New("boom")},
		),
		asAdminActor(
			toolsCallRequest(
				actorUserID.String(),
				"engram_consolidation_list",
				map[string]any{"project_id": "engram-vault"},
			),
		),
	)
	requireErrorCode(t, errorPayloadFromFrame(t, internalFrame), -32603)
}

func newEngramConsolidationListCompatibilityService(service EngramConsolidationListService) Service {
	return NewCompatibilityServiceWithDependencies(
		"1.2.3",
		CompatibilityServiceDependencies{EngramConsolidationList: service},
	)
}

func consolidationListFromFrame(
	t *testing.T,
	frame Frame,
	asToolsCallPath bool,
) []models.EngramConsolidationSuggestion {
	t.Helper()
	result := resultPayloadFromFrame(t, frame)
	payload := result
	if asToolsCallPath {
		payload = mapFromMap(t, result, "structuredContent")
	}
	listed, ok := payload["consolidation_suggestions"].([]models.EngramConsolidationSuggestion)
	if !ok {
		t.Fatalf("expected consolidation_suggestions payload")
	}
	return listed
}

type fakeEngramConsolidationListService struct {
	response []models.EngramConsolidationSuggestion
	err      error
	call     EngramConsolidationListRequest
}

func (service *fakeEngramConsolidationListService) ListEngramConsolidationSuggestions(
	_ context.Context,
	request EngramConsolidationListRequest,
) ([]models.EngramConsolidationSuggestion, error) {
	service.call = request
	if service.err != nil {
		return nil, service.err
	}
	return service.response, nil
}
