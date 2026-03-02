package main

import (
	"context"
	"errors"

	"engram/internal/admin"
	"engram/internal/models"

	"github.com/google/uuid"
)

func (adapter mcpEngramAdminAdapter) loadVisibleCollectionForMutation(
	ctx context.Context,
	actorUserID uuid.UUID,
	actorRole models.UserRole,
	collectionID uuid.UUID,
) (*models.EngramCollectionRecord, error) {
	collection, err := adapter.service.GetCollection(ctx, collectionID, true)
	if err != nil {
		if errors.Is(err, admin.ErrCollectionNotFound) {
			return nil, nil
		}
		return nil, err
	}
	if requestAllowsOwnerBypass(actorRole) {
		return collection, nil
	}
	if collection.OwnerUserID != actorUserID {
		return nil, nil
	}
	return collection, nil
}

type collectionMutationIdentity struct {
	ActorUserID  uuid.UUID
	ActorRole    models.UserRole
	CollectionID uuid.UUID
}

func runVisibleCollectionMutation[Result any](
	ctx context.Context,
	adapter mcpEngramAdminAdapter,
	identity collectionMutationIdentity,
	mutate func(ctx context.Context, collectionID uuid.UUID) (*Result, error),
) (*Result, error) {
	visible, err := adapter.loadVisibleCollectionForMutation(
		ctx,
		identity.ActorUserID,
		identity.ActorRole,
		identity.CollectionID,
	)
	if err != nil {
		return nil, err
	}
	if visible == nil {
		return nil, nil
	}
	result, err := mutate(ctx, identity.CollectionID)
	if err != nil {
		if errors.Is(err, admin.ErrCollectionNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return result, nil
}
