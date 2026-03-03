package main

import (
	"context"

	"engram/internal/admin"
	"engram/internal/mcp"
	"engram/internal/models"
)

func newMCPEngramConsolidationRefreshAdapter(service *admin.Service) mcp.EngramConsolidationRefreshService {
	if service == nil {
		return nil
	}
	return mcpEngramAdminAdapter{service: service}
}

func newMCPEngramConsolidationListAdapter(service *admin.Service) mcp.EngramConsolidationListService {
	if service == nil {
		return nil
	}
	return mcpEngramAdminAdapter{service: service}
}

func (adapter mcpEngramAdminAdapter) RefreshEngramConsolidationSuggestions(
	ctx context.Context,
	request mcp.EngramConsolidationRefreshRequest,
) (*mcp.EngramConsolidationRefreshResponse, error) {
	refreshed, err := adapter.service.RefreshEngramConsolidationSuggestions(
		ctx,
		admin.EngramConsolidationSuggestionRefreshRequest{
			ProjectID:    request.ProjectID,
			MinGroupSize: request.MinGroupSize,
		},
	)
	if err != nil {
		return nil, err
	}
	return &mcp.EngramConsolidationRefreshResponse{
		ProjectID:    refreshed.ProjectID,
		MinGroupSize: refreshed.MinGroupSize,
		SuggestedAt:  refreshed.SuggestedAt,
		UpdatedCount: refreshed.UpdatedCount,
	}, nil
}

func (adapter mcpEngramAdminAdapter) ListEngramConsolidationSuggestions(
	ctx context.Context,
	request mcp.EngramConsolidationListRequest,
) ([]models.EngramConsolidationSuggestion, error) {
	return adapter.service.ListEngramConsolidationSuggestions(
		ctx,
		admin.EngramConsolidationSuggestionListRequest{
			ProjectID: request.ProjectID,
			Status:    request.Status,
			Limit:     request.Limit,
			Offset:    request.Offset,
		},
	)
}
