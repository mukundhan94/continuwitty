package mcp

import (
	"context"
	"errors"
	"reflect"
	"testing"

	"engram/internal/models"

	"github.com/google/uuid"
)

func TestCompatibilityServiceChatDeleteSessionParity(t *testing.T) {
	actorUserID := uuid.MustParse("39800000-0000-0000-0000-000000000398")
	sessionID := uuid.MustParse("39800000-0000-0000-0000-000000000399")
	deleteService := &fakeSessionDeleteService{
		response: &SessionDeleteResponse{
			SessionID:            sessionID,
			Deleted:              true,
			LinkedEngramsDeleted: 2,
		},
	}
	params := map[string]any{
		"session_id":            sessionID.String(),
		"delete_linked_engrams": true,
		"reason":                "cleanup",
	}

	testCases := []struct {
		name            string
		request         StreamCallRequest
		asToolsCallPath bool
		expectedRole    models.UserRole
	}{
		{
			name:            "direct",
			request:         directToolRequest(actorUserID.String(), "chat.delete_session", params),
			asToolsCallPath: false,
			expectedRole:    models.UserRoleViewer,
		},
		{
			name:            "tools call",
			request:         toolsCallRequest(actorUserID.String(), "chat_delete_session", params),
			asToolsCallPath: true,
			expectedRole:    models.UserRoleAnalyst,
		},
	}

	for _, testCase := range testCases {
		testCase := testCase
		t.Run(testCase.name, func(t *testing.T) {
			frame := runCompatibilityRequestWithService(
				t,
				newSessionDeleteCompatibilityService(deleteService),
				testCase.request,
			)
			result := deleteSessionResultFromFrame(t, frame, testCase.asToolsCallPath)
			if !reflect.DeepEqual(*deleteService.response, result) {
				t.Fatalf("expected delete result payload to match service output")
			}
			assertDeleteSessionCall(
				t,
				deleteService.call,
				deleteSessionCallExpectation{
					actorUserID:         actorUserID,
					actorRole:           testCase.expectedRole,
					sessionID:           sessionID,
					deleteLinkedEngrams: true,
					reason:              stringPtr("cleanup"),
				},
			)
		})
	}
}

func TestCompatibilityServiceChatDeleteSessionUsesDefaults(t *testing.T) {
	actorUserID := uuid.MustParse("39900000-0000-0000-0000-000000000399")
	sessionID := uuid.MustParse("39900000-0000-0000-0000-000000000390")
	deleteService := &fakeSessionDeleteService{
		response: &SessionDeleteResponse{
			SessionID:            sessionID,
			Deleted:              true,
			LinkedEngramsDeleted: 0,
		},
	}
	frame := runCompatibilityRequestWithService(
		t,
		newSessionDeleteCompatibilityService(deleteService),
		directToolRequest(
			actorUserID.String(),
			"chat.delete_session",
			map[string]any{"session_id": sessionID.String()},
		),
	)
	_ = deleteSessionResultFromFrame(t, frame, false)
	if deleteService.call.DeleteLinkedEngrams {
		t.Fatalf("expected delete_linked_engrams default to false")
	}
	if deleteService.call.Reason != nil {
		t.Fatalf("expected reason to remain nil when omitted")
	}
}

func TestCompatibilityServiceChatDeleteSessionValidationErrors(t *testing.T) {
	service := newSessionDeleteCompatibilityService(&fakeSessionDeleteService{})
	testCases := []struct {
		name   string
		params map[string]any
	}{
		{name: "missing session id", params: map[string]any{}},
		{name: "invalid session id", params: map[string]any{"session_id": "bad"}},
		{name: "invalid delete linked engrams", params: map[string]any{"session_id": uuid.NewString(), "delete_linked_engrams": "bad"}},
		{name: "invalid reason", params: map[string]any{"session_id": uuid.NewString(), "reason": 123}},
	}

	for _, testCase := range testCases {
		testCase := testCase
		t.Run(testCase.name, func(t *testing.T) {
			frame := runCompatibilityRequestWithService(
				t,
				service,
				toolsCallRequest(
					"39010000-0000-0000-0000-000000000390",
					"chat_delete_session",
					testCase.params,
				),
			)
			requireErrorCode(t, errorPayloadFromFrame(t, frame), -32602)
		})
	}
}

