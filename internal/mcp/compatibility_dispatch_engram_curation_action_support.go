package mcp

import "context"

func (service *CompatibilityService) dispatchEngramCurationActionTool(
	ctx context.Context,
	actor Actor,
	params map[string]any,
) (map[string]any, bool, *toolDispatchError) {
	if service.engramCurationAction == nil {
		return nil, false, nil
	}
	request, dispatchErr := parseEngramCurationActionRequest(actor, params)
	if dispatchErr != nil {
		return nil, true, dispatchErr
	}
	updated, err := service.engramCurationAction.ActionMemoryCurationSuggestion(ctx, request)
	if err != nil {
		return nil, true, internalToolDispatchError()
	}
	if updated == nil {
		return nil, true, invalidParamsWithStatus(404, "Memory curation suggestion not found")
	}
	return map[string]any{"curation_suggestion": *updated}, true, nil
}
