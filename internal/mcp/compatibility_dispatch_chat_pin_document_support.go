package mcp

import "context"

func (service *CompatibilityService) dispatchChatPinDocumentTool(
	ctx context.Context,
	actor Actor,
	params map[string]any,
) (map[string]any, bool, *toolDispatchError) {
	if service.pinDocumentService == nil {
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
	pinned, err := service.pinDocumentService.PinDocument(
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
	if pinned == nil {
		return nil, true, invalidParamsWithStatus(404, "Document not found or session inaccessible")
	}
	return map[string]any{"pinned": *pinned}, true, nil
}
