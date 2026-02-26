package main

import (
	"context"

	"engram/internal/chat"
	"engram/internal/mcp"
	"engram/internal/models"
	"engram/internal/repository"
)

type mcpSessionListAdapter struct {
	db repository.Queryer
}

func newMCPSessionListAdapter(db repository.Queryer) mcp.SessionListService {
	if db == nil {
		return nil
	}
	return mcpSessionListAdapter{db: db}
}

func newMCPSessionGetAdapter(db repository.Queryer) mcp.SessionGetService {
	if db == nil {
		return nil
	}
	return mcpSessionListAdapter{db: db}
}

func newMCPTimelineListAdapter(db repository.Queryer) mcp.TimelineListService {
	if db == nil {
		return nil
	}
	return mcpSessionListAdapter{db: db}
}

func (adapter mcpSessionListAdapter) ListSessions(
	ctx context.Context,
	request mcp.SessionListRequest,
) ([]models.ChatSessionRecord, error) {
	return repository.ListChatSessions(
		ctx,
		adapter.db,
		repository.ChatSessionListInput{
			ActorUserID: request.ActorUserID,
			ProjectID:   request.ProjectID,
			Limit:       request.Limit,
			Offset:      request.Offset,
		},
	)
}

func (adapter mcpSessionListAdapter) GetSession(
	ctx context.Context,
	request mcp.SessionGetRequest,
) (*models.ChatSessionRecord, error) {
	return repository.GetChatSession(
		ctx,
		adapter.db,
		repository.ChatSessionGetInput{
			SessionID:   request.SessionID,
			ActorUserID: request.ActorUserID,
		},
	)
}

func (adapter mcpSessionListAdapter) ListTimeline(
	ctx context.Context,
	request mcp.TimelineListRequest,
) ([]models.ChatTimelineEvent, error) {
	linked, err := repository.ListSessionLinkedEngrams(
		ctx,
		adapter.db,
		repository.SessionLinkedEngramsListInput{
			SessionID:   request.SessionID,
			ActorUserID: request.ActorUserID,
			Limit:       request.Limit,
			Offset:      request.Offset,
		},
	)
	if err != nil {
		return nil, err
	}
	events := make([]models.ChatTimelineEvent, 0, len(linked))
	for _, item := range linked {
		semantics := chat.ClassifyTimelineEvent(item.Tags)
		events = append(events, models.ChatTimelineEvent{
			EventID:                  item.EngramID,
			SessionID:                request.SessionID,
			EventType:                semantics.EventType,
			Title:                    item.Title,
			Abstract:                 item.Abstract,
			Tags:                     append([]string(nil), item.Tags...),
			ConsolidationGroupKey:    semantics.ConsolidationGroupKey,
			ConsolidationMergedCount: semantics.ConsolidationMergedCount,
			CreatedAt:                item.CreatedAt,
		})
	}
	return events, nil
}
