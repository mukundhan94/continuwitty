package main

import (
	"context"

	"engram/internal/mcp"
	"engram/internal/models"
	"engram/internal/repository"
)

func newMCPLifecyclePolicyUpdateAdapter(db repository.Queryer) mcp.LifecyclePolicyUpdateService {
	if db == nil {
		return nil
	}
	return mcpSessionListAdapter{db: db}
}

func (adapter mcpSessionListAdapter) UpdateLifecyclePolicy(
	ctx context.Context,
	request mcp.SessionLifecyclePolicyUpdateRequest,
) (*models.ChatSessionRecord, error) {
	if lifecycleUpdateRequestIsEmpty(request) {
		return repository.GetChatSession(
			ctx,
			adapter.db,
			repository.ChatSessionGetInput{
				SessionID:   request.SessionID,
				ActorUserID: request.ActorUserID,
			},
		)
	}
	return repository.UpdateChatSession(
		ctx,
		adapter.db,
		repository.ChatSessionUpdateInput{
			SessionID:   request.SessionID,
			ActorUserID: request.ActorUserID,
			Payload: models.ChatSessionUpdateRequest{
				AutosaveEnabled:         request.AutosaveEnabled,
				AutosaveStrategy:        request.AutosaveStrategy,
				AutosaveIntervalMinutes: request.AutosaveIntervalMinutes,
				AutosaveMinMessages:     request.AutosaveMinMessages,
				RetentionDays:           request.RetentionDays,
				RetentionMaxSnapshots:   request.RetentionMaxSnapshots,
			},
		},
	)
}

func lifecycleUpdateRequestIsEmpty(request mcp.SessionLifecyclePolicyUpdateRequest) bool {
	switch {
	case request.AutosaveEnabled != nil:
		return false
	case request.AutosaveStrategy != nil:
		return false
	case request.AutosaveIntervalMinutes != nil:
		return false
	case request.AutosaveMinMessages != nil:
		return false
	case request.RetentionDays != nil:
		return false
	case request.RetentionMaxSnapshots != nil:
		return false
	default:
		return true
	}
}
