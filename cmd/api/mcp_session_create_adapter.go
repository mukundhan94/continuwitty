package main

import (
	"context"

	"engram/internal/chat"
	"engram/internal/mcp"
	"engram/internal/models"
	"engram/internal/repository"
)

func newMCPSessionCreateAdapter(db repository.Queryer) mcp.SessionCreateService {
	if db == nil {
		return nil
	}
	return mcpSessionListAdapter{db: db}
}

func (adapter mcpSessionListAdapter) CreateSession(
	ctx context.Context,
	request mcp.SessionCreateRequest,
) (*models.ChatSessionRecord, error) {
	normalizedPayload := chat.NormalizeSessionCreatePayload(request.Payload)
	if _, err := repository.EnsureProjectExists(
		ctx,
		adapter.db,
		repository.ProjectEnsureInput{
			ProjectID:   normalizedPayload.ProjectID,
			OwnerUserID: request.ActorUserID,
		},
	); err != nil {
		return nil, err
	}
	return repository.CreateChatSession(
		ctx,
		adapter.db,
		repository.ChatSessionCreateInput{
			OwnerUserID: request.ActorUserID,
			Payload:     normalizedPayload,
		},
	)
}
