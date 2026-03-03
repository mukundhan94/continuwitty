package mcp

import (
	"context"
	"errors"
	"reflect"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestCompatibilityServiceEngramRefreshConsolidationParity(t *testing.T) {
	actorUserID := uuid.MustParse("63310000-0000-0000-0000-000000000001")
	projectID := "engram-vault"
	minGroupSize := 3
	response := EngramConsolidationRefreshResponse{
		ProjectID:    &projectID,
		MinGroupSize: minGroupSize,
		SuggestedAt:  time.Date(2026, 3, 3, 15, 0, 0, 0, time.UTC),
		UpdatedCount: 5,
	}
	service := &fakeEngramConsolidationRefreshService{response: &response}
	params := map[string]any{
		"project_id":     projectID,
		"min_group_size": minGroupSize,
	}
	testCases := []struct {
		name            string
		request         StreamCallRequest
		asToolsCallPath bool
	}{
		{
			name: "direct",
			request: asAdminActor(
				directToolRequest(actorUserID.String(), "engram.refresh_consolidation", params),
			),
			asToolsCallPath: false,
		},
		{
			name: "tools call",
			request: asAdminActor(
				toolsCallRequest(actorUserID.String(), "engram_refresh_consolidation", params),
			),
			asToolsCallPath: true,
		},
	}
	for _, testCase := range testCases {
		testCase := testCase
		t.Run(testCase.name, func(t *testing.T) {
			frame := runCompatibilityRequestWithService(
				t,
				newEngramConsolidationRefreshCompatibilityService(service),
				testCase.request,
			)
			got := consolidationRefreshFromFrame(t, frame, testCase.asToolsCallPath)
			if !reflect.DeepEqual(response, got) {
				t.Fatalf("expected consolidation_refresh payload to match service output")
			}
			assertConsolidationRefreshCall(
				t,
				service.call,
				EngramConsolidationRefreshRequest{
					ActorUserID:  actorUserID,
					ActorRole:    "admin",
					ProjectID:    &projectID,
					MinGroupSize: &minGroupSize,
				},
			)
		})
	}
}

func TestCompatibilityServiceEngramRefreshConsolidationValidationAndErrors(t *testing.T) {
	actorUserID := uuid.MustParse("63320000-0000-0000-0000-000000000001")
	service := newEngramConsolidationRefreshCompatibilityService(&fakeEngramConsolidationRefreshService{})
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
			name: "invalid min group size",
			params: map[string]any{
				"min_group_size": "bad",
			},
			asAdmin: true,
		},
		{
			name: "min group size too low",
			params: map[string]any{
				"min_group_size": 1,
			},
			asAdmin: true,
		},
	}
	for _, testCase := range testCases {
		testCase := testCase
		t.Run(testCase.name, func(t *testing.T) {
			request := toolsCallRequest(actorUserID.String(), "engram_refresh_consolidation", testCase.params)
			if testCase.asAdmin {
				request = asAdminActor(request)
			}
			frame := runCompatibilityRequestWithService(t, service, request)
			requireErrorCode(t, errorPayloadFromFrame(t, frame), -32602)
		})
	}

	internalFrame := runCompatibilityRequestWithService(
		t,
		newEngramConsolidationRefreshCompatibilityService(
			&fakeEngramConsolidationRefreshService{err: errors.New("boom")},
		),
		asAdminActor(
			toolsCallRequest(
				actorUserID.String(),
				"engram_refresh_consolidation",
				map[string]any{"project_id": "engram-vault"},
			),
		),
	)
	requireErrorCode(t, errorPayloadFromFrame(t, internalFrame), -32603)
}

func newEngramConsolidationRefreshCompatibilityService(service EngramConsolidationRefreshService) Service {
	return NewCompatibilityServiceWithDependencies(
		"1.2.3",
		CompatibilityServiceDependencies{EngramConsolidationRefresh: service},
	)
}

func consolidationRefreshFromFrame(
	t *testing.T,
	frame Frame,
	asToolsCallPath bool,
) EngramConsolidationRefreshResponse {
	t.Helper()
	result := resultPayloadFromFrame(t, frame)
	payload := result
	if asToolsCallPath {
		payload = mapFromMap(t, result, "structuredContent")
	}
	refreshed, ok := payload["consolidation_refresh"].(EngramConsolidationRefreshResponse)
	if !ok {
		t.Fatalf("expected consolidation_refresh payload")
	}
	return refreshed
}

func assertConsolidationRefreshCall(
	t *testing.T,
	actual EngramConsolidationRefreshRequest,
	expected EngramConsolidationRefreshRequest,
) {
	t.Helper()
	if actual.ActorUserID != expected.ActorUserID {
		t.Fatalf("expected actor user id to be forwarded")
	}
	if actual.ActorRole != expected.ActorRole {
		t.Fatalf("expected actor role to be forwarded")
	}
	assertOptionalDeleteReason(t, actual.ProjectID, expected.ProjectID)
	if expected.MinGroupSize == nil {
		if actual.MinGroupSize != nil {
			t.Fatalf("expected nil min_group_size")
		}
		return
	}
	if actual.MinGroupSize == nil {
		t.Fatalf("expected min_group_size to be forwarded")
	}
	if *actual.MinGroupSize != *expected.MinGroupSize {
		t.Fatalf("expected min_group_size to match")
	}
}

type fakeEngramConsolidationRefreshService struct {
	response *EngramConsolidationRefreshResponse
	err      error
	call     EngramConsolidationRefreshRequest
}

func (service *fakeEngramConsolidationRefreshService) RefreshEngramConsolidationSuggestions(
	_ context.Context,
	request EngramConsolidationRefreshRequest,
) (*EngramConsolidationRefreshResponse, error) {
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
