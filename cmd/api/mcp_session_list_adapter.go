package main

import (
	"context"

	"engram/internal/mcp"
	"engram/internal/models"
	"engram/internal/repository"
)

type mcpSessionListAdapter struct {
	db repository.Queryer
}

func newMCPSessionListAdapter(db repository.Queryer) mcp.SessionListService {
	if db == nil {
		return nil
	}
	return mcpSessionListAdapter{db: db}
}

func newMCPSessionGetAdapter(db repository.Queryer) mcp.SessionGetService {
	if db == nil {
		return nil
	}
	return mcpSessionListAdapter{db: db}
}

func newMCPMessageListAdapter(db repository.Queryer) mcp.MessageListService {
	if db == nil {
		return nil
	}
	return mcpSessionListAdapter{db: db}
}

func (adapter mcpSessionListAdapter) ListSessions(
	ctx context.Context,
	request mcp.SessionListRequest,
) ([]models.ChatSessionRecord, error) {
	return repository.ListChatSessions(
		ctx,
		adapter.db,
		repository.ChatSessionListInput{
			ActorUserID: request.ActorUserID,
			ProjectID:   request.ProjectID,
			Limit:       request.Limit,
			Offset:      request.Offset,
		},
	)
}

func (adapter mcpSessionListAdapter) GetSession(
	ctx context.Context,
	request mcp.SessionGetRequest,
) (*models.ChatSessionRecord, error) {
	return repository.GetChatSession(
		ctx,
		adapter.db,
		repository.ChatSessionGetInput{
			SessionID:   request.SessionID,
			ActorUserID: request.ActorUserID,
		},
	)
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
