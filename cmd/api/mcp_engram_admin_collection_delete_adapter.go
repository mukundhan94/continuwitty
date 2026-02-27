package main

import (
	"context"

	"engram/internal/admin"
	"engram/internal/mcp"

	"github.com/google/uuid"
)

func newMCPEngramCollectionDeleteAdapter(service *admin.Service) mcp.EngramCollectionDeleteService {
	if service == nil {
		return nil
	}
	return mcpEngramAdminAdapter{service: service}
}

func (adapter mcpEngramAdminAdapter) DeleteCollection(
	ctx context.Context,
	request mcp.EngramCollectionDeleteRequest,
) (*mcp.EngramCollectionDeleteResponse, error) {
	identity := collectionMutationIdentity{
		ActorUserID:  request.ActorUserID,
		ActorRole:    request.ActorRole,
		CollectionID: request.CollectionID,
	}
	return runVisibleCollectionMutation(
		ctx,
		adapter,
		identity,
		func(ctx context.Context, collectionID uuid.UUID) (*mcp.EngramCollectionDeleteResponse, error) {
			deleted, err := adapter.service.DeleteCollection(
				ctx,
				collectionID,
				request.ActorUserID,
				admin.CollectionDeleteRequest{Reason: request.Reason},
			)
			if err != nil {
				return nil, err
			}
			return &mcp.EngramCollectionDeleteResponse{Deleted: deleted.Deleted}, nil
		},
	)
}
