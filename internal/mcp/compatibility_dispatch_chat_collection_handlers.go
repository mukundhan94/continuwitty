package mcp

import (
	"context"

	"github.com/google/uuid"
)

func registerChatCollectionToolHandlers(handlers map[string]implementedToolHandler) {
	handlers["chat.list_messages"] = chatListMessagesHandler()
	handlers["chat.list_timeline"] = chatListTimelineHandler()
	handlers["chat.list_pinned_engrams"] = chatListPinnedEngramsHandler()
	handlers["chat.list_pinned_documents"] = chatListPinnedDocumentsHandler()
}

func chatListMessagesHandler() implementedToolHandler {
	return func(
		service *CompatibilityService,
		ctx context.Context,
		actor Actor,
		params map[string]any,
	) (map[string]any, bool, *toolDispatchError) {
		if service.messageService == nil {
			return nil, false, nil
		}
		return service.dispatchSessionCollectionTool(
			ctx,
			actor,
			params,
			defaultChatMessagesLimit,
			defaultChatMessagesOffset,
			"messages",
			func(
				ctx context.Context,
				actorUserID uuid.UUID,
				sessionID uuid.UUID,
				paging pagingParams,
			) (any, error) {
				return service.messageService.ListMessages(
					ctx,
					MessageListRequest{
						ActorUserID: actorUserID,
						SessionID:   sessionID,
						Limit:       paging.limit,
						Offset:      paging.offset,
					},
				)
			},
		)
	}
}

func chatListTimelineHandler() implementedToolHandler {
	return func(
		service *CompatibilityService,
		ctx context.Context,
		actor Actor,
		params map[string]any,
	) (map[string]any, bool, *toolDispatchError) {
		if service.timelineService == nil {
			return nil, false, nil
		}
		return service.dispatchSessionCollectionTool(
			ctx,
			actor,
			params,
			defaultChatTimelineLimit,
			defaultChatTimelineOffset,
			"events",
			func(
				ctx context.Context,
				actorUserID uuid.UUID,
				sessionID uuid.UUID,
				paging pagingParams,
			) (any, error) {
				return service.timelineService.ListTimeline(
					ctx,
					TimelineListRequest{
						ActorUserID: actorUserID,
						SessionID:   sessionID,
						Limit:       paging.limit,
						Offset:      paging.offset,
					},
				)
			},
		)
	}
}

func chatListPinnedEngramsHandler() implementedToolHandler {
	return func(
		service *CompatibilityService,
		ctx context.Context,
		actor Actor,
		params map[string]any,
	) (map[string]any, bool, *toolDispatchError) {
		if service.pinnedEngramService == nil {
			return nil, false, nil
		}
		return service.dispatchSessionScopedCollectionTool(
			ctx,
			actor,
			params,
			"pinned_engrams",
			func(
				ctx context.Context,
				actorUserID uuid.UUID,
				sessionID uuid.UUID,
			) (any, error) {
				return service.pinnedEngramService.ListPinnedEngrams(
					ctx,
					SessionScopedRequest{
						ActorUserID: actorUserID,
						SessionID:   sessionID,
					},
				)
			},
		)
	}
}

func chatListPinnedDocumentsHandler() implementedToolHandler {
	return func(
		service *CompatibilityService,
		ctx context.Context,
		actor Actor,
		params map[string]any,
	) (map[string]any, bool, *toolDispatchError) {
		if service.pinnedDocumentService == nil {
			return nil, false, nil
		}
		return service.dispatchSessionScopedCollectionTool(
			ctx,
			actor,
			params,
			"pinned_documents",
			func(
				ctx context.Context,
				actorUserID uuid.UUID,
				sessionID uuid.UUID,
			) (any, error) {
				return service.pinnedDocumentService.ListPinnedDocuments(
					ctx,
					SessionScopedRequest{
						ActorUserID: actorUserID,
						SessionID:   sessionID,
					},
				)
			},
		)
	}
}
