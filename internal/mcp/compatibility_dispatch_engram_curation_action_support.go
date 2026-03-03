package mcp

import (
	"context"
	"errors"

	"engram/internal/admin"
)

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
		switch {
		case errors.Is(err, admin.ErrMemoryCurationSuggestionNotFound),
			errors.Is(err, admin.ErrConsolidationSuggestionNotFound),
			errors.Is(err, admin.ErrContradictionAlertNotFound):
			return nil, true, invalidParamsWithStatus(404, err.Error())
		case errors.Is(err, admin.ErrMemoryCurationSuggestionActionInvalid),
			errors.Is(err, admin.ErrMemoryCurationSuggestionPayloadInvalid),
			errors.Is(err, admin.ErrMemoryCurationSuggestionApplyUnsupported):
			return nil, true, invalidParamsWithStatus(400, err.Error())
		}
		return nil, true, internalToolDispatchError()
	}
	if updated == nil {
		return nil, true, invalidParamsWithStatus(404, "Memory curation suggestion not found")
	}
	return map[string]any{"curation_suggestion": *updated}, true, nil
}
