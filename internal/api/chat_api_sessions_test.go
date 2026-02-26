package api

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"engram/internal/chat"
	"engram/internal/models"

	"github.com/google/uuid"
)

type fakeChatSessionService struct {
	createSessionFn func(
		ctx context.Context,
		actorUserID uuid.UUID,
		payload models.ChatSessionCreateRequest,
	) (*models.ChatSessionRecord, error)
	listSessionsFn func(
		ctx context.Context,
		request chat.SessionListRequest,
	) ([]models.ChatSessionRecord, error)
	getSessionFn func(
		ctx context.Context,
		actorUserID uuid.UUID,
		sessionID uuid.UUID,
	) (*models.ChatSessionRecord, error)
	updateSessionFn func(
		ctx context.Context,
		actorUserID uuid.UUID,
		sessionID uuid.UUID,
		payload models.ChatSessionUpdateRequest,
	) (*models.ChatSessionRecord, error)
}

func newFakeChatSessionService() fakeChatSessionService {
	return fakeChatSessionService{
		createSessionFn: func(
			context.Context,
			uuid.UUID,
			models.ChatSessionCreateRequest,
		) (*models.ChatSessionRecord, error) {
			return nil, nil
		},
		listSessionsFn: func(
			context.Context,
			chat.SessionListRequest,
		) ([]models.ChatSessionRecord, error) {
			return []models.ChatSessionRecord{}, nil
		},
		getSessionFn: func(
			context.Context,
			uuid.UUID,
			uuid.UUID,
		) (*models.ChatSessionRecord, error) {
			return nil, nil
		},
		updateSessionFn: func(
			context.Context,
			uuid.UUID,
			uuid.UUID,
			models.ChatSessionUpdateRequest,
		) (*models.ChatSessionRecord, error) {
			return nil, nil
		},
	}
}

func (service fakeChatSessionService) CreateSession(
	ctx context.Context,
	actorUserID uuid.UUID,
	payload models.ChatSessionCreateRequest,
) (*models.ChatSessionRecord, error) {
	return service.createSessionFn(ctx, actorUserID, payload)
}

func (service fakeChatSessionService) ListSessions(
	ctx context.Context,
	request chat.SessionListRequest,
) ([]models.ChatSessionRecord, error) {
	return service.listSessionsFn(ctx, request)
}

func (service fakeChatSessionService) GetSession(
	ctx context.Context,
	actorUserID uuid.UUID,
	sessionID uuid.UUID,
) (*models.ChatSessionRecord, error) {
	return service.getSessionFn(ctx, actorUserID, sessionID)
}

func (service fakeChatSessionService) UpdateSession(
	ctx context.Context,
	actorUserID uuid.UUID,
	sessionID uuid.UUID,
	payload models.ChatSessionUpdateRequest,
) (*models.ChatSessionRecord, error) {
	return service.updateSessionFn(ctx, actorUserID, sessionID, payload)
}

func chatSessionFixture(
	sessionID uuid.UUID,
	actorID uuid.UUID,
	title string,
) models.ChatSessionRecord {
	return models.ChatSessionRecord{
		SessionID:               sessionID,
		OwnerUserID:             actorID,
		ProjectID:               "proj-1",
		Title:                   title,
		Provider:                models.ChatProviderOpenAI,
		ModelID:                 "gpt-4.1-mini",
		SystemPrompt:            "You are an assistant.",
		VisibilityScope:         models.VisibilityScopePrivate,
		AutosaveEnabled:         true,
		AutosaveStrategy:        models.ChatAutosaveStrategyInterval,
		AutosaveIntervalMinutes: 10,
		AutosaveMinMessages:     5,
		RetentionDays:           14,
		RetentionMaxSnapshots:   10,
		CreatedAt:               time.Date(2026, 2, 26, 8, 0, 0, 0, time.UTC),
		UpdatedAt:               time.Date(2026, 2, 26, 8, 0, 0, 0, time.UTC),
	}
}

