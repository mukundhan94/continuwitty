package main

import (
	"context"

	"engram/internal/mcp"
	"engram/internal/models"
	"engram/internal/repository"
)

func newMCPPinEngramAdapter(db repository.Queryer) mcp.PinEngramService {
	if db == nil {
		return nil
	}
	return mcpSessionListAdapter{db: db}
}

func (adapter mcpSessionListAdapter) PinEngram(
	ctx context.Context,
	request mcp.SessionPinEngramRequest,
) (*models.PinnedEngramRecord, error) {
	return repository.PinEngramToSession(
		ctx,
		adapter.db,
		repository.ChatPinEngramInput{
			SessionID:   request.SessionID,
			EngramID:    request.EngramID,
			ActorUserID: request.ActorUserID,
		},
	)
}
