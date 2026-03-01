package mcp

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"engram/internal/models"
	"engram/internal/repository"
)

const (
	defaultEngramLinkSuggestMaxCandidates = 20
	maxEngramLinkSuggestCandidates        = 50
)

func (service *CompatibilityService) dispatchEngramLinkCreateTool(
	ctx context.Context,
	actor Actor,
	params map[string]any,
) (map[string]any, bool, *toolDispatchError) {
	if service.engramLinkCreate == nil {
		return nil, false, nil
	}
	request, dispatchErr := parseEngramLinkCreateRequest(actor, params)
	if dispatchErr != nil {
		return nil, true, dispatchErr
	}
	created, err := service.engramLinkCreate.CreateEngramLink(ctx, request)
	if err != nil {
		return nil, true, mapEngramLinkDispatchError(err)
	}
	if created == nil {
		return nil, true, invalidParamsWithStatus(404, "Engram source or target not found")
	}
	return map[string]any{"link": *created}, true, nil
}

func (service *CompatibilityService) dispatchEngramLinkListTool(
	ctx context.Context,
	actor Actor,
	params map[string]any,
) (map[string]any, bool, *toolDispatchError) {
	if service.engramLinkList == nil {
		return nil, false, nil
	}
	request, dispatchErr := parseEngramLinkListRequest(actor, params)
	if dispatchErr != nil {
		return nil, true, dispatchErr
	}
	links, err := service.engramLinkList.ListEngramLinks(ctx, request)
	if err != nil {
		return nil, true, internalToolDispatchError()
	}
	return map[string]any{"links": links}, true, nil
}

func (service *CompatibilityService) dispatchEngramLinkUpdateTool(
	ctx context.Context,
	actor Actor,
	params map[string]any,
) (map[string]any, bool, *toolDispatchError) {
	if service.engramLinkUpdate == nil {
		return nil, false, nil
	}
	request, dispatchErr := parseEngramLinkUpdateRequest(actor, params)
	if dispatchErr != nil {
		return nil, true, dispatchErr
	}
	updated, err := service.engramLinkUpdate.UpdateEngramLink(ctx, request)
	if err != nil {
		return nil, true, mapEngramLinkDispatchError(err)
	}
	if updated == nil {
		return nil, true, invalidParamsWithStatus(404, "Engram link not found")
	}
	return map[string]any{"link": *updated}, true, nil
}

func (service *CompatibilityService) dispatchEngramLinkArchiveTool(
	ctx context.Context,
	actor Actor,
	params map[string]any,
) (map[string]any, bool, *toolDispatchError) {
	if service.engramLinkArchive == nil {
		return nil, false, nil
	}
	request, dispatchErr := parseEngramLinkArchiveRequest(actor, params)
	if dispatchErr != nil {
		return nil, true, dispatchErr
	}
	archived, err := service.engramLinkArchive.ArchiveEngramLink(ctx, request)
	if err != nil {
		return nil, true, mapEngramLinkDispatchError(err)
	}
	if archived == nil {
		return nil, true, invalidParamsWithStatus(404, "Engram link not found")
	}
	return map[string]any{"link": *archived}, true, nil
}

func (service *CompatibilityService) dispatchEngramLinkSuggestTool(
	ctx context.Context,
	actor Actor,
	params map[string]any,
) (map[string]any, bool, *toolDispatchError) {
	if service.engramLinkSuggest == nil {
		return nil, false, nil
	}
	request, dispatchErr := parseEngramLinkSuggestRequest(actor, params)
	if dispatchErr != nil {
		return nil, true, dispatchErr
	}
	suggestions, err := service.engramLinkSuggest.SuggestEngramLinks(ctx, request)
	if err != nil {
		return nil, true, internalToolDispatchError()
	}
	return map[string]any{"suggestions": suggestions}, true, nil
}

func (service *CompatibilityService) dispatchEngramTracePathTool(
	ctx context.Context,
	actor Actor,
	params map[string]any,
) (map[string]any, bool, *toolDispatchError) {
	if service.engramTracePath == nil {
		return nil, false, nil
	}
	request, dispatchErr := parseEngramTracePathRequest(actor, params)
	if dispatchErr != nil {
		return nil, true, dispatchErr
	}
	steps, err := service.engramTracePath.TraceEngramPath(ctx, request)
	if err != nil {
		return nil, true, internalToolDispatchError()
	}
	return map[string]any{"steps": steps}, true, nil
}

