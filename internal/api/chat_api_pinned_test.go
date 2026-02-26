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

func pinnedEngramFixture(
	sessionID uuid.UUID,
	engramID uuid.UUID,
	actorID uuid.UUID,
) models.PinnedEngramRecord {
	return models.PinnedEngramRecord{
		SessionID:      sessionID,
		EngramID:       engramID,
		PinnedByUserID: actorID,
		CreatedAt:      time.Date(2026, 2, 26, 11, 0, 0, 0, time.UTC),
	}
}

func pinnedDocumentFixture(
	sessionID uuid.UUID,
	documentID uuid.UUID,
	actorID uuid.UUID,
) models.PinnedDocumentRecord {
	return models.PinnedDocumentRecord{
		SessionID:      sessionID,
		DocumentID:     documentID,
		PinnedByUserID: actorID,
		CreatedAt:      time.Date(2026, 2, 26, 11, 0, 0, 0, time.UTC),
	}
}

func TestCreateChatRouterRegistersPinnedRoutesWhenSessionServiceConfigured(t *testing.T) {
	router := CreateChatRouter(
		newFakeChatSessionService(),
		nil,
		nil,
		staticChatActorResolver(uuid.MustParse("00000000-0000-0000-0000-000000000080")),
	)
	routes := collectChatRoutes(t, router)
	requireChatRoute(t, routes, "/api/v1/chat/sessions/{session_id}/engrams")
	requireChatRoute(t, routes, "/api/v1/chat/sessions/{session_id}/engrams/pin")
	requireChatRoute(t, routes, "/api/v1/chat/sessions/{session_id}/engrams/{engram_id}")
	requireChatRoute(t, routes, "/api/v1/chat/sessions/{session_id}/documents")
	requireChatRoute(t, routes, "/api/v1/chat/sessions/{session_id}/documents/pin")
	requireChatRoute(t, routes, "/api/v1/chat/sessions/{session_id}/documents/{document_id}")
}

func TestPinEngramHandlerWritesPinnedRecord(t *testing.T) {
	actorID := uuid.MustParse("00000000-0000-0000-0000-000000000081")
	sessionID := uuid.MustParse("00000000-0000-0000-0000-000000000082")
	engramID := uuid.MustParse("00000000-0000-0000-0000-000000000083")
	expected := pinnedEngramFixture(sessionID, engramID, actorID)
	captured := chat.SessionPinEngramRequest{}
	service := newFakeChatSessionService()
	service.pinEngramFn = func(
		_ context.Context,
		request chat.SessionPinEngramRequest,
	) (*models.PinnedEngramRecord, error) {
		captured = request
		return &expected, nil
	}
	router := CreateChatRouter(service, nil, nil, staticChatActorResolver(actorID))
	request := httptest.NewRequest(
		http.MethodPost,
		"/api/v1/chat/sessions/"+sessionID.String()+"/engrams/pin",
		strings.NewReader(`{"engram_id":"`+engramID.String()+`"}`),
	)
	request.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", response.Code)
	}
	pinned := decodeChatResponseBody[models.PinnedEngramRecord](t, response.Body.Bytes())
	if pinned.EngramID != engramID {
		t.Fatalf("unexpected engram id in response: %s", pinned.EngramID)
	}
	if pinned.SessionID != sessionID {
		t.Fatalf("unexpected session id in response: %s", pinned.SessionID)
	}
	if captured.ActorUserID != actorID {
		t.Fatalf("unexpected actor id in request: %s", captured.ActorUserID)
	}
	if captured.SessionID != sessionID {
		t.Fatalf("unexpected session id in request: %s", captured.SessionID)
	}
	if captured.EngramID != engramID {
		t.Fatalf("unexpected engram id in request: %s", captured.EngramID)
	}
}

func TestListPinnedDocumentsHandlerWritesRecords(t *testing.T) {
	actorID := uuid.MustParse("00000000-0000-0000-0000-000000000084")
	sessionID := uuid.MustParse("00000000-0000-0000-0000-000000000085")
	documentID := uuid.MustParse("00000000-0000-0000-0000-000000000086")
	service := newFakeChatSessionService()
	service.listPinnedDocumentsFn = func(
		_ context.Context,
		receivedActorID uuid.UUID,
		receivedSessionID uuid.UUID,
	) ([]models.PinnedDocumentRecord, error) {
		if receivedActorID != actorID {
			t.Fatalf("unexpected actor id for list pinned docs: %s", receivedActorID)
		}
		if receivedSessionID != sessionID {
			t.Fatalf("unexpected session id for list pinned docs: %s", receivedSessionID)
		}
		return []models.PinnedDocumentRecord{
			pinnedDocumentFixture(sessionID, documentID, actorID),
		}, nil
	}
	router := CreateChatRouter(service, nil, nil, staticChatActorResolver(actorID))
	request := httptest.NewRequest(
		http.MethodGet,
		"/api/v1/chat/sessions/"+sessionID.String()+"/documents",
		nil,
	)
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", response.Code)
	}
	documents := decodeChatResponseBody[[]models.PinnedDocumentRecord](t, response.Body.Bytes())
	if len(documents) != 1 || documents[0].DocumentID != documentID {
		t.Fatalf("unexpected pinned documents response: %#v", documents)
	}
}

func TestUnpinDocumentHandlerReturnsNoContent(t *testing.T) {
	actorID := uuid.MustParse("00000000-0000-0000-0000-000000000087")
	sessionID := uuid.MustParse("00000000-0000-0000-0000-000000000088")
	documentID := uuid.MustParse("00000000-0000-0000-0000-000000000089")
	captured := chat.SessionPinDocumentRequest{}
	service := newFakeChatSessionService()
	service.unpinDocumentFn = func(
		_ context.Context,
		request chat.SessionPinDocumentRequest,
	) error {
		captured = request
		return nil
	}
	router := CreateChatRouter(service, nil, nil, staticChatActorResolver(actorID))
	request := httptest.NewRequest(
		http.MethodDelete,
		"/api/v1/chat/sessions/"+sessionID.String()+"/documents/"+documentID.String(),
		nil,
	)
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)
	if response.Code != http.StatusNoContent {
		t.Fatalf("expected status 204, got %d", response.Code)
	}
	if captured.ActorUserID != actorID {
		t.Fatalf("unexpected actor id in unpin request: %s", captured.ActorUserID)
	}
	if captured.SessionID != sessionID {
		t.Fatalf("unexpected session id in unpin request: %s", captured.SessionID)
	}
	if captured.DocumentID != documentID {
		t.Fatalf("unexpected document id in unpin request: %s", captured.DocumentID)
	}
}
