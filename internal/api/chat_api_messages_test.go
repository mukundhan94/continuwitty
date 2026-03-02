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

func chatMessageFixture(
	messageID uuid.UUID,
	sessionID uuid.UUID,
	contentText string,
) models.ChatMessageRecord {
	return models.ChatMessageRecord{
		MessageID:      messageID,
		SessionID:      sessionID,
		Role:           "user",
		ContentText:    contentText,
		TokenUsageJSON: map[string]any{},
		UsedEngramIDs:  []uuid.UUID{},
		CreatedAt:      time.Date(2026, 2, 26, 10, 0, 0, 0, time.UTC),
	}
}

func TestCreateChatRouterRegistersMessageRoutesWhenServicesConfigured(t *testing.T) {
	router := CreateChatRouter(
		newFakeChatSessionService(),
		fakeChatStreamService{},
		nil,
		staticChatActorResolver(uuid.MustParse("00000000-0000-0000-0000-000000000070")),
	)
	routes := collectChatRoutes(t, router)
	requireChatRoute(t, routes, "/api/v1/chat/sessions/{session_id}/messages")
	requireChatRoute(t, routes, "/api/v1/chat/sessions/{session_id}/messages/stream")
}

func TestSendMessageHandlerWritesCreatedResponse(t *testing.T) {
	actorID := uuid.MustParse("00000000-0000-0000-0000-000000000071")
	sessionID := uuid.MustParse("00000000-0000-0000-0000-000000000072")
	messageID := uuid.MustParse("00000000-0000-0000-0000-000000000073")
	replyMessageID := uuid.MustParse("00000000-0000-0000-0000-000000000074")
	expected := chat.ChatSendResponse{
		SessionID:            sessionID,
		MessageID:            messageID,
		ReplyMessageID:       replyMessageID,
		AssistantText:        "Acknowledged",
		UsedEngramIDs:        []uuid.UUID{},
		UsedDocumentChunkIDs: []uuid.UUID{},
		SourceReferences:     []chat.ChatSourceReference{},
		DebugTrace:           nil,
	}
	capturedText := ""
	streamService := fakeChatStreamService{
		sendMessage: func(
			_ context.Context,
			receivedActorID uuid.UUID,
			receivedSessionID uuid.UUID,
			payload chat.ChatMessageCreateRequest,
		) (chat.ChatSendResponse, error) {
			assertSendIdentifiers(t, sendIdentifiersAssertion{
				receivedActorID:   receivedActorID,
				expectedActorID:   actorID,
				receivedSessionID: receivedSessionID,
				expectedSessionID: sessionID,
			})
			capturedText = payload.ContentText
			return expected, nil
		},
	}
	router := CreateChatRouter(nil, streamService, nil, staticChatActorResolver(actorID))
	request := httptest.NewRequest(
		http.MethodPost,
		"/api/v1/chat/sessions/"+sessionID.String()+"/messages",
		strings.NewReader(`{"content_text":"Hello world"}`),
	)
	request.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)
	if response.Code != http.StatusCreated {
		t.Fatalf("expected status 201, got %d", response.Code)
	}
	sent := decodeChatResponseBody[chat.ChatSendResponse](t, response.Body.Bytes())
	assertSendResponseIDs(t, sendResponseIDsAssertion{
		response:               sent,
		expectedSessionID:      sessionID,
		expectedMessageID:      messageID,
		expectedReplyMessageID: replyMessageID,
	})
	if capturedText != "Hello world" {
		t.Fatalf("expected content_text to be forwarded, got %q", capturedText)
	}
}

type sendIdentifiersAssertion struct {
	receivedActorID   uuid.UUID
	expectedActorID   uuid.UUID
	receivedSessionID uuid.UUID
	expectedSessionID uuid.UUID
}

func assertSendIdentifiers(t *testing.T, assertion sendIdentifiersAssertion) {
	t.Helper()
	if assertion.receivedActorID != assertion.expectedActorID {
		t.Fatalf("unexpected actor id: got=%s expected=%s", assertion.receivedActorID, assertion.expectedActorID)
	}
	if assertion.receivedSessionID != assertion.expectedSessionID {
		t.Fatalf("unexpected session id: got=%s expected=%s", assertion.receivedSessionID, assertion.expectedSessionID)
	}
}

type sendResponseIDsAssertion struct {
	response               chat.ChatSendResponse
	expectedSessionID      uuid.UUID
	expectedMessageID      uuid.UUID
	expectedReplyMessageID uuid.UUID
}

func assertSendResponseIDs(t *testing.T, assertion sendResponseIDsAssertion) {
	t.Helper()
	if assertion.response.SessionID != assertion.expectedSessionID {
		t.Fatalf("unexpected session id in send response: %#v", assertion.response)
	}
	if assertion.response.MessageID != assertion.expectedMessageID {
		t.Fatalf("unexpected message id in send response: %#v", assertion.response)
	}
	if assertion.response.ReplyMessageID != assertion.expectedReplyMessageID {
		t.Fatalf("unexpected reply message id in send response: %#v", assertion.response)
	}
}

