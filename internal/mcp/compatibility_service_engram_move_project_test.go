package mcp

import (
	"context"
	"errors"
	"reflect"
	"testing"
	"time"

	"engram/internal/admin"
	"engram/internal/models"
	"engram/internal/projects"

	"github.com/google/uuid"
)

func TestCompatibilityServiceEngramMoveProjectParity(t *testing.T) {
	actorUserID := uuid.MustParse("39890000-0000-0000-0000-000000000398")
	engramID := uuid.MustParse("39890000-0000-0000-0000-000000000399")
	expectedUpdatedAt := mustParseMoveRFC3339(t, "2026-02-21T00:00:00Z")
	moved := &models.AdminEngramRecord{
		EngramID:  engramID,
		ProjectID: "project-target",
		Title:     "Moved",
	}
	service := &fakeEngramMoveService{moved: moved}
	params := map[string]any{
		"engram_id":           engramID.String(),
		"target_project_id":   "project-target",
		"expected_updated_at": expectedUpdatedAt.Format(time.RFC3339),
		"reason":              "reorg",
	}

	testCases := []struct {
		name            string
		request         StreamCallRequest
		asToolsCallPath bool
		expectedRole    models.UserRole
	}{
		{
			name:            "direct",
			request:         directToolRequest(actorUserID.String(), "engram.move_project", params),
			asToolsCallPath: false,
			expectedRole:    models.UserRoleViewer,
		},
		{
			name:            "tools call",
			request:         toolsCallRequest(actorUserID.String(), "engram_move_project", params),
			asToolsCallPath: true,
			expectedRole:    models.UserRoleAnalyst,
		},
	}

	for _, testCase := range testCases {
		testCase := testCase
		t.Run(testCase.name, func(t *testing.T) {
			frame := runCompatibilityRequestWithService(
				t,
				newEngramMoveCompatibilityService(service),
				testCase.request,
			)
			engram := engramMoveResponseFromFrame(t, frame, testCase.asToolsCallPath)
			if !reflect.DeepEqual(*moved, engram) {
				t.Fatalf("expected move payload to match service output")
			}
			assertEngramMoveCall(
				t,
				service.call,
				engramMoveCallExpectation{
					actorUserID:       actorUserID,
					actorRole:         testCase.expectedRole,
					engramID:          engramID,
					targetProjectID:   "project-target",
					reason:            stringPtr("reorg"),
					expectedUpdatedAt: &expectedUpdatedAt,
				},
			)
		})
	}
}

func TestCompatibilityServiceEngramMoveProjectUsesDefaults(t *testing.T) {
	actorUserID := uuid.MustParse("39900000-0000-0000-0000-000000000398")
	engramID := uuid.MustParse("39900000-0000-0000-0000-000000000399")
	service := &fakeEngramMoveService{moved: &models.AdminEngramRecord{EngramID: engramID}}
	frame := runCompatibilityRequestWithService(
		t,
		newEngramMoveCompatibilityService(service),
		directToolRequest(
			actorUserID.String(),
			"engram.move_project",
			map[string]any{"engram_id": engramID.String()},
		),
	)
	_ = engramMoveResponseFromFrame(t, frame, false)
	if service.call.TargetProjectID != "" {
		t.Fatalf("expected default empty target_project_id")
	}
	if service.call.Reason != nil {
		t.Fatalf("expected optional reason omitted by default")
	}
	if service.call.ExpectedUpdatedAt != nil {
		t.Fatalf("expected optional expected_updated_at omitted by default")
	}
}

func TestCompatibilityServiceEngramMoveProjectValidationAndErrors(t *testing.T) {
	service := newEngramMoveCompatibilityService(&fakeEngramMoveService{})
	assertEngramMoveValidationErrors(t, service)
	assertEngramMoveErrorMappings(t)
}

func newEngramMoveCompatibilityService(service EngramMoveService) Service {
	return NewCompatibilityServiceWithDependencies(
		"1.2.3",
		CompatibilityServiceDependencies{EngramMove: service},
	)
}

func engramMoveResponseFromFrame(
	t *testing.T,
	frame Frame,
	asToolsCallPath bool,
) models.AdminEngramRecord {
	t.Helper()
	result := resultPayloadFromFrame(t, frame)
	payload := result
	if asToolsCallPath {
		payload = mapFromMap(t, result, "structuredContent")
	}
	engram, ok := payload["engram"].(models.AdminEngramRecord)
	if !ok {
		t.Fatalf("expected engram move payload")
	}
	return engram
}

type engramMoveCallExpectation struct {
	actorUserID       uuid.UUID
	actorRole         models.UserRole
	engramID          uuid.UUID
	targetProjectID   string
	reason            *string
	expectedUpdatedAt *time.Time
}

func assertEngramMoveCall(
	t *testing.T,
	call EngramMoveRequest,
	expected engramMoveCallExpectation,
) {
	t.Helper()
	if call.ActorUserID != expected.actorUserID {
		t.Fatalf("expected actor user id forwarded")
	}
	if call.ActorRole != expected.actorRole {
		t.Fatalf("expected normalized actor role forwarded")
	}
	if call.EngramID != expected.engramID {
		t.Fatalf("expected engram id forwarded")
	}
	if call.TargetProjectID != expected.targetProjectID {
		t.Fatalf("expected target_project_id forwarded")
	}
	assertOptionalDeleteReason(t, call.Reason, expected.reason)
	assertOptionalField(
		t,
		"expected_updated_at",
		call.ExpectedUpdatedAt,
		expected.expectedUpdatedAt,
		func(actual time.Time, expected time.Time) bool { return actual.Equal(expected) },
	)
}