func TestCompatibilityServiceChatDeleteSessionNotFoundAndServiceError(t *testing.T) {
	sessionID := uuid.MustParse("39020000-0000-0000-0000-000000000390")

	notFoundFrame := runCompatibilityRequestWithService(
		t,
		newSessionDeleteCompatibilityService(&fakeSessionDeleteService{}),
		toolsCallRequest(
			"39020000-0000-0000-0000-000000000391",
			"chat_delete_session",
			map[string]any{"session_id": sessionID.String()},
		),
	)
	notFoundPayload := errorPayloadFromFrame(t, notFoundFrame)
	requireErrorCode(t, notFoundPayload, -32602)
	assertDeleteSessionNotFoundErrorData(t, notFoundPayload)

	serviceErrorFrame := runCompatibilityRequestWithService(
		t,
		newSessionDeleteCompatibilityService(&fakeSessionDeleteService{err: errors.New("boom")}),
		toolsCallRequest(
			"39020000-0000-0000-0000-000000000392",
			"chat_delete_session",
			map[string]any{"session_id": sessionID.String()},
		),
	)
	requireErrorCode(t, errorPayloadFromFrame(t, serviceErrorFrame), -32603)
}

func newSessionDeleteCompatibilityService(sessionDeleteService SessionDeleteService) Service {
	return NewCompatibilityServiceWithDependencies(
		"1.2.3",
		CompatibilityServiceDependencies{SessionDelete: sessionDeleteService},
	)
}

func deleteSessionResultFromFrame(
	t *testing.T,
	frame Frame,
	asToolsCallPath bool,
) SessionDeleteResponse {
	t.Helper()
	result := resultPayloadFromFrame(t, frame)
	payload := result
	if asToolsCallPath {
		payload = mapFromMap(t, result, "structuredContent")
	}
	deleteResult, ok := payload["result"].(SessionDeleteResponse)
	if !ok {
		t.Fatalf("expected delete-session result payload")
	}
	return deleteResult
}

func assertDeleteSessionNotFoundErrorData(t *testing.T, errorPayload map[string]any) {
	t.Helper()
	data, ok := errorPayload["data"].(map[string]any)
	if !ok {
		t.Fatalf("expected error data payload")
	}
	if data["status_code"] != 404 {
		t.Fatalf("expected status_code 404 in error data")
	}
	if data["detail"] != "Session not found" {
		t.Fatalf("expected session-not-found detail in error data")
	}
}

type deleteSessionCallExpectation struct {
	actorUserID         uuid.UUID
	actorRole           models.UserRole
	sessionID           uuid.UUID
	deleteLinkedEngrams bool
	reason              *string
}

func assertDeleteSessionCall(
	t *testing.T,
	call SessionDeleteRequest,
	expected deleteSessionCallExpectation,
) {
	t.Helper()
	if call.ActorUserID != expected.actorUserID {
		t.Fatalf("expected actor user id forwarded")
	}
	if call.ActorRole != expected.actorRole {
		t.Fatalf("expected normalized actor role forwarded")
	}
	if call.SessionID != expected.sessionID {
		t.Fatalf("expected session id forwarded")
	}
	if call.DeleteLinkedEngrams != expected.deleteLinkedEngrams {
		t.Fatalf("expected delete_linked_engrams to be forwarded")
	}
	assertOptionalDeleteReason(t, call.Reason, expected.reason)
}

func assertOptionalDeleteReason(t *testing.T, actual *string, expected *string) {
	t.Helper()
	if expected == nil {
		if actual != nil {
			t.Fatalf("expected reason to be omitted")
		}
		return
	}
	if actual == nil || *actual != *expected {
		t.Fatalf("expected reason to be forwarded")
	}
}

type fakeSessionDeleteService struct {
	response *SessionDeleteResponse
	err      error
	call     SessionDeleteRequest
}

func (service *fakeSessionDeleteService) DeleteSession(
	_ context.Context,
	request SessionDeleteRequest,
) (*SessionDeleteResponse, error) {
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
