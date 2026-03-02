package mcp

import (
	"context"

	"engram/internal/models"
)

func (service *CompatibilityService) dispatchEngramLinkCreateTool(
	ctx context.Context,
	actor Actor,
	params map[string]any,
) (map[string]any, bool, *toolDispatchError) {
	if service.engramLinkCreate == nil {
		return nil, false, nil
	}
	request, dispatchErr := parseEngramLinkCreateRequest(actor, params)
	if dispatchErr != nil {
		return nil, true, dispatchErr
	}
	created, err := service.engramLinkCreate.CreateEngramLink(ctx, request)
	if err != nil {
		return nil, true, mapEngramLinkDispatchError(err)
	}
	return engramLinkRecordPayload(created, "Engram source or target not found")
}

func (service *CompatibilityService) dispatchEngramLinkUpdateTool(
	ctx context.Context,
	actor Actor,
	params map[string]any,
) (map[string]any, bool, *toolDispatchError) {
	if service.engramLinkUpdate == nil {
		return nil, false, nil
	}
	request, dispatchErr := parseEngramLinkUpdateRequest(actor, params)
	if dispatchErr != nil {
		return nil, true, dispatchErr
	}
	updated, err := service.engramLinkUpdate.UpdateEngramLink(ctx, request)
	if err != nil {
		return nil, true, mapEngramLinkDispatchError(err)
	}
	return engramLinkRecordPayload(updated, "Engram link not found")
}

func (service *CompatibilityService) dispatchEngramLinkArchiveTool(
	ctx context.Context,
	actor Actor,
	params map[string]any,
) (map[string]any, bool, *toolDispatchError) {
	if service.engramLinkArchive == nil {
		return nil, false, nil
	}
	request, dispatchErr := parseEngramLinkArchiveRequest(actor, params)
	if dispatchErr != nil {
		return nil, true, dispatchErr
	}
	archived, err := service.engramLinkArchive.ArchiveEngramLink(ctx, request)
	if err != nil {
		return nil, true, mapEngramLinkDispatchError(err)
	}
	return engramLinkRecordPayload(archived, "Engram link not found")
}

func engramLinkRecordPayload(
	record *models.EngramLinkRecord,
	notFoundDetail string,
) (map[string]any, bool, *toolDispatchError) {
	if record == nil {
		return nil, true, invalidParamsWithStatus(404, notFoundDetail)
	}
	return map[string]any{"link": *record}, true, nil
}
