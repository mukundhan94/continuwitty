package main

import (
	"context"

	"engram/internal/chat"
	"engram/internal/config"
	"engram/internal/mcp"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type mcpMessageSendAdapter struct {
	service *chat.ChatService
}

func newMCPMessageSendAdapter(
	settings config.Settings,
	pool *pgxpool.Pool,
) mcp.MessageSendService {
	if pool == nil {
		return nil
	}
	service := chat.NewChatService(chat.ChatServiceDependencies{
		Runtime:                        chat.NewChatMessageRuntime(buildChatMessageRuntimeDependencies(settings, pool)),
		ResolveProvider:                resolveChatProviderDependency(settings),
		RunSessionLifecycleMaintenance: runSessionLifecycleMaintenanceDependency(settings, pool),
	})
	return mcpMessageSendAdapter{service: service}
}

func (adapter mcpMessageSendAdapter) SendMessage(
	ctx context.Context,
	request mcp.SessionMessageSendRequest,
) (*mcp.MessageSendResponse, error) {
	if adapter.service == nil {
		return nil, nil
	}
	result, err := adapter.service.SendMessage(
		ctx,
		request.ActorUserID,
		request.SessionID,
		chat.ChatMessageCreateRequest{ContentText: request.ContentText},
	)
	if err != nil {
		return nil, err
	}
	return &mcp.MessageSendResponse{
		SessionID:            result.SessionID,
		MessageID:            result.MessageID,
		ReplyMessageID:       result.ReplyMessageID,
		AssistantText:        result.AssistantText,
		UsedEngramIDs:        append([]uuid.UUID(nil), result.UsedEngramIDs...),
		UsedDocumentChunkIDs: append([]uuid.UUID(nil), result.UsedDocumentChunkIDs...),
		SourceReferences:     result.SourceReferences,
		DebugTrace:           mapStringAnyCopy(result.DebugTrace),
	}, nil
}

func mapStringAnyCopy(input map[string]any) map[string]any {
	if input == nil {
		return nil
	}
	copied := make(map[string]any, len(input))
	for key, value := range input {
		copied[key] = value
	}
	return copied
}
