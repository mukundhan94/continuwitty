package mcp

import (
	"context"
	"errors"
	"reflect"
	"testing"
	"time"

	"engram/internal/chat"
	"engram/internal/models"
	"engram/internal/projects"

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
	service := newChatSaveAsEngramCompatibilityService(
		&fakeSessionSaveAsEngramService{},
		&fakeEngramCreateConversationService{},
	)
	testCases := []struct {
		name   string
		params map[string]any
	}{
		{name: "missing conversation markdown", params: map[string]any{}},
		{name: "invalid session id", params: map[string]any{"session_id": "bad"}},
		{name: "invalid visibility scope", params: map[string]any{"session_id": uuid.NewString(), "visibility_scope": "bad"}},
		{name: "invalid tags type", params: map[string]any{"session_id": uuid.NewString(), "tags": "bad"}},
		{name: "invalid tag item", params: map[string]any{"session_id": uuid.NewString(), "tags": []any{"ok", 1}}},
		{name: "invalid keywords type", params: map[string]any{"session_id": uuid.NewString(), "keywords": "bad"}},
		{name: "invalid source session", params: map[string]any{"conversation_markdown": "details", "source_session_id": "bad"}},
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

func TestCompatibilityServiceChatSaveSessionAsEngramConversationFallbackParity(t *testing.T) {
	actorUserID := uuid.MustParse("39130000-0000-0000-0000-000000000390")
	sourceSessionID := uuid.MustParse("39130000-0000-0000-0000-000000000391")
	response := chatSaveFallbackResponse()
	params := chatSaveFallbackParams(sourceSessionID)
	testCases := []struct {
		name            string
		request         StreamCallRequest
		asToolsCallPath bool
		expectedRole    models.UserRole
	}{
		{
			name:            "direct",
			request:         directToolRequest(actorUserID.String(), "chat.save_as_engram", params),
			asToolsCallPath: false,
			expectedRole:    models.UserRoleViewer,
		},
		{
			name:            "tools call",
			request:         toolsCallRequest(actorUserID.String(), "chat_save_as_engram", params),
			asToolsCallPath: true,
			expectedRole:    models.UserRoleAnalyst,
		},
	}

	for _, testCase := range testCases {
		testCase := testCase
		t.Run(testCase.name, func(t *testing.T) {
			assertChatSaveFallbackCase(
				t,
				response,
				chatSaveFallbackAssertionInput{
					request:         testCase.request,
					asToolsCallPath: testCase.asToolsCallPath,
					expectedRequest: chatSaveFallbackExpectedRequest(actorUserID, testCase.expectedRole, sourceSessionID),
				},
			)
		})
	}
}

func TestCompatibilityServiceChatSaveSessionAsEngramConversationFallbackUsesDefaultTitle(t *testing.T) {
	conversationService := &fakeEngramCreateConversationService{
		response: &EngramCreateFromConversationResponse{
			Engram: models.EngramCreateResponse{
				EngramID:  uuid.MustParse("39131000-0000-0000-0000-000000000392"),
				CreatedAt: time.Unix(1700003914, 0).UTC(),
			},
			EnrichmentReport: map[string]any{},
		},
	}
	frame := runCompatibilityRequestWithService(
		t,
		newChatSaveAsEngramCompatibilityService(nil, conversationService),
		directToolRequest(
			"39131000-0000-0000-0000-000000000390",
			"chat.save_as_engram",
			map[string]any{
				"project_id":            "project-default-title",
				"conversation_markdown": "Details",
			},
		),
	)
	_, _ = conversationSaveEngramFromFrame(t, frame, false)
	if conversationService.call.Title != "Conversation Snapshot" {
		t.Fatalf("expected fallback title default")
	}
}

func TestCompatibilityServiceChatSaveSessionAsEngramConversationFallbackErrorMappings(t *testing.T) {
	request := toolsCallRequest(
		"39132000-0000-0000-0000-000000000390",
		"chat_save_as_engram",
		map[string]any{
			"project_id":            "missing",
			"conversation_markdown": "Details",
		},
	)
	notFoundFrame := runCompatibilityRequestWithService(
		t,
		newChatSaveAsEngramCompatibilityService(
			nil,
			&fakeEngramCreateConversationService{err: projects.ErrProjectNotFound},
		),
		request,
	)
	notFoundPayload := errorPayloadFromFrame(t, notFoundFrame)
	requireErrorCode(t, notFoundPayload, -32602)
	assertErrorStatusCode(t, notFoundPayload, 404)
	assertErrorDetail(t, notFoundPayload, "Project not found")

	internalFrame := runCompatibilityRequestWithService(
		t,
		newChatSaveAsEngramCompatibilityService(
			nil,
			&fakeEngramCreateConversationService{err: errors.New("boom")},
		),
		request,
	)
	requireErrorCode(t, errorPayloadFromFrame(t, internalFrame), -32603)
}

func newSessionSaveAsEngramCompatibilityService(
	saveService SessionSaveAsEngramService,
) Service {
	return newChatSaveAsEngramCompatibilityService(saveService, nil)
}

func newChatSaveAsEngramCompatibilityService(
	saveService SessionSaveAsEngramService,
	conversationService EngramCreateFromConversationService,
) Service {
	return NewCompatibilityServiceWithDependencies(
		"1.2.3",
		CompatibilityServiceDependencies{
			SessionSaveAsEngram:      saveService,
			EngramCreateConversation: conversationService,
		},
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

func conversationSaveEngramFromFrame(
	t *testing.T,
	frame Frame,
	asToolsCallPath bool,
) (models.EngramCreateResponse, map[string]any) {
	t.Helper()
	result := resultPayloadFromFrame(t, frame)
	payload := result
	if asToolsCallPath {
		payload = mapFromMap(t, result, "structuredContent")
	}
	saved, ok := payload["saved_engram"].(models.EngramCreateResponse)
	if !ok {
		t.Fatalf("expected fallback saved_engram payload")
	}
	report := mapFromMap(t, payload, "enrichment_report")
	return saved, report
}

func assertChatSaveFallbackCase(
	t *testing.T,
	response *EngramCreateFromConversationResponse,
	input chatSaveFallbackAssertionInput,
) {
	t.Helper()
	conversationService := &fakeEngramCreateConversationService{response: response}
	frame := runCompatibilityRequestWithService(
		t,
		newChatSaveAsEngramCompatibilityService(nil, conversationService),
		input.request,
	)
	savedPayload, report := conversationSaveEngramFromFrame(t, frame, input.asToolsCallPath)
	if !reflect.DeepEqual(response.Engram, savedPayload) {
		t.Fatalf("expected fallback saved-engram payload")
	}
	if !reflect.DeepEqual(response.EnrichmentReport, report) {
		t.Fatalf("expected fallback enrichment report payload")
	}
	if !reflect.DeepEqual(input.expectedRequest, conversationService.call) {
		t.Fatalf("expected conversation create request forwarded")
	}
}

type chatSaveFallbackAssertionInput struct {
	request         StreamCallRequest
	asToolsCallPath bool
	expectedRequest EngramCreateFromConversationRequest
}

func chatSaveFallbackResponse() *EngramCreateFromConversationResponse {
	return &EngramCreateFromConversationResponse{
		Engram: models.EngramCreateResponse{
			EngramID:           uuid.MustParse("39130000-0000-0000-0000-000000000392"),
			CreatedAt:          time.Unix(1700003913, 0).UTC(),
			ResolvedProjectID:  stringPtr("project-fallback"),
			UsedDefaultProject: false,
		},
		EnrichmentReport: map[string]any{
			"enrichment_applied":   true,
			"auto_tags":            []string{"alpha"},
			"auto_keywords":        []string{"beta"},
			"abstract_derived":     true,
			"resolved_project_id":  "project-fallback",
			"used_default_project": false,
		},
	}
}

func chatSaveFallbackParams(sourceSessionID uuid.UUID) map[string]any {
	return map[string]any{
		"project_id":            " project-fallback ",
		"title":                 " Conversation Snapshot ",
		"abstract":              " summary ",
		"conversation_markdown": " Details ",
		"thread_id":             "thread-1",
		"visibility_scope":      "project",
		"tags":                  []any{"alpha"},
		"keywords":              []any{"beta"},
		"retrieval_text":        "retrieval text",
		"source_session_id":     sourceSessionID.String(),
	}
}

func chatSaveFallbackExpectedRequest(
	actorUserID uuid.UUID,
	actorRole models.UserRole,
	sourceSessionID uuid.UUID,
) EngramCreateFromConversationRequest {
	return EngramCreateFromConversationRequest{
		ActorUserID:          actorUserID,
		ActorRole:            actorRole,
		ProjectID:            "project-fallback",
		ThreadID:             stringPtr("thread-1"),
		Title:                "Conversation Snapshot",
		Abstract:             "summary",
		ConversationMarkdown: "Details",
		Tags:                 []string{"alpha"},
		Keywords:             []string{"beta"},
		VisibilityScope:      string(models.VisibilityScopeProject),
		RetrievalText:        stringPtr("retrieval text"),
		SourceSessionID:      &sourceSessionID,
		EnrichmentOrigin:     "mcp.chat.save_as_engram",
	}
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
