package mcp

import (
	"context"
	"errors"
	"reflect"
	"testing"

	"engram/internal/models"

	"github.com/google/uuid"
)

func TestCompatibilityServiceEngramRestoreParity(t *testing.T) {
	actorUserID := uuid.MustParse("39840000-0000-0000-0000-000000000398")
	engramID := uuid.MustParse("39840000-0000-0000-0000-000000000399")
	restoreService := &fakeEngramRestoreService{
		response: &EngramRestoreResponse{
			EngramID: engramID,
			Restored: true,
		},
	}
	params := map[string]any{"engram_id": engramID.String()}

	testCases := []struct {
		name            string
		request         StreamCallRequest
		asToolsCallPath bool
		expectedRole    models.UserRole
	}{
		{
			name:            "direct",
			request:         directToolRequest(actorUserID.String(), "engram.restore", params),
			asToolsCallPath: false,
			expectedRole:    models.UserRoleViewer,
		},
		{
			name:            "tools call",
			request:         toolsCallRequest(actorUserID.String(), "engram_restore", params),
			asToolsCallPath: true,
			expectedRole:    models.UserRoleAnalyst,
		},
	}

	for _, testCase := range testCases {
		testCase := testCase
		t.Run(testCase.name, func(t *testing.T) {
			frame := runCompatibilityRequestWithService(
				t,
				newEngramRestoreCompatibilityService(restoreService),
				testCase.request,
			)
			result := engramRestoreResultFromFrame(t, frame, testCase.asToolsCallPath)
			if !reflect.DeepEqual(*restoreService.response, result) {
				t.Fatalf("expected restore payload to match service output")
			}
			if restoreService.call.ActorUserID != actorUserID {
				t.Fatalf("expected actor user id forwarded")
			}
			if restoreService.call.ActorRole != testCase.expectedRole {
				t.Fatalf("expected normalized actor role forwarded")
			}
			if restoreService.call.EngramID != engramID {
				t.Fatalf("expected engram id forwarded")
			}
		})
	}
}

func TestCompatibilityServiceEngramRestoreValidationAndErrors(t *testing.T) {
	service := newEngramRestoreCompatibilityService(&fakeEngramRestoreService{})
	testCases := []struct {
		name   string
		params map[string]any
	}{
		{name: "missing engram id", params: map[string]any{}},
		{name: "invalid engram id", params: map[string]any{"engram_id": "bad"}},
	}

	for _, testCase := range testCases {
		testCase := testCase
		t.Run(testCase.name, func(t *testing.T) {
			frame := runCompatibilityRequestWithService(
				t,
				service,
				toolsCallRequest(
					"39850000-0000-0000-0000-000000000398",
					"engram_restore",
					testCase.params,
				),
			)
			requireErrorCode(t, errorPayloadFromFrame(t, frame), -32602)
		})
	}

	notFoundID := uuid.MustParse("39850000-0000-0000-0000-000000000399")
	notFoundFrame := runCompatibilityRequestWithService(
		t,
		service,
		toolsCallRequest(
			"39850000-0000-0000-0000-000000000400",
			"engram_restore",
			map[string]any{"engram_id": notFoundID.String()},
		),
	)
	notFoundPayload := errorPayloadFromFrame(t, notFoundFrame)
	requireErrorCode(t, notFoundPayload, -32004)
	assertEngramMutationNotFoundData(t, notFoundPayload, notFoundID)

	internalFrame := runCompatibilityRequestWithService(
		t,
		newEngramRestoreCompatibilityService(
			&fakeEngramRestoreService{err: errors.New("boom")},
		),
		toolsCallRequest(
			"39850000-0000-0000-0000-000000000401",
			"engram_restore",
			map[string]any{"engram_id": notFoundID.String()},
		),
	)
	requireErrorCode(t, errorPayloadFromFrame(t, internalFrame), -32603)
}

func newEngramRestoreCompatibilityService(service EngramRestoreService) Service {
	return NewCompatibilityServiceWithDependencies(
		"1.2.3",
		CompatibilityServiceDependencies{EngramRestore: service},
	)
}

func engramRestoreResultFromFrame(
	t *testing.T,
	frame Frame,
	asToolsCallPath bool,
) EngramRestoreResponse {
	t.Helper()
	result := resultPayloadFromFrame(t, frame)
	payload := result
	if asToolsCallPath {
		payload = mapFromMap(t, result, "structuredContent")
	}
	restoreResult, ok := payload["result"].(EngramRestoreResponse)
	if !ok {
		t.Fatalf("expected engram-restore result payload")
	}
	return restoreResult
}

type fakeEngramRestoreService struct {
	response *EngramRestoreResponse
	err      error
	call     EngramRestoreRequest
}

func (service *fakeEngramRestoreService) RestoreEngram(
	_ context.Context,
	request EngramRestoreRequest,
) (*EngramRestoreResponse, error) {
	service.call = request
	if service.err != nil {
		return nil, service.err
	}
	if service.response == nil {
		return nil, nil
	}
	response := *service.response
	return &response, nil
}
