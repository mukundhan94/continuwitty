package main

import (
	"context"
	"errors"
	"strings"

	"engram/internal/admin"
	"engram/internal/mcp"
	"engram/internal/models"

	"github.com/google/uuid"
)

type mcpEngramAdminAdapter struct {
	service *admin.Service
}

func newMCPEngramListAdapter(service *admin.Service) mcp.EngramListService {
	if service == nil {
		return nil
	}
	return mcpEngramAdminAdapter{service: service}
}

func newMCPEngramGetAdapter(service *admin.Service) mcp.EngramGetService {
	if service == nil {
		return nil
	}
	return mcpEngramAdminAdapter{service: service}
}

func newMCPEngramCollectionListAdapter(service *admin.Service) mcp.EngramCollectionListService {
	if service == nil {
		return nil
	}
	return mcpEngramAdminAdapter{service: service}
}

func newMCPEngramDeleteAdapter(service *admin.Service) mcp.EngramDeleteService {
	if service == nil {
		return nil
	}
	return mcpEngramAdminAdapter{service: service}
}

func newMCPEngramRestoreAdapter(service *admin.Service) mcp.EngramRestoreService {
	if service == nil {
		return nil
	}
	return mcpEngramAdminAdapter{service: service}
}

func (adapter mcpEngramAdminAdapter) ListEngrams(
	ctx context.Context,
	request mcp.EngramListRequest,
) ([]models.AdminEngramRecord, error) {
	return adapter.service.ListEngrams(
		ctx,
		admin.MemoryAdminEngramListRequest{
			MemoryAdminListRequest: admin.MemoryAdminListRequest{
				ProjectID:      request.ProjectID,
				OwnerUserID:    ownerScopeForRole(request.ActorUserID, request.ActorRole),
				IncludeDeleted: request.IncludeDeleted,
				Limit:          request.Limit,
				Offset:         request.Offset,
			},
			SessionID: request.SessionID,
			QueryText: request.QueryText,
		},
	)
}

func (adapter mcpEngramAdminAdapter) GetEngram(
	ctx context.Context,
	request mcp.EngramGetRequest,
) (*models.AdminEngramRecord, error) {
	engram, err := adapter.service.GetEngram(ctx, request.EngramID, request.IncludeDeleted)
	if err != nil {
		if errors.Is(err, admin.ErrEngramNotFound) {
			return nil, nil
		}
		return nil, err
	}
	if requestAllowsOwnerBypass(request.ActorRole) {
		return engram, nil
	}
	if engram.OwnerUserID == nil || *engram.OwnerUserID != request.ActorUserID {
		return nil, nil
	}
	return engram, nil
}

func (adapter mcpEngramAdminAdapter) ListCollections(
	ctx context.Context,
	request mcp.EngramCollectionListRequest,
) ([]models.EngramCollectionRecord, error) {
	return adapter.service.ListCollections(
		ctx,
		admin.MemoryAdminListRequest{
			ProjectID:      request.ProjectID,
			OwnerUserID:    ownerScopeForRole(request.ActorUserID, request.ActorRole),
			IncludeDeleted: request.IncludeDeleted,
			Limit:          request.Limit,
			Offset:         request.Offset,
		},
	)
}

func (adapter mcpEngramAdminAdapter) executeVisibleEngramMutation(
	ctx context.Context,
	actorUserID uuid.UUID,
	actorRole models.UserRole,
	engramID uuid.UUID,
	mutate func(context.Context, uuid.UUID) (engramMutationOutcome, error),
) (*engramMutationOutcome, error) {
	visible, err := adapter.loadVisibleEngramForMutation(ctx, actorUserID, actorRole, engramID)
	if err != nil {
		return nil, err
	}
	if visible == nil {
		return nil, nil
	}
	mutation, err := mutate(ctx, engramID)
	if err != nil {
		if errors.Is(err, admin.ErrEngramNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &mutation, nil
}

func (adapter mcpEngramAdminAdapter) loadVisibleEngramForMutation(
	ctx context.Context,
	actorUserID uuid.UUID,
	actorRole models.UserRole,
	engramID uuid.UUID,
) (*models.AdminEngramRecord, error) {
	return adapter.GetEngram(
		ctx,
		mcp.EngramGetRequest{
			ActorUserID:    actorUserID,
			ActorRole:      actorRole,
			EngramID:       engramID,
			IncludeDeleted: true,
		},
	)
}

func ownerScopeForRole(actorUserID uuid.UUID, actorRole models.UserRole) *uuid.UUID {
	if requestAllowsOwnerBypass(actorRole) {
		return nil
	}
	ownerUserID := actorUserID
	return &ownerUserID
}

func requestAllowsOwnerBypass(actorRole models.UserRole) bool {
	return strings.EqualFold(strings.TrimSpace(string(actorRole)), string(models.UserRoleAdmin))
}
