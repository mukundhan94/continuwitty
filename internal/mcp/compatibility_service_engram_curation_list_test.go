package mcp

import (
	"context"
	"errors"
	"reflect"
	"testing"

	"engram/internal/models"

	"github.com/google/uuid"
)

func TestCompatibilityServiceEngramCurationListParity(t *testing.T) {
	actorUserID := uuid.MustParse("74430000-0000-0000-0000-000000000001")
	projectID := "engram-vault"
	sessionID := uuid.MustParse("74430000-0000-0000-0000-000000000002")
	suggestionID := uuid.MustParse("74430000-0000-0000-0000-000000000010")
	suggestionType := models.MemoryCurationSuggestionTypeLink
	status := models.MemoryCurationSuggestionStatusSuggested
	service := &fakeEngramCurationListService{
		response: []models.MemoryCurationSuggestion{
			{
				SuggestionID:   suggestionID,
				ProjectID:      projectID,
				SessionID:      &sessionID,
				SuggestionType: suggestionType,
				Status:         status,
			},
		},
	}
	params := map[string]any{
		"project_id":      projectID,
		"session_id":      sessionID.String(),
		"suggestion_type": string(suggestionType),
		"status":          string(status),
		"limit":           12,
		"offset":          3,
	}
	testCases := []struct {
		name            string
		request         StreamCallRequest
		asToolsCallPath bool
	}{
		{
			name: "direct",
			request: asAdminActor(
				directToolRequest(actorUserID.String(), "engram.curation_list", params),
			),
			asToolsCallPath: false,
		},
		{
			name: "tools call",
			request: asAdminActor(
				toolsCallRequest(actorUserID.String(), "engram_curation_list", params),
			),
			asToolsCallPath: true,
		},
	}
	for _, testCase := range testCases {
		testCase := testCase
		t.Run(testCase.name, func(t *testing.T) {
			frame := runCompatibilityRequestWithService(
				t,
				newEngramCurationListCompatibilityService(service),
				testCase.request,
			)
			got := curationListFromFrame(t, frame, testCase.asToolsCallPath)
			if !reflect.DeepEqual(service.response, got) {
				t.Fatalf("expected curation_suggestions payload to match service output")
			}
			if !reflect.DeepEqual(
				EngramCurationListRequest{
					ActorUserID:    actorUserID,
					ActorRole:      "admin",
					ProjectID:      &projectID,
					SessionID:      &sessionID,
					SuggestionType: &suggestionType,
					Status:         &status,
					Limit:          12,
					Offset:         3,
				},
				service.call,
			) {
				t.Fatalf("expected curation list request to match")
			}
		})
	}
}

func TestCompatibilityServiceEngramCurationListValidationAndErrors(t *testing.T) {
	actorUserID := uuid.MustParse("74440000-0000-0000-0000-000000000001")
	service := newEngramCurationListCompatibilityService(&fakeEngramCurationListService{})
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
			name: "invalid session id",
			params: map[string]any{
				"session_id": "bad",
			},
			asAdmin: true,
		},
		{
			name: "invalid suggestion type",
			params: map[string]any{
				"suggestion_type": "bad",
			},
			asAdmin: true,
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
			request := toolsCallRequest(actorUserID.String(), "engram_curation_list", testCase.params)
			if testCase.asAdmin {
				request = asAdminActor(request)
			}
			frame := runCompatibilityRequestWithService(t, service, request)
			requireErrorCode(t, errorPayloadFromFrame(t, frame), -32602)
		})
	}

	internalFrame := runCompatibilityRequestWithService(
		t,
		newEngramCurationListCompatibilityService(
			&fakeEngramCurationListService{err: errors.New("boom")},
		),
		asAdminActor(
			toolsCallRequest(
				actorUserID.String(),
				"engram_curation_list",
				map[string]any{"project_id": "engram-vault"},
			),
		),
	)
	requireErrorCode(t, errorPayloadFromFrame(t, internalFrame), -32603)
}

func newEngramCurationListCompatibilityService(service EngramCurationListService) Service {
	return NewCompatibilityServiceWithDependencies(
		"1.2.3",
		CompatibilityServiceDependencies{EngramCurationList: service},
	)
}

func curationListFromFrame(
	t *testing.T,
	frame Frame,
	asToolsCallPath bool,
) []models.MemoryCurationSuggestion {
	t.Helper()
	result := resultPayloadFromFrame(t, frame)
	payload := result
	if asToolsCallPath {
		payload = mapFromMap(t, result, "structuredContent")
	}
	listed, ok := payload["curation_suggestions"].([]models.MemoryCurationSuggestion)
	if !ok {
		t.Fatalf("expected curation_suggestions payload")
	}
	return listed
}

type fakeEngramCurationListService struct {
	response []models.MemoryCurationSuggestion
	err      error
	call     EngramCurationListRequest
}

func (service *fakeEngramCurationListService) ListMemoryCurationSuggestions(
	_ context.Context,
	request EngramCurationListRequest,
) ([]models.MemoryCurationSuggestion, error) {
	service.call = request
	if service.err != nil {
		return nil, service.err
	}
	return service.response, nil
}
