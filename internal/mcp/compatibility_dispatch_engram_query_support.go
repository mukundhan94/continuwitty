package mcp

import (
	"context"
	"strings"
	"time"

	"engram/internal/models"
)

func (service *CompatibilityService) dispatchEngramQueryTool(
	ctx context.Context,
	actor Actor,
	params map[string]any,
) (map[string]any, bool, *toolDispatchError) {
	if service.engramQuery == nil {
		return nil, false, nil
	}
	request, dispatchErr := buildEngramQueryDispatchRequest(actor, params)
	if dispatchErr != nil {
		return nil, true, dispatchErr
	}
	results, err := service.engramQuery.QueryEngrams(ctx, request)
	if err != nil {
		return nil, true, internalToolDispatchError()
	}
	return map[string]any{"results": results}, true, nil
}

func buildEngramQueryDispatchRequest(
	actor Actor,
	params map[string]any,
) (EngramQueryDispatchRequest, *toolDispatchError) {
	query, dispatchErr := parseRequiredQueryParam(params)
	if dispatchErr != nil {
		return EngramQueryDispatchRequest{}, dispatchErr
	}
	payload, dispatchErr := parseEngramQueryPayload(params)
	if dispatchErr != nil {
		return EngramQueryDispatchRequest{}, dispatchErr
	}
	return EngramQueryDispatchRequest{
		ActorUserID: actor.UserID,
		Payload:     payload.withQuery(query),
	}, nil
}

type engramQueryPayloadParts struct {
	topK                    int
	projectID               *string
	tags                    []string
	keywords                []string
	createdAfter            *time.Time
	createdBefore           *time.Time
	accessCountMin          *int
	freshnessScoreMin       *float64
	lastAccessedAfter       *time.Time
	lastAccessedBefore      *time.Time
	freshnessComputedAfter  *time.Time
	freshnessComputedBefore *time.Time
	relationType            *models.EngramLinkRelationType
	traceDepth              *int
}

func parseRequiredQueryParam(params map[string]any) (string, *toolDispatchError) {
	query, ok := requiredStringParam(params, "query")
	if !ok {
		return "", invalidParamError("query")
	}
	return query, nil
}

func parseEngramQueryPayload(params map[string]any) (engramQueryPayloadParts, *toolDispatchError) {
	topK, dispatchErr := parseEngramQueryTopKParam(params)
	if dispatchErr != nil {
		return engramQueryPayloadParts{}, dispatchErr
	}
	tags, dispatchErr := parseOptionalParam(params, "tags", optionalStringArrayParam)
	if dispatchErr != nil {
		return engramQueryPayloadParts{}, dispatchErr
	}
	keywords, dispatchErr := parseOptionalParam(params, "keywords", optionalStringArrayParam)
	if dispatchErr != nil {
		return engramQueryPayloadParts{}, dispatchErr
	}
	temporalParts, dispatchErr := parseEngramQueryTemporalParts(params)
	if dispatchErr != nil {
		return engramQueryPayloadParts{}, dispatchErr
	}
	engagementParts, dispatchErr := parseEngramQueryEngagementParts(params)
	if dispatchErr != nil {
		return engramQueryPayloadParts{}, dispatchErr
	}
	traceParts, dispatchErr := parseEngramQueryTraceParts(params)
	if dispatchErr != nil {
		return engramQueryPayloadParts{}, dispatchErr
	}
	return engramQueryPayloadParts{
		topK:                    topK,
		projectID:               optionalProjectIDParam(params, "project_id"),
		tags:                    tags,
		keywords:                keywords,
		createdAfter:            temporalParts.createdAfter,
		createdBefore:           temporalParts.createdBefore,
		accessCountMin:          engagementParts.accessCountMin,
		freshnessScoreMin:       engagementParts.freshnessScoreMin,
		lastAccessedAfter:       temporalParts.lastAccessedAfter,
		lastAccessedBefore:      temporalParts.lastAccessedBefore,
		freshnessComputedAfter:  temporalParts.freshnessComputedAfter,
		freshnessComputedBefore: temporalParts.freshnessComputedBefore,
		relationType:            traceParts.relationType,
		traceDepth:              traceParts.traceDepth,
	}, nil
}

type engramQueryTemporalParts struct {
	createdAfter            *time.Time
	createdBefore           *time.Time
	lastAccessedAfter       *time.Time
	lastAccessedBefore      *time.Time
	freshnessComputedAfter  *time.Time
	freshnessComputedBefore *time.Time
}

