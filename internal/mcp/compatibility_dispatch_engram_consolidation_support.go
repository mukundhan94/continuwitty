package mcp

import (
	"context"
	"fmt"
	"strings"

	"engram/internal/models"
)

const (
	engramRefreshConsolidationAdminError = "engram.refresh_consolidation requires admin role"
	engramConsolidationListAdminError    = "engram.consolidation_list requires admin role"
)

func (service *CompatibilityService) dispatchEngramRefreshConsolidationTool(
	ctx context.Context,
	actor Actor,
	params map[string]any,
) (map[string]any, bool, *toolDispatchError) {
	if service.engramConsolidationRefresh == nil {
		return nil, false, nil
	}
	request, dispatchErr := parseEngramConsolidationRefreshRequest(actor, params)
	if dispatchErr != nil {
		return nil, true, dispatchErr
	}
	refreshed, err := service.engramConsolidationRefresh.RefreshEngramConsolidationSuggestions(ctx, request)
	if err != nil || refreshed == nil {
		return nil, true, internalToolDispatchError()
	}
	return map[string]any{"consolidation_refresh": *refreshed}, true, nil
}

func (service *CompatibilityService) dispatchEngramConsolidationListTool(
	ctx context.Context,
	actor Actor,
	params map[string]any,
) (map[string]any, bool, *toolDispatchError) {
	if service.engramConsolidationList == nil {
		return nil, false, nil
	}
	request, dispatchErr := parseEngramConsolidationListRequest(actor, params)
	if dispatchErr != nil {
		return nil, true, dispatchErr
	}
	suggestions, err := service.engramConsolidationList.ListEngramConsolidationSuggestions(ctx, request)
	if err != nil {
		return nil, true, internalToolDispatchError()
	}
	return map[string]any{"consolidation_suggestions": suggestions}, true, nil
}

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
