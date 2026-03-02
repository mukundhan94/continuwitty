package main

import (
	"context"

	"engram/internal/mcp"
	"engram/internal/models"
	"engram/internal/repository"
)

func newMCPPinnedEngramListAdapter(db repository.Queryer) mcp.PinnedEngramListService {
	if db == nil {
		return nil
	}
	return mcpSessionListAdapter{db: db}
}

func (adapter mcpSessionListAdapter) ListPinnedEngrams(
	ctx context.Context,
	request mcp.SessionScopedRequest,
) ([]models.EngramSummary, error) {
	return repository.ListPinnedEngramSummaries(
		ctx,
		adapter.db,
		repository.ChatPinnedListInput{
			SessionID:   request.SessionID,
			ActorUserID: request.ActorUserID,
		},
	)
}
