package mcp

import (
	"context"

	"github.com/google/uuid"
)

func (service *CompatibilityService) dispatchSessionCollectionTool(
	ctx context.Context,
	actor Actor,
	params map[string]any,
	defaultLimit int,
	defaultOffset int,
	collectionKey string,
	collect func(
		ctx context.Context,
		actorUserID uuid.UUID,
		sessionID uuid.UUID,
		paging pagingParams,
	) (any, error),
) (map[string]any, bool, *toolDispatchError) {
	session, handled, dispatchErr := service.lookupChatSession(ctx, actor, params)
	if dispatchErr != nil || !handled {
		return nil, handled, dispatchErr
	}
	paging, pagingErr := parsePagingParams(params, defaultLimit, defaultOffset)
	if pagingErr != nil {
		return nil, true, pagingErr
	}
	items, err := collect(ctx, actor.UserID, session.SessionID, paging)
	if err != nil {
		return nil, true, internalToolDispatchError()
	}
	return map[string]any{collectionKey: items}, true, nil
}

func (service *CompatibilityService) dispatchSessionScopedCollectionTool(
	ctx context.Context,
	actor Actor,
	params map[string]any,
	collectionKey string,
	collect func(
		ctx context.Context,
		actorUserID uuid.UUID,
		sessionID uuid.UUID,
	) (any, error),
) (map[string]any, bool, *toolDispatchError) {
	session, handled, dispatchErr := service.lookupChatSession(ctx, actor, params)
	if dispatchErr != nil || !handled {
		return nil, handled, dispatchErr
	}
	items, err := collect(ctx, actor.UserID, session.SessionID)
	if err != nil {
		return nil, true, internalToolDispatchError()
	}
	return map[string]any{collectionKey: items}, true, nil
}
