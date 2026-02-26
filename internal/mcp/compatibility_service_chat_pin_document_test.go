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

func TestCompatibilityServiceChatPinDocumentParity(t *testing.T) {
	actorUserID := uuid.MustParse("35000000-0000-0000-0000-000000000350")
	sessionID := uuid.MustParse("35000000-0000-0000-0000-000000000351")
	documentID := uuid.MustParse("35000000-0000-0000-0000-000000000352")
	pinService := &fakePinDocumentService{
		pinned: &models.PinnedDocumentRecord{
			SessionID:      sessionID,
			DocumentID:     documentID,
			PinnedByUserID: actorUserID,
			CreatedAt:      time.Unix(1700002000, 0).UTC(),
		},
	}

	testCases := []struct {
		name            string
		request         StreamCallRequest
		asToolsCallPath bool
	}{
		{
			name: "direct canonical",
			request: directToolRequest(
				actorUserID.String(),
				"chat.pin_document",
				map[string]any{"session_id": sessionID.String(), "document_id": documentID.String()},
			),
			asToolsCallPath: false,
		},
		{
			name: "tools canonical",
			request: toolsCallRequest(
				actorUserID.String(),
				"chat_pin_document",
				map[string]any{"session_id": sessionID.String(), "document_id": documentID.String()},
			),
			asToolsCallPath: true,
		},
	}

	for _, testCase := range testCases {
		testCase := testCase
		t.Run(testCase.name, func(t *testing.T) {
			frame := runCompatibilityRequestWithService(
				t,
				newChatPinDocumentCompatibilityService(pinService),
				testCase.request,
			)
			pinned := pinnedDocumentFromFrame(t, frame, testCase.asToolsCallPath)
			if !reflect.DeepEqual(*pinService.pinned, pinned) {
				t.Fatalf("expected pinned payload to match service output")
			}
			if pinService.call.actorUserID != actorUserID {
				t.Fatalf("expected actor user id forwarded")
			}
			if pinService.call.sessionID != sessionID {
				t.Fatalf("expected session id forwarded")
			}
			if pinService.call.documentID != documentID {
				t.Fatalf("expected document id forwarded")
			}
		})
	}
}

func TestCompatibilityServiceChatPinDocumentValidationErrors(t *testing.T) {
	sessionID := uuid.MustParse("35100000-0000-0000-0000-000000000351")
	documentID := uuid.MustParse("35100000-0000-0000-0000-000000000352")

	missingSessionFrame := runCompatibilityRequestWithService(
		t,
		newChatPinDocumentCompatibilityService(&fakePinDocumentService{}),
		toolsCallRequest(
			"35100000-0000-0000-0000-000000000353",
			"chat_pin_document",
			map[string]any{"document_id": documentID.String()},
		),
	)
	requireErrorCode(t, errorPayloadFromFrame(t, missingSessionFrame), -32602)

	invalidSessionFrame := runCompatibilityRequestWithService(
		t,
		newChatPinDocumentCompatibilityService(&fakePinDocumentService{}),
		toolsCallRequest(
			"35100000-0000-0000-0000-000000000354",
			"chat_pin_document",
			map[string]any{"session_id": "bad", "document_id": documentID.String()},
		),
	)
	requireErrorCode(t, errorPayloadFromFrame(t, invalidSessionFrame), -32602)

	missingDocumentFrame := runCompatibilityRequestWithService(
		t,
		newChatPinDocumentCompatibilityService(&fakePinDocumentService{}),
		toolsCallRequest(
			"35100000-0000-0000-0000-000000000355",
			"chat_pin_document",
			map[string]any{"session_id": sessionID.String()},
		),
	)
	requireErrorCode(t, errorPayloadFromFrame(t, missingDocumentFrame), -32602)

	invalidDocumentFrame := runCompatibilityRequestWithService(
		t,
		newChatPinDocumentCompatibilityService(&fakePinDocumentService{}),
		toolsCallRequest(
			"35100000-0000-0000-0000-000000000356",
			"chat_pin_document",
			map[string]any{"session_id": sessionID.String(), "document_id": "bad"},
		),
	)
	requireErrorCode(t, errorPayloadFromFrame(t, invalidDocumentFrame), -32602)
}

func TestCompatibilityServiceChatPinDocumentNotFoundAndServiceError(t *testing.T) {
	sessionID := uuid.MustParse("35200000-0000-0000-0000-000000000352")
	documentID := uuid.MustParse("35200000-0000-0000-0000-000000000353")

	notFoundFrame := runCompatibilityRequestWithService(
		t,
		newChatPinDocumentCompatibilityService(&fakePinDocumentService{}),
		toolsCallRequest(
			"35200000-0000-0000-0000-000000000354",
			"chat_pin_document",
			map[string]any{"session_id": sessionID.String(), "document_id": documentID.String()},
		),
	)
	notFoundPayload := errorPayloadFromFrame(t, notFoundFrame)
	requireErrorCode(t, notFoundPayload, -32602)
	assertPinDocumentNotFoundErrorData(t, notFoundPayload)

	serviceErrorFrame := runCompatibilityRequestWithService(
		t,
		newChatPinDocumentCompatibilityService(&fakePinDocumentService{err: errors.New("boom")}),
		toolsCallRequest(
			"35200000-0000-0000-0000-000000000355",
			"chat_pin_document",
			map[string]any{"session_id": sessionID.String(), "document_id": documentID.String()},
		),
	)
	requireErrorCode(t, errorPayloadFromFrame(t, serviceErrorFrame), -32603)
}

func newChatPinDocumentCompatibilityService(pinService PinDocumentService) Service {
	return NewCompatibilityServiceWithDependencies(
		"1.2.3",
		CompatibilityServiceDependencies{PinDocumentService: pinService},
	)
}

func pinnedDocumentFromFrame(
	t *testing.T,
	frame Frame,
	asToolsCallPath bool,
) models.PinnedDocumentRecord {
	t.Helper()
	result := resultPayloadFromFrame(t, frame)
	payload := result
	if asToolsCallPath {
		payload = mapFromMap(t, result, "structuredContent")
	}
	pinned, ok := payload["pinned"].(models.PinnedDocumentRecord)
	if !ok {
		t.Fatalf("expected pinned payload")
	}
	return pinned
}

func assertPinDocumentNotFoundErrorData(t *testing.T, errorPayload map[string]any) {
	t.Helper()
	data, ok := errorPayload["data"].(map[string]any)
	if !ok {
		t.Fatalf("expected error data payload")
	}
	if data["status_code"] != 404 {
		t.Fatalf("expected status_code 404 in error data")
	}
	if data["detail"] != "Document not found or session inaccessible" {
		t.Fatalf("expected pin-document-not-found detail in error data")
	}
}

type pinDocumentCall struct {
	actorUserID uuid.UUID
	sessionID   uuid.UUID
	documentID  uuid.UUID
}

type fakePinDocumentService struct {
	pinned *models.PinnedDocumentRecord
	err    error
	call   pinDocumentCall
}

func (service *fakePinDocumentService) PinDocument(
	_ context.Context,
	request SessionPinDocumentRequest,
) (*models.PinnedDocumentRecord, error) {
	service.call = pinDocumentCall{
		actorUserID: request.ActorUserID,
		sessionID:   request.SessionID,
		documentID:  request.DocumentID,
	}
	if service.err != nil {
		return nil, service.err
	}
	if service.pinned == nil {
		return nil, nil
	}
	record := *service.pinned
	return &record, nil
}
