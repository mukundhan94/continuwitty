package mcp

import (
	"context"

	"engram/internal/admin"
)

func (service *CompatibilityService) dispatchEngramCurationRefreshTool(
	ctx context.Context,
	actor Actor,
	params map[string]any,
) (map[string]any, bool, *toolDispatchError) {
	if service.engramCurationRefresh == nil {
		return nil, false, nil
	}
	request, dispatchErr := parseEngramCurationRefreshRequest(actor, params)
	if dispatchErr != nil {
		return nil, true, dispatchErr
	}
	refreshed, err := service.engramCurationRefresh.RefreshEngramLinkCurationSuggestions(ctx, request)
	if err != nil {
		switch {
		case err == admin.ErrEngramNotFound:
			return nil, true, invalidParamsWithStatus(404, err.Error())
		case err == admin.ErrProjectIDRequired,
			err == admin.ErrProjectScopeMismatch:
			return nil, true, invalidParamsWithStatus(400, err.Error())
		}
		return nil, true, internalToolDispatchError()
	}
	if refreshed == nil {
		return nil, true, internalToolDispatchError()
	}
	return map[string]any{"curation_refresh": *refreshed}, true, nil
}
