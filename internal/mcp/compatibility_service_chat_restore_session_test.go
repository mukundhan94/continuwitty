package mcp

import (
	"context"
	"errors"
	"reflect"
	"testing"

	"engram/internal/models"

	"github.com/google/uuid"
)

func TestCompatibilityServiceChatRestoreSessionParity(t *testing.T) {
	actorUserID := uuid.MustParse("39030000-0000-0000-0000-000000000390")
	sessionID := uuid.MustParse("39030000-0000-0000-0000-000000000391")
	restoreService := &fakeSessionRestoreService{
		response: &SessionRestoreResponse{
			SessionID: sessionID,
			Restored:  true,
		},
	}
	params := map[string]any{"session_id": sessionID.String()}

	testCases := []struct {
		name            string
		request         StreamCallRequest
		asToolsCallPath bool
		expectedRole    models.UserRole
	}{
		{
			name:            "direct",
			request:         directToolRequest(actorUserID.String(), "chat.restore_session", params),
			asToolsCallPath: false,
			expectedRole:    models.UserRoleViewer,
		},
		{
			name:            "tools call",
			request:         toolsCallRequest(actorUserID.String(), "chat_restore_session", params),
			asToolsCallPath: true,
			expectedRole:    models.UserRoleAnalyst,
		},
	}

	for _, testCase := range testCases {
		testCase := testCase
		t.Run(testCase.name, func(t *testing.T) {
			frame := runCompatibilityRequestWithService(
				t,
				newSessionRestoreCompatibilityService(restoreService),
				testCase.request,
			)
			result := restoreSessionResultFromFrame(t, frame, testCase.asToolsCallPath)
			if !reflect.DeepEqual(*restoreService.response, result) {
				t.Fatalf("expected restore result payload to match service output")
			}
			if restoreService.call.ActorUserID != actorUserID {
				t.Fatalf("expected actor user id forwarded")
			}
			if restoreService.call.ActorRole != testCase.expectedRole {
				t.Fatalf("expected normalized actor role forwarded")
			}
			if restoreService.call.SessionID != sessionID {
				t.Fatalf("expected session id forwarded")
			}
		})
	}
}

func TestCompatibilityServiceChatRestoreSessionValidationErrors(t *testing.T) {
	service := newSessionRestoreCompatibilityService(&fakeSessionRestoreService{})
	testCases := []struct {
		name   string
		params map[string]any
	}{
		{name: "missing session id", params: map[string]any{}},
		{name: "invalid session id", params: map[string]any{"session_id": "bad"}},
	}

	for _, testCase := range testCases {
		testCase := testCase
		t.Run(testCase.name, func(t *testing.T) {
			frame := runCompatibilityRequestWithService(
				t,
				service,
				toolsCallRequest(
					"39030000-0000-0000-0000-000000000392",
					"chat_restore_session",
					testCase.params,
				),
			)
			requireErrorCode(t, errorPayloadFromFrame(t, frame), -32602)
		})
	}
}

func TestCompatibilityServiceChatRestoreSessionNotFoundAndServiceError(t *testing.T) {
	sessionID := uuid.MustParse("39040000-0000-0000-0000-000000000390")

	notFoundFrame := runCompatibilityRequestWithService(
		t,
		newSessionRestoreCompatibilityService(&fakeSessionRestoreService{}),
		toolsCallRequest(
			"39040000-0000-0000-0000-000000000391",
			"chat_restore_session",
			map[string]any{"session_id": sessionID.String()},
		),
	)
	notFoundPayload := errorPayloadFromFrame(t, notFoundFrame)
	requireErrorCode(t, notFoundPayload, -32602)
	assertDeleteSessionNotFoundErrorData(t, notFoundPayload)

	serviceErrorFrame := runCompatibilityRequestWithService(
		t,
		newSessionRestoreCompatibilityService(&fakeSessionRestoreService{err: errors.New("boom")}),
		toolsCallRequest(
			"39040000-0000-0000-0000-000000000392",
			"chat_restore_session",
			map[string]any{"session_id": sessionID.String()},
		),
	)
	requireErrorCode(t, errorPayloadFromFrame(t, serviceErrorFrame), -32603)
}

func newSessionRestoreCompatibilityService(sessionRestoreService SessionRestoreService) Service {
	return NewCompatibilityServiceWithDependencies(
		"1.2.3",
		CompatibilityServiceDependencies{SessionRestore: sessionRestoreService},
	)
}

func restoreSessionResultFromFrame(
	t *testing.T,
	frame Frame,
	asToolsCallPath bool,
) SessionRestoreResponse {
	t.Helper()
	result := resultPayloadFromFrame(t, frame)
	payload := result
	if asToolsCallPath {
		payload = mapFromMap(t, result, "structuredContent")
	}
	restoreResult, ok := payload["result"].(SessionRestoreResponse)
	if !ok {
		t.Fatalf("expected restore-session result payload")
	}
	return restoreResult
}

type fakeSessionRestoreService struct {
	response *SessionRestoreResponse
	err      error
	call     SessionRestoreRequest
}

func (service *fakeSessionRestoreService) RestoreSession(
	_ context.Context,
	request SessionRestoreRequest,
) (*SessionRestoreResponse, error) {
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
