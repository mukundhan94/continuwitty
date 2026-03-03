package mcp

import (
	"context"
	"testing"

	"github.com/google/uuid"
)

func TestCompatibilityServiceChatSendMessageDirectStreamFrames(t *testing.T) {
	actorUserID := uuid.MustParse("39090000-0000-0000-0000-000000000390")
	sessionID := uuid.MustParse("39090000-0000-0000-0000-000000000391")
	streamService := &fakeMessageStreamService{
		events: []MessageStreamEvent{
			{Type: "chunk", Payload: map[string]any{"text": "hello"}},
			{Type: "done", Payload: map[string]any{"message_id": "m1"}},
		},
	}
	service := newMessageStreamCompatibilityService(nil, streamService)

	frames := collectAllFrames(
		t,
		service.StreamCall(
			context.Background(),
			directToolRequest(
				actorUserID.String(),
				"chat.send_message",
				map[string]any{
					"session_id":                     sessionID.String(),
					"content_text":                   "hi",
					"context_token_budget":           512,
					"link_recall_enabled":            true,
					"link_recall_depth":              2,
					"link_recall_max_neighbors":      6,
					"link_noise_suppression_enabled": true,
					"link_noise_score_threshold":     0.58,
				},
			),
		),
	)

	if len(frames) != 3 {
		t.Fatalf("expected chunk event, done event, and terminal success frame")
	}
	assertMCPEventFrame(t, frames[0], "chat.send_message", "chunk")
	assertMCPEventFrame(t, frames[1], "chat.send_message", "done")
	resultPayload := resultPayloadFromFrame(t, frames[2])
	messagePayload := mapFromMap(t, resultPayload, "message")
	if messagePayload["message_id"] != "m1" {
		t.Fatalf("expected terminal success payload to use done event message")
	}
	if !streamService.called {
		t.Fatalf("expected stream service call")
	}
	if streamService.call.ActorUserID != actorUserID || streamService.call.SessionID != sessionID {
		t.Fatalf("expected actor/session forwarded to stream service")
	}
	assertForwardedLinkRecallOptions(
		t,
		streamService.call,
		expectedLinkRecallOptions{
			contextTokenBudget:      512,
			enabled:                 true,
			depth:                   2,
			maxNeighbors:            6,
			noiseSuppressionEnabled: true,
			noiseScoreThreshold:     0.58,
		},
	)
}

func TestCompatibilityServiceToolsCallChatSendMessageStreamsWhenEnabled(t *testing.T) {
	actorUserID := uuid.MustParse("39100000-0000-0000-0000-000000000391")
	streamService := &fakeMessageStreamService{
		events: []MessageStreamEvent{
			{Type: "chunk", Payload: map[string]any{"text": "hello"}},
			{Type: "done", Payload: map[string]any{"message_id": "m-tools"}},
		},
	}
	service := newMessageStreamCompatibilityService(nil, streamService)

	frames := collectAllFrames(
		t,
		service.StreamCall(
			context.Background(),
			toolsCallRequest(
				actorUserID.String(),
				"chat_send_message",
				map[string]any{
					"session_id": uuid.MustParse("39100000-0000-0000-0000-000000000392").String(),
					"stream":     true,
				},
			),
		),
	)

	if len(frames) != 3 {
		t.Fatalf("expected chunk event, done event, and terminal success frame")
	}
	assertMCPEventFrame(t, frames[0], "chat_send_message", "chunk")
	assertMCPEventFrame(t, frames[1], "chat_send_message", "done")
	resultPayload := resultPayloadFromFrame(t, frames[2])
	if resultPayload["tool_name"] != "chat_send_message" {
		t.Fatalf("expected tool_name to echo tools/call request name")
	}
	structuredContent := mapFromMap(t, resultPayload, "structuredContent")
	messagePayload := mapFromMap(t, structuredContent, "message")
	if messagePayload["message_id"] != "m-tools" {
		t.Fatalf("expected terminal tools/call payload to use done event message")
	}
}

