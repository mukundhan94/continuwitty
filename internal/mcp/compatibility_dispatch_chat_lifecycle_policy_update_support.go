package mcp

import (
	"context"

	"engram/internal/models"
)

func (service *CompatibilityService) dispatchChatUpdateLifecyclePolicyTool(
	ctx context.Context,
	actor Actor,
	params map[string]any,
) (map[string]any, bool, *toolDispatchError) {
	if service.lifecyclePolicyUpdate == nil {
		return nil, false, nil
	}
	sessionID, ok := requiredUUIDParam(params, "session_id")
	if !ok {
		return nil, true, invalidParamError("session_id")
	}
	autosaveEnabled, ok := optionalBoolPointerParam(params, "autosave_enabled")
	if !ok {
		return nil, true, invalidParamError("autosave_enabled")
	}
	autosaveStrategy, ok := optionalAutosaveStrategyPointerParam(params, "autosave_strategy")
	if !ok {
		return nil, true, invalidParamError("autosave_strategy")
	}
	autosaveIntervalMinutes, ok := optionalIntPointerParam(params, "autosave_interval_minutes")
	if !ok {
		return nil, true, invalidParamError("autosave_interval_minutes")
	}
	autosaveMinMessages, ok := optionalIntPointerParam(params, "autosave_min_messages")
	if !ok {
		return nil, true, invalidParamError("autosave_min_messages")
	}
	retentionDays, ok := optionalIntPointerParam(params, "retention_days")
	if !ok {
		return nil, true, invalidParamError("retention_days")
	}
	retentionMaxSnapshots, ok := optionalIntPointerParam(params, "retention_max_snapshots")
	if !ok {
		return nil, true, invalidParamError("retention_max_snapshots")
	}
	updated, err := service.lifecyclePolicyUpdate.UpdateLifecyclePolicy(
		ctx,
		SessionLifecyclePolicyUpdateRequest{
			ActorUserID:             actor.UserID,
			SessionID:               sessionID,
			AutosaveEnabled:         autosaveEnabled,
			AutosaveStrategy:        autosaveStrategy,
			AutosaveIntervalMinutes: autosaveIntervalMinutes,
			AutosaveMinMessages:     autosaveMinMessages,
			RetentionDays:           retentionDays,
			RetentionMaxSnapshots:   retentionMaxSnapshots,
		},
	)
	if err != nil {
		return nil, true, internalToolDispatchError()
	}
	if updated == nil {
		return nil, true, invalidParamsWithStatus(404, "Chat session not found")
	}
	return map[string]any{"lifecycle_policy": lifecyclePolicyPayload(*updated)}, true, nil
}

func optionalBoolPointerParam(params map[string]any, key string) (*bool, bool) {
	value, found := optionalParamValue(params, key)
	if !found {
		return nil, true
	}
	typed, ok := value.(bool)
	if !ok {
		return nil, false
	}
	copy := typed
	return &copy, true
}

func optionalAutosaveStrategyPointerParam(
	params map[string]any,
	key string,
) (*models.ChatAutosaveStrategy, bool) {
	value, found := optionalParamValue(params, key)
	if !found {
		return nil, true
	}
	text, ok := value.(string)
	if !ok {
		return nil, false
	}
	parsed, err := models.ParseChatAutosaveStrategy(text)
	if err != nil {
		return nil, false
	}
	copy := parsed
	return &copy, true
}

func optionalIntPointerParam(params map[string]any, key string) (*int, bool) {
	value, found := optionalParamValue(params, key)
	if !found {
		return nil, true
	}
	parsed, ok := parseLooseInt(value)
	if !ok {
		return nil, false
	}
	copy := parsed
	return &copy, true
}

func parseLooseInt(value any) (int, bool) {
	switch typed := value.(type) {
	case int:
		return typed, true
	case float64:
		if typed != float64(int(typed)) {
			return 0, false
		}
		return int(typed), true
	default:
		return 0, false
	}
}
