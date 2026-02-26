package main

import (
	"context"

	"engram/internal/mcp"
	"engram/internal/models"
	"engram/internal/repository"

	"github.com/google/uuid"
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

func (adapter mcpSessionListAdapter) ListSessions(
	ctx context.Context,
	actorUserID uuid.UUID,
	limit int,
	offset int,
) ([]models.ChatSessionRecord, error) {
	return repository.ListChatSessions(
		ctx,
		adapter.db,
		repository.ChatSessionListInput{
			ActorUserID: actorUserID,
			ProjectID:   nil,
			Limit:       limit,
			Offset:      offset,
		},
	)
}
