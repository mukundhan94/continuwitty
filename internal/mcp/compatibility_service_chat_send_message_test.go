package mcp

import (
	"context"
	"errors"
	"reflect"
	"testing"

	"engram/internal/chat"

	"github.com/google/uuid"
)

func TestCompatibilityServiceChatSendMessageParity(t *testing.T) {
	actorUserID := uuid.MustParse("39050000-0000-0000-0000-000000000390")
	sessionID := uuid.MustParse("39050000-0000-0000-0000-000000000391")
	response := &MessageSendResponse{
		SessionID:            sessionID,
		MessageID:            uuid.MustParse("39050000-0000-0000-0000-000000000392"),
		ReplyMessageID:       uuid.MustParse("39050000-0000-0000-0000-000000000393"),
		AssistantText:        "Acknowledged.",
		UsedEngramIDs:        []uuid.UUID{uuid.MustParse("39050000-0000-0000-0000-000000000394")},
		UsedDocumentChunkIDs: []uuid.UUID{uuid.MustParse("39050000-0000-0000-0000-000000000395")},
		SourceReferences:     []map[string]any{{"source_type": "engram_source", "title": "Pinned"}},
	}
	sendService := &fakeMessageSendService{response: response}
	params := map[string]any{
		"session_id":   sessionID.String(),
		"content_text": "Summarize previous decisions",
	}

	testCases := []struct {
		name            string
		request         StreamCallRequest
		asToolsCallPath bool
	}{
		{
			name:            "direct",
			request:         directToolRequest(actorUserID.String(), "chat.send_message", params),
			asToolsCallPath: false,
		},
		{
			name:            "tools call",
			request:         toolsCallRequest(actorUserID.String(), "chat_send_message", params),
			asToolsCallPath: true,
		},
	}

	for _, testCase := range testCases {
		testCase := testCase
		t.Run(testCase.name, func(t *testing.T) {
			frame := runCompatibilityRequestWithService(
				t,
				newMessageSendCompatibilityService(sendService),
				testCase.request,
			)
			message := messageSendFromFrame(t, frame, testCase.asToolsCallPath)
			if !reflect.DeepEqual(*response, message) {
				t.Fatalf("expected send-message payload to match service output")
			}
			if sendService.call.ActorUserID != actorUserID {
				t.Fatalf("expected actor user id forwarded")
			}
			if sendService.call.SessionID != sessionID {
				t.Fatalf("expected session id forwarded")
			}
			if sendService.call.ContentText != "Summarize previous decisions" {
				t.Fatalf("expected content text forwarded")
			}
		})
	}
}

func TestCompatibilityServiceChatSendMessageUsesDefaultContentText(t *testing.T) {
	sendService := &fakeMessageSendService{
		response: &MessageSendResponse{
			SessionID:      uuid.MustParse("39060000-0000-0000-0000-000000000390"),
			MessageID:      uuid.MustParse("39060000-0000-0000-0000-000000000391"),
			ReplyMessageID: uuid.MustParse("39060000-0000-0000-0000-000000000392"),
		},
	}
	frame := runCompatibilityRequestWithService(
		t,
		newMessageSendCompatibilityService(sendService),
		directToolRequest(
			"39060000-0000-0000-0000-000000000393",
			"chat.send_message",
			map[string]any{"session_id": "39060000-0000-0000-0000-000000000394"},
		),
	)
	_ = messageSendFromFrame(t, frame, false)
	if sendService.call.ContentText != "" {
		t.Fatalf("expected content_text default to empty string")
	}
}

func TestCompatibilityServiceChatSendMessageForwardsLinkRecallOptions(t *testing.T) {
	sendService := &fakeMessageSendService{
		response: &MessageSendResponse{
			SessionID:      uuid.MustParse("39061000-0000-0000-0000-000000000390"),
			MessageID:      uuid.MustParse("39061000-0000-0000-0000-000000000391"),
			ReplyMessageID: uuid.MustParse("39061000-0000-0000-0000-000000000392"),
		},
	}
	frame := runCompatibilityRequestWithService(
		t,
		newMessageSendCompatibilityService(sendService),
		directToolRequest(
			"39061000-0000-0000-0000-000000000393",
			"chat.send_message",
			map[string]any{
				"session_id":                     "39061000-0000-0000-0000-000000000394",
				"content_text":                   "use linked recall",
				"link_recall_enabled":            false,
				"link_recall_depth":              2,
				"link_recall_max_neighbors":      7,
				"link_noise_suppression_enabled": false,
				"link_noise_score_threshold":     0.61,
			},
		),
	)
	_ = messageSendFromFrame(t, frame, false)
	requireBoolPointerField(t, sendService.call.LinkRecallEnabled, false, "link_recall_enabled")
	requireIntPointerField(t, sendService.call.LinkRecallDepth, 2, "link_recall_depth")
	requireIntPointerField(t, sendService.call.LinkRecallMaxNeighbors, 7, "link_recall_max_neighbors")
	requireBoolPointerField(t, sendService.call.LinkNoiseSuppressionEnabled, false, "link_noise_suppression_enabled")
	requireFloatPointerField(t, sendService.call.LinkNoiseScoreThreshold, 0.61, "link_noise_score_threshold")
}

