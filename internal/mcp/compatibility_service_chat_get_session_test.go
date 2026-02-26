package mcp

import (
	"context"
	"errors"
	"reflect"
	"testing"

	"engram/internal/models"

	"github.com/google/uuid"
)

func TestCompatibilityServiceChatGetSessionParity(t *testing.T) {
	actorUserID := uuid.MustParse("27000000-0000-0000-0000-000000000270")
	sessionID := uuid.MustParse("27000000-0000-0000-0000-000000000271")
	expectedSession := models.ChatSessionRecord{
		SessionID: sessionID,
		ProjectID: "proj-alpha",
		Title:     "Alpha session",
	}
	service := &fakeSessionGetService{session: &expectedSession}

	testCases := []struct {
		name            string
		request         StreamCallRequest
		asToolsCallPath bool
	}{
		{
			name: "direct",
			request: directToolRequest(
				actorUserID.String(),
				"chat.get_session",
				map[string]any{"session_id": sessionID.String()},
			),
			asToolsCallPath: false,
		},
		{
			name: "tools call",
			request: toolsCallRequest(
				actorUserID.String(),
				"chat_get_session",
				map[string]any{"session_id": sessionID.String()},
			),
			asToolsCallPath: true,
		},
	}

	for _, testCase := range testCases {
		testCase := testCase
		t.Run(testCase.name, func(t *testing.T) {
			frame := runCompatibilityRequestWithService(
				t,
				newSessionGetCompatibilityService(service),
				testCase.request,
			)
			session := chatSessionFromFrame(t, frame, testCase.asToolsCallPath)
			if !reflect.DeepEqual(expectedSession, session) {
				t.Fatalf("expected session payload to match service output")
			}
			if service.call.actorUserID != actorUserID {
				t.Fatalf("expected actor user id forwarded")
			}
			if service.call.sessionID != sessionID {
				t.Fatalf("expected session id forwarded")
			}
		})
	}
}

func TestCompatibilityServiceChatGetSessionErrors(t *testing.T) {
	notFoundFrame := runCompatibilityRequestWithService(
		t,
		newSessionGetCompatibilityService(&fakeSessionGetService{}),
		toolsCallRequest(
			"27100000-0000-0000-0000-000000000271",
			"chat_get_session",
			map[string]any{"session_id": "27100000-0000-0000-0000-000000000272"},
		),
	)
	notFoundPayload := errorPayloadFromFrame(t, notFoundFrame)
	requireErrorCode(t, notFoundPayload, -32602)
	assertNotFoundErrorData(t, notFoundPayload)

	invalidRequest := toolsCallRequest(
		"27200000-0000-0000-0000-000000000272",
		"chat_get_session",
		map[string]any{"session_id": "bad"},
	)
	invalidFrame := runCompatibilityRequestWithService(
		t,
		newSessionGetCompatibilityService(&fakeSessionGetService{}),
		invalidRequest,
	)
	invalidPayload := errorPayloadFromFrame(t, invalidFrame)
	requireErrorCode(t, invalidPayload, -32602)

	missingRequest := toolsCallRequest(
		"27300000-0000-0000-0000-000000000273",
		"chat_get_session",
		map[string]any{},
	)
	missingFrame := runCompatibilityRequestWithService(
		t,
		newSessionGetCompatibilityService(&fakeSessionGetService{}),
		missingRequest,
	)
	missingPayload := errorPayloadFromFrame(t, missingFrame)
	requireErrorCode(t, missingPayload, -32602)
}

func TestCompatibilityServiceChatGetSessionMapsServiceError(t *testing.T) {
	frame := runCompatibilityRequestWithService(
		t,
		newSessionGetCompatibilityService(&fakeSessionGetService{err: errors.New("boom")}),
		toolsCallRequest(
			"27400000-0000-0000-0000-000000000274",
			"chat_get_session",
			map[string]any{"session_id": "27400000-0000-0000-0000-000000000275"},
		),
	)
	errorPayload := errorPayloadFromFrame(t, frame)
	requireErrorCode(t, errorPayload, -32603)
}

func newSessionGetCompatibilityService(sessionGetService SessionGetService) Service {
	return NewCompatibilityServiceWithDependencies(
		"1.2.3",
		CompatibilityServiceDependencies{SessionGet: sessionGetService},
	)
}

func chatSessionFromFrame(t *testing.T, frame Frame, asToolsCallPath bool) models.ChatSessionRecord {
	t.Helper()
	result := resultPayloadFromFrame(t, frame)
	payload := result
	if asToolsCallPath {
		payload = mapFromMap(t, result, "structuredContent")
	}
	session, ok := payload["session"].(models.ChatSessionRecord)
	if !ok {
		t.Fatalf("expected session payload")
	}
	return session
}

func assertNotFoundErrorData(t *testing.T, errorPayload map[string]any) {
	t.Helper()
	data, ok := errorPayload["data"].(map[string]any)
	if !ok {
		t.Fatalf("expected error data payload")
	}
	if data["status_code"] != 404 {
		t.Fatalf("expected status_code 404 in error data")
	}
	if data["detail"] != "Chat session not found" {
		t.Fatalf("expected not-found detail in error data")
	}
}

type sessionGetCall struct {
	actorUserID uuid.UUID
	sessionID   uuid.UUID
}

type fakeSessionGetService struct {
	session *models.ChatSessionRecord
	err     error
	call    sessionGetCall
}

func (service *fakeSessionGetService) GetSession(
	_ context.Context,
	request SessionGetRequest,
) (*models.ChatSessionRecord, error) {
	service.call = sessionGetCall{
		actorUserID: request.ActorUserID,
		sessionID:   request.SessionID,
	}
	if service.err != nil {
		return nil, service.err
	}
	if service.session == nil {
		return nil, nil
	}
	record := *service.session
	return &record, nil
}
