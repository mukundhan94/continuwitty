package main

import (
	"context"

	"engram/internal/admin"
	"engram/internal/mcp"
	"engram/internal/models"
)

func newMCPEngramCurationListAdapter(service *admin.Service) mcp.EngramCurationListService {
	if service == nil {
		return nil
	}
	return mcpEngramAdminAdapter{service: service}
}

func newMCPEngramCurationActionAdapter(service *admin.Service) mcp.EngramCurationActionService {
	if service == nil {
		return nil
	}
	return mcpEngramAdminAdapter{service: service}
}

func newMCPEngramCurationRefreshAdapter(service *admin.Service) mcp.EngramCurationRefreshService {
	if service == nil {
		return nil
	}
	return mcpEngramAdminAdapter{service: service}
}

func (adapter mcpEngramAdminAdapter) ListMemoryCurationSuggestions(
	ctx context.Context,
	request mcp.EngramCurationListRequest,
) ([]models.MemoryCurationSuggestion, error) {
	return adapter.service.ListMemoryCurationSuggestions(
		ctx,
		admin.MemoryCurationSuggestionListRequest{
			ProjectID:      request.ProjectID,
			SessionID:      request.SessionID,
			SuggestionType: request.SuggestionType,
			Status:         request.Status,
			Limit:          request.Limit,
			Offset:         request.Offset,
		},
	)
}

func (adapter mcpEngramAdminAdapter) ActionMemoryCurationSuggestion(
	ctx context.Context,
	request mcp.EngramCurationActionRequest,
) (*models.MemoryCurationSuggestion, error) {
	return adapter.service.ActionMemoryCurationSuggestion(
		ctx,
		request.SuggestionID,
		request.ActorUserID,
		admin.MemoryCurationSuggestionActionRequest{
			ProjectID: request.ProjectID,
			Status:    request.Status,
		},
	)
}

func (adapter mcpEngramAdminAdapter) RefreshEngramLinkCurationSuggestions(
	ctx context.Context,
	request mcp.EngramCurationRefreshRequest,
) (*mcp.EngramCurationRefreshResponse, error) {
	refreshed, err := adapter.service.RefreshEngramLinkCurationSuggestions(
		ctx,
		request.ActorUserID,
		admin.EngramLinkCurationSuggestionRefreshRequest{
			SourceEngramID:    request.SourceEngramID,
			IncludeArchived:   request.IncludeArchived,
			Limit:             request.Limit,
			StaleAfterDays:    request.StaleAfterDays,
			LowValueThreshold: request.LowValueThreshold,
		},
	)
	if err != nil {
		return nil, err
	}
	return &mcp.EngramCurationRefreshResponse{
		ProjectID:      refreshed.ProjectID,
		SourceEngramID: refreshed.SourceEngramID,
		SuggestedAt:    refreshed.SuggestedAt,
		UpdatedCount:   refreshed.UpdatedCount,
	}, nil
}
