package mcp

import "context"

func (service *CompatibilityService) dispatchEngramCurationListTool(
	ctx context.Context,
	actor Actor,
	params map[string]any,
) (map[string]any, bool, *toolDispatchError) {
	if service.engramCurationList == nil {
		return nil, false, nil
	}
	request, dispatchErr := parseEngramCurationListRequest(actor, params)
	if dispatchErr != nil {
		return nil, true, dispatchErr
	}
	listed, err := service.engramCurationList.ListMemoryCurationSuggestions(ctx, request)
	if err != nil {
		return nil, true, internalToolDispatchError()
	}
	return map[string]any{"curation_suggestions": listed}, true, nil
}
