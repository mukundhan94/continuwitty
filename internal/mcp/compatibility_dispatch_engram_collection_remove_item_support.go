package mcp

import "context"

func (service *CompatibilityService) dispatchEngramCollectionRemoveItemTool(
	ctx context.Context,
	actor Actor,
	params map[string]any,
) (map[string]any, bool, *toolDispatchError) {
	if service.engramCollectionRemove == nil {
		return nil, false, nil
	}
	request, dispatchErr := parseEngramCollectionRemoveItemRequest(actor, params)
	if dispatchErr != nil {
		return nil, true, dispatchErr
	}
	return runCollectionMutationDispatch(
		request.CollectionID,
		func() (*EngramCollectionRemoveItemResponse, error) {
			return service.engramCollectionRemove.RemoveCollectionItem(ctx, request)
		},
		func(result EngramCollectionRemoveItemResponse) map[string]any {
			return map[string]any{"result": result}
		},
	)
}

func parseEngramCollectionRemoveItemRequest(
	actor Actor,
	params map[string]any,
) (EngramCollectionRemoveItemRequest, *toolDispatchError) {
	base, dispatchErr := parseCollectionActorRequest(actor, params)
	if dispatchErr != nil {
		return EngramCollectionRemoveItemRequest{}, dispatchErr
	}
	engramID, ok := requiredUUIDParam(params, "engram_id")
	if !ok {
		return EngramCollectionRemoveItemRequest{}, invalidParamError("engram_id")
	}
	return EngramCollectionRemoveItemRequest{
		ActorUserID:  base.ActorUserID,
		ActorRole:    base.ActorRole,
		CollectionID: base.CollectionID,
		EngramID:     engramID,
	}, nil
}
