package mcp

import (
	"fmt"
	"strings"

	"engram/internal/models"
)

const (
	engramRefreshConsolidationAdminError = "engram.refresh_consolidation requires admin role"
	engramConsolidationListAdminError    = "engram.consolidation_list requires admin role"
	engramConsolidationActionAdminError  = "engram.consolidation_action requires admin role"
)

func parseEngramConsolidationRefreshRequest(
	actor Actor,
	params map[string]any,
) (EngramConsolidationRefreshRequest, *toolDispatchError) {
	actorRole := normalizedActorRole(actor)
	if actorRole != models.UserRoleAdmin {
		return EngramConsolidationRefreshRequest{}, invalidParamsWithStatus(
			403,
			engramRefreshConsolidationAdminError,
		)
	}
	minGroupSize, ok := optionalMinGroupSizeParam(params, "min_group_size")
	if !ok {
		return EngramConsolidationRefreshRequest{}, invalidParamError("min_group_size")
	}
	return EngramConsolidationRefreshRequest{
		ActorUserID:  actor.UserID,
		ActorRole:    actorRole,
		ProjectID:    optionalProjectIDParam(params, "project_id"),
		MinGroupSize: minGroupSize,
	}, nil
}

func parseEngramConsolidationListRequest(
	actor Actor,
	params map[string]any,
) (EngramConsolidationListRequest, *toolDispatchError) {
	actorRole := normalizedActorRole(actor)
	if actorRole != models.UserRoleAdmin {
		return EngramConsolidationListRequest{}, invalidParamsWithStatus(
			403,
			engramConsolidationListAdminError,
		)
	}
	status, ok := optionalConsolidationStatusParam(params, "status")
	if !ok {
		return EngramConsolidationListRequest{}, invalidParamError("status")
	}
	paging, pagingErr := parsePagingParams(
		params,
		defaultEngramListLimit,
		defaultEngramListOffset,
	)
	if pagingErr != nil {
		return EngramConsolidationListRequest{}, pagingErr
	}
	return EngramConsolidationListRequest{
		ActorUserID: actor.UserID,
		ActorRole:   actorRole,
		ProjectID:   optionalProjectIDParam(params, "project_id"),
		Status:      status,
		Limit:       paging.limit,
		Offset:      paging.offset,
	}, nil
}

func parseEngramConsolidationActionRequest(
	actor Actor,
	params map[string]any,
) (EngramConsolidationActionRequest, *toolDispatchError) {
	actorRole := normalizedActorRole(actor)
	if actorRole != models.UserRoleAdmin {
		return EngramConsolidationActionRequest{}, invalidParamsWithStatus(
			403,
			engramConsolidationActionAdminError,
		)
	}
	suggestionID, ok := requiredUUIDParam(params, "suggestion_id")
	if !ok {
		return EngramConsolidationActionRequest{}, invalidParamError("suggestion_id")
	}
	status, ok := requiredConsolidationActionStatusParam(params, "status")
	if !ok {
		return EngramConsolidationActionRequest{}, invalidParamError("status")
	}
	return EngramConsolidationActionRequest{
		ActorUserID:  actor.UserID,
		ActorRole:    actorRole,
		SuggestionID: suggestionID,
		ProjectID:    optionalProjectIDParam(params, "project_id"),
		Status:       status,
	}, nil
}

func optionalMinGroupSizeParam(params map[string]any, key string) (*int, bool) {
	raw, found := optionalParamValue(params, key)
	if !found {
		return nil, true
	}
	parsed, ok := parseIntValue(raw)
	if !ok || parsed < 2 {
		return nil, false
	}
	return &parsed, true
}

func optionalConsolidationStatusParam(
	params map[string]any,
	key string,
) (*models.ConsolidationSuggestionStatus, bool) {
	raw, found := optionalParamValue(params, key)
	if !found {
		return nil, true
	}
	statusText := strings.TrimSpace(fmt.Sprint(raw))
	if statusText == "" {
		return nil, false
	}
	parsed, err := models.ParseConsolidationSuggestionStatus(statusText)
	if err != nil {
		return nil, false
	}
	return &parsed, true
}

func requiredConsolidationActionStatusParam(
	params map[string]any,
	key string,
) (models.ConsolidationSuggestionStatus, bool) {
	parsed, ok := optionalConsolidationStatusParam(params, key)
	if !ok || parsed == nil {
		return "", false
	}
	if *parsed == models.ConsolidationSuggestionStatusSuggested {
		return "", false
	}
	return *parsed, true
}
