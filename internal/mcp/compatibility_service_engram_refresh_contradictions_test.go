package mcp

import (
	"context"
	"errors"
	"reflect"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestCompatibilityServiceEngramRefreshContradictionsParity(t *testing.T) {
	actorUserID := uuid.MustParse("73410000-0000-0000-0000-000000000001")
	projectID := "engram-vault"
	response := EngramContradictionRefreshResponse{
		ProjectID:    &projectID,
		DetectedAt:   time.Date(2026, 3, 3, 20, 15, 0, 0, time.UTC),
		UpdatedCount: 7,
	}
	service := &fakeEngramContradictionRefreshService{response: &response}
	params := map[string]any{"project_id": projectID}
	testCases := []struct {
		name            string
		request         StreamCallRequest
		asToolsCallPath bool
	}{
		{
			name: "direct",
			request: asAdminActor(
				directToolRequest(actorUserID.String(), "engram.refresh_contradictions", params),
			),
			asToolsCallPath: false,
		},
		{
			name: "tools call",
			request: asAdminActor(
				toolsCallRequest(actorUserID.String(), "engram_refresh_contradictions", params),
			),
			asToolsCallPath: true,
		},
	}
	for _, testCase := range testCases {
		testCase := testCase
		t.Run(testCase.name, func(t *testing.T) {
			frame := runCompatibilityRequestWithService(
				t,
				newEngramContradictionRefreshCompatibilityService(service),
				testCase.request,
			)
			got := contradictionRefreshFromFrame(t, frame, testCase.asToolsCallPath)
			if !reflect.DeepEqual(response, got) {
				t.Fatalf("expected contradiction_refresh payload to match service output")
			}
			if !reflect.DeepEqual(
				EngramContradictionRefreshRequest{
					ActorUserID: actorUserID,
					ActorRole:   "admin",
					ProjectID:   &projectID,
				},
				service.call,
			) {
				t.Fatalf("expected contradiction refresh request to match")
			}
		})
	}
}

func TestCompatibilityServiceEngramRefreshContradictionsValidationAndErrors(t *testing.T) {
	actorUserID := uuid.MustParse("73420000-0000-0000-0000-000000000001")
	service := newEngramContradictionRefreshCompatibilityService(
		&fakeEngramContradictionRefreshService{},
	)

	forbiddenFrame := runCompatibilityRequestWithService(
		t,
		service,
		toolsCallRequest(
			actorUserID.String(),
			"engram_refresh_contradictions",
			map[string]any{"project_id": "engram-vault"},
		),
	)
	requireErrorCode(t, errorPayloadFromFrame(t, forbiddenFrame), -32602)

	internalFrame := runCompatibilityRequestWithService(
		t,
		newEngramContradictionRefreshCompatibilityService(
			&fakeEngramContradictionRefreshService{err: errors.New("boom")},
		),
		asAdminActor(
			toolsCallRequest(
				actorUserID.String(),
				"engram_refresh_contradictions",
				map[string]any{"project_id": "engram-vault"},
			),
		),
	)
	requireErrorCode(t, errorPayloadFromFrame(t, internalFrame), -32603)
}

func newEngramContradictionRefreshCompatibilityService(service EngramContradictionRefreshService) Service {
	return NewCompatibilityServiceWithDependencies(
		"1.2.3",
		CompatibilityServiceDependencies{EngramContradictionRefresh: service},
	)
}

func contradictionRefreshFromFrame(
	t *testing.T,
	frame Frame,
	asToolsCallPath bool,
) EngramContradictionRefreshResponse {
	t.Helper()
	result := resultPayloadFromFrame(t, frame)
	payload := result
	if asToolsCallPath {
		payload = mapFromMap(t, result, "structuredContent")
	}
	refreshed, ok := payload["contradiction_refresh"].(EngramContradictionRefreshResponse)
	if !ok {
		t.Fatalf("expected contradiction_refresh payload")
	}
	return refreshed
}

type fakeEngramContradictionRefreshService struct {
	response *EngramContradictionRefreshResponse
	err      error
	call     EngramContradictionRefreshRequest
}

func (service *fakeEngramContradictionRefreshService) RefreshEngramContradictionAlerts(
	_ context.Context,
	request EngramContradictionRefreshRequest,
) (*EngramContradictionRefreshResponse, error) {
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