func TestCompatibilityServiceToolsCallChatSendMessageStreamDisabledUsesNonStreamDispatch(t *testing.T) {
	actorUserID := uuid.MustParse("39110000-0000-0000-0000-000000000391")
	sendService := &fallbackMessageSendService{
		response: &MessageSendResponse{
			SessionID:      uuid.MustParse("39110000-0000-0000-0000-000000000392"),
			MessageID:      uuid.MustParse("39110000-0000-0000-0000-000000000393"),
			ReplyMessageID: uuid.MustParse("39110000-0000-0000-0000-000000000394"),
		},
	}
	streamService := &fakeMessageStreamService{}
	service := newMessageStreamCompatibilityService(sendService, streamService)

	frames := collectAllFrames(
		t,
		service.StreamCall(
			context.Background(),
			toolsCallRequest(
				actorUserID.String(),
				"chat_send_message",
				map[string]any{
					"session_id": uuid.MustParse("39110000-0000-0000-0000-000000000395").String(),
					"stream":     false,
				},
			),
		),
	)

	if len(frames) != 1 {
		t.Fatalf("expected a single non-stream terminal frame")
	}
	resultPayload := resultPayloadFromFrame(t, frames[0])
	structuredContent := mapFromMap(t, resultPayload, "structuredContent")
	if _, ok := structuredContent["message"]; !ok {
		t.Fatalf("expected non-stream dispatch message payload")
	}
	if streamService.called {
		t.Fatalf("expected stream service to remain unused when stream=false")
	}
}

func TestCompatibilityServiceChatSendMessageStreamRequiresDoneEvent(t *testing.T) {
	streamService := &fakeMessageStreamService{
		events: []MessageStreamEvent{{Type: "chunk", Payload: map[string]any{"text": "hello"}}},
	}
	service := newMessageStreamCompatibilityService(nil, streamService)

	frame := terminalFrame(
		t,
		collectAllFrames(
			t,
			service.StreamCall(
				context.Background(),
				directToolRequest(
					"39120000-0000-0000-0000-000000000391",
					"chat.send_message",
					map[string]any{"session_id": "39120000-0000-0000-0000-000000000392"},
				),
			),
		),
	)
	requireErrorCode(t, errorPayloadFromFrame(t, frame), -32021)
}

func TestCompatibilityServiceChatSendMessageStreamErrorEventMapsToProviderError(t *testing.T) {
	streamService := &fakeMessageStreamService{
		events: []MessageStreamEvent{
			{
				Type:    "error",
				Payload: map[string]any{"detail": "provider unavailable", "status_code": 503, "error_code": "provider_unavailable"},
			},
		},
	}
	service := newMessageStreamCompatibilityService(nil, streamService)

	frames := collectAllFrames(
		t,
		service.StreamCall(
			context.Background(),
			directToolRequest(
				"39130000-0000-0000-0000-000000000391",
				"chat.send_message",
				map[string]any{"session_id": "39130000-0000-0000-0000-000000000392"},
			),
		),
	)
	if len(frames) != 2 {
		t.Fatalf("expected event frame plus terminal error frame")
	}
	assertMCPEventFrame(t, frames[0], "chat.send_message", "error")
	errorPayload := errorPayloadFromFrame(t, frames[1])
	requireErrorCode(t, errorPayload, -32020)
	if errorPayload["message"] != "provider unavailable" {
		t.Fatalf("expected stream error detail to become rpc error message")
	}
}

func newMessageStreamCompatibilityService(
	sendService MessageSendService,
	streamService MessageStreamService,
) Service {
	return NewCompatibilityServiceWithDependencies(
		"1.2.3",
		CompatibilityServiceDependencies{
			MessageSend:   sendService,
			MessageStream: streamService,
		},
	)
}

