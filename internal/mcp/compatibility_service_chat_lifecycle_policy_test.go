package mcp

import (
	"errors"
	"testing"

	"engram/internal/models"

	"github.com/google/uuid"
)

func TestCompatibilityServiceChatGetLifecyclePolicyParity(t *testing.T) {
	actorUserID := uuid.MustParse("28000000-0000-0000-0000-000000000280")
	sessionID := uuid.MustParse("28000000-0000-0000-0000-000000000281")
	service := &fakeSessionGetService{
		session: &models.ChatSessionRecord{
			SessionID:               sessionID,
			AutosaveEnabled:         true,
			AutosaveStrategy:        models.ChatAutosaveStrategyMessageCount,
			AutosaveIntervalMinutes: 30,
			AutosaveMinMessages:     12,
			RetentionDays:           14,
			RetentionMaxSnapshots:   40,
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
				"chat.get_lifecycle_policy",
				map[string]any{"session_id": sessionID.String()},
			),
			asToolsCallPath: false,
		},
		{
			name: "tools call",
			request: toolsCallRequest(
				actorUserID.String(),
				"chat_get_lifecycle_policy",
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
				newSessionGetCompatibilityService(service),
				testCase.request,
			)
			policy := lifecyclePolicyFromFrame(t, frame, testCase.asToolsCallPath)
			assertLifecyclePolicyValues(t, policy)
			assertLifecyclePolicyRequestCall(t, service.call, actorUserID, sessionID)
		})
	}
}

func TestCompatibilityServiceChatGetLifecyclePolicyErrors(t *testing.T) {
	notFoundFrame := runCompatibilityRequestWithService(
		t,
		newSessionGetCompatibilityService(&fakeSessionGetService{}),
		toolsCallRequest(
			"28100000-0000-0000-0000-000000000281",
			"chat_get_lifecycle_policy",
			map[string]any{"session_id": "28100000-0000-0000-0000-000000000282"},
		),
	)
	notFoundPayload := errorPayloadFromFrame(t, notFoundFrame)
	requireErrorCode(t, notFoundPayload, -32602)
	assertNotFoundErrorData(t, notFoundPayload)

	invalidFrame := runCompatibilityRequestWithService(
		t,
		newSessionGetCompatibilityService(&fakeSessionGetService{}),
		toolsCallRequest(
			"28200000-0000-0000-0000-000000000282",
			"chat_get_lifecycle_policy",
			map[string]any{"session_id": "bad"},
		),
	)
	invalidPayload := errorPayloadFromFrame(t, invalidFrame)
	requireErrorCode(t, invalidPayload, -32602)

	missingFrame := runCompatibilityRequestWithService(
		t,
		newSessionGetCompatibilityService(&fakeSessionGetService{}),
		toolsCallRequest(
			"28300000-0000-0000-0000-000000000283",
			"chat_get_lifecycle_policy",
			map[string]any{},
		),
	)
	missingPayload := errorPayloadFromFrame(t, missingFrame)
	requireErrorCode(t, missingPayload, -32602)
}

func TestCompatibilityServiceChatGetLifecyclePolicyMapsServiceError(t *testing.T) {
	frame := runCompatibilityRequestWithService(
		t,
		newSessionGetCompatibilityService(&fakeSessionGetService{err: errors.New("boom")}),
		toolsCallRequest(
			"28400000-0000-0000-0000-000000000284",
			"chat_get_lifecycle_policy",
			map[string]any{"session_id": "28400000-0000-0000-0000-000000000285"},
		),
	)
	errorPayload := errorPayloadFromFrame(t, frame)
	requireErrorCode(t, errorPayload, -32603)
}

func lifecyclePolicyFromFrame(
	t *testing.T,
	frame Frame,
	asToolsCallPath bool,
) map[string]any {
	t.Helper()
	result := resultPayloadFromFrame(t, frame)
	payload := result
	if asToolsCallPath {
		payload = mapFromMap(t, result, "structuredContent")
	}
	policy, ok := payload["lifecycle_policy"].(map[string]any)
	if !ok {
		t.Fatalf("expected lifecycle_policy payload")
	}
	return policy
}

func assertLifecyclePolicyValues(t *testing.T, policy map[string]any) {
	t.Helper()
	expected := map[string]any{
		"autosave_enabled":          true,
		"autosave_strategy":         models.ChatAutosaveStrategyMessageCount,
		"autosave_interval_minutes": 30,
		"autosave_min_messages":     12,
		"retention_days":            14,
		"retention_max_snapshots":   40,
	}
	for key, value := range expected {
		if policy[key] != value {
			t.Fatalf("expected %s to be forwarded", key)
		}
	}
}

func assertLifecyclePolicyRequestCall(
	t *testing.T,
	call sessionGetCall,
	expectedActorUserID uuid.UUID,
	expectedSessionID uuid.UUID,
) {
	t.Helper()
	if call.actorUserID != expectedActorUserID {
		t.Fatalf("expected actor user id forwarded")
	}
	if call.sessionID != expectedSessionID {
		t.Fatalf("expected session id forwarded")
	}
}
