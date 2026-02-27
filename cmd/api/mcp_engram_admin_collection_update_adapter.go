package main

import (
	"context"

	"engram/internal/admin"
	"engram/internal/mcp"
	"engram/internal/models"

	"github.com/google/uuid"
)

func newMCPEngramCollectionUpdateAdapter(service *admin.Service) mcp.EngramCollectionUpdateService {
	if service == nil {
		return nil
	}
	return mcpEngramAdminAdapter{service: service}
}

func (adapter mcpEngramAdminAdapter) UpdateCollection(
	ctx context.Context,
	request mcp.EngramCollectionUpdateRequest,
) (*models.EngramCollectionRecord, error) {
	identity := collectionMutationIdentity{
		ActorUserID:  request.ActorUserID,
		ActorRole:    request.ActorRole,
		CollectionID: request.CollectionID,
	}
	return runVisibleCollectionMutation(
		ctx,
		adapter,
		identity,
		func(ctx context.Context, collectionID uuid.UUID) (*models.EngramCollectionRecord, error) {
			updated, err := adapter.service.UpdateCollection(
				ctx,
				collectionID,
				admin.CollectionUpdateRequest{
					ExpectedUpdatedAt: request.ExpectedUpdatedAt,
					Name:              request.Name,
					Description:       request.Description,
				},
			)
			if err != nil {
				return nil, err
			}
			if updated == nil {
				return nil, nil
			}
			collection := cloneEngramCollection(*updated)
			return &collection, nil
		},
	)
}
