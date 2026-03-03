package mcp

import (
	"context"
	"errors"
	"reflect"
	"testing"

	"engram/internal/models"

	"github.com/google/uuid"
)

func TestCompatibilityServiceEngramConsolidationActionParity(t *testing.T) {
	actorUserID := uuid.MustParse("63350000-0000-0000-0000-000000000001")
	suggestionID := uuid.MustParse("63350000-0000-0000-0000-000000000002")
	projectID := "engram-vault"
	response := models.EngramConsolidationSuggestion{
		SuggestionID: suggestionID,
		ProjectID:    projectID,
		Status:       models.ConsolidationSuggestionStatusMerged,
	}
	service := &fakeEngramConsolidationActionService{response: &response}
	params := map[string]any{
		"suggestion_id": suggestionID.String(),
		"project_id":    projectID,
		"status":        "merged",
	}
	testCases := []struct {
		name            string
		request         StreamCallRequest
		asToolsCallPath bool
	}{
		{
			name: "direct",
			request: asAdminActor(
				directToolRequest(actorUserID.String(), "engram.consolidation_action", params),
			),
			asToolsCallPath: false,
		},
		{
			name: "tools call",
			request: asAdminActor(
				toolsCallRequest(actorUserID.String(), "engram_consolidation_action", params),
			),
			asToolsCallPath: true,
		},
	}
	for _, testCase := range testCases {
		testCase := testCase
		t.Run(testCase.name, func(t *testing.T) {
			frame := runCompatibilityRequestWithService(
				t,
				newEngramConsolidationActionCompatibilityService(service),
				testCase.request,
			)
			got := consolidationActionFromFrame(t, frame, testCase.asToolsCallPath)
			if !reflect.DeepEqual(response, got) {
				t.Fatalf("expected consolidation_suggestion payload to match service output")
			}
			if !reflect.DeepEqual(
				EngramConsolidationActionRequest{
					ActorUserID:  actorUserID,
					ActorRole:    "admin",
					SuggestionID: suggestionID,
					ProjectID:    &projectID,
					Status:       models.ConsolidationSuggestionStatusMerged,
				},
				service.call,
			) {
				t.Fatalf("expected action request to match")
			}
		})
	}
}

func TestCompatibilityServiceEngramConsolidationActionValidationAndErrors(t *testing.T) {
	actorUserID := uuid.MustParse("63360000-0000-0000-0000-000000000001")
	service := newEngramConsolidationActionCompatibilityService(&fakeEngramConsolidationActionService{})
	testCases := []struct {
		name    string
		params  map[string]any
		asAdmin bool
	}{
		{
			name: "forbidden non admin",
			params: map[string]any{
				"suggestion_id": "63360000-0000-0000-0000-000000000002",
				"status":        "merged",
			},
			asAdmin: false,
		},
		{
			name: "invalid suggestion id",
			params: map[string]any{
				"suggestion_id": "bad",
				"status":        "merged",
			},
			asAdmin: true,
		},
		{
			name: "invalid status",
			params: map[string]any{
				"suggestion_id": "63360000-0000-0000-0000-000000000002",
				"status":        "suggested",
			},
			asAdmin: true,
		},
	}
	for _, testCase := range testCases {
		testCase := testCase
		t.Run(testCase.name, func(t *testing.T) {
			request := toolsCallRequest(actorUserID.String(), "engram_consolidation_action", testCase.params)
			if testCase.asAdmin {
				request = asAdminActor(request)
			}
			frame := runCompatibilityRequestWithService(t, service, request)
			requireErrorCode(t, errorPayloadFromFrame(t, frame), -32602)
		})
	}

	notFoundFrame := runCompatibilityRequestWithService(
		t,
		newEngramConsolidationActionCompatibilityService(&fakeEngramConsolidationActionService{}),
		asAdminActor(
			toolsCallRequest(
				actorUserID.String(),
				"engram_consolidation_action",
				map[string]any{
					"suggestion_id": "63360000-0000-0000-0000-000000000002",
					"status":        "merged",
				},
			),
		),
	)
	requireErrorCode(t, errorPayloadFromFrame(t, notFoundFrame), -32602)

	internalFrame := runCompatibilityRequestWithService(
		t,
		newEngramConsolidationActionCompatibilityService(
			&fakeEngramConsolidationActionService{err: errors.New("boom")},
		),
		asAdminActor(
			toolsCallRequest(
				actorUserID.String(),
				"engram_consolidation_action",
				map[string]any{
					"suggestion_id": "63360000-0000-0000-0000-000000000002",
					"status":        "merged",
				},
			),
		),
	)
	requireErrorCode(t, errorPayloadFromFrame(t, internalFrame), -32603)
}

func newEngramConsolidationActionCompatibilityService(service EngramConsolidationActionService) Service {
	return NewCompatibilityServiceWithDependencies(
		"1.2.3",
		CompatibilityServiceDependencies{EngramConsolidationAction: service},
	)
}

func consolidationActionFromFrame(
	t *testing.T,
	frame Frame,
	asToolsCallPath bool,
) models.EngramConsolidationSuggestion {
	t.Helper()
	result := resultPayloadFromFrame(t, frame)
	payload := result
	if asToolsCallPath {
		payload = mapFromMap(t, result, "structuredContent")
	}
	updated, ok := payload["consolidation_suggestion"].(models.EngramConsolidationSuggestion)
	if !ok {
		t.Fatalf("expected consolidation_suggestion payload")
	}
	return updated
}

type fakeEngramConsolidationActionService struct {
	response *models.EngramConsolidationSuggestion
	err      error
	call     EngramConsolidationActionRequest
}

func (service *fakeEngramConsolidationActionService) ActionEngramConsolidationSuggestion(
	_ context.Context,
	request EngramConsolidationActionRequest,
) (*models.EngramConsolidationSuggestion, error) {
	service.call = request
	if service.err != nil {
		return nil, service.err
	}
	if service.response == nil {
		return nil, nil
	}
	updated := *service.response
	return &updated, nil
}
