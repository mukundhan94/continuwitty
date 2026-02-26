package mcp

import (
	"context"
	"errors"
	"reflect"
	"testing"

	"engram/internal/models"

	"github.com/google/uuid"
)

func TestCompatibilityServiceChatListMessagesParity(t *testing.T) {
	actorUserID := uuid.MustParse("29000000-0000-0000-0000-000000000290")
	sessionID := uuid.MustParse("29000000-0000-0000-0000-000000000291")
	sessionGetService := &fakeSessionGetService{
		session: &models.ChatSessionRecord{SessionID: sessionID, ProjectID: "proj-alpha"},
	}
	messageService := &fakeMessageListService{
		messages: []models.ChatMessageRecord{
			{
				MessageID:   uuid.MustParse("29000000-0000-0000-0000-000000000292"),
				SessionID:   sessionID,
				Role:        "user",
				ContentText: "hello",
			},
			{
				MessageID:   uuid.MustParse("29000000-0000-0000-0000-000000000293"),
				SessionID:   sessionID,
				Role:        "assistant",
				ContentText: "world",
			},
		},
	}

	testCases := []struct {
		name            string
		request         StreamCallRequest
		asToolsCallPath bool
	}{
		{
			name: "direct",
			request: directToolRequest(
				actorUserID.String(),
				"chat.list_messages",
				map[string]any{
					"session_id": sessionID.String(),
					"limit":      12.0,
					"offset":     3.0,
				},
			),
			asToolsCallPath: false,
		},
		{
			name: "tools call",
			request: toolsCallRequest(
				actorUserID.String(),
				"chat_list_messages",
				map[string]any{
					"session_id": sessionID.String(),
					"limit":      12.0,
					"offset":     3.0,
				},
			),
			asToolsCallPath: true,
		},
	}

	for _, testCase := range testCases {
		testCase := testCase
		t.Run(testCase.name, func(t *testing.T) {
			frame := runCompatibilityRequestWithService(
				t,
				newChatMessagesCompatibilityService(sessionGetService, messageService),
				testCase.request,
			)
			messages := chatMessagesFromFrame(t, frame, testCase.asToolsCallPath)
			if !reflect.DeepEqual(messageService.messages, messages) {
				t.Fatalf("expected messages payload to match service output")
			}
			assertMessageListCall(
				t,
				messageService.call,
				messageListExpectation{
					actorUserID: actorUserID,
					sessionID:   sessionID,
					limit:       12,
					offset:      3,
				},
			)
		})
	}
}

func TestCompatibilityServiceChatListMessagesUsesDefaults(t *testing.T) {
	actorUserID := uuid.MustParse("29100000-0000-0000-0000-000000000291")
	sessionID := uuid.MustParse("29100000-0000-0000-0000-000000000292")
	messageService := &fakeMessageListService{}
	frame := runCompatibilityRequestWithService(
		t,
		newChatMessagesCompatibilityService(
			&fakeSessionGetService{session: &models.ChatSessionRecord{SessionID: sessionID}},
			messageService,
		),
		directToolRequest(actorUserID.String(), "chat.list_messages", map[string]any{
			"session_id": sessionID.String(),
		}),
	)
	_ = chatMessagesFromFrame(t, frame, false)
	assertMessageListCall(
		t,
		messageService.call,
		messageListExpectation{
			actorUserID: actorUserID,
			sessionID:   sessionID,
			limit:       defaultChatMessagesLimit,
			offset:      defaultChatMessagesOffset,
		},
	)
}

func TestCompatibilityServiceChatListMessagesErrors(t *testing.T) {
	sessionID := uuid.MustParse("29200000-0000-0000-0000-000000000292")

	notFoundFrame := runCompatibilityRequestWithService(
		t,
		newChatMessagesCompatibilityService(
			&fakeSessionGetService{},
			&fakeMessageListService{},
		),
		toolsCallRequest(
			"29200000-0000-0000-0000-000000000293",
			"chat_list_messages",
			map[string]any{"session_id": sessionID.String()},
		),
	)
	notFoundPayload := errorPayloadFromFrame(t, notFoundFrame)
	requireErrorCode(t, notFoundPayload, -32602)
	assertNotFoundErrorData(t, notFoundPayload)

	invalidLimitFrame := runCompatibilityRequestWithService(
		t,
		newChatMessagesCompatibilityService(
			&fakeSessionGetService{session: &models.ChatSessionRecord{SessionID: sessionID}},
			&fakeMessageListService{},
		),
		toolsCallRequest(
			"29200000-0000-0000-0000-000000000294",
			"chat_list_messages",
			map[string]any{"session_id": sessionID.String(), "limit": "bad"},
		),
	)
	invalidLimitPayload := errorPayloadFromFrame(t, invalidLimitFrame)
	requireErrorCode(t, invalidLimitPayload, -32602)

	invalidOffsetFrame := runCompatibilityRequestWithService(
		t,
		newChatMessagesCompatibilityService(
			&fakeSessionGetService{session: &models.ChatSessionRecord{SessionID: sessionID}},
			&fakeMessageListService{},
		),
		toolsCallRequest(
			"29200000-0000-0000-0000-000000000295",
			"chat_list_messages",
			map[string]any{"session_id": sessionID.String(), "offset": "bad"},
		),
	)
	invalidOffsetPayload := errorPayloadFromFrame(t, invalidOffsetFrame)
	requireErrorCode(t, invalidOffsetPayload, -32602)

	missingSessionFrame := runCompatibilityRequestWithService(
		t,
		newChatMessagesCompatibilityService(
			&fakeSessionGetService{session: &models.ChatSessionRecord{SessionID: sessionID}},
			&fakeMessageListService{},
		),
		toolsCallRequest(
			"29200000-0000-0000-0000-000000000296",
			"chat_list_messages",
			map[string]any{},
		),
	)
	missingPayload := errorPayloadFromFrame(t, missingSessionFrame)
	requireErrorCode(t, missingPayload, -32602)
}

