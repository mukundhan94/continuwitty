package mcp

import (
	"context"
	"errors"
	"testing"

	"engram/internal/models"

	"github.com/google/uuid"
)

func TestCompatibilityServiceChatUpdateLifecyclePolicyParity(t *testing.T) {
	actorUserID := uuid.MustParse("39000000-0000-0000-0000-000000000390")
	sessionID := uuid.MustParse("39000000-0000-0000-0000-000000000391")
	service := &fakeLifecyclePolicyUpdateService{
		session: &models.ChatSessionRecord{
			SessionID:               sessionID,
			AutosaveEnabled:         true,
			AutosaveStrategy:        models.ChatAutosaveStrategyInterval,
			AutosaveIntervalMinutes: 45,
			AutosaveMinMessages:     9,
			RetentionDays:           60,
			RetentionMaxSnapshots:   120,
		},
	}
	params := map[string]any{
		"session_id":                sessionID.String(),
		"autosave_enabled":          true,
		"autosave_strategy":         "interval",
		"autosave_interval_minutes": 45.0,
		"autosave_min_messages":     9.0,
		"retention_days":            60.0,
		"retention_max_snapshots":   120.0,
	}

	testCases := []struct {
		name            string
		request         StreamCallRequest
		asToolsCallPath bool
	}{
		{
			name:            "direct",
			request:         directToolRequest(actorUserID.String(), "chat.update_lifecycle_policy", params),
			asToolsCallPath: false,
		},
		{
			name:            "tools call",
			request:         toolsCallRequest(actorUserID.String(), "chat_update_lifecycle_policy", params),
			asToolsCallPath: true,
		},
	}

	for _, testCase := range testCases {
		testCase := testCase
		t.Run(testCase.name, func(t *testing.T) {
			frame := runCompatibilityRequestWithService(
				t,
				newLifecyclePolicyUpdateCompatibilityService(service),
				testCase.request,
			)
			policy := lifecyclePolicyFromFrame(t, frame, testCase.asToolsCallPath)
			if policy["autosave_enabled"] != true {
				t.Fatalf("expected autosave_enabled in policy payload")
			}
			if policy["autosave_strategy"] != models.ChatAutosaveStrategyInterval {
				t.Fatalf("expected autosave_strategy in policy payload")
			}
			if service.call.ActorUserID != actorUserID {
				t.Fatalf("expected actor user id forwarded")
			}
			if service.call.SessionID != sessionID {
				t.Fatalf("expected session id forwarded")
			}
			assertIntPointer(t, service.call.AutosaveIntervalMinutes, 45, "autosave_interval_minutes")
			assertIntPointer(t, service.call.AutosaveMinMessages, 9, "autosave_min_messages")
			assertIntPointer(t, service.call.RetentionDays, 60, "retention_days")
			assertIntPointer(t, service.call.RetentionMaxSnapshots, 120, "retention_max_snapshots")
		})
	}
}

func TestCompatibilityServiceChatUpdateLifecyclePolicyUsesEmptyPayload(t *testing.T) {
	actorUserID := uuid.MustParse("39100000-0000-0000-0000-000000000391")
	sessionID := uuid.MustParse("39100000-0000-0000-0000-000000000392")
	service := &fakeLifecyclePolicyUpdateService{
		session: &models.ChatSessionRecord{
			SessionID:               sessionID,
			AutosaveEnabled:         false,
			AutosaveStrategy:        models.ChatAutosaveStrategyOff,
			AutosaveIntervalMinutes: 30,
			AutosaveMinMessages:     6,
			RetentionDays:           30,
			RetentionMaxSnapshots:   60,
		},
	}
	frame := runCompatibilityRequestWithService(
		t,
		newLifecyclePolicyUpdateCompatibilityService(service),
		directToolRequest(
			actorUserID.String(),
			"chat.update_lifecycle_policy",
			map[string]any{"session_id": sessionID.String()},
		),
	)
	_ = lifecyclePolicyFromFrame(t, frame, false)
	if service.call.AutosaveEnabled != nil || service.call.AutosaveStrategy != nil {
		t.Fatalf("expected empty lifecycle update payload to preserve nil pointers")
	}
}

