package mcp

import (
	"context"

	"engram/internal/models"
)

type chatCreateSessionCoreFields struct {
	projectID        string
	title            string
	provider         models.ChatProvider
	visibilityScope  models.VisibilityScope
	autosaveStrategy models.ChatAutosaveStrategy
}

type chatCreateSessionAutosaveFields struct {
	autosaveEnabled         bool
	autosaveIntervalMinutes int
	autosaveMinMessages     int
}

type chatCreateSessionRetentionFields struct {
	retentionDays         int
	retentionMaxSnapshots int
}

func (service *CompatibilityService) dispatchChatCreateSessionTool(
	ctx context.Context,
	actor Actor,
	params map[string]any,
) (map[string]any, bool, *toolDispatchError) {
	if service.sessionCreate == nil {
		return nil, false, nil
	}
	payload, dispatchErr := parseChatCreateSessionPayload(params)
	if dispatchErr != nil {
		return nil, true, dispatchErr
	}
	session, err := service.sessionCreate.CreateSession(
		ctx,
		SessionCreateRequest{
			ActorUserID: actor.UserID,
			Payload:     payload,
		},
	)
	if err != nil {
		return nil, true, internalToolDispatchError()
	}
	if session == nil {
		return nil, true, internalToolDispatchError()
	}
	return map[string]any{"session": *session}, true, nil
}

func parseChatCreateSessionPayload(params map[string]any) (models.ChatSessionCreateRequest, *toolDispatchError) {
	core, dispatchErr := parseChatCreateSessionCoreFields(params)
	if dispatchErr != nil {
		return models.ChatSessionCreateRequest{}, dispatchErr
	}
	autosave, dispatchErr := parseChatCreateSessionAutosaveFields(params)
	if dispatchErr != nil {
		return models.ChatSessionCreateRequest{}, dispatchErr
	}
	retention, dispatchErr := parseChatCreateSessionRetentionFields(params)
	if dispatchErr != nil {
		return models.ChatSessionCreateRequest{}, dispatchErr
	}
	return models.ChatSessionCreateRequest{
		ProjectID:               core.projectID,
		Title:                   core.title,
		Provider:                core.provider,
		ModelID:                 stringParamWithDefault(params, "model_id", "gpt-4o-mini"),
		SystemPrompt:            stringParamWithDefault(params, "system_prompt", ""),
		VisibilityScope:         core.visibilityScope,
		AutosaveEnabled:         autosave.autosaveEnabled,
		AutosaveStrategy:        core.autosaveStrategy,
		AutosaveIntervalMinutes: autosave.autosaveIntervalMinutes,
		AutosaveMinMessages:     autosave.autosaveMinMessages,
		RetentionDays:           retention.retentionDays,
		RetentionMaxSnapshots:   retention.retentionMaxSnapshots,
	}, nil
}

func parseChatCreateSessionCoreFields(params map[string]any) (chatCreateSessionCoreFields, *toolDispatchError) {
	projectID, ok := requiredStringParam(params, "project_id")
	if !ok {
		return chatCreateSessionCoreFields{}, invalidParamError("project_id")
	}
	title, ok := requiredStringParam(params, "title")
	if !ok {
		return chatCreateSessionCoreFields{}, invalidParamError("title")
	}
	provider, ok := parseProviderParam(params)
	if !ok {
		return chatCreateSessionCoreFields{}, invalidParamError("provider")
	}
	visibilityScope, ok := parseVisibilityScopeParam(params)
	if !ok {
		return chatCreateSessionCoreFields{}, invalidParamError("visibility_scope")
	}
	autosaveStrategy, ok := parseAutosaveStrategyParam(params)
	if !ok {
		return chatCreateSessionCoreFields{}, invalidParamError("autosave_strategy")
	}
	return chatCreateSessionCoreFields{
		projectID:        projectID,
		title:            title,
		provider:         provider,
		visibilityScope:  visibilityScope,
		autosaveStrategy: autosaveStrategy,
	}, nil
}

func parseChatCreateSessionAutosaveFields(
	params map[string]any,
) (chatCreateSessionAutosaveFields, *toolDispatchError) {
	autosaveEnabled, ok := optionalBoolParam(params, "autosave_enabled", false)
	if !ok {
		return chatCreateSessionAutosaveFields{}, invalidParamError("autosave_enabled")
	}
	autosaveIntervalMinutes, ok := optionalIntParam(params, "autosave_interval_minutes", 30)
	if !ok {
		return chatCreateSessionAutosaveFields{}, invalidParamError("autosave_interval_minutes")
	}
	autosaveMinMessages, ok := optionalIntParam(params, "autosave_min_messages", 6)
	if !ok {
		return chatCreateSessionAutosaveFields{}, invalidParamError("autosave_min_messages")
	}
	return chatCreateSessionAutosaveFields{
		autosaveEnabled:         autosaveEnabled,
		autosaveIntervalMinutes: autosaveIntervalMinutes,
		autosaveMinMessages:     autosaveMinMessages,
	}, nil
}

func parseChatCreateSessionRetentionFields(
	params map[string]any,
) (chatCreateSessionRetentionFields, *toolDispatchError) {
	retentionDays, ok := optionalIntParam(params, "retention_days", 30)
	if !ok {
		return chatCreateSessionRetentionFields{}, invalidParamError("retention_days")
	}
	retentionMaxSnapshots, ok := optionalIntParam(params, "retention_max_snapshots", 60)
	if !ok {
		return chatCreateSessionRetentionFields{}, invalidParamError("retention_max_snapshots")
	}
	return chatCreateSessionRetentionFields{
		retentionDays:         retentionDays,
		retentionMaxSnapshots: retentionMaxSnapshots,
	}, nil
}

func parseProviderParam(params map[string]any) (models.ChatProvider, bool) {
	value := stringParamWithDefault(params, "provider", string(models.ChatProviderOpenAI))
	provider, err := models.ParseChatProvider(value)
	return provider, err == nil
}

func parseVisibilityScopeParam(params map[string]any) (models.VisibilityScope, bool) {
	value := stringParamWithDefault(params, "visibility_scope", string(models.VisibilityScopePrivate))
	scope, err := models.ParseVisibilityScope(value)
	return scope, err == nil
}

func parseAutosaveStrategyParam(params map[string]any) (models.ChatAutosaveStrategy, bool) {
	value := stringParamWithDefault(params, "autosave_strategy", string(models.ChatAutosaveStrategyOff))
	strategy, err := models.ParseChatAutosaveStrategy(value)
	return strategy, err == nil
}
