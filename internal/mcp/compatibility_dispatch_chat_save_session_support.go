package mcp

import (
	"context"
	"strings"

	"engram/internal/models"
)

func (service *CompatibilityService) dispatchChatSaveSessionAsEngramTool(
	ctx context.Context,
	actor Actor,
	params map[string]any,
) (map[string]any, bool, *toolDispatchError) {
	if saveAsEngramSessionRequested(params) {
		return service.dispatchChatSaveSessionAsEngramFromSession(ctx, actor, params)
	}
	return service.dispatchChatSaveSessionAsEngramFromConversation(ctx, actor, params)
}

func saveAsEngramSessionRequested(params map[string]any) bool {
	sessionID, found := optionalParamValue(params, "session_id")
	if !found {
		return false
	}
	return strings.TrimSpace(stringParam(sessionID)) != ""
}

func (service *CompatibilityService) dispatchChatSaveSessionAsEngramFromSession(
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

func (service *CompatibilityService) dispatchChatSaveSessionAsEngramFromConversation(
	ctx context.Context,
	actor Actor,
	params map[string]any,
) (map[string]any, bool, *toolDispatchError) {
	if service.engramCreateConversation == nil {
		return nil, false, nil
	}
	request, dispatchErr := parseChatSaveAsEngramConversationRequest(actor, params)
	if dispatchErr != nil {
		return nil, true, dispatchErr
	}
	created, err := service.engramCreateConversation.CreateEngramFromConversation(ctx, request)
	if err != nil {
		return nil, true, mapEngramCreateError(err)
	}
	if created == nil {
		return nil, true, internalToolDispatchError()
	}
	return map[string]any{
		"saved_engram":      created.Engram,
		"enrichment_report": created.EnrichmentReport,
	}, true, nil
}

func parseChatSaveAsEngramConversationRequest(
	actor Actor,
	params map[string]any,
) (EngramCreateFromConversationRequest, *toolDispatchError) {
	if strings.TrimSpace(stringParamWithDefault(params, "conversation_markdown", "")) == "" {
		return EngramCreateFromConversationRequest{}, missingParamError("conversation_markdown")
	}
	fallbackParams := cloneToolParams(params)
	if strings.TrimSpace(stringParamWithDefault(fallbackParams, "title", "")) == "" {
		fallbackParams["title"] = "Conversation Snapshot"
	}
	request, dispatchErr := parseEngramCreateFromConversationRequest(actor, fallbackParams)
	if dispatchErr != nil {
		return EngramCreateFromConversationRequest{}, dispatchErr
	}
	request.EnrichmentOrigin = "mcp.chat.save_as_engram"
	return request, nil
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
