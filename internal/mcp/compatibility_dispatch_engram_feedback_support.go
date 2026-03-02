package mcp

import (
	"context"

	"engram/internal/models"
)

func (service *CompatibilityService) dispatchEngramFeedbackTool(
	ctx context.Context,
	actor Actor,
	params map[string]any,
) (map[string]any, bool, *toolDispatchError) {
	if service.engramFeedback == nil {
		return nil, false, nil
	}
	request, dispatchErr := parseEngramFeedbackRequest(actor, params)
	if dispatchErr != nil {
		return nil, true, dispatchErr
	}
	record, err := service.engramFeedback.SubmitEngramFeedback(ctx, request)
	if err != nil {
		return nil, true, internalToolDispatchError()
	}
	if record == nil {
		return nil, true, engramNotFoundDispatchError(request.EngramID)
	}
	return map[string]any{"feedback": *record}, true, nil
}

func parseEngramFeedbackRequest(
	actor Actor,
	params map[string]any,
) (EngramFeedbackRequest, *toolDispatchError) {
	engramID, ok := requiredUUIDParam(params, "engram_id")
	if !ok {
		return EngramFeedbackRequest{}, invalidParamError("engram_id")
	}
	feedbackTypeRaw, ok := requiredStringParam(params, "feedback_type")
	if !ok {
		return EngramFeedbackRequest{}, invalidParamError("feedback_type")
	}
	feedbackType, err := models.ParseEngramFeedbackType(feedbackTypeRaw)
	if err != nil {
		return EngramFeedbackRequest{}, invalidParamError("feedback_type")
	}
	note, ok := optionalStringPointerParam(params, "note")
	if !ok {
		return EngramFeedbackRequest{}, invalidParamError("note")
	}
	return EngramFeedbackRequest{
		ActorUserID:  actor.UserID,
		EngramID:     engramID,
		FeedbackType: feedbackType,
		Note:         note,
	}, nil
}
