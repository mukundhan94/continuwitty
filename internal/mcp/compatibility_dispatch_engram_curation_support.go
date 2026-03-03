package mcp

import (
	"fmt"
	"strings"

	"engram/internal/models"
)

const (
	engramCurationListAdminError   = "engram.curation_list requires admin role"
	engramCurationActionAdminError = "engram.curation_action requires admin role"
)

func parseEngramCurationListRequest(
	actor Actor,
	params map[string]any,
) (EngramCurationListRequest, *toolDispatchError) {
	actorRole := normalizedActorRole(actor)
	if actorRole != models.UserRoleAdmin {
		return EngramCurationListRequest{}, invalidParamsWithStatus(403, engramCurationListAdminError)
	}
	sessionID, ok := optionalUUIDParam(params, "session_id")
	if !ok {
		return EngramCurationListRequest{}, invalidParamError("session_id")
	}
	suggestionType, ok := optionalMemoryCurationSuggestionTypeParam(params, "suggestion_type")
	if !ok {
		return EngramCurationListRequest{}, invalidParamError("suggestion_type")
	}
	status, ok := optionalMemoryCurationSuggestionStatusParam(params, "status")
	if !ok {
		return EngramCurationListRequest{}, invalidParamError("status")
	}
	paging, pagingErr := parsePagingParams(
		params,
		defaultEngramListLimit,
		defaultEngramListOffset,
	)
	if pagingErr != nil {
		return EngramCurationListRequest{}, pagingErr
	}
	return EngramCurationListRequest{
		ActorUserID:    actor.UserID,
		ActorRole:      actorRole,
		ProjectID:      optionalProjectIDParam(params, "project_id"),
		SessionID:      sessionID,
		SuggestionType: suggestionType,
		Status:         status,
		Limit:          paging.limit,
		Offset:         paging.offset,
	}, nil
}

func parseEngramCurationActionRequest(
	actor Actor,
	params map[string]any,
) (EngramCurationActionRequest, *toolDispatchError) {
	actorRole := normalizedActorRole(actor)
	if actorRole != models.UserRoleAdmin {
		return EngramCurationActionRequest{}, invalidParamsWithStatus(
			403,
			engramCurationActionAdminError,
		)
	}
	suggestionID, ok := requiredUUIDParam(params, "suggestion_id")
	if !ok {
		return EngramCurationActionRequest{}, invalidParamError("suggestion_id")
	}
	status, ok := requiredMemoryCurationActionStatusParam(params, "status")
	if !ok {
		return EngramCurationActionRequest{}, invalidParamError("status")
	}
	return EngramCurationActionRequest{
		ActorUserID:  actor.UserID,
		ActorRole:    actorRole,
		SuggestionID: suggestionID,
		ProjectID:    optionalProjectIDParam(params, "project_id"),
		Status:       status,
	}, nil
}

func optionalMemoryCurationSuggestionTypeParam(
	params map[string]any,
	key string,
) (*models.MemoryCurationSuggestionType, bool) {
	raw, found := optionalParamValue(params, key)
	if !found {
		return nil, true
	}
	value := strings.TrimSpace(fmt.Sprint(raw))
	if value == "" {
		return nil, false
	}
	parsed, err := models.ParseMemoryCurationSuggestionType(value)
	if err != nil {
		return nil, false
	}
	return &parsed, true
}

func optionalMemoryCurationSuggestionStatusParam(
	params map[string]any,
	key string,
) (*models.MemoryCurationSuggestionStatus, bool) {
	raw, found := optionalParamValue(params, key)
	if !found {
		return nil, true
	}
	value := strings.TrimSpace(fmt.Sprint(raw))
	if value == "" {
		return nil, false
	}
	parsed, err := models.ParseMemoryCurationSuggestionStatus(value)
	if err != nil {
		return nil, false
	}
	return &parsed, true
}

func requiredMemoryCurationActionStatusParam(
	params map[string]any,
	key string,
) (models.MemoryCurationSuggestionStatus, bool) {
	status, ok := optionalMemoryCurationSuggestionStatusParam(params, key)
	if !ok || status == nil {
		return "", false
	}
	if *status == models.MemoryCurationSuggestionStatusSuggested {
		return "", false
	}
	return *status, true
}
