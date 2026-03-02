package main

import (
	"context"

	"engram/internal/mcp"
	"engram/internal/models"
	"engram/internal/repository"
)

func newMCPPinnedDocumentListAdapter(db repository.Queryer) mcp.PinnedDocumentListService {
	if db == nil {
		return nil
	}
	return mcpSessionListAdapter{db: db}
}

func (adapter mcpSessionListAdapter) ListPinnedDocuments(
	ctx context.Context,
	request mcp.SessionScopedRequest,
) ([]models.PinnedDocumentRecord, error) {
	return repository.ListPinnedDocuments(
		ctx,
		adapter.db,
		repository.ChatPinnedListInput{
			SessionID:   request.SessionID,
			ActorUserID: request.ActorUserID,
		},
	)
}
