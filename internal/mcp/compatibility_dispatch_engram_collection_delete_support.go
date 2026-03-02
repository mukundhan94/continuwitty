package mcp

import "context"

func (service *CompatibilityService) dispatchEngramCollectionDeleteTool(
	ctx context.Context,
	actor Actor,
	params map[string]any,
) (map[string]any, bool, *toolDispatchError) {
	if service.engramCollectionDelete == nil {
		return nil, false, nil
	}
	request, dispatchErr := parseEngramCollectionDeleteRequest(actor, params)
	if dispatchErr != nil {
		return nil, true, dispatchErr
	}
	return runCollectionMutationDispatch(
		request.CollectionID,
		func() (*EngramCollectionDeleteResponse, error) {
			return service.engramCollectionDelete.DeleteCollection(ctx, request)
		},
		func(result EngramCollectionDeleteResponse) map[string]any {
			return map[string]any{"result": result}
		},
	)
}

func parseEngramCollectionDeleteRequest(
	actor Actor,
	params map[string]any,
) (EngramCollectionDeleteRequest, *toolDispatchError) {
	base, dispatchErr := parseCollectionActorRequest(actor, params)
	if dispatchErr != nil {
		return EngramCollectionDeleteRequest{}, dispatchErr
	}
	reason, ok := optionalStringPointerParam(params, "reason")
	if !ok {
		return EngramCollectionDeleteRequest{}, invalidParamError("reason")
	}
	return EngramCollectionDeleteRequest{
		ActorUserID:  base.ActorUserID,
		ActorRole:    base.ActorRole,
		CollectionID: base.CollectionID,
		Reason:       reason,
	}, nil
}