func parseEngramLinkCreateRequest(
	actor Actor,
	params map[string]any,
) (EngramLinkCreateRequest, *toolDispatchError) {
	sourceEngramID, ok := requiredUUIDParam(params, "engram_id")
	if !ok {
		return EngramLinkCreateRequest{}, invalidParamError("engram_id")
	}
	targetEngramID, ok := requiredUUIDParam(params, "target_engram_id")
	if !ok {
		return EngramLinkCreateRequest{}, invalidParamError("target_engram_id")
	}
	relationType, dispatchErr := parseEngramLinkRelationType(params, "relation_type", models.EngramLinkRelationRelatedTo)
	if dispatchErr != nil {
		return EngramLinkCreateRequest{}, dispatchErr
	}
	origin, dispatchErr := parseEngramLinkOrigin(params, "origin", models.EngramLinkOriginManual)
	if dispatchErr != nil {
		return EngramLinkCreateRequest{}, dispatchErr
	}
	status, dispatchErr := parseEngramLinkStatus(params, "status", models.EngramLinkStatusActive)
	if dispatchErr != nil {
		return EngramLinkCreateRequest{}, dispatchErr
	}
	weight, dispatchErr := parseBoundedFloat(params, "weight", 0.6)
	if dispatchErr != nil {
		return EngramLinkCreateRequest{}, dispatchErr
	}
	temporalWeight, dispatchErr := parseBoundedFloat(params, "temporal_weight", 0.5)
	if dispatchErr != nil {
		return EngramLinkCreateRequest{}, dispatchErr
	}
	confidence, dispatchErr := parseBoundedFloat(params, "confidence", 0.5)
	if dispatchErr != nil {
		return EngramLinkCreateRequest{}, dispatchErr
	}
	evidenceJSON, ok := optionalMapParam(params, "evidence_json")
	if !ok {
		return EngramLinkCreateRequest{}, invalidParamError("evidence_json")
	}
	lastReinforcedAt, ok := optionalRFC3339TimePointerParam(params, "last_reinforced_at")
	if !ok {
		return EngramLinkCreateRequest{}, invalidParamError("last_reinforced_at")
	}
	return EngramLinkCreateRequest{
		ActorUserID:      actor.UserID,
		SourceEngramID:   sourceEngramID,
		TargetEngramID:   targetEngramID,
		RelationType:     relationType,
		Weight:           weight,
		TemporalWeight:   temporalWeight,
		Confidence:       confidence,
		Origin:           origin,
		Status:           status,
		EvidenceJSON:     evidenceJSON,
		LastReinforcedAt: lastReinforcedAt,
	}, nil
}

