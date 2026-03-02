package main

import (
	"context"
	"errors"

	"engram/internal/chat"
	"engram/internal/mcp"
	"engram/internal/models"
	"engram/internal/repository"
)

func newMCPSessionContinueAdapter(db repository.Queryer) mcp.SessionContinueService {
	if db == nil {
		return nil
	}
	return mcpSessionListAdapter{db: db}
}

func (adapter mcpSessionListAdapter) ContinueSession(
	ctx context.Context,
	request mcp.SessionContinueRequest,
) (*models.ContinueSessionResponse, error) {
	continued, err := chat.NewSessionOperationsService(adapter.db).ContinueSession(
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
	return &continued, nil
}

func isChatSessionNotFoundError(err error) bool {
	var serviceErr *chat.ChatServiceError
	if !errors.As(err, &serviceErr) {
		return false
	}
	return serviceErr.StatusCode() == 404
}