func TestCompatibilityServiceChatSendMessageValidationErrors(t *testing.T) {
	service := newMessageSendCompatibilityService(&fakeMessageSendService{})
	testCases := []struct {
		name   string
		params map[string]any
	}{
		{name: "missing session id", params: map[string]any{"content_text": "x"}},
		{name: "invalid session id", params: map[string]any{"session_id": "bad", "content_text": "x"}},
	}

	for _, testCase := range testCases {
		testCase := testCase
		t.Run(testCase.name, func(t *testing.T) {
			frame := runCompatibilityRequestWithService(
				t,
				service,
				toolsCallRequest(
					"39070000-0000-0000-0000-000000000390",
					"chat_send_message",
					testCase.params,
				),
			)
			requireErrorCode(t, errorPayloadFromFrame(t, frame), -32602)
		})
	}
}

func TestCompatibilityServiceChatSendMessageErrorMappings(t *testing.T) {
	validationFrame := runCompatibilityRequestWithService(
		t,
		newMessageSendCompatibilityService(
			&fakeMessageSendService{err: chat.NewChatValidationError("Message content cannot be empty")},
		),
		toolsCallRequest(
			"39080000-0000-0000-0000-000000000390",
			"chat_send_message",
			map[string]any{"session_id": "39080000-0000-0000-0000-000000000391"},
		),
	)
	validationPayload := errorPayloadFromFrame(t, validationFrame)
	requireErrorCode(t, validationPayload, -32010)
	assertErrorMessage(t, validationPayload, "Message content cannot be empty")
	assertErrorStatusCode(t, validationPayload, 400)

	providerFrame := runCompatibilityRequestWithService(
		t,
		newMessageSendCompatibilityService(
			&fakeMessageSendService{
				err: chat.NewChatProviderExecutionError(
					"provider unavailable",
					503,
					"provider_unavailable",
				),
			},
		),
		toolsCallRequest(
			"39080000-0000-0000-0000-000000000392",
			"chat_send_message",
			map[string]any{"session_id": "39080000-0000-0000-0000-000000000393"},
		),
	)
	providerPayload := errorPayloadFromFrame(t, providerFrame)
	requireErrorCode(t, providerPayload, -32020)
	assertErrorMessage(t, providerPayload, "provider unavailable")
	assertErrorStatusCode(t, providerPayload, 503)
	assertErrorCodeField(t, providerPayload, "provider_unavailable")

	internalFrame := runCompatibilityRequestWithService(
		t,
		newMessageSendCompatibilityService(&fakeMessageSendService{err: errors.New("boom")}),
		toolsCallRequest(
			"39080000-0000-0000-0000-000000000394",
			"chat_send_message",
			map[string]any{"session_id": "39080000-0000-0000-0000-000000000395"},
		),
	)
	requireErrorCode(t, errorPayloadFromFrame(t, internalFrame), -32603)
}

func newMessageSendCompatibilityService(messageSendService MessageSendService) Service {
	return NewCompatibilityServiceWithDependencies(
		"1.2.3",
		CompatibilityServiceDependencies{MessageSend: messageSendService},
	)
}

func messageSendFromFrame(
	t *testing.T,
	frame Frame,
	asToolsCallPath bool,
) MessageSendResponse {
	t.Helper()
	result := resultPayloadFromFrame(t, frame)
	payload := result
	if asToolsCallPath {
		payload = mapFromMap(t, result, "structuredContent")
	}
	message, ok := payload["message"].(MessageSendResponse)
	if !ok {
		t.Fatalf("expected send-message payload")
	}
	return message
}

func assertErrorMessage(t *testing.T, errorPayload map[string]any, expected string) {
	t.Helper()
	if errorPayload["message"] != expected {
		t.Fatalf("expected error message %q", expected)
	}
}

func assertErrorStatusCode(t *testing.T, errorPayload map[string]any, expected int) {
	t.Helper()
	data := mapFromMap(t, errorPayload, "data")
	if data["status_code"] != expected {
		t.Fatalf("expected status_code %d in error data", expected)
	}
}

func assertErrorCodeField(t *testing.T, errorPayload map[string]any, expected string) {
	t.Helper()
	data := mapFromMap(t, errorPayload, "data")
	if data["error_code"] != expected {
		t.Fatalf("expected error_code %q in error data", expected)
	}
}

func requireBoolPointerField(t *testing.T, value *bool, expected bool, field string) {
	t.Helper()
	if value == nil || *value != expected {
		t.Fatalf("expected %s=%v, got %v", field, expected, value)
	}
}

func requireIntPointerField(t *testing.T, value *int, expected int, field string) {
	t.Helper()
	if value == nil || *value != expected {
		t.Fatalf("expected %s=%d, got %v", field, expected, value)
	}
}

func requireFloatPointerField(t *testing.T, value *float64, expected float64, field string) {
	t.Helper()
	if value == nil || *value != expected {
		t.Fatalf("expected %s=%v, got %v", field, expected, value)
	}
}

type fakeMessageSendService struct {
	response *MessageSendResponse
	err      error
	call     SessionMessageSendRequest
}

func (service *fakeMessageSendService) SendMessage(
	_ context.Context,
	request SessionMessageSendRequest,
) (*MessageSendResponse, error) {
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
