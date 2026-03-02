package mcp

func parseSessionMessageSendRequest(
	actor Actor,
	params map[string]any,
) (SessionMessageSendRequest, *toolDispatchError) {
	sessionID, ok := requiredUUIDParam(params, "session_id")
	if !ok {
		return SessionMessageSendRequest{}, invalidParamError("session_id")
	}
	linkRecallEnabled, ok := optionalBoolPointerParam(params, "link_recall_enabled")
	if !ok {
		return SessionMessageSendRequest{}, invalidParamError("link_recall_enabled")
	}
	linkRecallDepth, ok := optionalIntPointerParam(params, "link_recall_depth")
	if !ok {
		return SessionMessageSendRequest{}, invalidParamError("link_recall_depth")
	}
	linkRecallMaxNeighbors, ok := optionalIntPointerParam(params, "link_recall_max_neighbors")
	if !ok {
		return SessionMessageSendRequest{}, invalidParamError("link_recall_max_neighbors")
	}
	linkNoiseSuppressionEnabled, ok := optionalBoolPointerParam(params, "link_noise_suppression_enabled")
	if !ok {
		return SessionMessageSendRequest{}, invalidParamError("link_noise_suppression_enabled")
	}
	linkNoiseScoreThreshold, ok := optionalBoundedFloatPointer(params, "link_noise_score_threshold")
	if !ok {
		return SessionMessageSendRequest{}, invalidParamError("link_noise_score_threshold")
	}
	return SessionMessageSendRequest{
		ActorUserID:                 actor.UserID,
		SessionID:                   sessionID,
		ContentText:                 stringParamWithDefault(params, "content_text", ""),
		LinkRecallEnabled:           linkRecallEnabled,
		LinkRecallDepth:             linkRecallDepth,
		LinkRecallMaxNeighbors:      linkRecallMaxNeighbors,
		LinkNoiseSuppressionEnabled: linkNoiseSuppressionEnabled,
		LinkNoiseScoreThreshold:     linkNoiseScoreThreshold,
	}, nil
}
