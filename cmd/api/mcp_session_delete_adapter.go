package main

import (
	"context"
	"errors"

	"engram/internal/admin"
	"engram/internal/mcp"
	"engram/internal/models"

	"github.com/google/uuid"
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

func newMCPSessionRestoreAdapter(memoryAdminService *admin.Service) mcp.SessionRestoreService {
	if memoryAdminService == nil {
		return nil
	}
	return mcpSessionDeleteAdapter{memoryAdminService: memoryAdminService}
}

func (adapter mcpSessionDeleteAdapter) DeleteSession(
	ctx context.Context,
	request mcp.SessionDeleteRequest,
) (*mcp.SessionDeleteResponse, error) {
	accessible, err := adapter.accessibleSession(
		ctx,
		request.ActorUserID,
		request.ActorRole,
		request.SessionID,
	)
	if err != nil {
		return nil, err
	}
	if accessible == nil {
		return nil, nil
	}
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

func (adapter mcpSessionDeleteAdapter) RestoreSession(
	ctx context.Context,
	request mcp.SessionRestoreRequest,
) (*mcp.SessionRestoreResponse, error) {
	accessible, err := adapter.accessibleSession(
		ctx,
		request.ActorUserID,
		request.ActorRole,
		request.SessionID,
	)
	if err != nil {
		return nil, err
	}
	if accessible == nil {
		return nil, nil
	}
	restored, err := adapter.memoryAdminService.RestoreSession(ctx, request.SessionID)
	if err != nil {
		if errors.Is(err, admin.ErrSessionNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &mcp.SessionRestoreResponse{
		SessionID: restored.SessionID,
		Restored:  restored.Restored,
	}, nil
}

func (adapter mcpSessionDeleteAdapter) accessibleSession(
	ctx context.Context,
	actorUserID uuid.UUID,
	actorRole models.UserRole,
	sessionID uuid.UUID,
) (*models.AdminChatSessionRecord, error) {
	session, err := adapter.memoryAdminService.GetSession(ctx, sessionID, true)
	if err != nil {
		return nil, err
	}
	if session == nil {
		return nil, nil
	}
	if actorRole != models.UserRoleAdmin && session.OwnerUserID != actorUserID {
		return nil, nil
	}
	return session, nil
}
