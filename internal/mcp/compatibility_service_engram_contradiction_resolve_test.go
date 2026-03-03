package mcp

import (
	"context"
	"errors"
	"reflect"
	"testing"

	"engram/internal/models"

	"github.com/google/uuid"
)

func TestCompatibilityServiceEngramContradictionResolveParity(t *testing.T) {
	actorUserID := uuid.MustParse("73450000-0000-0000-0000-000000000001")
	alertID := uuid.MustParse("73450000-0000-0000-0000-000000000002")
	projectID := "engram-vault"
	response := models.EngramContradictionAlert{
		AlertID:   alertID,
		ProjectID: projectID,
		Status:    models.ContradictionAlertStatusResolved,
	}
	service := &fakeEngramContradictionResolveService{response: &response}
	params := map[string]any{
		"alert_id":   alertID.String(),
		"project_id": projectID,
		"status":     "resolved",
	}
	testCases := []struct {
		name            string
		request         StreamCallRequest
		asToolsCallPath bool
	}{
		{
			name: "direct",
			request: asAdminActor(
				directToolRequest(actorUserID.String(), "engram.contradiction_resolve", params),
			),
			asToolsCallPath: false,
		},
		{
			name: "tools call",
			request: asAdminActor(
				toolsCallRequest(actorUserID.String(), "engram_contradiction_resolve", params),
			),
			asToolsCallPath: true,
		},
	}
	for _, testCase := range testCases {
		testCase := testCase
		t.Run(testCase.name, func(t *testing.T) {
			frame := runCompatibilityRequestWithService(
				t,
				newEngramContradictionResolveCompatibilityService(service),
				testCase.request,
			)
			got := contradictionResolveFromFrame(t, frame, testCase.asToolsCallPath)
			if !reflect.DeepEqual(response, got) {
				t.Fatalf("expected contradiction_alert payload to match service output")
			}
			if !reflect.DeepEqual(
				EngramContradictionResolveRequest{
					ActorUserID: actorUserID,
					ActorRole:   "admin",
					AlertID:     alertID,
					ProjectID:   &projectID,
					Status:      models.ContradictionAlertStatusResolved,
				},
				service.call,
			) {
				t.Fatalf("expected contradiction resolve request to match")
			}
		})
	}
}

func TestCompatibilityServiceEngramContradictionResolveValidationAndErrors(t *testing.T) {
	actorUserID := uuid.MustParse("73460000-0000-0000-0000-000000000001")
	service := newEngramContradictionResolveCompatibilityService(
		&fakeEngramContradictionResolveService{},
	)
	testCases := []struct {
		name    string
		params  map[string]any
		asAdmin bool
	}{
		{
			name: "forbidden non admin",
			params: map[string]any{
				"alert_id": "73460000-0000-0000-0000-000000000002",
				"status":   "resolved",
			},
			asAdmin: false,
		},
		{
			name: "invalid alert id",
			params: map[string]any{
				"alert_id": "bad",
				"status":   "resolved",
			},
			asAdmin: true,
		},
		{
			name: "invalid status",
			params: map[string]any{
				"alert_id": "73460000-0000-0000-0000-000000000002",
				"status":   "open",
			},
			asAdmin: true,
		},
	}
	for _, testCase := range testCases {
		testCase := testCase
		t.Run(testCase.name, func(t *testing.T) {
			request := toolsCallRequest(actorUserID.String(), "engram_contradiction_resolve", testCase.params)
			if testCase.asAdmin {
				request = asAdminActor(request)
			}
			frame := runCompatibilityRequestWithService(t, service, request)
			requireErrorCode(t, errorPayloadFromFrame(t, frame), -32602)
		})
	}

	notFoundFrame := runCompatibilityRequestWithService(
		t,
		newEngramContradictionResolveCompatibilityService(&fakeEngramContradictionResolveService{}),
		asAdminActor(
			toolsCallRequest(
				actorUserID.String(),
				"engram_contradiction_resolve",
				map[string]any{
					"alert_id": "73460000-0000-0000-0000-000000000002",
					"status":   "resolved",
				},
			),
		),
	)
	requireErrorCode(t, errorPayloadFromFrame(t, notFoundFrame), -32602)

	internalFrame := runCompatibilityRequestWithService(
		t,
		newEngramContradictionResolveCompatibilityService(
			&fakeEngramContradictionResolveService{err: errors.New("boom")},
		),
		asAdminActor(
			toolsCallRequest(
				actorUserID.String(),
				"engram_contradiction_resolve",
				map[string]any{
					"alert_id": "73460000-0000-0000-0000-000000000002",
					"status":   "resolved",
				},
			),
		),
	)
	requireErrorCode(t, errorPayloadFromFrame(t, internalFrame), -32603)
}

func newEngramContradictionResolveCompatibilityService(service EngramContradictionResolveService) Service {
	return NewCompatibilityServiceWithDependencies(
		"1.2.3",
		CompatibilityServiceDependencies{EngramContradictionResolve: service},
	)
}

func contradictionResolveFromFrame(
	t *testing.T,
	frame Frame,
	asToolsCallPath bool,
) models.EngramContradictionAlert {
	t.Helper()
	result := resultPayloadFromFrame(t, frame)
	payload := result
	if asToolsCallPath {
		payload = mapFromMap(t, result, "structuredContent")
	}
	resolved, ok := payload["contradiction_alert"].(models.EngramContradictionAlert)
	if !ok {
		t.Fatalf("expected contradiction_alert payload")
	}
	return resolved
}

type fakeEngramContradictionResolveService struct {
	response *models.EngramContradictionAlert
	err      error
	call     EngramContradictionResolveRequest
}

func (service *fakeEngramContradictionResolveService) ResolveEngramContradictionAlert(
	_ context.Context,
	request EngramContradictionResolveRequest,
) (*models.EngramContradictionAlert, error) {
	service.call = request
	if service.err != nil {
		return nil, service.err
	}
	if service.response == nil {
		return nil, nil
	}
	resolved := *service.response
	return &resolved, nil
}
