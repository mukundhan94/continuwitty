package main

import (
	"context"

	"engram/internal/admin"
	"engram/internal/mcp"

	"github.com/google/uuid"
)

func newMCPEngramCollectionRemoveItemAdapter(service *admin.Service) mcp.EngramCollectionRemoveItemService {
	if service == nil {
		return nil
	}
	return mcpEngramAdminAdapter{service: service}
}

func (adapter mcpEngramAdminAdapter) RemoveCollectionItem(
	ctx context.Context,
	request mcp.EngramCollectionRemoveItemRequest,
) (*mcp.EngramCollectionRemoveItemResponse, error) {
	identity := collectionMutationIdentity{
		ActorUserID:  request.ActorUserID,
		ActorRole:    request.ActorRole,
		CollectionID: request.CollectionID,
	}
	return runVisibleCollectionMutation(
		ctx,
		adapter,
		identity,
		func(ctx context.Context, collectionID uuid.UUID) (*mcp.EngramCollectionRemoveItemResponse, error) {
			removed, err := adapter.service.RemoveCollectionItem(
				ctx,
				collectionID,
				request.EngramID,
			)
			if err != nil {
				return nil, err
			}
			return &mcp.EngramCollectionRemoveItemResponse{Removed: removed.Removed}, nil
		},
	)
}
