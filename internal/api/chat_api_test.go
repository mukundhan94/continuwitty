package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"engram/internal/chat"

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

func TestCreateChatRouterRegistersStreamEndpoint(t *testing.T) {
	router := CreateChatRouter(
		fakeChatStreamService{},
		func(*http.Request) (map[string]any, error) {
			return map[string]any{"user_id": uuid.MustParse("00000000-0000-0000-0000-000000000001").String()}, nil
		},
	)
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
	if _, ok := paths["/api/v1/chat/sessions/{session_id}/messages/stream"]; !ok {
		t.Fatalf("expected stream route to be registered")
	}
}
