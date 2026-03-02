package mcp

import "engram/internal/models"

const (
	defaultEngramLinkSuggestMaxCandidates = 20
	maxEngramLinkSuggestCandidates        = 50
)

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

func parseEngramLinkSuggestRequest(
	actor Actor,
	params map[string]any,
) (EngramLinkSuggestRequest, *toolDispatchError) {
	sourceEngramID, ok := requiredUUIDParam(params, "engram_id")
	if !ok {
		return EngramLinkSuggestRequest{}, invalidParamError("engram_id")
	}
	limit, dispatchErr := parseRangedIntParam(params, intRangeSpec{
		key:          "limit",
		defaultValue: defaultEngramLinkSuggestLimit,
		minimum:      1,
		maximum:      maxEngramLinkSuggestCandidates,
	})
	if dispatchErr != nil {
		return EngramLinkSuggestRequest{}, dispatchErr
	}
	maxCandidates, dispatchErr := parseRangedIntParam(params, intRangeSpec{
		key:          "max_candidates",
		defaultValue: defaultEngramLinkSuggestMaxCandidates,
		minimum:      1,
		maximum:      maxEngramLinkSuggestCandidates,
	})
	if dispatchErr != nil {
		return EngramLinkSuggestRequest{}, dispatchErr
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
	maxDepth, dispatchErr := parseRangedIntParam(params, intRangeSpec{
		key:          "max_depth",
		defaultValue: 2,
		minimum:      1,
		maximum:      6,
	})
	if dispatchErr != nil {
		return EngramTracePathRequest{}, dispatchErr
	}
	maxNeighbors, dispatchErr := parseRangedIntParam(params, intRangeSpec{
		key:          "max_neighbors",
		defaultValue: 20,
		minimum:      1,
		maximum:      200,
	})
	if dispatchErr != nil {
		return EngramTracePathRequest{}, dispatchErr
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

type intRangeSpec struct {
	key          string
	defaultValue int
	minimum      int
	maximum      int
}

func parseRangedIntParam(params map[string]any, spec intRangeSpec) (int, *toolDispatchError) {
	value, ok := optionalIntParam(params, spec.key, spec.defaultValue)
	if !ok {
		return 0, invalidParamError(spec.key)
	}
	if value < spec.minimum || value > spec.maximum {
		return 0, invalidParamError(spec.key)
	}
	return value, nil
}

func optionalEngramLinkRelationTypePointer(
	params map[string]any,
	key engramLinkParamKey,
) (*models.EngramLinkRelationType, bool) {
	return parseOptionalEngramLinkEnumPointer(
		params,
		key,
		models.ParseEngramLinkRelationType,
	)
}

func optionalEngramLinkStatusPointer(
	params map[string]any,
	key engramLinkParamKey,
) (*models.EngramLinkStatus, bool) {
	return parseOptionalEngramLinkEnumPointer(
		params,
		key,
		models.ParseEngramLinkStatus,
	)
}
