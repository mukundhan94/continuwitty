package mcp

import "context"

func (service *CompatibilityService) dispatchEngramContradictionListTool(
	ctx context.Context,
	actor Actor,
	params map[string]any,
) (map[string]any, bool, *toolDispatchError) {
	if service.engramContradictionList == nil {
		return nil, false, nil
	}
	request, dispatchErr := parseEngramContradictionListRequest(actor, params)
	if dispatchErr != nil {
		return nil, true, dispatchErr
	}
	alerts, err := service.engramContradictionList.ListEngramContradictionAlerts(ctx, request)
	if err != nil {
		return nil, true, internalToolDispatchError()
	}
	return map[string]any{"contradiction_alerts": alerts}, true, nil
}