func parseEngramQueryTemporalParts(params map[string]any) (engramQueryTemporalParts, *toolDispatchError) {
	createdAfter, dispatchErr := parseOptionalParam(params, "created_after", optionalRFC3339TimeParam)
	if dispatchErr != nil {
		return engramQueryTemporalParts{}, dispatchErr
	}
	createdBefore, dispatchErr := parseOptionalParam(params, "created_before", optionalRFC3339TimeParam)
	if dispatchErr != nil {
		return engramQueryTemporalParts{}, dispatchErr
	}
	lastAccessedAfter, dispatchErr := parseOptionalParam(params, "last_accessed_after", optionalRFC3339TimeParam)
	if dispatchErr != nil {
		return engramQueryTemporalParts{}, dispatchErr
	}
	lastAccessedBefore, dispatchErr := parseOptionalParam(params, "last_accessed_before", optionalRFC3339TimeParam)
	if dispatchErr != nil {
		return engramQueryTemporalParts{}, dispatchErr
	}
	freshnessComputedAfter, dispatchErr := parseOptionalParam(params, "freshness_computed_after", optionalRFC3339TimeParam)
	if dispatchErr != nil {
		return engramQueryTemporalParts{}, dispatchErr
	}
	freshnessComputedBefore, dispatchErr := parseOptionalParam(params, "freshness_computed_before", optionalRFC3339TimeParam)
	if dispatchErr != nil {
		return engramQueryTemporalParts{}, dispatchErr
	}
	parts := engramQueryTemporalParts{
		createdAfter:            createdAfter,
		createdBefore:           createdBefore,
		lastAccessedAfter:       lastAccessedAfter,
		lastAccessedBefore:      lastAccessedBefore,
		freshnessComputedAfter:  freshnessComputedAfter,
		freshnessComputedBefore: freshnessComputedBefore,
	}
	dispatchErr = validateTemporalWindows(temporalWindowSpecs(parts))
	if dispatchErr != nil {
		return engramQueryTemporalParts{}, dispatchErr
	}
	return parts, nil
}

func temporalWindowSpecs(parts engramQueryTemporalParts) []temporalWindowSpec {
	return []temporalWindowSpec{
		{after: parts.createdAfter, before: parts.createdBefore, afterParam: "created_after"},
		{after: parts.lastAccessedAfter, before: parts.lastAccessedBefore, afterParam: "last_accessed_after"},
		{
			after:      parts.freshnessComputedAfter,
			before:     parts.freshnessComputedBefore,
			afterParam: "freshness_computed_after",
		},
	}
}

type engramQueryEngagementParts struct {
	accessCountMin    *int
	freshnessScoreMin *float64
}

func parseEngramQueryEngagementParts(params map[string]any) (engramQueryEngagementParts, *toolDispatchError) {
	accessCountMin, dispatchErr := parseEngramQueryAccessCountMin(params)
	if dispatchErr != nil {
		return engramQueryEngagementParts{}, dispatchErr
	}
	freshnessScoreMin, dispatchErr := parseEngramQueryFreshnessScoreMin(params)
	if dispatchErr != nil {
		return engramQueryEngagementParts{}, dispatchErr
	}
	return engramQueryEngagementParts{
		accessCountMin:    accessCountMin,
		freshnessScoreMin: freshnessScoreMin,
	}, nil
}

type engramQueryTraceParts struct {
	relationType *models.EngramLinkRelationType
	traceDepth   *int
}

func parseEngramQueryTraceParts(params map[string]any) (engramQueryTraceParts, *toolDispatchError) {
	relationType, ok := optionalEngramLinkRelationTypePointer(params, engramLinkParamKey("relation_type"))
	if !ok {
		return engramQueryTraceParts{}, invalidParamError("relation_type")
	}
	traceDepth, dispatchErr := parseTraceDepthParam(params, relationType)
	if dispatchErr != nil {
		return engramQueryTraceParts{}, dispatchErr
	}
	return engramQueryTraceParts{relationType: relationType, traceDepth: traceDepth}, nil
}

func parseTraceDepthParam(
	params map[string]any,
	relationType *models.EngramLinkRelationType,
) (*int, *toolDispatchError) {
	value, ok := optionalIntPointerParam(params, "trace_depth")
	if !ok {
		return nil, invalidParamError("trace_depth")
	}
	if value == nil {
		if relationType == nil {
			return nil, nil
		}
		defaultDepth := 1
		return &defaultDepth, nil
	}
	if *value < 0 || *value > 1 {
		return nil, invalidParamError("trace_depth")
	}
	if relationType != nil && *value == 0 {
		return nil, invalidParamError("trace_depth")
	}
	return value, nil
}