func assertSingleListedSession(t *testing.T, listed []models.ChatSessionRecord, sessionID uuid.UUID) {
	t.Helper()
	if len(listed) != 1 {
		t.Fatalf("expected one listed session, got %d", len(listed))
	}
	if listed[0].SessionID != sessionID {
		t.Fatalf("unexpected listed sessions: %#v", listed)
	}
}

func assertDefaultListRequest(
	t *testing.T,
	request chat.SessionListRequest,
	actorID uuid.UUID,
	projectID string,
) {
	t.Helper()
	if request.ActorUserID != actorID {
		t.Fatalf("unexpected actor id: %s", request.ActorUserID)
	}
	if request.Limit != 50 || request.Offset != 0 {
		t.Fatalf("unexpected paging request: %#v", request)
	}
	if request.ProjectID == nil || *request.ProjectID != projectID {
		t.Fatalf("expected project filter in list request: %#v", request.ProjectID)
	}
}

type updateSessionCall struct {
	actorUserID uuid.UUID
	sessionID   uuid.UUID
	title       string
}

func TestCreateChatRouterRegistersSessionRoutesWhenSessionServiceConfigured(t *testing.T) {
	router := CreateChatRouter(
		newFakeChatSessionService(),
		nil,
		nil,
		staticChatActorResolver(uuid.MustParse("00000000-0000-0000-0000-000000000041")),
	)
	routes := collectChatRoutes(t, router)
	requireChatRoute(t, routes, "/api/v1/chat/sessions")
	requireChatRoute(t, routes, "/api/v1/chat/sessions/{session_id}")
}

func TestCreateSessionHandlerWritesCreatedResponse(t *testing.T) {
	actorID := uuid.MustParse("00000000-0000-0000-0000-000000000042")
	sessionID := uuid.MustParse("00000000-0000-0000-0000-000000000043")
	expected := chatSessionFixture(sessionID, actorID, "Design Notes")
	capturedActorID := uuid.Nil
	capturedPayload := models.ChatSessionCreateRequest{}
	service := newFakeChatSessionService()
	service.createSessionFn = func(
		_ context.Context,
		receivedActorID uuid.UUID,
		payload models.ChatSessionCreateRequest,
	) (*models.ChatSessionRecord, error) {
		capturedActorID = receivedActorID
		capturedPayload = payload
		return &expected, nil
	}
	router := CreateChatRouter(
		service,
		nil,
		nil,
		staticChatActorResolver(actorID),
	)
	request := httptest.NewRequest(
		http.MethodPost,
		"/api/v1/chat/sessions",
		strings.NewReader(`{"project_id":"proj-1","title":"Design Notes","provider":"openai","model_id":"gpt-4.1-mini","system_prompt":"You are an assistant.","visibility_scope":"private","autosave_enabled":true,"autosave_strategy":"interval","autosave_interval_minutes":10,"autosave_min_messages":5,"retention_days":14,"retention_max_snapshots":10}`),
	)
	request.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)
	if response.Code != http.StatusCreated {
		t.Fatalf("expected status 201, got %d", response.Code)
	}
	created := decodeChatResponseBody[models.ChatSessionRecord](t, response.Body.Bytes())
	if created.SessionID != sessionID {
		t.Fatalf("unexpected created session id: %s", created.SessionID)
	}
	if capturedActorID != actorID {
		t.Fatalf("unexpected actor id passed to service: %s", capturedActorID)
	}
	if capturedPayload.ProjectID != "proj-1" || capturedPayload.Title != "Design Notes" {
		t.Fatalf("unexpected create payload: %#v", capturedPayload)
	}
}

func TestListSessionsHandlerUsesDefaultPaging(t *testing.T) {
	actorID := uuid.MustParse("00000000-0000-0000-0000-000000000044")
	sessionID := uuid.MustParse("00000000-0000-0000-0000-000000000045")
	capturedRequest := chat.SessionListRequest{}
	service := newFakeChatSessionService()
	service.listSessionsFn = func(
		_ context.Context,
		request chat.SessionListRequest,
	) ([]models.ChatSessionRecord, error) {
		capturedRequest = request
		return []models.ChatSessionRecord{chatSessionFixture(sessionID, actorID, "Session")}, nil
	}
	router := CreateChatRouter(
		service,
		nil,
		nil,
		staticChatActorResolver(actorID),
	)
	request := httptest.NewRequest(http.MethodGet, "/api/v1/chat/sessions?project_id=proj-1", nil)
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", response.Code)
	}
	listed := decodeChatResponseBody[[]models.ChatSessionRecord](t, response.Body.Bytes())
	assertSingleListedSession(t, listed, sessionID)
	assertDefaultListRequest(t, capturedRequest, actorID, "proj-1")
}

