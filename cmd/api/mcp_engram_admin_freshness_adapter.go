package main

import (
	"context"

	"engram/internal/admin"
	"engram/internal/mcp"
)

func newMCPEngramFreshnessRefreshAdapter(service *admin.Service) mcp.EngramFreshnessRefreshService {
	if service == nil {
		return nil
	}
	return mcpEngramAdminAdapter{service: service}
}

func (adapter mcpEngramAdminAdapter) RefreshEngramFreshness(
	ctx context.Context,
	request mcp.EngramFreshnessRefreshRequest,
) (*mcp.EngramFreshnessRefreshResponse, error) {
	refreshed, err := adapter.service.RefreshEngramFreshness(
		ctx,
		admin.EngramFreshnessRefreshRequest{
			ProjectID:    request.ProjectID,
			HalfLifeDays: request.HalfLifeDays,
		},
	)
	if err != nil {
		return nil, err
	}
	return &mcp.EngramFreshnessRefreshResponse{
		ProjectID:     refreshed.ProjectID,
		HalfLifeDays:  refreshed.HalfLifeDays,
		ReferenceTime: refreshed.ReferenceTime,
		UpdatedCount:  refreshed.UpdatedCount,
	}, nil
}