func assertMCPEventFrame(t *testing.T, frame Frame, expectedTool string, expectedEvent string) {
	t.Helper()
	if frame["method"] != "mcp.event" {
		t.Fatalf("expected mcp.event frame method")
	}
	params := mapFromMap(t, frame, "params")
	if params["tool"] != expectedTool {
		t.Fatalf("expected event tool %q", expectedTool)
	}
	if params["event"] != expectedEvent {
		t.Fatalf("expected event type %q", expectedEvent)
	}
}

func collectAllFrames(t *testing.T, frames <-chan Frame) []Frame {
	t.Helper()
	collected := make([]Frame, 0)
	for frame := range frames {
		collected = append(collected, frame)
	}
	if len(collected) == 0 {
		t.Fatalf("expected at least one frame")
	}
	return collected
}

func terminalFrame(t *testing.T, frames []Frame) Frame {
	t.Helper()
	return frames[len(frames)-1]
}

type expectedLinkRecallOptions struct {
	contextTokenBudget      int
	enabled                 bool
	depth                   int
	maxNeighbors            int
	noiseSuppressionEnabled bool
	noiseScoreThreshold     float64
}

func assertForwardedLinkRecallOptions(
	t *testing.T,
	request SessionMessageSendRequest,
	expected expectedLinkRecallOptions,
) {
	t.Helper()
	requireIntOption(t, request.ContextTokenBudget, expected.contextTokenBudget, "context_token_budget")
	requireBoolOption(t, request.LinkRecallEnabled, expected.enabled, "link_recall_enabled")
	requireIntOption(t, request.LinkRecallDepth, expected.depth, "link_recall_depth")
	requireIntOption(t, request.LinkRecallMaxNeighbors, expected.maxNeighbors, "link_recall_max_neighbors")
	requireBoolOption(
		t,
		request.LinkNoiseSuppressionEnabled,
		expected.noiseSuppressionEnabled,
		"link_noise_suppression_enabled",
	)
	requireFloatOption(
		t,
		request.LinkNoiseScoreThreshold,
		expected.noiseScoreThreshold,
		"link_noise_score_threshold",
	)
}

func requireBoolOption(t *testing.T, value *bool, expected bool, field string) {
	t.Helper()
	if value == nil || *value != expected {
		t.Fatalf("expected %s=%v forwarded to stream service", field, expected)
	}
}

func requireIntOption(t *testing.T, value *int, expected int, field string) {
	t.Helper()
	if value == nil || *value != expected {
		t.Fatalf("expected %s=%d forwarded to stream service", field, expected)
	}
}

func requireFloatOption(t *testing.T, value *float64, expected float64, field string) {
	t.Helper()
	if value == nil || *value != expected {
		t.Fatalf("expected %s=%.2f forwarded to stream service", field, expected)
	}
}

type fakeMessageStreamService struct {
	events []MessageStreamEvent
	err    error
	call   SessionMessageSendRequest
	called bool
}

func (service *fakeMessageStreamService) StreamMessageEvents(
	_ context.Context,
	request SessionMessageSendRequest,
) ([]MessageStreamEvent, error) {
	service.called = true
	service.call = request
	if service.err != nil {
		return nil, service.err
	}
	events := make([]MessageStreamEvent, 0, len(service.events))
	for _, event := range service.events {
		events = append(
			events,
			MessageStreamEvent{
				Type:    event.Type,
				Payload: mapStringAnyCopy(event.Payload),
			},
		)
	}
	return events, nil
}

type fallbackMessageSendService struct {
	response *MessageSendResponse
	err      error
}

func (service *fallbackMessageSendService) SendMessage(
	_ context.Context,
	_ SessionMessageSendRequest,
) (*MessageSendResponse, error) {
	if service.err != nil {
		return nil, service.err
	}
	if service.response == nil {
		return nil, nil
	}
	response := *service.response
	return &response, nil
}

func mapStringAnyCopy(input map[string]any) map[string]any {
	if input == nil {
		return nil
	}
	copied := make(map[string]any, len(input))
	for key, value := range input {
		copied[key] = value
	}
	return copied
}
