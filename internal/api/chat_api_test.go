package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"
	"time"

	"engram/internal/chat"
	"engram/internal/models"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

type fakeChatStreamService struct {
	streamMessageEvents func(
		ctx context.Context,
		actorUserID uuid.UUID,
		sessionID uuid.UUID,
		payload chat.ChatMessageCreateRequest,
	) ([]chat.StreamEvent, error)
}

func (service fakeChatStreamService) StreamMessageEvents(
	ctx context.Context,
	actorUserID uuid.UUID,
	sessionID uuid.UUID,
	payload chat.ChatMessageCreateRequest,
) ([]chat.StreamEvent, error) {
	if service.streamMessageEvents == nil {
		return []chat.StreamEvent{}, nil
	}
	return service.streamMessageEvents(ctx, actorUserID, sessionID, payload)
}

type fakeChatSessionDerivativeService struct {
	saveSessionAsEngram func(
		ctx context.Context,
		actorUserID uuid.UUID,
		sessionID uuid.UUID,
		payload models.SaveSessionAsEngramRequest,
	) (models.SaveSessionAsEngramResponse, error)
	continueSession func(
		ctx context.Context,
		actorUserID uuid.UUID,
		sessionID uuid.UUID,
		payload models.ContinueSessionRequest,
	) (models.ContinueSessionResponse, error)
}

type saveSessionAsEngramCall struct {
	actorUserID uuid.UUID
	sessionID   uuid.UUID
	payload     models.SaveSessionAsEngramRequest
}

type continueSessionCall struct {
	actorUserID uuid.UUID
	sessionID   uuid.UUID
	payload     models.ContinueSessionRequest
}

func (service fakeChatSessionDerivativeService) SaveSessionAsEngram(
	ctx context.Context,
	actorUserID uuid.UUID,
	sessionID uuid.UUID,
	payload models.SaveSessionAsEngramRequest,
) (models.SaveSessionAsEngramResponse, error) {
	operation := service.saveSessionAsEngram
	if operation == nil {
		return models.SaveSessionAsEngramResponse{}, nil
	}
	result, err := operation(ctx, actorUserID, sessionID, payload)
	if err != nil {
		return models.SaveSessionAsEngramResponse{}, err
	}
	return result, nil
}

func (service fakeChatSessionDerivativeService) ContinueSession(
	ctx context.Context,
	actorUserID uuid.UUID,
	sessionID uuid.UUID,
	payload models.ContinueSessionRequest,
) (models.ContinueSessionResponse, error) {
	if service.continueSession == nil {
		return models.ContinueSessionResponse{}, nil
	}
	return service.continueSession(ctx, actorUserID, sessionID, payload)
}

func staticChatActorResolver(userID uuid.UUID) ChatActorResolver {
	return func(*http.Request) (map[string]any, error) {
		return map[string]any{"user_id": userID.String()}, nil
	}
}

func collectChatRoutes(t *testing.T, router chi.Router) map[string]struct{} {
	t.Helper()
	paths := map[string]struct{}{}
	if err := chi.Walk(
		router,
		func(method string, route string, _ http.Handler, _ ...func(http.Handler) http.Handler) error {
			paths[route] = struct{}{}
			return nil
		},
	); err != nil {
		t.Fatalf("walk routes: %v", err)
	}
	return paths
}

func requireChatRoute(t *testing.T, routes map[string]struct{}, path string) {
	t.Helper()
	if _, ok := routes[path]; !ok {
		t.Fatalf("expected route %q to be registered", path)
	}
}

func decodeChatResponseBody[T any](t *testing.T, body []byte) T {
	t.Helper()
	var value T
	if err := json.Unmarshal(body, &value); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	return value
}

func buildContinueSessionResponse(
	actorID uuid.UUID,
	continuedSessionID uuid.UUID,
	carriedEngramID uuid.UUID,
) models.ContinueSessionResponse {
	timestamp := time.Date(2026, 2, 25, 11, 0, 0, 0, time.UTC)
	return models.ContinueSessionResponse{
		Session: models.ChatSessionRecord{
			SessionID:               continuedSessionID,
			OwnerUserID:             actorID,
			ProjectID:               "proj-1",
			Title:                   "Continued Session",
			Provider:                models.ChatProviderOpenAI,
			ModelID:                 "gpt-4.1-mini",
			SystemPrompt:            "",
			VisibilityScope:         models.VisibilityScopePrivate,
			AutosaveEnabled:         true,
			AutosaveStrategy:        models.ChatAutosaveStrategyInterval,
			AutosaveIntervalMinutes: 10,
			AutosaveMinMessages:     5,
			RetentionDays:           14,
			RetentionMaxSnapshots:   10,
			CreatedAt:               timestamp,
			UpdatedAt:               timestamp,
		},
		CarriedEngramIDs: []uuid.UUID{carriedEngramID},
	}
}

func stringRef(value string) *string {
	return &value
}

