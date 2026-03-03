package mcp

import "context"

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
