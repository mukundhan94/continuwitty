package mcp

import "context"

func (service *CompatibilityService) dispatchEngramCollectionAddItemsTool(
	ctx context.Context,
	actor Actor,
	params map[string]any,
) (map[string]any, bool, *toolDispatchError) {
	if service.engramCollectionAddItems == nil {
		return nil, false, nil
	}
	request, dispatchErr := parseEngramCollectionAddItemsRequest(actor, params)
	if dispatchErr != nil {
		return nil, true, dispatchErr
	}
	return runCollectionMutationDispatch(
		request.CollectionID,
		func() (*EngramCollectionAddItemsResponse, error) {
			return service.engramCollectionAddItems.AddCollectionItems(ctx, request)
		},
		func(result EngramCollectionAddItemsResponse) map[string]any {
			return map[string]any{"result": result}
		},
	)
}

func parseEngramCollectionAddItemsRequest(
	actor Actor,
	params map[string]any,
) (EngramCollectionAddItemsRequest, *toolDispatchError) {
	base, dispatchErr := parseCollectionActorRequest(actor, params)
	if dispatchErr != nil {
		return EngramCollectionAddItemsRequest{}, dispatchErr
	}
	engramIDs, ok := optionalUUIDSliceParam(params, "engram_ids")
	if !ok {
		return EngramCollectionAddItemsRequest{}, invalidParamError("engram_ids")
	}
	return EngramCollectionAddItemsRequest{
		ActorUserID:  base.ActorUserID,
		ActorRole:    base.ActorRole,
		CollectionID: base.CollectionID,
		EngramIDs:    engramIDs,
	}, nil
}
