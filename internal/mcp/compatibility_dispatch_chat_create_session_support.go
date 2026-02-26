package mcp

import (
	"context"

	"engram/internal/models"
)

func (service *CompatibilityService) dispatchChatCreateSessionTool(
	ctx context.Context,
	actor Actor,
	params map[string]any,
) (map[string]any, bool, *toolDispatchError) {
	if service.sessionCreate == nil {
		return nil, false, nil
	}
	projectID, ok := requiredStringParam(params, "project_id")
	if !ok {
		return nil, true, invalidParamError("project_id")
	}
	title, ok := requiredStringParam(params, "title")
	if !ok {
		return nil, true, invalidParamError("title")
	}
	provider, ok := parseProviderParam(params)
	if !ok {
		return nil, true, invalidParamError("provider")
	}
	visibilityScope, ok := parseVisibilityScopeParam(params)
	if !ok {
		return nil, true, invalidParamError("visibility_scope")
	}
	autosaveStrategy, ok := parseAutosaveStrategyParam(params)
	if !ok {
		return nil, true, invalidParamError("autosave_strategy")
	}
	autosaveEnabled, ok := optionalBoolParam(params, "autosave_enabled", false)
	if !ok {
		return nil, true, invalidParamError("autosave_enabled")
	}
	autosaveIntervalMinutes, ok := optionalIntParam(params, "autosave_interval_minutes", 30)
	if !ok {
		return nil, true, invalidParamError("autosave_interval_minutes")
	}
	autosaveMinMessages, ok := optionalIntParam(params, "autosave_min_messages", 6)
	if !ok {
		return nil, true, invalidParamError("autosave_min_messages")
	}
	retentionDays, ok := optionalIntParam(params, "retention_days", 30)
	if !ok {
		return nil, true, invalidParamError("retention_days")
	}
	retentionMaxSnapshots, ok := optionalIntParam(params, "retention_max_snapshots", 60)
	if !ok {
		return nil, true, invalidParamError("retention_max_snapshots")
	}
	session, err := service.sessionCreate.CreateSession(
		ctx,
		SessionCreateRequest{
			ActorUserID: actor.UserID,
			Payload: models.ChatSessionCreateRequest{
				ProjectID:               projectID,
				Title:                   title,
				Provider:                provider,
				ModelID:                 stringParamWithDefault(params, "model_id", "gpt-4o-mini"),
				SystemPrompt:            stringParamWithDefault(params, "system_prompt", ""),
				VisibilityScope:         visibilityScope,
				AutosaveEnabled:         autosaveEnabled,
				AutosaveStrategy:        autosaveStrategy,
				AutosaveIntervalMinutes: autosaveIntervalMinutes,
				AutosaveMinMessages:     autosaveMinMessages,
				RetentionDays:           retentionDays,
				RetentionMaxSnapshots:   retentionMaxSnapshots,
			},
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