func TestListSessionsHandlerRejectsInvalidLimit(t *testing.T) {
	actorID := uuid.MustParse("00000000-0000-0000-0000-000000000046")
	router := CreateChatRouter(
		newFakeChatSessionService(),
		nil,
		nil,
		staticChatActorResolver(actorID),
	)
	request := httptest.NewRequest(http.MethodGet, "/api/v1/chat/sessions?limit=0", nil)
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)
	if response.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", response.Code)
	}
	body := decodeChatResponseBody[map[string]string](t, response.Body.Bytes())
	if body["detail"] != "Invalid query parameters" {
		t.Fatalf("unexpected detail: %q", body["detail"])
	}
}

func TestGetSessionHandlerMapsChatServiceError(t *testing.T) {
	actorID := uuid.MustParse("00000000-0000-0000-0000-000000000047")
	sessionID := uuid.MustParse("00000000-0000-0000-0000-000000000048")
	service := newFakeChatSessionService()
	service.getSessionFn = func(
		_ context.Context,
		_ uuid.UUID,
		_ uuid.UUID,
	) (*models.ChatSessionRecord, error) {
		return nil, chat.NewChatServiceError("not found", http.StatusNotFound)
	}
	router := CreateChatRouter(
		service,
		nil,
		nil,
		staticChatActorResolver(actorID),
	)
	request := httptest.NewRequest(
		http.MethodGet,
		"/api/v1/chat/sessions/"+sessionID.String(),
		nil,
	)
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)
	if response.Code != http.StatusNotFound {
		t.Fatalf("expected status 404, got %d", response.Code)
	}
	body := decodeChatResponseBody[map[string]string](t, response.Body.Bytes())
	if body["detail"] != "not found" {
		t.Fatalf("unexpected detail: %q", body["detail"])
	}
}

func TestUpdateSessionHandlerWritesUpdatedResponse(t *testing.T) {
	actorID := uuid.MustParse("00000000-0000-0000-0000-000000000049")
	sessionID := uuid.MustParse("00000000-0000-0000-0000-000000000050")
	updatedTitle := "Updated Session"
	expected := chatSessionFixture(sessionID, actorID, updatedTitle)
	expectedCall := updateSessionCall{
		actorUserID: actorID,
		sessionID:   sessionID,
		title:       updatedTitle,
	}
	capturedCall := updateSessionCall{}
	service := newFakeChatSessionService()
	service.updateSessionFn = func(
		_ context.Context,
		receivedActorID uuid.UUID,
		receivedSessionID uuid.UUID,
		payload models.ChatSessionUpdateRequest,
	) (*models.ChatSessionRecord, error) {
		capturedTitle := ""
		if payload.Title != nil {
			capturedTitle = *payload.Title
		}
		capturedCall = updateSessionCall{
			actorUserID: receivedActorID,
			sessionID:   receivedSessionID,
			title:       capturedTitle,
		}
		return &expected, nil
	}
	router := CreateChatRouter(
		service,
		nil,
		nil,
		staticChatActorResolver(actorID),
	)
	request := httptest.NewRequest(
		http.MethodPatch,
		"/api/v1/chat/sessions/"+sessionID.String(),
		strings.NewReader(`{"title":"Updated Session"}`),
	)
	request.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", response.Code)
	}
	updated := decodeChatResponseBody[models.ChatSessionRecord](t, response.Body.Bytes())
	if updated.SessionID != sessionID || updated.Title != updatedTitle {
		t.Fatalf("unexpected updated session response: %#v", updated)
	}
	if capturedCall != expectedCall {
		t.Fatalf("unexpected update call: %#v", capturedCall)
	}
}
