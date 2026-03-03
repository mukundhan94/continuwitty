package mcp

import (
	"context"
	"errors"
	"reflect"
	"testing"

	"engram/internal/admin"
	"engram/internal/models"

	"github.com/google/uuid"
)

func TestCompatibilityServiceEngramCurationActionParity(t *testing.T) {
	actorUserID := uuid.MustParse("74450000-0000-0000-0000-000000000001")
	suggestionID := uuid.MustParse("74450000-0000-0000-0000-000000000002")
	projectID := "engram-vault"
	response := models.MemoryCurationSuggestion{
		SuggestionID: suggestionID,
		ProjectID:    projectID,
		Status:       models.MemoryCurationSuggestionStatusApplied,
	}
	service := &fakeEngramCurationActionService{response: &response}
	params := map[string]any{
		"suggestion_id": suggestionID.String(),
		"project_id":    projectID,
		"status":        "applied",
	}
	testCases := []struct {
		name            string
		request         StreamCallRequest
		asToolsCallPath bool
	}{
		{
			name: "direct",
			request: asAdminActor(
				directToolRequest(actorUserID.String(), "engram.curation_action", params),
			),
			asToolsCallPath: false,
		},
		{
			name: "tools call",
			request: asAdminActor(
				toolsCallRequest(actorUserID.String(), "engram_curation_action", params),
			),
			asToolsCallPath: true,
		},
	}
	for _, testCase := range testCases {
		testCase := testCase
		t.Run(testCase.name, func(t *testing.T) {
			frame := runCompatibilityRequestWithService(
				t,
				newEngramCurationActionCompatibilityService(service),
				testCase.request,
			)
			got := curationActionFromFrame(t, frame, testCase.asToolsCallPath)
			if !reflect.DeepEqual(response, got) {
				t.Fatalf("expected curation_suggestion payload to match service output")
			}
			if !reflect.DeepEqual(
				EngramCurationActionRequest{
					ActorUserID:  actorUserID,
					ActorRole:    "admin",
					SuggestionID: suggestionID,
					ProjectID:    &projectID,
					Status:       models.MemoryCurationSuggestionStatusApplied,
				},
				service.call,
			) {
				t.Fatalf("expected curation action request to match")
			}
		})
	}
}

func TestCompatibilityServiceEngramCurationActionValidation(t *testing.T) {
	actorUserID := uuid.MustParse("74460000-0000-0000-0000-000000000001")
	testCases := []struct {
		name    string
		params  map[string]any
		asAdmin bool
	}{
		{
			name: "forbidden non admin",
			params: map[string]any{
				"suggestion_id": "74460000-0000-0000-0000-000000000002",
				"status":        "applied",
			},
			asAdmin: false,
		},
		{
			name: "invalid suggestion id",
			params: map[string]any{
				"suggestion_id": "bad",
				"status":        "applied",
			},
			asAdmin: true,
		},
		{
			name: "invalid status",
			params: map[string]any{
				"suggestion_id": "74460000-0000-0000-0000-000000000002",
				"status":        "suggested",
			},
			asAdmin: true,
		},
	}
	for _, testCase := range testCases {
		testCase := testCase
		t.Run(testCase.name, func(t *testing.T) {
			frame := runCurationActionFrame(
				t,
				curationActionFrameInput{
					actorUserID:   actorUserID,
					actionService: &fakeEngramCurationActionService{},
					params:        testCase.params,
					asAdmin:       testCase.asAdmin,
				},
			)
			requireErrorCode(t, errorPayloadFromFrame(t, frame), -32602)
		})
	}

	notFoundFrame := runCurationActionFrame(
		t,
		curationActionFrameInput{
			actorUserID:   actorUserID,
			actionService: &fakeEngramCurationActionService{},
			params: map[string]any{
				"suggestion_id": "74460000-0000-0000-0000-000000000002",
				"status":        "applied",
			},
			asAdmin: true,
		},
	)
	requireErrorCode(t, errorPayloadFromFrame(t, notFoundFrame), -32602)
}

func TestCompatibilityServiceEngramCurationActionServiceErrors(t *testing.T) {
	actorUserID := uuid.MustParse("74460000-0000-0000-0000-000000000001")
	params := map[string]any{
		"suggestion_id": "74460000-0000-0000-0000-000000000002",
		"status":        "applied",
	}

	internalFrame := runCurationActionFrame(
		t,
		curationActionFrameInput{
			actorUserID:   actorUserID,
			actionService: &fakeEngramCurationActionService{err: errors.New("boom")},
			params:        params,
			asAdmin:       true,
		},
	)
	requireErrorCode(t, errorPayloadFromFrame(t, internalFrame), -32603)

	invalidPayloadFrame := runCurationActionFrame(
		t,
		curationActionFrameInput{
			actorUserID:   actorUserID,
			actionService: &fakeEngramCurationActionService{err: admin.ErrMemoryCurationSuggestionPayloadInvalid},
			params:        params,
			asAdmin:       true,
		},
	)
	requireErrorCode(t, errorPayloadFromFrame(t, invalidPayloadFrame), -32602)
}

func newEngramCurationActionCompatibilityService(service EngramCurationActionService) Service {
	return NewCompatibilityServiceWithDependencies(
		"1.2.3",
		CompatibilityServiceDependencies{EngramCurationAction: service},
	)
}

type curationActionFrameInput struct {
	actorUserID   uuid.UUID
	actionService EngramCurationActionService
	params        map[string]any
	asAdmin       bool
}

func runCurationActionFrame(t *testing.T, input curationActionFrameInput) Frame {
	t.Helper()
	request := toolsCallRequest(input.actorUserID.String(), "engram_curation_action", input.params)
	if input.asAdmin {
		request = asAdminActor(request)
	}
	return runCompatibilityRequestWithService(
		t,
		newEngramCurationActionCompatibilityService(input.actionService),
		request,
	)
}

func curationActionFromFrame(
	t *testing.T,
	frame Frame,
	asToolsCallPath bool,
) models.MemoryCurationSuggestion {
	t.Helper()
	result := resultPayloadFromFrame(t, frame)
	payload := result
	if asToolsCallPath {
		payload = mapFromMap(t, result, "structuredContent")
	}
	updated, ok := payload["curation_suggestion"].(models.MemoryCurationSuggestion)
	if !ok {
		t.Fatalf("expected curation_suggestion payload")
	}
	return updated
}

type fakeEngramCurationActionService struct {
	response *models.MemoryCurationSuggestion
	err      error
	call     EngramCurationActionRequest
}

func (service *fakeEngramCurationActionService) ActionMemoryCurationSuggestion(
	_ context.Context,
	request EngramCurationActionRequest,
) (*models.MemoryCurationSuggestion, error) {
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
