package mcp

import (
	"context"
	"errors"
	"reflect"
	"testing"
	"time"

	"engram/internal/chat"
	"engram/internal/models"

	"github.com/google/uuid"
)

func TestCompatibilityServiceChatSaveSessionAsEngramParity(t *testing.T) {
	actorUserID := uuid.MustParse("39090000-0000-0000-0000-000000000390")
	sessionID := uuid.MustParse("39090000-0000-0000-0000-000000000391")
	saved := &models.SaveSessionAsEngramResponse{
		EngramID:  uuid.MustParse("39090000-0000-0000-0000-000000000392"),
		SessionID: sessionID,
		CreatedAt: time.Unix(1700003901, 0).UTC(),
	}
	saveService := &fakeSessionSaveAsEngramService{response: saved}
	params := map[string]any{
		"session_id":       sessionID.String(),
		"title":            "Session Snapshot",
		"abstract":         "Summary",
		"visibility_scope": "project",
		"tags":             []any{"alpha", "beta"},
		"keywords":         []any{"plan", "risk"},
	}

	testCases := []struct {
		name            string
		request         StreamCallRequest
		asToolsCallPath bool
	}{
		{
			name:            "direct",
			request:         directToolRequest(actorUserID.String(), "chat.save_as_engram", params),
			asToolsCallPath: false,
		},
		{
			name:            "tools call",
			request:         toolsCallRequest(actorUserID.String(), "chat_save_as_engram", params),
			asToolsCallPath: true,
		},
	}

	for _, testCase := range testCases {
		testCase := testCase
		t.Run(testCase.name, func(t *testing.T) {
			frame := runCompatibilityRequestWithService(
				t,
				newSessionSaveAsEngramCompatibilityService(saveService),
				testCase.request,
			)
			savedPayload := savedEngramFromFrame(t, frame, testCase.asToolsCallPath)
			if !reflect.DeepEqual(*saved, savedPayload) {
				t.Fatalf("expected saved-engram payload to match service output")
			}
			if saveService.call.ActorUserID != actorUserID {
				t.Fatalf("expected actor user id forwarded")
			}
			if saveService.call.SessionID != sessionID {
				t.Fatalf("expected session id forwarded")
			}
			expectedPayload := models.SaveSessionAsEngramRequest{
				Title:           "Session Snapshot",
				Abstract:        "Summary",
				VisibilityScope: models.VisibilityScopeProject,
				Tags:            []string{"alpha", "beta"},
				Keywords:        []string{"plan", "risk"},
			}
			if !reflect.DeepEqual(expectedPayload, saveService.call.Payload) {
				t.Fatalf("expected save-as-engram payload to be forwarded")
			}
		})
	}
}

func TestCompatibilityServiceChatSaveSessionAsEngramUsesDefaults(t *testing.T) {
	sessionID := uuid.MustParse("39100000-0000-0000-0000-000000000390")
	saveService := &fakeSessionSaveAsEngramService{
		response: &models.SaveSessionAsEngramResponse{
			EngramID:  uuid.MustParse("39100000-0000-0000-0000-000000000391"),
			SessionID: sessionID,
			CreatedAt: time.Unix(1700003910, 0).UTC(),
		},
	}
	frame := runCompatibilityRequestWithService(
		t,
		newSessionSaveAsEngramCompatibilityService(saveService),
		directToolRequest(
			"39100000-0000-0000-0000-000000000392",
			"chat.save_as_engram",
			map[string]any{"session_id": sessionID.String()},
		),
	)
	_ = savedEngramFromFrame(t, frame, false)
	expected := models.SaveSessionAsEngramRequest{
		Title:           "Session Snapshot",
		Abstract:        "",
		VisibilityScope: models.VisibilityScopePrivate,
		Tags:            []string{},
		Keywords:        []string{},
	}
	if !reflect.DeepEqual(expected, saveService.call.Payload) {
		t.Fatalf("expected default save-as-engram payload")
	}
}