func TestCompatibilityServiceChatUpdateLifecyclePolicyValidationErrors(t *testing.T) {
	service := newLifecyclePolicyUpdateCompatibilityService(&fakeLifecyclePolicyUpdateService{})
	testCases := []struct {
		name   string
		params map[string]any
	}{
		{name: "missing session id", params: map[string]any{}},
		{name: "invalid session id", params: map[string]any{"session_id": "bad"}},
		{name: "invalid autosave enabled", params: map[string]any{"session_id": uuid.NewString(), "autosave_enabled": "bad"}},
		{name: "invalid autosave strategy", params: map[string]any{"session_id": uuid.NewString(), "autosave_strategy": "bad"}},
		{name: "invalid autosave interval", params: map[string]any{"session_id": uuid.NewString(), "autosave_interval_minutes": "bad"}},
		{name: "invalid autosave min messages", params: map[string]any{"session_id": uuid.NewString(), "autosave_min_messages": "bad"}},
		{name: "invalid retention days", params: map[string]any{"session_id": uuid.NewString(), "retention_days": "bad"}},
		{name: "invalid retention max", params: map[string]any{"session_id": uuid.NewString(), "retention_max_snapshots": "bad"}},
	}

	for _, testCase := range testCases {
		testCase := testCase
		t.Run(testCase.name, func(t *testing.T) {
			frame := runCompatibilityRequestWithService(
				t,
				service,
				toolsCallRequest(
					"39200000-0000-0000-0000-000000000392",
					"chat_update_lifecycle_policy",
					testCase.params,
				),
			)
			requireErrorCode(t, errorPayloadFromFrame(t, frame), -32602)
		})
	}
}

func TestCompatibilityServiceChatUpdateLifecyclePolicyNotFoundAndServiceError(t *testing.T) {
	sessionID := uuid.MustParse("39300000-0000-0000-0000-000000000393")

	notFoundFrame := runCompatibilityRequestWithService(
		t,
		newLifecyclePolicyUpdateCompatibilityService(&fakeLifecyclePolicyUpdateService{}),
		toolsCallRequest(
			"39300000-0000-0000-0000-000000000394",
			"chat_update_lifecycle_policy",
			map[string]any{"session_id": sessionID.String()},
		),
	)
	notFoundPayload := errorPayloadFromFrame(t, notFoundFrame)
	requireErrorCode(t, notFoundPayload, -32602)
	assertNotFoundErrorData(t, notFoundPayload)

	serviceErrorFrame := runCompatibilityRequestWithService(
		t,
		newLifecyclePolicyUpdateCompatibilityService(
			&fakeLifecyclePolicyUpdateService{err: errors.New("boom")},
		),
		toolsCallRequest(
			"39300000-0000-0000-0000-000000000395",
			"chat_update_lifecycle_policy",
			map[string]any{"session_id": sessionID.String()},
		),
	)
	requireErrorCode(t, errorPayloadFromFrame(t, serviceErrorFrame), -32603)
}

func newLifecyclePolicyUpdateCompatibilityService(
	updateService LifecyclePolicyUpdateService,
) Service {
	return NewCompatibilityServiceWithDependencies(
		"1.2.3",
		CompatibilityServiceDependencies{LifecyclePolicyUpdate: updateService},
	)
}

func assertIntPointer(t *testing.T, value *int, expected int, field string) {
	t.Helper()
	if value == nil || *value != expected {
		t.Fatalf("expected %s=%d", field, expected)
	}
}

type fakeLifecyclePolicyUpdateService struct {
	session *models.ChatSessionRecord
	err     error
	call    SessionLifecyclePolicyUpdateRequest
}

func (service *fakeLifecyclePolicyUpdateService) UpdateLifecyclePolicy(
	_ context.Context,
	request SessionLifecyclePolicyUpdateRequest,
) (*models.ChatSessionRecord, error) {
	service.call = request
	if service.err != nil {
		return nil, service.err
	}
	if service.session == nil {
		return nil, nil
	}
	record := *service.session
	return &record, nil
}
