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

func TestCompatibilityServiceChatContinueSessionParity(t *testing.T) {
	actorUserID := uuid.MustParse("39400000-0000-0000-0000-000000000394")
	sessionID := uuid.MustParse("39400000-0000-0000-0000-000000000395")
	carriedEngramID := uuid.MustParse("39400000-0000-0000-0000-000000000396")
	continued := &models.ContinueSessionResponse{
		Session: models.ChatSessionRecord{
			SessionID:               uuid.MustParse("39400000-0000-0000-0000-000000000397"),
			OwnerUserID:             actorUserID,
			ProjectID:               "proj-alpha",
			Title:                   "Planning (continued)",
			Provider:                models.ChatProviderOpenAI,
			ModelID:                 "gpt-4o-mini",
			SystemPrompt:            "",
			VisibilityScope:         models.VisibilityScopePrivate,
			AutosaveEnabled:         false,
			AutosaveStrategy:        models.ChatAutosaveStrategyOff,
			AutosaveIntervalMinutes: 30,
			AutosaveMinMessages:     6,
			RetentionDays:           30,
			RetentionMaxSnapshots:   60,
			CreatedAt:               time.Unix(1700003900, 0).UTC(),
			UpdatedAt:               time.Unix(1700003900, 0).UTC(),
		},
		CarriedEngramIDs: []uuid.UUID{carriedEngramID},
	}
	continueService := &fakeSessionContinueService{continuation: continued}
	params := map[string]any{
		"session_id": sessionID.String(),
		"title":      "Planning (continued)",
	}

	testCases := []struct {
		name            string
		request         StreamCallRequest
		asToolsCallPath bool
	}{
		{
			name:            "direct",
			request:         directToolRequest(actorUserID.String(), "chat.continue_session", params),
			asToolsCallPath: false,
		},
		{
			name:            "tools call",
			request:         toolsCallRequest(actorUserID.String(), "chat_continue_session", params),
			asToolsCallPath: true,
		},
	}

	for _, testCase := range testCases {
		testCase := testCase
		t.Run(testCase.name, func(t *testing.T) {
			frame := runCompatibilityRequestWithService(
				t,
				newSessionContinueCompatibilityService(continueService),
				testCase.request,
			)
			response := continuationFromFrame(t, frame, testCase.asToolsCallPath)
			if !reflect.DeepEqual(*continued, response) {
				t.Fatalf("expected continuation payload to match service output")
			}
			if continueService.call.ActorUserID != actorUserID {
				t.Fatalf("expected actor user id forwarded")
			}
			if continueService.call.SessionID != sessionID {
				t.Fatalf("expected session id forwarded")
			}
			if continueService.call.Payload.Title == nil || *continueService.call.Payload.Title != "Planning (continued)" {
				t.Fatalf("expected title to be forwarded")
			}
		})
	}
}

func TestCompatibilityServiceChatContinueSessionUsesNilTitleByDefault(t *testing.T) {
	actorUserID := uuid.MustParse("39500000-0000-0000-0000-000000000395")
	sessionID := uuid.MustParse("39500000-0000-0000-0000-000000000396")
	continueService := &fakeSessionContinueService{
		continuation: &models.ContinueSessionResponse{
			Session: models.ChatSessionRecord{
				SessionID: uuid.MustParse("39500000-0000-0000-0000-000000000397"),
				Title:     "Default continued title",
			},
		},
	}
	frame := runCompatibilityRequestWithService(
		t,
		newSessionContinueCompatibilityService(continueService),
		directToolRequest(
			actorUserID.String(),
			"chat.continue_session",
			map[string]any{"session_id": sessionID.String()},
		),
	)
	_ = continuationFromFrame(t, frame, false)
	if continueService.call.Payload.Title != nil {
		t.Fatalf("expected continue title to remain nil when omitted")
	}
}

func TestCompatibilityServiceChatContinueSessionValidationErrors(t *testing.T) {
	service := newSessionContinueCompatibilityService(&fakeSessionContinueService{})
	testCases := []struct {
		name   string
		params map[string]any
	}{
		{name: "missing session id", params: map[string]any{}},
		{name: "invalid session id", params: map[string]any{"session_id": "bad"}},
		{name: "invalid title type", params: map[string]any{"session_id": uuid.NewString(), "title": 42}},
	}

	for _, testCase := range testCases {
		testCase := testCase
		t.Run(testCase.name, func(t *testing.T) {
			frame := runCompatibilityRequestWithService(
				t,
				service,
				toolsCallRequest(
					"39600000-0000-0000-0000-000000000396",
					"chat_continue_session",
					testCase.params,
				),
			)
			requireErrorCode(t, errorPayloadFromFrame(t, frame), -32602)
		})
	}
}

func TestCompatibilityServiceChatContinueSessionNotFoundAndServiceError(t *testing.T) {
	sessionID := uuid.MustParse("39700000-0000-0000-0000-000000000397")

	notFoundFrame := runCompatibilityRequestWithService(
		t,
		newSessionContinueCompatibilityService(&fakeSessionContinueService{}),
		toolsCallRequest(
			"39700000-0000-0000-0000-000000000398",
			"chat_continue_session",
			map[string]any{"session_id": sessionID.String()},
		),
	)
	notFoundPayload := errorPayloadFromFrame(t, notFoundFrame)
	requireErrorCode(t, notFoundPayload, -32602)
	assertNotFoundErrorData(t, notFoundPayload)

	serviceErrorFrame := runCompatibilityRequestWithService(
		t,
		newSessionContinueCompatibilityService(
			&fakeSessionContinueService{err: errors.New("boom")},
		),
		toolsCallRequest(
			"39700000-0000-0000-0000-000000000399",
			"chat_continue_session",
			map[string]any{"session_id": sessionID.String()},
		),
	)
	requireErrorCode(t, errorPayloadFromFrame(t, serviceErrorFrame), -32603)
}

func newSessionContinueCompatibilityService(sessionContinueService SessionContinueService) Service {
	return NewCompatibilityServiceWithDependencies(
		"1.2.3",
		CompatibilityServiceDependencies{SessionContinue: sessionContinueService},
	)
}

func continuationFromFrame(
	t *testing.T,
	frame Frame,
	asToolsCallPath bool,
) models.ContinueSessionResponse {
	t.Helper()
	result := resultPayloadFromFrame(t, frame)
	payload := result
	if asToolsCallPath {
		payload = mapFromMap(t, result, "structuredContent")
	}
	continuation, ok := payload["continuation"].(models.ContinueSessionResponse)
	if !ok {
		t.Fatalf("expected continuation payload")
	}
	return continuation
}

type fakeSessionContinueService struct {
	continuation *models.ContinueSessionResponse
	err          error
	call         SessionContinueRequest
}

func (service *fakeSessionContinueService) ContinueSession(
	_ context.Context,
	request SessionContinueRequest,
) (*models.ContinueSessionResponse, error) {
	service.call = request
	if service.err != nil {
		return nil, service.err
	}
	if service.continuation == nil {
		return nil, nil
	}
	response := *service.continuation
	return &response, nil
}
