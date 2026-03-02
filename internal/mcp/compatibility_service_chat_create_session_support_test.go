package mcp

import (
	"context"

	"engram/internal/models"
)

func newSessionCreateCompatibilityService(sessionCreateService SessionCreateService) Service {
	return NewCompatibilityServiceWithDependencies(
		"1.2.3",
		CompatibilityServiceDependencies{SessionCreate: sessionCreateService},
	)
}

func createSessionFullParams() map[string]any {
	return map[string]any{
		"project_id":                "proj-alpha",
		"title":                     "Planning",
		"provider":                  "anthropic",
		"model_id":                  "claude-3-5-sonnet-latest",
		"system_prompt":             "Focus on milestones",
		"visibility_scope":          "project",
		"autosave_enabled":          true,
		"autosave_strategy":         "interval",
		"autosave_interval_minutes": 45.0,
		"autosave_min_messages":     9.0,
		"retention_days":            60.0,
		"retention_max_snapshots":   120.0,
	}
}

func expectedSessionCreatePayload() models.ChatSessionCreateRequest {
	return models.ChatSessionCreateRequest{
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
	}
}

func defaultSessionCreatePayload(projectID string, title string) models.ChatSessionCreateRequest {
	return models.ChatSessionCreateRequest{
		ProjectID:               projectID,
		Title:                   title,
		Provider:                models.ChatProviderOpenAI,
		ModelID:                 "gpt-4o-mini",
		SystemPrompt:            "",
		VisibilityScope:         models.VisibilityScopePrivate,
		AutosaveEnabled:         false,
		AutosaveStrategy:        models.ChatAutosaveStrategyOff,
		AutosaveIntervalMinutes: 30,
		AutosaveMinMessages:     6,
		RetentionDays:           30,
		RetentionMaxSnapshots:   60,
	}
}

type fakeSessionCreateService struct {
	session *models.ChatSessionRecord
	err     error
	call    SessionCreateRequest
}

func (service *fakeSessionCreateService) CreateSession(
	_ context.Context,
	request SessionCreateRequest,
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
