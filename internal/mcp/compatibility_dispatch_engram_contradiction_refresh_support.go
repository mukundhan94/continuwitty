package mcp

import "context"

func (service *CompatibilityService) dispatchEngramRefreshContradictionsTool(
	ctx context.Context,
	actor Actor,
	params map[string]any,
) (map[string]any, bool, *toolDispatchError) {
	if service.engramContradictionRefresh == nil {
		return nil, false, nil
	}
	request, dispatchErr := parseEngramContradictionRefreshRequest(actor, params)
	if dispatchErr != nil {
		return nil, true, dispatchErr
	}
	refreshed, err := service.engramContradictionRefresh.RefreshEngramContradictionAlerts(ctx, request)
	if err != nil || refreshed == nil {
		return nil, true, internalToolDispatchError()
	}
	return map[string]any{"contradiction_refresh": *refreshed}, true, nil
}
