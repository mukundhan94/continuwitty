package mcp

import (
	"errors"
	"reflect"
	"testing"
	"time"

	"engram/internal/models"

	"github.com/google/uuid"
)

func TestCompatibilityServiceChatCreateSessionParity(t *testing.T) {
	actorUserID := uuid.MustParse("38000000-0000-0000-0000-000000000380")
	created := &models.ChatSessionRecord{
		SessionID:               uuid.MustParse("38000000-0000-0000-0000-000000000381"),
		OwnerUserID:             actorUserID,
		ProjectID:               "proj-alpha",
		Title:                   "Planning",
		Provider:                models.ChatProviderAnthropic,
		ModelID:                 "claude-3-5-sonnet-latest",
		SystemPrompt:            "Focus on milestones",
		VisibilityScope:         models.VisibilityScopeProject,
		AutosaveEnabled:         true,
		AutosaveStrategy:        models.ChatAutosaveStrategyInterval,
		AutosaveIntervalMinutes: 45,
		AutosaveMinMessages:     9,
		RetentionDays:           60,
		RetentionMaxSnapshots:   120,
		CreatedAt:               time.Unix(1700002600, 0).UTC(),
		UpdatedAt:               time.Unix(1700002600, 0).UTC(),
	}
	service := &fakeSessionCreateService{session: created}
	params := createSessionFullParams()

	testCases := []struct {
		name            string
		request         StreamCallRequest
		asToolsCallPath bool
	}{
		{
			name: "direct",
			request: directToolRequest(
				actorUserID.String(),
				"chat.create_session",
				params,
			),
			asToolsCallPath: false,
		},
		{
			name: "tools call",
			request: toolsCallRequest(
				actorUserID.String(),
				"chat_create_session",
				params,
			),
			asToolsCallPath: true,
		},
	}

	for _, testCase := range testCases {
		testCase := testCase
		t.Run(testCase.name, func(t *testing.T) {
			frame := runCompatibilityRequestWithService(
				t,
				newSessionCreateCompatibilityService(service),
				testCase.request,
			)
			session := chatSessionFromFrame(t, frame, testCase.asToolsCallPath)
			if !reflect.DeepEqual(*created, session) {
				t.Fatalf("expected created session payload")
			}
			if service.call.ActorUserID != actorUserID {
				t.Fatalf("expected actor user id forwarded")
			}
			expectedPayload := expectedSessionCreatePayload()
			if !reflect.DeepEqual(expectedPayload, service.call.Payload) {
				t.Fatalf("expected create payload to be forwarded")
			}
		})
	}
}

func TestCompatibilityServiceChatCreateSessionUsesDefaults(t *testing.T) {
	actorUserID := uuid.MustParse("38100000-0000-0000-0000-000000000381")
	service := &fakeSessionCreateService{
		session: &models.ChatSessionRecord{
			SessionID:   uuid.MustParse("38100000-0000-0000-0000-000000000382"),
			OwnerUserID: actorUserID,
			ProjectID:   "proj-default",
			Title:       "Defaulted",
		},
	}
	frame := runCompatibilityRequestWithService(
		t,
		newSessionCreateCompatibilityService(service),
		directToolRequest(
			actorUserID.String(),
			"chat.create_session",
			map[string]any{"project_id": "proj-default", "title": "Defaulted"},
		),
	)
	_ = chatSessionFromFrame(t, frame, false)

	expected := defaultSessionCreatePayload("proj-default", "Defaulted")
	if !reflect.DeepEqual(expected, service.call.Payload) {
		t.Fatalf("expected defaulted create payload")
	}
}

func TestCompatibilityServiceChatCreateSessionValidationErrors(t *testing.T) {
	service := newSessionCreateCompatibilityService(&fakeSessionCreateService{})

	testCases := []struct {
		name   string
		params map[string]any
	}{
		{name: "missing project", params: map[string]any{"title": "x"}},
		{name: "missing title", params: map[string]any{"project_id": "proj"}},
		{name: "invalid provider", params: map[string]any{"project_id": "proj", "title": "x", "provider": "bad"}},
		{name: "invalid visibility", params: map[string]any{"project_id": "proj", "title": "x", "visibility_scope": "bad"}},
		{name: "invalid autosave strategy", params: map[string]any{"project_id": "proj", "title": "x", "autosave_strategy": "bad"}},
		{name: "invalid autosave enabled", params: map[string]any{"project_id": "proj", "title": "x", "autosave_enabled": "bad"}},
		{name: "invalid autosave interval", params: map[string]any{"project_id": "proj", "title": "x", "autosave_interval_minutes": "bad"}},
		{name: "invalid autosave min messages", params: map[string]any{"project_id": "proj", "title": "x", "autosave_min_messages": "bad"}},
		{name: "invalid retention days", params: map[string]any{"project_id": "proj", "title": "x", "retention_days": "bad"}},
		{name: "invalid retention max", params: map[string]any{"project_id": "proj", "title": "x", "retention_max_snapshots": "bad"}},
	}

	for _, testCase := range testCases {
		testCase := testCase
		t.Run(testCase.name, func(t *testing.T) {
			frame := runCompatibilityRequestWithService(
				t,
				service,
				toolsCallRequest(
					"38200000-0000-0000-0000-000000000382",
					"chat_create_session",
					testCase.params,
				),
			)
			requireErrorCode(t, errorPayloadFromFrame(t, frame), -32602)
		})
	}
}

func TestCompatibilityServiceChatCreateSessionServiceError(t *testing.T) {
	frame := runCompatibilityRequestWithService(
		t,
		newSessionCreateCompatibilityService(&fakeSessionCreateService{err: errors.New("boom")}),
		toolsCallRequest(
			"38300000-0000-0000-0000-000000000383",
			"chat_create_session",
			map[string]any{"project_id": "proj", "title": "x"},
		),
	)
	requireErrorCode(t, errorPayloadFromFrame(t, frame), -32603)
}
