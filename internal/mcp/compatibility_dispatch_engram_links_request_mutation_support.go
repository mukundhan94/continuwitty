package mcp

import (
	"time"

	"engram/internal/models"

	"github.com/google/uuid"
)

type engramLinkCreateDescriptors struct {
	relationType models.EngramLinkRelationType
	origin       models.EngramLinkOrigin
	status       models.EngramLinkStatus
}

type engramLinkCreateWeights struct {
	weight         float64
	temporalWeight float64
	confidence     float64
}

type engramLinkCreateMetadata struct {
	evidenceJSON     map[string]any
	lastReinforcedAt *time.Time
}

func parseEngramLinkCreateRequest(
	actor Actor,
	params map[string]any,
) (EngramLinkCreateRequest, *toolDispatchError) {
	sourceEngramID, targetEngramID, dispatchErr := parseEngramLinkCreateEndpoints(params)
	if dispatchErr != nil {
		return EngramLinkCreateRequest{}, dispatchErr
	}
	descriptors, dispatchErr := parseEngramLinkCreateDescriptors(params)
	if dispatchErr != nil {
		return EngramLinkCreateRequest{}, dispatchErr
	}
	weights, dispatchErr := parseEngramLinkCreateWeights(params)
	if dispatchErr != nil {
		return EngramLinkCreateRequest{}, dispatchErr
	}
	metadata, dispatchErr := parseEngramLinkCreateMetadata(params)
	if dispatchErr != nil {
		return EngramLinkCreateRequest{}, dispatchErr
	}
	return EngramLinkCreateRequest{
		ActorUserID:      actor.UserID,
		SourceEngramID:   sourceEngramID,
		TargetEngramID:   targetEngramID,
		RelationType:     descriptors.relationType,
		Weight:           weights.weight,
		TemporalWeight:   weights.temporalWeight,
		Confidence:       weights.confidence,
		Origin:           descriptors.origin,
		Status:           descriptors.status,
		EvidenceJSON:     metadata.evidenceJSON,
		LastReinforcedAt: metadata.lastReinforcedAt,
	}, nil
}

func parseEngramLinkCreateEndpoints(params map[string]any) (uuid.UUID, uuid.UUID, *toolDispatchError) {
	sourceEngramID, ok := requiredUUIDParam(params, "engram_id")
	if !ok {
		return uuid.Nil, uuid.Nil, invalidParamError("engram_id")
	}
	targetEngramID, ok := requiredUUIDParam(params, "target_engram_id")
	if !ok {
		return uuid.Nil, uuid.Nil, invalidParamError("target_engram_id")
	}
	return sourceEngramID, targetEngramID, nil
}

func parseEngramLinkCreateDescriptors(
	params map[string]any,
) (engramLinkCreateDescriptors, *toolDispatchError) {
	relationType, dispatchErr := parseOptionalEngramLinkEnumValue(
		params,
		"relation_type",
		models.EngramLinkRelationRelatedTo,
		models.ParseEngramLinkRelationType,
	)
	if dispatchErr != nil {
		return engramLinkCreateDescriptors{}, dispatchErr
	}
	origin, dispatchErr := parseOptionalEngramLinkEnumValue(
		params,
		"origin",
		models.EngramLinkOriginManual,
		models.ParseEngramLinkOrigin,
	)
	if dispatchErr != nil {
		return engramLinkCreateDescriptors{}, dispatchErr
	}
	status, dispatchErr := parseOptionalEngramLinkEnumValue(
		params,
		"status",
		models.EngramLinkStatusActive,
		models.ParseEngramLinkStatus,
	)
	if dispatchErr != nil {
		return engramLinkCreateDescriptors{}, dispatchErr
	}
	return engramLinkCreateDescriptors{
		relationType: relationType,
		origin:       origin,
		status:       status,
	}, nil
}

func parseEngramLinkCreateWeights(
	params map[string]any,
) (engramLinkCreateWeights, *toolDispatchError) {
	weight, dispatchErr := parseBoundedFloat(params, "weight", 0.6)
	if dispatchErr != nil {
		return engramLinkCreateWeights{}, dispatchErr
	}
	temporalWeight, dispatchErr := parseBoundedFloat(params, "temporal_weight", 0.5)
	if dispatchErr != nil {
		return engramLinkCreateWeights{}, dispatchErr
	}
	confidence, dispatchErr := parseBoundedFloat(params, "confidence", 0.5)
	if dispatchErr != nil {
		return engramLinkCreateWeights{}, dispatchErr
	}
	return engramLinkCreateWeights{
		weight:         weight,
		temporalWeight: temporalWeight,
		confidence:     confidence,
	}, nil
}

