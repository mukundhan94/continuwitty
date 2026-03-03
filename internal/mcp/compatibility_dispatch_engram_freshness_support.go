package mcp

import (
	"context"

	"engram/internal/models"
)

const engramRefreshFreshnessAdminError = "engram.refresh_freshness requires admin role"

func (service *CompatibilityService) dispatchEngramRefreshFreshnessTool(
	ctx context.Context,
	actor Actor,
	params map[string]any,
) (map[string]any, bool, *toolDispatchError) {
	if service.engramFreshnessRefresh == nil {
		return nil, false, nil
	}
	request, dispatchErr := parseEngramFreshnessRefreshRequest(actor, params)
	if dispatchErr != nil {
		return nil, true, dispatchErr
	}
	refreshed, err := service.engramFreshnessRefresh.RefreshEngramFreshness(ctx, request)
	if err != nil {
		return nil, true, internalToolDispatchError()
	}
	if refreshed == nil {
		return nil, true, internalToolDispatchError()
	}
	return map[string]any{"freshness_refresh": *refreshed}, true, nil
}

func parseEngramFreshnessRefreshRequest(
	actor Actor,
	params map[string]any,
) (EngramFreshnessRefreshRequest, *toolDispatchError) {
	actorRole := normalizedActorRole(actor)
	if actorRole != models.UserRoleAdmin {
		return EngramFreshnessRefreshRequest{}, invalidParamsWithStatus(
			403,
			engramRefreshFreshnessAdminError,
		)
	}
	projectID := optionalProjectIDParam(params, "project_id")
	halfLifeDays, ok := optionalPositiveFloatParam(params, "half_life_days")
	if !ok {
		return EngramFreshnessRefreshRequest{}, invalidParamError("half_life_days")
	}
	return EngramFreshnessRefreshRequest{
		ActorUserID:  actor.UserID,
		ActorRole:    actorRole,
		ProjectID:    projectID,
		HalfLifeDays: halfLifeDays,
	}, nil
}

func optionalPositiveFloatParam(params map[string]any, key string) (*float64, bool) {
	raw, found := optionalParamValue(params, key)
	if !found {
		return nil, true
	}
	parsed, ok := parseFloatValue(raw)
	if !ok || parsed <= 0 {
		return nil, false
	}
	return &parsed, true
}
