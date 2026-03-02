package mcp

import "context"

func (service *CompatibilityService) dispatchUserListProjectsTool(
	ctx context.Context,
	actor Actor,
) (map[string]any, bool, *toolDispatchError) {
	if service.sessionService == nil {
		return nil, false, nil
	}
	sessions, err := service.sessionService.ListSessions(
		ctx,
		SessionListRequest{
			ActorUserID: actor.UserID,
			ProjectID:   nil,
			Limit:       defaultUserProjectsLimit,
			Offset:      defaultUserProjectsOffset,
		},
	)
	if err != nil {
		return nil, true, internalToolDispatchError()
	}
	return map[string]any{
		"project_ids": uniqueSortedProjectIDs(sessions),
	}, true, nil
}

func (service *CompatibilityService) dispatchChatListSessionsTool(
	ctx context.Context,
	actor Actor,
	params map[string]any,
) (map[string]any, bool, *toolDispatchError) {
	if service.sessionService == nil {
		return nil, false, nil
	}
	paging, pagingErr := parsePagingParams(
		params,
		defaultChatSessionsLimit,
		defaultChatSessionsOffset,
	)
	if pagingErr != nil {
		return nil, true, pagingErr
	}
	sessions, err := service.sessionService.ListSessions(
		ctx,
		SessionListRequest{
			ActorUserID: actor.UserID,
			ProjectID:   optionalProjectIDParam(params, "project_id"),
			Limit:       paging.limit,
			Offset:      paging.offset,
		},
	)
	if err != nil {
		return nil, true, internalToolDispatchError()
	}
	return map[string]any{"sessions": sessions}, true, nil
}
