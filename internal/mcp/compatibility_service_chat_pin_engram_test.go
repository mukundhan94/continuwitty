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

func TestCompatibilityServiceChatPinEngramParity(t *testing.T) {
	actorUserID := uuid.MustParse("33000000-0000-0000-0000-000000000330")
	sessionID := uuid.MustParse("33000000-0000-0000-0000-000000000331")
	engramID := uuid.MustParse("33000000-0000-0000-0000-000000000332")
	pinService := &fakePinEngramService{
		pinned: &models.PinnedEngramRecord{
			SessionID:      sessionID,
			EngramID:       engramID,
			PinnedByUserID: actorUserID,
			CreatedAt:      time.Unix(1700001500, 0).UTC(),
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
				"chat.pin_engram",
				map[string]any{"session_id": sessionID.String(), "engram_id": engramID.String()},
			),
			asToolsCallPath: false,
		},
		{
			name: "direct alias",
			request: directToolRequest(
				actorUserID.String(),
				"engram.pin_to_session",
				map[string]any{"session_id": sessionID.String(), "engram_id": engramID.String()},
			),
			asToolsCallPath: false,
		},
		{
			name: "tools alias",
			request: toolsCallRequest(
				actorUserID.String(),
				"engram_pin_to_session",
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
				newChatPinEngramCompatibilityService(pinService),
				testCase.request,
			)
			pinned := pinnedEngramFromFrame(t, frame, testCase.asToolsCallPath)
			if !reflect.DeepEqual(*pinService.pinned, pinned) {
				t.Fatalf("expected pinned payload to match service output")
			}
			if pinService.call.actorUserID != actorUserID {
				t.Fatalf("expected actor user id forwarded")
			}
			if pinService.call.sessionID != sessionID {
				t.Fatalf("expected session id forwarded")
			}
			if pinService.call.engramID != engramID {
				t.Fatalf("expected engram id forwarded")
			}
		})
	}
}

func TestCompatibilityServiceChatPinEngramErrors(t *testing.T) {
	sessionID := uuid.MustParse("33100000-0000-0000-0000-000000000331")
	engramID := uuid.MustParse("33100000-0000-0000-0000-000000000332")

	missingSessionFrame := runCompatibilityRequestWithService(
		t,
		newChatPinEngramCompatibilityService(&fakePinEngramService{}),
		toolsCallRequest(
			"33100000-0000-0000-0000-000000000333",
			"chat_pin_engram",
			map[string]any{"engram_id": engramID.String()},
		),
	)
	missingSessionPayload := errorPayloadFromFrame(t, missingSessionFrame)
	requireErrorCode(t, missingSessionPayload, -32602)

	invalidSessionFrame := runCompatibilityRequestWithService(
		t,
		newChatPinEngramCompatibilityService(&fakePinEngramService{}),
		toolsCallRequest(
			"33100000-0000-0000-0000-000000000334",
			"chat_pin_engram",
			map[string]any{"session_id": "bad", "engram_id": engramID.String()},
		),
	)
	invalidSessionPayload := errorPayloadFromFrame(t, invalidSessionFrame)
	requireErrorCode(t, invalidSessionPayload, -32602)

	missingEngramFrame := runCompatibilityRequestWithService(
		t,
		newChatPinEngramCompatibilityService(&fakePinEngramService{}),
		toolsCallRequest(
			"33100000-0000-0000-0000-000000000335",
			"chat_pin_engram",
			map[string]any{"session_id": sessionID.String()},
		),
	)
	missingEngramPayload := errorPayloadFromFrame(t, missingEngramFrame)
	requireErrorCode(t, missingEngramPayload, -32602)

	invalidEngramFrame := runCompatibilityRequestWithService(
		t,
		newChatPinEngramCompatibilityService(&fakePinEngramService{}),
		toolsCallRequest(
			"33100000-0000-0000-0000-000000000336",
			"chat_pin_engram",
			map[string]any{"session_id": sessionID.String(), "engram_id": "bad"},
		),
	)
	invalidEngramPayload := errorPayloadFromFrame(t, invalidEngramFrame)
	requireErrorCode(t, invalidEngramPayload, -32602)
}

func TestCompatibilityServiceChatPinEngramNotFoundAndServiceError(t *testing.T) {
	sessionID := uuid.MustParse("33200000-0000-0000-0000-000000000332")
	engramID := uuid.MustParse("33200000-0000-0000-0000-000000000333")

	notFoundFrame := runCompatibilityRequestWithService(
		t,
		newChatPinEngramCompatibilityService(&fakePinEngramService{}),
		toolsCallRequest(
			"33200000-0000-0000-0000-000000000334",
			"chat_pin_engram",
			map[string]any{"session_id": sessionID.String(), "engram_id": engramID.String()},
		),
	)
	notFoundPayload := errorPayloadFromFrame(t, notFoundFrame)
	requireErrorCode(t, notFoundPayload, -32602)
	assertPinNotFoundErrorData(t, notFoundPayload)

	serviceErrorFrame := runCompatibilityRequestWithService(
		t,
		newChatPinEngramCompatibilityService(&fakePinEngramService{err: errors.New("boom")}),
		toolsCallRequest(
			"33200000-0000-0000-0000-000000000335",
			"chat_pin_engram",
			map[string]any{"session_id": sessionID.String(), "engram_id": engramID.String()},
		),
	)
	serviceErrorPayload := errorPayloadFromFrame(t, serviceErrorFrame)
	requireErrorCode(t, serviceErrorPayload, -32603)
}

func newChatPinEngramCompatibilityService(pinService PinEngramService) Service {
	return NewCompatibilityServiceWithDependencies(
		"1.2.3",
		CompatibilityServiceDependencies{PinEngramService: pinService},
	)
}

func pinnedEngramFromFrame(
	t *testing.T,
	frame Frame,
	asToolsCallPath bool,
) models.PinnedEngramRecord {
	t.Helper()
	result := resultPayloadFromFrame(t, frame)
	payload := result
	if asToolsCallPath {
		payload = mapFromMap(t, result, "structuredContent")
	}
	pinned, ok := payload["pinned"].(models.PinnedEngramRecord)
	if !ok {
		t.Fatalf("expected pinned payload")
	}
	return pinned
}

func assertPinNotFoundErrorData(t *testing.T, errorPayload map[string]any) {
	t.Helper()
	data, ok := errorPayload["data"].(map[string]any)
	if !ok {
		t.Fatalf("expected error data payload")
	}
	if data["status_code"] != 404 {
		t.Fatalf("expected status_code 404 in error data")
	}
	if data["detail"] != "Engram not found or session inaccessible" {
		t.Fatalf("expected pin-not-found detail in error data")
	}
}

type pinEngramCall struct {
	actorUserID uuid.UUID
	sessionID   uuid.UUID
	engramID    uuid.UUID
}

type fakePinEngramService struct {
	pinned *models.PinnedEngramRecord
	err    error
	call   pinEngramCall
}

func (service *fakePinEngramService) PinEngram(
	_ context.Context,
	request SessionPinEngramRequest,
) (*models.PinnedEngramRecord, error) {
	service.call = pinEngramCall{
		actorUserID: request.ActorUserID,
		sessionID:   request.SessionID,
		engramID:    request.EngramID,
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
