package mcp

import (
	"context"
	"errors"
	"reflect"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestCompatibilityServiceEngramRefreshFreshnessParity(t *testing.T) {
	actorUserID := uuid.MustParse("53310000-0000-0000-0000-000000000001")
	projectID := "engram-vault"
	halfLifeDays := 35.0
	response := EngramFreshnessRefreshResponse{
		ProjectID:     &projectID,
		HalfLifeDays:  halfLifeDays,
		ReferenceTime: time.Date(2026, 3, 3, 13, 0, 0, 0, time.UTC),
		UpdatedCount:  7,
	}
	service := &fakeEngramFreshnessRefreshService{response: &response}
	params := map[string]any{
		"project_id":     projectID,
		"half_life_days": halfLifeDays,
	}
	testCases := []struct {
		name            string
		request         StreamCallRequest
		asToolsCallPath bool
	}{
		{
			name: "direct",
			request: asAdminActor(
				directToolRequest(actorUserID.String(), "engram.refresh_freshness", params),
			),
			asToolsCallPath: false,
		},
		{
			name: "tools call",
			request: asAdminActor(
				toolsCallRequest(actorUserID.String(), "engram_refresh_freshness", params),
			),
			asToolsCallPath: true,
		},
	}
	for _, testCase := range testCases {
		testCase := testCase
		t.Run(testCase.name, func(t *testing.T) {
			frame := runCompatibilityRequestWithService(
				t,
				newEngramFreshnessCompatibilityService(service),
				testCase.request,
			)
			got := freshnessRefreshFromFrame(t, frame, testCase.asToolsCallPath)
			if !reflect.DeepEqual(response, got) {
				t.Fatalf("expected freshness payload to match service output")
			}
			assertFreshnessRefreshCall(
				t,
				service.call,
				EngramFreshnessRefreshRequest{
					ActorUserID:  actorUserID,
					ActorRole:    "admin",
					ProjectID:    &projectID,
					HalfLifeDays: &halfLifeDays,
				},
			)
		})
	}
}

func TestCompatibilityServiceEngramRefreshFreshnessValidationAndErrors(t *testing.T) {
	actorUserID := uuid.MustParse("53320000-0000-0000-0000-000000000001")
	service := newEngramFreshnessCompatibilityService(&fakeEngramFreshnessRefreshService{})
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
			name: "invalid half life",
			params: map[string]any{
				"half_life_days": "bad",
			},
			asAdmin: true,
		},
		{
			name: "non positive half life",
			params: map[string]any{
				"half_life_days": 0,
			},
			asAdmin: true,
		},
	}
	for _, testCase := range testCases {
		testCase := testCase
		t.Run(testCase.name, func(t *testing.T) {
			request := toolsCallRequest(actorUserID.String(), "engram_refresh_freshness", testCase.params)
			if testCase.asAdmin {
				request = asAdminActor(request)
			}
			frame := runCompatibilityRequestWithService(t, service, request)
			errorPayload := errorPayloadFromFrame(t, frame)
			if testCase.name == "forbidden non admin" {
				requireErrorCode(t, errorPayload, -32602)
				return
			}
			requireErrorCode(t, errorPayload, -32602)
		})
	}

	internalFrame := runCompatibilityRequestWithService(
		t,
		newEngramFreshnessCompatibilityService(
			&fakeEngramFreshnessRefreshService{err: errors.New("boom")},
		),
		asAdminActor(
			toolsCallRequest(
				actorUserID.String(),
				"engram_refresh_freshness",
				map[string]any{"project_id": "engram-vault"},
			),
		),
	)
	requireErrorCode(t, errorPayloadFromFrame(t, internalFrame), -32603)
}

func newEngramFreshnessCompatibilityService(service EngramFreshnessRefreshService) Service {
	return NewCompatibilityServiceWithDependencies(
		"1.2.3",
		CompatibilityServiceDependencies{EngramFreshnessRefresh: service},
	)
}

func freshnessRefreshFromFrame(
	t *testing.T,
	frame Frame,
	asToolsCallPath bool,
) EngramFreshnessRefreshResponse {
	t.Helper()
	result := resultPayloadFromFrame(t, frame)
	payload := result
	if asToolsCallPath {
		payload = mapFromMap(t, result, "structuredContent")
	}
	refreshed, ok := payload["freshness_refresh"].(EngramFreshnessRefreshResponse)
	if !ok {
		t.Fatalf("expected freshness_refresh payload")
	}
	return refreshed
}

func assertFreshnessRefreshCall(
	t *testing.T,
	actual EngramFreshnessRefreshRequest,
	expected EngramFreshnessRefreshRequest,
) {
	t.Helper()
	if actual.ActorUserID != expected.ActorUserID {
		t.Fatalf("expected actor user id to be forwarded")
	}
	if actual.ActorRole != expected.ActorRole {
		t.Fatalf("expected actor role to be forwarded")
	}
	assertOptionalDeleteReason(t, actual.ProjectID, expected.ProjectID)
	if expected.HalfLifeDays == nil {
		if actual.HalfLifeDays != nil {
			t.Fatalf("expected nil half_life_days")
		}
		return
	}
	if actual.HalfLifeDays == nil {
		t.Fatalf("expected half_life_days to be forwarded")
	}
	if *actual.HalfLifeDays != *expected.HalfLifeDays {
		t.Fatalf("expected half_life_days to match")
	}
}

func asAdminActor(request StreamCallRequest) StreamCallRequest {
	request.Actor.Role = "admin"
	return request
}

type fakeEngramFreshnessRefreshService struct {
	response *EngramFreshnessRefreshResponse
	err      error
	call     EngramFreshnessRefreshRequest
}

func (service *fakeEngramFreshnessRefreshService) RefreshEngramFreshness(
	_ context.Context,
	request EngramFreshnessRefreshRequest,
) (*EngramFreshnessRefreshResponse, error) {
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