func TestCompatibilityServiceChatListMessagesMapsServiceError(t *testing.T) {
	sessionID := uuid.MustParse("29300000-0000-0000-0000-000000000293")
	frame := runCompatibilityRequestWithService(
		t,
		newChatMessagesCompatibilityService(
			&fakeSessionGetService{session: &models.ChatSessionRecord{SessionID: sessionID}},
			&fakeMessageListService{err: errors.New("boom")},
		),
		toolsCallRequest(
			"29300000-0000-0000-0000-000000000294",
			"chat_list_messages",
			map[string]any{"session_id": sessionID.String()},
		),
	)
	errorPayload := errorPayloadFromFrame(t, frame)
	requireErrorCode(t, errorPayload, -32603)
}

func newChatMessagesCompatibilityService(
	sessionGetService SessionGetService,
	messageService MessageListService,
) Service {
	return NewCompatibilityServiceWithDependencies(
		"1.2.3",
		CompatibilityServiceDependencies{
			SessionGet:     sessionGetService,
			MessageService: messageService,
		},
	)
}

func chatMessagesFromFrame(t *testing.T, frame Frame, asToolsCallPath bool) []models.ChatMessageRecord {
	t.Helper()
	result := resultPayloadFromFrame(t, frame)
	payload := result
	if asToolsCallPath {
		payload = mapFromMap(t, result, "structuredContent")
	}
	rawMessages, ok := payload["messages"]
	if !ok {
		t.Fatalf("expected messages payload")
	}
	return toChatMessageRecords(t, rawMessages)
}

func toChatMessageRecords(t *testing.T, value any) []models.ChatMessageRecord {
	t.Helper()
	switch typed := value.(type) {
	case []models.ChatMessageRecord:
		return typed
	case []any:
		records := make([]models.ChatMessageRecord, 0, len(typed))
		for _, item := range typed {
			record, ok := item.(models.ChatMessageRecord)
			if !ok {
				t.Fatalf("expected ChatMessageRecord item, got %T", item)
			}
			records = append(records, record)
		}
		return records
	default:
		t.Fatalf("expected []ChatMessageRecord payload, got %T", value)
		return nil
	}
}

func assertMessageListCall(
	t *testing.T,
	call messageListCall,
	expected messageListExpectation,
) {
	t.Helper()
	if call.actorUserID != expected.actorUserID {
		t.Fatalf("expected actor user id forwarded")
	}
	if call.sessionID != expected.sessionID {
		t.Fatalf("expected session id forwarded")
	}
	if call.limit != expected.limit || call.offset != expected.offset {
		t.Fatalf("expected limit/offset %d/%d, got %d/%d", expected.limit, expected.offset, call.limit, call.offset)
	}
}

type messageListCall struct {
	actorUserID uuid.UUID
	sessionID   uuid.UUID
	limit       int
	offset      int
}

type messageListExpectation struct {
	actorUserID uuid.UUID
	sessionID   uuid.UUID
	limit       int
	offset      int
}

type fakeMessageListService struct {
	messages []models.ChatMessageRecord
	err      error
	call     messageListCall
}

func (service *fakeMessageListService) ListMessages(
	_ context.Context,
	request MessageListRequest,
) ([]models.ChatMessageRecord, error) {
	service.call = messageListCall{
		actorUserID: request.ActorUserID,
		sessionID:   request.SessionID,
		limit:       request.Limit,
		offset:      request.Offset,
	}
	if service.err != nil {
		return nil, service.err
	}
	return service.messages, nil
}