func TestListMessagesHandlerUsesDefaultPaging(t *testing.T) {
	actorID := uuid.MustParse("00000000-0000-0000-0000-000000000075")
	sessionID := uuid.MustParse("00000000-0000-0000-0000-000000000076")
	messageID := uuid.MustParse("00000000-0000-0000-0000-000000000077")
	capturedRequest := chat.SessionMessagesRequest{}
	service := newFakeChatSessionService()
	service.listMessagesFn = func(
		_ context.Context,
		request chat.SessionMessagesRequest,
	) ([]models.ChatMessageRecord, error) {
		capturedRequest = request
		return []models.ChatMessageRecord{
			chatMessageFixture(messageID, sessionID, "Hello"),
		}, nil
	}
	router := CreateChatRouter(service, nil, nil, staticChatActorResolver(actorID))
	request := httptest.NewRequest(
		http.MethodGet,
		"/api/v1/chat/sessions/"+sessionID.String()+"/messages",
		nil,
	)
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", response.Code)
	}
	messages := decodeChatResponseBody[[]models.ChatMessageRecord](t, response.Body.Bytes())
	if len(messages) != 1 || messages[0].MessageID != messageID {
		t.Fatalf("unexpected messages response: %#v", messages)
	}
	if capturedRequest.ActorUserID != actorID || capturedRequest.SessionID != sessionID {
		t.Fatalf("unexpected message request identifiers: %#v", capturedRequest)
	}
	if capturedRequest.Limit != 200 || capturedRequest.Offset != 0 {
		t.Fatalf("unexpected message request paging: %#v", capturedRequest)
	}
}

func TestSendMessageHandlerForwardsLinkRecallOverrides(t *testing.T) {
	actorID := uuid.MustParse("00000000-0000-0000-0000-000000000170")
	sessionID := uuid.MustParse("00000000-0000-0000-0000-000000000171")
	capturedPayload := chat.ChatMessageCreateRequest{}
	streamService := fakeChatStreamService{
		sendMessage: func(
			_ context.Context,
			_ uuid.UUID,
			_ uuid.UUID,
			payload chat.ChatMessageCreateRequest,
		) (chat.ChatSendResponse, error) {
			capturedPayload = payload
			return chat.ChatSendResponse{
				SessionID:            sessionID,
				MessageID:            uuid.MustParse("00000000-0000-0000-0000-000000000172"),
				ReplyMessageID:       uuid.MustParse("00000000-0000-0000-0000-000000000173"),
				AssistantText:        "ok",
				UsedEngramIDs:        []uuid.UUID{},
				UsedEngramLinkIDs:    []uuid.UUID{},
				EngramTracePaths:     []chat.EngramTracePath{},
				UsedDocumentChunkIDs: []uuid.UUID{},
				SourceReferences:     []chat.ChatSourceReference{},
			}, nil
		},
	}
	router := CreateChatRouter(nil, streamService, nil, staticChatActorResolver(actorID))
	request := httptest.NewRequest(
		http.MethodPost,
		"/api/v1/chat/sessions/"+sessionID.String()+"/messages",
		strings.NewReader(
			`{"content_text":"Hello world","link_recall_enabled":false,"link_recall_depth":2,"link_recall_max_neighbors":9,"link_noise_suppression_enabled":false,"link_noise_score_threshold":0.66}`,
		),
	)
	request.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)

	if response.Code != http.StatusCreated {
		t.Fatalf("expected status 201, got %d", response.Code)
	}
	assertForwardedChatLinkControls(t, capturedPayload)
}

func TestListMessagesHandlerRejectsInvalidLimit(t *testing.T) {
	actorID := uuid.MustParse("00000000-0000-0000-0000-000000000078")
	sessionID := uuid.MustParse("00000000-0000-0000-0000-000000000079")
	router := CreateChatRouter(
		newFakeChatSessionService(),
		nil,
		nil,
		staticChatActorResolver(actorID),
	)
	request := httptest.NewRequest(
		http.MethodGet,
		"/api/v1/chat/sessions/"+sessionID.String()+"/messages?limit=0",
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

func assertForwardedChatLinkControls(t *testing.T, payload chat.ChatMessageCreateRequest) {
	t.Helper()
	requireBoolPointerChat(t, payload.LinkRecallEnabled, false, "link_recall_enabled")
	requireIntPointerChat(t, payload.LinkRecallDepth, 2, "link_recall_depth")
	requireIntPointerChat(t, payload.LinkRecallMaxNeighbors, 9, "link_recall_max_neighbors")
	requireBoolPointerChat(t, payload.LinkNoiseSuppressionEnabled, false, "link_noise_suppression_enabled")
	requireFloatPointerChat(t, payload.LinkNoiseScoreThreshold, 0.66, "link_noise_score_threshold")
}

func requireBoolPointerChat(t *testing.T, value *bool, expected bool, field string) {
	t.Helper()
	if value == nil || *value != expected {
		t.Fatalf("expected %s=%v to be forwarded", field, expected)
	}
}

func requireIntPointerChat(t *testing.T, value *int, expected int, field string) {
	t.Helper()
	if value == nil || *value != expected {
		t.Fatalf("expected %s=%d to be forwarded", field, expected)
	}
}

func requireFloatPointerChat(t *testing.T, value *float64, expected float64, field string) {
	t.Helper()
	if value == nil || *value != expected {
		t.Fatalf("expected %s=%v to be forwarded", field, expected)
	}
}
