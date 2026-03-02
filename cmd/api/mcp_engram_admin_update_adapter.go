package main

import (
	"context"
	"errors"

	"engram/internal/admin"
	"engram/internal/mcp"
	"engram/internal/models"
)

func newMCPEngramUpdateAdapter(service *admin.Service) mcp.EngramUpdateService {
	if service == nil {
		return nil
	}
	return mcpEngramAdminAdapter{service: service}
}

func (adapter mcpEngramAdminAdapter) UpdateEngram(
	ctx context.Context,
	request mcp.EngramUpdateRequest,
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
	updated, err := adapter.service.UpdateEngram(
		ctx,
		request.EngramID,
		request.ActorUserID,
		admin.EngramUpdateRequest{
			ExpectedUpdatedAt:       request.ExpectedUpdatedAt,
			Title:                   request.Title,
			Abstract:                request.Abstract,
			DetailedSummaryMarkdown: request.DetailedSummaryMarkdown,
			Tags:                    request.Tags,
			Keywords:                request.Keywords,
			VisibilityScope:         request.VisibilityScope,
			Sources:                 request.Sources,
		},
	)
	if err != nil {
		if errors.Is(err, admin.ErrEngramNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return updated, nil
}
