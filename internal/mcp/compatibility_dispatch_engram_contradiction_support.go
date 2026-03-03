package mcp

import (
	"fmt"
	"strings"

	"engram/internal/models"
)

const (
	engramRefreshContradictionsAdminError = "engram.refresh_contradictions requires admin role"
	engramContradictionListAdminError     = "engram.contradiction_list requires admin role"
	engramContradictionResolveAdminError  = "engram.contradiction_resolve requires admin role"
)

func parseEngramContradictionRefreshRequest(
	actor Actor,
	params map[string]any,
) (EngramContradictionRefreshRequest, *toolDispatchError) {
	actorRole := normalizedActorRole(actor)
	if actorRole != models.UserRoleAdmin {
		return EngramContradictionRefreshRequest{}, invalidParamsWithStatus(
			403,
			engramRefreshContradictionsAdminError,
		)
	}
	return EngramContradictionRefreshRequest{
		ActorUserID: actor.UserID,
		ActorRole:   actorRole,
		ProjectID:   optionalProjectIDParam(params, "project_id"),
	}, nil
}

func parseEngramContradictionListRequest(
	actor Actor,
	params map[string]any,
) (EngramContradictionListRequest, *toolDispatchError) {
	actorRole := normalizedActorRole(actor)
	if actorRole != models.UserRoleAdmin {
		return EngramContradictionListRequest{}, invalidParamsWithStatus(
			403,
			engramContradictionListAdminError,
		)
	}
	status, ok := optionalContradictionAlertStatusParam(params, "status")
	if !ok {
		return EngramContradictionListRequest{}, invalidParamError("status")
	}
	paging, pagingErr := parsePagingParams(
		params,
		defaultEngramListLimit,
		defaultEngramListOffset,
	)
	if pagingErr != nil {
		return EngramContradictionListRequest{}, pagingErr
	}
	return EngramContradictionListRequest{
		ActorUserID: actor.UserID,
		ActorRole:   actorRole,
		ProjectID:   optionalProjectIDParam(params, "project_id"),
		Status:      status,
		Limit:       paging.limit,
		Offset:      paging.offset,
	}, nil
}

func parseEngramContradictionResolveRequest(
	actor Actor,
	params map[string]any,
) (EngramContradictionResolveRequest, *toolDispatchError) {
	actorRole := normalizedActorRole(actor)
	if actorRole != models.UserRoleAdmin {
		return EngramContradictionResolveRequest{}, invalidParamsWithStatus(
			403,
			engramContradictionResolveAdminError,
		)
	}
	alertID, ok := requiredUUIDParam(params, "alert_id")
	if !ok {
		return EngramContradictionResolveRequest{}, invalidParamError("alert_id")
	}
	status, ok := requiredContradictionResolveStatusParam(params, "status")
	if !ok {
		return EngramContradictionResolveRequest{}, invalidParamError("status")
	}
	return EngramContradictionResolveRequest{
		ActorUserID: actor.UserID,
		ActorRole:   actorRole,
		AlertID:     alertID,
		ProjectID:   optionalProjectIDParam(params, "project_id"),
		Status:      status,
	}, nil
}

func optionalContradictionAlertStatusParam(
	params map[string]any,
	key string,
) (*models.ContradictionAlertStatus, bool) {
	raw, found := optionalParamValue(params, key)
	if !found {
		return nil, true
	}
	statusText := strings.TrimSpace(fmt.Sprint(raw))
	if statusText == "" {
		return nil, false
	}
	parsed, err := models.ParseContradictionAlertStatus(statusText)
	if err != nil {
		return nil, false
	}
	return &parsed, true
}

func requiredContradictionResolveStatusParam(
	params map[string]any,
	key string,
) (models.ContradictionAlertStatus, bool) {
	parsed, ok := optionalContradictionAlertStatusParam(params, key)
	if !ok || parsed == nil {
		return "", false
	}
	if *parsed == models.ContradictionAlertStatusOpen {
		return "", false
	}
	return *parsed, true
}
