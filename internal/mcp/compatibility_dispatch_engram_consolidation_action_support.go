package mcp

import "context"

func (service *CompatibilityService) dispatchEngramConsolidationActionTool(
	ctx context.Context,
	actor Actor,
	params map[string]any,
) (map[string]any, bool, *toolDispatchError) {
	if service.engramConsolidationAction == nil {
		return nil, false, nil
	}
	request, dispatchErr := parseEngramConsolidationActionRequest(actor, params)
	if dispatchErr != nil {
		return nil, true, dispatchErr
	}
	updated, err := service.engramConsolidationAction.ActionEngramConsolidationSuggestion(ctx, request)
	if err != nil {
		return nil, true, internalToolDispatchError()
	}
	if updated == nil {
		return nil, true, invalidParamsWithStatus(404, "Consolidation suggestion not found")
	}
	return map[string]any{"consolidation_suggestion": *updated}, true, nil
}
