package mcp

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
)

func TestCompatibilityServiceChatUnpinDocumentParity(t *testing.T) {
	actorUserID := uuid.MustParse("36000000-0000-0000-0000-000000000360")
	sessionID := uuid.MustParse("36000000-0000-0000-0000-000000000361")
	documentID := uuid.MustParse("36000000-0000-0000-0000-000000000362")
	unpinService := &fakeUnpinDocumentService{removed: true}

	testCases := []struct {
		name            string
		request         StreamCallRequest
		asToolsCallPath bool
	}{
		{
			name: "direct canonical",
			request: directToolRequest(
				actorUserID.String(),
				"chat.unpin_document",
				map[string]any{"session_id": sessionID.String(), "document_id": documentID.String()},
			),
			asToolsCallPath: false,
		},
		{
			name: "tools canonical",
			request: toolsCallRequest(
				actorUserID.String(),
				"chat_unpin_document",
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
				newChatUnpinDocumentCompatibilityService(unpinService),
				testCase.request,
			)
			if !removedFlagFromFrame(t, frame, testCase.asToolsCallPath) {
				t.Fatalf("expected removed flag in response")
			}
			if unpinService.call.actorUserID != actorUserID {
				t.Fatalf("expected actor user id forwarded")
			}
			if unpinService.call.sessionID != sessionID {
				t.Fatalf("expected session id forwarded")
			}
			if unpinService.call.documentID != documentID {
				t.Fatalf("expected document id forwarded")
			}
		})
	}
}

func TestCompatibilityServiceChatUnpinDocumentValidationErrors(t *testing.T) {
	sessionID := uuid.MustParse("36100000-0000-0000-0000-000000000361")
	documentID := uuid.MustParse("36100000-0000-0000-0000-000000000362")

	missingSessionFrame := runCompatibilityRequestWithService(
		t,
		newChatUnpinDocumentCompatibilityService(&fakeUnpinDocumentService{removed: true}),
		toolsCallRequest(
			"36100000-0000-0000-0000-000000000363",
			"chat_unpin_document",
			map[string]any{"document_id": documentID.String()},
		),
	)
	requireErrorCode(t, errorPayloadFromFrame(t, missingSessionFrame), -32602)

	invalidSessionFrame := runCompatibilityRequestWithService(
		t,
		newChatUnpinDocumentCompatibilityService(&fakeUnpinDocumentService{removed: true}),
		toolsCallRequest(
			"36100000-0000-0000-0000-000000000364",
			"chat_unpin_document",
			map[string]any{"session_id": "bad", "document_id": documentID.String()},
		),
	)
	requireErrorCode(t, errorPayloadFromFrame(t, invalidSessionFrame), -32602)

	missingDocumentFrame := runCompatibilityRequestWithService(
		t,
		newChatUnpinDocumentCompatibilityService(&fakeUnpinDocumentService{removed: true}),
		toolsCallRequest(
			"36100000-0000-0000-0000-000000000365",
			"chat_unpin_document",
			map[string]any{"session_id": sessionID.String()},
		),
	)
	requireErrorCode(t, errorPayloadFromFrame(t, missingDocumentFrame), -32602)

	invalidDocumentFrame := runCompatibilityRequestWithService(
		t,
		newChatUnpinDocumentCompatibilityService(&fakeUnpinDocumentService{removed: true}),
		toolsCallRequest(
			"36100000-0000-0000-0000-000000000366",
			"chat_unpin_document",
			map[string]any{"session_id": sessionID.String(), "document_id": "bad"},
		),
	)
	requireErrorCode(t, errorPayloadFromFrame(t, invalidDocumentFrame), -32602)
}

func TestCompatibilityServiceChatUnpinDocumentNotFoundAndServiceError(t *testing.T) {
	sessionID := uuid.MustParse("36200000-0000-0000-0000-000000000362")
	documentID := uuid.MustParse("36200000-0000-0000-0000-000000000363")

	notFoundFrame := runCompatibilityRequestWithService(
		t,
		newChatUnpinDocumentCompatibilityService(&fakeUnpinDocumentService{}),
		toolsCallRequest(
			"36200000-0000-0000-0000-000000000364",
			"chat_unpin_document",
			map[string]any{"session_id": sessionID.String(), "document_id": documentID.String()},
		),
	)
	notFoundPayload := errorPayloadFromFrame(t, notFoundFrame)
	requireErrorCode(t, notFoundPayload, -32602)
	assertUnpinDocumentNotFoundErrorData(t, notFoundPayload)

	serviceErrorFrame := runCompatibilityRequestWithService(
		t,
		newChatUnpinDocumentCompatibilityService(&fakeUnpinDocumentService{err: errors.New("boom")}),
		toolsCallRequest(
			"36200000-0000-0000-0000-000000000365",
			"chat_unpin_document",
			map[string]any{"session_id": sessionID.String(), "document_id": documentID.String()},
		),
	)
	requireErrorCode(t, errorPayloadFromFrame(t, serviceErrorFrame), -32603)
}

func newChatUnpinDocumentCompatibilityService(unpinService UnpinDocumentService) Service {
	return NewCompatibilityServiceWithDependencies(
		"1.2.3",
		CompatibilityServiceDependencies{UnpinDocumentService: unpinService},
	)
}

func assertUnpinDocumentNotFoundErrorData(t *testing.T, errorPayload map[string]any) {
	t.Helper()
	data, ok := errorPayload["data"].(map[string]any)
	if !ok {
		t.Fatalf("expected error data payload")
	}
	if data["status_code"] != 404 {
		t.Fatalf("expected status_code 404 in error data")
	}
	if data["detail"] != "Pinned document not found for session" {
		t.Fatalf("expected unpin-document-not-found detail in error data")
	}
}

type unpinDocumentCall struct {
	actorUserID uuid.UUID
	sessionID   uuid.UUID
	documentID  uuid.UUID
}

type fakeUnpinDocumentService struct {
	removed bool
	err     error
	call    unpinDocumentCall
}

func (service *fakeUnpinDocumentService) UnpinDocument(
	_ context.Context,
	request SessionPinDocumentRequest,
) (bool, error) {
	service.call = unpinDocumentCall{
		actorUserID: request.ActorUserID,
		sessionID:   request.SessionID,
		documentID:  request.DocumentID,
	}
	if service.err != nil {
		return false, service.err
	}
	return service.removed, nil
}
