package mcp

import (
	"context"
	"strings"

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
	sessionID, ok := optionalUUIDParam(params, "session_id")
	if !ok {
		return EngramFeedbackRequest{}, invalidParamError("session_id")
	}
	note, ok := optionalStringPointerParam(params, "note")
	if !ok {
		return EngramFeedbackRequest{}, invalidParamError("note")
	}
	relevanceScore, ok := optionalFeedbackRelevanceScoreParam(params, "relevance_score")
	if !ok {
		return EngramFeedbackRequest{}, invalidParamError("relevance_score")
	}
	integrationDepth, ok := optionalFeedbackIntegrationDepthParam(params, "integration_depth")
	if !ok {
		return EngramFeedbackRequest{}, invalidParamError("integration_depth")
	}
	return EngramFeedbackRequest{
		ActorUserID:      actor.UserID,
		EngramID:         engramID,
		SessionID:        sessionID,
		FeedbackType:     feedbackType,
		IntegrationDepth: integrationDepth,
		Note:             normalizeOptionalTrimmedParamString(note),
		RelevanceScore:   relevanceScore,
	}, nil
}

func optionalFeedbackRelevanceScoreParam(
	params map[string]any,
	key string,
) (*int, bool) {
	rawValue, found := optionalParamValue(params, key)
	if !found {
		return nil, true
	}
	parsed, ok := parseIntValue(rawValue)
	if !ok {
		return nil, false
	}
	if parsed < 1 || parsed > 5 {
		return nil, false
	}
	return &parsed, true
}

func optionalFeedbackIntegrationDepthParam(
	params map[string]any,
	key string,
) (*models.EngramFeedbackIntegrationDepth, bool) {
	rawValue, found := optionalStringPointerParam(params, key)
	if !found {
		return nil, true
	}
	if rawValue == nil {
		return nil, true
	}
	parsed, err := models.ParseEngramFeedbackIntegrationDepth(*rawValue)
	if err != nil {
		return nil, false
	}
	normalized := parsed
	return &normalized, true
}

func normalizeOptionalTrimmedParamString(value *string) *string {
	if value == nil {
		return nil
	}
	trimmed := strings.TrimSpace(*value)
	if trimmed == "" {
		return nil
	}
	return &trimmed
}
