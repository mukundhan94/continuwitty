package main

import (
	"context"

	"engram/internal/admin"
	"engram/internal/models"

	"github.com/google/uuid"
)

type engramMutationKind string

const (
	engramMutationDelete  engramMutationKind = "delete"
	engramMutationRestore engramMutationKind = "restore"
)

type engramMutationRequest struct {
	ActorUserID uuid.UUID
	ActorRole   models.UserRole
	EngramID    uuid.UUID
	Reason      *string
	Kind        engramMutationKind
}

type engramMutationOutcome struct {
	EngramID uuid.UUID
	Applied  bool
}

func runEngramMutation[T any](
	ctx context.Context,
	adapter mcpEngramAdminAdapter,
	request engramMutationRequest,
	build func(engramMutationOutcome) *T,
) (*T, error) {
	mutation, err := adapter.executeVisibleEngramMutation(
		ctx,
		request.ActorUserID,
		request.ActorRole,
		request.EngramID,
		func(ctx context.Context, engramID uuid.UUID) (engramMutationOutcome, error) {
			applied, err := adapter.applyEngramMutation(ctx, request, engramID)
			if err != nil {
				return engramMutationOutcome{}, err
			}
			return engramMutationOutcome{EngramID: engramID, Applied: applied}, nil
		},
	)
	if err != nil {
		return nil, err
	}
	if mutation == nil {
		return nil, nil
	}
	return build(*mutation), nil
}

func (adapter mcpEngramAdminAdapter) applyEngramMutation(
	ctx context.Context,
	request engramMutationRequest,
	engramID uuid.UUID,
) (bool, error) {
	if request.Kind == engramMutationRestore {
		restored, err := adapter.service.RestoreEngram(ctx, engramID)
		if err != nil {
			return false, err
		}
		return restored.Restored, nil
	}
	deleted, err := adapter.service.DeleteEngram(
		ctx,
		engramID,
		request.ActorUserID,
		admin.EngramDeleteRequest{Reason: request.Reason},
	)
	if err != nil {
		return false, err
	}
	return deleted.Deleted, nil
}