func TestActorUserIDResolvesUUIDFromActorPayload(t *testing.T) {
	expectedUserID := uuid.New()
	resolvedUserID, err := actorUserID(
		httptest.NewRequest(http.MethodGet, "/api/v1/chat/sessions", nil),
		func(*http.Request) (map[string]any, error) {
			return map[string]any{"user_id": expectedUserID.String()}, nil
		},
	)
	if err != nil {
		t.Fatalf("actor user id: %v", err)
	}
	if resolvedUserID != expectedUserID {
		t.Fatalf("expected user id %s, got %s", expectedUserID, resolvedUserID)
	}
}

func TestHandleChatServiceErrorReturnsSuccessResult(t *testing.T) {
	result, err := handleChatServiceError(func() (map[string]bool, error) {
		return map[string]bool{"ok": true}, nil
	})
	if err != nil {
		t.Fatalf("handle chat service error: %v", err)
	}
	if !result["ok"] {
		t.Fatalf("expected success result")
	}
}

func TestHandleChatServiceErrorMapsChatServiceException(t *testing.T) {
	_, err := handleChatServiceError(func() (map[string]bool, error) {
		return nil, chat.NewChatServiceError("forbidden", 403)
	})
	if err == nil {
		t.Fatalf("expected mapped chat error")
	}
	httpErr, ok := err.(*chatHTTPError)
	if !ok {
		t.Fatalf("expected chatHTTPError, got %T", err)
	}
	if httpErr.StatusCode() != 403 {
		t.Fatalf("expected status 403, got %d", httpErr.StatusCode())
	}
	if httpErr.Detail() != "forbidden" {
		t.Fatalf("expected detail forbidden, got %q", httpErr.Detail())
	}
}

func TestSSEEventEncodesPayloadLine(t *testing.T) {
	event := sseEvent("chunk", map[string]any{"delta": "assistant", "tokens": 2})
	lines := strings.Split(strings.TrimSpace(event), "\n")
	if len(lines) < 2 {
		t.Fatalf("expected two SSE lines, got %d", len(lines))
	}
	if lines[0] != "event: chunk" {
		t.Fatalf("unexpected event line: %q", lines[0])
	}
	if !strings.HasPrefix(lines[1], "data: ") {
		t.Fatalf("expected data line, got %q", lines[1])
	}
	var payload struct {
		Delta  string `json:"delta"`
		Tokens int    `json:"tokens"`
	}
	if err := json.Unmarshal([]byte(strings.TrimPrefix(lines[1], "data: ")), &payload); err != nil {
		t.Fatalf("decode payload: %v", err)
	}
	if payload.Delta != "assistant" || payload.Tokens != 2 {
		t.Fatalf("unexpected payload: %#v", payload)
	}
}

func TestCreateChatRouterRegistersConfiguredEndpoints(t *testing.T) {
	actorID := uuid.MustParse("00000000-0000-0000-0000-000000000001")
	testCases := []struct {
		name           string
		streamService  ChatStreamService
		sessionService ChatSessionDerivativeService
		expectedRoutes []string
	}{
		{
			name:           "stream route",
			streamService:  fakeChatStreamService{},
			sessionService: nil,
			expectedRoutes: []string{"/api/v1/chat/sessions/{session_id}/messages/stream"},
		},
		{
			name:           "session derivative routes",
			streamService:  nil,
			sessionService: fakeChatSessionDerivativeService{},
			expectedRoutes: []string{
				"/api/v1/chat/sessions/{session_id}/save-engram",
				"/api/v1/chat/sessions/{session_id}/continue",
			},
		},
		{
			name:           "all routes",
			streamService:  fakeChatStreamService{},
			sessionService: fakeChatSessionDerivativeService{},
			expectedRoutes: []string{
				"/api/v1/chat/sessions/{session_id}/messages/stream",
				"/api/v1/chat/sessions/{session_id}/save-engram",
				"/api/v1/chat/sessions/{session_id}/continue",
			},
		},
	}
	for _, testCase := range testCases {
		testCase := testCase
		t.Run(testCase.name, func(t *testing.T) {
			router := CreateChatRouter(
				nil,
				testCase.streamService,
				testCase.sessionService,
				staticChatActorResolver(actorID),
			)
			routes := collectChatRoutes(t, router)
			for _, route := range testCase.expectedRoutes {
				requireChatRoute(t, routes, route)
			}
		})
	}
}

