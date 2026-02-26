package main

import (
	"context"
	"errors"

	"engram/internal/admin"
	"engram/internal/mcp"
)

type mcpSessionDeleteAdapter struct {
	memoryAdminService *admin.Service
}

func newMCPSessionDeleteAdapter(memoryAdminService *admin.Service) mcp.SessionDeleteService {
	if memoryAdminService == nil {
		return nil
	}
	return mcpSessionDeleteAdapter{memoryAdminService: memoryAdminService}
}

func (adapter mcpSessionDeleteAdapter) DeleteSession(
	ctx context.Context,
	request mcp.SessionDeleteRequest,
) (*mcp.SessionDeleteResponse, error) {
	deleted, err := adapter.memoryAdminService.DeleteSession(
		ctx,
		request.SessionID,
		request.ActorUserID,
		admin.SessionDeleteRequest{
			DeleteLinkedEngrams: request.DeleteLinkedEngrams,
			Reason:              request.Reason,
		},
	)
	if err != nil {
		if errors.Is(err, admin.ErrSessionNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &mcp.SessionDeleteResponse{
		SessionID:            deleted.SessionID,
		Deleted:              deleted.Deleted,
		LinkedEngramsDeleted: deleted.LinkedEngramsDeleted,
	}, nil
}
