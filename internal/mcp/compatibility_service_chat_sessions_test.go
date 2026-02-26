package mcp

import (
	"reflect"
	"testing"

	"engram/internal/models"

	"github.com/google/uuid"
)

func TestCompatibilityServiceChatListSessionsParity(t *testing.T) {
	actorUserID := uuid.MustParse("26000000-0000-0000-0000-000000000260")
	service := &fakeSessionListService{
		sessions: []models.ChatSessionRecord{
			{SessionID: uuid.MustParse("26000000-0000-0000-0000-000000000261"), ProjectID: "proj-alpha", Title: "Alpha"},
			{SessionID: uuid.MustParse("26000000-0000-0000-0000-000000000262"), ProjectID: "proj-alpha", Title: "Alpha-2"},
		},
	}

	testCases := []struct {
		name            string
		request         StreamCallRequest
		asToolsCallPath bool
	}{
		{
			name:            "direct",
			request:         directToolRequest(actorUserID.String(), "chat.list_sessions", map[string]any{"project_id": "proj-alpha", "limit": 25.0, "offset": 2.0}),
			asToolsCallPath: false,
		},
		{
			name:            "tools call",
			request:         toolsCallRequest(actorUserID.String(), "chat_list_sessions", map[string]any{"project_id": "proj-alpha", "limit": 25.0, "offset": 2.0}),
			asToolsCallPath: true,
		},
	}

	for _, testCase := range testCases {
		testCase := testCase
		t.Run(testCase.name, func(t *testing.T) {
			frame := runCompatibilityRequestWithService(
				t,
				newUserProjectsCompatibilityService(service),
				testCase.request,
			)
			sessions := chatSessionsFromFrame(t, frame, testCase.asToolsCallPath)
			if !reflect.DeepEqual(service.sessions, sessions) {
				t.Fatalf("expected sessions payload to match service output")
			}
			assertSessionListCall(
				t,
				service.call,
				sessionListExpectation{
					actorUserID: actorUserID,
					projectID:   stringPtr("proj-alpha"),
					limit:       25,
					offset:      2,
				},
			)
		})
	}
}

func TestCompatibilityServiceChatListSessionsUsesDefaults(t *testing.T) {
	actorUserID := uuid.MustParse("26100000-0000-0000-0000-000000000261")
	service := &fakeSessionListService{sessions: []models.ChatSessionRecord{}}

	frame := runCompatibilityRequestWithService(
		t,
		newUserProjectsCompatibilityService(service),
		directToolRequest(actorUserID.String(), "chat.list_sessions", map[string]any{}),
	)
	_ = chatSessionsFromFrame(t, frame, false)
	assertSessionListCall(
		t,
		service.call,
		sessionListExpectation{
			actorUserID: actorUserID,
			projectID:   nil,
			limit:       defaultChatSessionsLimit,
			offset:      defaultChatSessionsOffset,
		},
	)
}

func TestCompatibilityServiceChatListSessionsRejectsInvalidLimit(t *testing.T) {
	frame := runCompatibilityRequestWithService(
		t,
		newUserProjectsCompatibilityService(&fakeSessionListService{}),
		toolsCallRequest(
			"26200000-0000-0000-0000-000000000262",
			"chat_list_sessions",
			map[string]any{"limit": "bad"},
		),
	)
	errorPayload := errorPayloadFromFrame(t, frame)
	requireErrorCode(t, errorPayload, -32602)
}

func chatSessionsFromFrame(t *testing.T, frame Frame, asToolsCallPath bool) []models.ChatSessionRecord {
	t.Helper()
	result := resultPayloadFromFrame(t, frame)
	payload := result
	if asToolsCallPath {
		payload = mapFromMap(t, result, "structuredContent")
	}
	rawSessions, ok := payload["sessions"]
	if !ok {
		t.Fatalf("expected sessions payload")
	}
	return toChatSessionRecords(t, rawSessions)
}

func toChatSessionRecords(t *testing.T, value any) []models.ChatSessionRecord {
	t.Helper()
	switch typed := value.(type) {
	case []models.ChatSessionRecord:
		return typed
	case []any:
		records := make([]models.ChatSessionRecord, 0, len(typed))
		for _, item := range typed {
			record, ok := item.(models.ChatSessionRecord)
			if !ok {
				t.Fatalf("expected ChatSessionRecord item, got %T", item)
			}
			records = append(records, record)
		}
		return records
	default:
		t.Fatalf("expected []ChatSessionRecord payload, got %T", value)
		return nil
	}
}

func assertSessionListCall(
	t *testing.T,
	call sessionListCall,
	expected sessionListExpectation,
) {
	t.Helper()
	if call.actorUserID != expected.actorUserID {
		t.Fatalf("expected actor user id forwarded")
	}
	assertProjectFilter(t, call.projectID, expected.projectID)
	if call.limit != expected.limit || call.offset != expected.offset {
		t.Fatalf("expected limit/offset %d/%d, got %d/%d", expected.limit, expected.offset, call.limit, call.offset)
	}
}

func assertProjectFilter(t *testing.T, actual *string, expected *string) {
	t.Helper()
	if expected == nil {
		if actual != nil {
			t.Fatalf("expected nil project filter")
		}
		return
	}
	if actual == nil || *actual != *expected {
		t.Fatalf("expected project filter %q", *expected)
	}
}

func stringPtr(value string) *string {
	return &value
}

type sessionListExpectation struct {
	actorUserID uuid.UUID
	projectID   *string
	limit       int
	offset      int
}
