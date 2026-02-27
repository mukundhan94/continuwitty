package mcp

import (
	"context"

	"engram/internal/models"
)

func (service *CompatibilityService) dispatchEngramCollectionUpdateTool(
	ctx context.Context,
	actor Actor,
	params map[string]any,
) (map[string]any, bool, *toolDispatchError) {
	if service.engramCollectionUpdate == nil {
		return nil, false, nil
	}
	request, dispatchErr := parseEngramCollectionUpdateRequest(actor, params)
	if dispatchErr != nil {
		return nil, true, dispatchErr
	}
	return runCollectionMutationDispatch(
		request.CollectionID,
		func() (*models.EngramCollectionRecord, error) {
			return service.engramCollectionUpdate.UpdateCollection(ctx, request)
		},
		func(collection models.EngramCollectionRecord) map[string]any {
			return map[string]any{"collection": collection}
		},
	)
}

func parseEngramCollectionUpdateRequest(
	actor Actor,
	params map[string]any,
) (EngramCollectionUpdateRequest, *toolDispatchError) {
	base, dispatchErr := parseCollectionActorRequest(actor, params)
	if dispatchErr != nil {
		return EngramCollectionUpdateRequest{}, dispatchErr
	}
	name, ok := optionalStringPointerParam(params, "name")
	if !ok {
		return EngramCollectionUpdateRequest{}, invalidParamError("name")
	}
	description, ok := optionalStringPointerParam(params, "description")
	if !ok {
		return EngramCollectionUpdateRequest{}, invalidParamError("description")
	}
	expectedUpdatedAt, ok := optionalRFC3339TimePointerParam(params, "expected_updated_at")
	if !ok {
		return EngramCollectionUpdateRequest{}, invalidParamError("expected_updated_at")
	}
	return EngramCollectionUpdateRequest{
		ActorUserID:       base.ActorUserID,
		ActorRole:         base.ActorRole,
		CollectionID:      base.CollectionID,
		ExpectedUpdatedAt: expectedUpdatedAt,
		Name:              name,
		Description:       description,
	}, nil
}
