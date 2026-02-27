package mcp

import "context"

func (service *CompatibilityService) dispatchProjectExportBundleTool(
	ctx context.Context,
	actor Actor,
	params map[string]any,
) (map[string]any, bool, *toolDispatchError) {
	if service.projectExport == nil {
		return nil, false, nil
	}
	request, dispatchErr := parseProjectExportRequest(actor, params)
	if dispatchErr != nil {
		return nil, true, dispatchErr
	}
	return runProjectTransferDispatch(
		func() (*ProjectExportResponse, error) {
			return service.projectExport.ExportProjectBundle(ctx, request)
		},
		func(response ProjectExportResponse) map[string]any {
			return map[string]any{"bundle": response.Bundle}
		},
	)
}

func parseProjectExportRequest(
	actor Actor,
	params map[string]any,
) (ProjectExportRequest, *toolDispatchError) {
	projectID, ok := requiredStringParam(params, "project_id")
	if !ok {
		return ProjectExportRequest{}, invalidParamError("project_id")
	}
	collectionIDs, ok := optionalUUIDSliceParam(params, "collection_ids")
	if !ok {
		return ProjectExportRequest{}, invalidParamError("collection_ids")
	}
	includeEmbeddings, ok := optionalBoolParam(params, "include_embeddings", false)
	if !ok {
		return ProjectExportRequest{}, invalidParamError("include_embeddings")
	}
	return ProjectExportRequest{
		ActorUserID:       actor.UserID,
		ActorRole:         normalizedActorRole(actor),
		ProjectID:         projectID,
		CollectionIDs:     collectionIDs,
		IncludeEmbeddings: includeEmbeddings,
	}, nil
}
