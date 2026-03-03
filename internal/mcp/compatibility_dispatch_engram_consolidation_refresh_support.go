package mcp

import "context"

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
