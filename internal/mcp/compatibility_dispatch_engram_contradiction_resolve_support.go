package mcp

import "context"

func (service *CompatibilityService) dispatchEngramContradictionResolveTool(
	ctx context.Context,
	actor Actor,
	params map[string]any,
) (map[string]any, bool, *toolDispatchError) {
	if service.engramContradictionResolve == nil {
		return nil, false, nil
	}
	request, dispatchErr := parseEngramContradictionResolveRequest(actor, params)
	if dispatchErr != nil {
		return nil, true, dispatchErr
	}
	resolved, err := service.engramContradictionResolve.ResolveEngramContradictionAlert(ctx, request)
	if err != nil {
		return nil, true, internalToolDispatchError()
	}
	if resolved == nil {
		return nil, true, invalidParamsWithStatus(404, "Contradiction alert not found")
	}
	return map[string]any{"contradiction_alert": *resolved}, true, nil
}
