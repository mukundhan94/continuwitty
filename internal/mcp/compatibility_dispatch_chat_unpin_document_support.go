package mcp

import "context"

func (service *CompatibilityService) dispatchChatUnpinDocumentTool(
	ctx context.Context,
	actor Actor,
	params map[string]any,
) (map[string]any, bool, *toolDispatchError) {
	if service.unpinDocumentService == nil {
		return nil, false, nil
	}
	sessionID, ok := requiredUUIDParam(params, "session_id")
	if !ok {
		return nil, true, invalidParamError("session_id")
	}
	documentID, ok := requiredUUIDParam(params, "document_id")
	if !ok {
		return nil, true, invalidParamError("document_id")
	}
	removed, err := service.unpinDocumentService.UnpinDocument(
		ctx,
		SessionPinDocumentRequest{
			ActorUserID: actor.UserID,
			SessionID:   sessionID,
			DocumentID:  documentID,
		},
	)
	if err != nil {
		return nil, true, internalToolDispatchError()
	}
	if !removed {
		return nil, true, invalidParamsWithStatus(404, "Pinned document not found for session")
	}
	return map[string]any{"removed": true}, true, nil
}
