package main

import (
	"context"

	"engram/internal/mcp"
	"engram/internal/repository"
)

func newMCPUnpinDocumentAdapter(db repository.Queryer) mcp.UnpinDocumentService {
	if db == nil {
		return nil
	}
	return mcpSessionListAdapter{db: db}
}

func (adapter mcpSessionListAdapter) UnpinDocument(
	ctx context.Context,
	request mcp.SessionPinDocumentRequest,
) (bool, error) {
	return repository.UnpinDocumentFromSession(
		ctx,
		adapter.db,
		repository.ChatPinDocumentInput{
			SessionID:   request.SessionID,
			DocumentID:  request.DocumentID,
			ActorUserID: request.ActorUserID,
		},
	)
}
