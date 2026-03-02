package mcp

import (
	"context"
	"errors"
	"reflect"
	"testing"
	"time"

	"engram/internal/models"

	"github.com/google/uuid"
)

func TestCompatibilityServiceChatListTimelineParity(t *testing.T) {
	actorUserID, sessionID, sessionGetService, timelineService := newTimelineParityFixtures()
	for _, testCase := range newTimelineParityCases(actorUserID, sessionID) {
		testCase := testCase
		t.Run(testCase.name, func(t *testing.T) {
			assertTimelineParityCase(
				t,
				timelineParityRunInput{
					service:         newChatTimelineCompatibilityService(sessionGetService, timelineService),
					timelineService: timelineService,
					testCase:        testCase,
					expectedCall: timelineListExpectation{
						actorUserID: actorUserID,
						sessionID:   sessionID,
						limit:       9,
						offset:      4,
					},
				},
			)
		})
	}
}

type timelineParityCase struct {
	name            string
	request         StreamCallRequest
	asToolsCallPath bool
}

func newTimelineParityFixtures() (
	uuid.UUID,
	uuid.UUID,
	*fakeSessionGetService,
	*fakeTimelineListService,
) {
	actorUserID := uuid.MustParse("30000000-0000-0000-0000-000000000300")
	sessionID := uuid.MustParse("30000000-0000-0000-0000-000000000301")
	sessionGetService := &fakeSessionGetService{
		session: &models.ChatSessionRecord{SessionID: sessionID, ProjectID: "proj-alpha"},
	}
	eventID := uuid.MustParse("30000000-0000-0000-0000-000000000302")
	createdAt := time.Unix(1700000000, 0).UTC()
	timelineService := &fakeTimelineListService{
		events: []models.ChatTimelineEvent{
			{
				EventID:                  eventID,
				SessionID:                sessionID,
				EventType:                "consolidation_merge",
				Title:                    "Merge summary",
				Abstract:                 "Consolidated snapshots",
				Tags:                     []string{"consolidated", "consolidation_group_key:incident-42", "consolidation_merged_count:3"},
				ConsolidationGroupKey:    stringPtr("incident-42"),
				ConsolidationMergedCount: intPtr(3),
				CreatedAt:                createdAt,
			},
		},
	}
	return actorUserID, sessionID, sessionGetService, timelineService
}

func newTimelineParityCases(actorUserID uuid.UUID, sessionID uuid.UUID) []timelineParityCase {
	return []timelineParityCase{
		{
			name: "direct",
			request: directToolRequest(
				actorUserID.String(),
				"chat.list_timeline",
				map[string]any{
					"session_id": sessionID.String(),
					"limit":      9.0,
					"offset":     4.0,
				},
			),
			asToolsCallPath: false,
		},
		{
			name: "tools call",
			request: toolsCallRequest(
				actorUserID.String(),
				"chat_list_timeline",
				map[string]any{
					"session_id": sessionID.String(),
					"limit":      9.0,
					"offset":     4.0,
				},
			),
			asToolsCallPath: true,
		},
	}
}

type timelineParityRunInput struct {
	service         Service
	timelineService *fakeTimelineListService
	testCase        timelineParityCase
	expectedCall    timelineListExpectation
}

func assertTimelineParityCase(
	t *testing.T,
	input timelineParityRunInput,
) {
	t.Helper()
	frame := runCompatibilityRequestWithService(t, input.service, input.testCase.request)
	events := chatTimelineFromFrame(t, frame, input.testCase.asToolsCallPath)
	if !reflect.DeepEqual(input.timelineService.events, events) {
		t.Fatalf("expected events payload to match service output")
	}
	assertTimelineListCall(t, input.timelineService.call, input.expectedCall)
}

func TestCompatibilityServiceChatListTimelineUsesDefaults(t *testing.T) {
	actorUserID := uuid.MustParse("30100000-0000-0000-0000-000000000301")
	sessionID := uuid.MustParse("30100000-0000-0000-0000-000000000302")
	timelineService := &fakeTimelineListService{}
	frame := runCompatibilityRequestWithService(
		t,
		newChatTimelineCompatibilityService(
			&fakeSessionGetService{session: &models.ChatSessionRecord{SessionID: sessionID}},
			timelineService,
		),
		directToolRequest(actorUserID.String(), "chat.list_timeline", map[string]any{
			"session_id": sessionID.String(),
		}),
	)
	_ = chatTimelineFromFrame(t, frame, false)
	assertTimelineListCall(
		t,
		timelineService.call,
		timelineListExpectation{
			actorUserID: actorUserID,
			sessionID:   sessionID,
			limit:       defaultChatTimelineLimit,
			offset:      defaultChatTimelineOffset,
		},
	)
}

