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

func TestCompatibilityServiceChatListPinnedEngramsParity(t *testing.T) {
	actorUserID := uuid.MustParse("31000000-0000-0000-0000-000000000310")
	sessionID := uuid.MustParse("31000000-0000-0000-0000-000000000311")
	sessionGetService := &fakeSessionGetService{
		session: &models.ChatSessionRecord{SessionID: sessionID, ProjectID: "proj-alpha"},
	}
	pinnedService := &fakePinnedEngramListService{
		pinned: []models.EngramSummary{
			{
				EngramID:        uuid.MustParse("31000000-0000-0000-0000-000000000312"),
				ProjectID:       "proj-alpha",
				Title:           "Pinned summary",
				Abstract:        "Important context",
				CreatedAt:       time.Unix(1700000500, 0).UTC(),
				Tags:            []string{"chat", "priority"},
				Keywords:        []string{"migration"},
				VisibilityScope: "private",
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
				"chat.list_pinned_engrams",
				map[string]any{"session_id": sessionID.String()},
			),
			asToolsCallPath: false,
		},
		{
			name: "tools call",
			request: toolsCallRequest(
				actorUserID.String(),
				"chat_list_pinned_engrams",
				map[string]any{"session_id": sessionID.String()},
			),
			asToolsCallPath: true,
		},
	}

	for _, testCase := range testCases {
		testCase := testCase
		t.Run(testCase.name, func(t *testing.T) {
			frame := runCompatibilityRequestWithService(
				t,
				newPinnedEngramsCompatibilityService(sessionGetService, pinnedService),
				testCase.request,
			)
			pinned := pinnedEngramsFromFrame(t, frame, testCase.asToolsCallPath)
			if !reflect.DeepEqual(pinnedService.pinned, pinned) {
				t.Fatalf("expected pinned_engrams payload to match service output")
			}
			if pinnedService.call.actorUserID != actorUserID {
				t.Fatalf("expected actor user id forwarded")
			}
			if pinnedService.call.sessionID != sessionID {
				t.Fatalf("expected session id forwarded")
			}
		})
	}
}

func TestCompatibilityServiceChatListPinnedEngramsErrors(t *testing.T) {
	sessionID := uuid.MustParse("31100000-0000-0000-0000-000000000311")

	notFoundFrame := runCompatibilityRequestWithService(
		t,
		newPinnedEngramsCompatibilityService(
			&fakeSessionGetService{},
			&fakePinnedEngramListService{},
		),
		toolsCallRequest(
			"31100000-0000-0000-0000-000000000312",
			"chat_list_pinned_engrams",
			map[string]any{"session_id": sessionID.String()},
		),
	)
	notFoundPayload := errorPayloadFromFrame(t, notFoundFrame)
	requireErrorCode(t, notFoundPayload, -32602)
	assertNotFoundErrorData(t, notFoundPayload)

	invalidFrame := runCompatibilityRequestWithService(
		t,
		newPinnedEngramsCompatibilityService(
			&fakeSessionGetService{session: &models.ChatSessionRecord{SessionID: sessionID}},
			&fakePinnedEngramListService{},
		),
		toolsCallRequest(
			"31100000-0000-0000-0000-000000000313",
			"chat_list_pinned_engrams",
			map[string]any{"session_id": "bad"},
		),
	)
	invalidPayload := errorPayloadFromFrame(t, invalidFrame)
	requireErrorCode(t, invalidPayload, -32602)

	missingFrame := runCompatibilityRequestWithService(
		t,
		newPinnedEngramsCompatibilityService(
			&fakeSessionGetService{session: &models.ChatSessionRecord{SessionID: sessionID}},
			&fakePinnedEngramListService{},
		),
		toolsCallRequest(
			"31100000-0000-0000-0000-000000000314",
			"chat_list_pinned_engrams",
			map[string]any{},
		),
	)
	missingPayload := errorPayloadFromFrame(t, missingFrame)
	requireErrorCode(t, missingPayload, -32602)
}

func TestCompatibilityServiceChatListPinnedEngramsMapsServiceError(t *testing.T) {
	sessionID := uuid.MustParse("31200000-0000-0000-0000-000000000312")
	frame := runCompatibilityRequestWithService(
		t,
		newPinnedEngramsCompatibilityService(
			&fakeSessionGetService{session: &models.ChatSessionRecord{SessionID: sessionID}},
			&fakePinnedEngramListService{err: errors.New("boom")},
		),
		toolsCallRequest(
			"31200000-0000-0000-0000-000000000313",
			"chat_list_pinned_engrams",
			map[string]any{"session_id": sessionID.String()},
		),
	)
	errorPayload := errorPayloadFromFrame(t, frame)
	requireErrorCode(t, errorPayload, -32603)
}

func newPinnedEngramsCompatibilityService(
	sessionGetService SessionGetService,
	pinnedService PinnedEngramListService,
) Service {
	return NewCompatibilityServiceWithDependencies(
		"1.2.3",
		CompatibilityServiceDependencies{
			SessionGet:          sessionGetService,
			PinnedEngramService: pinnedService,
		},
	)
}

func pinnedEngramsFromFrame(t *testing.T, frame Frame, asToolsCallPath bool) []models.EngramSummary {
	t.Helper()
	result := resultPayloadFromFrame(t, frame)
	payload := result
	if asToolsCallPath {
		payload = mapFromMap(t, result, "structuredContent")
	}
	rawPinned, ok := payload["pinned_engrams"]
	if !ok {
		t.Fatalf("expected pinned_engrams payload")
	}
	return toEngramSummaries(t, rawPinned)
}

func toEngramSummaries(t *testing.T, value any) []models.EngramSummary {
	t.Helper()
	switch typed := value.(type) {
	case []models.EngramSummary:
		return typed
	case []any:
		records := make([]models.EngramSummary, 0, len(typed))
		for _, item := range typed {
			record, ok := item.(models.EngramSummary)
			if !ok {
				t.Fatalf("expected EngramSummary item, got %T", item)
			}
			records = append(records, record)
		}
		return records
	default:
		t.Fatalf("expected []EngramSummary payload, got %T", value)
		return nil
	}
}

type pinnedEngramListCall struct {
	actorUserID uuid.UUID
	sessionID   uuid.UUID
}

type fakePinnedEngramListService struct {
	pinned []models.EngramSummary
	err    error
	call   pinnedEngramListCall
}

func (service *fakePinnedEngramListService) ListPinnedEngrams(
	_ context.Context,
	request SessionScopedRequest,
) ([]models.EngramSummary, error) {
	service.call = pinnedEngramListCall{
		actorUserID: request.ActorUserID,
		sessionID:   request.SessionID,
	}
	if service.err != nil {
		return nil, service.err
	}
	return service.pinned, nil
}