func parseOptionalParam[T any](
	params map[string]any,
	key string,
	parser func(map[string]any, string) (T, bool),
) (T, *toolDispatchError) {
	parsed, ok := parser(params, key)
	if !ok {
		var zero T
		return zero, invalidParamError(key)
	}
	return parsed, nil
}

func (parts engramQueryPayloadParts) withQuery(query string) models.EngramQueryRequest {
	return models.EngramQueryRequest{
		Query:                   query,
		TopK:                    parts.topK,
		ProjectID:               parts.projectID,
		Tags:                    parts.tags,
		Keywords:                parts.keywords,
		CreatedAfter:            parts.createdAfter,
		CreatedBefore:           parts.createdBefore,
		AccessCountMin:          parts.accessCountMin,
		FreshnessScoreMin:       parts.freshnessScoreMin,
		LastAccessedAfter:       parts.lastAccessedAfter,
		LastAccessedBefore:      parts.lastAccessedBefore,
		FreshnessComputedAfter:  parts.freshnessComputedAfter,
		FreshnessComputedBefore: parts.freshnessComputedBefore,
		RelationType:            parts.relationType,
		TraceDepth:              parts.traceDepth,
	}
}

type temporalWindowSpec struct {
	after      *time.Time
	before     *time.Time
	afterParam string
}

func validateTemporalWindows(specs []temporalWindowSpec) *toolDispatchError {
	for _, spec := range specs {
		if spec.after == nil || spec.before == nil {
			continue
		}
		if spec.after.After(*spec.before) {
			return invalidParamError(spec.afterParam)
		}
	}
	return nil
}

func parseEngramQueryTopKParam(params map[string]any) (int, *toolDispatchError) {
	topK, ok := optionalIntParam(params, "top_k", 5)
	if !ok {
		return 0, invalidParamError("top_k")
	}
	if topK < 1 {
		return 0, invalidParamError("top_k")
	}
	if topK > 50 {
		return 0, invalidParamError("top_k")
	}
	return topK, nil
}

func optionalStringArrayParam(params map[string]any, key string) ([]string, bool) {
	value, found := optionalParamValue(params, key)
	if !found {
		return []string{}, true
	}
	switch typed := value.(type) {
	case []string:
		return append([]string(nil), typed...), true
	case []any:
		return stringArrayFromAnySlice(typed)
	default:
		return nil, false
	}
}

func stringArrayFromAnySlice(values []any) ([]string, bool) {
	result := make([]string, 0, len(values))
	for _, item := range values {
		text, ok := item.(string)
		if !ok {
			return nil, false
		}
		result = append(result, text)
	}
	return result, true
}

func optionalRFC3339TimeParam(params map[string]any, key string) (*time.Time, bool) {
	value, found := optionalParamValue(params, key)
	if !found {
		return nil, true
	}
	text, ok := value.(string)
	if !ok {
		return nil, false
	}
	return parseRFC3339Pointer(text)
}

func parseRFC3339Pointer(raw string) (*time.Time, bool) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return nil, false
	}
	parsed, err := time.Parse(time.RFC3339, trimmed)
	if err != nil {
		return nil, false
	}
	return &parsed, true
}

func parseEngramQueryAccessCountMin(params map[string]any) (*int, *toolDispatchError) {
	value, ok := optionalIntPointerParam(params, "access_count_min")
	if !ok {
		return nil, invalidParamError("access_count_min")
	}
	if value != nil && *value < 0 {
		return nil, invalidParamError("access_count_min")
	}
	return value, nil
}

func parseEngramQueryFreshnessScoreMin(params map[string]any) (*float64, *toolDispatchError) {
	rawValue, found := optionalParamValue(params, "freshness_score_min")
	if !found {
		return nil, nil
	}
	parsed, dispatchErr := parseFreshnessScore(rawValue)
	if dispatchErr != nil {
		return nil, dispatchErr
	}
	copy := parsed
	return &copy, nil
}

func parseFreshnessScore(value any) (float64, *toolDispatchError) {
	parsed, ok := parseFloatValue(value)
	if !ok {
		return 0, invalidParamError("freshness_score_min")
	}
	if parsed < 0 {
		return 0, invalidParamError("freshness_score_min")
	}
	if parsed > 1 {
		return 0, invalidParamError("freshness_score_min")
	}
	return parsed, nil
}