func TestSaveSessionAsEngramHandlerWritesCreatedResponse(t *testing.T) {
	actorID := uuid.MustParse("00000000-0000-0000-0000-000000000010")
	sessionID := uuid.MustParse("00000000-0000-0000-0000-000000000011")
	engramID := uuid.MustParse("00000000-0000-0000-0000-000000000012")
	expectedPayload := models.SaveSessionAsEngramRequest{
		Title:           "Snapshot",
		Abstract:        "Summary",
		VisibilityScope: models.VisibilityScopePrivate,
		Tags:            []string{"tag-1"},
		Keywords:        []string{"kw-1"},
	}
	expected := models.SaveSessionAsEngramResponse{
		EngramID:  engramID,
		SessionID: sessionID,
		CreatedAt: time.Date(2026, 2, 25, 10, 0, 0, 0, time.UTC),
	}
	expectedCall := saveSessionAsEngramCall{
		actorUserID: actorID,
		sessionID:   sessionID,
		payload:     expectedPayload,
	}
	capturedCall := saveSessionAsEngramCall{}
	router := CreateChatRouter(
		nil,
		nil,
		fakeChatSessionDerivativeService{
			saveSessionAsEngram: func(
				_ context.Context,
				receivedActorID uuid.UUID,
				receivedSessionID uuid.UUID,
				payload models.SaveSessionAsEngramRequest,
			) (models.SaveSessionAsEngramResponse, error) {
				capturedCall = saveSessionAsEngramCall{
					actorUserID: receivedActorID,
					sessionID:   receivedSessionID,
					payload:     payload,
				}
				return expected, nil
			},
		},
		staticChatActorResolver(actorID),
	)
	request := httptest.NewRequest(
		http.MethodPost,
		"/api/v1/chat/sessions/"+sessionID.String()+"/save-engram",
		strings.NewReader(`{"title":"Snapshot","abstract":"Summary","visibility_scope":"private","tags":["tag-1"],"keywords":["kw-1"]}`),
	)
	request.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)

	if response.Code != http.StatusCreated {
		t.Fatalf("expected status 201, got %d", response.Code)
	}
	body := decodeChatResponseBody[models.SaveSessionAsEngramResponse](t, response.Body.Bytes())
	if body != expected {
		t.Fatalf("unexpected response body: %#v", body)
	}
	if !reflect.DeepEqual(capturedCall, expectedCall) {
		t.Fatalf("unexpected save-session call: %#v", capturedCall)
	}
}

func TestContinueSessionHandlerWritesCreatedResponse(t *testing.T) {
	actorID := uuid.MustParse("00000000-0000-0000-0000-000000000020")
	sourceSessionID := uuid.MustParse("00000000-0000-0000-0000-000000000021")
	continuedSessionID := uuid.MustParse("00000000-0000-0000-0000-000000000022")
	carriedEngramID := uuid.MustParse("00000000-0000-0000-0000-000000000023")
	expectedTitle := "Carry Forward"
	expected := buildContinueSessionResponse(actorID, continuedSessionID, carriedEngramID)
	expectedCall := continueSessionCall{
		actorUserID: actorID,
		sessionID:   sourceSessionID,
		payload: models.ContinueSessionRequest{
			Title: stringRef(expectedTitle),
		},
	}
	capturedCall := continueSessionCall{}
	router := CreateChatRouter(
		nil,
		nil,
		fakeChatSessionDerivativeService{
			continueSession: func(
				_ context.Context,
				receivedActorID uuid.UUID,
				receivedSessionID uuid.UUID,
				payload models.ContinueSessionRequest,
			) (models.ContinueSessionResponse, error) {
				capturedCall = continueSessionCall{
					actorUserID: receivedActorID,
					sessionID:   receivedSessionID,
					payload:     payload,
				}
				return expected, nil
			},
		},
		staticChatActorResolver(actorID),
	)
	request := httptest.NewRequest(
		http.MethodPost,
		"/api/v1/chat/sessions/"+sourceSessionID.String()+"/continue",
		strings.NewReader(`{"title":"Carry Forward"}`),
	)
	request.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)

	if response.Code != http.StatusCreated {
		t.Fatalf("expected status 201, got %d", response.Code)
	}
	body := decodeChatResponseBody[models.ContinueSessionResponse](t, response.Body.Bytes())
	if body.Session.SessionID != continuedSessionID {
		t.Fatalf("expected session id %s, got %s", continuedSessionID, body.Session.SessionID)
	}
	if len(body.CarriedEngramIDs) != 1 || body.CarriedEngramIDs[0] != carriedEngramID {
		t.Fatalf("unexpected carried engram ids: %#v", body.CarriedEngramIDs)
	}
	if !reflect.DeepEqual(capturedCall, expectedCall) {
		t.Fatalf("unexpected continue-session call: %#v", capturedCall)
	}
}

func TestContinueSessionHandlerMapsChatServiceError(t *testing.T) {
	actorID := uuid.MustParse("00000000-0000-0000-0000-000000000030")
	sessionID := uuid.MustParse("00000000-0000-0000-0000-000000000031")
	router := CreateChatRouter(
		nil,
		nil,
		fakeChatSessionDerivativeService{
			continueSession: func(
				_ context.Context,
				_ uuid.UUID,
				_ uuid.UUID,
				_ models.ContinueSessionRequest,
			) (models.ContinueSessionResponse, error) {
				return models.ContinueSessionResponse{}, chat.NewChatServiceError("forbidden", http.StatusForbidden)
			},
		},
		staticChatActorResolver(actorID),
	)
	request := httptest.NewRequest(
		http.MethodPost,
		"/api/v1/chat/sessions/"+sessionID.String()+"/continue",
		strings.NewReader(`{"title":"Next"}`),
	)
	request.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)
	if response.Code != http.StatusForbidden {
		t.Fatalf("expected status 403, got %d", response.Code)
	}
	body := decodeChatResponseBody[map[string]string](t, response.Body.Bytes())
	if body["detail"] != "forbidden" {
		t.Fatalf("expected forbidden detail, got %q", body["detail"])
	}
}
