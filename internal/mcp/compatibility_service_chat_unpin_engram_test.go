package mcp

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
)

func TestCompatibilityServiceChatUnpinEngramParity(t *testing.T) {
	actorUserID := uuid.MustParse("34000000-0000-0000-0000-000000000340")
	sessionID := uuid.MustParse("34000000-0000-0000-0000-000000000341")
	engramID := uuid.MustParse("34000000-0000-0000-0000-000000000342")
	unpinService := &fakeUnpinEngramService{removed: true}

	testCases := []struct {
		name            string
		request         StreamCallRequest
		asToolsCallPath bool
	}{
		{
			name: "direct canonical",
			request: directToolRequest(
				actorUserID.String(),
				"chat.unpin_engram",
				map[string]any{"session_id": sessionID.String(), "engram_id": engramID.String()},
			),
			asToolsCallPath: false,
		},
		{
			name: "tools canonical",
			request: toolsCallRequest(
				actorUserID.String(),
				"chat_unpin_engram",
				map[string]any{"session_id": sessionID.String(), "engram_id": engramID.String()},
			),
			asToolsCallPath: true,
		},
	}

	for _, testCase := range testCases {
		testCase := testCase
		t.Run(testCase.name, func(t *testing.T) {
			frame := runCompatibilityRequestWithService(
				t,
				newChatUnpinEngramCompatibilityService(unpinService),
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
			if unpinService.call.engramID != engramID {
				t.Fatalf("expected engram id forwarded")
			}
		})
	}
}

func TestCompatibilityServiceChatUnpinEngramValidationErrors(t *testing.T) {
	sessionID := uuid.MustParse("34100000-0000-0000-0000-000000000341")
	engramID := uuid.MustParse("34100000-0000-0000-0000-000000000342")

	missingSessionFrame := runCompatibilityRequestWithService(
		t,
		newChatUnpinEngramCompatibilityService(&fakeUnpinEngramService{removed: true}),
		toolsCallRequest(
			"34100000-0000-0000-0000-000000000343",
			"chat_unpin_engram",
			map[string]any{"engram_id": engramID.String()},
		),
	)
	requireErrorCode(t, errorPayloadFromFrame(t, missingSessionFrame), -32602)

	invalidSessionFrame := runCompatibilityRequestWithService(
		t,
		newChatUnpinEngramCompatibilityService(&fakeUnpinEngramService{removed: true}),
		toolsCallRequest(
			"34100000-0000-0000-0000-000000000344",
			"chat_unpin_engram",
			map[string]any{"session_id": "bad", "engram_id": engramID.String()},
		),
	)
	requireErrorCode(t, errorPayloadFromFrame(t, invalidSessionFrame), -32602)

	missingEngramFrame := runCompatibilityRequestWithService(
		t,
		newChatUnpinEngramCompatibilityService(&fakeUnpinEngramService{removed: true}),
		toolsCallRequest(
			"34100000-0000-0000-0000-000000000345",
			"chat_unpin_engram",
			map[string]any{"session_id": sessionID.String()},
		),
	)
	requireErrorCode(t, errorPayloadFromFrame(t, missingEngramFrame), -32602)

	invalidEngramFrame := runCompatibilityRequestWithService(
		t,
		newChatUnpinEngramCompatibilityService(&fakeUnpinEngramService{removed: true}),
		toolsCallRequest(
			"34100000-0000-0000-0000-000000000346",
			"chat_unpin_engram",
			map[string]any{"session_id": sessionID.String(), "engram_id": "bad"},
		),
	)
	requireErrorCode(t, errorPayloadFromFrame(t, invalidEngramFrame), -32602)
}

func TestCompatibilityServiceChatUnpinEngramNotFoundAndServiceError(t *testing.T) {
	sessionID := uuid.MustParse("34200000-0000-0000-0000-000000000342")
	engramID := uuid.MustParse("34200000-0000-0000-0000-000000000343")

	notFoundFrame := runCompatibilityRequestWithService(
		t,
		newChatUnpinEngramCompatibilityService(&fakeUnpinEngramService{}),
		toolsCallRequest(
			"34200000-0000-0000-0000-000000000344",
			"chat_unpin_engram",
			map[string]any{"session_id": sessionID.String(), "engram_id": engramID.String()},
		),
	)
	notFoundPayload := errorPayloadFromFrame(t, notFoundFrame)
	requireErrorCode(t, notFoundPayload, -32602)
	assertUnpinNotFoundErrorData(t, notFoundPayload)

	serviceErrorFrame := runCompatibilityRequestWithService(
		t,
		newChatUnpinEngramCompatibilityService(&fakeUnpinEngramService{err: errors.New("boom")}),
		toolsCallRequest(
			"34200000-0000-0000-0000-000000000345",
			"chat_unpin_engram",
			map[string]any{"session_id": sessionID.String(), "engram_id": engramID.String()},
		),
	)
	requireErrorCode(t, errorPayloadFromFrame(t, serviceErrorFrame), -32603)
}

func newChatUnpinEngramCompatibilityService(unpinService UnpinEngramService) Service {
	return NewCompatibilityServiceWithDependencies(
		"1.2.3",
		CompatibilityServiceDependencies{UnpinEngramService: unpinService},
	)
}

func removedFlagFromFrame(t *testing.T, frame Frame, asToolsCallPath bool) bool {
	t.Helper()
	result := resultPayloadFromFrame(t, frame)
	payload := result
	if asToolsCallPath {
		payload = mapFromMap(t, result, "structuredContent")
	}
	removed, ok := payload["removed"].(bool)
	if !ok {
		t.Fatalf("expected removed flag")
	}
	return removed
}

func assertUnpinNotFoundErrorData(t *testing.T, errorPayload map[string]any) {
	t.Helper()
	data, ok := errorPayload["data"].(map[string]any)
	if !ok {
		t.Fatalf("expected error data payload")
	}
	if data["status_code"] != 404 {
		t.Fatalf("expected status_code 404 in error data")
	}
	if data["detail"] != "Pinned engram not found for session" {
		t.Fatalf("expected unpin-not-found detail in error data")
	}
}

type unpinEngramCall struct {
	actorUserID uuid.UUID
	sessionID   uuid.UUID
	engramID    uuid.UUID
}

type fakeUnpinEngramService struct {
	removed bool
	err     error
	call    unpinEngramCall
}

func (service *fakeUnpinEngramService) UnpinEngram(
	_ context.Context,
	request SessionPinEngramRequest,
) (bool, error) {
	service.call = unpinEngramCall{
		actorUserID: request.ActorUserID,
		sessionID:   request.SessionID,
		engramID:    request.EngramID,
	}
	if service.err != nil {
		return false, service.err
	}
	return service.removed, nil
}
