package main

import (
	"context"

	"engram/internal/chat"
	"engram/internal/mcp"
	"engram/internal/models"
	"engram/internal/repository"
)

func newMCPSaveSessionAsEngramAdapter(db repository.Queryer) mcp.SessionSaveAsEngramService {
	if db == nil {
		return nil
	}
	return mcpSessionListAdapter{db: db}
}

func (adapter mcpSessionListAdapter) SaveSessionAsEngram(
	ctx context.Context,
	request mcp.SessionSaveAsEngramRequest,
) (*models.SaveSessionAsEngramResponse, error) {
	saved, err := chat.NewSessionOperationsService(adapter.db).SaveSessionAsEngram(
		ctx,
		request.ActorUserID,
		request.SessionID,
		request.Payload,
	)
	if err != nil {
		if isChatSessionNotFoundError(err) {
			return nil, nil
		}
		return nil, err
	}
	return &saved, nil
}
