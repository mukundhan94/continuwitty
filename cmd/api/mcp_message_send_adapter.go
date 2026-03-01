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

func newMCPMessageAdapter(
	settings config.Settings,
	pool *pgxpool.Pool,
	observability chat.ObservabilityRecorder,
) *mcpMessageSendAdapter {
	if pool == nil {
		return nil
	}
	service := buildChatService(settings, pool, observability)
	return &mcpMessageSendAdapter{service: service}
}

func (adapter *mcpMessageSendAdapter) SendMessage(
	ctx context.Context,
	request mcp.SessionMessageSendRequest,
) (*mcp.MessageSendResponse, error) {
	if adapter == nil || adapter.service == nil {
		return nil, nil
	}
	result, err := adapter.service.SendMessage(
		ctx,
		request.ActorUserID,
		request.SessionID,
		chat.ChatMessageCreateRequest{
			ContentText:                 request.ContentText,
			LinkRecallEnabled:           request.LinkRecallEnabled,
			LinkRecallDepth:             request.LinkRecallDepth,
			LinkRecallMaxNeighbors:      request.LinkRecallMaxNeighbors,
			LinkNoiseSuppressionEnabled: request.LinkNoiseSuppressionEnabled,
			LinkNoiseScoreThreshold:     request.LinkNoiseScoreThreshold,
		},
	)
	if err != nil {
		return nil, err
	}
	return &mcp.MessageSendResponse{
		SessionID:            result.SessionID,
		MessageID:            result.MessageID,
		ReplyMessageID:       result.ReplyMessageID,
		AssistantText:        result.AssistantText,
		PromptPolicyVersion:  result.PromptPolicyVersion,
		UsedEngramIDs:        append([]uuid.UUID(nil), result.UsedEngramIDs...),
		UsedEngramLinkIDs:    append([]uuid.UUID(nil), result.UsedEngramLinkIDs...),
		EngramTracePaths:     result.EngramTracePaths,
		UsedDocumentChunkIDs: append([]uuid.UUID(nil), result.UsedDocumentChunkIDs...),
		SourceReferences:     result.SourceReferences,
		DebugTrace:           mapStringAnyCopy(result.DebugTrace),
	}, nil
}

func (adapter *mcpMessageSendAdapter) StreamMessageEvents(
	ctx context.Context,
	request mcp.SessionMessageSendRequest,
) ([]mcp.MessageStreamEvent, error) {
	if adapter == nil || adapter.service == nil {
		return nil, nil
	}
	events, err := adapter.service.StreamMessageEvents(
		ctx,
		request.ActorUserID,
		request.SessionID,
		chat.ChatMessageCreateRequest{
			ContentText:                 request.ContentText,
			LinkRecallEnabled:           request.LinkRecallEnabled,
			LinkRecallDepth:             request.LinkRecallDepth,
			LinkRecallMaxNeighbors:      request.LinkRecallMaxNeighbors,
			LinkNoiseSuppressionEnabled: request.LinkNoiseSuppressionEnabled,
			LinkNoiseScoreThreshold:     request.LinkNoiseScoreThreshold,
		},
	)
	if err != nil {
		return nil, err
	}
	converted := make([]mcp.MessageStreamEvent, 0, len(events))
	for _, event := range events {
		converted = append(
			converted,
			mcp.MessageStreamEvent{
				Type:    event.Type,
				Payload: mapStringAnyCopy(event.Payload),
			},
		)
	}
	return converted, nil
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
