package mcp

import "context"

func (service *CompatibilityService) dispatchEngramLinkListTool(
	ctx context.Context,
	actor Actor,
	params map[string]any,
) (map[string]any, bool, *toolDispatchError) {
	if service.engramLinkList == nil {
		return nil, false, nil
	}
	request, dispatchErr := parseEngramLinkListRequest(actor, params)
	if dispatchErr != nil {
		return nil, true, dispatchErr
	}
	links, err := service.engramLinkList.ListEngramLinks(ctx, request)
	if err != nil {
		return nil, true, internalToolDispatchError()
	}
	return map[string]any{"links": links}, true, nil
}

func (service *CompatibilityService) dispatchEngramLinkSuggestTool(
	ctx context.Context,
	actor Actor,
	params map[string]any,
) (map[string]any, bool, *toolDispatchError) {
	if service.engramLinkSuggest == nil {
		return nil, false, nil
	}
	request, dispatchErr := parseEngramLinkSuggestRequest(actor, params)
	if dispatchErr != nil {
		return nil, true, dispatchErr
	}
	suggestions, err := service.engramLinkSuggest.SuggestEngramLinks(ctx, request)
	if err != nil {
		return nil, true, internalToolDispatchError()
	}
	return map[string]any{"suggestions": suggestions}, true, nil
}

func (service *CompatibilityService) dispatchEngramTracePathTool(
	ctx context.Context,
	actor Actor,
	params map[string]any,
) (map[string]any, bool, *toolDispatchError) {
	if service.engramTracePath == nil {
		return nil, false, nil
	}
	request, dispatchErr := parseEngramTracePathRequest(actor, params)
	if dispatchErr != nil {
		return nil, true, dispatchErr
	}
	steps, err := service.engramTracePath.TraceEngramPath(ctx, request)
	if err != nil {
		return nil, true, internalToolDispatchError()
	}
	return map[string]any{"steps": steps}, true, nil
}
