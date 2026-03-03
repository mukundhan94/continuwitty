package mcp

import (
	"context"
	"errors"
	"reflect"
	"testing"

	"engram/internal/models"

	"github.com/google/uuid"
)

func TestCompatibilityServiceEngramContradictionListParity(t *testing.T) {
	actorUserID := uuid.MustParse("73430000-0000-0000-0000-000000000001")
	projectID := "engram-vault"
	status := models.ContradictionAlertStatusOpen
	alertID := uuid.MustParse("73430000-0000-0000-0000-000000000010")
	service := &fakeEngramContradictionListService{
		response: []models.EngramContradictionAlert{
			{
				AlertID:   alertID,
				ProjectID: projectID,
				Status:    models.ContradictionAlertStatusOpen,
			},
		},
	}
	params := map[string]any{
		"project_id": projectID,
		"status":     string(status),
		"limit":      12,
		"offset":     3,
	}
	testCases := []struct {
		name            string
		request         StreamCallRequest
		asToolsCallPath bool
	}{
		{
			name: "direct",
			request: asAdminActor(
				directToolRequest(actorUserID.String(), "engram.contradiction_list", params),
			),
			asToolsCallPath: false,
		},
		{
			name: "tools call",
			request: asAdminActor(
				toolsCallRequest(actorUserID.String(), "engram_contradiction_list", params),
			),
			asToolsCallPath: true,
		},
	}
	for _, testCase := range testCases {
		testCase := testCase
		t.Run(testCase.name, func(t *testing.T) {
			frame := runCompatibilityRequestWithService(
				t,
				newEngramContradictionListCompatibilityService(service),
				testCase.request,
			)
			got := contradictionListFromFrame(t, frame, testCase.asToolsCallPath)
			if !reflect.DeepEqual(service.response, got) {
				t.Fatalf("expected contradiction_alerts payload to match service output")
			}
			if !reflect.DeepEqual(
				EngramContradictionListRequest{
					ActorUserID: actorUserID,
					ActorRole:   "admin",
					ProjectID:   &projectID,
					Status:      &status,
					Limit:       12,
					Offset:      3,
				},
				service.call,
			) {
				t.Fatalf("expected contradiction list request to match")
			}
		})
	}
}

func TestCompatibilityServiceEngramContradictionListValidationAndErrors(t *testing.T) {
	actorUserID := uuid.MustParse("73440000-0000-0000-0000-000000000001")
	service := newEngramContradictionListCompatibilityService(&fakeEngramContradictionListService{})
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
			request := toolsCallRequest(actorUserID.String(), "engram_contradiction_list", testCase.params)
			if testCase.asAdmin {
				request = asAdminActor(request)
			}
			frame := runCompatibilityRequestWithService(t, service, request)
			requireErrorCode(t, errorPayloadFromFrame(t, frame), -32602)
		})
	}

	internalFrame := runCompatibilityRequestWithService(
		t,
		newEngramContradictionListCompatibilityService(
			&fakeEngramContradictionListService{err: errors.New("boom")},
		),
		asAdminActor(
			toolsCallRequest(
				actorUserID.String(),
				"engram_contradiction_list",
				map[string]any{"project_id": "engram-vault"},
			),
		),
	)
	requireErrorCode(t, errorPayloadFromFrame(t, internalFrame), -32603)
}

func newEngramContradictionListCompatibilityService(service EngramContradictionListService) Service {
	return NewCompatibilityServiceWithDependencies(
		"1.2.3",
		CompatibilityServiceDependencies{EngramContradictionList: service},
	)
}

func contradictionListFromFrame(
	t *testing.T,
	frame Frame,
	asToolsCallPath bool,
) []models.EngramContradictionAlert {
	t.Helper()
	result := resultPayloadFromFrame(t, frame)
	payload := result
	if asToolsCallPath {
		payload = mapFromMap(t, result, "structuredContent")
	}
	listed, ok := payload["contradiction_alerts"].([]models.EngramContradictionAlert)
	if !ok {
		t.Fatalf("expected contradiction_alerts payload")
	}
	return listed
}

type fakeEngramContradictionListService struct {
	response []models.EngramContradictionAlert
	err      error
	call     EngramContradictionListRequest
}

func (service *fakeEngramContradictionListService) ListEngramContradictionAlerts(
	_ context.Context,
	request EngramContradictionListRequest,
) ([]models.EngramContradictionAlert, error) {
	service.call = request
	if service.err != nil {
		return nil, service.err
	}
	return service.response, nil
}
