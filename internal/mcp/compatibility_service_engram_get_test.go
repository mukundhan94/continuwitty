package mcp

import (
	"context"
	"errors"
	"reflect"
	"testing"
	"time"

	"engram/internal/models"

	"github.com/google/uuid"
)

func TestCompatibilityServiceEngramGetParity(t *testing.T) {
	actorUserID := uuid.MustParse("39300000-0000-0000-0000-000000000393")
	engramID := uuid.MustParse("39300000-0000-0000-0000-000000000394")
	record := &models.AdminEngramRecord{
		EngramID:                engramID,
		ProjectID:               "proj-alpha",
		Title:                   "Snapshot",
		Abstract:                "Summary",
		DetailedSummaryMarkdown: "Details",
		VisibilityScope:         models.VisibilityScopePrivate,
		CreatedAt:               time.Unix(1700003930, 0).UTC(),
		UpdatedAt:               time.Unix(1700003930, 0).UTC(),
	}
	service := &fakeEngramGetService{engram: record}

	testCases := []struct {
		name            string
		request         StreamCallRequest
		asToolsCallPath bool
		expectedRole    models.UserRole
	}{
		{
			name: "direct",
			request: directToolRequest(
				actorUserID.String(),
				"engram.get",
				map[string]any{"engram_id": engramID.String(), "include_deleted": false},
			),
			asToolsCallPath: false,
			expectedRole:    models.UserRoleViewer,
		},
		{
			name: "tools call",
			request: toolsCallRequest(
				actorUserID.String(),
				"engram_get",
				map[string]any{"engram_id": engramID.String(), "include_deleted": false},
			),
			asToolsCallPath: true,
			expectedRole:    models.UserRoleAnalyst,
		},
	}

	for _, testCase := range testCases {
		testCase := testCase
		t.Run(testCase.name, func(t *testing.T) {
			frame := runCompatibilityRequestWithService(
				t,
				newEngramGetCompatibilityService(service),
				testCase.request,
			)
			engram := engramFromFrame(t, frame, testCase.asToolsCallPath)
			if !reflect.DeepEqual(*record, engram) {
				t.Fatalf("expected engram payload to match service output")
			}
			assertEngramGetCall(
				t,
				service.call,
				engramGetExpectation{
					actorUserID:    actorUserID,
					actorRole:      testCase.expectedRole,
					engramID:       engramID,
					includeDeleted: false,
				},
			)
		})
	}
}

func TestCompatibilityServiceEngramGetUsesDefaultIncludeDeleted(t *testing.T) {
	actorUserID := uuid.MustParse("39310000-0000-0000-0000-000000000393")
	engramID := uuid.MustParse("39310000-0000-0000-0000-000000000394")
	service := &fakeEngramGetService{
		engram: &models.AdminEngramRecord{
			EngramID:                engramID,
			ProjectID:               "proj-default",
			Title:                   "Snapshot",
			Abstract:                "Summary",
			DetailedSummaryMarkdown: "Details",
		},
	}
	frame := runCompatibilityRequestWithService(
		t,
		newEngramGetCompatibilityService(service),
		directToolRequest(
			actorUserID.String(),
			"engram.get",
			map[string]any{"engram_id": engramID.String()},
		),
	)
	_ = engramFromFrame(t, frame, false)
	assertEngramGetCall(
		t,
		service.call,
		engramGetExpectation{
			actorUserID:    actorUserID,
			actorRole:      models.UserRoleViewer,
			engramID:       engramID,
			includeDeleted: true,
		},
	)
}

func TestCompatibilityServiceEngramGetValidationAndErrorMappings(t *testing.T) {
	service := newEngramGetCompatibilityService(&fakeEngramGetService{})
	testCases := []struct {
		name   string
		params map[string]any
	}{
		{name: "missing engram id", params: map[string]any{}},
		{name: "invalid engram id", params: map[string]any{"engram_id": "bad"}},
		{name: "invalid include_deleted", params: map[string]any{"engram_id": uuid.NewString(), "include_deleted": "bad"}},
	}

	for _, testCase := range testCases {
		testCase := testCase
		t.Run(testCase.name, func(t *testing.T) {
			frame := runCompatibilityRequestWithService(
				t,
				service,
				toolsCallRequest(
					"39320000-0000-0000-0000-000000000393",
					"engram_get",
					testCase.params,
				),
			)
			requireErrorCode(t, errorPayloadFromFrame(t, frame), -32602)
		})
	}

	notFoundFrame := runCompatibilityRequestWithService(
		t,
		service,
		toolsCallRequest(
			"39320000-0000-0000-0000-000000000394",
			"engram_get",
			map[string]any{"engram_id": uuid.NewString()},
		),
	)
	notFoundPayload := errorPayloadFromFrame(t, notFoundFrame)
	requireErrorCode(t, notFoundPayload, -32602)
	assertEngramNotFoundData(t, notFoundPayload)

	internalFrame := runCompatibilityRequestWithService(
		t,
		newEngramGetCompatibilityService(&fakeEngramGetService{err: errors.New("boom")}),
		toolsCallRequest(
			"39320000-0000-0000-0000-000000000395",
			"engram_get",
			map[string]any{"engram_id": uuid.NewString()},
		),
	)
	requireErrorCode(t, errorPayloadFromFrame(t, internalFrame), -32603)
}

func newEngramGetCompatibilityService(service EngramGetService) Service {
	return NewCompatibilityServiceWithDependencies(
		"1.2.3",
		CompatibilityServiceDependencies{EngramGet: service},
	)
}

func engramFromFrame(t *testing.T, frame Frame, asToolsCallPath bool) models.AdminEngramRecord {
	t.Helper()
	result := resultPayloadFromFrame(t, frame)
	payload := result
	if asToolsCallPath {
		payload = mapFromMap(t, result, "structuredContent")
	}
	engram, ok := payload["engram"].(models.AdminEngramRecord)
	if !ok {
		t.Fatalf("expected engram payload")
	}
	return engram
}

type engramGetExpectation struct {
	actorUserID    uuid.UUID
	actorRole      models.UserRole
	engramID       uuid.UUID
	includeDeleted bool
}

func assertEngramGetCall(t *testing.T, call EngramGetRequest, expected engramGetExpectation) {
	t.Helper()
	if call.ActorUserID != expected.actorUserID {
		t.Fatalf("expected actor user id forwarded")
	}
	if call.ActorRole != expected.actorRole {
		t.Fatalf("expected actor role forwarded")
	}
	if call.EngramID != expected.engramID {
		t.Fatalf("expected engram id forwarded")
	}
	if call.IncludeDeleted != expected.includeDeleted {
		t.Fatalf("expected include_deleted forwarded")
	}
}

func assertEngramNotFoundData(t *testing.T, errorPayload map[string]any) {
	t.Helper()
	data := mapFromMap(t, errorPayload, "data")
	if data["status_code"] != 404 {
		t.Fatalf("expected status_code 404 in error data")
	}
	if data["detail"] != "Engram not found" {
		t.Fatalf("expected engram not-found detail in error data")
	}
}

type fakeEngramGetService struct {
	engram *models.AdminEngramRecord
	err    error
	call   EngramGetRequest
}

func (service *fakeEngramGetService) GetEngram(
	_ context.Context,
	request EngramGetRequest,
) (*models.AdminEngramRecord, error) {
	service.call = request
	if service.err != nil {
		return nil, service.err
	}
	if service.engram == nil {
		return nil, nil
	}
	record := *service.engram
	return &record, nil
}