func parseEngramLinkCreateMetadata(
	params map[string]any,
) (engramLinkCreateMetadata, *toolDispatchError) {
	evidenceJSON, ok := optionalMapParam(params, "evidence_json")
	if !ok {
		return engramLinkCreateMetadata{}, invalidParamError("evidence_json")
	}
	lastReinforcedAt, ok := optionalRFC3339TimePointerParam(params, "last_reinforced_at")
	if !ok {
		return engramLinkCreateMetadata{}, invalidParamError("last_reinforced_at")
	}
	return engramLinkCreateMetadata{
		evidenceJSON:     evidenceJSON,
		lastReinforcedAt: lastReinforcedAt,
	}, nil
}

func parseEngramLinkUpdateRequest(
	actor Actor,
	params map[string]any,
) (EngramLinkUpdateRequest, *toolDispatchError) {
	linkID, ok := requiredUUIDParam(params, "link_id")
	if !ok {
		return EngramLinkUpdateRequest{}, invalidParamError("link_id")
	}
	weight, temporalWeight, confidence, dispatchErr := parseEngramLinkUpdateFloats(params)
	if dispatchErr != nil {
		return EngramLinkUpdateRequest{}, dispatchErr
	}
	status, evidenceJSON, lastReinforcedAt, dispatchErr := parseEngramLinkUpdateMetadata(params)
	if dispatchErr != nil {
		return EngramLinkUpdateRequest{}, dispatchErr
	}
	request := EngramLinkUpdateRequest{
		ActorUserID:      actor.UserID,
		LinkID:           linkID,
		Weight:           weight,
		TemporalWeight:   temporalWeight,
		Confidence:       confidence,
		Status:           status,
		EvidenceJSON:     evidenceJSON,
		LastReinforcedAt: lastReinforcedAt,
	}
	if !hasEngramLinkUpdateFields(request) {
		return EngramLinkUpdateRequest{}, invalidParamsWithStatus(422, "No update fields provided")
	}
	return request, nil
}

func parseEngramLinkUpdateFloats(
	params map[string]any,
) (*float64, *float64, *float64, *toolDispatchError) {
	weight, dispatchErr := parseOptionalBoundedFloatField(params, "weight")
	if dispatchErr != nil {
		return nil, nil, nil, dispatchErr
	}
	temporalWeight, dispatchErr := parseOptionalBoundedFloatField(params, "temporal_weight")
	if dispatchErr != nil {
		return nil, nil, nil, dispatchErr
	}
	confidence, dispatchErr := parseOptionalBoundedFloatField(params, "confidence")
	if dispatchErr != nil {
		return nil, nil, nil, dispatchErr
	}
	return weight, temporalWeight, confidence, nil
}

func parseOptionalBoundedFloatField(
	params map[string]any,
	key engramLinkParamKey,
) (*float64, *toolDispatchError) {
	value, ok := optionalBoundedFloatPointer(params, key)
	if !ok {
		return nil, invalidParamError(string(key))
	}
	return value, nil
}

func parseEngramLinkUpdateMetadata(
	params map[string]any,
) (*models.EngramLinkStatus, *map[string]any, *time.Time, *toolDispatchError) {
	status, ok := optionalEngramLinkStatusPointer(params, "status")
	if !ok {
		return nil, nil, nil, invalidParamError("status")
	}
	evidenceJSON, ok := optionalMapPointerParam(params, "evidence_json")
	if !ok {
		return nil, nil, nil, invalidParamError("evidence_json")
	}
	lastReinforcedAt, ok := optionalRFC3339TimePointerParam(params, "last_reinforced_at")
	if !ok {
		return nil, nil, nil, invalidParamError("last_reinforced_at")
	}
	return status, evidenceJSON, lastReinforcedAt, nil
}

func hasEngramLinkUpdateFields(request EngramLinkUpdateRequest) bool {
	return request.Weight != nil ||
		request.TemporalWeight != nil ||
		request.Confidence != nil ||
		request.Status != nil ||
		request.EvidenceJSON != nil ||
		request.LastReinforcedAt != nil
}

func parseEngramLinkArchiveRequest(
	actor Actor,
	params map[string]any,
) (EngramLinkArchiveRequest, *toolDispatchError) {
	linkID, ok := requiredUUIDParam(params, "link_id")
	if !ok {
		return EngramLinkArchiveRequest{}, invalidParamError("link_id")
	}
	return EngramLinkArchiveRequest{
		ActorUserID: actor.UserID,
		LinkID:      linkID,
	}, nil
}
