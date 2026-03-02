package main

import (
	"context"

	"engram/internal/admin"
	"engram/internal/mcp"

	"github.com/google/uuid"
)

func newMCPEngramCollectionAddItemsAdapter(service *admin.Service) mcp.EngramCollectionAddItemsService {
	if service == nil {
		return nil
	}
	return mcpEngramAdminAdapter{service: service}
}

func (adapter mcpEngramAdminAdapter) AddCollectionItems(
	ctx context.Context,
	request mcp.EngramCollectionAddItemsRequest,
) (*mcp.EngramCollectionAddItemsResponse, error) {
	identity := collectionMutationIdentity{
		ActorUserID:  request.ActorUserID,
		ActorRole:    request.ActorRole,
		CollectionID: request.CollectionID,
	}
	return runVisibleCollectionMutation(
		ctx,
		adapter,
		identity,
		func(ctx context.Context, collectionID uuid.UUID) (*mcp.EngramCollectionAddItemsResponse, error) {
			added, err := adapter.service.AddCollectionItems(
				ctx,
				collectionID,
				request.ActorUserID,
				admin.CollectionItemsUpdateRequest{EngramIDs: append([]uuid.UUID(nil), request.EngramIDs...)},
			)
			if err != nil {
				return nil, err
			}
			return &mcp.EngramCollectionAddItemsResponse{Added: added.Added}, nil
		},
	)
}
