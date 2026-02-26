package main

import (
	"context"

	"engram/internal/mcp"
	"engram/internal/models"
	"engram/internal/repository"
)

func newMCPProjectDocumentListAdapter(db repository.Queryer) mcp.ProjectDocumentListService {
	if db == nil {
		return nil
	}
	return mcpSessionListAdapter{db: db}
}

func (adapter mcpSessionListAdapter) ListProjectDocuments(
	ctx context.Context,
	request mcp.ProjectDocumentListRequest,
) ([]models.DocumentRecord, error) {
	return repository.ListDocuments(
		ctx,
		adapter.db,
		repository.DocumentListInput{
			ActorUserID: request.ActorUserID,
			ProjectID:   request.ProjectID,
			Limit:       request.Limit,
			Offset:      request.Offset,
		},
	)
}
