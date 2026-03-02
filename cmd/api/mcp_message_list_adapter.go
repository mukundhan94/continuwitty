package main

import (
	"context"

	"engram/internal/mcp"
	"engram/internal/models"
	"engram/internal/repository"
)

func newMCPMessageListAdapter(db repository.Queryer) mcp.MessageListService {
	if db == nil {
		return nil
	}
	return mcpSessionListAdapter{db: db}
}

func (adapter mcpSessionListAdapter) ListMessages(
	ctx context.Context,
	request mcp.MessageListRequest,
) ([]models.ChatMessageRecord, error) {
	return repository.ListChatMessages(
		ctx,
		adapter.db,
		repository.ChatMessageListInput{
			SessionID:   request.SessionID,
			ActorUserID: request.ActorUserID,
			Limit:       request.Limit,
			Offset:      request.Offset,
		},
	)
}
