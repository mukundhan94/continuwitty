package mcp

import "context"

func (service *CompatibilityService) dispatchEngramListTool(
	ctx context.Context,
	actor Actor,
	params map[string]any,
) (map[string]any, bool, *toolDispatchError) {
	if service.engramList == nil {
		return nil, false, nil
	}
	paging, pagingErr := parsePagingParams(
		params,
		defaultEngramListLimit,
		defaultEngramListOffset,
	)
	if pagingErr != nil {
		return nil, true, pagingErr
	}
	includeDeleted, ok := optionalBoolParam(params, "include_deleted", false)
	if !ok {
		return nil, true, invalidParamError("include_deleted")
	}
	sessionID, ok := optionalUUIDParam(params, "session_id")
	if !ok {
		return nil, true, invalidParamError("session_id")
	}
	queryText, ok := optionalStringPointerParam(params, "q")
	if !ok {
		return nil, true, invalidParamError("q")
	}
	engrams, err := service.engramList.ListEngrams(
		ctx,
		EngramListRequest{
			ActorUserID:    actor.UserID,
			ActorRole:      normalizedActorRole(actor),
			ProjectID:      optionalProjectIDParam(params, "project_id"),
			SessionID:      sessionID,
			QueryText:      queryText,
			IncludeDeleted: includeDeleted,
			Limit:          paging.limit,
			Offset:         paging.offset,
		},
	)
	if err != nil {
		return nil, true, internalToolDispatchError()
	}
	return map[string]any{"engrams": engrams}, true, nil
}

func (service *CompatibilityService) dispatchEngramGetTool(
	ctx context.Context,
	actor Actor,
	params map[string]any,
) (map[string]any, bool, *toolDispatchError) {
	if service.engramGet == nil {
		return nil, false, nil
	}
	engramID, ok := requiredUUIDParam(params, "engram_id")
	if !ok {
		return nil, true, invalidParamError("engram_id")
	}
	includeDeleted, ok := optionalBoolParam(params, "include_deleted", true)
	if !ok {
		return nil, true, invalidParamError("include_deleted")
	}
	engram, err := service.engramGet.GetEngram(
		ctx,
		EngramGetRequest{
			ActorUserID:    actor.UserID,
			ActorRole:      normalizedActorRole(actor),
			EngramID:       engramID,
			IncludeDeleted: includeDeleted,
		},
	)
	if err != nil {
		return nil, true, internalToolDispatchError()
	}
	if engram == nil {
		return nil, true, invalidParamsWithStatus(404, "Engram not found")
	}
	return map[string]any{"engram": *engram}, true, nil
}

func (service *CompatibilityService) dispatchEngramCollectionListTool(
	ctx context.Context,
	actor Actor,
	params map[string]any,
) (map[string]any, bool, *toolDispatchError) {
	if service.engramCollectionList == nil {
		return nil, false, nil
	}
	paging, pagingErr := parsePagingParams(
		params,
		defaultCollectionListLimit,
		defaultCollectionListOffset,
	)
	if pagingErr != nil {
		return nil, true, pagingErr
	}
	includeDeleted, ok := optionalBoolParam(params, "include_deleted", false)
	if !ok {
		return nil, true, invalidParamError("include_deleted")
	}
	collections, err := service.engramCollectionList.ListCollections(
		ctx,
		EngramCollectionListRequest{
			ActorUserID:    actor.UserID,
			ActorRole:      normalizedActorRole(actor),
			ProjectID:      optionalProjectIDParam(params, "project_id"),
			IncludeDeleted: includeDeleted,
			Limit:          paging.limit,
			Offset:         paging.offset,
		},
	)
	if err != nil {
		return nil, true, internalToolDispatchError()
	}
	return map[string]any{"collections": collections}, true, nil
}

func (service *CompatibilityService) dispatchEngramRehydrateTool(
	ctx context.Context,
	actor Actor,
	params map[string]any,
) (map[string]any, bool, *toolDispatchError) {
	if service.engramRehydrate == nil {
		return nil, false, nil
	}
	engramID, ok := requiredUUIDParam(params, "engram_id")
	if !ok {
		return nil, true, invalidParamError("engram_id")
	}
	bundle, err := service.engramRehydrate.RehydrateEngram(
		ctx,
		EngramRehydrateRequest{
			ActorUserID: actor.UserID,
			EngramID:    engramID,
		},
	)
	if err != nil {
		return nil, true, internalToolDispatchError()
	}
	if bundle == nil {
		return nil, true, &toolDispatchError{
			code:    -32004,
			message: "Engram not found",
			data: map[string]any{
				"engram_id": engramID.String(),
			},
		}
	}
	return map[string]any{"bundle": *bundle}, true, nil
}
