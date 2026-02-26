package mcp

import (
	"context"

	"engram/internal/models"
)

func (service *CompatibilityService) dispatchChatSaveSessionAsEngramTool(
	ctx context.Context,
	actor Actor,
	params map[string]any,
) (map[string]any, bool, *toolDispatchError) {
	if service.sessionSaveAsEngram == nil {
		return nil, false, nil
	}
	sessionID, ok := requiredUUIDParam(params, "session_id")
	if !ok {
		return nil, true, invalidParamError("session_id")
	}
	visibilityScope, ok := parseSaveSessionVisibilityScopeParam(params)
	if !ok {
		return nil, true, invalidParamError("visibility_scope")
	}
	tags, ok := optionalStringSliceParam(params, "tags")
	if !ok {
		return nil, true, invalidParamError("tags")
	}
	keywords, ok := optionalStringSliceParam(params, "keywords")
	if !ok {
		return nil, true, invalidParamError("keywords")
	}
	saved, err := service.sessionSaveAsEngram.SaveSessionAsEngram(
		ctx,
		SessionSaveAsEngramRequest{
			ActorUserID: actor.UserID,
			SessionID:   sessionID,
			Payload: models.SaveSessionAsEngramRequest{
				Title:           stringParamWithDefault(params, "title", "Session Snapshot"),
				Abstract:        stringParamWithDefault(params, "abstract", ""),
				VisibilityScope: visibilityScope,
				Tags:            tags,
				Keywords:        keywords,
			},
		},
	)
	if err != nil {
		return nil, true, mapChatSendError(err)
	}
	if saved == nil {
		return nil, true, invalidParamsWithStatus(404, "Chat session not found")
	}
	return map[string]any{"saved_engram": *saved}, true, nil
}

func parseSaveSessionVisibilityScopeParam(params map[string]any) (models.VisibilityScope, bool) {
	value := stringParamWithDefault(params, "visibility_scope", string(models.VisibilityScopePrivate))
	scope, err := models.ParseVisibilityScope(value)
	return scope, err == nil
}

func optionalStringSliceParam(params map[string]any, key string) ([]string, bool) {
	value, found := optionalParamValue(params, key)
	if !found {
		return []string{}, true
	}
	items, ok := value.([]any)
	if !ok {
		return nil, false
	}
	result := make([]string, 0, len(items))
	for _, item := range items {
		text, ok := item.(string)
		if !ok {
			return nil, false
		}
		result = append(result, text)
	}
	return result, true
}
