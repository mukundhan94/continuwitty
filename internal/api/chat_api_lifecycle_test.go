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

type lifecycleSessionCall struct {
	actorUserID uuid.UUID
	sessionID   uuid.UUID
}

func timelineEventFixture(sessionID uuid.UUID) models.ChatTimelineEvent {
	return models.ChatTimelineEvent{
		EventID:               uuid.MustParse("00000000-0000-0000-0000-000000000061"),
		SessionID:             sessionID,
		EventType:             "engram_saved",
		Title:                 "Saved as engram",
		Abstract:              "Captured conversation snapshot",
		Tags:                  []string{"memory"},
		ConsolidationGroupKey: nil,
		CreatedAt:             time.Date(2026, 2, 26, 9, 0, 0, 0, time.UTC),
	}
}

func TestCreateChatRouterRegistersLifecycleRoutesWhenSessionServiceConfigured(t *testing.T) {
	router := CreateChatRouter(
		newFakeChatSessionService(),
		nil,
		nil,
		staticChatActorResolver(uuid.MustParse("00000000-0000-0000-0000-000000000060")),
	)
	routes := collectChatRoutes(t, router)
	requireChatRoute(t, routes, "/api/v1/chat/sessions/{session_id}/lifecycle-policy")
	requireChatRoute(t, routes, "/api/v1/chat/sessions/{session_id}/timeline")
}

func TestGetLifecyclePolicyHandlerWritesResponse(t *testing.T) {
	actorID := uuid.MustParse("00000000-0000-0000-0000-000000000062")
	sessionID := uuid.MustParse("00000000-0000-0000-0000-000000000063")
	expected := chat.ChatLifecyclePolicy{
		AutosaveEnabled:         true,
		AutosaveStrategy:        models.ChatAutosaveStrategyInterval,
		AutosaveIntervalMinutes: 15,
		AutosaveMinMessages:     5,
		RetentionDays:           14,
		RetentionMaxSnapshots:   12,
	}
	capturedCall := lifecycleSessionCall{}
	service := newFakeChatSessionService()
	service.getLifecyclePolicyFn = func(
		_ context.Context,
		receivedActorID uuid.UUID,
		receivedSessionID uuid.UUID,
	) (chat.ChatLifecyclePolicy, error) {
		capturedCall = lifecycleSessionCall{
			actorUserID: receivedActorID,
			sessionID:   receivedSessionID,
		}
		return expected, nil
	}
	router := CreateChatRouter(service, nil, nil, staticChatActorResolver(actorID))
	request := httptest.NewRequest(
		http.MethodGet,
		"/api/v1/chat/sessions/"+sessionID.String()+"/lifecycle-policy",
		nil,
	)
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", response.Code)
	}
	policy := decodeChatResponseBody[chat.ChatLifecyclePolicy](t, response.Body.Bytes())
	if policy != expected {
		t.Fatalf("unexpected lifecycle policy: %#v", policy)
	}
	if capturedCall.actorUserID != actorID || capturedCall.sessionID != sessionID {
		t.Fatalf("unexpected lifecycle call: %#v", capturedCall)
	}
}

func TestUpdateLifecyclePolicyHandlerWritesResponse(t *testing.T) {
	actorID := uuid.MustParse("00000000-0000-0000-0000-000000000064")
	sessionID := uuid.MustParse("00000000-0000-0000-0000-000000000065")
	expected := chat.ChatLifecyclePolicy{
		AutosaveEnabled:         false,
		AutosaveStrategy:        models.ChatAutosaveStrategyOff,
		AutosaveIntervalMinutes: 10,
		AutosaveMinMessages:     5,
		RetentionDays:           30,
		RetentionMaxSnapshots:   15,
	}
	capturedDays := 0
	service := newFakeChatSessionService()
	service.updateLifecyclePolicyFn = func(
		_ context.Context,
		_ uuid.UUID,
		_ uuid.UUID,
		payload chat.ChatLifecyclePolicyUpdateRequest,
	) (chat.ChatLifecyclePolicy, error) {
		if payload.RetentionDays != nil {
			capturedDays = *payload.RetentionDays
		}
		return expected, nil
	}
	router := CreateChatRouter(service, nil, nil, staticChatActorResolver(actorID))
	request := httptest.NewRequest(
		http.MethodPatch,
		"/api/v1/chat/sessions/"+sessionID.String()+"/lifecycle-policy",
		strings.NewReader(`{"retention_days":30}`),
	)
	request.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", response.Code)
	}
	policy := decodeChatResponseBody[chat.ChatLifecyclePolicy](t, response.Body.Bytes())
	if policy != expected {
		t.Fatalf("unexpected lifecycle policy response: %#v", policy)
	}
	if capturedDays != 30 {
		t.Fatalf("expected retention days payload to be forwarded, got %d", capturedDays)
	}
}

func TestListTimelineEventsHandlerUsesDefaultPaging(t *testing.T) {
	actorID := uuid.MustParse("00000000-0000-0000-0000-000000000066")
	sessionID := uuid.MustParse("00000000-0000-0000-0000-000000000067")
	capturedRequest := chat.SessionTimelineRequest{}
	service := newFakeChatSessionService()
	service.listTimelineEventsFn = func(
		_ context.Context,
		request chat.SessionTimelineRequest,
	) ([]models.ChatTimelineEvent, error) {
		capturedRequest = request
		return []models.ChatTimelineEvent{timelineEventFixture(sessionID)}, nil
	}
	router := CreateChatRouter(service, nil, nil, staticChatActorResolver(actorID))
	request := httptest.NewRequest(
		http.MethodGet,
		"/api/v1/chat/sessions/"+sessionID.String()+"/timeline",
		nil,
	)
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", response.Code)
	}
	events := decodeChatResponseBody[[]models.ChatTimelineEvent](t, response.Body.Bytes())
	if len(events) != 1 || events[0].SessionID != sessionID {
		t.Fatalf("unexpected timeline events: %#v", events)
	}
	if capturedRequest.ActorUserID != actorID || capturedRequest.SessionID != sessionID {
		t.Fatalf("unexpected timeline identifiers: %#v", capturedRequest)
	}
	if capturedRequest.Limit != 100 || capturedRequest.Offset != 0 {
		t.Fatalf("unexpected timeline paging: %#v", capturedRequest)
	}
}

func TestListTimelineEventsHandlerRejectsInvalidLimit(t *testing.T) {
	actorID := uuid.MustParse("00000000-0000-0000-0000-000000000068")
	sessionID := uuid.MustParse("00000000-0000-0000-0000-000000000069")
	router := CreateChatRouter(
		newFakeChatSessionService(),
		nil,
		nil,
		staticChatActorResolver(actorID),
	)
	request := httptest.NewRequest(
		http.MethodGet,
		"/api/v1/chat/sessions/"+sessionID.String()+"/timeline?limit=0",
		nil,
	)
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