func assertEngramMoveValidationErrors(t *testing.T, service Service) {
	t.Helper()
	testCases := []struct {
		name   string
		params map[string]any
	}{
		{name: "missing engram id", params: map[string]any{}},
		{name: "invalid engram id", params: map[string]any{"engram_id": "bad"}},
		{name: "invalid reason", params: map[string]any{"engram_id": uuid.NewString(), "reason": 123}},
		{name: "invalid expected updated at", params: map[string]any{"engram_id": uuid.NewString(), "expected_updated_at": "bad"}},
	}
	for _, testCase := range testCases {
		testCase := testCase
		t.Run(testCase.name, func(t *testing.T) {
			frame := runCompatibilityRequestWithService(
				t,
				service,
				toolsCallRequest(
					"39910000-0000-0000-0000-000000000398",
					"engram_move_project",
					testCase.params,
				),
			)
			requireErrorCode(t, errorPayloadFromFrame(t, frame), -32602)
		})
	}
}

func assertEngramMoveErrorMappings(t *testing.T) {
	t.Helper()
	notFoundID := uuid.MustParse("39910000-0000-0000-0000-000000000399")
	testCases := []struct {
		name           string
		service        Service
		params         map[string]any
		expectedCode   int
		expectedStatus *int
		expectedDetail *string
		expectNotFound bool
	}{
		{
			name:           "not found",
			service:        newEngramMoveCompatibilityService(&fakeEngramMoveService{}),
			params:         map[string]any{"engram_id": notFoundID.String()},
			expectedCode:   -32004,
			expectNotFound: true,
		},
		{
			name:           "stale",
			service:        newEngramMoveCompatibilityService(&fakeEngramMoveService{err: admin.ErrEngramStale}),
			params:         map[string]any{"engram_id": notFoundID.String()},
			expectedCode:   -32602,
			expectedStatus: moveIntPtr(409),
			expectedDetail: stringPtr(admin.ErrEngramStale.Error()),
		},
		{
			name:         "invalid target",
			service:      newEngramMoveCompatibilityService(&fakeEngramMoveService{err: projects.ErrProjectIDMustNotBeBlank}),
			params:       map[string]any{"engram_id": notFoundID.String()},
			expectedCode: -32602,
		},
		{
			name:           "project not found",
			service:        newEngramMoveCompatibilityService(&fakeEngramMoveService{err: projects.ErrProjectNotFound}),
			params:         map[string]any{"engram_id": notFoundID.String(), "target_project_id": "missing"},
			expectedCode:   -32602,
			expectedStatus: moveIntPtr(404),
			expectedDetail: stringPtr("Project not found"),
		},
		{
			name:           "missing default",
			service:        newEngramMoveCompatibilityService(&fakeEngramMoveService{err: projects.ErrProjectIDRequiredWhenNoDefaultProject}),
			params:         map[string]any{"engram_id": notFoundID.String()},
			expectedCode:   -32602,
			expectedStatus: moveIntPtr(422),
		},
		{
			name:           "hidden default",
			service:        newEngramMoveCompatibilityService(&fakeEngramMoveService{err: projects.ErrDefaultProjectNotAccessible}),
			params:         map[string]any{"engram_id": notFoundID.String()},
			expectedCode:   -32602,
			expectedStatus: moveIntPtr(422),
		},
		{
			name:         "internal",
			service:      newEngramMoveCompatibilityService(&fakeEngramMoveService{err: errors.New("boom")}),
			params:       map[string]any{"engram_id": notFoundID.String()},
			expectedCode: -32603,
		},
	}
	for _, testCase := range testCases {
		testCase := testCase
		t.Run(testCase.name, func(t *testing.T) {
			frame := runCompatibilityRequestWithService(
				t,
				testCase.service,
				toolsCallRequest(
					uuid.MustParse("39910000-0000-0000-0000-000000000400").String(),
					"engram_move_project",
					testCase.params,
				),
			)
			errorPayload := errorPayloadFromFrame(t, frame)
			requireErrorCode(t, errorPayload, testCase.expectedCode)
			if testCase.expectNotFound {
				assertEngramMutationNotFoundData(t, errorPayload, notFoundID)
			}
			if testCase.expectedStatus != nil {
				assertErrorStatusCode(t, errorPayload, *testCase.expectedStatus)
			}
			if testCase.expectedDetail != nil {
				assertErrorDetail(t, errorPayload, *testCase.expectedDetail)
			}
		})
	}
}

func moveIntPtr(value int) *int {
	return &value
}

func mustParseMoveRFC3339(t *testing.T, value string) time.Time {
	t.Helper()
	parsed, err := time.Parse(time.RFC3339, value)
	if err != nil {
		t.Fatalf("expected RFC3339 time in test: %v", err)
	}
	return parsed
}

type fakeEngramMoveService struct {
	moved *models.AdminEngramRecord
	err   error
	call  EngramMoveRequest
}

func (service *fakeEngramMoveService) MoveEngram(
	_ context.Context,
	request EngramMoveRequest,
) (*models.AdminEngramRecord, error) {
	service.call = request
	if service.err != nil {
		return nil, service.err
	}
	if service.moved == nil {
		return nil, nil
	}
	moved := *service.moved
	return &moved, nil
}
