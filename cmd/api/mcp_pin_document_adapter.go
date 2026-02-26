package main

import (
	"context"

	"engram/internal/mcp"
	"engram/internal/models"
	"engram/internal/repository"
)

func newMCPPinDocumentAdapter(db repository.Queryer) mcp.PinDocumentService {
	if db == nil {
		return nil
	}
	return mcpSessionListAdapter{db: db}
}

func (adapter mcpSessionListAdapter) PinDocument(
	ctx context.Context,
	request mcp.SessionPinDocumentRequest,
) (*models.PinnedDocumentRecord, error) {
	return repository.PinDocumentToSession(
		ctx,
		adapter.db,
		repository.ChatPinDocumentInput{
			SessionID:   request.SessionID,
			DocumentID:  request.DocumentID,
			ActorUserID: request.ActorUserID,
		},
	)
}