func TestCompatibilityServiceChatListTimelineErrors(t *testing.T) {
	sessionID := uuid.MustParse("30200000-0000-0000-0000-000000000302")

	notFoundFrame := runCompatibilityRequestWithService(
		t,
		newChatTimelineCompatibilityService(&fakeSessionGetService{}, &fakeTimelineListService{}),
		toolsCallRequest(
			"30200000-0000-0000-0000-000000000303",
			"chat_list_timeline",
			map[string]any{"session_id": sessionID.String()},
		),
	)
	notFoundPayload := errorPayloadFromFrame(t, notFoundFrame)
	requireErrorCode(t, notFoundPayload, -32602)
	assertNotFoundErrorData(t, notFoundPayload)

	invalidLimitFrame := runCompatibilityRequestWithService(
		t,
		newChatTimelineCompatibilityService(
			&fakeSessionGetService{session: &models.ChatSessionRecord{SessionID: sessionID}},
			&fakeTimelineListService{},
		),
		toolsCallRequest(
			"30200000-0000-0000-0000-000000000304",
			"chat_list_timeline",
			map[string]any{"session_id": sessionID.String(), "limit": "bad"},
		),
	)
	invalidLimitPayload := errorPayloadFromFrame(t, invalidLimitFrame)
	requireErrorCode(t, invalidLimitPayload, -32602)

	invalidOffsetFrame := runCompatibilityRequestWithService(
		t,
		newChatTimelineCompatibilityService(
			&fakeSessionGetService{session: &models.ChatSessionRecord{SessionID: sessionID}},
			&fakeTimelineListService{},
		),
		toolsCallRequest(
			"30200000-0000-0000-0000-000000000305",
			"chat_list_timeline",
			map[string]any{"session_id": sessionID.String(), "offset": "bad"},
		),
	)
	invalidOffsetPayload := errorPayloadFromFrame(t, invalidOffsetFrame)
	requireErrorCode(t, invalidOffsetPayload, -32602)

	missingSessionFrame := runCompatibilityRequestWithService(
		t,
		newChatTimelineCompatibilityService(
			&fakeSessionGetService{session: &models.ChatSessionRecord{SessionID: sessionID}},
			&fakeTimelineListService{},
		),
		toolsCallRequest(
			"30200000-0000-0000-0000-000000000306",
			"chat_list_timeline",
			map[string]any{},
		),
	)
	missingPayload := errorPayloadFromFrame(t, missingSessionFrame)
	requireErrorCode(t, missingPayload, -32602)
}

func TestCompatibilityServiceChatListTimelineMapsServiceError(t *testing.T) {
	sessionID := uuid.MustParse("30300000-0000-0000-0000-000000000303")
	frame := runCompatibilityRequestWithService(
		t,
		newChatTimelineCompatibilityService(
			&fakeSessionGetService{session: &models.ChatSessionRecord{SessionID: sessionID}},
			&fakeTimelineListService{err: errors.New("boom")},
		),
		toolsCallRequest(
			"30300000-0000-0000-0000-000000000304",
			"chat_list_timeline",
			map[string]any{"session_id": sessionID.String()},
		),
	)
	errorPayload := errorPayloadFromFrame(t, frame)
	requireErrorCode(t, errorPayload, -32603)
}

func newChatTimelineCompatibilityService(
	sessionGetService SessionGetService,
	timelineService TimelineListService,
) Service {
	return NewCompatibilityServiceWithDependencies(
		"1.2.3",
		CompatibilityServiceDependencies{
			SessionGet:      sessionGetService,
			TimelineService: timelineService,
		},
	)
}

func chatTimelineFromFrame(t *testing.T, frame Frame, asToolsCallPath bool) []models.ChatTimelineEvent {
	t.Helper()
	result := resultPayloadFromFrame(t, frame)
	payload := result
	if asToolsCallPath {
		payload = mapFromMap(t, result, "structuredContent")
	}
	rawEvents, ok := payload["events"]
	if !ok {
		t.Fatalf("expected events payload")
	}
	return toChatTimelineEvents(t, rawEvents)
}

func toChatTimelineEvents(t *testing.T, value any) []models.ChatTimelineEvent {
	t.Helper()
	switch typed := value.(type) {
	case []models.ChatTimelineEvent:
		return typed
	case []any:
		events := make([]models.ChatTimelineEvent, 0, len(typed))
		for _, item := range typed {
			event, ok := item.(models.ChatTimelineEvent)
			if !ok {
				t.Fatalf("expected ChatTimelineEvent item, got %T", item)
			}
			events = append(events, event)
		}
		return events
	default:
		t.Fatalf("expected []ChatTimelineEvent payload, got %T", value)
		return nil
	}
}

func assertTimelineListCall(
	t *testing.T,
	call timelineListCall,
	expected timelineListExpectation,
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

type timelineListCall struct {
	actorUserID uuid.UUID
	sessionID   uuid.UUID
	limit       int
	offset      int
}

type timelineListExpectation struct {
	actorUserID uuid.UUID
	sessionID   uuid.UUID
	limit       int
	offset      int
}

type fakeTimelineListService struct {
	events []models.ChatTimelineEvent
	err    error
	call   timelineListCall
}

func (service *fakeTimelineListService) ListTimeline(
	_ context.Context,
	request TimelineListRequest,
) ([]models.ChatTimelineEvent, error) {
	service.call = timelineListCall{
		actorUserID: request.ActorUserID,
		sessionID:   request.SessionID,
		limit:       request.Limit,
		offset:      request.Offset,
	}
	if service.err != nil {
		return nil, service.err
	}
	return service.events, nil
}

func intPtr(value int) *int {
	return &value
}
