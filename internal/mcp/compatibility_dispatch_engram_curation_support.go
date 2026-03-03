package mcp

import (
	"fmt"
	"strings"

	"engram/internal/models"
)

const (
	engramCurationListAdminError    = "engram.curation_list requires admin role"
	engramCurationRefreshAdminError = "engram.curation_refresh_links requires admin role"
	engramCurationActionAdminError  = "engram.curation_action requires admin role"
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
	suggestionType, ok := optionalParsedValue(
		params,
		"suggestion_type",
		models.ParseMemoryCurationSuggestionType,
	)
	if !ok {
		return EngramCurationListRequest{}, invalidParamError("suggestion_type")
	}
	status, ok := optionalParsedValue(
		params,
		"status",
		models.ParseMemoryCurationSuggestionStatus,
	)
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

func parseEngramCurationRefreshRequest(
	actor Actor,
	params map[string]any,
) (EngramCurationRefreshRequest, *toolDispatchError) {
	actorRole := normalizedActorRole(actor)
	if actorRole != models.UserRoleAdmin {
		return EngramCurationRefreshRequest{}, invalidParamsWithStatus(
			403,
			engramCurationRefreshAdminError,
		)
	}
	sourceEngramID, ok := requiredUUIDParam(params, "source_engram_id")
	if !ok {
		return EngramCurationRefreshRequest{}, invalidParamError("source_engram_id")
	}
	includeArchived, ok := optionalBoolParam(params, "include_archived", false)
	if !ok {
		return EngramCurationRefreshRequest{}, invalidParamError("include_archived")
	}
	limit, ok := optionalIntParam(params, "limit", 0)
	if !ok {
		return EngramCurationRefreshRequest{}, invalidParamError("limit")
	}
	staleAfterDays, ok := optionalIntParam(params, "stale_after_days", 0)
	if !ok {
		return EngramCurationRefreshRequest{}, invalidParamError("stale_after_days")
	}
	lowValueThreshold, ok := optionalCurationFloatParam(params, "low_value_threshold", 0)
	if !ok {
		return EngramCurationRefreshRequest{}, invalidParamError("low_value_threshold")
	}
	return EngramCurationRefreshRequest{
		ActorUserID:       actor.UserID,
		ActorRole:         actorRole,
		ProjectID:         optionalProjectIDParam(params, "project_id"),
		SourceEngramID:    sourceEngramID,
		IncludeArchived:   includeArchived,
		Limit:             limit,
		StaleAfterDays:    staleAfterDays,
		LowValueThreshold: lowValueThreshold,
	}, nil
}

func optionalParsedValue[T any](
	params map[string]any,
	key string,
	parse func(string) (T, error),
) (*T, bool) {
	raw, found := optionalParamValue(params, key)
	if !found {
		return nil, true
	}
	value := strings.TrimSpace(fmt.Sprint(raw))
	if value == "" {
		return nil, false
	}
	parsed, err := parse(value)
	if err != nil {
		return nil, false
	}
	return &parsed, true
}

func requiredMemoryCurationActionStatusParam(
	params map[string]any,
	key string,
) (models.MemoryCurationSuggestionStatus, bool) {
	status, ok := optionalParsedValue(
		params,
		key,
		models.ParseMemoryCurationSuggestionStatus,
	)
	if !ok || status == nil {
		return "", false
	}
	if *status == models.MemoryCurationSuggestionStatusSuggested {
		return "", false
	}
	return *status, true
}

func optionalCurationFloatParam(
	params map[string]any,
	key string,
	defaultValue float64,
) (float64, bool) {
	raw, found := optionalParamValue(params, key)
	if !found {
		return defaultValue, true
	}
	parsed, ok := parseFloatValue(raw)
	if !ok {
		return 0, false
	}
	return parsed, true
}
