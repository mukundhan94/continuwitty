package mcp

import "context"

func (service *CompatibilityService) dispatchChatListProjectDocumentsTool(
	ctx context.Context,
	actor Actor,
	params map[string]any,
) (map[string]any, bool, *toolDispatchError) {
	if service.projectDocumentService == nil {
		return nil, false, nil
	}
	paging, pagingErr := parsePagingParams(
		params,
		defaultChatProjectDocumentsLimit,
		defaultChatProjectDocumentsOffset,
	)
	if pagingErr != nil {
		return nil, true, pagingErr
	}
	documents, err := service.projectDocumentService.ListProjectDocuments(
		ctx,
		ProjectDocumentListRequest{
			ActorUserID: actor.UserID,
			ProjectID:   optionalProjectIDParam(params, "project_id"),
			Limit:       paging.limit,
			Offset:      paging.offset,
		},
	)
	if err != nil {
		return nil, true, internalToolDispatchError()
	}
	return map[string]any{"documents": documents}, true, nil
}
