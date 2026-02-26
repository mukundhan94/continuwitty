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

func TestCompatibilityServiceChatListPinnedDocumentsParity(t *testing.T) {
	actorUserID := uuid.MustParse("32000000-0000-0000-0000-000000000320")
	sessionID := uuid.MustParse("32000000-0000-0000-0000-000000000321")
	sessionGetService := &fakeSessionGetService{
		session: &models.ChatSessionRecord{SessionID: sessionID, ProjectID: "proj-alpha"},
	}
	pinnedService := &fakePinnedDocumentListService{
		pinned: []models.PinnedDocumentRecord{
			{
				SessionID:      sessionID,
				DocumentID:     uuid.MustParse("32000000-0000-0000-0000-000000000322"),
				PinnedByUserID: actorUserID,
				CreatedAt:      time.Unix(1700001000, 0).UTC(),
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
				"chat.list_pinned_documents",
				map[string]any{"session_id": sessionID.String()},
			),
			asToolsCallPath: false,
		},
		{
			name: "tools call",
			request: toolsCallRequest(
				actorUserID.String(),
				"chat_list_pinned_documents",
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
				newPinnedDocumentsCompatibilityService(sessionGetService, pinnedService),
				testCase.request,
			)
			pinned := pinnedDocumentsFromFrame(t, frame, testCase.asToolsCallPath)
			if !reflect.DeepEqual(pinnedService.pinned, pinned) {
				t.Fatalf("expected pinned_documents payload to match service output")
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

func TestCompatibilityServiceChatListPinnedDocumentsErrors(t *testing.T) {
	sessionID := uuid.MustParse("32100000-0000-0000-0000-000000000321")

	notFoundFrame := runCompatibilityRequestWithService(
		t,
		newPinnedDocumentsCompatibilityService(
			&fakeSessionGetService{},
			&fakePinnedDocumentListService{},
		),
		toolsCallRequest(
			"32100000-0000-0000-0000-000000000322",
			"chat_list_pinned_documents",
			map[string]any{"session_id": sessionID.String()},
		),
	)
	notFoundPayload := errorPayloadFromFrame(t, notFoundFrame)
	requireErrorCode(t, notFoundPayload, -32602)
	assertNotFoundErrorData(t, notFoundPayload)

	invalidFrame := runCompatibilityRequestWithService(
		t,
		newPinnedDocumentsCompatibilityService(
			&fakeSessionGetService{session: &models.ChatSessionRecord{SessionID: sessionID}},
			&fakePinnedDocumentListService{},
		),
		toolsCallRequest(
			"32100000-0000-0000-0000-000000000323",
			"chat_list_pinned_documents",
			map[string]any{"session_id": "bad"},
		),
	)
	invalidPayload := errorPayloadFromFrame(t, invalidFrame)
	requireErrorCode(t, invalidPayload, -32602)

	missingFrame := runCompatibilityRequestWithService(
		t,
		newPinnedDocumentsCompatibilityService(
			&fakeSessionGetService{session: &models.ChatSessionRecord{SessionID: sessionID}},
			&fakePinnedDocumentListService{},
		),
		toolsCallRequest(
			"32100000-0000-0000-0000-000000000324",
			"chat_list_pinned_documents",
			map[string]any{},
		),
	)
	missingPayload := errorPayloadFromFrame(t, missingFrame)
	requireErrorCode(t, missingPayload, -32602)
}

func TestCompatibilityServiceChatListPinnedDocumentsMapsServiceError(t *testing.T) {
	sessionID := uuid.MustParse("32200000-0000-0000-0000-000000000322")
	frame := runCompatibilityRequestWithService(
		t,
		newPinnedDocumentsCompatibilityService(
			&fakeSessionGetService{session: &models.ChatSessionRecord{SessionID: sessionID}},
			&fakePinnedDocumentListService{err: errors.New("boom")},
		),
		toolsCallRequest(
			"32200000-0000-0000-0000-000000000323",
			"chat_list_pinned_documents",
			map[string]any{"session_id": sessionID.String()},
		),
	)
	errorPayload := errorPayloadFromFrame(t, frame)
	requireErrorCode(t, errorPayload, -32603)
}

func newPinnedDocumentsCompatibilityService(
	sessionGetService SessionGetService,
	pinnedService PinnedDocumentListService,
) Service {
	return NewCompatibilityServiceWithDependencies(
		"1.2.3",
		CompatibilityServiceDependencies{
			SessionGet:            sessionGetService,
			PinnedDocumentService: pinnedService,
		},
	)
}

func pinnedDocumentsFromFrame(
	t *testing.T,
	frame Frame,
	asToolsCallPath bool,
) []models.PinnedDocumentRecord {
	t.Helper()
	result := resultPayloadFromFrame(t, frame)
	payload := result
	if asToolsCallPath {
		payload = mapFromMap(t, result, "structuredContent")
	}
	rawPinned, ok := payload["pinned_documents"]
	if !ok {
		t.Fatalf("expected pinned_documents payload")
	}
	return toPinnedDocumentRecords(t, rawPinned)
}

func toPinnedDocumentRecords(t *testing.T, value any) []models.PinnedDocumentRecord {
	t.Helper()
	switch typed := value.(type) {
	case []models.PinnedDocumentRecord:
		return typed
	case []any:
		records := make([]models.PinnedDocumentRecord, 0, len(typed))
		for _, item := range typed {
			record, ok := item.(models.PinnedDocumentRecord)
			if !ok {
				t.Fatalf("expected PinnedDocumentRecord item, got %T", item)
			}
			records = append(records, record)
		}
		return records
	default:
		t.Fatalf("expected []PinnedDocumentRecord payload, got %T", value)
		return nil
	}
}

type pinnedDocumentListCall struct {
	actorUserID uuid.UUID
	sessionID   uuid.UUID
}

type fakePinnedDocumentListService struct {
	pinned []models.PinnedDocumentRecord
	err    error
	call   pinnedDocumentListCall
}

func (service *fakePinnedDocumentListService) ListPinnedDocuments(
	_ context.Context,
	request SessionScopedRequest,
) ([]models.PinnedDocumentRecord, error) {
	service.call = pinnedDocumentListCall{
		actorUserID: request.ActorUserID,
		sessionID:   request.SessionID,
	}
	if service.err != nil {
		return nil, service.err
	}
	return service.pinned, nil
}