func parseEngramLinkListRequest(
	actor Actor,
	params map[string]any,
) (EngramLinkListRequest, *toolDispatchError) {
	sourceEngramID, ok := requiredUUIDParam(params, "engram_id")
	if !ok {
		return EngramLinkListRequest{}, invalidParamError("engram_id")
	}
	relationType, ok := optionalEngramLinkRelationTypePointer(params, "relation_type")
	if !ok {
		return EngramLinkListRequest{}, invalidParamError("relation_type")
	}
	includeArchived, ok := optionalBoolParam(params, "include_archived", false)
	if !ok {
		return EngramLinkListRequest{}, invalidParamError("include_archived")
	}
	paging, pagingErr := parsePagingParams(
		params,
		defaultEngramLinkListLimit,
		defaultEngramLinkListOffset,
	)
	if pagingErr != nil {
		return EngramLinkListRequest{}, pagingErr
	}
	return EngramLinkListRequest{
		ActorUserID:     actor.UserID,
		SourceEngramID:  sourceEngramID,
		RelationType:    relationType,
		IncludeArchived: includeArchived,
		Limit:           paging.limit,
		Offset:          paging.offset,
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
	weight, ok := optionalBoundedFloatPointer(params, "weight")
	if !ok {
		return EngramLinkUpdateRequest{}, invalidParamError("weight")
	}
	temporalWeight, ok := optionalBoundedFloatPointer(params, "temporal_weight")
	if !ok {
		return EngramLinkUpdateRequest{}, invalidParamError("temporal_weight")
	}
	confidence, ok := optionalBoundedFloatPointer(params, "confidence")
	if !ok {
		return EngramLinkUpdateRequest{}, invalidParamError("confidence")
	}
	status, ok := optionalEngramLinkStatusPointer(params, "status")
	if !ok {
		return EngramLinkUpdateRequest{}, invalidParamError("status")
	}
	evidenceJSON, ok := optionalMapPointerParam(params, "evidence_json")
	if !ok {
		return EngramLinkUpdateRequest{}, invalidParamError("evidence_json")
	}
	lastReinforcedAt, ok := optionalRFC3339TimePointerParam(params, "last_reinforced_at")
	if !ok {
		return EngramLinkUpdateRequest{}, invalidParamError("last_reinforced_at")
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

func parseEngramLinkSuggestRequest(
	actor Actor,
	params map[string]any,
) (EngramLinkSuggestRequest, *toolDispatchError) {
	sourceEngramID, ok := requiredUUIDParam(params, "engram_id")
	if !ok {
		return EngramLinkSuggestRequest{}, invalidParamError("engram_id")
	}
	limit, ok := optionalIntParam(params, "limit", defaultEngramLinkSuggestLimit)
	if !ok {
		return EngramLinkSuggestRequest{}, invalidParamError("limit")
	}
	if limit < 1 || limit > maxEngramLinkSuggestCandidates {
		return EngramLinkSuggestRequest{}, invalidParamError("limit")
	}
	maxCandidates, ok := optionalIntParam(params, "max_candidates", defaultEngramLinkSuggestMaxCandidates)
	if !ok {
		return EngramLinkSuggestRequest{}, invalidParamError("max_candidates")
	}
	if maxCandidates < 1 || maxCandidates > maxEngramLinkSuggestCandidates {
		return EngramLinkSuggestRequest{}, invalidParamError("max_candidates")
	}
	if maxCandidates < limit {
		maxCandidates = limit
	}
	minimumScore, dispatchErr := parseBoundedFloat(params, "minimum_score", 0)
	if dispatchErr != nil {
		return EngramLinkSuggestRequest{}, dispatchErr
	}
	includeArchived, ok := optionalBoolParam(params, "include_archived", false)
	if !ok {
		return EngramLinkSuggestRequest{}, invalidParamError("include_archived")
	}
	return EngramLinkSuggestRequest{
		ActorUserID:     actor.UserID,
		SourceEngramID:  sourceEngramID,
		Limit:           limit,
		MaxCandidates:   maxCandidates,
		MinimumScore:    minimumScore,
		IncludeArchived: includeArchived,
	}, nil
}

func parseEngramTracePathRequest(
	actor Actor,
	params map[string]any,
) (EngramTracePathRequest, *toolDispatchError) {
	rootEngramID, ok := requiredUUIDParam(params, "engram_id")
	if !ok {
		return EngramTracePathRequest{}, invalidParamError("engram_id")
	}
	maxDepth, ok := optionalIntParam(params, "max_depth", 2)
	if !ok || maxDepth < 1 || maxDepth > 6 {
		return EngramTracePathRequest{}, invalidParamError("max_depth")
	}
	maxNeighbors, ok := optionalIntParam(params, "max_neighbors", 20)
	if !ok || maxNeighbors < 1 || maxNeighbors > 200 {
		return EngramTracePathRequest{}, invalidParamError("max_neighbors")
	}
	includeArchived, ok := optionalBoolParam(params, "include_archived", false)
	if !ok {
		return EngramTracePathRequest{}, invalidParamError("include_archived")
	}
	return EngramTracePathRequest{
		ActorUserID:     actor.UserID,
		RootEngramID:    rootEngramID,
		MaxDepth:        maxDepth,
		MaxNeighbors:    maxNeighbors,
		IncludeArchived: includeArchived,
	}, nil
}

func parseEngramLinkRelationType(
	params map[string]any,
	key string,
	defaultValue models.EngramLinkRelationType,
) (models.EngramLinkRelationType, *toolDispatchError) {
	value, ok := optionalStringPointerParam(params, key)
	if !ok {
		return "", invalidParamError(key)
	}
	if value == nil {
		return defaultValue, nil
	}
	parsed, err := models.ParseEngramLinkRelationType(strings.TrimSpace(*value))
	if err != nil {
		return "", invalidParamError(key)
	}
	return parsed, nil
}

func parseEngramLinkOrigin(
	params map[string]any,
	key string,
	defaultValue models.EngramLinkOrigin,
) (models.EngramLinkOrigin, *toolDispatchError) {
	value, ok := optionalStringPointerParam(params, key)
	if !ok {
		return "", invalidParamError(key)
	}
	if value == nil {
		return defaultValue, nil
	}
	parsed, err := models.ParseEngramLinkOrigin(strings.TrimSpace(*value))
	if err != nil {
		return "", invalidParamError(key)
	}
	return parsed, nil
}

func parseEngramLinkStatus(
	params map[string]any,
	key string,
	defaultValue models.EngramLinkStatus,
) (models.EngramLinkStatus, *toolDispatchError) {
	value, ok := optionalStringPointerParam(params, key)
	if !ok {
		return "", invalidParamError(key)
	}
	if value == nil {
		return defaultValue, nil
	}
	parsed, err := models.ParseEngramLinkStatus(strings.TrimSpace(*value))
	if err != nil {
		return "", invalidParamError(key)
	}
	return parsed, nil
}

func parseBoundedFloat(
	params map[string]any,
	key string,
	defaultValue float64,
) (float64, *toolDispatchError) {
	value, ok := optionalFloatParam(params, key)
	if !ok {
		return 0, invalidParamError(key)
	}
	if value == nil {
		return defaultValue, nil
	}
	if *value < 0 || *value > 1 {
		return 0, invalidParamError(key)
	}
	return *value, nil
}

func optionalBoundedFloatPointer(params map[string]any, key string) (*float64, bool) {
	value, ok := optionalFloatParam(params, key)
	if !ok {
		return nil, false
	}
	if value == nil {
		return nil, true
	}
	if *value < 0 || *value > 1 {
		return nil, false
	}
	return value, true
}

func optionalFloatParam(params map[string]any, key string) (*float64, bool) {
	value, found := optionalParamValue(params, key)
	if !found {
		return nil, true
	}
	switch typed := value.(type) {
	case float64:
		floatValue := typed
		return &floatValue, true
	case float32:
		floatValue := float64(typed)
		return &floatValue, true
	case int:
		floatValue := float64(typed)
		return &floatValue, true
	case int64:
		floatValue := float64(typed)
		return &floatValue, true
	case json.Number:
		parsed, err := typed.Float64()
		if err != nil {
			return nil, false
		}
		return &parsed, true
	case string:
		parsed, err := strconv.ParseFloat(strings.TrimSpace(typed), 64)
		if err != nil {
			return nil, false
		}
		return &parsed, true
	default:
		parsed, err := strconv.ParseFloat(strings.TrimSpace(fmt.Sprint(value)), 64)
		if err != nil {
			return nil, false
		}
		return &parsed, true
	}
}

func optionalEngramLinkRelationTypePointer(
	params map[string]any,
	key string,
) (*models.EngramLinkRelationType, bool) {
	value, ok := optionalStringPointerParam(params, key)
	if !ok {
		return nil, false
	}
	if value == nil {
		return nil, true
	}
	parsed, err := models.ParseEngramLinkRelationType(strings.TrimSpace(*value))
	if err != nil {
		return nil, false
	}
	return &parsed, true
}

func optionalEngramLinkStatusPointer(
	params map[string]any,
	key string,
) (*models.EngramLinkStatus, bool) {
	value, ok := optionalStringPointerParam(params, key)
	if !ok {
		return nil, false
	}
	if value == nil {
		return nil, true
	}
	parsed, err := models.ParseEngramLinkStatus(strings.TrimSpace(*value))
	if err != nil {
		return nil, false
	}
	return &parsed, true
}

func optionalMapParam(params map[string]any, key string) (map[string]any, bool) {
	value, ok := optionalMapPointerParam(params, key)
	if !ok {
		return nil, false
	}
	if value == nil {
		return map[string]any{}, true
	}
	if *value == nil {
		return map[string]any{}, true
	}
	return *value, true
}

func optionalMapPointerParam(params map[string]any, key string) (*map[string]any, bool) {
	value, found := optionalParamValue(params, key)
	if !found {
		return nil, true
	}
	encoded, err := json.Marshal(value)
	if err != nil {
		return nil, false
	}
	decoded := map[string]any{}
	if err := json.Unmarshal(encoded, &decoded); err != nil {
		return nil, false
	}
	return &decoded, true
}

func mapEngramLinkDispatchError(err error) *toolDispatchError {
	switch {
	case errors.Is(err, repository.ErrEngramLinkExists):
		return invalidParamsWithStatus(409, repository.ErrEngramLinkExists.Error())
	default:
		return internalToolDispatchError()
	}
}