func TestCompatibilityServiceChatSaveSessionAsEngramValidationErrors(t *testing.T) {
	service := newSessionSaveAsEngramCompatibilityService(&fakeSessionSaveAsEngramService{})
	testCases := []struct {
		name   string
		params map[string]any
	}{
		{name: "missing session id", params: map[string]any{}},
		{name: "invalid session id", params: map[string]any{"session_id": "bad"}},
		{name: "invalid visibility scope", params: map[string]any{"session_id": uuid.NewString(), "visibility_scope": "bad"}},
		{name: "invalid tags type", params: map[string]any{"session_id": uuid.NewString(), "tags": "bad"}},
		{name: "invalid tag item", params: map[string]any{"session_id": uuid.NewString(), "tags": []any{"ok", 1}}},
		{name: "invalid keywords type", params: map[string]any{"session_id": uuid.NewString(), "keywords": "bad"}},
	}

	for _, testCase := range testCases {
		testCase := testCase
		t.Run(testCase.name, func(t *testing.T) {
			frame := runCompatibilityRequestWithService(
				t,
				service,
				toolsCallRequest(
					"39110000-0000-0000-0000-000000000390",
					"chat_save_as_engram",
					testCase.params,
				),
			)
			requireErrorCode(t, errorPayloadFromFrame(t, frame), -32602)
		})
	}
}

func TestCompatibilityServiceChatSaveSessionAsEngramNotFoundAndErrorMappings(t *testing.T) {
	sessionID := uuid.MustParse("39120000-0000-0000-0000-000000000390")

	notFoundFrame := runCompatibilityRequestWithService(
		t,
		newSessionSaveAsEngramCompatibilityService(&fakeSessionSaveAsEngramService{}),
		toolsCallRequest(
			"39120000-0000-0000-0000-000000000391",
			"chat_save_as_engram",
			map[string]any{"session_id": sessionID.String()},
		),
	)
	notFoundPayload := errorPayloadFromFrame(t, notFoundFrame)
	requireErrorCode(t, notFoundPayload, -32602)
	assertNotFoundErrorData(t, notFoundPayload)

	validationFrame := runCompatibilityRequestWithService(
		t,
		newSessionSaveAsEngramCompatibilityService(
			&fakeSessionSaveAsEngramService{
				err: chat.NewChatValidationError("Cannot save an empty chat session as engram"),
			},
		),
		toolsCallRequest(
			"39120000-0000-0000-0000-000000000392",
			"chat_save_as_engram",
			map[string]any{"session_id": sessionID.String()},
		),
	)
	validationPayload := errorPayloadFromFrame(t, validationFrame)
	requireErrorCode(t, validationPayload, -32010)
	assertErrorMessage(t, validationPayload, "Cannot save an empty chat session as engram")
	assertErrorStatusCode(t, validationPayload, 400)

	internalFrame := runCompatibilityRequestWithService(
		t,
		newSessionSaveAsEngramCompatibilityService(
			&fakeSessionSaveAsEngramService{err: errors.New("boom")},
		),
		toolsCallRequest(
			"39120000-0000-0000-0000-000000000393",
			"chat_save_as_engram",
			map[string]any{"session_id": sessionID.String()},
		),
	)
	requireErrorCode(t, errorPayloadFromFrame(t, internalFrame), -32603)
}

func newSessionSaveAsEngramCompatibilityService(
	saveService SessionSaveAsEngramService,
) Service {
	return NewCompatibilityServiceWithDependencies(
		"1.2.3",
		CompatibilityServiceDependencies{SessionSaveAsEngram: saveService},
	)
}

func savedEngramFromFrame(
	t *testing.T,
	frame Frame,
	asToolsCallPath bool,
) models.SaveSessionAsEngramResponse {
	t.Helper()
	result := resultPayloadFromFrame(t, frame)
	payload := result
	if asToolsCallPath {
		payload = mapFromMap(t, result, "structuredContent")
	}
	saved, ok := payload["saved_engram"].(models.SaveSessionAsEngramResponse)
	if !ok {
		t.Fatalf("expected saved_engram payload")
	}
	return saved
}

type fakeSessionSaveAsEngramService struct {
	response *models.SaveSessionAsEngramResponse
	err      error
	call     SessionSaveAsEngramRequest
}

func (service *fakeSessionSaveAsEngramService) SaveSessionAsEngram(
	_ context.Context,
	request SessionSaveAsEngramRequest,
) (*models.SaveSessionAsEngramResponse, error) {
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
