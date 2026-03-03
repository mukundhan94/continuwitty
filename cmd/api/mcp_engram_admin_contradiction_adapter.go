package main

import (
	"context"

	"engram/internal/admin"
	"engram/internal/mcp"
	"engram/internal/models"
)

func newMCPEngramContradictionRefreshAdapter(service *admin.Service) mcp.EngramContradictionRefreshService {
	if service == nil {
		return nil
	}
	return mcpEngramAdminAdapter{service: service}
}

func newMCPEngramContradictionListAdapter(service *admin.Service) mcp.EngramContradictionListService {
	if service == nil {
		return nil
	}
	return mcpEngramAdminAdapter{service: service}
}

func newMCPEngramContradictionResolveAdapter(service *admin.Service) mcp.EngramContradictionResolveService {
	if service == nil {
		return nil
	}
	return mcpEngramAdminAdapter{service: service}
}

func (adapter mcpEngramAdminAdapter) RefreshEngramContradictionAlerts(
	ctx context.Context,
	request mcp.EngramContradictionRefreshRequest,
) (*mcp.EngramContradictionRefreshResponse, error) {
	refreshed, err := adapter.service.RefreshEngramContradictionAlerts(
		ctx,
		admin.EngramContradictionAlertRefreshRequest{
			ProjectID: request.ProjectID,
		},
	)
	if err != nil {
		return nil, err
	}
	return &mcp.EngramContradictionRefreshResponse{
		ProjectID:    refreshed.ProjectID,
		DetectedAt:   refreshed.DetectedAt,
		UpdatedCount: refreshed.UpdatedCount,
	}, nil
}

func (adapter mcpEngramAdminAdapter) ListEngramContradictionAlerts(
	ctx context.Context,
	request mcp.EngramContradictionListRequest,
) ([]models.EngramContradictionAlert, error) {
	return adapter.service.ListEngramContradictionAlerts(
		ctx,
		admin.EngramContradictionAlertListRequest{
			ProjectID: request.ProjectID,
			Status:    request.Status,
			Limit:     request.Limit,
			Offset:    request.Offset,
		},
	)
}

func (adapter mcpEngramAdminAdapter) ResolveEngramContradictionAlert(
	ctx context.Context,
	request mcp.EngramContradictionResolveRequest,
) (*models.EngramContradictionAlert, error) {
	return adapter.service.ResolveEngramContradictionAlert(
		ctx,
		request.AlertID,
		request.ActorUserID,
		admin.EngramContradictionAlertResolveRequest{
			ProjectID: request.ProjectID,
			Status:    request.Status,
		},
	)
}
