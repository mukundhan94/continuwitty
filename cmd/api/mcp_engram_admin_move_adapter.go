package main

import (
	"context"
	"errors"

	"engram/internal/admin"
	"engram/internal/mcp"
	"engram/internal/models"
)

func newMCPEngramMoveAdapter(service *admin.Service) mcp.EngramMoveService {
	if service == nil {
		return nil
	}
	return mcpEngramAdminAdapter{service: service}
}

func (adapter mcpEngramAdminAdapter) MoveEngram(
	ctx context.Context,
	request mcp.EngramMoveRequest,
) (*models.AdminEngramRecord, error) {
	visible, err := adapter.loadVisibleEngramForMutation(
		ctx,
		request.ActorUserID,
		request.ActorRole,
		request.EngramID,
	)
	if err != nil {
		return nil, err
	}
	if visible == nil {
		return nil, nil
	}
	moved, err := adapter.service.MoveEngram(
		ctx,
		request.EngramID,
		request.ActorUserID,
		string(request.ActorRole),
		admin.EngramMoveRequest{
			TargetProjectID:   request.TargetProjectID,
			Reason:            request.Reason,
			ExpectedUpdatedAt: request.ExpectedUpdatedAt,
		},
	)
	if err != nil {
		if errors.Is(err, admin.ErrEngramNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return moved, nil
}
