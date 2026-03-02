package main

import (
	"context"

	"engram/internal/mcp"
	"engram/internal/repository"
)

func newMCPUnpinEngramAdapter(db repository.Queryer) mcp.UnpinEngramService {
	if db == nil {
		return nil
	}
	return mcpSessionListAdapter{db: db}
}

func (adapter mcpSessionListAdapter) UnpinEngram(
	ctx context.Context,
	request mcp.SessionPinEngramRequest,
) (bool, error) {
	return repository.UnpinEngramFromSession(
		ctx,
		adapter.db,
		repository.ChatPinEngramInput{
			SessionID:   request.SessionID,
			EngramID:    request.EngramID,
			ActorUserID: request.ActorUserID,
		},
	)
}
